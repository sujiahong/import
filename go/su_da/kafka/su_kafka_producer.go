package su_kafka

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"go.local/su_errors"
	slog "go.local/su_log"
	"go.uber.org/zap"
)

type KafkaProducerConfig struct {
	AddrSlice         []string
	Topic             string
	Async             bool
	ChannelBufferSize int
	ReturnSuccesses   bool
	FlushMessages     int
	FlushFrequency    time.Duration
	Compression       sarama.CompressionCodec
	RequiredAcks      sarama.RequiredAcks
	RetryInterval     time.Duration
	LogMessages       bool
}

var (
	newSaramaAsyncProducer = sarama.NewAsyncProducer
	newSaramaSyncProducer  = sarama.NewSyncProducer
)

type KafkaProducer struct {
	AddrSlice     []string
	Topic         string
	Async         bool
	client        sarama.SyncProducer
	asclient      sarama.AsyncProducer
	cfg           KafkaProducerConfig
	mu            sync.RWMutex
	reconnectMu   sync.Mutex
	ctx           context.Context
	cancel        func()
	closeOnce     sync.Once
	closeErr      error
	retryInterval time.Duration
	logMessages   bool
}

type syncSendResult struct {
	partition int32
	offset    int64
	err       error
}

func NewKafkaProducer(a_addr []string, a_topic string, a_async bool) *KafkaProducer {
	return NewKafkaProducerWithConfig(KafkaProducerConfig{
		AddrSlice: a_addr,
		Topic:     a_topic,
		Async:     a_async,
	})
}

func NewKafkaProducerWithConfig(cfg KafkaProducerConfig) *KafkaProducer {
	cfg = defaultKafkaProducerConfig(cfg)
	client, asclient, err := newKafkaProducerClients(cfg)
	if err != nil {
		slog.Error("kafka NewProducer failed", zap.Error(err))
		return nil
	}
	kp := &KafkaProducer{
		AddrSlice:     cfg.AddrSlice,
		Topic:         cfg.Topic,
		Async:         cfg.Async,
		client:        client,
		asclient:      asclient,
		cfg:           cfg,
		retryInterval: cfg.RetryInterval,
		logMessages:   cfg.LogMessages,
	}
	kp.ctx, kp.cancel = context.WithCancel(context.Background())
	if kp.Async {
		if cfg.ReturnSuccesses {
			go kp.handleSuccess(asclient)
		}
		go kp.handleError(asclient)
	}
	slog.Info("创建kafka生成者完成", zap.Any("a_topic", cfg.Topic), zap.Any("a_async", cfg.Async))
	return kp
}

func (kp *KafkaProducer) Send(a_msg string) error {
	timeNow := time.Now().UnixNano()
	key := strconv.FormatInt(timeNow, 10)
	return kp.SendWithKey(key, a_msg)
}

func (kp *KafkaProducer) SendWithKey(a_key, a_msg string) error {
	return kp.SendContext(context.Background(), a_key, a_msg)
}

func (kp *KafkaProducer) SendContext(ctx context.Context, a_key, a_msg string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = su_errors.NewRetryable(su_errors.CodeUnavailable, "kafka producer send failed")
			slog.Error("kafka producer send panic", zap.Any("recover", r))
		}
	}()
	if kp == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "kafka producer is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	kp.mu.RLock()
	topic := kp.Topic
	async := kp.Async
	client := kp.client
	asclient := kp.asclient
	producerCtx := kp.ctx
	kp.mu.RUnlock()

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(a_key),
		Value: sarama.StringEncoder(a_msg),
	}
	if async {
		if asclient == nil {
			return su_errors.New(su_errors.CodeUnavailable, "kafka async producer is not connected")
		}
		if producerCtx == nil {
			return su_errors.New(su_errors.CodeInternal, "kafka producer context is nil")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-producerCtx.Done():
			return su_errors.New(su_errors.CodeInternal, "kafka producer is closed")
		case asclient.Input() <- msg:
			if kp.logMessages {
				slog.Info("kafka async send", zap.Any("a_key", a_key))
			}
			return nil
		}
	}
	if client == nil {
		return su_errors.New(su_errors.CodeUnavailable, "kafka sync producer is not connected")
	}
	var producerDone <-chan struct{}
	if producerCtx != nil {
		producerDone = producerCtx.Done()
	}
	ctxDone := ctx.Done()
	select {
	case <-ctxDone:
		return ctx.Err()
	case <-producerDone:
		return su_errors.New(su_errors.CodeInternal, "kafka producer is closed")
	default:
	}

	if ctxDone == nil {
		return kp.sendSyncMessageWithRetry(client, msg)
	}

	resultCh := make(chan syncSendResult, 1)
	go func() {
		pid, offset, sendErr := client.SendMessage(msg)
		resultCh <- syncSendResult{partition: pid, offset: offset, err: sendErr}
	}()

	select {
	case <-ctxDone:
		return ctx.Err()
	case <-producerDone:
		return su_errors.New(su_errors.CodeInternal, "kafka producer is closed")
	case result := <-resultCh:
		if result.err != nil {
			if retryErr := kp.reconnectAndRetrySync(ctx, msg); retryErr == nil {
				return nil
			}
			slog.Error("kafka SendMessage failed", zap.Error(result.err))
			return su_errors.WrapRetryable(su_errors.CodeUnavailable, "kafka send message failed", result.err)
		}
		if kp.logMessages {
			slog.Info("kafka sync send", zap.Any("pid", result.partition), zap.Any("offset", result.offset))
		}
		return nil
	}
}

func (kp *KafkaProducer) sendSyncMessageWithRetry(client sarama.SyncProducer, msg *sarama.ProducerMessage) error {
	pid, offset, err := client.SendMessage(msg)
	if err != nil {
		if retryErr := kp.reconnectAndRetrySync(context.Background(), msg); retryErr == nil {
			return nil
		}
		slog.Error("kafka SendMessage failed", zap.Error(err))
		return su_errors.WrapRetryable(su_errors.CodeUnavailable, "kafka send message failed", err)
	}
	if kp.logMessages {
		slog.Info("kafka sync send", zap.Any("pid", pid), zap.Any("offset", offset))
	}
	return nil
}

func (kp *KafkaProducer) reconnectAndRetrySync(ctx context.Context, msg *sarama.ProducerMessage) error {
	if err := kp.Reconnect(); err != nil {
		return err
	}
	kp.mu.RLock()
	client := kp.client
	producerCtx := kp.ctx
	kp.mu.RUnlock()
	if client == nil {
		return su_errors.New(su_errors.CodeUnavailable, "kafka sync producer is not connected")
	}
	var producerDone <-chan struct{}
	if producerCtx != nil {
		producerDone = producerCtx.Done()
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-producerDone:
		return su_errors.New(su_errors.CodeInternal, "kafka producer is closed")
	default:
	}
	pid, offset, err := client.SendMessage(msg)
	if err != nil {
		return su_errors.WrapRetryable(su_errors.CodeUnavailable, "kafka send message retry failed", err)
	}
	if kp.logMessages {
		slog.Info("kafka sync send retry", zap.Any("pid", pid), zap.Any("offset", offset))
	}
	return nil
}

func (kp *KafkaProducer) Reconnect() error {
	if kp == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "kafka producer is nil")
	}
	kp.reconnectMu.Lock()
	defer kp.reconnectMu.Unlock()
	kp.mu.RLock()
	cfg := kp.cfg
	addrSlice := kp.AddrSlice
	topic := kp.Topic
	async := kp.Async
	producerCtx := kp.ctx
	kp.mu.RUnlock()
	if producerCtx == nil {
		return su_errors.New(su_errors.CodeInternal, "kafka producer context is nil")
	}
	select {
	case <-producerCtx.Done():
		return su_errors.New(su_errors.CodeInternal, "kafka producer is closed")
	default:
	}
	if len(cfg.AddrSlice) == 0 {
		cfg.AddrSlice = addrSlice
		cfg.Topic = topic
		cfg.Async = async
	}
	client, asclient, err := newKafkaProducerClients(cfg)
	if err != nil {
		return su_errors.WrapRetryable(su_errors.CodeUnavailable, "kafka producer reconnect failed", err)
	}
	select {
	case <-producerCtx.Done():
		if asclient != nil {
			_ = asclient.Close()
		}
		if client != nil {
			_ = client.Close()
		}
		return su_errors.New(su_errors.CodeInternal, "kafka producer is closed")
	default:
	}
	kp.mu.Lock()
	oldClient := kp.client
	oldAsyncClient := kp.asclient
	kp.client = client
	kp.asclient = asclient
	kp.mu.Unlock()
	if cfg.Async {
		if cfg.ReturnSuccesses {
			go kp.handleSuccess(asclient)
		}
		go kp.handleError(asclient)
	}
	if oldAsyncClient != nil {
		_ = oldAsyncClient.Close()
	}
	if oldClient != nil {
		_ = oldClient.Close()
	}
	return nil
}

func (kp *KafkaProducer) reconnectAsyncProducer(failedClient sarama.AsyncProducer) error {
	if kp == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "kafka producer is nil")
	}
	kp.reconnectMu.Lock()
	defer kp.reconnectMu.Unlock()
	kp.mu.RLock()
	cfg := kp.cfg
	addrSlice := kp.AddrSlice
	topic := kp.Topic
	producerCtx := kp.ctx
	currentClient := kp.asclient
	kp.mu.RUnlock()
	if producerCtx == nil {
		return su_errors.New(su_errors.CodeInternal, "kafka producer context is nil")
	}
	select {
	case <-producerCtx.Done():
		return su_errors.New(su_errors.CodeInternal, "kafka producer is closed")
	default:
	}
	if failedClient != nil && currentClient != failedClient {
		return nil
	}
	if len(cfg.AddrSlice) == 0 {
		cfg.AddrSlice = addrSlice
	}
	if cfg.Topic == "" {
		cfg.Topic = topic
	}
	cfg.Async = true
	client, asclient, err := newKafkaProducerClients(cfg)
	if err != nil {
		return su_errors.WrapRetryable(su_errors.CodeUnavailable, "kafka async producer reconnect failed", err)
	}
	select {
	case <-producerCtx.Done():
		if asclient != nil {
			_ = asclient.Close()
		}
		if client != nil {
			_ = client.Close()
		}
		return su_errors.New(su_errors.CodeInternal, "kafka producer is closed")
	default:
	}
	kp.mu.Lock()
	oldAsyncClient := kp.asclient
	kp.asclient = asclient
	kp.mu.Unlock()
	if cfg.ReturnSuccesses {
		go kp.handleSuccess(asclient)
	}
	go kp.handleError(asclient)
	if oldAsyncClient != nil {
		_ = oldAsyncClient.Close()
	}
	return nil
}

func (kp *KafkaProducer) recoverAsyncProducer(failedClient sarama.AsyncProducer, producerErr *sarama.ProducerError) {
	if producerErr == nil {
		return
	}
	msg := producerErr.Msg
	reconnectClient := failedClient
	for {
		if err := kp.reconnectAsyncProducer(reconnectClient); err != nil {
			if kp.isProducerClosed() {
				return
			}
			slog.Error("kafka async producer reconnect failed", zap.Error(err))
			if !kp.waitProducerRetry() {
				return
			}
			continue
		}
		if msg == nil {
			return
		}
		client, err := kp.getAsyncProducer()
		if err != nil {
			if kp.isProducerClosed() {
				return
			}
			slog.Error("kafka async producer retry skipped", zap.Error(err))
			if !kp.waitProducerRetry() {
				return
			}
			reconnectClient = nil
			continue
		}
		if err := kp.retryAsyncMessage(client, msg); err != nil {
			if kp.isProducerClosed() {
				return
			}
			slog.Error("kafka async message retry failed", zap.Error(err))
			reconnectClient = client
			if !kp.waitProducerRetry() {
				return
			}
			continue
		}
		return
	}
}

func (kp *KafkaProducer) getAsyncProducer() (sarama.AsyncProducer, error) {
	kp.mu.RLock()
	client := kp.asclient
	producerCtx := kp.ctx
	kp.mu.RUnlock()
	if producerCtx == nil {
		return nil, su_errors.New(su_errors.CodeInternal, "kafka producer context is nil")
	}
	select {
	case <-producerCtx.Done():
		return nil, su_errors.New(su_errors.CodeInternal, "kafka producer is closed")
	default:
	}
	if client == nil {
		return nil, su_errors.New(su_errors.CodeUnavailable, "kafka async producer is not connected")
	}
	return client, nil
}

func (kp *KafkaProducer) retryAsyncMessage(asclient sarama.AsyncProducer, msg *sarama.ProducerMessage) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = su_errors.NewRetryable(su_errors.CodeUnavailable, "kafka async producer retry failed")
			slog.Error("kafka async producer retry panic", zap.Any("recover", r))
		}
	}()
	if asclient == nil {
		return su_errors.New(su_errors.CodeUnavailable, "kafka async producer is not connected")
	}
	kp.mu.RLock()
	producerCtx := kp.ctx
	kp.mu.RUnlock()
	if producerCtx == nil {
		return su_errors.New(su_errors.CodeInternal, "kafka producer context is nil")
	}
	select {
	case <-producerCtx.Done():
		return su_errors.New(su_errors.CodeInternal, "kafka producer is closed")
	case asclient.Input() <- msg:
		if kp.logMessages {
			slog.Info("kafka async send retry")
		}
		return nil
	}
}

func (kp *KafkaProducer) waitProducerRetry() bool {
	kp.mu.RLock()
	producerCtx := kp.ctx
	retryInterval := kp.retryInterval
	kp.mu.RUnlock()
	if producerCtx == nil {
		return false
	}
	if retryInterval <= 0 {
		retryInterval = time.Second
	}
	timer := time.NewTimer(retryInterval)
	defer timer.Stop()
	select {
	case <-producerCtx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (kp *KafkaProducer) isProducerClosed() bool {
	if kp == nil {
		return true
	}
	kp.mu.RLock()
	producerCtx := kp.ctx
	kp.mu.RUnlock()
	if producerCtx == nil {
		return true
	}
	select {
	case <-producerCtx.Done():
		return true
	default:
		return false
	}
}

func (kp *KafkaProducer) Close() error {
	if kp == nil {
		return nil
	}
	kp.closeOnce.Do(func() {
		kp.reconnectMu.Lock()
		defer kp.reconnectMu.Unlock()
		kp.mu.Lock()
		client := kp.client
		asclient := kp.asclient
		async := kp.Async
		kp.client = nil
		kp.asclient = nil
		kp.mu.Unlock()
		if async {
			if asclient != nil {
				kp.closeErr = asclient.Close()
			}
			if kp.cancel != nil {
				kp.cancel()
			}
			return
		}
		if client != nil {
			kp.closeErr = client.Close()
		}
		if kp.cancel != nil {
			kp.cancel()
		}
	})
	return kp.closeErr
}

func (kp *KafkaProducer) handleSuccess(asclient sarama.AsyncProducer) {
	if asclient == nil {
		return
	}
	for {
		select {
		case <-kp.ctx.Done():
			return
		case pm, ok := <-asclient.Successes():
			if !ok {
				return
			}
			if kp.logMessages && pm != nil {
				slog.Info("kafka success", zap.Int32("Partition", pm.Partition), zap.Int64("Offset", pm.Offset))
			}
		}
	}
}

func (kp *KafkaProducer) handleError(asclient sarama.AsyncProducer) {
	if asclient == nil {
		return
	}
	for {
		select {
		case <-kp.ctx.Done():
			return
		case err, ok := <-asclient.Errors():
			if !ok {
				return
			}
			if err != nil {
				slog.Error("kafka error", zap.Error(err))
				go kp.recoverAsyncProducer(asclient, err)
			}
		}
	}
}

func newKafkaProducerClients(cfg KafkaProducerConfig) (sarama.SyncProducer, sarama.AsyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = cfg.RequiredAcks
	config.Producer.Return.Errors = true
	config.Producer.Return.Successes = cfg.ReturnSuccesses
	config.Producer.Compression = cfg.Compression
	config.ChannelBufferSize = cfg.ChannelBufferSize
	if cfg.FlushMessages > 0 {
		config.Producer.Flush.Messages = cfg.FlushMessages
	}
	if cfg.FlushFrequency > 0 {
		config.Producer.Flush.Frequency = cfg.FlushFrequency
	}
	if cfg.Async {
		asclient, err := newSaramaAsyncProducer(cfg.AddrSlice, config)
		return nil, asclient, err
	}
	config.Producer.Return.Successes = true
	client, err := newSaramaSyncProducer(cfg.AddrSlice, config)
	return client, nil, err
}

func defaultKafkaProducerConfig(cfg KafkaProducerConfig) KafkaProducerConfig {
	if cfg.ChannelBufferSize <= 0 {
		cfg.ChannelBufferSize = 2000
	}
	if cfg.RetryInterval <= 0 {
		cfg.RetryInterval = time.Second
	}
	return cfg
}
