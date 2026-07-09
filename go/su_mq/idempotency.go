package su_mq

import (
	"context"
	"sync"
)

type Idempotency interface {
	Seen(ctx context.Context, key string) (bool, error)
	Mark(ctx context.Context, key string) error
}

type NopIdempotency struct{}

func (NopIdempotency) Seen(ctx context.Context, key string) (bool, error) {
	return false, nil
}

func (NopIdempotency) Mark(ctx context.Context, key string) error {
	return nil
}

type MemoryIdempotency struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func NewMemoryIdempotency() *MemoryIdempotency {
	return &MemoryIdempotency{seen: make(map[string]struct{})}
}

func (i *MemoryIdempotency) Seen(ctx context.Context, key string) (bool, error) {
	if i == nil {
		return false, nil
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	_, ok := i.seen[key]
	return ok, nil
}

func (i *MemoryIdempotency) Mark(ctx context.Context, key string) error {
	if i == nil {
		return nil
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.seen[key] = struct{}{}
	return nil
}
