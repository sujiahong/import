package su_mq

import (
	"context"
	"testing"

	"github.com/IBM/sarama"
	"go.local/su_errors"
)

func TestNewKafkaConsumerValidatesConfig(t *testing.T) {
	if _, err := NewKafkaConsumer(KafkaConsumerConfig{}, func(ctx context.Context, msg *sarama.ConsumerMessage) error { return nil }); err == nil {
		t.Fatal("expected empty addr error")
	} else if su_errors.CodeOf(err) != su_errors.CodeInvalidArgument {
		t.Fatalf("error code = %d, want invalid argument", su_errors.CodeOf(err))
	}
	if _, err := NewKafkaConsumer(KafkaConsumerConfig{AddrSlice: []string{"127.0.0.1:9092"}}, func(ctx context.Context, msg *sarama.ConsumerMessage) error { return nil }); err == nil {
		t.Fatal("expected empty topic error")
	} else if su_errors.CodeOf(err) != su_errors.CodeInvalidArgument {
		t.Fatalf("error code = %d, want invalid argument", su_errors.CodeOf(err))
	}
	if _, err := NewKafkaConsumer(KafkaConsumerConfig{AddrSlice: []string{"127.0.0.1:9092"}, Topic: "topic"}, nil); err == nil {
		t.Fatal("expected nil handler error")
	} else if su_errors.CodeOf(err) != su_errors.CodeInvalidArgument {
		t.Fatalf("error code = %d, want invalid argument", su_errors.CodeOf(err))
	}
}

func TestKafkaConsumerNilSafe(t *testing.T) {
	var kc *KafkaConsumer
	if err := kc.StartAllPartitions(); err == nil {
		t.Fatal("expected nil consumer start error")
	} else if su_errors.CodeOf(err) != su_errors.CodeInvalidArgument {
		t.Fatalf("error code = %d, want invalid argument", su_errors.CodeOf(err))
	}
	if err := kc.StartPartition(1); err == nil {
		t.Fatal("expected nil consumer partition start error")
	}
	kc.Close()
}

func TestKafkaPartitionHandlerIncludesMessage(t *testing.T) {
	var handler KafkaPartitionHandler = func(partitionID int32, msg *sarama.ConsumerMessage) {
		if partitionID != 1 {
			t.Fatalf("partitionID = %d, want 1", partitionID)
		}
		if string(msg.Value) != "payload" {
			t.Fatalf("message value = %q, want payload", msg.Value)
		}
	}

	handler(1, &sarama.ConsumerMessage{Value: []byte("payload")})
}
