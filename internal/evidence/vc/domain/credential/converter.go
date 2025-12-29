package credential

import (
	"credo/internal/evidence/vc/domain/shared"
	"credo/internal/evidence/vc/models"
)

// ToModel converts a domain Credential to an infrastructure VerifiableCredential model.
// This is used when persisting credentials to the store.
func ToModel(c *Credential) models.VerifiableCredential {
	return models.VerifiableCredential{
		ID:       c.id,
		Type:     c.credType,
		Subject:  c.subject,
		Issuer:   c.issuer,
		IssuedAt: c.issuedAt.Time(),
		Claims:   models.Claims(c.claims.ToMap()),
	}
}

// FromModel converts an infrastructure VerifiableCredential model to a domain Credential.
// This is used when loading credentials from the store.
// Returns an error if the model violates domain invariants.
func FromModel(m models.VerifiableCredential) (*Credential, error) {
	issuedAt, err := shared.NewIssuedAt(m.IssuedAt)
	if err != nil {
		return nil, err
	}

	// Reconstruct typed claims based on credential type
	claims := claimsFromMap(m.Type, m.Claims)

	return New(
		m.ID,
		m.Type,
		m.Subject,
		m.Issuer,
		issuedAt,
		claims,
	)
}

// claimsFromMap reconstructs typed ClaimSet from untyped map based on credential type.
func claimsFromMap(credType models.CredentialType, m models.Claims) ClaimSet {
	switch credType {
	case models.CredentialTypeAgeOver18:
		return AgeOver18ClaimsFromMap(m)
	default:
		// Fallback for unknown types - return AgeOver18 as default
		return AgeOver18ClaimsFromMap(m)
	}
}
