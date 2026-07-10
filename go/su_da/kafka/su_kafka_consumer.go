package su_kafka

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"go.local/su_errors"
	slog "go.local/su_log"
	"go.local/su_util"
	"go.uber.org/zap"
)

// KafkaBackpressureMode 定义消费处理队列满时的背压策略。
type KafkaBackpressureMode int

const (
	KafkaBackpressureBlock KafkaBackpressureMode = iota
	KafkaBackpressureDrop
)

var newSaramaConsumer = sarama.NewConsumer

// HandleFunc 是旧版按分区回调的 Kafka 消息处理函数。
type HandleFunc func(a_partion_id int32, msg *sarama.ConsumerMessage)

// HandleMessageFunc 是新版带 context 的 Kafka 消息处理函数。
type HandleMessageFunc func(ctx context.Context, msg *sarama.ConsumerMessage) error

// KafkaConsumerConfig 定义 Kafka consumer 的连接、并发、关闭和背压配置。
type KafkaConsumerConfig struct {
	AddrSlice        []string              // Kafka broker 地址列表。
	Topic            string                // 订阅的 topic 名称。
	ClientID         string                // Sarama consumer client id。
	WorkerNum        uint32                // 消息处理 worker 数量，0 表示按 CPU/分区数自动设置。
	QueueSize        uint32                // worker 池任务队列大小。
	CloseTimeout     time.Duration         // Close 等待 goroutine 和 worker 池退出的最长时间。
	RetryInterval    time.Duration         // 分区订阅失败或断开后的重试间隔。
	BackpressureMode KafkaBackpressureMode // worker 队列满时的背压策略。
	LogMessages      bool                  // 是否记录消费和关闭过程的详细日志。
}

// KafkaConsumer 封装 sarama.Consumer，并管理分区消费 goroutine 与处理 worker 池。
type KafkaConsumer struct {
	AddrSlice        []string              // Kafka broker 地址列表。
	Topic            string                // 当前消费的 topic。
	client           sarama.Consumer       // 当前底层 Sarama consumer。
	cfg              KafkaConsumerConfig   // 最近一次创建 consumer 使用的配置。
	mu               sync.RWMutex          // 保护 client 读写和重连交换。
	reconnectMu      sync.Mutex            // 串行化 consumer 重连流程。
	ClientID         string                // Sarama consumer client id。
	processFunc      HandleFunc            // 旧版分区回调。
	messageFunc      HandleMessageFunc     // 新版带 context 的消息回调。
	pool             *su_util.GoPool       // 消息处理 worker 池。
	ctx              context.Context       // consumer 生命周期上下文。
	cancel           func()                // 取消 consumer 生命周期上下文。
	closeOnce        sync.Once             // 保证 Close 只执行一次。
	wg               sync.WaitGroup        // 等待分区消费 goroutine 退出。
	poolMu           sync.Mutex            // 保护 worker 池惰性创建。
	workerNum        uint32                // 实际使用的 worker 数量。
	queueSize        uint32                // worker 池任务队列大小。
	closeTimeout     time.Duration         // 关闭等待超时。
	retryInterval    time.Duration         // 分区重订阅等待间隔。
	backpressureMode KafkaBackpressureMode // worker 队列满时的背压策略。
	logMessages      bool                  // 是否记录详细消费日志。
}

// NewKafkaConsumer 使用旧版分区回调创建 Kafka consumer。
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

// NewKafkaConsumerWithConfig 使用完整配置和新版消息回调创建 Kafka consumer。
func NewKafkaConsumerWithConfig(cfg KafkaConsumerConfig, handler HandleMessageFunc) *KafkaConsumer {
	cfg = defaultKafkaConsumerConfig(cfg)
	client, err := newKafkaConsumerClient(cfg)
	if err != nil {
		slog.Error("kafka NewConsumer failed", zap.Error(err))
		return nil
	}

	kc := &KafkaConsumer{
		AddrSlice:        cfg.AddrSlice,
		Topic:            cfg.Topic,
		client:           client,
		cfg:              cfg,
		ClientID:         cfg.ClientID,
		messageFunc:      handler,
		workerNum:        cfg.WorkerNum,
		queueSize:        cfg.QueueSize,
		closeTimeout:     cfg.CloseTimeout,
		retryInterval:    cfg.RetryInterval,
		backpressureMode: cfg.BackpressureMode,
		logMessages:      cfg.LogMessages,
	}
	kc.ctx, kc.cancel = context.WithCancel(context.Background())
	if cfg.WorkerNum > 0 {
		kc.ensurePool(0)
	}
	return kc
}

// ConsumeAllPartion 获取 topic 的所有分区并为每个分区启动消费循环。
func (kc *KafkaConsumer) ConsumeAllPartion() {
	client := kc.getClient()
	if kc == nil || client == nil {
		slog.Error("kafka consumer is not connected")
		return
	}
	partitionList, err := client.Partitions(kc.Topic)
	if err != nil {
		slog.Error("kafka get paritions failed", zap.Error(err))
		if reconnectErr := kc.reconnectConsumer(); reconnectErr != nil {
			slog.Error("kafka consumer reconnect failed", zap.Error(reconnectErr))
			return
		}
		client = kc.getClient()
		if client == nil {
			return
		}
		partitionList, err = client.Partitions(kc.Topic)
		if err != nil {
			slog.Error("kafka get paritions failed after reconnect", zap.Error(err))
			return
		}
	}
	kc.ensurePool(len(partitionList))
	for _, partionID := range partitionList {
		kc.ConsumeOnePartion(partionID)
	}
}

// ConsumeOnePartion 为指定分区启动独立 goroutine，断线或通道关闭后按配置重试订阅。
func (kc *KafkaConsumer) ConsumeOnePartion(a_partion_id int32) {
	if kc == nil || kc.getClient() == nil {
		slog.Error("kafka consumer is not connected")
		return
	}
	kc.ensurePool(0)
	kc.wg.Add(1)
	go func() {
		defer kc.wg.Done()
		defer su_util.RecoverPanic()
		for {
			client := kc.getClient()
			if client == nil {
				if !kc.waitRetry(a_partion_id) {
					return
				}
				continue
			}
			pc, err := client.ConsumePartition(kc.Topic, a_partion_id, sarama.OffsetNewest)
			if err != nil {
				slog.Error("failed to start consumer for partition", zap.Error(err), zap.Int32("partion_id", a_partion_id))
				if reconnectErr := kc.reconnectConsumer(); reconnectErr != nil {
					slog.Error("kafka consumer reconnect failed", zap.Error(reconnectErr), zap.Int32("partion_id", a_partion_id))
				}
				if !kc.waitRetry(a_partion_id) {
					return
				}
				continue
			}
			if !kc.consumePartitionMessages(pc, a_partion_id) {
				return
			}
			if !kc.waitRetry(a_partion_id) {
				return
			}
		}
	}()
}

// consumePartitionMessages 读取分区消息与错误；返回 true 表示需要外层重建 PartitionConsumer。
func (kc *KafkaConsumer) consumePartitionMessages(pc sarama.PartitionConsumer, a_partion_id int32) bool {
	defer pc.AsyncClose()
	for {
		select {
		case <-kc.ctx.Done():
			if kc.logMessages {
				slog.Info("消费一个分区结束", zap.Int32("partion_id", a_partion_id))
			}
			return false
		case err, ok := <-pc.Errors():
			if !ok {
				return true
			}
			if err != nil {
				slog.Error("kafka consume partition error", zap.Error(err), zap.Int32("partion_id", a_partion_id))
			}
		case msg, ok := <-pc.Messages():
			if !ok {
				if kc.logMessages {
					slog.Info("消费分区通道关闭，准备重订阅", zap.Int32("partion_id", a_partion_id))
				}
				return true
			}
			if kc.logMessages {
				slog.Info("kafka message", zap.Int32("partition", msg.Partition), zap.Int64("offset", msg.Offset))
			}
			kc.dispatchMessage(msg, a_partion_id)
		}
	}
}

// waitRetry 在重试间隔或 consumer 关闭之间等待，返回 false 表示应停止消费循环。
func (kc *KafkaConsumer) waitRetry(partitionID int32) bool {
	timer := time.NewTimer(kc.retryInterval)
	defer timer.Stop()
	select {
	case <-kc.ctx.Done():
		if kc.logMessages {
			slog.Info("消费一个分区结束", zap.Int32("partion_id", partitionID))
		}
		return false
	case <-timer.C:
		return true
	}
}

// getClient 并发安全地返回当前 sarama consumer。
func (kc *KafkaConsumer) getClient() sarama.Consumer {
	if kc == nil {
		return nil
	}
	kc.mu.RLock()
	defer kc.mu.RUnlock()
	return kc.client
}

// reconnectConsumer 重建 sarama consumer，并在交换完成后关闭旧 client。
func (kc *KafkaConsumer) reconnectConsumer() error {
	if kc == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "kafka consumer is nil")
	}
	kc.reconnectMu.Lock()
	defer kc.reconnectMu.Unlock()
	select {
	case <-kc.ctx.Done():
		return su_errors.New(su_errors.CodeInternal, "kafka consumer is closed")
	default:
	}
	client, err := newKafkaConsumerClient(kc.cfg)
	if err != nil {
		return su_errors.WrapRetryable(su_errors.CodeUnavailable, "kafka consumer reconnect failed", err)
	}
	select {
	case <-kc.ctx.Done():
		_ = client.Close()
		return su_errors.New(su_errors.CodeInternal, "kafka consumer is closed")
	default:
	}
	kc.mu.Lock()
	oldClient := kc.client
	kc.client = client
	kc.mu.Unlock()
	if oldClient != nil {
		_ = oldClient.Close()
	}
	return nil
}

// Close 停止消费、关闭 sarama consumer，并等待分区 goroutine 和 worker 池退出。
func (kc *KafkaConsumer) Close() {
	if kc == nil {
		return
	}
	kc.closeOnce.Do(func() {
		if kc.cancel != nil {
			kc.cancel()
		}
		kc.reconnectMu.Lock()
		defer kc.reconnectMu.Unlock()
		kc.mu.Lock()
		client := kc.client
		kc.client = nil
		kc.mu.Unlock()
		if client != nil {
			_ = client.Close()
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

// newKafkaConsumerClient 根据配置创建底层 sarama consumer。
func newKafkaConsumerClient(cfg KafkaConsumerConfig) (sarama.Consumer, error) {
	config := sarama.NewConfig()
	config.ClientID = cfg.ClientID
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Offsets.AutoCommit.Interval = 10 * time.Second
	config.Consumer.MaxProcessingTime = 10 * time.Second
	return newSaramaConsumer(cfg.AddrSlice, config)
}

// dispatchMessage 将 Kafka 消息投递到 worker 池，并按配置选择阻塞或丢弃策略。
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

// ensurePool 按分区数量和配置惰性创建消息处理 worker 池。
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

// defaultKafkaConsumerConfig 填充 Kafka consumer 的默认队列、关闭和重试配置。
func defaultKafkaConsumerConfig(cfg KafkaConsumerConfig) KafkaConsumerConfig {
	if cfg.QueueSize == 0 {
		cfg.QueueSize = 4096
	}
	if cfg.CloseTimeout <= 0 {
		cfg.CloseTimeout = 5 * time.Second
	}
	if cfg.RetryInterval <= 0 {
		cfg.RetryInterval = time.Second
	}
	return cfg
}
