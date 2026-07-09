package su_redis

import (
	"testing"

	"github.com/garyburd/redigo/redis"
)

func BenchmarkRedisDo(b *testing.B) {
	rc := newBenchRedisClient(16, true)
	defer rc.Close()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := rc.Do("PING"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRedisDoParallel(b *testing.B) {
	rc := newBenchRedisClient(64, true)
	defer rc.Close()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := rc.Do("PING"); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkRedisDoParallelContended(b *testing.B) {
	rc := newBenchRedisClient(1, true)
	defer rc.Close()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := rc.Do("PING"); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func newBenchRedisClient(maxActive int, wait bool) *RedisClient {
	rc := &RedisClient{}
	rc.setPoolForTest(&redis.Pool{
		MaxIdle:   maxActive,
		MaxActive: maxActive,
		Wait:      wait,
		Dial: func() (redis.Conn, error) {
			return fakeRedisConn{}, nil
		},
	})
	return rc
}
