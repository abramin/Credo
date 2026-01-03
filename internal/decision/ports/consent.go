package ports

import (
	"context"

	id "credo/pkg/domain"
)

// ConsentPort defines the interface for consent checks in the decision engine.
// It mirrors registry's consent port but is defined per module to avoid coupling.
type ConsentPort interface {
	// RequireConsent enforces consent requirement.
	// Returns nil if consent is active, error otherwise.
	RequireConsent(ctx context.Context, userID id.UserID, purpose id.ConsentPurpose) error
}
