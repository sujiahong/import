package su_mq

import (
	"context"
	"fmt"
	"hash/fnv"
)

// Message 是 su_mq 内部统一的消费消息模型。
type Message struct {
	ID        string
	Source    string
	Topic     string
	Key       []byte
	Value     []byte
	Headers   map[string]string
	Partition int32
	Offset    int64
	Raw       any
}

// Handler 处理一条标准化后的消息。
type Handler func(ctx context.Context, msg Message) error

// HandlerRegistry 按 topic 保存消息处理函数。
type HandlerRegistry struct {
	handlers map[string]Handler
}

// NewHandlerRegistry 创建空的 handler registry。
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{handlers: make(map[string]Handler)}
}

// Register 为 topic 注册处理函数；空 topic 或 nil handler 会被忽略。
func (r *HandlerRegistry) Register(topic string, handler Handler) {
	if r == nil || topic == "" || handler == nil {
		return
	}
	r.handlers[topic] = handler
}

// Handler 返回 topic 对应的处理函数。
func (r *HandlerRegistry) Handler(topic string) Handler {
	if r == nil {
		return nil
	}
	return r.handlers[topic]
}

// ProcessorOptions 定义消息处理的重试、死信、幂等和指标插件。
type ProcessorOptions struct {
	RetryPolicy RetryPolicy
	DeadLetter  DeadLetter
	Idempotency Idempotency
	Metrics     MQMetrics
}

// Processor 负责围绕业务 handler 执行幂等检查、重试、死信和指标记录。
type Processor struct {
	opts ProcessorOptions
}

// NewProcessor 创建消息处理器，并补齐缺省组件。
func NewProcessor(opts ProcessorOptions) *Processor {
	return &Processor{opts: normalizeProcessorOptions(opts)}
}

// Process 执行一条消息的完整处理流程。
func (p *Processor) Process(ctx context.Context, msg Message, handler Handler) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if handler == nil {
		return nil
	}
	opts := normalizeProcessorOptions(p.opts)
	msgID := messageID(msg)
	if opts.Idempotency != nil {
		seen, err := opts.Idempotency.Seen(ctx, msgID)
		if err != nil {
			opts.Metrics.ConsumeError(msg, err)
			return err
		}
		if seen {
			opts.Metrics.ConsumeSkipped(msg)
			return nil
		}
	}

	var err error
	for attempt := 0; ; attempt++ {
		err = handler(ctx, msg)
		if err == nil {
			if opts.Idempotency != nil {
				if markErr := opts.Idempotency.Mark(ctx, msgID); markErr != nil {
					opts.Metrics.ConsumeError(msg, markErr)
					return markErr
				}
			}
			opts.Metrics.ConsumeSuccess(msg)
			return nil
		}
		delay, retry := opts.RetryPolicy.Next(attempt, err)
		if !retry {
			break
		}
		opts.Metrics.ConsumeRetry(msg, attempt+1, err)
		if delay > 0 {
			select {
			case <-ctx.Done():
				opts.Metrics.ConsumeError(msg, ctx.Err())
				return ctx.Err()
			case <-after(delay):
			}
		}
	}

	opts.Metrics.ConsumeError(msg, err)
	if opts.DeadLetter != nil {
		if dlqErr := opts.DeadLetter.Publish(ctx, msg, err); dlqErr != nil {
			return dlqErr
		}
	}
	return err
}

// normalizeProcessorOptions 为未设置的处理组件填充 no-op 默认实现。
func normalizeProcessorOptions(opts ProcessorOptions) ProcessorOptions {
	if opts.RetryPolicy == nil {
		opts.RetryPolicy = NoRetry{}
	}
	if opts.DeadLetter == nil {
		opts.DeadLetter = NopDeadLetter{}
	}
	if opts.Idempotency == nil {
		opts.Idempotency = NopIdempotency{}
	}
	if opts.Metrics == nil {
		opts.Metrics = NopMQMetrics{}
	}
	return opts
}

// messageID 生成幂等使用的消息 ID，优先使用显式 ID 和 Kafka offset。
func messageID(msg Message) string {
	if msg.ID != "" {
		return msg.ID
	}
	if msg.Source == "kafka" && msg.Offset >= 0 {
		return fmt.Sprintf("%s:%s:%d:%d", msg.Source, msg.Topic, msg.Partition, msg.Offset)
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(msg.Source))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(msg.Topic))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(msg.Key)
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(msg.Value)
	return fmt.Sprintf("%s:%s:%x", msg.Source, msg.Topic, h.Sum64())
}
