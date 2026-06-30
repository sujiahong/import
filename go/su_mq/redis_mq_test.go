package su_mq

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRedisListConsumerMultiReaderWorker(t *testing.T) {
	client := newFakeRedisClient()
	var handled atomic.Int32
	done := make(chan struct{})
	consumer, err := NewRedisListConsumerWithClient(RedisListConsumerConfig{
		ListKey:      "jobs",
		ReaderNum:    2,
		WorkerNum:    2,
		QueueSize:    8,
		PopTimeout:   time.Second,
		CloseTimeout: time.Second,
	}, client, func(ctx context.Context, msg RedisListMessage) error {
		if msg.ListKey != "jobs" {
			t.Errorf("ListKey = %s, want jobs", msg.ListKey)
		}
		if handled.Add(1) == 2 {
			close(done)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("new consumer failed: %v", err)
	}
	if err := consumer.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	client.push([]interface{}{[]byte("jobs"), []byte("a")})
	client.push([]interface{}{[]byte("jobs"), []byte("b")})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("messages were not handled")
	}
	if err := consumer.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
}

func TestRedisListConsumerStartTwiceReturnsError(t *testing.T) {
	client := newFakeRedisClient()
	consumer, err := NewRedisListConsumerWithClient(RedisListConsumerConfig{
		ListKey:      "jobs",
		ReaderNum:    1,
		WorkerNum:    1,
		QueueSize:    1,
		PopTimeout:   time.Second,
		CloseTimeout: time.Second,
	}, client, func(ctx context.Context, msg RedisListMessage) error { return nil })
	if err != nil {
		t.Fatalf("new consumer failed: %v", err)
	}
	if err := consumer.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if err := consumer.Start(); err == nil {
		t.Fatal("expected start twice error")
	}
	if err := consumer.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
}

func TestRedisListConsumerConcurrentClose(t *testing.T) {
	client := newFakeRedisClient()
	consumer, err := NewRedisListConsumerWithClient(RedisListConsumerConfig{
		ListKey:      "jobs",
		ReaderNum:    2,
		WorkerNum:    2,
		QueueSize:    2,
		PopTimeout:   time.Second,
		CloseTimeout: time.Second,
	}, client, func(ctx context.Context, msg RedisListMessage) error { return nil })
	if err != nil {
		t.Fatalf("new consumer failed: %v", err)
	}
	if err := consumer.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = consumer.Close()
		}()
	}
	wg.Wait()
}

func TestRedisListConsumerNilSafe(t *testing.T) {
	var consumer *RedisListConsumer
	if err := consumer.Start(); err == nil {
		t.Fatal("expected nil start error")
	}
	if err := consumer.Close(); err != nil {
		t.Fatalf("nil close failed: %v", err)
	}
}

type fakeRedisClient struct {
	replies chan interface{}
	closed  chan struct{}
	once    sync.Once
}

func newFakeRedisClient() *fakeRedisClient {
	return &fakeRedisClient{
		replies: make(chan interface{}, 16),
		closed:  make(chan struct{}),
	}
}

func (fc *fakeRedisClient) push(reply interface{}) {
	fc.replies <- reply
}

func (fc *fakeRedisClient) Do(cmd string, args ...interface{}) (interface{}, error) {
	if cmd != "BRPOP" {
		return nil, errors.New("unexpected command")
	}
	select {
	case reply := <-fc.replies:
		return reply, nil
	case <-fc.closed:
		return nil, errors.New("closed")
	case <-time.After(10 * time.Millisecond):
		return nil, nil
	}
}

func (fc *fakeRedisClient) Close() error {
	fc.once.Do(func() {
		close(fc.closed)
	})
	return nil
}
