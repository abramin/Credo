package ops

import (
	"sync"
	"time"
)

// CircuitBreaker prevents thundering herd on audit store outages.
// When the store is unhealthy, the circuit opens and events are dropped
// without attempting persistence.
type CircuitBreaker struct {
	mu sync.RWMutex

	threshold int           // failures to trigger open
	cooldown  time.Duration // how long to stay open

	failures  int       // consecutive failures
	openUntil time.Time // when to transition from open to half-open
	isOpen    bool
}

// NewCircuitBreaker creates a circuit breaker.
// threshold: number of consecutive failures to open the circuit
// cooldown: how long to stay open before trying again
func NewCircuitBreaker(threshold int, cooldown time.Duration) *CircuitBreaker {
	if threshold <= 0 {
		threshold = 5
	}
	if cooldown <= 0 {
		cooldown = time.Minute
	}
	return &CircuitBreaker{
		threshold: threshold,
		cooldown:  cooldown,
	}
}

// Allow returns true if the circuit is closed (healthy) or half-open (testing).
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.RLock()
	if !cb.isOpen {
		cb.mu.RUnlock()
		return true
	}

	// Check if cooldown expired
	expired := time.Now().After(cb.openUntil)
	cb.mu.RUnlock()

	if expired {
		// Transition to half-open - allow one request through
		cb.mu.Lock()
		defer cb.mu.Unlock()

		// Double-check after acquiring write lock
		if cb.isOpen && time.Now().After(cb.openUntil) {
			cb.isOpen = false
			cb.failures = 0
		}
		return !cb.isOpen
	}

	return false
}

// RecordSuccess records a successful operation, closing the circuit.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.isOpen = false
}

// RecordFailure records a failed operation, potentially opening the circuit.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	if cb.failures >= cb.threshold {
		cb.isOpen = true
		cb.openUntil = time.Now().Add(cb.cooldown)
	}
}

// IsOpen returns true if the circuit is currently open.
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.isOpen
}

// Reset manually closes the circuit.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.isOpen = false
}
