package su_app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	su_redis "go.local/su_da/redis"
	su_mysql "go.local/su_da/su_sql"
	"go.local/su_errors"
	"go.local/su_metrics"
	"go.local/su_mq"
)

type fakeConnector struct {
	starts   int
	stops    int
	startErr error
	stopErr  error
}

func (f *fakeConnector) Connect() error {
	f.starts++
	return f.startErr
}

func (f *fakeConnector) Close() error {
	f.stops++
	return f.stopErr
}

type fakeRedisMQConsumer struct {
	starts int
	stops  int
}

func (f *fakeRedisMQConsumer) Start() error {
	f.starts++
	return nil
}

func (f *fakeRedisMQConsumer) Close() error {
	f.stops++
	return nil
}

type fakeKafkaMQConsumer struct {
	startAll       int
	startPartition int
	partitionID    int32
	closes         int
}

func (f *fakeKafkaMQConsumer) StartAllPartitions() error {
	f.startAll++
	return nil
}

func (f *fakeKafkaMQConsumer) StartPartition(partitionID int32) error {
	f.startPartition++
	f.partitionID = partitionID
	return nil
}

func (f *fakeKafkaMQConsumer) Close() {
	f.closes++
}

func TestConfigModuleLoadsConfig(t *testing.T) {
	type cfg struct {
		Name string `json:"name"`
		Port int    `json:"port"`
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"name":"svc","port":1000}`), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("APP_PORT", "2000")
	var c cfg
	module := NewConfigModule(path, "APP", &c)
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if c.Name != "svc" || c.Port != 2000 {
		t.Fatalf("config = %+v", c)
	}
}

func TestRedisModuleLifecycle(t *testing.T) {
	conn := &fakeConnector{}
	module := &RedisModule{
		Config: su_redis.RedisConfig{RemoteAddr: "127.0.0.1:6379"},
		Factory: func(cfg su_redis.RedisConfig) (RedisConnector, error) {
			return conn, nil
		},
	}
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := module.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if conn.starts != 1 || conn.stops != 1 {
		t.Fatalf("connector lifecycle = %d/%d, want 1/1", conn.starts, conn.stops)
	}
}

func TestMysqlModuleLifecycle(t *testing.T) {
	conn := &fakeConnector{}
	module := &MysqlModule{
		Config: su_mysql.MysqlConfig{DSN: "u:p@tcp(127.0.0.1:3306)/db"},
		Factory: func(cfg su_mysql.MysqlConfig) (MysqlConnector, error) {
			return conn, nil
		},
	}
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := module.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if conn.starts != 1 || conn.stops != 1 {
		t.Fatalf("connector lifecycle = %d/%d, want 1/1", conn.starts, conn.stops)
	}
}

func TestRedisMQModuleLifecycle(t *testing.T) {
	consumer := &fakeRedisMQConsumer{}
	module := &RedisMQModule{
		Config: su_mq.RedisListConsumerConfig{ListKey: "jobs"},
		Factory: func(cfg su_mq.RedisListConsumerConfig, handler su_mq.RedisListHandler) (RedisMQConsumer, error) {
			return consumer, nil
		},
	}
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := module.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if consumer.starts != 1 || consumer.stops != 1 {
		t.Fatalf("consumer lifecycle = %d/%d, want 1/1", consumer.starts, consumer.stops)
	}
}

func TestKafkaMQModuleLifecycle(t *testing.T) {
	consumer := &fakeKafkaMQConsumer{}
	module := &KafkaMQModule{
		Config: su_mq.KafkaConsumerConfig{Topic: "topic", AddrSlice: []string{"127.0.0.1:9092"}},
		Factory: func(cfg su_mq.KafkaConsumerConfig, handler su_mq.KafkaMessageHandler) (KafkaMQConsumer, error) {
			return consumer, nil
		},
	}
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := module.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if consumer.startAll != 1 || consumer.closes != 1 {
		t.Fatalf("consumer lifecycle = %d/%d, want 1/1", consumer.startAll, consumer.closes)
	}
}

func TestKafkaMQModuleStartsSinglePartition(t *testing.T) {
	consumer := &fakeKafkaMQConsumer{}
	partitionID := int32(3)
	module := &KafkaMQModule{
		Config:      su_mq.KafkaConsumerConfig{Topic: "topic", AddrSlice: []string{"127.0.0.1:9092"}},
		PartitionID: &partitionID,
		Factory: func(cfg su_mq.KafkaConsumerConfig, handler su_mq.KafkaMessageHandler) (KafkaMQConsumer, error) {
			return consumer, nil
		},
	}
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if consumer.startPartition != 1 || consumer.partitionID != partitionID || consumer.startAll != 0 {
		t.Fatalf("partition start = %d/%d all=%d, want 1/%d all=0", consumer.startPartition, consumer.partitionID, consumer.startAll, partitionID)
	}
}

func TestMetricsModuleLifecycle(t *testing.T) {
	metrics := su_metrics.NewMemoryMetrics()
	module := NewMetricsModule(metrics)
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	su_metrics.Default.IncCounter("requests", su_metrics.Labels{"route": "login"})
	if got := metrics.Counter("requests", su_metrics.Labels{"route": "login"}); got != 1 {
		t.Fatalf("counter = %v, want 1", got)
	}
	if module.Get() != metrics {
		t.Fatal("module Get() did not return configured metrics")
	}
	if err := module.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestMetricsModuleDefaultsToNoop(t *testing.T) {
	module := NewMetricsModule(nil)
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if module.Get() == nil {
		t.Fatal("metrics should not be nil")
	}
}

func TestModuleErrorCodes(t *testing.T) {
	if err := (*ConfigModule)(nil).Start(context.Background()); su_errors.CodeOf(err) != su_errors.CodeInvalidArgument {
		t.Fatalf("ConfigModule code = %d, want invalid argument", su_errors.CodeOf(err))
	}
	if err := NewLogModule("").Start(context.Background()); su_errors.CodeOf(err) != su_errors.CodeInvalidArgument {
		t.Fatalf("LogModule code = %d, want invalid argument", su_errors.CodeOf(err))
	}

	depErr := errors.New("redis down")
	redisModule := &RedisModule{Client: &fakeConnector{startErr: depErr}}
	err := redisModule.Start(context.Background())
	if su_errors.CodeOf(err) != su_errors.CodeUnavailable {
		t.Fatalf("RedisModule code = %d, want unavailable", su_errors.CodeOf(err))
	}
	if !su_errors.Retryable(err) {
		t.Fatal("RedisModule error should be retryable")
	}
	if !errors.Is(err, depErr) {
		t.Fatal("RedisModule error should wrap dependency error")
	}
}
