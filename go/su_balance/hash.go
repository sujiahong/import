package su_balance

import (
	"context"
	"hash/fnv"
)

type Hash struct{}

func NewHash() *Hash {
	return &Hash{}
}

func (b *Hash) Pick(ctx context.Context, nodes []Node, key string) (Node, error) {
	available := healthyNodes(nodes)
	if len(available) == 0 {
		return Node{}, ErrNoAvailableNode
	}
	if key == "" {
		key = available[0].ID + available[0].Addr
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return available[int(h.Sum32()%uint32(len(available)))], nil
}
