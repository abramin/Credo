package security

import (
	"sync"

	audit "credo/pkg/platform/audit"
)

// RingBuffer is a bounded, thread-safe buffer for security events.
// When full, the oldest events are dropped to make room for new ones.
type RingBuffer struct {
	mu       sync.Mutex
	events   []audit.SecurityEvent
	head     int // next write position
	tail     int // next read position
	count    int
	capacity int

	// Stats
	dropped int64
}

// NewRingBuffer creates a ring buffer with the given capacity.
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = 10000 // default
	}
	return &RingBuffer{
		events:   make([]audit.SecurityEvent, capacity),
		capacity: capacity,
	}
}

// TryEnqueue attempts to add an event to the buffer.
// Returns false if the buffer is full (caller should call DropOldest then retry).
func (b *RingBuffer) TryEnqueue(event audit.SecurityEvent) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.count >= b.capacity {
		return false
	}

	b.events[b.head] = event
	b.head = (b.head + 1) % b.capacity
	b.count++
	return true
}

// Enqueue adds an event, dropping the oldest if necessary.
func (b *RingBuffer) Enqueue(event audit.SecurityEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.count >= b.capacity {
		// Drop oldest
		b.tail = (b.tail + 1) % b.capacity
		b.count--
		b.dropped++
	}

	b.events[b.head] = event
	b.head = (b.head + 1) % b.capacity
	b.count++
}

// DropOldest removes the oldest event from the buffer.
// Returns false if buffer is empty.
func (b *RingBuffer) DropOldest() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.count == 0 {
		return false
	}

	b.tail = (b.tail + 1) % b.capacity
	b.count--
	b.dropped++
	return true
}

// DequeueBatch removes up to n events from the buffer.
func (b *RingBuffer) DequeueBatch(n int) []audit.SecurityEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.count == 0 {
		return nil
	}

	if n > b.count {
		n = b.count
	}

	result := make([]audit.SecurityEvent, n)
	for i := 0; i < n; i++ {
		result[i] = b.events[b.tail]
		b.tail = (b.tail + 1) % b.capacity
	}
	b.count -= n

	return result
}

// Len returns the current number of events in the buffer.
func (b *RingBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.count
}

// Dropped returns the total number of dropped events.
func (b *RingBuffer) Dropped() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.dropped
}
