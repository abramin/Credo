package cache

import (
	"context"
	"credo/internal/evidence/registry/models"
)

// RegistryCacheStore provides registry caching operations.
// Note: This is a stub implementation with unimplemented methods.
type RegistryCacheStore struct {
	// *[string[string]*[string[string}

}

// FindCitizen retrieves a cached citizen record by national ID.
func (r RegistryCacheStore) FindCitizen(ctx context.Context, nationalID string) (*models.CitizenRecord, error) {
	panic("unimplemented")
}

// FindSanction retrieves a cached sanctions record by national ID.
func (r RegistryCacheStore) FindSanction(ctx context.Context, nationalID string) (*models.SanctionsRecord, error) {
	panic("unimplemented")
}

// SaveSanction stores a sanctions record in the cache.
func (r RegistryCacheStore) SaveSanction(ctx context.Context, param any) error {
	panic("unimplemented")
}

// SaveCitizen stores a citizen record in the cache.
func (r RegistryCacheStore) SaveCitizen(ctx context.Context, param any) error {
	panic("unimplemented")
}
