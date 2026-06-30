package su_kafka

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"go.local/my_util"
)

func TestKafkaProducerNilSafeErrors(t *testing.T) {
	var kp *KafkaProducer
	if err := kp.Send("msg"); err == nil {
		t.Fatal("expected nil producer send error")
	}
	if err := kp.Close(); err != nil {
		t.Fatalf("nil producer close failed: %v", err)
	}
}

func TestKafkaAsyncProducerClosedContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	kp := &KafkaProducer{Async: true, ctx: ctx}

	if err := kp.SendWithKey("k", "v"); err == nil {
		t.Fatal("expected unconnected async producer error")
	}
	if err := kp.Close(); err != nil {
		t.Fatalf("async producer close failed: %v", err)
	}
}

func TestKafkaConsumerCloseIsNilSafe(t *testing.T) {
	var kc *KafkaConsumer
	kc.Close()

	kc = &KafkaConsumer{}
	kc.Close()
	kc.Close()
}

func TestKafkaProducerSendContextTimeout(t *testing.T) {
	fp := newFakeAsyncProducer(0)
	kp := &KafkaProducer{
		Topic:    "test",
		Async:    true,
		asclient: fp,
		ctx:      context.Background(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	err := kp.SendContext(ctx, "k", "v")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("SendContext error = %v, want deadline exceeded", err)
	}
}

func TestKafkaConsumerEnsurePoolUsesPartitionCount(t *testing.T) {
	kc := &KafkaConsumer{queueSize: 1}
	kc.ensurePool(8)
	defer kc.pool.Stop()

	if kc.workerNum != 8 {
		t.Fatalf("workerNum = %d, want 8", kc.workerNum)
	}
}

func TestKafkaConsumerDispatchMessageHandler(t *testing.T) {
	kc := &KafkaConsumer{
		ctx:       context.Background(),
		pool:      my_util.NewGoPool(1, 4),
		queueSize: 4,
	}
	defer kc.pool.Stop()
	called := make(chan int32, 1)
	kc.messageFunc = func(ctx context.Context, msg *sarama.ConsumerMessage) error {
		called <- msg.Partition
		return nil
	}

	kc.dispatchMessage(&sarama.ConsumerMessage{Partition: 3, Offset: 7}, 3)
	select {
	case partition := <-called:
		if partition != 3 {
			t.Fatalf("partition = %d, want 3", partition)
		}
	case <-time.After(time.Second):
		t.Fatal("handler was not called")
	}
}

func TestKafkaConsumerDispatchLegacyHandler(t *testing.T) {
	kc := &KafkaConsumer{
		ctx:       context.Background(),
		pool:      my_util.NewGoPool(1, 4),
		queueSize: 4,
	}
	defer kc.pool.Stop()
	var called atomic.Int32
	done := make(chan struct{})
	kc.processFunc = func(partitionID int32) {
		called.Store(partitionID)
		close(done)
	}

	kc.dispatchMessage(&sarama.ConsumerMessage{Partition: 2, Offset: 8}, 2)
	select {
	case <-done:
		if got := called.Load(); got != 2 {
			t.Fatalf("partition = %d, want 2", got)
		}
	case <-time.After(time.Second):
		t.Fatal("legacy handler was not called")
	}
}

type fakeAsyncProducer struct {
	input     chan *sarama.ProducerMessage
	successes chan *sarama.ProducerMessage
	errors    chan *sarama.ProducerError
}

func newFakeAsyncProducer(inputBuffer int) *fakeAsyncProducer {
	return &fakeAsyncProducer{
		input:     make(chan *sarama.ProducerMessage, inputBuffer),
		successes: make(chan *sarama.ProducerMessage),
		errors:    make(chan *sarama.ProducerError),
	}
}

func (fp *fakeAsyncProducer) AsyncClose() {}

func (fp *fakeAsyncProducer) Close() error {
	close(fp.input)
	close(fp.successes)
	close(fp.errors)
	return nil
}

func (fp *fakeAsyncProducer) Input() chan<- *sarama.ProducerMessage {
	return fp.input
}

func (fp *fakeAsyncProducer) Successes() <-chan *sarama.ProducerMessage {
	return fp.successes
}

func (fp *fakeAsyncProducer) Errors() <-chan *sarama.ProducerError {
	return fp.errors
}

func (fp *fakeAsyncProducer) IsTransactional() bool { return false }

func (fp *fakeAsyncProducer) TxnStatus() sarama.ProducerTxnStatusFlag { return 0 }

func (fp *fakeAsyncProducer) BeginTxn() error { return nil }

func (fp *fakeAsyncProducer) CommitTxn() error { return nil }

func (fp *fakeAsyncProducer) AbortTxn() error { return nil }

func (fp *fakeAsyncProducer) AddOffsetsToTxn(offsets map[string][]*sarama.PartitionOffsetMetadata, groupID string) error {
	return nil
}

func (fp *fakeAsyncProducer) AddMessageToTxn(msg *sarama.ConsumerMessage, groupID string, metadata *string) error {
	return nil
}
