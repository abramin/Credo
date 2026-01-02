package service

import (
	"context"

	"credo/internal/auth/models"
	id "credo/pkg/domain"
	dErrors "credo/pkg/domain-errors"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// TestSessionRevocation_ValidationAndAuthorization tests session revocation (PRD-016)
//
// AGENTS.MD JUSTIFICATION (per testing.md doctrine):
// These unit tests verify behaviors NOT covered by Gherkin:
// - validation errors: Tests input validation error codes (fast feedback)
// - session not found: Tests error propagation from store
// - different user forbidden: Tests multi-user authorization check (unique)
func (s *ServiceSuite) TestSessionRevocation_ValidationAndAuthorization() {
	ctx := context.Background()

	s.Run("invalid user returns unauthorized", func() {
		err := s.service.RevokeSession(ctx, id.UserID(uuid.Nil), id.SessionID(uuid.New()))
		s.Require().Error(err)
		s.True(dErrors.HasCode(err, dErrors.CodeUnauthorized))
	})

	s.Run("missing session id returns bad request", func() {
		err := s.service.RevokeSession(ctx, id.UserID(uuid.New()), id.SessionID(uuid.Nil))
		s.Require().Error(err)
		s.True(dErrors.HasCode(err, dErrors.CodeBadRequest))
	})

	s.Run("session not found returns not found", func() {
		userID := id.UserID(uuid.New())
		sessionID := id.SessionID(uuid.New())
		s.mockSessionStore.EXPECT().FindByID(gomock.Any(), sessionID).Return(nil, dErrors.New(dErrors.CodeNotFound, "nope"))

		err := s.service.RevokeSession(ctx, userID, sessionID)
		s.Require().Error(err)
		s.True(dErrors.HasCode(err, dErrors.CodeNotFound))
	})

	s.Run("session belonging to different user returns forbidden", func() {
		userID := id.UserID(uuid.New())
		otherUserID := id.UserID(uuid.New())
		sessionID := id.SessionID(uuid.New())
		session := &models.Session{
			ID:     sessionID,
			UserID: otherUserID,
			Status: models.SessionStatusActive,
		}
		s.mockSessionStore.EXPECT().FindByID(gomock.Any(), sessionID).Return(session, nil)
		s.mockAuditPublisher.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		err := s.service.RevokeSession(ctx, userID, sessionID)
		s.Require().Error(err)
		s.True(dErrors.HasCode(err, dErrors.CodeForbidden))
	})
}

func (s *ServiceSuite) TestSessionRevocation_LogoutAll() {
	ctx := context.Background()
	userID := id.UserID(uuid.New())
	currentSessionID := id.SessionID(uuid.New())

	sessions := []*models.Session{
		{
			ID:     currentSessionID,
			UserID: userID,
			Status: models.SessionStatusActive,
		},
		{
			ID:     id.SessionID(uuid.New()),
			UserID: userID,
			Status: models.SessionStatusActive,
		},
		{
			ID:     id.SessionID(uuid.New()),
			UserID: userID,
			Status: models.SessionStatusActive,
		},
	}

	s.Run("except_current=true revokes all except current session", func() {
		s.mockSessionStore.EXPECT().ListByUser(gomock.Any(), userID).Return(sessions, nil)
		// Should revoke 2 sessions (not the current one)
		s.mockSessionStore.EXPECT().RevokeSessionIfActive(gomock.Any(), sessions[1].ID, gomock.Any()).Return(nil)
		s.mockSessionStore.EXPECT().RevokeSessionIfActive(gomock.Any(), sessions[2].ID, gomock.Any()).Return(nil)
		s.mockRefreshStore.EXPECT().DeleteBySessionID(gomock.Any(), sessions[1].ID).Return(nil)
		s.mockRefreshStore.EXPECT().DeleteBySessionID(gomock.Any(), sessions[2].ID).Return(nil)
		s.mockAuditPublisher.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		result, err := s.service.LogoutAll(ctx, userID, currentSessionID, true)

		s.Require().NoError(err)
		s.Equal(2, result.RevokedCount)
	})

	s.Run("except_current=false revokes all including current session", func() {
		s.mockSessionStore.EXPECT().ListByUser(gomock.Any(), userID).Return(sessions, nil)
		// Should revoke all 3 sessions
		s.mockSessionStore.EXPECT().RevokeSessionIfActive(gomock.Any(), sessions[0].ID, gomock.Any()).Return(nil)
		s.mockSessionStore.EXPECT().RevokeSessionIfActive(gomock.Any(), sessions[1].ID, gomock.Any()).Return(nil)
		s.mockSessionStore.EXPECT().RevokeSessionIfActive(gomock.Any(), sessions[2].ID, gomock.Any()).Return(nil)
		s.mockRefreshStore.EXPECT().DeleteBySessionID(gomock.Any(), sessions[0].ID).Return(nil)
		s.mockRefreshStore.EXPECT().DeleteBySessionID(gomock.Any(), sessions[1].ID).Return(nil)
		s.mockRefreshStore.EXPECT().DeleteBySessionID(gomock.Any(), sessions[2].ID).Return(nil)
		s.mockAuditPublisher.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		result, err := s.service.LogoutAll(ctx, userID, currentSessionID, false)

		s.Require().NoError(err)
		s.Equal(3, result.RevokedCount)
	})

	s.Run("invalid user ID returns unauthorized", func() {
		_, err := s.service.LogoutAll(ctx, id.UserID(uuid.Nil), currentSessionID, true)

		s.Require().Error(err)
		s.True(dErrors.HasCode(err, dErrors.CodeUnauthorized))
	})

	s.Run("session store error returns internal error", func() {
		s.mockSessionStore.EXPECT().ListByUser(gomock.Any(), userID).Return(nil, assert.AnError)

		_, err := s.service.LogoutAll(ctx, userID, currentSessionID, true)

		s.Require().Error(err)
		s.True(dErrors.HasCode(err, dErrors.CodeInternal))
	})

	s.Run("partial revocation failure continues and returns partial count", func() {
		// Current session (excluded), session 1 will fail to revoke, session 2 will succeed
		currentSession := &models.Session{
			ID:     currentSessionID,
			UserID: userID,
			Status: models.SessionStatusActive,
		}
		failingSession := &models.Session{
			ID:     id.SessionID(uuid.New()),
			UserID: userID,
			Status: models.SessionStatusActive,
		}
		succeedingSession := &models.Session{
			ID:     id.SessionID(uuid.New()),
			UserID: userID,
			Status: models.SessionStatusActive,
		}
		allSessions := []*models.Session{currentSession, failingSession, succeedingSession}

		s.mockSessionStore.EXPECT().ListByUser(gomock.Any(), userID).Return(allSessions, nil)
		// First session fails to revoke
		s.mockSessionStore.EXPECT().RevokeSessionIfActive(gomock.Any(), failingSession.ID, gomock.Any()).
			Return(assert.AnError)
		// Second session succeeds
		s.mockSessionStore.EXPECT().RevokeSessionIfActive(gomock.Any(), succeedingSession.ID, gomock.Any()).
			Return(nil)
		s.mockRefreshStore.EXPECT().DeleteBySessionID(gomock.Any(), succeedingSession.ID).Return(nil)
		s.mockAuditPublisher.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		result, err := s.service.LogoutAll(ctx, userID, currentSessionID, true)

		// Should succeed with partial count (only 1 revoked, not 2)
		s.Require().NoError(err)
		s.Equal(1, result.RevokedCount)
	})
}
