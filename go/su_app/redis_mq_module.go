package su_app

import (
	"context"

	"go.local/su_errors"
	"go.local/su_mq"
)

/* 使用示例
app.Register(su_app.NewRedisMQModule(su_mq.RedisListConsumerConfig{
    RedisConfig: su_redis.RedisConfig{
        RemoteAddr: "127.0.0.1:6379",
        ConnNum:    4,
    },
    ListKey:     "jobs",
    ReaderNum:   2,
    WorkerNum:   8,
    QueueSize:   4096,
    RetryPolicy: su_mq.FixedRetry{MaxAttempts: 3, Delay: time.Second},
    DeadLetter:  su_mq.NewMemoryDeadLetter(),
    Idempotency: su_mq.NewMemoryIdempotency(),
}, func(ctx context.Context, msg su_mq.RedisListMessage) error {
    // 业务处理
    return nil
}))
*/

type RedisMQConsumer interface {
	Start() error
	Close() error
}

type RedisMQFactory func(cfg su_mq.RedisListConsumerConfig, handler su_mq.RedisListHandler) (RedisMQConsumer, error)

type RedisMQModule struct {
	Config           su_mq.RedisListConsumerConfig
	Handler          su_mq.RedisListHandler
	Factory          RedisMQFactory
	Consumer         RedisMQConsumer
	DisableAutoStart bool
}

func NewRedisMQModule(cfg su_mq.RedisListConsumerConfig, handler su_mq.RedisListHandler) *RedisMQModule {
	return &RedisMQModule{Config: cfg, Handler: handler}
}

func (m *RedisMQModule) Name() string {
	return "redis_mq"
}

func (m *RedisMQModule) Start(ctx context.Context) error {
	if m == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "redis mq module is nil")
	}
	if m.Consumer == nil {
		factory := m.Factory
		if factory == nil {
			factory = defaultRedisMQFactory
		}
		consumer, err := factory(m.Config, m.Handler)
		if err != nil {
			return su_errors.WrapRetryable(su_errors.CodeUnavailable, "create redis mq consumer failed", err)
		}
		m.Consumer = consumer
	}
	if !m.DisableAutoStart {
		if err := m.Consumer.Start(); err != nil {
			return su_errors.Wrap(su_errors.CodeInternal, "start redis mq consumer failed", err)
		}
	}
	return nil
}

func (m *RedisMQModule) Stop(ctx context.Context) error {
	if m == nil || m.Consumer == nil {
		return nil
	}
	if err := m.Consumer.Close(); err != nil {
		return su_errors.Wrap(su_errors.CodeInternal, "close redis mq consumer failed", err)
	}
	return nil
}

func defaultRedisMQFactory(cfg su_mq.RedisListConsumerConfig, handler su_mq.RedisListHandler) (RedisMQConsumer, error) {
	return su_mq.NewRedisListConsumer(cfg, handler)
}
