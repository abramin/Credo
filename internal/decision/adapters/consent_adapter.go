package adapters

import (
	"context"

	consentModels "credo/internal/consent/models"
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
// This is implemented via RequireConsent - if no error, consent exists.
func (a *ConsentAdapter) HasConsent(ctx context.Context, userID string, purpose string) (bool, error) {
	uid, err := id.ParseUserID(userID)
	if err != nil {
		return false, err
	}

	consentPurpose, err := consentModels.ParsePurpose(purpose)
	if err != nil {
		return false, err
	}

	err = a.consent.Require(ctx, uid, consentPurpose)
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
// Returns nil if consent is active, error otherwise.
func (a *ConsentAdapter) RequireConsent(ctx context.Context, userID string, purpose string) error {
	uid, err := id.ParseUserID(userID)
	if err != nil {
		return err
	}

	consentPurpose, err := consentModels.ParsePurpose(purpose)
	if err != nil {
		return err
	}

	return a.consent.Require(ctx, uid, consentPurpose)
}
