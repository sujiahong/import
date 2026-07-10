package su_mq

import (
	"context"
	"time"

	"github.com/IBM/sarama"
	skafka "go.local/su_da/kafka"
	"go.local/su_errors"
)

type KafkaBackpressureMode = skafka.KafkaBackpressureMode

const (
	KafkaBackpressureBlock = skafka.KafkaBackpressureBlock
	KafkaBackpressureDrop  = skafka.KafkaBackpressureDrop
)

type KafkaConsumerConfig struct {
	AddrSlice        []string
	Topic            string
	ClientID         string
	WorkerNum        uint32
	QueueSize        uint32
	CloseTimeout     time.Duration
	RetryInterval    time.Duration
	BackpressureMode KafkaBackpressureMode
	LogMessages      bool
	RetryPolicy      RetryPolicy
	DeadLetter       DeadLetter
	Idempotency      Idempotency
	Metrics          MQMetrics
}

type KafkaMessageHandler = skafka.HandleMessageFunc
type KafkaPartitionHandler = skafka.HandleFunc

type KafkaConsumer struct {
	consumer *skafka.KafkaConsumer
}

func NewKafkaConsumer(cfg KafkaConsumerConfig, handler KafkaMessageHandler) (*KafkaConsumer, error) {
	if len(cfg.AddrSlice) == 0 {
		return nil, su_errors.New(su_errors.CodeInvalidArgument, "kafka addr slice is empty")
	}
	if cfg.Topic == "" {
		return nil, su_errors.New(su_errors.CodeInvalidArgument, "kafka topic is empty")
	}
	if handler == nil {
		return nil, su_errors.New(su_errors.CodeInvalidArgument, "kafka message handler is nil")
	}
	processor := NewProcessor(processorOptionsFromKafkaConfig(cfg))
	wrappedHandler := func(ctx context.Context, msg *sarama.ConsumerMessage) error {
		return processor.Process(ctx, kafkaMessageToMessage(cfg.Topic, msg), func(ctx context.Context, message Message) error {
			return handler(ctx, msg)
		})
	}
	consumer := skafka.NewKafkaConsumerWithConfig(skafka.KafkaConsumerConfig{
		AddrSlice:        cfg.AddrSlice,
		Topic:            cfg.Topic,
		ClientID:         cfg.ClientID,
		WorkerNum:        cfg.WorkerNum,
		QueueSize:        cfg.QueueSize,
		CloseTimeout:     cfg.CloseTimeout,
		RetryInterval:    cfg.RetryInterval,
		BackpressureMode: cfg.BackpressureMode,
		LogMessages:      cfg.LogMessages,
	}, wrappedHandler)
	if consumer == nil {
		return nil, su_errors.NewRetryable(su_errors.CodeUnavailable, "kafka consumer create failed")
	}
	return &KafkaConsumer{consumer: consumer}, nil
}

func NewKafkaPartitionConsumer(addr []string, topic, clientID string, handler KafkaPartitionHandler) (*KafkaConsumer, error) {
	if len(addr) == 0 {
		return nil, su_errors.New(su_errors.CodeInvalidArgument, "kafka addr slice is empty")
	}
	if topic == "" {
		return nil, su_errors.New(su_errors.CodeInvalidArgument, "kafka topic is empty")
	}
	if handler == nil {
		return nil, su_errors.New(su_errors.CodeInvalidArgument, "kafka partition handler is nil")
	}
	consumer := skafka.NewKafkaConsumer(addr, topic, clientID, handler)
	if consumer == nil {
		return nil, su_errors.NewRetryable(su_errors.CodeUnavailable, "kafka consumer create failed")
	}
	return &KafkaConsumer{consumer: consumer}, nil
}

func (kc *KafkaConsumer) StartAllPartitions() error {
	if kc == nil || kc.consumer == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "kafka consumer is nil")
	}
	kc.consumer.ConsumeAllPartion()
	return nil
}

func (kc *KafkaConsumer) StartPartition(partitionID int32) error {
	if kc == nil || kc.consumer == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "kafka consumer is nil")
	}
	kc.consumer.ConsumeOnePartion(partitionID)
	return nil
}

func (kc *KafkaConsumer) Close() {
	if kc == nil || kc.consumer == nil {
		return
	}
	kc.consumer.Close()
}

func processorOptionsFromKafkaConfig(cfg KafkaConsumerConfig) ProcessorOptions {
	return ProcessorOptions{
		RetryPolicy: cfg.RetryPolicy,
		DeadLetter:  cfg.DeadLetter,
		Idempotency: cfg.Idempotency,
		Metrics:     cfg.Metrics,
	}
}

func kafkaMessageToMessage(topic string, msg *sarama.ConsumerMessage) Message {
	if msg == nil {
		return Message{Source: "kafka", Topic: topic}
	}
	headers := make(map[string]string, len(msg.Headers))
	for _, header := range msg.Headers {
		if header == nil {
			continue
		}
		headers[string(header.Key)] = string(header.Value)
	}
	if msg.Topic != "" {
		topic = msg.Topic
	}
	return Message{
		Source:    "kafka",
		Topic:     topic,
		Key:       append([]byte(nil), msg.Key...),
		Value:     append([]byte(nil), msg.Value...),
		Headers:   headers,
		Partition: msg.Partition,
		Offset:    msg.Offset,
		Raw:       msg,
	}
}
