package authorizationcode

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"credo/internal/auth/models"
	"credo/pkg/platform/sentinel"
)

// translateAuthCodeError converts domain errors from ValidateForConsume to sentinel errors.
func translateAuthCodeError(err error) error {
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
//

// InMemoryAuthorizationCodeStore stores authorization codes in memory for tests/dev.
type InMemoryAuthorizationCodeStore struct {
	mu        sync.RWMutex
	authCodes map[string]*models.AuthorizationCodeRecord
}

// New constructs an empty in-memory auth code store.
func New() *InMemoryAuthorizationCodeStore {
	return &InMemoryAuthorizationCodeStore{
		authCodes: make(map[string]*models.AuthorizationCodeRecord),
	}
}

func (s *InMemoryAuthorizationCodeStore) Create(_ context.Context, authCode *models.AuthorizationCodeRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authCodes[authCode.Code] = authCode
	return nil
}

func (s *InMemoryAuthorizationCodeStore) FindByCode(_ context.Context, code string) (*models.AuthorizationCodeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if authCode, ok := s.authCodes[code]; ok {
		return authCode, nil
	}
	return nil, fmt.Errorf("authorization code not found: %w", sentinel.ErrNotFound)
}

func (s *InMemoryAuthorizationCodeStore) MarkUsed(_ context.Context, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.authCodes[code]
	if !ok {
		return fmt.Errorf("authorization code not found: %w", sentinel.ErrNotFound)
	}
	record.MarkUsed()
	return nil
}

// ConsumeAuthCode marks the authorization code as used if valid.
// It validates using domain logic, then marks the code as used via domain method.
// Returns the code record and an error if any validation fails.
// Errors are returned as sentinel errors per store boundary contract.
func (s *InMemoryAuthorizationCodeStore) ConsumeAuthCode(_ context.Context, code string, redirectURI string, now time.Time) (*models.AuthorizationCodeRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.authCodes[code]
	if !ok {
		return nil, fmt.Errorf("authorization code not found: %w", sentinel.ErrNotFound)
	}

	// Validate using domain method, translate to sentinel errors per store contract
	if err := record.ValidateForConsume(redirectURI, now); err != nil {
		return record, translateAuthCodeError(err)
	}

	// Mark as used via domain method
	record.MarkUsed()
	return record, nil
}

// DeleteExpiredCodes removes all authorization codes that have expired as of the given time.
// The time parameter is injected for testability (no hidden time.Now() calls).
func (s *InMemoryAuthorizationCodeStore) DeleteExpiredCodes(_ context.Context, now time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	deletedCount := 0
	for code, record := range s.authCodes {
		if record.ExpiresAt.Before(now) {
			delete(s.authCodes, code)
			deletedCount++
		}
	}
	return deletedCount, nil
}
