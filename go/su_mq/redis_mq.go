package su_mq

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"time"

	"github.com/garyburd/redigo/redis"
	sredis "go.local/su_da/redis"
	slog "go.local/su_log"
	"go.uber.org/zap"
)

type RedisBackpressureMode int

const (
	RedisBackpressureBlock RedisBackpressureMode = iota
	RedisBackpressureDrop
)

type RedisListConsumerConfig struct {
	RedisConfig      sredis.RedisConfig
	ListKey          string
	ReaderNum        int
	WorkerNum        int
	QueueSize        int
	PopTimeout       time.Duration
	CloseTimeout     time.Duration
	RetryInterval    time.Duration
	BackpressureMode RedisBackpressureMode
	LogMessages      bool
}

type RedisListMessage struct {
	ListKey string
	Value   []byte
}

type RedisListHandler func(ctx context.Context, msg RedisListMessage) error

type redisDoCloser interface {
	Do(cmd string, args ...interface{}) (interface{}, error)
	Close() error
}

type RedisListConsumer struct {
	cfg     RedisListConsumerConfig
	client  redisDoCloser
	handler RedisListHandler

	ctx    context.Context
	cancel context.CancelFunc
	jobs   chan RedisListMessage

	mu         sync.Mutex
	started    bool
	closed     bool
	jobMu      sync.RWMutex
	jobsClosed bool

	readerWg  sync.WaitGroup
	workerWg  sync.WaitGroup
	closeOnce sync.Once
	closeErr  error
}

func NewRedisListConsumer(cfg RedisListConsumerConfig, handler RedisListHandler) (*RedisListConsumer, error) {
	cfg = defaultRedisListConsumerConfig(cfg)
	if cfg.RedisConfig.RemoteAddr == "" {
		return nil, errors.New("redis remote addr is empty")
	}
	client, err := sredis.NewRedisClientWithConfig(cfg.RedisConfig)
	if err != nil {
		return nil, err
	}
	if err := client.Connect(); err != nil {
		return nil, err
	}
	return NewRedisListConsumerWithClient(cfg, client, handler)
}

func NewRedisListConsumerWithClient(cfg RedisListConsumerConfig, client redisDoCloser, handler RedisListHandler) (*RedisListConsumer, error) {
	cfg = defaultRedisListConsumerConfig(cfg)
	if cfg.ListKey == "" {
		return nil, errors.New("redis list key is empty")
	}
	if client == nil {
		return nil, errors.New("redis client is nil")
	}
	if handler == nil {
		return nil, errors.New("redis list handler is nil")
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &RedisListConsumer{
		cfg:     cfg,
		client:  client,
		handler: handler,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

func (rc *RedisListConsumer) Start() error {
	if rc == nil {
		return errors.New("redis list consumer is nil")
	}
	rc.mu.Lock()
	if rc.closed {
		rc.mu.Unlock()
		return errors.New("redis list consumer is closed")
	}
	if rc.started {
		rc.mu.Unlock()
		return errors.New("redis list consumer already started")
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

func (rc *RedisListConsumer) runWorker(index int) {
	defer rc.workerWg.Done()
	for msg := range rc.jobs {
		if err := rc.handler(rc.ctx, msg); err != nil {
			slog.Error("redis list handler failed", zap.Error(err), zap.Int("worker", index), zap.String("list", msg.ListKey))
		}
	}
}

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

func (rc *RedisListConsumer) closeJobs() {
	rc.jobMu.Lock()
	defer rc.jobMu.Unlock()
	if rc.jobsClosed {
		return
	}
	close(rc.jobs)
	rc.jobsClosed = true
}

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

func parseRedisListMessage(reply interface{}) (RedisListMessage, error) {
	values, err := redis.Values(reply, nil)
	if err != nil {
		return RedisListMessage{}, err
	}
	if len(values) != 2 {
		return RedisListMessage{}, errors.New("redis list reply length is invalid")
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
