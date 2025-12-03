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
	flow     *OIDCFlow
	users    UserStore
	sessions SessionStore
}

func NewService(flow *OIDCFlow, users UserStore, sessions SessionStore) *Service {
	return &Service{
		flow:     flow,
		users:    users,
		sessions: sessions,
	}
}

func (s *Service) Authorize(ctx context.Context, req models.AuthorizationRequest) (models.AuthorizationResult, error) {
	_ = ctx
	return s.flow.StartAuthorization(req)
}

func (s *Service) Consent(ctx context.Context, req models.ConsentRequest) (models.ConsentResult, error) {
	_ = ctx
	return s.flow.RecordConsent(req)
}

func (s *Service) Token(ctx context.Context, req models.TokenRequest) (models.TokenResult, error) {
	_ = ctx
	return s.flow.ExchangeToken(req)
}

// OIDCFlow holds the business rules for the mock OIDC login and consent dance.
// Storage and HTTP concerns live in other layers; this stays pure.
type OIDCFlow struct{}

func NewOIDCFlow() *OIDCFlow {
	return &OIDCFlow{}
}

func (f *OIDCFlow) StartAuthorization(req models.AuthorizationRequest) (models.AuthorizationResult, error) {
	return models.AuthorizationResult{
		SessionID:   "todo-session-id",
		RedirectURI: req.RedirectURI,
	}, nil
}

func (f *OIDCFlow) RecordConsent(req models.ConsentRequest) (models.ConsentResult, error) {
	return models.ConsentResult{
		SessionID: req.SessionID,
		Approved:  req.Approved,
	}, nil
}

func (f *OIDCFlow) ExchangeToken(req models.TokenRequest) (models.TokenResult, error) {
	return models.TokenResult{
		AccessToken: "todo-access",
		IDToken:     "todo-id",
		ExpiresIn:   3600,
	}, nil
}
