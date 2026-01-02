package session

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

// AGENTS.MD JUSTIFICATION: Session store invariants (not-found, revocation, monotonic timestamps)
// are exercised here because feature tests do not cover in-memory persistence semantics.
type SessionStoreSuite struct {
	suite.Suite
	store *InMemorySessionStore
}

func (s *SessionStoreSuite) SetupTest() {
	s.store = New()
}

func TestSessionStoreSuite(t *testing.T) {
	suite.Run(t, new(SessionStoreSuite))
}

// TestSessionLookup tests session retrieval behavior.
func (s *SessionStoreSuite) TestSessionLookup() {
	s.Run("returns stored session when found", func() {
		store := New()
		session := &models.Session{
			ID:             id.SessionID(uuid.New()),
			UserID:         id.UserID(uuid.New()),
			RequestedScope: []string{"openid"},
			Status:         models.SessionStatusPendingConsent,
			CreatedAt:      time.Now(),
			ExpiresAt:      time.Now().Add(time.Hour),
		}

		err := store.Create(context.Background(), session)
		s.Require().NoError(err)

		foundByID, err := store.FindByID(context.Background(), session.ID)
		s.Require().NoError(err)
		s.Equal(session, foundByID)
	})

	s.Run("returns ErrNotFound when session does not exist", func() {
		_, err := s.store.FindByID(context.Background(), id.SessionID(uuid.New()))
		s.Require().ErrorIs(err, sentinel.ErrNotFound)
	})
}

// TestSessionStatusTransitions tests session status update behavior.
func (s *SessionStoreSuite) TestSessionStatusTransitions() {
	s.Run("updates session status successfully", func() {
		store := New()
		session := &models.Session{
			ID:             id.SessionID(uuid.New()),
			UserID:         id.UserID(uuid.New()),
			RequestedScope: []string{"openid"},
			Status:         models.SessionStatusPendingConsent,
			CreatedAt:      time.Now(),
			ExpiresAt:      time.Now().Add(time.Hour),
		}

		err := store.Create(context.Background(), session)
		s.Require().NoError(err)

		session.Status = models.SessionStatusActive
		err = store.UpdateSession(context.Background(), session)
		s.Require().NoError(err)

		found, err := store.FindByID(context.Background(), session.ID)
		s.Require().NoError(err)
		s.Equal(models.SessionStatusActive, found.Status)
	})

	s.Run("update on non-existent session returns ErrNotFound", func() {
		session := &models.Session{
			ID:             id.SessionID(uuid.New()),
			UserID:         id.UserID(uuid.New()),
			RequestedScope: []string{"openid"},
			Status:         models.SessionStatusActive,
			CreatedAt:      time.Now(),
			ExpiresAt:      time.Now().Add(time.Hour),
		}

		err := s.store.UpdateSession(context.Background(), session)
		s.Require().ErrorIs(err, sentinel.ErrNotFound)
	})
}

// TestSessionRevocation tests the revocation behavior and idempotency.
func (s *SessionStoreSuite) TestSessionRevocation() {
	s.Run("revokes active session and sets RevokedAt timestamp", func() {
		store := New()
		session := &models.Session{
			ID:        id.SessionID(uuid.New()),
			UserID:    id.UserID(uuid.New()),
			Status:    models.SessionStatusActive,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		s.Require().NoError(store.Create(context.Background(), session))

		err := store.RevokeSessionIfActive(context.Background(), session.ID, time.Now())
		s.Require().NoError(err)

		found, err := store.FindByID(context.Background(), session.ID)
		s.Require().NoError(err)
		s.Equal(models.SessionStatusRevoked, found.Status)
		s.Require().NotNil(found.RevokedAt)
	})

	s.Run("revoking already-revoked session returns ErrSessionRevoked", func() {
		store := New()
		session := &models.Session{
			ID:        id.SessionID(uuid.New()),
			UserID:    id.UserID(uuid.New()),
			Status:    models.SessionStatusActive,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		s.Require().NoError(store.Create(context.Background(), session))
		s.Require().NoError(store.RevokeSessionIfActive(context.Background(), session.ID, time.Now()))

		err := store.RevokeSessionIfActive(context.Background(), session.ID, time.Now())
		s.Require().ErrorIs(err, ErrSessionRevoked)
	})

	s.Run("revoking non-existent session returns ErrNotFound", func() {
		err := s.store.RevokeSessionIfActive(context.Background(), id.SessionID(uuid.New()), time.Now())
		s.Require().ErrorIs(err, sentinel.ErrNotFound)
	})
}

// TestSessionDeletionByUser tests bulk deletion for user cleanup (GDPR delete).
func (s *SessionStoreSuite) TestSessionDeletionByUser() {
	s.Run("deletes all sessions for user and leaves others intact", func() {
		store := New()
		userID := id.UserID(uuid.New())
		otherUserID := id.UserID(uuid.New())
		matching := &models.Session{ID: id.SessionID(uuid.New()), UserID: userID}
		other := &models.Session{ID: id.SessionID(uuid.New()), UserID: otherUserID}

		s.Require().NoError(store.Create(context.Background(), matching))
		s.Require().NoError(store.Create(context.Background(), other))

		err := store.DeleteSessionsByUser(context.Background(), userID)
		s.Require().NoError(err)

		_, err = store.FindByID(context.Background(), matching.ID)
		s.Require().ErrorIs(err, sentinel.ErrNotFound)

		fetchedOther, err := store.FindByID(context.Background(), other.ID)
		s.Require().NoError(err)
		s.Equal(other, fetchedOther)
	})

	s.Run("deleting sessions for user with no sessions returns ErrNotFound", func() {
		err := s.store.DeleteSessionsByUser(context.Background(), id.UserID(uuid.New()))
		s.Require().ErrorIs(err, sentinel.ErrNotFound)
	})
}
