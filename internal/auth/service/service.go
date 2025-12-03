package service

import (
	"context"

	"id-gateway/internal/auth/models"
)

type UserStore interface {
	Save(ctx context.Context, user models.User) error
	FindByID(ctx context.Context, id string) (models.User, error)
}

type SessionStore interface {
	Save(ctx context.Context, session models.Session) error
	FindByID(ctx context.Context, id string) (models.Session, error)
}

// Service adapts OIDC flow interactions into a callable façade. It keeps
// transport concerns out of business logic.
type Service struct {
	users    UserStore
	sessions SessionStore
}

func NewService(users UserStore, sessions SessionStore) *Service {
	return &Service{
		users:    users,
		sessions: sessions,
	}
}

func (s *Service) Authorize(ctx context.Context, req *models.AuthorizationRequest) (*models.AuthorizationResult, error) {
	_ = ctx
	return nil, nil
}

func (s *Service) Consent(ctx context.Context, req *models.ConsentRequest) (*models.ConsentResult, error) {
	_ = ctx
	return nil, nil
}

func (s *Service) Token(ctx context.Context, req *models.TokenRequest) (*models.TokenResult, error) {
	_ = ctx
	return nil, nil
}
