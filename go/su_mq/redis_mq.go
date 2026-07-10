package su_mq

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/garyburd/redigo/redis"
	sredis "go.local/su_da/redis"
	"go.local/su_errors"
	slog "go.local/su_log"
	"go.uber.org/zap"
)

// RedisBackpressureMode 定义 Redis list 消费队列满时的背压策略。
type RedisBackpressureMode int

const (
	RedisBackpressureBlock RedisBackpressureMode = iota
	RedisBackpressureDrop
)

// RedisListConsumerConfig 定义 Redis list consumer 的连接、并发和处理器配置。
type RedisListConsumerConfig struct {
	RedisConfig      sredis.RedisConfig    // 底层 Redis client 配置。
	ListKey          string                // BRPOP 消费的 Redis list key。
	ReaderNum        int                   // 并发 BRPOP reader 数量。
	WorkerNum        int                   // 消息处理 worker 数量。
	QueueSize        int                   // reader 到 worker 的内存队列大小。
	PopTimeout       time.Duration         // BRPOP 阻塞超时。
	CloseTimeout     time.Duration         // Close 等待读/写 goroutine 的超时。
	RetryInterval    time.Duration         // BRPOP 失败后的重试间隔。
	BackpressureMode RedisBackpressureMode // worker 队列满时的背压策略。
	LogMessages      bool                  // 是否记录每条消息日志。
	RetryPolicy      RetryPolicy           // 业务 handler 失败后的重试策略。
	DeadLetter       DeadLetter            // 最终失败后的死信发布器。
	Idempotency      Idempotency           // 消息幂等检查和标记器。
	Metrics          MQMetrics             // 消费指标回调。
}

// RedisListMessage 表示从 Redis list 弹出的一条消息。
type RedisListMessage struct {
	ListKey string // 来源 Redis list key。
	Value   []byte // 消息 payload。
}

// RedisListHandler 处理一条 Redis list 消息。
type RedisListHandler func(ctx context.Context, msg RedisListMessage) error

// redisDoCloser 抽象 Redis client，便于注入真实 client 或测试 fake。
type redisDoCloser interface {
	Do(cmd string, args ...interface{}) (interface{}, error)
	Close() error
}

// RedisListConsumer 使用 BRPOP 从 Redis list 拉取消息并分发到 worker。
type RedisListConsumer struct {
	cfg       RedisListConsumerConfig // 当前 consumer 配置。
	client    redisDoCloser           // Redis 命令 client。
	handler   RedisListHandler        // 业务消息处理函数。
	processor *Processor              // 通用消息处理器。

	ctx    context.Context       // consumer 生命周期上下文。
	cancel context.CancelFunc    // 取消 consumer 生命周期上下文。
	jobs   chan RedisListMessage // reader 到 worker 的消息队列。

	mu         sync.Mutex   // 保护 started/closed 状态。
	started    bool         // 是否已启动。
	closed     bool         // 是否已关闭。
	jobMu      sync.RWMutex // 保护 jobs channel 关闭状态。
	jobsClosed bool         // jobs channel 是否已关闭。

	readerWg  sync.WaitGroup // 等待 reader goroutine 退出。
	workerWg  sync.WaitGroup // 等待 worker goroutine 退出。
	closeOnce sync.Once      // 保证 Close 只执行一次。
	closeErr  error          // Close 返回的底层错误。
}

// NewRedisListConsumer 创建并连接 Redis client 后构造 list consumer。
func NewRedisListConsumer(cfg RedisListConsumerConfig, handler RedisListHandler) (*RedisListConsumer, error) {
	cfg = defaultRedisListConsumerConfig(cfg)
	if cfg.RedisConfig.RemoteAddr == "" {
		return nil, su_errors.New(su_errors.CodeInvalidArgument, "redis remote addr is empty")
	}
	client, err := sredis.NewRedisClientWithConfig(cfg.RedisConfig)
	if err != nil {
		return nil, su_errors.Wrap(su_errors.CodeInvalidArgument, "create redis client failed", err)
	}
	if err := client.Connect(); err != nil {
		return nil, su_errors.WrapRetryable(su_errors.CodeUnavailable, "connect redis failed", err)
	}
	return NewRedisListConsumerWithClient(cfg, client, handler)
}

// NewRedisListConsumerWithClient 使用外部 Redis client 构造 list consumer。
func NewRedisListConsumerWithClient(cfg RedisListConsumerConfig, client redisDoCloser, handler RedisListHandler) (*RedisListConsumer, error) {
	cfg = defaultRedisListConsumerConfig(cfg)
	if cfg.ListKey == "" {
		return nil, su_errors.New(su_errors.CodeInvalidArgument, "redis list key is empty")
	}
	if client == nil {
		return nil, su_errors.New(su_errors.CodeInvalidArgument, "redis client is nil")
	}
	if handler == nil {
		return nil, su_errors.New(su_errors.CodeInvalidArgument, "redis list handler is nil")
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &RedisListConsumer{
		cfg:       cfg,
		client:    client,
		handler:   handler,
		processor: NewProcessor(processorOptionsFromRedisConfig(cfg)),
		ctx:       ctx,
		cancel:    cancel,
	}, nil
}

// Start 启动 reader 和 worker goroutine；重复启动会返回错误。
func (rc *RedisListConsumer) Start() error {
	if rc == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "redis list consumer is nil")
	}
	rc.mu.Lock()
	if rc.closed {
		rc.mu.Unlock()
		return su_errors.New(su_errors.CodeInternal, "redis list consumer is closed")
	}
	if rc.started {
		rc.mu.Unlock()
		return su_errors.New(su_errors.CodeInternal, "redis list consumer already started")
	}
	rc.started = true
	rc.jobs = make(chan RedisListMessage, rc.cfg.QueueSize)
	rc.mu.Unlock()

	for i := 0; i < rc.cfg.WorkerNum; i++ {
		rc.workerWg.Add(1)
		go rc.runWorker(i)
	}
	for i := 0; i < rc.cfg.ReaderNum; i++ {
		rc.readerWg.Add(1)
		go rc.readLoop(i)
	}
	return nil
}

// Close 停止 reader、关闭任务队列、等待 worker 退出并关闭 Redis client。
func (rc *RedisListConsumer) Close() error {
	if rc == nil {
		return nil
	}
	rc.closeOnce.Do(func() {
		rc.mu.Lock()
		rc.closed = true
		started := rc.started
		rc.mu.Unlock()
		if rc.cancel != nil {
			rc.cancel()
		}
		if started {
			rc.waitReaders()
			rc.closeJobs()
			rc.waitWorkers()
		}
		if rc.client != nil {
			rc.closeErr = rc.client.Close()
		}
	})
	return rc.closeErr
}

// readLoop 持续执行 BRPOP，失败后按 RetryInterval 等待重试。
func (rc *RedisListConsumer) readLoop(index int) {
	defer rc.readerWg.Done()
	timeoutSeconds := int(rc.cfg.PopTimeout / time.Second)
	if timeoutSeconds <= 0 {
		timeoutSeconds = 1
	}
	for {
		select {
		case <-rc.ctx.Done():
			return
		default:
		}
		reply, err := rc.client.Do("BRPOP", rc.cfg.ListKey, timeoutSeconds)
		if err != nil {
			select {
			case <-rc.ctx.Done():
				return
			case <-time.After(rc.cfg.RetryInterval):
			}
			slog.Error("redis list brpop failed", zap.Error(err), zap.Int("reader", index), zap.String("list", rc.cfg.ListKey))
			continue
		}
		if reply == nil {
			continue
		}
		msg, err := parseRedisListMessage(reply)
		if err != nil {
			slog.Error("redis list parse message failed", zap.Error(err), zap.Int("reader", index), zap.String("list", rc.cfg.ListKey))
			continue
		}
		select {
		case <-rc.ctx.Done():
			return
		default:
		}
		if rc.cfg.LogMessages {
			slog.Info("redis list message", zap.String("list", msg.ListKey), zap.Int("bytes", len(msg.Value)))
		}
		rc.dispatch(msg)
	}
}

// runWorker 从任务队列取消息，并通过 Processor 执行业务 handler。
func (rc *RedisListConsumer) runWorker(index int) {
	defer rc.workerWg.Done()
	for msg := range rc.jobs {
		mqMsg := Message{
			Source: "redis",
			Topic:  msg.ListKey,
			Value:  append([]byte(nil), msg.Value...),
			Raw:    msg,
		}
		err := rc.processor.Process(rc.ctx, mqMsg, func(ctx context.Context, message Message) error {
			return rc.handler(ctx, msg)
		})
		if err != nil {
			slog.Error("redis list handler failed", zap.Error(err), zap.Int("worker", index), zap.String("list", msg.ListKey))
		}
	}
}

// dispatch 按背压策略把消息投递到 worker 队列。
func (rc *RedisListConsumer) dispatch(msg RedisListMessage) {
	rc.jobMu.RLock()
	defer rc.jobMu.RUnlock()
	if rc.jobsClosed {
		return
	}
	if rc.cfg.BackpressureMode == RedisBackpressureDrop {
		select {
		case rc.jobs <- msg:
		default:
			slog.Warn("redis list message dropped", zap.String("list", msg.ListKey))
		}
		return
	}
	select {
	case <-rc.ctx.Done():
	case rc.jobs <- msg:
	}
}

// closeJobs 只关闭一次 worker 任务队列。
func (rc *RedisListConsumer) closeJobs() {
	rc.jobMu.Lock()
	defer rc.jobMu.Unlock()
	if rc.jobsClosed {
		return
	}
	close(rc.jobs)
	rc.jobsClosed = true
}

// waitReaders 等待读取 goroutine 退出，超过 CloseTimeout 后记录告警。
func (rc *RedisListConsumer) waitReaders() {
	done := make(chan struct{})
	go func() {
		rc.readerWg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(rc.cfg.CloseTimeout):
		slog.Warn("redis list wait readers timeout")
	}
}

// waitWorkers 等待 worker goroutine 退出，超过 CloseTimeout 后记录告警。
func (rc *RedisListConsumer) waitWorkers() {
	done := make(chan struct{})
	go func() {
		rc.workerWg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(rc.cfg.CloseTimeout):
		slog.Warn("redis list wait workers timeout")
	}
}

// parseRedisListMessage 解析 BRPOP 返回的 list key 和 payload。
func parseRedisListMessage(reply interface{}) (RedisListMessage, error) {
	values, err := redis.Values(reply, nil)
	if err != nil {
		return RedisListMessage{}, err
	}
	if len(values) != 2 {
		return RedisListMessage{}, su_errors.New(su_errors.CodeInvalidArgument, "redis list reply length is invalid")
	}
	listKey, err := redis.String(values[0], nil)
	if err != nil {
		return RedisListMessage{}, err
	}
	value, err := redis.Bytes(values[1], nil)
	if err != nil {
		return RedisListMessage{}, err
	}
	return RedisListMessage{ListKey: listKey, Value: value}, nil
}

// defaultRedisListConsumerConfig 填充 Redis list consumer 默认并发、队列和超时配置。
func defaultRedisListConsumerConfig(cfg RedisListConsumerConfig) RedisListConsumerConfig {
	if cfg.ReaderNum <= 0 {
		cfg.ReaderNum = 1
	}
	if cfg.WorkerNum <= 0 {
		cfg.WorkerNum = runtime.NumCPU()
		if cfg.WorkerNum <= 0 {
			cfg.WorkerNum = 1
		}
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 4096
	}
	if cfg.PopTimeout <= 0 {
		cfg.PopTimeout = time.Second
	}
	if cfg.CloseTimeout <= 0 {
		cfg.CloseTimeout = 5 * time.Second
	}
	if cfg.RetryInterval <= 0 {
		cfg.RetryInterval = time.Second
	}
	return cfg
}

// processorOptionsFromRedisConfig 提取 Redis list consumer 的通用 Processor 配置。
func processorOptionsFromRedisConfig(cfg RedisListConsumerConfig) ProcessorOptions {
	return ProcessorOptions{
		RetryPolicy: cfg.RetryPolicy,
		DeadLetter:  cfg.DeadLetter,
		Idempotency: cfg.Idempotency,
		Metrics:     cfg.Metrics,
	}
}
