package adapters

import (
	"context"

	"credo/internal/admin/types"
	authModels "credo/internal/auth/models"
	id "credo/pkg/domain"
)

// AuthUserStore is the interface that auth user stores implement.
type AuthUserStore interface {
	ListAll(ctx context.Context) (map[id.UserID]*authModels.User, error)
	FindByID(ctx context.Context, userID id.UserID) (*authModels.User, error)
}

// UserStoreAdapter adapts an auth user store to admin's UserStore interface.
type UserStoreAdapter struct {
	store AuthUserStore
}

// NewUserStoreAdapter creates a new adapter wrapping an auth user store.
func NewUserStoreAdapter(store AuthUserStore) *UserStoreAdapter {
	return &UserStoreAdapter{store: store}
}

// ListAll returns all users mapped to admin types.
func (a *UserStoreAdapter) ListAll(ctx context.Context) (map[id.UserID]*types.AdminUser, error) {
	users, err := a.store.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[id.UserID]*types.AdminUser, len(users))
	for k, u := range users {
		result[k] = mapUser(u)
	}
	return result, nil
}

// FindByID returns a user by ID mapped to admin type.
func (a *UserStoreAdapter) FindByID(ctx context.Context, userID id.UserID) (*types.AdminUser, error) {
	user, err := a.store.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return mapUser(user), nil
}

func mapUser(u *authModels.User) *types.AdminUser {
	return &types.AdminUser{
		ID:        u.ID,
		TenantID:  u.TenantID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Verified:  u.Verified,
		Active:    u.IsActive(),
	}
}
