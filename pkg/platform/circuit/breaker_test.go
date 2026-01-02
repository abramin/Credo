package circuit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBreaker_InitialState(t *testing.T) {
	b := New("test")
	assert.False(t, b.IsOpen())
	assert.Equal(t, StateClosed, b.State())
	assert.Equal(t, "test", b.Name())
}

func TestBreaker_OpensAfterThreshold(t *testing.T) {
	b := New("test", WithFailureThreshold(3))

	// First two failures don't open
	useFallback, change := b.RecordFailure()
	assert.False(t, useFallback)
	assert.False(t, change.Opened)

	useFallback, change = b.RecordFailure()
	assert.False(t, useFallback)
	assert.False(t, change.Opened)

	// Third failure opens the circuit
	useFallback, change = b.RecordFailure()
	assert.True(t, useFallback)
	assert.True(t, change.Opened)
	assert.True(t, b.IsOpen())
}

func TestBreaker_ClosesAfterSuccessThreshold(t *testing.T) {
	b := New("test", WithFailureThreshold(1), WithSuccessThreshold(2))

	// Open the circuit
	b.RecordFailure()
	assert.True(t, b.IsOpen())

	// First success doesn't close
	usePrimary, change := b.RecordSuccess()
	assert.False(t, usePrimary)
	assert.False(t, change.Closed)
	assert.True(t, b.IsOpen())

	// Second success closes
	usePrimary, change = b.RecordSuccess()
	assert.True(t, usePrimary)
	assert.True(t, change.Closed)
	assert.False(t, b.IsOpen())
}

func TestBreaker_SuccessResetsFailureCount(t *testing.T) {
	b := New("test", WithFailureThreshold(3))

	// Two failures
	b.RecordFailure()
	b.RecordFailure()
	assert.False(t, b.IsOpen())

	// Success resets count
	b.RecordSuccess()

	// Two more failures don't open (count was reset)
	b.RecordFailure()
	b.RecordFailure()
	assert.False(t, b.IsOpen())

	// Third failure opens
	b.RecordFailure()
	assert.True(t, b.IsOpen())
}

func TestBreaker_FailureResetsSuccessCount(t *testing.T) {
	b := New("test", WithFailureThreshold(1), WithSuccessThreshold(3))

	// Open the circuit
	b.RecordFailure()
	assert.True(t, b.IsOpen())

	// Two successes
	b.RecordSuccess()
	b.RecordSuccess()

	// Failure resets success count (stays open)
	b.RecordFailure()
	assert.True(t, b.IsOpen())

	// Need 3 successes again to close
	b.RecordSuccess()
	b.RecordSuccess()
	assert.True(t, b.IsOpen())
	b.RecordSuccess()
	assert.False(t, b.IsOpen())
}

func TestBreaker_Reset(t *testing.T) {
	b := New("test", WithFailureThreshold(1))

	// Open the circuit
	b.RecordFailure()
	assert.True(t, b.IsOpen())

	// Reset closes it
	b.Reset()
	assert.False(t, b.IsOpen())
	assert.Equal(t, StateClosed, b.State())
}

func TestBreaker_OpenCircuitReturnsFallback(t *testing.T) {
	b := New("test", WithFailureThreshold(1))

	// Open the circuit
	b.RecordFailure()

	// Additional failures return fallback without state change
	useFallback, change := b.RecordFailure()
	assert.True(t, useFallback)
	assert.False(t, change.Opened) // Already open, no state change
}
