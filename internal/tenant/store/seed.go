package store

import (
	"context"
	"time"

	"github.com/google/uuid"

	clientstore "credo/internal/tenant/store/client"
	tenantstore "credo/internal/tenant/store/tenant"

	"credo/internal/tenant/models"
)

// SeedBootstrapTenant creates a default tenant and client for backward compatibility.
func SeedBootstrapTenant(ts *tenantstore.InMemory, cs *clientstore.InMemory) (*models.Tenant, *models.Client) {
	now := time.Now()
	t := &models.Tenant{ID: uuid.New(), Name: "default", Status: models.TenantStatusActive, CreatedAt: now}
	_ = ts.CreateIfNameAvailable(context.Background(), t)

	redirectURIs := []string{
		"http://localhost:3000/callback",
		"http://localhost",
	}

	c := &models.Client{
		ID:            uuid.New(),
		TenantID:      t.ID,
		Name:          "default-client",
		ClientID:      "test-client",
		RedirectURIs:  redirectURIs,
		AllowedGrants: []string{"authorization_code", "refresh_token"},
		AllowedScopes: []string{"openid", "profile"},
		Status:        models.ClientStatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	_ = cs.Create(context.Background(), c)
	return t, c
}
