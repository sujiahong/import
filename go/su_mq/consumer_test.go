package su_mq

import (
	"context"
	"errors"
	"testing"

	"go.local/su_metrics"
)

func TestProcessorRetryThenSuccess(t *testing.T) {
	metrics := su_metrics.NewMemoryMetrics()
	processor := NewProcessor(ProcessorOptions{
		RetryPolicy: FixedRetry{MaxAttempts: 2},
		Metrics:     NewDefaultMQMetrics(metrics),
	})
	var calls int
	err := processor.Process(context.Background(), Message{Source: "test", Topic: "topic", ID: "1"}, func(ctx context.Context, msg Message) error {
		calls++
		if calls == 1 {
			return errors.New("temporary")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
	if got := metrics.Counter("mq_consume_retry_total", su_metrics.Labels{"source": "test", "topic": "topic", "status": "retry", "attempt": "1"}); got != 1 {
		t.Fatalf("retry metric = %v, want 1", got)
	}
}

func TestProcessorDeadLetterAfterFailure(t *testing.T) {
	dlq := NewMemoryDeadLetter()
	processor := NewProcessor(ProcessorOptions{DeadLetter: dlq})
	err := processor.Process(context.Background(), Message{Source: "test", Topic: "topic", ID: "1"}, func(ctx context.Context, msg Message) error {
		return errors.New("failed")
	})
	if err == nil {
		t.Fatal("Process() error = nil, want failure")
	}
	if got := len(dlq.Messages()); got != 1 {
		t.Fatalf("dead letter messages = %d, want 1", got)
	}
}

func TestProcessorIdempotencySkipsSeenMessage(t *testing.T) {
	idem := NewMemoryIdempotency()
	msg := Message{Source: "test", Topic: "topic", ID: "1"}
	if err := idem.Mark(context.Background(), messageID(msg)); err != nil {
		t.Fatalf("Mark() error = %v", err)
	}
	processor := NewProcessor(ProcessorOptions{Idempotency: idem})
	var calls int
	if err := processor.Process(context.Background(), msg, func(ctx context.Context, msg Message) error {
		calls++
		return nil
	}); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if calls != 0 {
		t.Fatalf("calls = %d, want 0", calls)
	}
}
