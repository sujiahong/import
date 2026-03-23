package health

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var ErrHealthCheckFailed = errors.New("health check failed")

type CheckResult struct {
	InstanceID string
	Healthy    bool
	Latency    time.Duration
	Error      error
	CheckTime  time.Time
}

type HealthChecker interface {
	Check(ctx context.Context, instanceID, addr string, port int) *CheckResult
	Start(ctx context.Context) error
	Stop() error
}

type TCPHealthChecker struct {
	timeout time.Duration
}

func NewTCPHealthChecker(timeout time.Duration) *TCPHealthChecker {
	return &TCPHealthChecker{
		timeout: timeout,
	}
}

func (c *TCPHealthChecker) Start(ctx context.Context) error {
	return nil
}

func (c *TCPHealthChecker) Stop() error {
	return nil
}

func (c *TCPHealthChecker) Check(ctx context.Context, instanceID, addr string, port int) *CheckResult {
	start := time.Now()

	address := net.JoinHostPort(addr, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", address, c.timeout)

	result := &CheckResult{
		InstanceID: instanceID,
		CheckTime:  time.Now(),
	}

	if err != nil {
		result.Healthy = false
		result.Error = err
		result.Latency = time.Since(start)
		return result
	}
	defer conn.Close()

	result.Healthy = true
	result.Latency = time.Since(start)
	return result
}

type HTTPHealthChecker struct {
	timeout   time.Duration
	checkPath string
}

func NewHTTPHealthChecker(timeout time.Duration, checkPath string) *HTTPHealthChecker {
	return &HTTPHealthChecker{
		timeout:   timeout,
		checkPath: checkPath,
	}
}

func (c *HTTPHealthChecker) Start(ctx context.Context) error {
	return nil
}

func (c *HTTPHealthChecker) Stop() error {
	return nil
}

func (c *HTTPHealthChecker) Check(ctx context.Context, instanceID, addr string, port int) *CheckResult {
	start := time.Now()

	result := &CheckResult{
		InstanceID: instanceID,
		CheckTime:  time.Now(),
	}

	url := "http://" + net.JoinHostPort(addr, strconv.Itoa(port)) + c.checkPath

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		result.Healthy = false
		result.Error = err
		result.Latency = time.Since(start)
		return result
	}

	client := &http.Client{Timeout: c.timeout}
	resp, err := client.Do(req)
	if err != nil {
		result.Healthy = false
		result.Error = err
		result.Latency = time.Since(start)
		return result
	}
	defer resp.Body.Close()

	result.Healthy = resp.StatusCode >= 200 && resp.StatusCode < 300
	result.Latency = time.Since(start)
	return result
}

type HealthCheckManager struct {
	checker     HealthChecker
	instances   map[string]*HealthStatus
	mu          sync.RWMutex
	checkPeriod time.Duration
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

type HealthStatus struct {
	InstanceID  string
	Address     string
	Port        int
	Healthy     bool
	LastCheck   time.Time
	Latency     time.Duration
	Consecutive int
}

func NewHealthCheckManager(checker HealthChecker, period time.Duration) *HealthCheckManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &HealthCheckManager{
		checker:     checker,
		instances:   make(map[string]*HealthStatus),
		checkPeriod: period,
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (m *HealthCheckManager) AddInstance(instanceID, addr string, port int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.instances[instanceID] = &HealthStatus{
		InstanceID: instanceID,
		Address:    addr,
		Port:       port,
		Healthy:    true,
	}
}

func (m *HealthCheckManager) RemoveInstance(instanceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.instances, instanceID)
}

func (m *HealthCheckManager) GetHealth(instanceID string) (bool, time.Time, time.Duration) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if status, ok := m.instances[instanceID]; ok {
		return status.Healthy, status.LastCheck, status.Latency
	}
	return false, time.Time{}, 0
}

func (m *HealthCheckManager) GetAllHealth() map[string]*HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*HealthStatus)
	for k, v := range m.instances {
		result[k] = v
	}
	return result
}

func (m *HealthCheckManager) Start(ctx context.Context) error {
	m.wg.Add(1)
	go m.checkLoop()

	m.wg.Add(1)
	go m.cleanLoop(ctx)

	return nil
}

func (m *HealthCheckManager) Stop() {
	m.cancel()
	m.wg.Wait()
}

func (m *HealthCheckManager) checkLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.checkPeriod)
	defer ticker.Stop()

	m.checkAll()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkAll()
		}
	}
}

func (m *HealthCheckManager) cleanLoop(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.cleanStaleEntries()
		}
	}
}

func (m *HealthCheckManager) checkAll() {
	m.mu.RLock()
	instances := make([]string, 0, len(m.instances))
	for id := range m.instances {
		instances = append(instances, id)
	}
	m.mu.RUnlock()

	var wg sync.WaitGroup
	for _, id := range instances {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			select {
			case <-m.ctx.Done():
				return
			default:
			}

			m.mu.RLock()
			status := m.instances[id]
			m.mu.RUnlock()

			if status != nil {
				result := m.checker.Check(m.ctx, id, status.Address, status.Port)
				m.updateStatus(result)
			}
		}(id)
	}
	wg.Wait()
}

func (m *HealthCheckManager) updateStatus(result *CheckResult) {
	m.mu.Lock()
	defer m.mu.Unlock()

	status, ok := m.instances[result.InstanceID]
	if !ok {
		return
	}

	status.LastCheck = result.CheckTime
	status.Latency = result.Latency

	if result.Healthy {
		status.Consecutive = 0
		status.Healthy = true
	} else {
		status.Consecutive++
		if status.Consecutive >= 3 {
			status.Healthy = false
		}
	}
}

func (m *HealthCheckManager) cleanStaleEntries() {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-5 * time.Minute)
	for _, status := range m.instances {
		if status.LastCheck.Before(cutoff) {
			status.Healthy = false
		}
	}
}
