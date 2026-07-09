package su_kafka

import (
	"context"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"go.local/su_errors"
	slog "go.local/su_log"
	"go.local/su_util"
	"go.uber.org/zap"
)

type KafkaBackpressureMode int

const (
	KafkaBackpressureBlock KafkaBackpressureMode = iota
	KafkaBackpressureDrop
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
	LogMessages       bool
}

type KafkaProducer struct {
	AddrSlice   []string
	Topic       string
	Async       bool
	client      sarama.SyncProducer
	asclient    sarama.AsyncProducer
	ctx         context.Context
	cancel      func()
	closeOnce   sync.Once
	closeErr    error
	logMessages bool
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

	kp := &KafkaProducer{
		AddrSlice:   cfg.AddrSlice,
		Topic:       cfg.Topic,
		Async:       cfg.Async,
		logMessages: cfg.LogMessages,
	}
	var err error
	if cfg.Async {
		kp.asclient, err = sarama.NewAsyncProducer(cfg.AddrSlice, config)
		if err != nil {
			slog.Error("kafka NewAsyncProducer failed", zap.Error(err))
			return nil
		}
	} else {
		config.Producer.Return.Successes = true
		kp.client, err = sarama.NewSyncProducer(cfg.AddrSlice, config)
		if err != nil {
			slog.Error("kafka NewSyncProducer failed", zap.Error(err))
			return nil
		}
	}
	kp.ctx, kp.cancel = context.WithCancel(context.Background())
	if kp.Async {
		if cfg.ReturnSuccesses {
			go kp.handleSuccess()
		}
		go kp.handleError()
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
	msg := &sarama.ProducerMessage{
		Topic: kp.Topic,
		Key:   sarama.StringEncoder(a_key),
		Value: sarama.StringEncoder(a_msg),
	}
	if kp.Async {
		if kp.asclient == nil {
			return su_errors.New(su_errors.CodeUnavailable, "kafka async producer is not connected")
		}
		if kp.ctx == nil {
			return su_errors.New(su_errors.CodeInternal, "kafka producer context is nil")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-kp.ctx.Done():
			return su_errors.New(su_errors.CodeInternal, "kafka producer is closed")
		case kp.asclient.Input() <- msg:
			if kp.logMessages {
				slog.Info("kafka async send", zap.Any("a_key", a_key))
			}
			return nil
		}
	}
	if kp.client == nil {
		return su_errors.New(su_errors.CodeUnavailable, "kafka sync producer is not connected")
	}
	var producerDone <-chan struct{}
	if kp.ctx != nil {
		producerDone = kp.ctx.Done()
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
		return kp.sendSyncMessage(msg)
	}

	resultCh := make(chan syncSendResult, 1)
	go func() {
		pid, offset, sendErr := kp.client.SendMessage(msg)
		resultCh <- syncSendResult{partition: pid, offset: offset, err: sendErr}
	}()

	select {
	case <-ctxDone:
		return ctx.Err()
	case <-producerDone:
		return su_errors.New(su_errors.CodeInternal, "kafka producer is closed")
	case result := <-resultCh:
		if result.err != nil {
			slog.Error("kafka SendMessage failed", zap.Error(result.err))
			return su_errors.WrapRetryable(su_errors.CodeUnavailable, "kafka send message failed", result.err)
		}
		if kp.logMessages {
			slog.Info("kafka sync send", zap.Any("pid", result.partition), zap.Any("offset", result.offset))
		}
		return nil
	}
}

func (kp *KafkaProducer) sendSyncMessage(msg *sarama.ProducerMessage) error {
	pid, offset, err := kp.client.SendMessage(msg)
	if err != nil {
		slog.Error("kafka SendMessage failed", zap.Error(err))
		return su_errors.WrapRetryable(su_errors.CodeUnavailable, "kafka send message failed", err)
	}
	if kp.logMessages {
		slog.Info("kafka sync send", zap.Any("pid", pid), zap.Any("offset", offset))
	}
	return nil
}

func (kp *KafkaProducer) Close() error {
	if kp == nil {
		return nil
	}
	kp.closeOnce.Do(func() {
		if kp.cancel != nil {
			kp.cancel()
		}
		if kp.Async {
			if kp.asclient != nil {
				kp.closeErr = kp.asclient.Close()
			}
			return
		}
		if kp.client != nil {
			kp.closeErr = kp.client.Close()
		}
	})
	return kp.closeErr
}

func (kp *KafkaProducer) handleSuccess() {
	for {
		select {
		case <-kp.ctx.Done():
			return
		case pm, ok := <-kp.asclient.Successes():
			if !ok {
				return
			}
			if kp.logMessages && pm != nil {
				slog.Info("kafka success", zap.Int32("Partition", pm.Partition), zap.Int64("Offset", pm.Offset))
			}
		}
	}
}

func (kp *KafkaProducer) handleError() {
	for {
		select {
		case <-kp.ctx.Done():
			return
		case err, ok := <-kp.asclient.Errors():
			if !ok {
				return
			}
			if err != nil {
				slog.Error("kafka error", zap.Error(err))
			}
		}
	}
}
/* msg *sarama.ConsumerMessage
msg.Topic       // topic 名
msg.Partition   // 分区 id
msg.Offset      // 当前消息 offset
msg.Key         // 消息 key，[]byte
msg.Value       // 消息内容，[]byte
msg.Headers     // Kafka headers
msg.Timestamp   // 消息时间
*/
type HandleFunc func(a_partion_id int32, msg *sarama.ConsumerMessage)
type HandleMessageFunc func(ctx context.Context, msg *sarama.ConsumerMessage) error

type KafkaConsumerConfig struct {
	AddrSlice        []string
	Topic            string
	ClientID         string
	WorkerNum        uint32
	QueueSize        uint32
	CloseTimeout     time.Duration
	BackpressureMode KafkaBackpressureMode
	LogMessages      bool
}

type KafkaConsumer struct {
	AddrSlice        []string
	Topic            string
	client           sarama.Consumer
	ClientID         string
	processFunc      HandleFunc
	messageFunc      HandleMessageFunc
	pool             *su_util.GoPool
	ctx              context.Context
	cancel           func()
	closeOnce        sync.Once
	wg               sync.WaitGroup
	poolMu           sync.Mutex
	workerNum        uint32
	queueSize        uint32
	closeTimeout     time.Duration
	backpressureMode KafkaBackpressureMode
	logMessages      bool
}

func NewKafkaConsumer(a_addr []string, a_topic, a_cli_id string, a_func HandleFunc) *KafkaConsumer {
	kc := NewKafkaConsumerWithConfig(KafkaConsumerConfig{
		AddrSlice: a_addr,
		Topic:     a_topic,
		ClientID:  a_cli_id,
	}, nil)
	if kc != nil {
		kc.processFunc = a_func
	}
	return kc
}

func NewKafkaConsumerWithConfig(cfg KafkaConsumerConfig, handler HandleMessageFunc) *KafkaConsumer {
	cfg = defaultKafkaConsumerConfig(cfg)
	config := sarama.NewConfig()
	config.ClientID = cfg.ClientID
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Offsets.AutoCommit.Interval = 10 * time.Second
	config.Consumer.MaxProcessingTime = 10 * time.Second

	kc := &KafkaConsumer{
		AddrSlice:        cfg.AddrSlice,
		Topic:            cfg.Topic,
		ClientID:         cfg.ClientID,
		messageFunc:      handler,
		workerNum:        cfg.WorkerNum,
		queueSize:        cfg.QueueSize,
		closeTimeout:     cfg.CloseTimeout,
		backpressureMode: cfg.BackpressureMode,
		logMessages:      cfg.LogMessages,
	}
	kc.ctx, kc.cancel = context.WithCancel(context.Background())
	var err error
	kc.client, err = sarama.NewConsumer(cfg.AddrSlice, config)
	if err != nil {
		slog.Error("kafka NewConsumer failed", zap.Error(err))
		return nil
	}
	if cfg.WorkerNum > 0 {
		kc.ensurePool(0)
	}
	return kc
}

func (kc *KafkaConsumer) ConsumeAllPartion() {
	if kc == nil || kc.client == nil {
		slog.Error("kafka consumer is not connected")
		return
	}
	partitionList, err := kc.client.Partitions(kc.Topic)
	if err != nil {
		slog.Error("kafka get paritions failed", zap.Error(err))
		return
	}
	kc.ensurePool(len(partitionList))
	for _, partionID := range partitionList {
		kc.ConsumeOnePartion(partionID)
	}
}

func (kc *KafkaConsumer) ConsumeOnePartion(a_partion_id int32) {
	if kc == nil || kc.client == nil {
		slog.Error("kafka consumer is not connected")
		return
	}
	kc.ensurePool(0)
	pc, err := kc.client.ConsumePartition(kc.Topic, a_partion_id, sarama.OffsetNewest)
	if err != nil {
		slog.Error("failed to start consumer for partition", zap.Error(err))
		return
	}
	kc.wg.Add(1)
	go func() {
		defer kc.wg.Done()
		defer su_util.RecoverPanic()
		defer pc.AsyncClose()
		for {
			select {
			case <-kc.ctx.Done():
				if kc.logMessages {
					slog.Info("消费一个分区结束", zap.Int32("partion_id", a_partion_id))
				}
				return
			case err, ok := <-pc.Errors():
				if !ok {
					return
				}
				if err != nil {
					slog.Error("kafka consume partition error", zap.Error(err), zap.Int32("partion_id", a_partion_id))
				}
			case msg, ok := <-pc.Messages():
				if !ok {
					if kc.logMessages {
						slog.Info("消费一个分区结束", zap.Int32("partion_id", a_partion_id))
					}
					return
				}
				if kc.logMessages {
					slog.Info("kafka message", zap.Int32("partition", msg.Partition), zap.Int64("offset", msg.Offset))
				}
				kc.dispatchMessage(msg, a_partion_id)
			}
		}
	}()
}

func (kc *KafkaConsumer) Close() {
	if kc == nil {
		return
	}
	kc.closeOnce.Do(func() {
		if kc.cancel != nil {
			kc.cancel()
		}
		if kc.client != nil {
			_ = kc.client.Close()
		}
		done := make(chan struct{})
		go func() {
			kc.wg.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(kc.closeTimeout):
		}
		kc.poolMu.Lock()
		pool := kc.pool
		kc.poolMu.Unlock()
		if pool != nil {
			if !pool.StopAndDrain(kc.closeTimeout) {
				slog.Warn("kafka consumer pool drain timeout")
			}
		}
	})
}

func (kc *KafkaConsumer) dispatchMessage(msg *sarama.ConsumerMessage, partionID int32) {
	kc.poolMu.Lock()
	pool := kc.pool
	kc.poolMu.Unlock()
	if pool == nil || msg == nil {
		return
	}
	task := func() {
		if kc.messageFunc != nil {
			if err := kc.messageFunc(kc.ctx, msg); err != nil {
				slog.Error("kafka message handler failed", zap.Error(err), zap.Int32("partition", msg.Partition), zap.Int64("offset", msg.Offset))
			}
			return
		}
		if kc.processFunc != nil {
			kc.processFunc(partionID, msg)
		}
	}
	shardingID := uint64(msg.Offset)
	if msg.Offset < 0 {
		shardingID = uint64(partionID)
	}
	if kc.backpressureMode == KafkaBackpressureDrop {
		if !pool.TrySendTask(shardingID, task) {
			slog.Warn("kafka consumer task dropped", zap.Int32("partition", partionID), zap.Int64("offset", msg.Offset))
		}
		return
	}
	if !pool.SendTask(shardingID, task) {
		slog.Warn("kafka consumer task rejected", zap.Int32("partition", partionID), zap.Int64("offset", msg.Offset))
	}
}

func (kc *KafkaConsumer) ensurePool(partitionCount int) {
	kc.poolMu.Lock()
	defer kc.poolMu.Unlock()
	if kc.pool != nil {
		return
	}
	workerNum := kc.workerNum
	if workerNum == 0 {
		workerNum = uint32(runtime.NumCPU())
		if partitionCount > int(workerNum) {
			workerNum = uint32(partitionCount)
		}
		if workerNum == 0 {
			workerNum = 1
		}
		kc.workerNum = workerNum
	}
	kc.pool = su_util.NewGoPool(workerNum, kc.queueSize)
}

func defaultKafkaProducerConfig(cfg KafkaProducerConfig) KafkaProducerConfig {
	if cfg.ChannelBufferSize <= 0 {
		cfg.ChannelBufferSize = 2000
	}
	return cfg
}

func defaultKafkaConsumerConfig(cfg KafkaConsumerConfig) KafkaConsumerConfig {
	if cfg.QueueSize == 0 {
		cfg.QueueSize = 4096
	}
	if cfg.CloseTimeout <= 0 {
		cfg.CloseTimeout = 5 * time.Second
	}
	return cfg
}
