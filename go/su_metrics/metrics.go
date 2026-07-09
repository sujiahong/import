package su_metrics

import "sync"

type Labels map[string]string

type Metrics interface {
	IncCounter(name string, labels Labels)
	AddCounter(name string, value float64, labels Labels)
	SetGauge(name string, value float64, labels Labels)
	Observe(name string, value float64, labels Labels)
}

type NoopMetrics struct{}

func (NoopMetrics) IncCounter(name string, labels Labels)                {}
func (NoopMetrics) AddCounter(name string, value float64, labels Labels) {}
func (NoopMetrics) SetGauge(name string, value float64, labels Labels)   {}
func (NoopMetrics) Observe(name string, value float64, labels Labels)    {}

var Default Metrics = NoopMetrics{}

type MemoryMetrics struct {
	mu         sync.RWMutex
	counters   map[string]float64
	gauges     map[string]float64
	histograms map[string][]float64
}

func NewMemoryMetrics() *MemoryMetrics {
	return &MemoryMetrics{
		counters:   make(map[string]float64),
		gauges:     make(map[string]float64),
		histograms: make(map[string][]float64),
	}
}

func (m *MemoryMetrics) IncCounter(name string, labels Labels) {
	m.AddCounter(name, 1, labels)
}

func (m *MemoryMetrics) AddCounter(name string, value float64, labels Labels) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[metricKey(name, labels)] += value
}

func (m *MemoryMetrics) SetGauge(name string, value float64, labels Labels) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[metricKey(name, labels)] = value
}

func (m *MemoryMetrics) Observe(name string, value float64, labels Labels) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := metricKey(name, labels)
	m.histograms[key] = append(m.histograms[key], value)
}

func (m *MemoryMetrics) Counter(name string, labels Labels) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.counters[metricKey(name, labels)]
}

func (m *MemoryMetrics) Gauge(name string, labels Labels) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.gauges[metricKey(name, labels)]
}

func (m *MemoryMetrics) Observations(name string, labels Labels) []float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	values := m.histograms[metricKey(name, labels)]
	return append([]float64(nil), values...)
}
