package vc

import (
	"context"
	"sync"
)

type InMemoryStore struct {
	mu          sync.RWMutex
	credentials map[string]IssueResult
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{credentials: make(map[string]IssueResult)}
}

func (s *InMemoryStore) Save(_ context.Context, credential IssueResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentials[credential.ID] = credential
	return nil
}

func (s *InMemoryStore) FindByID(_ context.Context, id string) (IssueResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if vc, ok := s.credentials[id]; ok {
		return vc, nil
	}
	return IssueResult{}, ErrNotFound
}
