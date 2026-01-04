package service

import (
	"context"
	"errors"

	"credo/internal/auth/models"
	id "credo/pkg/domain"
	dErrors "credo/pkg/domain-errors"
	"credo/pkg/platform/audit"
	"credo/pkg/platform/sentinel"
)

// DeleteUser deletes a user and revokes their sessions as an admin operation.
// Uses RunInTx to ensure atomic deletion of sessions and user.
func (s *Service) DeleteUser(ctx context.Context, userID id.UserID) error {
	if userID.IsNil() {
		return dErrors.New(dErrors.CodeBadRequest, "user ID required")
	}

	// Capture user before deletion to enrich audit events (outside transaction)
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sentinel.ErrNotFound) {
			return dErrors.New(dErrors.CodeNotFound, "user not found")
		}
		return dErrors.Wrap(err, dErrors.CodeInternal, "failed to lookup user")
	}

	auditAttrs := []any{
		"user_id", userID.String(),
		"email", user.Email,
		"reason", models.RevocationReasonUserDeleted.String(),
	}

	// Atomic deletion: sessions and user within same transaction
	if err := s.tx.RunInTx(ctx, func(stores txAuthStores) error {
		// Delete sessions (ignore not found - user may have no sessions)
		if err := stores.Sessions.DeleteSessionsByUser(ctx, userID); err != nil {
			if !errors.Is(err, sentinel.ErrNotFound) {
				return dErrors.Wrap(err, dErrors.CodeInternal, "failed to delete user sessions")
			}
		}

		// Delete user
		if err := stores.Users.Delete(ctx, userID); err != nil {
			if errors.Is(err, sentinel.ErrNotFound) {
				return dErrors.New(dErrors.CodeNotFound, "user not found")
			}
			return dErrors.Wrap(err, dErrors.CodeInternal, "failed to delete user")
		}

		return nil
	}); err != nil {
		return err
	}

	// Audit events after successful transaction commit
	s.logAudit(ctx, string(audit.EventSessionsRevoked), auditAttrs...)
	s.logAudit(ctx, string(audit.EventUserDeleted), auditAttrs...)

	return nil
}
