package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type ServiceInstance struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Address   string            `json:"address"`
	Port      int               `json:"port"`
	Metadata  map[string]string `json:"metadata"`
	Version   string            `json:"version"`
	Status    string            `json:"status"`
	Heartbeat time.Time         `json:"heartbeat"`
}

type ServiceRegistry interface {
	Register(ctx context.Context, instance *ServiceInstance) error
	Deregister(ctx context.Context, serviceName, instanceID string) error
	Discover(ctx context.Context, serviceName string) ([]*ServiceInstance, error)
	Watch(serviceName string, handler func([]*ServiceInstance)) error
	Heartbeat(ctx context.Context, instanceID string) error
}

type inMemoryRegistry struct {
	services map[string]map[string]*ServiceInstance
	watchers map[string][]chan []*ServiceInstance
	mu       sync.RWMutex
}

func NewInMemoryRegistry() ServiceRegistry {
	return &inMemoryRegistry{
		services: make(map[string]map[string]*ServiceInstance),
		watchers: make(map[string][]chan []*ServiceInstance),
	}
}

func (r *inMemoryRegistry) Register(ctx context.Context, instance *ServiceInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	instance.Status = "healthy"
	instance.Heartbeat = time.Now()

	if _, ok := r.services[instance.Name]; !ok {
		r.services[instance.Name] = make(map[string]*ServiceInstance)
	}
	r.services[instance.Name][instance.ID] = instance

	r.notifyWatchers(instance.Name)
	return nil
}

func (r *inMemoryRegistry) Deregister(ctx context.Context, serviceName, instanceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if instances, ok := r.services[serviceName]; ok {
		delete(instances, instanceID)
		r.notifyWatchers(serviceName)
	}
	return nil
}

func (r *inMemoryRegistry) Discover(ctx context.Context, serviceName string) ([]*ServiceInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if instances, ok := r.services[serviceName]; ok {
		result := make([]*ServiceInstance, 0, len(instances))
		for _, inst := range instances {
			if inst.Status == "healthy" {
				result = append(result, inst)
			}
		}
		return result, nil
	}
	return nil, fmt.Errorf("service %s not found", serviceName)
}

func (r *inMemoryRegistry) Watch(serviceName string, handler func([]*ServiceInstance)) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ch := make(chan []*ServiceInstance, 10)
	r.watchers[serviceName] = append(r.watchers[serviceName], ch)

	go func() {
		for {
			select {
			case instances := <-ch:
				handler(instances)
			}
		}
	}()
	return nil
}

func (r *inMemoryRegistry) Heartbeat(ctx context.Context, instanceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, instances := range r.services {
		if inst, ok := instances[instanceID]; ok {
			inst.Heartbeat = time.Now()
			inst.Status = "healthy"
			return nil
		}
	}
	return fmt.Errorf("instance %s not found", instanceID)
}

func (r *inMemoryRegistry) notifyWatchers(serviceName string) {
	if watchers, ok := r.watchers[serviceName]; ok {
		instances, _ := r.Discover(context.Background(), serviceName)
		for _, w := range watchers {
			select {
			case w <- instances:
			default:
			}
		}
	}
}

func (r *inMemoryRegistry) GetAllInstances(serviceName string) []*ServiceInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if instances, ok := r.services[serviceName]; ok {
		result := make([]*ServiceInstance, 0, len(instances))
		for _, inst := range instances {
			result = append(result, inst)
		}
		return result
	}
	return nil
}

func (r *inMemoryRegistry) RemoveExpiredInstances(timeout time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for serviceName, instances := range r.services {
		for id, inst := range instances {
			if now.Sub(inst.Heartbeat) > timeout {
				delete(instances, id)
				r.notifyWatchers(serviceName)
			}
		}
	}
}

type RegistryClient struct {
	registry ServiceRegistry
	self     *ServiceInstance
}

func NewRegistryClient(registry ServiceRegistry, self *ServiceInstance) *RegistryClient {
	return &RegistryClient{
		registry: registry,
		self:     self,
	}
}

func (c *RegistryClient) StartHeartbeat(ctx context.Context, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := c.registry.Heartbeat(ctx, c.self.ID); err != nil {
				return err
			}
		}
	}
}

func (c *RegistryClient) RegisterAndKeepalive(ctx context.Context) error {
	if err := c.registry.Register(ctx, c.self); err != nil {
		return err
	}
	return c.StartHeartbeat(ctx, 5*time.Second)
}

type ServiceInstanceJSON struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Address   string            `json:"address"`
	Port      int               `json:"port"`
	Metadata  map[string]string `json:"metadata"`
	Version   string            `json:"version"`
	Status    string            `json:"status"`
	Heartbeat time.Time         `json:"heartbeat"`
}

func (s *ServiceInstance) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

func ServiceInstanceFromJSON(data []byte) (*ServiceInstance, error) {
	var s ServiceInstance
	err := json.Unmarshal(data, &s)
	return &s, err
}
