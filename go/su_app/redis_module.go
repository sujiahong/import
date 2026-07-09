package su_app

import (
	"context"

	su_redis "go.local/su_da/redis"
	"go.local/su_errors"
)

type RedisConnector interface {
	Connect() error
	Close() error
}

type RedisFactory func(cfg su_redis.RedisConfig) (RedisConnector, error)

type RedisModule struct {
	Config  su_redis.RedisConfig
	Factory RedisFactory
	Client  RedisConnector
}

func NewRedisModule(cfg su_redis.RedisConfig) *RedisModule {
	return &RedisModule{Config: cfg}
}

func (m *RedisModule) Name() string {
	return "redis"
}

func (m *RedisModule) Start(ctx context.Context) error {
	if m == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "redis module is nil")
	}
	if m.Client == nil {
		factory := m.Factory
		if factory == nil {
			factory = defaultRedisFactory
		}
		client, err := factory(m.Config)
		if err != nil {
			return su_errors.WrapRetryable(su_errors.CodeUnavailable, "create redis client failed", err)
		}
		m.Client = client
	}
	if err := m.Client.Connect(); err != nil {
		return su_errors.WrapRetryable(su_errors.CodeUnavailable, "connect redis failed", err)
	}
	return nil
}

func (m *RedisModule) Stop(ctx context.Context) error {
	if m == nil || m.Client == nil {
		return nil
	}
	if err := m.Client.Close(); err != nil {
		return su_errors.Wrap(su_errors.CodeInternal, "close redis failed", err)
	}
	return nil
}

func defaultRedisFactory(cfg su_redis.RedisConfig) (RedisConnector, error) {
	return su_redis.NewRedisClientWithConfig(cfg)
}
