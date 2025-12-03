package storage

import (
	"context"
	"sync"
	"time"

	"id-gateway/internal/domain"
	"id-gateway/internal/policy"
)

// In-memory stores keep the initial implementation lightweight and testable. They
// intentionally favor clarity over performance.
type InMemoryUserStore struct {
	mu    sync.RWMutex
	users map[string]domain.User
}

func NewInMemoryUserStore() *InMemoryUserStore {
	return &InMemoryUserStore{users: make(map[string]domain.User)}
}

func (s *InMemoryUserStore) Save(_ context.Context, user domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[user.ID] = user
	return nil
}

func (s *InMemoryUserStore) FindByID(_ context.Context, id string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if user, ok := s.users[id]; ok {
		return user, nil
	}
	return domain.User{}, ErrNotFound
}

type InMemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]domain.Session
}

func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{sessions: make(map[string]domain.Session)}
}

func (s *InMemorySessionStore) Save(_ context.Context, session domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

func (s *InMemorySessionStore) FindByID(_ context.Context, id string) (domain.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if session, ok := s.sessions[id]; ok {
		return session, nil
	}
	return domain.Session{}, ErrNotFound
}

type InMemoryVCStore struct {
	mu          sync.RWMutex
	credentials map[string]domain.IssueVCResult
}

func NewInMemoryVCStore() *InMemoryVCStore {
	return &InMemoryVCStore{credentials: make(map[string]domain.IssueVCResult)}
}

func (s *InMemoryVCStore) Save(_ context.Context, credential domain.IssueVCResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentials[credential.ID] = credential
	return nil
}

func (s *InMemoryVCStore) FindByID(_ context.Context, id string) (domain.IssueVCResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if vc, ok := s.credentials[id]; ok {
		return vc, nil
	}
	return domain.IssueVCResult{}, ErrNotFound
}

type InMemoryRegistryCache struct {
	mu        sync.RWMutex
	citizens  map[string]cachedCitizen
	sanctions map[string]cachedSanction
}

func NewInMemoryRegistryCache() *InMemoryRegistryCache {
	return &InMemoryRegistryCache{
		citizens:  make(map[string]cachedCitizen),
		sanctions: make(map[string]cachedSanction),
	}
}

func (c *InMemoryRegistryCache) SaveCitizen(_ context.Context, record domain.CitizenRecord) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.citizens[record.NationalID] = cachedCitizen{record: record, storedAt: time.Now()}
	return nil
}

func (c *InMemoryRegistryCache) FindCitizen(_ context.Context, nationalID string) (domain.CitizenRecord, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if cached, ok := c.citizens[nationalID]; ok {
		if time.Since(cached.storedAt) < policy.RegistryCacheTTL {
			return cached.record, nil
		}
	}
	return domain.CitizenRecord{}, ErrNotFound
}

func (c *InMemoryRegistryCache) SaveSanction(_ context.Context, record domain.SanctionsRecord) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sanctions[record.NationalID] = cachedSanction{record: record, storedAt: time.Now()}
	return nil
}

func (c *InMemoryRegistryCache) FindSanction(_ context.Context, nationalID string) (domain.SanctionsRecord, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if cached, ok := c.sanctions[nationalID]; ok {
		if time.Since(cached.storedAt) < policy.RegistryCacheTTL {
			return cached.record, nil
		}
	}
	return domain.SanctionsRecord{}, ErrNotFound
}

type cachedCitizen struct {
	record   domain.CitizenRecord
	storedAt time.Time
}

type cachedSanction struct {
	record   domain.SanctionsRecord
	storedAt time.Time
}

type InMemoryConsentStore struct {
	mu       sync.RWMutex
	consents map[string][]domain.ConsentRecord
}

func NewInMemoryConsentStore() *InMemoryConsentStore {
	return &InMemoryConsentStore{consents: make(map[string][]domain.ConsentRecord)}
}

func (s *InMemoryConsentStore) Save(_ context.Context, consent domain.ConsentRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.consents[consent.UserID] = append(s.consents[consent.UserID], consent)
	return nil
}

func (s *InMemoryConsentStore) ListByUser(_ context.Context, userID string) ([]domain.ConsentRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]domain.ConsentRecord{}, s.consents[userID]...), nil
}

func (s *InMemoryConsentStore) Revoke(_ context.Context, userID string, purpose domain.ConsentPurpose, revokedAt time.Time) error {
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

type InMemoryAuditStore struct {
	mu     sync.RWMutex
	events map[string][]domain.AuditEvent
}

func NewInMemoryAuditStore() *InMemoryAuditStore {
	return &InMemoryAuditStore{events: make(map[string][]domain.AuditEvent)}
}

func (s *InMemoryAuditStore) Append(_ context.Context, event domain.AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events[event.UserID] = append(s.events[event.UserID], event)
	return nil
}

func (s *InMemoryAuditStore) ListByUser(_ context.Context, userID string) ([]domain.AuditEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]domain.AuditEvent{}, s.events[userID]...), nil
}
