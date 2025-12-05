package store

import (
	"context"

	"id-gateway/internal/auth/models"
	dErrors "id-gateway/pkg/domain-errors"

	"github.com/google/uuid"
)

var (
	// ErrNotFound keeps storage-specific not found errors consistent across user/session
	// implementations.
	ErrNotFound = dErrors.New(dErrors.CodeNotFound, "record not found")
)

type UserStore interface {
	Save(ctx context.Context, user *models.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindOrCreateByEmail(ctx context.Context, email string, user *models.User) (*models.User, error)
	DeleteUser(ctx context.Context, id string) error // For GDPR
}

type SessionStore interface {
	Save(ctx context.Context, session *models.Session) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Session, error)
	FindByCode(ctx context.Context, code string) (*models.Session, error)
	DeleteSession(ctx context.Context, id uuid.UUID) error
	DeleteSessionsByUser(ctx context.Context, userID uuid.UUID) error // For GDPR
}
