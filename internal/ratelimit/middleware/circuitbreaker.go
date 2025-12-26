package middleware

import "sync"

// CircuitBreaker tracks consecutive limiter errors for fail-safe rate limiting (PRD-017 FR-7):
// - Track consecutive limiter errors.
// - Open circuit after N failures; during open, use in-memory fallback.
// - When open, set X-RateLimit-Status: degraded so callers know they're in fallback mode.
// - Close circuit after M consecutive successful primary checks.
type CircuitBreaker struct {
	mu               sync.Mutex
	state            circuitState
	failureCount     int
	successCount     int
	failureThreshold int
	successThreshold int
}

type circuitState int

const (
	circuitClosed circuitState = iota
	circuitOpen
)

func newCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		state:            circuitClosed,
		failureThreshold: 5,
		successThreshold: 3,
	}
}

func (c *CircuitBreaker) IsOpen() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state == circuitOpen
}

func (c *CircuitBreaker) RecordFailure() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failureCount++
	c.successCount = 0
	if c.state == circuitOpen {
		return true
	}
	if c.failureCount >= c.failureThreshold {
		c.state = circuitOpen
		return true
	}
	return false
}

// RecordSuccess records a successful check and returns whether the circuit is now closed.
// Returns true if the circuit is closed (use primary), false if still open (use fallback).
func (c *CircuitBreaker) RecordSuccess() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state == circuitOpen {
		c.successCount++
		if c.successCount >= c.successThreshold {
			c.state = circuitClosed
			c.failureCount = 0
			c.successCount = 0
			return true
		}
		return false
	}
	c.failureCount = 0
	return true
}

// ShouldUsePrimary returns true if the circuit is closed and primary limiter should be used.
// This is an alias for checking circuit state without recording success/failure.
func (c *CircuitBreaker) ShouldUsePrimary() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state == circuitClosed
}
