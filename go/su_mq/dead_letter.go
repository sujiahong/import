package su_mq

import (
	"context"
	"sync"
)

type DeadLetter interface {
	Publish(ctx context.Context, msg Message, err error) error
}

type NopDeadLetter struct{}

func (NopDeadLetter) Publish(ctx context.Context, msg Message, err error) error {
	return nil
}

type DeadLetterMessage struct {
	Message Message
	Err     error
}

type MemoryDeadLetter struct {
	mu       sync.Mutex
	messages []DeadLetterMessage
}

func NewMemoryDeadLetter() *MemoryDeadLetter {
	return &MemoryDeadLetter{}
}

func (d *MemoryDeadLetter) Publish(ctx context.Context, msg Message, err error) error {
	if d == nil {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.messages = append(d.messages, DeadLetterMessage{Message: msg, Err: err})
	return nil
}

func (d *MemoryDeadLetter) Messages() []DeadLetterMessage {
	if d == nil {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]DeadLetterMessage(nil), d.messages...)
}
