package su_metrics

import "testing"

func TestMemoryMetrics(t *testing.T) {
	m := NewMemoryMetrics()
	labels := Labels{"route": "1"}
	m.IncCounter("rpc_total", labels)
	m.AddCounter("rpc_total", 2, labels)
	m.SetGauge("conn", 10, nil)
	m.Observe("latency", 1.5, labels)

	if got := m.Counter("rpc_total", labels); got != 3 {
		t.Fatalf("counter = %v, want 3", got)
	}
	if got := m.Gauge("conn", nil); got != 10 {
		t.Fatalf("gauge = %v, want 10", got)
	}
	if got := m.Observations("latency", labels); len(got) != 1 || got[0] != 1.5 {
		t.Fatalf("observations = %v", got)
	}
}
