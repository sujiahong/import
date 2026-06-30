package su_redis

import (
	"testing"
	"time"
)

func TestRedisDoBeforeConnectReturnsError(t *testing.T) {
	rc := NewRedisClient("127.0.0.1:6379", 1)

	if _, err := rc.Do("PING"); err == nil {
		t.Fatal("expected error before redis connect")
	}
}

func TestRedisConfigConstructorDefaults(t *testing.T) {
	rc, err := NewRedisClientWithConfig(RedisConfig{
		RemoteAddr:  "127.0.0.1:6379",
		ConnNum:     4,
		MaxActive:   8,
		IdleTimeout: time.Minute,
		Wait:        false,
	})
	if err != nil {
		t.Fatalf("config constructor failed: %v", err)
	}
	if err := rc.Connect(); err != nil {
		t.Fatalf("connect config failed: %v", err)
	}
	defer rc.Close()
	if rc.pool.MaxIdle != 4 {
		t.Fatalf("MaxIdle = %d, want 4", rc.pool.MaxIdle)
	}
	if rc.pool.MaxActive != 8 {
		t.Fatalf("MaxActive = %d, want 8", rc.pool.MaxActive)
	}
	if rc.pool.IdleTimeout != time.Minute {
		t.Fatalf("IdleTimeout = %v, want %v", rc.pool.IdleTimeout, time.Minute)
	}
	if rc.pool.Wait {
		t.Fatal("Wait should respect explicit false")
	}
}

func TestRedisConnectDefaultsAndClose(t *testing.T) {
	rc := NewRedisClient("127.0.0.1:6379", 0)

	if err := rc.Connect(); err != nil {
		t.Fatalf("connect config failed: %v", err)
	}
	if rc.ConnNum != 1 {
		t.Fatalf("ConnNum = %d, want 1", rc.ConnNum)
	}
	if rc.pool == nil {
		t.Fatal("pool is nil after Connect")
	}
	if err := rc.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if err := rc.Close(); err != nil {
		t.Fatalf("second close failed: %v", err)
	}
}
