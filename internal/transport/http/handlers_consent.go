package httptransport

import (
	"context"
	"log/slog"
	"time"

	consentHandlers "id-gateway/internal/consent/handlers"
	consentModel "id-gateway/internal/consent/models"
	"id-gateway/internal/platform/metrics"
	"id-gateway/internal/platform/middleware"
)

// ConsentService defines the interface for consent operations.
type ConsentService interface {
	Grant(ctx context.Context, userID string, purposes []consentModel.ConsentPurpose, ttl time.Duration) ([]*consentModel.ConsentRecord, error)
	Revoke(ctx context.Context, userID string, purpose consentModel.ConsentPurpose) error
	Require(ctx context.Context, userID string, purpose consentModel.ConsentPurpose, now time.Time) error
	List(ctx context.Context, userID string) ([]*consentModel.ConsentRecord, error)
}

// ConsentHandler is deprecated: use consentHandlers.Handler directly from consent/handlers package instead.
// This type is kept for backward compatibility during migration.
type ConsentHandler = consentHandlers.Handler

// NewConsentHandler creates a new ConsentHandler (deprecated wrapper).
// Deprecated: Use consentHandlers.New() from id-gateway/internal/consent/handlers directly.
func NewConsentHandler(
	consent ConsentService,
	logger *slog.Logger,
	metrics *metrics.Metrics,
	jwtValidator middleware.JWTValidator) *ConsentHandler {
	return consentHandlers.New(consent, logger, metrics, jwtValidator)
}
