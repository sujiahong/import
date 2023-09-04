package su_redis

import (
	"github.com/garyburd/redigo/redis"
	slog "go/su_log"
	"go.uber.org/zap"
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
	rc.pool = redis.NewPool(func()(redis.Conn, error){
		c, err := redis.Dial("tcp", rc.RemoteAddr)
		slog.Info("dial ...... ", zap.Any("RemoteAddr: ", rc.RemoteAddr), zap.Error(err))
		return c, err
	}, rc.ConnNum)
	slog.Info("连接redis", zap.Any("RemoteAddr: ", rc.RemoteAddr))
}

func (rc *RedisClient)Do(cmd string, args ...interface{}) (interface{}, error) {
	c := rc.pool.Get()
	defer c.Close()
	return c.Do(cmd, args)
}