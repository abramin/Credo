package httptransport

import (
	"context"
	"log/slog"

	authHandlers "id-gateway/internal/auth/handlers"
	authModel "id-gateway/internal/auth/models"
	"id-gateway/internal/platform/metrics"

	"github.com/google/uuid"
)

// AuthService defines the interface for authentication operations.
type AuthService interface {
	Authorize(ctx context.Context, req *authModel.AuthorizationRequest) (*authModel.AuthorizationResult, error)
	Token(ctx context.Context, req *authModel.TokenRequest) (*authModel.TokenResult, error)
	UserInfo(ctx context.Context, sessionID uuid.UUID) (*authModel.UserInfoResult, error)
}

// AuthHandler is deprecated: use authHandlers.Handler directly from auth/handlers package instead.
// This type is kept for backward compatibility during migration.
type AuthHandler = authHandlers.Handler

// NewAuthHandler creates a new AuthHandler (deprecated wrapper).
// Deprecated: Use authHandlers.New() from id-gateway/internal/auth/handlers directly.
func NewAuthHandler(auth AuthService, logger *slog.Logger, regulatedMode bool, metrics *metrics.Metrics) *AuthHandler {
	return authHandlers.New(auth, logger, regulatedMode, metrics)
}
