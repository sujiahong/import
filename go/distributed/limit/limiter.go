package limit

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrRateLimitExceeded = errors.New("rate limit exceeded")

type RateLimiter interface {
	Allow(ctx context.Context) (bool, error)
}

type TokenBucketLimiter struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
}

func NewTokenBucketLimiter(rate float64, capacity float64) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		tokens:     capacity,
		maxTokens:  capacity,
		refillRate: rate,
		lastRefill: time.Now(),
	}
}

func (l *TokenBucketLimiter) Allow(ctx context.Context) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.refill()

	if l.tokens >= 1 {
		l.tokens--
		return true, nil
	}

	return false, ErrRateLimitExceeded
}

func (l *TokenBucketLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(l.lastRefill).Seconds()
	l.tokens += elapsed * l.refillRate
	if l.tokens > l.maxTokens {
		l.tokens = l.maxTokens
	}
	l.lastRefill = now
}

func (l *TokenBucketLimiter) Tokens() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refill()
	return l.tokens
}

func (l *TokenBucketLimiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tokens = l.maxTokens
	l.lastRefill = time.Now()
}

type SlidingWindowLimiter struct {
	mu          sync.Mutex
	maxRequests int
	windowSize  time.Duration
	requests    []time.Time
}

func NewSlidingWindowLimiter(maxRequests int, windowSize time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		maxRequests: maxRequests,
		windowSize:  windowSize,
		requests:    make([]time.Time, 0),
	}
}

func (l *SlidingWindowLimiter) Allow(ctx context.Context) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.windowSize)

	var validRequests []time.Time
	for _, t := range l.requests {
		if t.After(cutoff) {
			validRequests = append(validRequests, t)
		}
	}

	l.requests = validRequests

	if len(l.requests) < l.maxRequests {
		l.requests = append(l.requests, now)
		return true, nil
	}

	return false, ErrRateLimitExceeded
}

func (l *SlidingWindowLimiter) CurrentCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.windowSize)

	count := 0
	for _, t := range l.requests {
		if t.After(cutoff) {
			count++
		}
	}
	return count
}

func (l *SlidingWindowLimiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.requests = make([]time.Time, 0)
}

type LeakyBucketLimiter struct {
	mu       sync.Mutex
	capacity int
	rate     time.Duration
	water    int
	lastLeak time.Time
}

func NewLeakyBucketLimiter(rate time.Duration, capacity int) *LeakyBucketLimiter {
	return &LeakyBucketLimiter{
		capacity: capacity,
		rate:     rate,
		water:    0,
		lastLeak: time.Now(),
	}
}

func (l *LeakyBucketLimiter) Allow(ctx context.Context) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.leak()

	if l.water < l.capacity {
		l.water++
		return true, nil
	}

	return false, ErrRateLimitExceeded
}

func (l *LeakyBucketLimiter) leak() {
	now := time.Now()
	elapsed := now.Sub(l.lastLeak)
	leaks := int(elapsed / l.rate)
	l.water = max(0, l.water-leaks)
	l.lastLeak = now
}

func (l *LeakyBucketLimiter) Water() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.leak()
	return l.water
}

func (l *LeakyBucketLimiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.water = 0
	l.lastLeak = time.Now()
}

type MultiLimiter struct {
	limiters []RateLimiter
	mu       sync.RWMutex
}

func NewMultiLimiter(limiters ...RateLimiter) *MultiLimiter {
	return &MultiLimiter{
		limiters: limiters,
	}
}

func (m *MultiLimiter) Allow(ctx context.Context) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, limiter := range m.limiters {
		allowed, err := limiter.Allow(ctx)
		if !allowed || err != nil {
			return false, err
		}
	}
	return true, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type LimiterPool struct {
	limiters map[string]RateLimiter
	mu       sync.RWMutex
}

func NewLimiterPool() *LimiterPool {
	return &LimiterPool{
		limiters: make(map[string]RateLimiter),
	}
}

func (p *LimiterPool) Get(key string) RateLimiter {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.limiters[key]
}

func (p *LimiterPool) Set(key string, limiter RateLimiter) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.limiters[key] = limiter
}

func (p *LimiterPool) Delete(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.limiters, key)
}
