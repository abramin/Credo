package service

import (
	"credo/internal/evidence/registry/domain/citizen"
	"credo/internal/evidence/registry/domain/sanctions"
	"credo/internal/evidence/registry/domain/shared"
	"credo/internal/evidence/registry/models"
	"credo/internal/evidence/registry/providers"
	id "credo/pkg/domain"
)

// EvidenceToCitizenVerification converts generic Evidence to a domain CitizenVerification aggregate.
// Invalid national IDs are converted to zero values to allow graceful degradation.
func EvidenceToCitizenVerification(ev *providers.Evidence) citizen.CitizenVerification {
	if ev == nil || ev.ProviderType != providers.ProviderTypeCitizen {
		return citizen.CitizenVerification{}
	}

	nationalIDStr := getString(ev.Data, "national_id")
	nationalID, _ := id.ParseNationalID(nationalIDStr)
	// If validation fails, nationalID is zero value - allows graceful degradation

	confidence, _ := shared.NewConfidence(ev.Confidence)
	checkedAt := shared.NewCheckedAt(ev.CheckedAt)
	providerID := shared.NewProviderID(ev.ProviderID)

	details := citizen.PersonalDetails{
		FullName:    getString(ev.Data, "full_name"),
		DateOfBirth: getString(ev.Data, "date_of_birth"),
		Address:     getString(ev.Data, "address"),
	}

	return citizen.NewCitizenVerification(
		nationalID,
		details,
		getBool(ev.Data, "valid"),
		checkedAt,
		providerID,
		confidence,
	)
}

// EvidenceToSanctionsCheck converts generic Evidence to a domain SanctionsCheck aggregate.
// Invalid national IDs are converted to zero values to allow graceful degradation.
func EvidenceToSanctionsCheck(ev *providers.Evidence) sanctions.SanctionsCheck {
	if ev == nil || ev.ProviderType != providers.ProviderTypeSanctions {
		return sanctions.SanctionsCheck{}
	}

	nationalIDStr := getString(ev.Data, "national_id")
	nationalID, _ := id.ParseNationalID(nationalIDStr)
	// If validation fails, nationalID is zero value - allows graceful degradation

	confidence, _ := shared.NewConfidence(ev.Confidence)
	checkedAt := shared.NewCheckedAt(ev.CheckedAt)
	providerID := shared.NewProviderID(ev.ProviderID)
	source := sanctions.NewSource(getString(ev.Data, "source"))

	listed := getBool(ev.Data, "listed")
	if listed {
		// For listed subjects, use the appropriate constructor with listing details
		return sanctions.NewListedSanctionsCheck(
			nationalID,
			sanctions.ListTypeSanctions, // Default to sanctions type
			"",                          // Reason not provided in current mock
			"",                          // ListedDate not provided in current mock
			source,
			checkedAt,
			providerID,
			confidence,
		)
	}

	return sanctions.NewSanctionsCheck(
		nationalID,
		source,
		checkedAt,
		providerID,
		confidence,
	)
}

// CitizenVerificationToRecord converts a domain CitizenVerification to an infrastructure CitizenRecord.
// This is the outbound conversion for persistence and transport.
func CitizenVerificationToRecord(cv citizen.CitizenVerification) *models.CitizenRecord {
	return &models.CitizenRecord{
		NationalID:  cv.NationalID().String(),
		FullName:    cv.FullName(),
		DateOfBirth: cv.DateOfBirth(),
		Address:     cv.Address(),
		Valid:       cv.IsValid(),
		CheckedAt:   cv.CheckedAt().Time(),
	}
}

// SanctionsCheckToRecord converts a domain SanctionsCheck to an infrastructure SanctionsRecord.
// This is the outbound conversion for persistence and transport.
func SanctionsCheckToRecord(sc sanctions.SanctionsCheck) *models.SanctionsRecord {
	return &models.SanctionsRecord{
		NationalID: sc.NationalID().String(),
		Listed:     sc.IsListed(),
		Source:     sc.Source().String(),
		CheckedAt:  sc.CheckedAt().Time(),
	}
}

// EvidenceToCitizenRecord converts generic Evidence to a CitizenRecord via domain aggregate.
// This is a convenience function that chains Evidence → Domain → Infrastructure.
// Returns nil if the evidence is nil or not a citizen type.
func EvidenceToCitizenRecord(ev *providers.Evidence) *models.CitizenRecord {
	if ev == nil || ev.ProviderType != providers.ProviderTypeCitizen {
		return nil
	}

	verification := EvidenceToCitizenVerification(ev)
	return CitizenVerificationToRecord(verification)
}

// EvidenceToSanctionsRecord converts generic Evidence to a SanctionsRecord via domain aggregate.
// This is a convenience function that chains Evidence → Domain → Infrastructure.
// Returns nil if the evidence is nil or not a sanctions type.
func EvidenceToSanctionsRecord(ev *providers.Evidence) *models.SanctionsRecord {
	if ev == nil || ev.ProviderType != providers.ProviderTypeSanctions {
		return nil
	}

	check := EvidenceToSanctionsCheck(ev)
	return SanctionsCheckToRecord(check)
}

func getString(data map[string]interface{}, key string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return ""
}

func getBool(data map[string]interface{}, key string) bool {
	if v, ok := data[key].(bool); ok {
		return v
	}
	return false
}
