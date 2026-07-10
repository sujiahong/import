package su_redis

import (
	"sync"
	"time"

	"github.com/garyburd/redigo/redis"
	"go.local/su_errors"
	slog "go.local/su_log"
	"go.uber.org/zap"
)

// RedisConfig 定义 Redis 连接池、数据库和超时配置。
type RedisConfig struct {
	RemoteAddr   string        // Redis tcp 地址。
	ConnNum      int           // 兼容旧接口的连接数配置。
	MaxIdle      int           // 连接池最大空闲连接数。
	MaxActive    int           // 连接池最大活跃连接数。
	IdleTimeout  time.Duration // 空闲连接保留时间。
	DB           int           // Redis 数据库编号。
	DialTimeout  time.Duration // 建连超时。
	ReadTimeout  time.Duration // 读超时。
	WriteTimeout time.Duration // 写超时。
	Wait         bool          // 连接池耗尽时是否等待可用连接。
}

// RedisClient 封装 redigo 连接池，并提供并发安全的连接池重建和关闭能力。
type RedisClient struct {
	pool        *redis.Pool                           // 当前 redigo 连接池。
	RemoteAddr  string                                // 当前 Redis 地址。
	ConnNum     int                                   // 兼容旧接口的连接数配置。
	cfg         RedisConfig                           // 最近一次创建连接池使用的配置。
	dial        func(RedisConfig) (redis.Conn, error) // 建连函数，测试中可替换。
	mu          sync.RWMutex                          // 保护连接池指针和配置。
	reconnectMu sync.Mutex                            // 串行化 Connect/Reconnect/Close。
	closeOnce   sync.Once                             // 保证 Close 只执行一次。
	closeErr    error                                 // Close 返回的底层错误。
}

// NewRedisClient 使用地址和连接数创建 Redis client。
func NewRedisClient(redis_addr string, conn_num int) *RedisClient {
	cfg := defaultRedisConfig(RedisConfig{RemoteAddr: redis_addr, ConnNum: conn_num, Wait: true})
	cfg.RemoteAddr = redis_addr
	return &RedisClient{
		RemoteAddr: cfg.RemoteAddr,
		ConnNum:    cfg.ConnNum,
		cfg:        cfg,
		dial:       dialRedis,
	}
}

// NewRedisClientWithConfig 使用完整配置创建 Redis client。
func NewRedisClientWithConfig(cfg RedisConfig) (*RedisClient, error) {
	cfg = defaultRedisConfig(cfg)
	if cfg.RemoteAddr == "" {
		return nil, su_errors.New(su_errors.CodeInvalidArgument, "redis remote addr is empty")
	}
	return &RedisClient{
		RemoteAddr: cfg.RemoteAddr,
		ConnNum:    cfg.ConnNum,
		cfg:        cfg,
		dial:       dialRedis,
	}, nil
}

// Connect 创建新的 Redis 连接池，首次 PING 成功后发布到 client。
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
	dial := rc.dial
	if dial == nil {
		dial = dialRedis
	}
	pool := &redis.Pool{
		MaxIdle:     cfg.MaxIdle,
		MaxActive:   cfg.MaxActive,
		IdleTimeout: cfg.IdleTimeout,
		Wait:        cfg.Wait,
		Dial: func() (redis.Conn, error) {
			c, err := dial(cfg)
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
	if err := pingRedisPool(pool, cfg.RemoteAddr); err != nil {
		_ = pool.Close()
		return err
	}
	rc.mu.Lock()
	oldPool := rc.pool
	rc.pool = pool
	rc.cfg = cfg
	rc.RemoteAddr = cfg.RemoteAddr
	rc.ConnNum = cfg.ConnNum
	rc.dial = dial
	rc.closeOnce = sync.Once{}
	rc.closeErr = nil
	rc.mu.Unlock()
	if oldPool != nil {
		_ = oldPool.Close()
	}
	slog.Info("连接redis", zap.Any("RemoteAddr: ", cfg.RemoteAddr))
	return nil
}

// dialRedis 根据配置建立单条 Redis TCP 连接。
func dialRedis(cfg RedisConfig) (redis.Conn, error) {
	return redis.Dial("tcp", cfg.RemoteAddr,
		redis.DialDatabase(cfg.DB),
		redis.DialConnectTimeout(cfg.DialTimeout),
		redis.DialReadTimeout(cfg.ReadTimeout),
		redis.DialWriteTimeout(cfg.WriteTimeout),
	)
}

// pingRedisPool 从连接池借出连接并执行 PING，用于连接池发布前的健康检查。
func pingRedisPool(pool *redis.Pool, remoteAddr string) error {
	c := pool.Get()
	defer c.Close()
	if err := c.Err(); err != nil {
		slog.Error("redis initial connection failed", zap.Any("RemoteAddr: ", remoteAddr), zap.Error(err))
		return su_errors.WrapRetryable(su_errors.CodeUnavailable, "redis connection error", err)
	}
	if _, err := c.Do("PING"); err != nil {
		slog.Error("redis initial ping failed", zap.Any("RemoteAddr: ", remoteAddr), zap.Error(err))
		return su_errors.WrapRetryable(su_errors.CodeUnavailable, "redis ping failed", err)
	}
	return nil
}

// Reconnect 重建 Redis 连接池；运行时命令错误不会自动调用该方法。
func (rc *RedisClient) Reconnect() error {
	return rc.Connect()
}

// Test 使用单连接执行简单写入，主要用于手工连通性验证。
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

// Close 关闭当前 Redis 连接池，并与 Connect/Reconnect 互斥。
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

// Do 从连接池借出连接并执行 Redis 命令，连接或命令错误会包装为 retryable。
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

// defaultRedisConfig 填充 Redis 连接池和超时默认值。
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
