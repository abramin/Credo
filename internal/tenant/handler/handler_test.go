package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"credo/internal/platform/middleware"
	"credo/internal/tenant/service"
	"credo/internal/tenant/store"
)

const adminToken = "secret-token"

func TestAdminTokenRequired(t *testing.T) {
	router := newTenantRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/admin/tenants/"+uuid.New().String(), nil)
	// No admin token header set
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when admin token missing, got %d", rec.Code)
	}
}

func TestCreateTenantAndClientViaHandlers(t *testing.T) {
	router := newTenantRouter(t)

	tenantPayload := map[string]string{"name": "Acme"}
	body, _ := json.Marshal(tenantPayload)
	req := httptest.NewRequest(http.MethodPost, "/admin/tenants", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Token", adminToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating tenant, got %d", rec.Code)
	}

	var tenantResp struct {
		TenantID uuid.UUID `json:"tenant_id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&tenantResp); err != nil {
		t.Fatalf("failed to decode tenant response: %v", err)
	}
	if tenantResp.TenantID == uuid.Nil {
		t.Fatalf("expected tenant_id in response")
	}

	clientPayload := map[string]any{
		"tenant_id":      tenantResp.TenantID,
		"name":           "Web",
		"redirect_uris":  []string{"https://app.example.com/callback"},
		"allowed_grants": []string{"authorization_code"},
		"allowed_scopes": []string{"openid"},
	}
	clientBody, _ := json.Marshal(clientPayload)
	clientReq := httptest.NewRequest(http.MethodPost, "/admin/clients", bytes.NewReader(clientBody))
	clientReq.Header.Set("Content-Type", "application/json")
	clientReq.Header.Set("X-Admin-Token", adminToken)
	clientRec := httptest.NewRecorder()
	router.ServeHTTP(clientRec, clientReq)
	if clientRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating client, got %d", clientRec.Code)
	}

	var clientResp struct {
		ID           uuid.UUID `json:"id"`
		TenantID     uuid.UUID `json:"tenant_id"`
		ClientSecret string    `json:"client_secret"`
	}
	if err := json.NewDecoder(clientRec.Body).Decode(&clientResp); err != nil {
		t.Fatalf("failed to decode client response: %v", err)
	}
	if clientResp.ID == uuid.Nil || clientResp.ClientSecret == "" {
		t.Fatalf("expected client id and secret in response")
	}
	if clientResp.TenantID != tenantResp.TenantID {
		t.Fatalf("expected client tenant_id to match created tenant")
	}

	tenantGetReq := httptest.NewRequest(http.MethodGet, "/admin/tenants/"+tenantResp.TenantID.String(), nil)
	tenantGetReq.Header.Set("X-Admin-Token", adminToken)
	tenantGetRec := httptest.NewRecorder()
	router.ServeHTTP(tenantGetRec, tenantGetReq)
	if tenantGetRec.Code != http.StatusOK {
		t.Fatalf("expected 200 fetching tenant, got %d", tenantGetRec.Code)
	}

	var tenantDetails struct {
		ClientCount int `json:"client_count"`
		UserCount   int `json:"user_count"`
	}
	if err := json.NewDecoder(tenantGetRec.Body).Decode(&tenantDetails); err != nil {
		t.Fatalf("failed to decode tenant details: %v", err)
	}
	if tenantDetails.ClientCount != 1 {
		t.Fatalf("expected client_count 1, got %d", tenantDetails.ClientCount)
	}
}

func newTenantRouter(t *testing.T) http.Handler {
	t.Helper()
	tenants := store.NewInMemoryTenantStore()
	clients := store.NewInMemoryClientStore()
	svc := service.New(tenants, clients, nil)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	h := New(svc, logger)
	r := chi.NewRouter()
	r.Use(middleware.RequireAdminToken(adminToken, logger))
	h.Register(r)
	return r
}
