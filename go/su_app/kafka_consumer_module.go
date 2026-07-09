package su_app

import (
	"context"

	su_kafka "go.local/su_da/kafka"
	"go.local/su_errors"
)

type KafkaConsumerRunner interface {
	ConsumeAllPartion()
	Close()
}

type KafkaConsumerFactory func(cfg su_kafka.KafkaConsumerConfig, handler su_kafka.HandleMessageFunc) (KafkaConsumerRunner, error)

type KafkaConsumerModule struct {
	Config             su_kafka.KafkaConsumerConfig
	Handler            su_kafka.HandleMessageFunc
	Factory            KafkaConsumerFactory
	Consumer           KafkaConsumerRunner
	DisableAutoConsume bool
}

func NewKafkaConsumerModule(cfg su_kafka.KafkaConsumerConfig, handler su_kafka.HandleMessageFunc) *KafkaConsumerModule {
	return &KafkaConsumerModule{Config: cfg, Handler: handler}
}

func (m *KafkaConsumerModule) Name() string {
	return "kafka_consumer"
}

func (m *KafkaConsumerModule) Start(ctx context.Context) error {
	if m == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "kafka consumer module is nil")
	}
	if m.Consumer == nil {
		factory := m.Factory
		if factory == nil {
			factory = defaultKafkaConsumerFactory
		}
		consumer, err := factory(m.Config, m.Handler)
		if err != nil {
			return su_errors.WrapRetryable(su_errors.CodeUnavailable, "create kafka consumer failed", err)
		}
		m.Consumer = consumer
	}
	if !m.DisableAutoConsume {
		m.Consumer.ConsumeAllPartion()
	}
	return nil
}

func (m *KafkaConsumerModule) Stop(ctx context.Context) error {
	if m == nil || m.Consumer == nil {
		return nil
	}
	m.Consumer.Close()
	return nil
}

func defaultKafkaConsumerFactory(cfg su_kafka.KafkaConsumerConfig, handler su_kafka.HandleMessageFunc) (KafkaConsumerRunner, error) {
	consumer := su_kafka.NewKafkaConsumerWithConfig(cfg, handler)
	if consumer == nil {
		return nil, su_errors.NewRetryable(su_errors.CodeUnavailable, "create kafka consumer failed")
	}
	return consumer, nil
}
