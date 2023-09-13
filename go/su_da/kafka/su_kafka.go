package su_kafka

import (
	slog "go/su_log"
	"go.uber.org/zap"
	"github.com/IBM/sarama"
	"time"
	"strconv"
	"context"
)

type KafkaProducer struct {
	AddrSlice     []string
	Topic         string
	Async         bool
	client        sarama.SyncProducer
	asclient      sarama.AsyncProducer
	ctx           context.Context
	cancel        func()
}

func NewKafkaProducer(a_addr []string, a_topic string, a_async bool) *KafkaProducer {
	config := sarama.NewConfig()
	// config.Producer.RequiredAcks = sarama.WaitForAll
	// config.Producer.Partitioner = sarama.NewRandomPartitioner
	config.Producer.Return.Errors = true
	config.ChannelBufferSize = 2000
	var err error
	kp := &KafkaProducer{AddrSlice: a_addr, Topic: a_topic, Async: a_async}
	if a_async {
		kp.asclient, err = sarama.NewAsyncProducer(a_addr, config)
		if err != nil {
			slog.Error("kafka NewAsyncProducer failed", zap.Error(err))
			return nil
		}
	}else {
		config.Producer.Return.Successes = true
		kp.client, err = sarama.NewSyncProducer(a_addr, config)
		if err != nil {
			slog.Error("kafka NewSyncProducer failed", zap.Error(err))
			return nil
		}
	}
	kp.ctx, kp.cancel = context.WithCancel(context.Background())
	go kp.handleSuccess()
	go kp.handleError()
	return kp
}

func (kp *KafkaProducer)Send(a_msg string) error {
	time_now := time.Now().UnixNano()
	key := strconv.FormatInt(time_now, 10)
	msg := &sarama.ProducerMessage{}
	msg.Topic = kp.Topic
	msg.Key = sarama.StringEncoder(key)
	msg.Value = sarama.StringEncoder(a_msg)
	if kp.Async {
		kp.asclient.Input() <- msg
		slog.Info("Send finish", zap.Any("key", key), zap.Any("a_msg", a_msg))
	}else {
		pid, offset, err := kp.client.SendMessage(msg)
		if err != nil {
			slog.Error("kafka SendMessage failed", zap.Error(err))
			return err
		}
		slog.Info("Send success", zap.Any("pid", pid), zap.Any("offset", offset))
	}
	return nil
}

func (kp *KafkaProducer)SendWithKey(a_key, a_msg string) error {
	msg := &sarama.ProducerMessage{}
	msg.Topic = kp.Topic
	msg.Key = sarama.StringEncoder(a_key)
	msg.Value = sarama.StringEncoder(a_msg)
	if kp.Async {
		kp.asclient.Input() <- msg
		slog.Info("SendWithKey finish", zap.Any("a_key", a_key), zap.Any("a_msg", a_msg))
	}else {
		pid, offset, err := kp.client.SendMessage(msg)
		if err != nil {
			slog.Error("kafka SendMessage failed", zap.Error(err))
			return err
		}
		slog.Info("SendWithKey success", zap.Any("pid", pid), zap.Any("offset", offset))
	}
	return nil
}

func (kp *KafkaProducer)Close() error {
	if kp.Async {
		return kp.asclient.Close()
	}else {
		return kp.client.Close()
	}
	kp.cancel()
}

func (kp *KafkaProducer)handleSuccess(){
	for {
		select {
		case <- kp.ctx.Done():
			return
		case pm := <- kp.asclient.Successes():
			if pm != nil {
				slog.Info("kafka success", zap.Bool("async", kp.Async),
					zap.Int32("Partition", pm.Partition),
					zap.Int64("Offset", pm.Offset),
					zap.Any("Key", pm.Key),
					zap.Any("Value", pm.Value))
			}
		}
	}
}

func (kp * KafkaProducer)handleError(){
	for {
		select {
		case <- kp.ctx.Done():
			return
		case err := <- kp.asclient.Errors():
			if err != nil {
				slog.Info("kafka error", zap.Error(err), zap.Bool("async", kp.Async),
					zap.Int32("Partition", err.Msg.Partition),
					zap.Int64("Offset", err.Msg.Offset),
					zap.Any("Key", err.Msg.Key),
					zap.Any("Value", err.Msg.Value))
			}
		}
	}
}

type HandleFunc func()
type KafkaConsumer struct {
	AddrSlice     []string
	Topic         string
	client        sarama.Consumer
	ClientID      string
	processFunc   HandleFunc
}

func NewKafkaConsumer(a_addr []string, a_topic, a_cli_id string, a_func HandleFunc) *KafkaConsumer{
	config := sarama.NewConfig()
	config.ClientID = a_cli_id
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Offsets.AutoCommit.Interval = time.Duration(10)*time.Second
	config.Consumer.MaxProcessingTime = 10 * time.Second

	kc := &KafkaConsumer{AddrSlice: a_addr, Topic: a_topic, ClientID: a_cli_id, processFunc: a_func}
	var err error
	kc.client, err = sarama.NewConsumer(a_addr, nil)
	if err != nil {
		slog.Error("kafka NewConsummer failed", zap.Error(err))
		return nil
	}
	return kc
}

func (kc *KafkaConsumer)ConsumeAllPartion(){
	partitionList, err := kc.client.Partitions(kc.Topic)
	if err != nil {
		slog.Error("kafka get paritions failed", zap.Error(err))
		return
	}
	for parition := range partitionList {
		pc, err := kc.client.ConsumePartition(kc.Topic, int32(parition), sarama.OffsetNewest)
		if err != nil {
			slog.Error("failed to start consumer for partition", zap.Error(err))
			return
		}
		defer pc.AsyncClose()
		go func(sarama.PartitionConsumer){
			for msg := range pc.Messages() {
				slog.Info("消息", zap.Int32("msg.Partition", msg.Partition), zap.Int64("msg.Offset", msg.Offset),
					zap.Any("msg.Key", msg.Key), zap.Any("msg.Value", msg.Value))
			}
		}(pc)
	}
}

func (kc *KafkaConsumer)ConsumeOnePartion(a_id int32){
	pc, err := kc.client.ConsumePartition(kc.Topic, a_id, sarama.OffsetNewest)
	if err != nil {
		slog.Error("failed to start consumer for partition", zap.Error(err))
		return
	}
	defer pc.AsyncClose()
	
	for msg := range pc.Messages() {
		slog.Info("消息", zap.Int32("msg.Partition", msg.Partition), zap.Int64("msg.Offset", msg.Offset),
			zap.Any("msg.Key", msg.Key), zap.Any("msg.Value", msg.Value))
		kc.processFunc()
	}
}

func (kc *KafkaConsumer)Close() {
	kc.client.Close()
}