package su_balance

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

type Random struct {
	mu   sync.Mutex
	rand *rand.Rand
}

func NewRandom() *Random {
	return &Random{rand: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

func (b *Random) Pick(ctx context.Context, nodes []Node, key string) (Node, error) {
	available := healthyNodes(nodes)
	if len(available) == 0 {
		return Node{}, ErrNoAvailableNode
	}
	b.mu.Lock()
	idx := b.rand.Intn(len(available))
	b.mu.Unlock()
	return available[idx], nil
}
