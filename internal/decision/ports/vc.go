package ports

import (
	"context"

	vcmodels "credo/internal/evidence/vc/models"
	id "credo/pkg/domain"
)

// VCPort defines the interface for VC lookups in the decision engine.
// This port allows finding existing credentials by subject/type without
// depending on the VC store implementation directly.
type VCPort interface {
	// FindBySubjectAndType retrieves a credential by user ID and type.
	// Returns nil, nil if no credential exists (not an error).
	// Returns nil, error only for infrastructure failures.
	FindBySubjectAndType(ctx context.Context, userID id.UserID, credType vcmodels.CredentialType) (*vcmodels.CredentialRecord, error)
}
