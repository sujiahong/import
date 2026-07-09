package su_mq

import "time"

var after = time.After

type RetryPolicy interface {
	Next(attempt int, err error) (delay time.Duration, retry bool)
}

type NoRetry struct{}

func (NoRetry) Next(attempt int, err error) (time.Duration, bool) {
	return 0, false
}

type FixedRetry struct {
	MaxAttempts int
	Delay       time.Duration
}

func (r FixedRetry) Next(attempt int, err error) (time.Duration, bool) {
	if r.MaxAttempts <= 0 {
		return 0, false
	}
	return r.Delay, attempt < r.MaxAttempts
}
