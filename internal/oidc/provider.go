package oidc

import (
	"context"

	"id-gateway/internal/domain"
)

// Provider adapts the pure OIDCFlow domain into a service that can be called by
// handlers or background tasks. Transport-specific concerns stay out.
type Provider struct {
	flow *domain.OIDCFlow
}

func NewProvider(flow *domain.OIDCFlow) *Provider {
	return &Provider{flow: flow}
}

func (p *Provider) Authorize(ctx context.Context, req domain.AuthorizationRequest) (domain.AuthorizationResult, error) {
	_ = ctx
	return p.flow.StartAuthorization(req)
}

func (p *Provider) Consent(ctx context.Context, req domain.ConsentRequest) (domain.ConsentResult, error) {
	_ = ctx
	return p.flow.RecordConsent(req)
}

func (p *Provider) Token(ctx context.Context, req domain.TokenRequest) (domain.TokenResult, error) {
	_ = ctx
	return p.flow.ExchangeToken(req)
}
