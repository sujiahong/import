package circuit

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrCircuitOpen = errors.New("circuit is open")

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

type Circuit struct {
	mu               sync.RWMutex
	failures         int
	successes        int
	state            State
	threshold        int
	timeout          time.Duration
	halfOpenRequests int
	lastFailureTime  time.Time

	onOpen     func()
	onClose    func()
	onHalfOpen func()
}

func NewCircuit(threshold int, timeout time.Duration) *Circuit {
	return &Circuit{
		state:     StateClosed,
		threshold: threshold,
		timeout:   timeout,
	}
}

func (c *Circuit) Execute(ctx context.Context, fn func() error) error {
	if !c.allowRequest() {
		return ErrCircuitOpen
	}

	err := fn()

	c.mu.Lock()
	defer c.mu.Unlock()

	if err != nil {
		c.failures++
		c.lastFailureTime = time.Now()

		if c.state == StateHalfOpen || c.failures >= c.threshold {
			c.setState(StateOpen)
		}
		return err
	}

	c.successes++
	if c.state == StateHalfOpen {
		if c.successes >= c.halfOpenRequests {
			c.setState(StateClosed)
		}
	}

	return nil
}

func (c *Circuit) allowRequest() bool {
	c.mu.RLock()
	state := c.state
	lastFailure := c.lastFailureTime
	c.mu.RUnlock()

	switch state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(lastFailure) > c.timeout {
			c.mu.Lock()
			if c.state == StateOpen {
				c.setState(StateHalfOpen)
			}
			c.mu.Unlock()
		}
		return false
	case StateHalfOpen:
		return true
	}
	return false
}

func (c *Circuit) setState(state State) {
	if c.state == state {
		return
	}

	c.state = state

	switch state {
	case StateOpen:
		if c.onOpen != nil {
			c.onOpen()
		}
	case StateClosed:
		c.failures = 0
		c.successes = 0
		if c.onClose != nil {
			c.onClose()
		}
	case StateHalfOpen:
		c.successes = 0
		if c.onHalfOpen != nil {
			c.onHalfOpen()
		}
	}
}

func (c *Circuit) State() State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

func (c *Circuit) Failures() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.failures
}

func (c *Circuit) SetThreshold(threshold int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.threshold = threshold
}

func (c *Circuit) SetTimeout(timeout time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.timeout = timeout
}

func (c *Circuit) OnOpen(fn func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onOpen = fn
}

func (c *Circuit) OnClose(fn func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onClose = fn
}

func (c *Circuit) OnHalfOpen(fn func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onHalfOpen = fn
}

func (c *Circuit) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = StateClosed
	c.failures = 0
	c.successes = 0
}

type CircuitBreaker interface {
	Execute(ctx context.Context, fn func() error) error
	GetState() State
	Reset()
}

func NewCircuitBreaker(threshold int, timeout time.Duration) CircuitBreaker {
	c := NewCircuit(threshold, timeout)
	return &circuitWrapper{c: c}
}

type circuitWrapper struct {
	c *Circuit
}

func (w *circuitWrapper) Execute(ctx context.Context, fn func() error) error {
	return w.c.Execute(ctx, fn)
}

func (w *circuitWrapper) GetState() State {
	return w.c.State()
}

func (w *circuitWrapper) Reset() {
	w.c.Reset()
}
