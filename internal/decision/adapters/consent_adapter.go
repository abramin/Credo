package adapters

import (
	"context"

	consentService "credo/internal/consent/service"
	"credo/internal/decision/ports"
	id "credo/pkg/domain"
	dErrors "credo/pkg/domain-errors"
)

// ConsentAdapter implements ports.ConsentPort by calling the consent service.
// This maintains hexagonal architecture boundaries while keeping
// everything in a single process.
type ConsentAdapter struct {
	consent *consentService.Service
}

// NewConsentAdapter creates a new consent adapter.
func NewConsentAdapter(consent *consentService.Service) ports.ConsentPort {
	return &ConsentAdapter{consent: consent}
}

// HasConsent checks if a user has active consent for a purpose.
// Side effects: calls the consent service; missing consent maps to false.
func (a *ConsentAdapter) HasConsent(ctx context.Context, userID id.UserID, purpose id.ConsentPurpose) (bool, error) {
	err := a.consent.Require(ctx, userID, purpose)
	if err != nil {
		// Check if it's a missing consent error (expected) vs infrastructure error
		if dErrors.HasCode(err, dErrors.CodeMissingConsent) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// RequireConsent enforces consent requirement.
// Side effects: calls the consent service and returns its error.
func (a *ConsentAdapter) RequireConsent(ctx context.Context, userID id.UserID, purpose id.ConsentPurpose) error {
	return a.consent.Require(ctx, userID, purpose)
}
