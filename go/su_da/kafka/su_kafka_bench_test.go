package su_kafka

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"go.local/su_util"
)

func BenchmarkKafkaProducerAsyncSendContext(b *testing.B) {
	fp := newFakeAsyncProducer(1024)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			case <-fp.input:
			}
		}
	}()
	defer close(done)

	kp := &KafkaProducer{
		Topic:    "bench",
		Async:    true,
		asclient: fp,
		ctx:      context.Background(),
	}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := kp.SendContext(ctx, "key", "value"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkKafkaProducerSyncSendContext(b *testing.B) {
	kp := &KafkaProducer{
		Topic:  "bench",
		client: fastSyncProducer{},
		ctx:    context.Background(),
	}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := kp.SendContext(ctx, "key", "value"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkKafkaProducerSyncSendContextCancelable(b *testing.B) {
	kp := &KafkaProducer{
		Topic:  "bench",
		client: fastSyncProducer{},
		ctx:    context.Background(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := kp.SendContext(ctx, "key", "value"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkKafkaConsumerDispatchMessage(b *testing.B) {
	var handled atomic.Uint64
	kc := &KafkaConsumer{
		ctx:          context.Background(),
		pool:         su_util.NewGoPool(8, 1024),
		queueSize:    1024,
		closeTimeout: 5 * time.Second,
		messageFunc: func(ctx context.Context, msg *sarama.ConsumerMessage) error {
			handled.Add(1)
			return nil
		},
	}
	defer kc.pool.StopAndDrain(5 * time.Second)

	msg := &sarama.ConsumerMessage{Partition: 1, Offset: 1, Value: []byte("value")}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg.Offset = int64(i)
		kc.dispatchMessage(msg, 1)
	}
	b.StopTimer()

	deadline := time.After(5 * time.Second)
	for handled.Load() < uint64(b.N) {
		select {
		case <-deadline:
			b.Fatalf("handled %d messages, want %d", handled.Load(), b.N)
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

type fastSyncProducer struct{}

func (fastSyncProducer) SendMessage(msg *sarama.ProducerMessage) (partition int32, offset int64, err error) {
	return 0, 1, nil
}

func (fastSyncProducer) SendMessages(msgs []*sarama.ProducerMessage) error { return nil }

func (fastSyncProducer) Close() error { return nil }

func (fastSyncProducer) IsTransactional() bool { return false }

func (fastSyncProducer) TxnStatus() sarama.ProducerTxnStatusFlag { return 0 }

func (fastSyncProducer) BeginTxn() error { return nil }

func (fastSyncProducer) CommitTxn() error { return nil }

func (fastSyncProducer) AbortTxn() error { return nil }

func (fastSyncProducer) AddOffsetsToTxn(offsets map[string][]*sarama.PartitionOffsetMetadata, groupID string) error {
	return nil
}

func (fastSyncProducer) AddMessageToTxn(msg *sarama.ConsumerMessage, groupID string, metadata *string) error {
	return nil
}
