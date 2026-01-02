package authorizationcode

import (
	"context"
	"testing"
	"time"

	"credo/internal/auth/models"
	id "credo/pkg/domain"
	"credo/pkg/platform/sentinel"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// AGENTS.MD JUSTIFICATION: Authorization code persistence and consume semantics
// (expired, already used, redirect mismatch) are enforced here beyond feature tests.
type AuthCodeStoreSuite struct {
	suite.Suite
	store *InMemoryAuthorizationCodeStore
}

func (s *AuthCodeStoreSuite) SetupTest() {
	s.store = New()
}

func TestAuthCodeStoreSuite(t *testing.T) {
	suite.Run(t, new(AuthCodeStoreSuite))
}

// TestCodeLookup tests code retrieval behavior.
func (s *AuthCodeStoreSuite) TestCodeLookup() {
	s.Run("returns stored code when found", func() {
		store := New()
		authCode := &models.AuthorizationCodeRecord{
			SessionID:   id.SessionID(uuid.New()),
			ExpiresAt:   time.Now().Add(time.Minute * 10),
			Code:        "authz_123456",
			CreatedAt:   time.Now(),
			RedirectURI: "https://example.com/callback",
		}

		err := store.Create(context.Background(), authCode)
		s.Require().NoError(err)

		foundByCode, err := store.FindByCode(context.Background(), "authz_123456")
		s.Require().NoError(err)
		s.Equal(authCode, foundByCode)
	})

	s.Run("returns ErrNotFound when code does not exist", func() {
		_, err := s.store.FindByCode(context.Background(), "non_existent_code")
		s.Require().ErrorIs(err, sentinel.ErrNotFound)
	})
}

// TestCodeConsumption tests the atomic consume-once semantics of authorization codes.
func (s *AuthCodeStoreSuite) TestCodeConsumption() {
	ctx := context.Background()
	now := time.Now()

	s.Run("fresh code can be consumed once", func() {
		store := New()
		record := &models.AuthorizationCodeRecord{
			Code:        "authz_execute",
			SessionID:   id.SessionID(uuid.New()),
			RedirectURI: "https://app/callback",
			ExpiresAt:   now.Add(time.Minute),
			Used:        false,
			CreatedAt:   now.Add(-time.Minute),
		}
		s.Require().NoError(store.Create(ctx, record))

		consumed, err := store.Execute(ctx, record.Code,
			func(r *models.AuthorizationCodeRecord) error {
				return r.ValidateForConsume(record.RedirectURI, now)
			},
			func(r *models.AuthorizationCodeRecord) {
				r.MarkUsed()
			},
		)
		s.Require().NoError(err)
		s.True(consumed.Used)
	})

	s.Run("consumed code returns already-used error with record for replay detection", func() {
		store := New()
		record := &models.AuthorizationCodeRecord{
			Code:        "authz_reuse",
			SessionID:   id.SessionID(uuid.New()),
			RedirectURI: "https://app/callback",
			ExpiresAt:   now.Add(time.Minute),
			Used:        false,
			CreatedAt:   now.Add(-time.Minute),
		}
		s.Require().NoError(store.Create(ctx, record))

		// First consume succeeds
		_, err := store.Execute(ctx, record.Code,
			func(r *models.AuthorizationCodeRecord) error {
				return r.ValidateForConsume(record.RedirectURI, now)
			},
			func(r *models.AuthorizationCodeRecord) { r.MarkUsed() },
		)
		s.Require().NoError(err)

		// Second consume fails but returns record
		consumed, err := store.Execute(ctx, record.Code,
			func(r *models.AuthorizationCodeRecord) error {
				return r.ValidateForConsume(record.RedirectURI, now)
			},
			func(r *models.AuthorizationCodeRecord) { r.MarkUsed() },
		)
		s.Require().Error(err)
		s.Contains(err.Error(), "already used")
		s.NotNil(consumed) // Record returned for replay detection
	})

	s.Run("expired code returns expired error", func() {
		store := New()
		expired := &models.AuthorizationCodeRecord{
			Code:        "authz_expired",
			SessionID:   id.SessionID(uuid.New()),
			RedirectURI: "https://app/callback",
			ExpiresAt:   now.Add(-time.Minute),
			Used:        false,
			CreatedAt:   now.Add(-2 * time.Minute),
		}
		s.Require().NoError(store.Create(ctx, expired))

		_, err := store.Execute(ctx, expired.Code,
			func(r *models.AuthorizationCodeRecord) error {
				return r.ValidateForConsume(expired.RedirectURI, now)
			},
			func(r *models.AuthorizationCodeRecord) { r.MarkUsed() },
		)
		s.Require().Error(err)
		s.Contains(err.Error(), "expired")
	})

	s.Run("redirect_uri mismatch returns error", func() {
		store := New()
		record := &models.AuthorizationCodeRecord{
			Code:        "authz_redirect",
			SessionID:   id.SessionID(uuid.New()),
			RedirectURI: "https://expected",
			ExpiresAt:   now.Add(time.Minute),
			Used:        false,
			CreatedAt:   now.Add(-time.Minute),
		}
		s.Require().NoError(store.Create(ctx, record))

		_, err := store.Execute(ctx, record.Code,
			func(r *models.AuthorizationCodeRecord) error {
				return r.ValidateForConsume("https://wrong", now)
			},
			func(r *models.AuthorizationCodeRecord) { r.MarkUsed() },
		)
		s.Require().Error(err)
		s.Contains(err.Error(), "redirect_uri mismatch")
	})

	s.Run("non-existent code returns ErrNotFound", func() {
		_, err := s.store.Execute(ctx, "missing",
			func(r *models.AuthorizationCodeRecord) error { return nil },
			func(r *models.AuthorizationCodeRecord) {},
		)
		s.Require().ErrorIs(err, sentinel.ErrNotFound)
	})
}
