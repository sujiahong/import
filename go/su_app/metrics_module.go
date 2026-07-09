package su_app

import (
	"context"

	"go.local/su_metrics"
)

type MetricsModule struct {
	Metrics su_metrics.Metrics
}

func NewMetricsModule(metrics su_metrics.Metrics) *MetricsModule {
	return &MetricsModule{Metrics: metrics}
}

func (m *MetricsModule) Name() string {
	return "metrics"
}

func (m *MetricsModule) Start(ctx context.Context) error {
	if m == nil {
		return nil
	}
	if m.Metrics == nil {
		m.Metrics = su_metrics.NoopMetrics{}
	}
	su_metrics.Default = m.Metrics
	return nil
}

func (m *MetricsModule) Stop(ctx context.Context) error {
	return nil
}

func (m *MetricsModule) Get() su_metrics.Metrics {
	if m == nil || m.Metrics == nil {
		return su_metrics.NoopMetrics{}
	}
	return m.Metrics
}
