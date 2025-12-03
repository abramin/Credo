package auth

import "context"

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

func (s *Service) Authorize(ctx context.Context, req AuthorizationRequest) (AuthorizationResult, error) {
	_ = ctx
	return s.flow.StartAuthorization(req)
}

func (s *Service) Consent(ctx context.Context, req ConsentRequest) (ConsentResult, error) {
	_ = ctx
	return s.flow.RecordConsent(req)
}

func (s *Service) Token(ctx context.Context, req TokenRequest) (TokenResult, error) {
	_ = ctx
	return s.flow.ExchangeToken(req)
}

// OIDCFlow holds the business rules for the mock OIDC login and consent dance.
// Storage and HTTP concerns live in other layers; this stays pure.
type OIDCFlow struct{}

func NewOIDCFlow() *OIDCFlow {
	return &OIDCFlow{}
}

func (f *OIDCFlow) StartAuthorization(req AuthorizationRequest) (AuthorizationResult, error) {
	return AuthorizationResult{
		SessionID:   "todo-session-id",
		RedirectURI: req.RedirectURI,
	}, nil
}

func (f *OIDCFlow) RecordConsent(req ConsentRequest) (ConsentResult, error) {
	return ConsentResult{
		SessionID: req.SessionID,
		Approved:  req.Approved,
	}, nil
}

func (f *OIDCFlow) ExchangeToken(req TokenRequest) (TokenResult, error) {
	return TokenResult{
		AccessToken: "todo-access",
		IDToken:     "todo-id",
		ExpiresIn:   3600,
	}, nil
}
