package ports

import "context"

// ConsentPort defines the interface for consent checks in the decision engine
// Identical to registry's ConsentPort - we define it per module to avoid coupling
type ConsentPort interface {
	// HasConsent checks if a user has active consent for a purpose
	HasConsent(ctx context.Context, userID string, purpose string) (bool, error)

	// RequireConsent enforces consent requirement
	// Returns nil if consent is active, error otherwise
	RequireConsent(ctx context.Context, userID string, purpose string) error
}
