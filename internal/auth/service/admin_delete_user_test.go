package service

import (
	"context"
	"errors"

	"credo/internal/auth/models"
	id "credo/pkg/domain"
	dErrors "credo/pkg/domain-errors"
	"credo/pkg/platform/sentinel"

	"github.com/google/uuid"
)

// TestAdminUserDeletion_ErrorPropagation tests the admin user deletion error handling
// NOTE: Happy path and no-sessions-found cases are covered by Cucumber E2E tests
// in e2e/features/admin_gdpr.feature. These unit tests focus on error propagation.
func (s *ServiceSuite) TestAdminUserDeletion_ErrorPropagation() {
	ctx := context.Background()
	userID := id.UserID(uuid.New())
	existingUser := &models.User{ID: userID, Email: "user@example.com"}

	s.Run("user lookup fails", func() {
		s.mockUserStore.EXPECT().FindByID(ctx, userID).Return(nil, errors.New("db down"))

		err := s.service.DeleteUser(ctx, userID)
		s.Require().Error(err)
		s.True(dErrors.HasCode(err, dErrors.CodeInternal))
	})

	s.Run("user not found", func() {
		s.mockUserStore.EXPECT().FindByID(ctx, userID).Return(nil, sentinel.ErrNotFound)

		err := s.service.DeleteUser(ctx, userID)
		s.Require().Error(err)
		s.True(dErrors.HasCode(err, dErrors.CodeNotFound))
	})

	s.Run("session delete fails", func() {
		s.mockUserStore.EXPECT().FindByID(ctx, userID).Return(existingUser, nil)
		s.mockSessionStore.EXPECT().DeleteSessionsByUser(ctx, userID).Return(errors.New("redis down"))

		err := s.service.DeleteUser(ctx, userID)
		s.Require().Error(err)
		s.True(dErrors.HasCode(err, dErrors.CodeInternal))
	})

	s.Run("user delete fails", func() {
		s.mockUserStore.EXPECT().FindByID(ctx, userID).Return(existingUser, nil)
		s.mockSessionStore.EXPECT().DeleteSessionsByUser(ctx, userID).Return(nil)
		s.mockUserStore.EXPECT().Delete(ctx, userID).Return(errors.New("write fail"))

		err := s.service.DeleteUser(ctx, userID)
		s.Require().Error(err)
		s.True(dErrors.HasCode(err, dErrors.CodeInternal))
	})
}

// TestAdminUserDeletion_AuditEnrichment was removed during tri-publisher migration.
// Audit event content verification is no longer possible with fire-and-forget security publisher.
// Audit correctness is now verified through integration tests with real audit stores.

