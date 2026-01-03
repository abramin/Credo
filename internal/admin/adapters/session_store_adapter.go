package adapters

import (
	"context"

	"credo/internal/admin/types"
	authModels "credo/internal/auth/models"
	id "credo/pkg/domain"
)

// AuthSessionStore is the interface that auth session stores implement.
type AuthSessionStore interface {
	ListAll(ctx context.Context) (map[id.SessionID]*authModels.Session, error)
	ListByUser(ctx context.Context, userID id.UserID) ([]*authModels.Session, error)
}

// SessionStoreAdapter adapts an auth session store to admin's SessionStore interface.
type SessionStoreAdapter struct {
	store AuthSessionStore
}

// NewSessionStoreAdapter creates a new adapter wrapping an auth session store.
func NewSessionStoreAdapter(store AuthSessionStore) *SessionStoreAdapter {
	return &SessionStoreAdapter{store: store}
}

// ListAll returns all sessions mapped to admin types.
func (a *SessionStoreAdapter) ListAll(ctx context.Context) (map[id.SessionID]*types.AdminSession, error) {
	sessions, err := a.store.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[id.SessionID]*types.AdminSession, len(sessions))
	for k, s := range sessions {
		result[k] = mapSession(s)
	}
	return result, nil
}

// ListByUser returns sessions for a user mapped to admin types.
func (a *SessionStoreAdapter) ListByUser(ctx context.Context, userID id.UserID) ([]*types.AdminSession, error) {
	sessions, err := a.store.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]*types.AdminSession, len(sessions))
	for i, s := range sessions {
		result[i] = mapSession(s)
	}
	return result, nil
}

func mapSession(s *authModels.Session) *types.AdminSession {
	return &types.AdminSession{
		ID:        s.ID,
		UserID:    s.UserID,
		CreatedAt: s.CreatedAt,
		ExpiresAt: s.ExpiresAt,
		Active:    s.IsActive(),
	}
}
