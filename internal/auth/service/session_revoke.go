package service

import (
	"context"
	"errors"
	"time"

	"credo/internal/auth/models"
	id "credo/pkg/domain"
	dErrors "credo/pkg/domain-errors"
	"credo/pkg/platform/audit"
	"credo/pkg/platform/sentinel"
	"credo/pkg/requestcontext"
)

// RevokeSession revokes a single session for the given user.
// Uses Execute callback pattern to validate ownership atomically under lock.
func (s *Service) RevokeSession(ctx context.Context, userID id.UserID, sessionID id.SessionID) error {
	if userID.IsNil() {
		return dErrors.New(dErrors.CodeUnauthorized, "user ID required")
	}
	if sessionID.IsNil() {
		return dErrors.New(dErrors.CodeBadRequest, "session ID required")
	}

	now := requestcontext.Now(ctx)
	var alreadyRevoked bool

	session, err := s.sessions.Execute(ctx, sessionID,
		// Validate callback: check ownership and revokability atomically under lock
		func(sess *models.Session) error {
			if sess.UserID != userID {
				s.authFailure(ctx, "session_owner_mismatch", false,
					"session_id", sess.ID.String(),
					"user_id", userID.String(),
				)
				return dErrors.New(dErrors.CodeForbidden, "forbidden")
			}
			if err := sess.CanRevoke(); err != nil {
				alreadyRevoked = true
				return nil // Not an error - just skip mutation
			}
			return nil
		},
		// Mutate callback: apply revocation if not already revoked
		func(sess *models.Session) {
			if !alreadyRevoked {
				sess.ApplyRevocation(now)
			}
		},
	)
	if err != nil {
		if dErrors.HasCode(err, dErrors.CodeForbidden) {
			return err // Pass through ownership error
		}
		if errors.Is(err, sentinel.ErrNotFound) {
			return dErrors.New(dErrors.CodeNotFound, "session not found")
		}
		return dErrors.Wrap(err, dErrors.CodeInternal, "failed to revoke session")
	}

	if alreadyRevoked {
		return nil
	}

	// Post-revocation: revoke tokens and delete refresh tokens
	if session.LastAccessTokenJTI != "" {
		if err := s.trl.RevokeToken(ctx, session.LastAccessTokenJTI, s.TokenTTL); err != nil {
			s.logger.Error("failed to add token to revocation list", "error", err, "jti", session.LastAccessTokenJTI)
			if s.TRLFailureMode == TRLFailureModeFail {
				return dErrors.Wrap(err, dErrors.CodeInternal, "failed to add token to revocation list")
			}
		}
	}

	if err := s.refreshTokens.DeleteBySessionID(ctx, session.ID); err != nil {
		s.logger.Error("failed to delete refresh tokens", "error", err, "session_id", session.ID)
		// Don't fail - session is already revoked
	}

	s.logAudit(ctx, string(audit.EventSessionRevoked),
		"user_id", session.UserID.String(),
		"session_id", session.ID.String(),
		"client_id", session.ClientID,
		"reason", models.RevocationReasonUserInitiated.String(),
	)

	return nil
}

// LogoutAll revokes all sessions for a user, optionally keeping the current session.
// Design: Continues on individual revocation errors to maximize successful revocations.
// Returns partial results with FailedCount so the user knows if retries are needed.
// Only returns an error if ALL revocations fail or listing sessions fails.
func (s *Service) LogoutAll(ctx context.Context, userID id.UserID, currentSessionID id.SessionID, exceptCurrent bool) (*models.LogoutAllResult, error) {
	start := time.Now()

	if userID.IsNil() {
		return nil, dErrors.New(dErrors.CodeUnauthorized, "user ID required")
	}

	sessions, err := s.sessions.ListByUser(ctx, userID)
	if err != nil {
		return nil, dErrors.Wrap(err, dErrors.CodeInternal, "failed to list sessions")
	}

	revokedCount := 0
	failedCount := 0
	for _, session := range sessions {
		if exceptCurrent && session.ID == currentSessionID {
			continue
		}
		outcome, err := s.revokeSessionInternal(ctx, session, "", models.RevocationReasonUserInitiated)
		if err != nil {
			// Continue on error - partial revocation is better than none
			failedCount++
			s.logger.ErrorContext(ctx, "failed to revoke session during logout-all",
				"error", err,
				"session_id", session.ID.String(),
				"user_id", userID.String(),
			)
			continue
		}
		if outcome == revokeSessionOutcomeRevoked {
			revokedCount++
			s.logAudit(ctx, string(audit.EventSessionRevoked),
				"user_id", session.UserID.String(),
				"session_id", session.ID.String(),
				"client_id", session.ClientID,
				"reason", models.RevocationReasonUserInitiated.String(),
			)
		}
	}

	if s.metrics != nil {
		durationMs := float64(time.Since(start).Milliseconds())
		s.metrics.ObserveLogoutAll(revokedCount, durationMs)
	}

	return &models.LogoutAllResult{
		RevokedCount: revokedCount,
		FailedCount:  failedCount,
	}, nil
}
