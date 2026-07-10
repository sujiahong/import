package su_mq

import "time"

var after = time.After

// RetryPolicy 定义消费失败后的下一次重试延迟和是否继续重试。
type RetryPolicy interface {
	Next(attempt int, err error) (delay time.Duration, retry bool)
}

// NoRetry 表示不进行任何重试。
type NoRetry struct{}

// Next 永远返回不重试。
func (NoRetry) Next(attempt int, err error) (time.Duration, bool) {
	return 0, false
}

// FixedRetry 使用固定延迟和最大尝试次数控制重试。
type FixedRetry struct {
	MaxAttempts int
	Delay       time.Duration
}

// Next 返回固定延迟，并在 attempt 未达到 MaxAttempts 时继续重试。
func (r FixedRetry) Next(attempt int, err error) (time.Duration, bool) {
	if r.MaxAttempts <= 0 {
		return 0, false
	}
	return r.Delay, attempt < r.MaxAttempts
}
