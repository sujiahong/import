package su_balance

import (
	"context"
	"sync/atomic"
)

type RoundRobin struct {
	next uint64
}

func NewRoundRobin() *RoundRobin {
	return &RoundRobin{}
}

func (b *RoundRobin) Pick(ctx context.Context, nodes []Node, key string) (Node, error) {
	available := healthyNodes(nodes)
	if len(available) == 0 {
		return Node{}, ErrNoAvailableNode
	}
	idx := atomic.AddUint64(&b.next, 1) - 1
	return available[int(idx%uint64(len(available)))], nil
}
