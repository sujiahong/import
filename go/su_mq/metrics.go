package su_mq

import (
	"strconv"

	"go.local/su_metrics"
)

type MQMetrics interface {
	ConsumeSuccess(msg Message)
	ConsumeError(msg Message, err error)
	ConsumeRetry(msg Message, attempt int, err error)
	ConsumeSkipped(msg Message)
}

type NopMQMetrics struct{}

func (NopMQMetrics) ConsumeSuccess(msg Message)                       {}
func (NopMQMetrics) ConsumeError(msg Message, err error)              {}
func (NopMQMetrics) ConsumeRetry(msg Message, attempt int, err error) {}
func (NopMQMetrics) ConsumeSkipped(msg Message)                       {}

type DefaultMQMetrics struct {
	Metrics su_metrics.Metrics
}

func NewDefaultMQMetrics(metrics su_metrics.Metrics) DefaultMQMetrics {
	if metrics == nil {
		metrics = su_metrics.Default
	}
	return DefaultMQMetrics{Metrics: metrics}
}

func (m DefaultMQMetrics) ConsumeSuccess(msg Message) {
	m.metrics().IncCounter("mq_consume_total", labels(msg, "success", ""))
}

func (m DefaultMQMetrics) ConsumeError(msg Message, err error) {
	m.metrics().IncCounter("mq_consume_total", labels(msg, "error", ""))
}

func (m DefaultMQMetrics) ConsumeRetry(msg Message, attempt int, err error) {
	m.metrics().IncCounter("mq_consume_retry_total", labels(msg, "retry", strconv.Itoa(attempt)))
}

func (m DefaultMQMetrics) ConsumeSkipped(msg Message) {
	m.metrics().IncCounter("mq_consume_total", labels(msg, "skipped", ""))
}

func (m DefaultMQMetrics) metrics() su_metrics.Metrics {
	if m.Metrics == nil {
		return su_metrics.NoopMetrics{}
	}
	return m.Metrics
}

func labels(msg Message, status string, attempt string) su_metrics.Labels {
	labels := su_metrics.Labels{
		"source": msg.Source,
		"topic":  msg.Topic,
		"status": status,
	}
	if attempt != "" {
		labels["attempt"] = attempt
	}
	return labels
}
