package su_redis

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/garyburd/redigo/redis"
	"go.local/su_errors"
)

func TestRedisDoBeforeConnectReturnsError(t *testing.T) {
	rc := NewRedisClient("127.0.0.1:6379", 1)

	if _, err := rc.Do("PING"); err == nil {
		t.Fatal("expected error before redis connect")
	} else if su_errors.CodeOf(err) != su_errors.CodeUnavailable {
		t.Fatalf("error code = %d, want unavailable", su_errors.CodeOf(err))
	}
}

func TestNewRedisClientEmptyAddrDoesNotReturnNil(t *testing.T) {
	rc := NewRedisClient("", 1)
	if rc == nil {
		t.Fatal("NewRedisClient returned nil")
	}
	if err := rc.Connect(); err == nil {
		t.Fatal("expected connect error for empty redis addr")
	} else if su_errors.CodeOf(err) != su_errors.CodeInvalidArgument {
		t.Fatalf("error code = %d, want invalid argument", su_errors.CodeOf(err))
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
	rc.dial = fakeRedisDialer(fakeRedisConn{}, nil)
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
	rc.dial = fakeRedisDialer(fakeRedisConn{}, nil)

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

func TestRedisConnectPingsBeforePublishingPool(t *testing.T) {
	rc := NewRedisClient("127.0.0.1:6379", 1)
	rc.dial = fakeRedisDialer(fakeRedisConn{doErr: errors.New("redis down")}, nil)

	if err := rc.Connect(); err == nil {
		t.Fatal("Connect() error = nil, want ping failure")
	} else if su_errors.CodeOf(err) != su_errors.CodeUnavailable {
		t.Fatalf("error code = %d, want unavailable", su_errors.CodeOf(err))
	}
	rc.mu.RLock()
	pool := rc.pool
	rc.mu.RUnlock()
	if pool != nil {
		t.Fatal("pool should not be published after ping failure")
	}
}

func TestRedisReconnectResetsCloseOnce(t *testing.T) {
	rc := &RedisClient{}
	rc.setPoolForTest(&redis.Pool{
		MaxActive: 1,
		Dial: func() (redis.Conn, error) {
			return fakeRedisConn{}, nil
		},
	})
	if err := rc.Close(); err != nil {
		t.Fatalf("first close failed: %v", err)
	}
	rc.setPoolForTest(&redis.Pool{
		MaxActive: 1,
		Dial: func() (redis.Conn, error) {
			return fakeRedisConn{}, nil
		},
	})
	if err := rc.Close(); err != nil {
		t.Fatalf("second close after reconnect failed: %v", err)
	}
	rc.mu.RLock()
	pool := rc.pool
	rc.mu.RUnlock()
	if pool != nil {
		t.Fatal("pool was not closed after reconnect")
	}
}

func TestRedisCloseWaitsForReconnectLock(t *testing.T) {
	rc := &RedisClient{
		pool: &redis.Pool{
			MaxActive: 1,
			Dial: func() (redis.Conn, error) {
				return fakeRedisConn{}, nil
			},
		},
	}
	rc.closeOnce = sync.Once{}
	rc.reconnectMu.Lock()
	closeDone := make(chan error, 1)
	go func() {
		closeDone <- rc.Close()
	}()
	select {
	case err := <-closeDone:
		t.Fatalf("Close completed while reconnectMu was held: %v", err)
	case <-time.After(10 * time.Millisecond):
	}
	rc.reconnectMu.Unlock()
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("Close error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Close did not finish after reconnectMu was released")
	}
}

func TestRedisCloseDoesNotWaitForBlockedPoolGet(t *testing.T) {
	rc := &RedisClient{
		pool: &redis.Pool{
			MaxActive: 1,
			Wait:      true,
			Dial: func() (redis.Conn, error) {
				return fakeRedisConn{}, nil
			},
		},
	}
	c := rc.pool.Get()
	defer c.Close()

	doStarted := make(chan struct{})
	doDone := make(chan struct{})
	go func() {
		close(doStarted)
		_, _ = rc.Do("PING")
		close(doDone)
	}()
	select {
	case <-doStarted:
	case <-time.After(time.Second):
		t.Fatal("Do goroutine did not start")
	}
	time.Sleep(10 * time.Millisecond)

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- rc.Close()
	}()
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("Close error = %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Close blocked behind Do waiting in pool.Get")
	}
	_ = c.Close()
	select {
	case <-doDone:
	case <-time.After(time.Second):
		t.Fatal("Do did not unblock after held connection closed")
	}
}

func (rc *RedisClient) setPoolForTest(pool *redis.Pool) {
	rc.mu.Lock()
	rc.pool = pool
	rc.closeOnce = sync.Once{}
	rc.closeErr = nil
	rc.mu.Unlock()
}

func fakeRedisDialer(conn redis.Conn, err error) func(RedisConfig) (redis.Conn, error) {
	return func(RedisConfig) (redis.Conn, error) {
		return conn, err
	}
}

type fakeRedisConn struct {
	connErr error
	doErr   error
}

func (fakeRedisConn) Close() error { return nil }

func (c fakeRedisConn) Err() error { return c.connErr }

func (c fakeRedisConn) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	if c.doErr != nil {
		return nil, c.doErr
	}
	return "PONG", nil
}

func (fakeRedisConn) Send(commandName string, args ...interface{}) error { return nil }

func (fakeRedisConn) Flush() error { return nil }

func (fakeRedisConn) Receive() (reply interface{}, err error) { return nil, nil }
