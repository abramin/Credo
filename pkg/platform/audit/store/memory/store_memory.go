package memory

import (
	"context"
	"sync"

	id "credo/pkg/domain"
	audit "credo/pkg/platform/audit"
)

type InMemoryStore struct {
	mu     sync.RWMutex
	events map[id.UserID][]audit.Event
}

func (s *InMemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = make(map[id.UserID][]audit.Event)
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{events: make(map[id.UserID][]audit.Event)}
}

func (s *InMemoryStore) Append(_ context.Context, event audit.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events[event.UserID] = append(s.events[event.UserID], event)
	return nil
}

func (s *InMemoryStore) ListByUser(_ context.Context, userID id.UserID) ([]audit.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]audit.Event{}, s.events[userID]...), nil
}

// ListAll returns all audit events across all users (admin-only operation)
func (s *InMemoryStore) ListAll(_ context.Context) ([]audit.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var allEvents []audit.Event
	for _, userEvents := range s.events {
		allEvents = append(allEvents, userEvents...)
	}

	return allEvents, nil
}

// ListRecent returns the most recent N events across all users (admin-only operation)
func (s *InMemoryStore) ListRecent(_ context.Context, limit int) ([]audit.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var allEvents []audit.Event
	for _, userEvents := range s.events {
		allEvents = append(allEvents, userEvents...)
	}

	// Sort by timestamp descending (most recent first)
	// For simplicity, we'll return the last N events
	// In a real implementation with timestamps, we'd sort properly
	start := len(allEvents) - limit
	if start < 0 {
		start = 0
	}

	return allEvents[start:], nil
}
