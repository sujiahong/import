package lb

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"distributed/registry"
)

type Balancer interface {
	Select(ctx context.Context, instances []*registry.ServiceInstance) (*registry.ServiceInstance, error)
}

type RoundRobinBalancer struct {
	mu       sync.Mutex
	counters map[string]uint64
}

func NewRoundRobinBalancer() *RoundRobinBalancer {
	return &RoundRobinBalancer{
		counters: make(map[string]uint64),
	}
}

func (b *RoundRobinBalancer) Select(ctx context.Context, instances []*registry.ServiceInstance) (*registry.ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, ErrNoInstances
	}

	b.mu.Lock()
	count := b.counters[""]
	b.counters[""]++
	b.mu.Unlock()

	return instances[count%uint64(len(instances))], nil
}

type RandomBalancer struct {
	random *rand.Rand
	mu     sync.Mutex
}

func NewRandomBalancer() *RandomBalancer {
	return &RandomBalancer{
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (b *RandomBalancer) Select(ctx context.Context, instances []*registry.ServiceInstance) (*registry.ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, ErrNoInstances
	}

	b.mu.Lock()
	index := b.random.Intn(len(instances))
	b.mu.Unlock()

	return instances[index], nil
}

type WeightedRoundRobinBalancer struct {
	mu       sync.Mutex
	counters map[string]uint64
	weights  map[string]int
}

func NewWeightedRoundRobinBalancer() *WeightedRoundRobinBalancer {
	return &WeightedRoundRobinBalancer{
		counters: make(map[string]uint64),
		weights:  make(map[string]int),
	}
}

func (b *WeightedRoundRobinBalancer) SetWeight(instanceID string, weight int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.weights[instanceID] = weight
}

func (b *WeightedRoundRobinBalancer) Select(ctx context.Context, instances []*registry.ServiceInstance) (*registry.ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, ErrNoInstances
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	totalWeight := 0
	for _, inst := range instances {
		weight := b.weights[inst.ID]
		if weight <= 0 {
			weight = 1
		}
		totalWeight += weight
	}

	count := b.counters[""]
	b.counters[""]++

	cur := uint64(0)
	for _, inst := range instances {
		weight := b.weights[inst.ID]
		if weight <= 0 {
			weight = 1
		}
		cur += uint64(weight)
		if count%uint64(totalWeight) < cur {
			return inst, nil
		}
	}

	return instances[0], nil
}

type LeastConnectionsBalancer struct {
	mu          sync.Mutex
	connections map[string]int
}

func NewLeastConnectionsBalancer() *LeastConnectionsBalancer {
	return &LeastConnectionsBalancer{
		connections: make(map[string]int),
	}
}

func (b *LeastConnectionsBalancer) AddConnection(instanceID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.connections[instanceID]++
}

func (b *LeastConnectionsBalancer) RemoveConnection(instanceID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.connections[instanceID] > 0 {
		b.connections[instanceID]--
	}
}

func (b *LeastConnectionsBalancer) Select(ctx context.Context, instances []*registry.ServiceInstance) (*registry.ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, ErrNoInstances
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	var selected *registry.ServiceInstance
	minConn := int(^uint(0) >> 1)

	for _, inst := range instances {
		conn := b.connections[inst.ID]
		if conn < minConn {
			minConn = conn
			selected = inst
		}
	}

	if selected != nil {
		b.connections[selected.ID]++
	}

	return selected, nil
}

type ConsistentHashBalancer struct {
	mu           sync.Mutex
	hashRing     map[uint64]string
	sortedKeys   []uint64
	virtualNodes int
}

func NewConsistentHashBalancer(virtualNodes int) *ConsistentHashBalancer {
	return &ConsistentHashBalancer{
		hashRing:     make(map[uint64]string),
		virtualNodes: virtualNodes,
	}
}

func (b *ConsistentHashBalancer) AddInstance(instanceID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i := 0; i < b.virtualNodes; i++ {
		key := hash(fmt.Sprintf("%s-%d", instanceID, i))
		b.hashRing[key] = instanceID
		b.sortedKeys = append(b.sortedKeys, key)
	}
	b.sortKeys()
}

func (b *ConsistentHashBalancer) RemoveInstance(instanceID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i := 0; i < b.virtualNodes; i++ {
		key := hash(fmt.Sprintf("%s-%d", instanceID, i))
		delete(b.hashRing, key)
		b.sortedKeys = b.removeKey(b.sortedKeys, key)
	}
}

func (b *ConsistentHashBalancer) sortKeys() {
	for i := 0; i < len(b.sortedKeys)-1; i++ {
		for j := i + 1; j < len(b.sortedKeys); j++ {
			if b.sortedKeys[i] > b.sortedKeys[j] {
				b.sortedKeys[i], b.sortedKeys[j] = b.sortedKeys[j], b.sortedKeys[i]
			}
		}
	}
}

func (b *ConsistentHashBalancer) removeKey(keys []uint64, key uint64) []uint64 {
	result := make([]uint64, 0, len(keys))
	for _, k := range keys {
		if k != key {
			result = append(result, k)
		}
	}
	return result
}

func (b *ConsistentHashBalancer) Select(ctx context.Context, instances []*registry.ServiceInstance) (*registry.ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, ErrNoInstances
	}

	key := hash(time.Now().String())
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, k := range b.sortedKeys {
		if key <= k {
			instanceID := b.hashRing[k]
			for _, inst := range instances {
				if inst.ID == instanceID {
					return inst, nil
				}
			}
		}
	}

	return instances[0], nil
}

func hash(s string) uint64 {
	h := uint64(5381)
	for i := 0; i < len(s); i++ {
		h = h*33 + uint64(s[i])
	}
	return h
}

type HealthCheckBalancer struct {
	balancer    Balancer
	healthMap   map[string]bool
	mu          sync.RWMutex
	checkPeriod time.Duration
}

func NewHealthCheckBalancer(balancer Balancer, checkPeriod time.Duration) *HealthCheckBalancer {
	return &HealthCheckBalancer{
		balancer:    balancer,
		healthMap:   make(map[string]bool),
		checkPeriod: checkPeriod,
	}
}

func (b *HealthCheckBalancer) SetHealthy(instanceID string, healthy bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.healthMap[instanceID] = healthy
}

func (b *HealthCheckBalancer) Select(ctx context.Context, instances []*registry.ServiceInstance) (*registry.ServiceInstance, error) {
	b.mu.RLock()
	healthyInstances := make([]*registry.ServiceInstance, 0, len(instances))
	for _, inst := range instances {
		if b.healthMap[inst.ID] || !b.healthMap[inst.ID] && len(b.healthMap) == 0 {
			healthyInstances = append(healthyInstances, inst)
		}
	}
	b.mu.RUnlock()

	if len(healthyInstances) == 0 {
		return nil, ErrNoHealthyInstances
	}

	return b.balancer.Select(ctx, healthyInstances)
}
