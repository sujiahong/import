package su_redis

import (
	"github.com/garyburd/redigo/redis"
	slog "go/su_log"
	"go.uber.org/zap"
	"time"
)
type RedisClient struct {
	pool *redis.Pool   /////redis连接池
	RemoteAddr string
	ConnNum  int
}

func NewRedisClient(redis_addr string, conn_num int) *RedisClient{
	return &RedisClient{RemoteAddr: redis_addr, ConnNum: conn_num}
}

func (rc *RedisClient)Connect()  {
	rc.pool = &redis.Pool{
		MaxIdle: rc.ConnNum,
		MaxActive:3,
		IdleTimeout:30,
		Wait: true,
		Dial: func()(redis.Conn, error){
			c, err := redis.Dial("tcp", rc.RemoteAddr, redis.DialDatabase(1))
			slog.Info("dial ...... ", zap.Any("RemoteAddr: ", rc.RemoteAddr), zap.Error(err))
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			// if time.Since(t) < 1*time.Minute{
			// 	return nil
			// }
			_, err := c.Do("PING")
			slog.Info("Ping. ", zap.Any("RemoteAddr: ", rc.RemoteAddr), zap.Error(err))
			return err
		},
	}
	slog.Info("连接redis", zap.Any("RemoteAddr: ", rc.RemoteAddr))
}

func (rc *RedisClient)Test() {
	c, err := redis.Dial("tcp", rc.RemoteAddr)
    if err != nil {
        slog.Info("dial ConnectSingle ...... ", zap.Any("RemoteAddr: ", rc.RemoteAddr), zap.Error(err))
        return
    }
	slog.Info("连接redis ConnectSingle", zap.Any("RemoteAddr: ", rc.RemoteAddr))
	_, err = c.Do("set", "aa", 12)
	slog.Info("redis  set", zap.Error(err))
	return
}

func (rc *RedisClient)Do(cmd string, args ...interface{}) (interface{}, error) {
	c := rc.pool.Get()
	defer c.Close()
	return c.Do(cmd, args...)
}