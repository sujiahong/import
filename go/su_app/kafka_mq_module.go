package su_app

import (
	"context"

	"go.local/su_errors"
	"go.local/su_mq"
)

/* 使用示例
app.Register(su_app.NewKafkaMQModule(su_mq.KafkaConsumerConfig{
    AddrSlice: []string{"127.0.0.1:9092"},
    Topic:     "order",
    ClientID:  "order-service",
    WorkerNum: 8,
    QueueSize: 4096,
    RetryPolicy: su_mq.FixedRetry{
        MaxAttempts: 3,
        Delay:       time.Second,
    },
    DeadLetter:  su_mq.NewMemoryDeadLetter(),
    Idempotency: su_mq.NewMemoryIdempotency(),
}, func(ctx context.Context, msg *sarama.ConsumerMessage) error {
    // 业务处理
    return nil
}))
*/

type KafkaMQConsumer interface {
	StartAllPartitions() error
	StartPartition(partitionID int32) error
	Close()
}

type KafkaMQFactory func(cfg su_mq.KafkaConsumerConfig, handler su_mq.KafkaMessageHandler) (KafkaMQConsumer, error)

type KafkaMQModule struct {
	Config             su_mq.KafkaConsumerConfig
	Handler            su_mq.KafkaMessageHandler
	Factory            KafkaMQFactory
	Consumer           KafkaMQConsumer
	DisableAutoConsume bool
	PartitionID        *int32
}

func NewKafkaMQModule(cfg su_mq.KafkaConsumerConfig, handler su_mq.KafkaMessageHandler) *KafkaMQModule {
	return &KafkaMQModule{Config: cfg, Handler: handler}
}

func (m *KafkaMQModule) Name() string {
	return "kafka_mq"
}

func (m *KafkaMQModule) Start(ctx context.Context) error {
	if m == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "kafka mq module is nil")
	}
	if m.Consumer == nil {
		factory := m.Factory
		if factory == nil {
			factory = defaultKafkaMQFactory
		}
		consumer, err := factory(m.Config, m.Handler)
		if err != nil {
			return su_errors.WrapRetryable(su_errors.CodeUnavailable, "create kafka mq consumer failed", err)
		}
		m.Consumer = consumer
	}
	if m.DisableAutoConsume {
		return nil
	}
	if m.PartitionID != nil {
		if err := m.Consumer.StartPartition(*m.PartitionID); err != nil {
			return su_errors.Wrap(su_errors.CodeInternal, "start kafka mq partition consumer failed", err)
		}
		return nil
	}
	if err := m.Consumer.StartAllPartitions(); err != nil {
		return su_errors.Wrap(su_errors.CodeInternal, "start kafka mq consumer failed", err)
	}
	return nil
}

func (m *KafkaMQModule) Stop(ctx context.Context) error {
	if m == nil || m.Consumer == nil {
		return nil
	}
	m.Consumer.Close()
	return nil
}

func defaultKafkaMQFactory(cfg su_mq.KafkaConsumerConfig, handler su_mq.KafkaMessageHandler) (KafkaMQConsumer, error) {
	return su_mq.NewKafkaConsumer(cfg, handler)
}
