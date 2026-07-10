package su_redis

import (
	"sync"
	"time"

	"github.com/garyburd/redigo/redis"
	"go.local/su_errors"
	slog "go.local/su_log"
	"go.uber.org/zap"
)

type RedisConfig struct {
	RemoteAddr   string
	ConnNum      int
	MaxIdle      int
	MaxActive    int
	IdleTimeout  time.Duration
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	Wait         bool
}

type RedisClient struct {
	pool        *redis.Pool /////redis连接池
	RemoteAddr  string
	ConnNum     int
	cfg         RedisConfig
	mu          sync.RWMutex
	reconnectMu sync.Mutex
	closeOnce   sync.Once
	closeErr    error
}

func NewRedisClient(redis_addr string, conn_num int) *RedisClient {
	cfg := defaultRedisConfig(RedisConfig{RemoteAddr: redis_addr, ConnNum: conn_num, Wait: true})
	cfg.RemoteAddr = redis_addr
	return &RedisClient{
		RemoteAddr: cfg.RemoteAddr,
		ConnNum:    cfg.ConnNum,
		cfg:        cfg,
	}
}

func NewRedisClientWithConfig(cfg RedisConfig) (*RedisClient, error) {
	cfg = defaultRedisConfig(cfg)
	if cfg.RemoteAddr == "" {
		return nil, su_errors.New(su_errors.CodeInvalidArgument, "redis remote addr is empty")
	}
	return &RedisClient{
		RemoteAddr: cfg.RemoteAddr,
		ConnNum:    cfg.ConnNum,
		cfg:        cfg,
	}, nil
}

func (rc *RedisClient) Connect() error {
	if rc == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "redis client is nil")
	}
	rc.reconnectMu.Lock()
	defer rc.reconnectMu.Unlock()
	cfg := defaultRedisConfig(rc.cfg)
	if cfg.RemoteAddr == "" {
		cfg.RemoteAddr = rc.RemoteAddr
	}
	if cfg.RemoteAddr == "" {
		return su_errors.New(su_errors.CodeInvalidArgument, "redis remote addr is empty")
	}
	pool := &redis.Pool{
		MaxIdle:     cfg.MaxIdle,
		MaxActive:   cfg.MaxActive,
		IdleTimeout: cfg.IdleTimeout,
		Wait:        cfg.Wait,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", cfg.RemoteAddr,
				redis.DialDatabase(cfg.DB),
				redis.DialConnectTimeout(cfg.DialTimeout),
				redis.DialReadTimeout(cfg.ReadTimeout),
				redis.DialWriteTimeout(cfg.WriteTimeout),
			)
			slog.Info("dial ...... ", zap.Any("RemoteAddr: ", cfg.RemoteAddr), zap.Error(err))
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			slog.Info("Ping. ", zap.Any("RemoteAddr: ", rc.RemoteAddr), zap.Error(err))
			return err
		},
	}
	rc.mu.Lock()
	oldPool := rc.pool
	rc.pool = pool
	rc.cfg = cfg
	rc.RemoteAddr = cfg.RemoteAddr
	rc.ConnNum = cfg.ConnNum
	rc.closeOnce = sync.Once{}
	rc.closeErr = nil
	rc.mu.Unlock()
	if oldPool != nil {
		_ = oldPool.Close()
	}
	slog.Info("连接redis", zap.Any("RemoteAddr: ", cfg.RemoteAddr))
	return nil
}

func (rc *RedisClient) Reconnect() error {
	return rc.Connect()
}

func (rc *RedisClient) Test() {
	c, err := redis.Dial("tcp", rc.RemoteAddr)
	if err != nil {
		slog.Info("dial ConnectSingle ...... ", zap.Any("RemoteAddr: ", rc.RemoteAddr), zap.Error(err))
		return
	}
	defer c.Close()
	slog.Info("连接redis ConnectSingle", zap.Any("RemoteAddr: ", rc.RemoteAddr))
	_, err = c.Do("set", "aa", 12)
	slog.Info("redis  set", zap.Error(err))
	return
}

func (rc *RedisClient) Close() error {
	if rc == nil {
		return nil
	}
	rc.reconnectMu.Lock()
	defer rc.reconnectMu.Unlock()
	rc.closeOnce.Do(func() {
		rc.mu.Lock()
		if rc.pool == nil {
			rc.mu.Unlock()
			return
		}
		pool := rc.pool
		rc.pool = nil
		rc.mu.Unlock()
		rc.closeErr = pool.Close()
	})
	return rc.closeErr
}

func (rc *RedisClient) Do(cmd string, args ...interface{}) (interface{}, error) {
	if rc == nil {
		return nil, su_errors.New(su_errors.CodeUnavailable, "redis client is not connected")
	}
	rc.mu.RLock()
	pool := rc.pool
	rc.mu.RUnlock()
	if pool == nil {
		return nil, su_errors.New(su_errors.CodeUnavailable, "redis client is not connected")
	}
	c := pool.Get()
	defer c.Close()
	if err := c.Err(); err != nil {
		return nil, su_errors.WrapRetryable(su_errors.CodeUnavailable, "redis connection error", err)
	}
	reply, err := c.Do(cmd, args...)
	if err != nil {
		return nil, su_errors.WrapRetryable(su_errors.CodeUnavailable, "redis command failed", err)
	}
	return reply, nil
}

func defaultRedisConfig(cfg RedisConfig) RedisConfig {
	if cfg.ConnNum <= 0 {
		cfg.ConnNum = 1
	}
	if cfg.MaxIdle <= 0 {
		cfg.MaxIdle = cfg.ConnNum
	}
	if cfg.MaxActive <= 0 {
		cfg.MaxActive = cfg.ConnNum
	}
	if cfg.IdleTimeout <= 0 {
		cfg.IdleTimeout = 30 * time.Second
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 5 * time.Second
	}
	if cfg.ReadTimeout <= 0 {
		cfg.ReadTimeout = 5 * time.Second
	}
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = 5 * time.Second
	}
	return cfg
}
