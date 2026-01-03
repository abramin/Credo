package adapters

import (
	"context"

	"credo/internal/auth/types"
	tenantModels "credo/internal/tenant/models"
)

// tenantClientResolver is the interface that tenant service implements.
// Defined locally to avoid coupling auth adapters to tenant service package.
type tenantClientResolver interface {
	ResolveClient(ctx context.Context, clientID string) (*tenantModels.Client, *tenantModels.Tenant, error)
}

// TenantClientResolver adapts tenant service to auth.ClientResolver.
// This adapter maps tenant models to auth-local DTOs at the boundary.
type TenantClientResolver struct {
	tenantSvc tenantClientResolver
}

// NewTenantClientResolver creates a new adapter wrapping the tenant service.
func NewTenantClientResolver(svc tenantClientResolver) *TenantClientResolver {
	return &TenantClientResolver{tenantSvc: svc}
}

// ResolveClient resolves a client by OAuth client ID and maps to auth types.
func (a *TenantClientResolver) ResolveClient(ctx context.Context, clientID string) (*types.ResolvedClient, *types.ResolvedTenant, error) {
	client, tenant, err := a.tenantSvc.ResolveClient(ctx, clientID)
	if err != nil {
		return nil, nil, err
	}

	return mapClient(client), mapTenant(tenant), nil
}

func mapClient(c *tenantModels.Client) *types.ResolvedClient {
	return &types.ResolvedClient{
		ID:            c.ID,
		TenantID:      c.TenantID,
		OAuthClientID: c.OAuthClientID,
		RedirectURIs:  c.RedirectURIs,
		AllowedScopes: c.AllowedScopes,
		Active:        c.IsActive(),
	}
}

func mapTenant(t *tenantModels.Tenant) *types.ResolvedTenant {
	return &types.ResolvedTenant{
		ID:     t.ID,
		Active: t.IsActive(),
	}
}
