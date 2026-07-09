package su_balance

import (
	"context"
	"errors"
	"testing"
)

func TestRoundRobinSkipsUnhealthy(t *testing.T) {
	b := NewRoundRobin()
	nodes := []Node{
		{ID: "bad", Healthy: false},
		{ID: "a", Healthy: true},
		{ID: "b", Healthy: true},
	}
	first, err := b.Pick(context.Background(), nodes, "")
	if err != nil {
		t.Fatalf("Pick() error = %v", err)
	}
	second, err := b.Pick(context.Background(), nodes, "")
	if err != nil {
		t.Fatalf("Pick() error = %v", err)
	}
	if first.ID != "a" || second.ID != "b" {
		t.Fatalf("picked %s then %s, want a then b", first.ID, second.ID)
	}
}

func TestHashStable(t *testing.T) {
	b := NewHash()
	nodes := []Node{{ID: "a", Healthy: true}, {ID: "b", Healthy: true}}
	first, err := b.Pick(context.Background(), nodes, "user-1")
	if err != nil {
		t.Fatalf("Pick() error = %v", err)
	}
	for i := 0; i < 10; i++ {
		got, err := b.Pick(context.Background(), nodes, "user-1")
		if err != nil {
			t.Fatalf("Pick() error = %v", err)
		}
		if got.ID != first.ID {
			t.Fatalf("hash pick changed: %s -> %s", first.ID, got.ID)
		}
	}
}

func TestNoAvailableNode(t *testing.T) {
	_, err := NewRoundRobin().Pick(context.Background(), nil, "")
	if !errors.Is(err, ErrNoAvailableNode) {
		t.Fatalf("Pick() error = %v, want ErrNoAvailableNode", err)
	}
}
