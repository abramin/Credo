package adapters

import (
	"context"
	"errors"

	"credo/internal/decision/ports"
	vcmodels "credo/internal/evidence/vc/models"
	vcstore "credo/internal/evidence/vc/store"
	id "credo/pkg/domain"
)

// VCAdapter implements ports.VCPort by directly calling the VC store.
// This maintains hexagonal architecture boundaries while keeping
// everything in a single process.
type VCAdapter struct {
	store vcstore.Store
}

// NewVCAdapter creates a new VC adapter.
func NewVCAdapter(store vcstore.Store) ports.VCPort {
	return &VCAdapter{store: store}
}

// FindBySubjectAndType retrieves a credential by user ID and type.
// Side effects: calls the VC store and may perform external I/O.
// Returns nil, nil if no credential exists (not an error).
func (a *VCAdapter) FindBySubjectAndType(ctx context.Context, userID id.UserID, credType vcmodels.CredentialType) (*vcmodels.CredentialRecord, error) {
	record, err := a.store.FindBySubjectAndType(ctx, userID, credType)
	if err != nil {
		if errors.Is(err, vcstore.ErrNotFound) {
			return nil, nil // Not found is not an error for VC lookup
		}
		return nil, err
	}
	return &record, nil
}
