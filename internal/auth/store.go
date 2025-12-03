package auth

import (
	"context"

	pkgerrors "id-gateway/pkg/errors"
)

var (
	// ErrNotFound keeps storage-specific 404s consistent across user/session
	// implementations.
	ErrNotFound = pkgerrors.New(pkgerrors.CodeNotFound, "record not found")
)

type UserStore interface {
	Save(ctx context.Context, user User) error
	FindByID(ctx context.Context, id string) (User, error)
}

type SessionStore interface {
	Save(ctx context.Context, session Session) error
	FindByID(ctx context.Context, id string) (Session, error)
}
