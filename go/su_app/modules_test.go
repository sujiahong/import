package su_app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	su_kafka "go.local/su_da/kafka"
	su_redis "go.local/su_da/redis"
	su_mysql "go.local/su_da/su_sql"
	"go.local/su_errors"
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

type fakeKafkaConsumer struct {
	consumes int
	closes   int
}

func (f *fakeKafkaConsumer) ConsumeAllPartion() {
	f.consumes++
}

func (f *fakeKafkaConsumer) Close() {
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

func TestKafkaConsumerModuleLifecycle(t *testing.T) {
	consumer := &fakeKafkaConsumer{}
	module := &KafkaConsumerModule{
		Config: su_kafka.KafkaConsumerConfig{Topic: "topic"},
		Factory: func(cfg su_kafka.KafkaConsumerConfig, handler su_kafka.HandleMessageFunc) (KafkaConsumerRunner, error) {
			return consumer, nil
		},
	}
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := module.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if consumer.consumes != 1 || consumer.closes != 1 {
		t.Fatalf("consumer lifecycle = %d/%d, want 1/1", consumer.consumes, consumer.closes)
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
