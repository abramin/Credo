package store

import (
	"context"
	"id-gateway/internal/auth/models"
	"sync"
)

// In-memory stores keep the initial implementation lightweight and testable.
// They intentionally favor clarity over performance.
type InMemoryUserStore struct {
	mu    sync.RWMutex
	users map[string]models.User
}

func NewInMemoryUserStore() *InMemoryUserStore {
	return &InMemoryUserStore{users: make(map[string]models.User)}
}

func (s *InMemoryUserStore) Save(_ context.Context, user models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[user.ID] = user
	return nil
}

func (s *InMemoryUserStore) FindByID(_ context.Context, id string) (models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if user, ok := s.users[id]; ok {
		return user, nil
	}
	return models.User{}, ErrNotFound
}

type InMemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]models.Session
}

func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{sessions: make(map[string]models.Session)}
}

func (s *InMemorySessionStore) Save(_ context.Context, session models.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

func (s *InMemorySessionStore) FindByID(_ context.Context, id string) (models.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if session, ok := s.sessions[id]; ok {
		return session, nil
	}
	return models.Session{}, ErrNotFound
}
