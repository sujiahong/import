package su_kafka

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"go.local/su_util"
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

func TestKafkaConsumerCloseDrainsPool(t *testing.T) {
	pool := su_util.NewGoPool(1, 2)
	ctx, cancel := context.WithCancel(context.Background())
	kc := &KafkaConsumer{
		ctx:          ctx,
		cancel:       cancel,
		pool:         pool,
		closeTimeout: time.Second,
	}
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	drained := make(chan struct{})

	if !pool.SendTask(0, func() {
		close(firstStarted)
		<-releaseFirst
	}) {
		t.Fatal("failed to send first task")
	}
	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first task did not start")
	}
	if !pool.SendTask(0, func() {
		close(drained)
	}) {
		t.Fatal("failed to send queued task")
	}

	done := make(chan struct{})
	go func() {
		kc.Close()
		close(done)
	}()

	close(releaseFirst)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Close did not return")
	}
	select {
	case <-drained:
	default:
		t.Fatal("Close returned before draining queued task")
	}
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

func TestKafkaProducerSyncSendContextTimeout(t *testing.T) {
	fp := &fakeSyncProducer{sendDone: make(chan struct{})}
	kp := &KafkaProducer{
		Topic:  "test",
		client: fp,
		ctx:    context.Background(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	err := kp.SendContext(ctx, "k", "v")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("SendContext error = %v, want deadline exceeded", err)
	}
	if !fp.called.Load() {
		t.Fatal("sync producer SendMessage was not called")
	}
}

func TestKafkaProducerSyncSendReconnectsAndRetries(t *testing.T) {
	oldProducer := &fakeSyncProducer{sendErr: errors.New("broker unavailable")}
	newProducer := &fakeSyncProducer{}
	var created atomic.Int32
	originalNewSyncProducer := newSaramaSyncProducer
	newSaramaSyncProducer = func(addrs []string, config *sarama.Config) (sarama.SyncProducer, error) {
		created.Add(1)
		return newProducer, nil
	}
	defer func() { newSaramaSyncProducer = originalNewSyncProducer }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	kp := &KafkaProducer{
		AddrSlice: []string{"127.0.0.1:9092"},
		Topic:     "test",
		client:    oldProducer,
		cfg: KafkaProducerConfig{
			AddrSlice: []string{"127.0.0.1:9092"},
			Topic:     "test",
		},
		ctx: ctx,
	}

	if err := kp.SendContext(context.Background(), "k", "v"); err != nil {
		t.Fatalf("SendContext failed: %v", err)
	}
	if created.Load() != 1 {
		t.Fatalf("created producers = %d, want 1", created.Load())
	}
	if !oldProducer.closed.Load() {
		t.Fatal("old producer was not closed")
	}
	if !newProducer.called.Load() {
		t.Fatal("new producer was not used for retry")
	}
}

func TestKafkaAsyncProducerErrorReconnectsAndRetriesMessage(t *testing.T) {
	oldProducer := newFakeAsyncProducer(1)
	newProducer := newFakeAsyncProducer(1)
	var created atomic.Int32
	originalNewAsyncProducer := newSaramaAsyncProducer
	newSaramaAsyncProducer = func(addrs []string, config *sarama.Config) (sarama.AsyncProducer, error) {
		if created.Add(1) == 1 {
			return nil, errors.New("broker still unavailable")
		}
		return newProducer, nil
	}
	defer func() { newSaramaAsyncProducer = originalNewAsyncProducer }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	kp := &KafkaProducer{
		AddrSlice: []string{"127.0.0.1:9092"},
		Topic:     "test",
		Async:     true,
		asclient:  oldProducer,
		cfg: KafkaProducerConfig{
			AddrSlice:     []string{"127.0.0.1:9092"},
			Topic:         "test",
			Async:         true,
			RetryInterval: time.Millisecond,
		},
		ctx:           ctx,
		cancel:        cancel,
		retryInterval: time.Millisecond,
	}
	defer func() { _ = kp.Close() }()
	go kp.handleError(oldProducer)

	msg := &sarama.ProducerMessage{
		Topic: "test",
		Key:   sarama.StringEncoder("k"),
		Value: sarama.StringEncoder("v"),
	}
	oldProducer.errors <- &sarama.ProducerError{Msg: msg, Err: errors.New("broker unavailable")}

	select {
	case retried := <-newProducer.input:
		if retried != msg {
			t.Fatal("retried message did not match failed message")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for async message retry")
	}
	if created.Load() != 2 {
		t.Fatalf("created async producers = %d, want 2", created.Load())
	}
	if !oldProducer.closed.Load() {
		t.Fatal("old async producer was not closed")
	}
}

func TestKafkaAsyncProducerCloseDrainsBeforeCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	fp := newFakeAsyncProducer(1)
	fp.closeHook = func() {
		select {
		case <-ctx.Done():
			t.Fatal("producer context canceled before async producer close")
		default:
		}
	}
	kp := &KafkaProducer{
		Topic:    "test",
		Async:    true,
		asclient: fp,
		ctx:      ctx,
		cancel:   cancel,
	}

	if err := kp.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	select {
	case <-ctx.Done():
	default:
		t.Fatal("producer context was not canceled after async close")
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
		pool:      su_util.NewGoPool(1, 4),
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
		pool:      su_util.NewGoPool(1, 4),
		queueSize: 4,
	}
	defer kc.pool.Stop()
	var called atomic.Int32
	done := make(chan struct{})
	kc.processFunc = func(partitionID int32, msg *sarama.ConsumerMessage) {
		if string(msg.Value) != "payload" {
			t.Errorf("message value = %q, want payload", msg.Value)
		}
		called.Store(partitionID)
		close(done)
	}

	kc.dispatchMessage(&sarama.ConsumerMessage{Partition: 2, Offset: 8, Value: []byte("payload")}, 2)
	select {
	case <-done:
		if got := called.Load(); got != 2 {
			t.Fatalf("partition = %d, want 2", got)
		}
	case <-time.After(time.Second):
		t.Fatal("legacy handler was not called")
	}
}

func TestKafkaConsumerResubscribesAfterPartitionChannelClose(t *testing.T) {
	firstPC := newFakePartitionConsumer()
	secondPC := newFakePartitionConsumer()
	fc := &fakeConsumer{partitionConsumers: []sarama.PartitionConsumer{firstPC, secondPC}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	kc := &KafkaConsumer{
		Topic:         "test",
		client:        fc,
		ctx:           ctx,
		cancel:        cancel,
		pool:          su_util.NewGoPool(1, 4),
		queueSize:     4,
		closeTimeout:  time.Second,
		retryInterval: time.Millisecond,
	}
	defer kc.Close()

	received := make(chan string, 2)
	kc.messageFunc = func(ctx context.Context, msg *sarama.ConsumerMessage) error {
		received <- string(msg.Value)
		return nil
	}

	kc.ConsumeOnePartion(0)
	firstPC.messages <- &sarama.ConsumerMessage{Partition: 0, Offset: 1, Value: []byte("first")}
	if got := waitKafkaMessage(t, received); got != "first" {
		t.Fatalf("first message = %q, want first", got)
	}
	close(firstPC.messages)
	close(firstPC.errors)
	secondPC.messages <- &sarama.ConsumerMessage{Partition: 0, Offset: 2, Value: []byte("second")}
	if got := waitKafkaMessage(t, received); got != "second" {
		t.Fatalf("second message = %q, want second", got)
	}
	if fc.consumeCalls.Load() < 2 {
		t.Fatalf("consume calls = %d, want at least 2", fc.consumeCalls.Load())
	}
}

func waitKafkaMessage(t *testing.T, ch <-chan string) string {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for kafka message")
		return ""
	}
}

type fakeAsyncProducer struct {
	input     chan *sarama.ProducerMessage
	successes chan *sarama.ProducerMessage
	errors    chan *sarama.ProducerError
	closeHook func()
	closed    atomic.Bool
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
	if !fp.closed.CompareAndSwap(false, true) {
		return nil
	}
	if fp.closeHook != nil {
		fp.closeHook()
	}
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

type fakeSyncProducer struct {
	called   atomic.Bool
	closed   atomic.Bool
	sendDone chan struct{}
	sendErr  error
}

func (fp *fakeSyncProducer) SendMessage(msg *sarama.ProducerMessage) (partition int32, offset int64, err error) {
	fp.called.Store(true)
	if fp.sendDone != nil {
		<-fp.sendDone
	}
	return 0, 0, fp.sendErr
}

func (fp *fakeSyncProducer) SendMessages(msgs []*sarama.ProducerMessage) error { return nil }

func (fp *fakeSyncProducer) Close() error {
	fp.closed.Store(true)
	if fp.sendDone != nil {
		close(fp.sendDone)
	}
	return nil
}

func (fp *fakeSyncProducer) IsTransactional() bool { return false }

func (fp *fakeSyncProducer) TxnStatus() sarama.ProducerTxnStatusFlag { return 0 }

func (fp *fakeSyncProducer) BeginTxn() error { return nil }

func (fp *fakeSyncProducer) CommitTxn() error { return nil }

func (fp *fakeSyncProducer) AbortTxn() error { return nil }

func (fp *fakeSyncProducer) AddOffsetsToTxn(offsets map[string][]*sarama.PartitionOffsetMetadata, groupID string) error {
	return nil
}

func (fp *fakeSyncProducer) AddMessageToTxn(msg *sarama.ConsumerMessage, groupID string, metadata *string) error {
	return nil
}

type fakeConsumer struct {
	mu                 sync.Mutex
	partitionConsumers []sarama.PartitionConsumer
	consumeCalls       atomic.Int32
	closed             atomic.Bool
}

func (fc *fakeConsumer) Topics() ([]string, error) { return []string{"test"}, nil }

func (fc *fakeConsumer) Partitions(topic string) ([]int32, error) { return []int32{0}, nil }

func (fc *fakeConsumer) ConsumePartition(topic string, partition int32, offset int64) (sarama.PartitionConsumer, error) {
	fc.consumeCalls.Add(1)
	fc.mu.Lock()
	defer fc.mu.Unlock()
	if len(fc.partitionConsumers) == 0 {
		return nil, errors.New("no partition consumer")
	}
	pc := fc.partitionConsumers[0]
	fc.partitionConsumers = fc.partitionConsumers[1:]
	return pc, nil
}

func (fc *fakeConsumer) HighWaterMarks() map[string]map[int32]int64 { return nil }

func (fc *fakeConsumer) Close() error {
	fc.closed.Store(true)
	return nil
}

func (fc *fakeConsumer) Pause(topicPartitions map[string][]int32) {}

func (fc *fakeConsumer) Resume(topicPartitions map[string][]int32) {}

func (fc *fakeConsumer) PauseAll() {}

func (fc *fakeConsumer) ResumeAll() {}

type fakePartitionConsumer struct {
	messages    chan *sarama.ConsumerMessage
	errors      chan *sarama.ConsumerError
	asyncClosed atomic.Bool
}

func newFakePartitionConsumer() *fakePartitionConsumer {
	return &fakePartitionConsumer{
		messages: make(chan *sarama.ConsumerMessage, 2),
		errors:   make(chan *sarama.ConsumerError, 2),
	}
}

func (fpc *fakePartitionConsumer) AsyncClose() { fpc.asyncClosed.Store(true) }

func (fpc *fakePartitionConsumer) Close() error {
	fpc.AsyncClose()
	return nil
}

func (fpc *fakePartitionConsumer) Messages() <-chan *sarama.ConsumerMessage { return fpc.messages }

func (fpc *fakePartitionConsumer) Errors() <-chan *sarama.ConsumerError { return fpc.errors }

func (fpc *fakePartitionConsumer) HighWaterMarkOffset() int64 { return 0 }

func (fpc *fakePartitionConsumer) Pause() {}

func (fpc *fakePartitionConsumer) Resume() {}

func (fpc *fakePartitionConsumer) IsPaused() bool { return false }
