package su_mq

import (
	"context"
	"sync"
)

// Idempotency 定义消息幂等检查和处理成功后的标记接口。
type Idempotency interface {
	Seen(ctx context.Context, key string) (bool, error)
	Mark(ctx context.Context, key string) error
}

// NopIdempotency 是不做幂等检查的默认实现。
type NopIdempotency struct{}

// Seen 永远返回未处理。
func (NopIdempotency) Seen(ctx context.Context, key string) (bool, error) {
	return false, nil
}

// Mark 忽略幂等标记并返回成功。
func (NopIdempotency) Mark(ctx context.Context, key string) error {
	return nil
}

// MemoryIdempotency 使用内存 map 保存已处理消息 ID。
type MemoryIdempotency struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

// NewMemoryIdempotency 创建内存幂等记录器。
func NewMemoryIdempotency() *MemoryIdempotency {
	return &MemoryIdempotency{seen: make(map[string]struct{})}
}

// Seen 判断消息 ID 是否已在内存中标记。
func (i *MemoryIdempotency) Seen(ctx context.Context, key string) (bool, error) {
	if i == nil {
		return false, nil
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	_, ok := i.seen[key]
	return ok, nil
}

// Mark 将消息 ID 标记为已处理。
func (i *MemoryIdempotency) Mark(ctx context.Context, key string) error {
	if i == nil {
		return nil
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.seen[key] = struct{}{}
	return nil
}
