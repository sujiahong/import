package su_balance

import (
	"context"
	"errors"
)

type Node struct {
	ID      string
	Addr    string
	Weight  int
	Healthy bool
	Meta    map[string]string
}

type Balancer interface {
	Pick(ctx context.Context, nodes []Node, key string) (Node, error)
}

var ErrNoAvailableNode = errors.New("no available node")

func healthyNodes(nodes []Node) []Node {
	available := make([]Node, 0, len(nodes))
	for _, node := range nodes {
		if node.ID == "" && node.Addr == "" {
			continue
		}
		if !node.Healthy {
			continue
		}
		weight := node.Weight
		if weight <= 0 {
			weight = 1
		}
		for i := 0; i < weight; i++ {
			available = append(available, node)
		}
	}
	return available
}
