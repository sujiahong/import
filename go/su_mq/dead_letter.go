package su_mq

import (
	"context"
	"sync"
)

// DeadLetter 定义消费最终失败后发布死信消息的接口。
type DeadLetter interface {
	Publish(ctx context.Context, msg Message, err error) error
}

// NopDeadLetter 是不写入任何死信的默认实现。
type NopDeadLetter struct{}

// Publish 忽略死信消息并返回成功。
func (NopDeadLetter) Publish(ctx context.Context, msg Message, err error) error {
	return nil
}

// DeadLetterMessage 保存一条死信消息及其失败原因。
type DeadLetterMessage struct {
	Message Message // 原始消费消息。
	Err     error   // 消费最终失败原因。
}

// MemoryDeadLetter 将死信消息保存在内存中，主要用于测试或本地观察。
type MemoryDeadLetter struct {
	mu       sync.Mutex          // 保护 messages。
	messages []DeadLetterMessage // 内存死信消息列表。
}

// NewMemoryDeadLetter 创建内存死信队列。
func NewMemoryDeadLetter() *MemoryDeadLetter {
	return &MemoryDeadLetter{}
}

// Publish 将失败消息追加到内存死信队列。
func (d *MemoryDeadLetter) Publish(ctx context.Context, msg Message, err error) error {
	if d == nil {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.messages = append(d.messages, DeadLetterMessage{Message: msg, Err: err})
	return nil
}

// Messages 返回当前内存死信消息的快照。
func (d *MemoryDeadLetter) Messages() []DeadLetterMessage {
	if d == nil {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]DeadLetterMessage(nil), d.messages...)
}
