package su_mq

import (
	"context"
	"fmt"
	"hash/fnv"
)

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

type Handler func(ctx context.Context, msg Message) error

type HandlerRegistry struct {
	handlers map[string]Handler
}

func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{handlers: make(map[string]Handler)}
}

func (r *HandlerRegistry) Register(topic string, handler Handler) {
	if r == nil || topic == "" || handler == nil {
		return
	}
	r.handlers[topic] = handler
}

func (r *HandlerRegistry) Handler(topic string) Handler {
	if r == nil {
		return nil
	}
	return r.handlers[topic]
}

type ProcessorOptions struct {
	RetryPolicy RetryPolicy
	DeadLetter  DeadLetter
	Idempotency Idempotency
	Metrics     MQMetrics
}

type Processor struct {
	opts ProcessorOptions
}

func NewProcessor(opts ProcessorOptions) *Processor {
	return &Processor{opts: normalizeProcessorOptions(opts)}
}

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
