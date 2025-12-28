package refreshtoken

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"credo/internal/auth/models"
	id "credo/pkg/domain"
	"credo/pkg/platform/sentinel"
)

// translateRefreshTokenError converts domain errors from ValidateForConsume to sentinel errors.
func translateRefreshTokenError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "expired"):
		return fmt.Errorf("%s: %w", msg, sentinel.ErrExpired)
	case strings.Contains(msg, "already used"):
		return fmt.Errorf("%s: %w", msg, sentinel.ErrAlreadyUsed)
	default:
		return fmt.Errorf("%s: %w", msg, sentinel.ErrInvalidState)
	}
}

// Error Contract:
// All store methods follow this error pattern:
// - Return ErrNotFound when the requested entity does not exist
// - Return nil for successful operations
// - Return wrapped errors with context for infrastructure failures (future: DB errors, network issues, etc.)
// InMemoryRefreshTokenStore stores refresh tokens in memory for tests/dev.
type InMemoryRefreshTokenStore struct {
	mu     sync.RWMutex
	tokens map[string]*models.RefreshTokenRecord
}

// New constructs an empty in-memory refresh token store.
func New() *InMemoryRefreshTokenStore {
	return &InMemoryRefreshTokenStore{tokens: make(map[string]*models.RefreshTokenRecord)}
}

func (s *InMemoryRefreshTokenStore) Create(_ context.Context, token *models.RefreshTokenRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token.Token] = token
	return nil
}

func (s *InMemoryRefreshTokenStore) Find(_ context.Context, token string) (*models.RefreshTokenRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if record, ok := s.tokens[token]; ok {
		return record, nil
	}
	return nil, fmt.Errorf("refresh token not found: %w", sentinel.ErrNotFound)
}

func (s *InMemoryRefreshTokenStore) FindBySessionID(_ context.Context, sessionID id.SessionID, now time.Time) (*models.RefreshTokenRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var best *models.RefreshTokenRecord
	for _, token := range s.tokens {
		if token.SessionID != sessionID {
			continue
		}
		if token.Used {
			continue
		}
		if !token.ExpiresAt.IsZero() && token.ExpiresAt.Before(now) {
			continue
		}
		if best == nil || token.CreatedAt.After(best.CreatedAt) {
			best = token
		}
	}
	if best == nil {
		return nil, fmt.Errorf("refresh token not found: %w", sentinel.ErrNotFound)
	}
	return best, nil
}

func (s *InMemoryRefreshTokenStore) DeleteBySessionID(_ context.Context, sessionID id.SessionID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := false
	for key, token := range s.tokens {
		if token.SessionID == sessionID {
			delete(s.tokens, key)
			found = true
		}
	}
	if !found {
		return sentinel.ErrNotFound
	}
	return nil
}

// ConsumeRefreshToken marks the refresh token as used if valid.
// It validates using domain logic, then marks the token as used via domain method.
// Returns the token record and an error if any validation fails.
// Errors are returned as sentinel errors per store boundary contract.
// IMPORTANT: Returns the record even on ErrAlreadyUsed to enable replay detection.
func (s *InMemoryRefreshTokenStore) ConsumeRefreshToken(_ context.Context, token string, now time.Time) (*models.RefreshTokenRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.tokens[token]
	if !ok {
		return nil, fmt.Errorf("refresh token not found: %w", sentinel.ErrNotFound)
	}

	// Validate using domain method, translate to sentinel errors per store contract
	if err := record.ValidateForConsume(now); err != nil {
		return record, translateRefreshTokenError(err)
	}

	// Mark as used via domain method (also records LastRefreshedAt)
	record.MarkUsed(now)
	return record, nil
}

// DeleteExpiredTokens removes all refresh tokens that have expired as of the given time.
// The time parameter is injected for testability (no hidden time.Now() calls).
func (s *InMemoryRefreshTokenStore) DeleteExpiredTokens(_ context.Context, now time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	deletedCount := 0
	for key, token := range s.tokens {
		if token.ExpiresAt.Before(now) {
			delete(s.tokens, key)
			deletedCount++
		}
	}
	return deletedCount, nil
}

func (s *InMemoryRefreshTokenStore) DeleteUsedTokens(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	deletedCount := 0
	for key, token := range s.tokens {
		if token.Used {
			delete(s.tokens, key)
			deletedCount++
		}
	}
	return deletedCount, nil
}
