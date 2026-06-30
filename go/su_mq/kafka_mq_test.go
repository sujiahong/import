package su_mq

import (
	"context"
	"testing"

	"github.com/IBM/sarama"
)

func TestNewKafkaConsumerValidatesConfig(t *testing.T) {
	if _, err := NewKafkaConsumer(KafkaConsumerConfig{}, func(ctx context.Context, msg *sarama.ConsumerMessage) error { return nil }); err == nil {
		t.Fatal("expected empty addr error")
	}
	if _, err := NewKafkaConsumer(KafkaConsumerConfig{AddrSlice: []string{"127.0.0.1:9092"}}, func(ctx context.Context, msg *sarama.ConsumerMessage) error { return nil }); err == nil {
		t.Fatal("expected empty topic error")
	}
	if _, err := NewKafkaConsumer(KafkaConsumerConfig{AddrSlice: []string{"127.0.0.1:9092"}, Topic: "topic"}, nil); err == nil {
		t.Fatal("expected nil handler error")
	}
}

func TestKafkaConsumerNilSafe(t *testing.T) {
	var kc *KafkaConsumer
	if err := kc.StartAllPartitions(); err == nil {
		t.Fatal("expected nil consumer start error")
	}
	if err := kc.StartPartition(1); err == nil {
		t.Fatal("expected nil consumer partition start error")
	}
	kc.Close()
}
