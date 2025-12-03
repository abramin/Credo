package consent

import (
	"context"
	"sync"
	"time"
)

type InMemoryStore struct {
	mu       sync.RWMutex
	consents map[string][]ConsentRecord
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{consents: make(map[string][]ConsentRecord)}
}

func (s *InMemoryStore) Save(_ context.Context, consent ConsentRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.consents[consent.UserID] = append(s.consents[consent.UserID], consent)
	return nil
}

func (s *InMemoryStore) ListByUser(_ context.Context, userID string) ([]ConsentRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]ConsentRecord{}, s.consents[userID]...), nil
}

func (s *InMemoryStore) Revoke(_ context.Context, userID string, purpose ConsentPurpose, revokedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	records := s.consents[userID]
	for i := range records {
		if records[i].Purpose == purpose {
			records[i].RevokedAt = &revokedAt
		}
	}
	s.consents[userID] = records
	return nil
}
