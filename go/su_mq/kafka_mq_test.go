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

func TestKafkaMessageToMessage(t *testing.T) {
	msg := kafkaMessageToMessage("fallback", &sarama.ConsumerMessage{
		Topic:     "topic",
		Key:       []byte("key"),
		Value:     []byte("value"),
		Partition: 2,
		Offset:    3,
		Headers: []*sarama.RecordHeader{
			{Key: []byte("trace_id"), Value: []byte("t1")},
		},
	})
	if msg.Source != "kafka" || msg.Topic != "topic" || string(msg.Key) != "key" || string(msg.Value) != "value" {
		t.Fatalf("unexpected message: %+v", msg)
	}
	if msg.Partition != 2 || msg.Offset != 3 {
		t.Fatalf("partition/offset = %d/%d, want 2/3", msg.Partition, msg.Offset)
	}
	if msg.Headers["trace_id"] != "t1" {
		t.Fatalf("header trace_id = %q, want t1", msg.Headers["trace_id"])
	}
}
