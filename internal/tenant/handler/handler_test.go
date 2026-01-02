package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"credo/internal/tenant/models"
	"credo/internal/tenant/readmodels"
	"credo/internal/tenant/service"
	clientstore "credo/internal/tenant/store/client"
	tenantstore "credo/internal/tenant/store/tenant"
	id "credo/pkg/domain"
	dErrors "credo/pkg/domain-errors"
	auditpublisher "credo/pkg/platform/audit/publisher"
	auditmemory "credo/pkg/platform/audit/store/memory"
	adminmw "credo/pkg/platform/middleware/admin"
)

const adminToken = "secret-token"

type HandlerSuite struct {
	suite.Suite
	router http.Handler
}

func (s *HandlerSuite) SetupTest() {
	tenants := tenantstore.NewInMemory()
	clients := clientstore.NewInMemory()
	auditStore := auditmemory.NewInMemoryStore()
	svc, err := service.New(
		tenants,
		clients,
		nil,
		service.WithAuditPublisher(auditpublisher.NewPublisher(auditStore)),
	)
	s.Require().NoError(err)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	h := New(svc, logger)
	r := chi.NewRouter()
	r.Use(adminmw.RequireAdminToken(adminToken, logger))
	h.Register(r)
	s.router = r
}

func TestHandlerSuite(t *testing.T) {
	suite.Run(t, new(HandlerSuite))
}

// TestAdminTokenRequired verifies middleware wiring - admin endpoints reject
// requests without valid admin token. This validates handler-to-middleware
// integration that E2E tests also cover, but kept here to catch wiring regressions
// in isolation without spinning up the full server.
func (s *HandlerSuite) TestAdminTokenRequired() {
	req := httptest.NewRequest(http.MethodGet, "/admin/tenants/"+uuid.New().String(), nil)
	// No admin token header set
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	s.Equal(http.StatusUnauthorized, rec.Code, "expected 401 when admin token missing")
}

// ErrorMappingSuite tests domain error to HTTP status code translation.
// Feature files can only assert final HTTP status codes; these tests verify
// that specific domain error codes are correctly mapped through the handler layer.
type ErrorMappingSuite struct {
	suite.Suite
	handler *Handler
	router  http.Handler
}

func TestErrorMappingSuite(t *testing.T) {
	suite.Run(t, new(ErrorMappingSuite))
}

func (s *ErrorMappingSuite) SetupTest() {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	s.handler = New(&stubService{}, logger)

	r := chi.NewRouter()
	r.Use(adminmw.RequireAdminToken(adminToken, logger))
	s.handler.Register(r)
	s.router = r
}

// TestTenantErrorMapping verifies tenant endpoint error code translation.
func (s *ErrorMappingSuite) TestTenantErrorMapping() {
	s.Run("CodeNotFound maps to 404", func() {
		req := s.newRequest(http.MethodGet, "/admin/tenants/"+uuid.New().String())
		rec := httptest.NewRecorder()
		s.router.ServeHTTP(rec, req)

		s.Equal(http.StatusNotFound, rec.Code)
		s.assertErrorCode(rec, "not_found")
	})

	s.Run("CodeConflict maps to 409 on deactivate", func() {
		// stubService returns CodeConflict for deactivate (simulating already inactive)
		req := s.newRequest(http.MethodPost, "/admin/tenants/"+stubConflictTenantID.String()+"/deactivate")
		rec := httptest.NewRecorder()
		s.router.ServeHTTP(rec, req)

		s.Equal(http.StatusConflict, rec.Code)
		s.assertErrorCode(rec, "conflict")
	})

	s.Run("CodeConflict maps to 409 on reactivate", func() {
		req := s.newRequest(http.MethodPost, "/admin/tenants/"+stubConflictTenantID.String()+"/reactivate")
		rec := httptest.NewRecorder()
		s.router.ServeHTTP(rec, req)

		s.Equal(http.StatusConflict, rec.Code)
		s.assertErrorCode(rec, "conflict")
	})
}

// TestClientErrorMapping verifies client endpoint error code translation.
func (s *ErrorMappingSuite) TestClientErrorMapping() {
	s.Run("CodeNotFound maps to 404", func() {
		req := s.newRequest(http.MethodGet, "/admin/clients/"+uuid.New().String())
		rec := httptest.NewRecorder()
		s.router.ServeHTTP(rec, req)

		s.Equal(http.StatusNotFound, rec.Code)
		s.assertErrorCode(rec, "not_found")
	})

	s.Run("CodeConflict maps to 409 on deactivate", func() {
		req := s.newRequest(http.MethodPost, "/admin/clients/"+stubConflictClientID.String()+"/deactivate")
		rec := httptest.NewRecorder()
		s.router.ServeHTTP(rec, req)

		s.Equal(http.StatusConflict, rec.Code)
		s.assertErrorCode(rec, "conflict")
	})

	s.Run("CodeConflict maps to 409 on reactivate", func() {
		req := s.newRequest(http.MethodPost, "/admin/clients/"+stubConflictClientID.String()+"/reactivate")
		rec := httptest.NewRecorder()
		s.router.ServeHTTP(rec, req)

		s.Equal(http.StatusConflict, rec.Code)
		s.assertErrorCode(rec, "conflict")
	})

	s.Run("CodeValidation maps to 400 on rotate secret for public client", func() {
		req := s.newRequest(http.MethodPost, "/admin/clients/"+stubPublicClientID.String()+"/rotate-secret")
		rec := httptest.NewRecorder()
		s.router.ServeHTTP(rec, req)

		s.Equal(http.StatusBadRequest, rec.Code)
		s.assertErrorCode(rec, "validation_error")
	})
}

func (s *ErrorMappingSuite) newRequest(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("X-Admin-Token", adminToken)
	return req
}

func (s *ErrorMappingSuite) assertErrorCode(rec *httptest.ResponseRecorder, expectedCode string) {
	var resp map[string]string
	err := json.NewDecoder(rec.Body).Decode(&resp)
	s.Require().NoError(err)
	s.Equal(expectedCode, resp["error"])
}

// Stub IDs for specific error scenarios
var (
	stubConflictTenantID = id.TenantID(uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	stubConflictClientID = id.ClientID(uuid.MustParse("22222222-2222-2222-2222-222222222222"))
	stubPublicClientID   = id.ClientID(uuid.MustParse("33333333-3333-3333-3333-333333333333"))
)

// stubService is a minimal Service implementation for testing error mapping.
// It returns specific errors based on IDs to test handler error translation.
type stubService struct{}

func (s *stubService) CreateTenant(ctx context.Context, name string) (*models.Tenant, error) {
	return nil, dErrors.New(dErrors.CodeInternal, "not implemented")
}

func (s *stubService) GetTenantDetails(ctx context.Context, tenantID id.TenantID) (*readmodels.TenantDetails, error) {
	return nil, dErrors.New(dErrors.CodeNotFound, "tenant not found")
}

func (s *stubService) GetTenantByName(ctx context.Context, name string) (*models.Tenant, error) {
	return nil, dErrors.New(dErrors.CodeNotFound, "tenant not found")
}

func (s *stubService) DeactivateTenant(ctx context.Context, tenantID id.TenantID) (*models.Tenant, error) {
	if tenantID == stubConflictTenantID {
		return nil, dErrors.New(dErrors.CodeConflict, "tenant already inactive")
	}
	return nil, dErrors.New(dErrors.CodeNotFound, "tenant not found")
}

func (s *stubService) ReactivateTenant(ctx context.Context, tenantID id.TenantID) (*models.Tenant, error) {
	if tenantID == stubConflictTenantID {
		return nil, dErrors.New(dErrors.CodeConflict, "tenant already active")
	}
	return nil, dErrors.New(dErrors.CodeNotFound, "tenant not found")
}

func (s *stubService) CreateClient(ctx context.Context, cmd *service.CreateClientCommand) (*models.Client, string, error) {
	return nil, "", dErrors.New(dErrors.CodeInternal, "not implemented")
}

func (s *stubService) GetClient(ctx context.Context, clientID id.ClientID) (*models.Client, error) {
	return nil, dErrors.New(dErrors.CodeNotFound, "client not found")
}

func (s *stubService) GetClientForTenant(ctx context.Context, tenantID id.TenantID, clientID id.ClientID) (*models.Client, error) {
	return nil, dErrors.New(dErrors.CodeNotFound, "client not found")
}

func (s *stubService) UpdateClient(ctx context.Context, clientID id.ClientID, cmd *service.UpdateClientCommand) (*models.Client, string, error) {
	return nil, "", dErrors.New(dErrors.CodeNotFound, "client not found")
}

func (s *stubService) UpdateClientForTenant(ctx context.Context, tenantID id.TenantID, clientID id.ClientID, cmd *service.UpdateClientCommand) (*models.Client, string, error) {
	return nil, "", dErrors.New(dErrors.CodeNotFound, "client not found")
}

func (s *stubService) DeactivateClient(ctx context.Context, clientID id.ClientID) (*models.Client, error) {
	if clientID == stubConflictClientID {
		return nil, dErrors.New(dErrors.CodeConflict, "client already inactive")
	}
	return nil, dErrors.New(dErrors.CodeNotFound, "client not found")
}

func (s *stubService) ReactivateClient(ctx context.Context, clientID id.ClientID) (*models.Client, error) {
	if clientID == stubConflictClientID {
		return nil, dErrors.New(dErrors.CodeConflict, "client already active")
	}
	return nil, dErrors.New(dErrors.CodeNotFound, "client not found")
}

func (s *stubService) RotateClientSecret(ctx context.Context, clientID id.ClientID) (*models.Client, string, error) {
	if clientID == stubPublicClientID {
		return nil, "", dErrors.New(dErrors.CodeValidation, "cannot rotate secret for public client")
	}
	return nil, "", dErrors.New(dErrors.CodeNotFound, "client not found")
}

func (s *stubService) RotateClientSecretForTenant(ctx context.Context, tenantID id.TenantID, clientID id.ClientID) (*models.Client, string, error) {
	return nil, "", dErrors.New(dErrors.CodeNotFound, "client not found")
}
