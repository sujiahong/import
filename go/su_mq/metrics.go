package su_mq

import (
	"strconv"

	"go.local/su_metrics"
)

// MQMetrics 定义消费成功、失败、重试和跳过时的指标回调。
type MQMetrics interface {
	ConsumeSuccess(msg Message)
	ConsumeError(msg Message, err error)
	ConsumeRetry(msg Message, attempt int, err error)
	ConsumeSkipped(msg Message)
}

// NopMQMetrics 是不记录任何指标的默认实现。
type NopMQMetrics struct{}

// ConsumeSuccess 忽略消费成功事件。
func (NopMQMetrics) ConsumeSuccess(msg Message) {}

// ConsumeError 忽略消费失败事件。
func (NopMQMetrics) ConsumeError(msg Message, err error) {}

// ConsumeRetry 忽略消费重试事件。
func (NopMQMetrics) ConsumeRetry(msg Message, attempt int, err error) {}

// ConsumeSkipped 忽略幂等跳过事件。
func (NopMQMetrics) ConsumeSkipped(msg Message) {}

// DefaultMQMetrics 将消费事件记录到 su_metrics。
type DefaultMQMetrics struct {
	Metrics su_metrics.Metrics
}

// NewDefaultMQMetrics 创建默认指标适配器；nil metrics 会使用 su_metrics.Default。
func NewDefaultMQMetrics(metrics su_metrics.Metrics) DefaultMQMetrics {
	if metrics == nil {
		metrics = su_metrics.Default
	}
	return DefaultMQMetrics{Metrics: metrics}
}

// ConsumeSuccess 记录消费成功计数。
func (m DefaultMQMetrics) ConsumeSuccess(msg Message) {
	m.metrics().IncCounter("mq_consume_total", labels(msg, "success", ""))
}

// ConsumeError 记录消费失败计数。
func (m DefaultMQMetrics) ConsumeError(msg Message, err error) {
	m.metrics().IncCounter("mq_consume_total", labels(msg, "error", ""))
}

// ConsumeRetry 记录消费重试计数和尝试次数。
func (m DefaultMQMetrics) ConsumeRetry(msg Message, attempt int, err error) {
	m.metrics().IncCounter("mq_consume_retry_total", labels(msg, "retry", strconv.Itoa(attempt)))
}

// ConsumeSkipped 记录幂等跳过计数。
func (m DefaultMQMetrics) ConsumeSkipped(msg Message) {
	m.metrics().IncCounter("mq_consume_total", labels(msg, "skipped", ""))
}

// metrics 返回实际使用的指标实现。
func (m DefaultMQMetrics) metrics() su_metrics.Metrics {
	if m.Metrics == nil {
		return su_metrics.NoopMetrics{}
	}
	return m.Metrics
}

// labels 生成消费指标的通用标签集合。
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
