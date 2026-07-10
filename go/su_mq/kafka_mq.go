package su_mq

import (
	"context"
	"time"

	"github.com/IBM/sarama"
	skafka "go.local/su_da/kafka"
	"go.local/su_errors"
)

// KafkaBackpressureMode 复用底层 Kafka consumer 的背压策略类型。
type KafkaBackpressureMode = skafka.KafkaBackpressureMode

const (
	KafkaBackpressureBlock = skafka.KafkaBackpressureBlock
	KafkaBackpressureDrop  = skafka.KafkaBackpressureDrop
)

// KafkaConsumerConfig 定义 su_mq Kafka consumer 的连接、处理器和背压配置。
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

// KafkaMessageHandler 是带 context 的 Kafka 消息处理函数。
type KafkaMessageHandler = skafka.HandleMessageFunc

// KafkaPartitionHandler 是旧版按分区消费的 Kafka 消息处理函数。
type KafkaPartitionHandler = skafka.HandleFunc

// KafkaConsumer 封装底层 su_da/kafka consumer，并接入通用 Processor。
type KafkaConsumer struct {
	consumer *skafka.KafkaConsumer
}

// NewKafkaConsumer 创建 Kafka consumer，并用 Processor 包装业务 handler。
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

// NewKafkaPartitionConsumer 创建旧版按分区回调的 Kafka consumer。
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

// StartAllPartitions 启动 topic 下全部分区的消费。
func (kc *KafkaConsumer) StartAllPartitions() error {
	if kc == nil || kc.consumer == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "kafka consumer is nil")
	}
	kc.consumer.ConsumeAllPartion()
	return nil
}

// StartPartition 启动指定分区的消费。
func (kc *KafkaConsumer) StartPartition(partitionID int32) error {
	if kc == nil || kc.consumer == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "kafka consumer is nil")
	}
	kc.consumer.ConsumeOnePartion(partitionID)
	return nil
}

// Close 关闭底层 Kafka consumer。
func (kc *KafkaConsumer) Close() {
	if kc == nil || kc.consumer == nil {
		return
	}
	kc.consumer.Close()
}

// processorOptionsFromKafkaConfig 提取 Kafka consumer 的通用 Processor 配置。
func processorOptionsFromKafkaConfig(cfg KafkaConsumerConfig) ProcessorOptions {
	return ProcessorOptions{
		RetryPolicy: cfg.RetryPolicy,
		DeadLetter:  cfg.DeadLetter,
		Idempotency: cfg.Idempotency,
		Metrics:     cfg.Metrics,
	}
}

// kafkaMessageToMessage 将 sarama.ConsumerMessage 转为 su_mq 统一消息模型。
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
