package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"credo/internal/ratelimit/models"
	"credo/internal/ratelimit/service"
	"credo/internal/ratelimit/store/allowlist"
	"credo/internal/ratelimit/store/bucket"
)

// HandlerSuite provides shared test setup for rate limit handler tests.
// Per AGENTS.md: Uses real components, not mocks.
// Per testing.md: Handler tests validate HTTP concerns (parsing, response mapping).
type HandlerSuite struct {
	suite.Suite
	router  http.Handler
	handler *Handler
}

func (s *HandlerSuite) SetupTest() {
	// Use real in-memory stores - no mocks per AGENTS.md
	buckets := bucket.NewInMemoryBucketStore()
	allowlistStore := allowlist.NewInMemoryAllowlistStore()

	svc, err := service.New(buckets, allowlistStore)
	require.NoError(s.T(), err)

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	s.handler = New(svc, logger)

	r := chi.NewRouter()
	// TODO: Add admin auth middleware when implemented
	// r.Use(middleware.RequireAdminToken(adminToken, logger))
	s.handler.RegisterAdmin(r)
	s.router = r
}

func TestHandlerSuite(t *testing.T) {
	suite.Run(t, new(HandlerSuite))
}

// =============================================================================
// HandleAddAllowlist Tests
// Per PRD-017 FR-4: POST /admin/rate-limit/allowlist
// =============================================================================

func (s *HandlerSuite) TestAddAllowlist_InvalidJSON() {
	// Handler test: validates request parsing (HTTP concern)
	req := httptest.NewRequest(http.MethodPost, "/admin/rate-limit/allowlist",
		bytes.NewReader([]byte("not valid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	assert.Equal(s.T(), http.StatusBadRequest, rec.Code,
		"expected 400 for invalid JSON")
}

func (s *HandlerSuite) TestAddAllowlist_ValidRequest() {
	s.T().Skip("TODO: Enable after HandleAddAllowlist is implemented")

	payload := models.AddAllowlistRequest{
		Type:       models.AllowlistTypeIP,
		Identifier: "192.168.1.100",
		Reason:     "Internal monitoring service",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/admin/rate-limit/allowlist",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	require.Equal(s.T(), http.StatusOK, rec.Code, "expected 200 for valid request")

	var resp models.AllowlistEntryResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(s.T(), err, "failed to decode response")
	assert.True(s.T(), resp.Allowlisted, "expected allowlisted=true")
	assert.Equal(s.T(), "192.168.1.100", resp.Identifier)
}

func (s *HandlerSuite) TestAddAllowlist_WithExpiration() {
	s.T().Skip("TODO: Enable after HandleAddAllowlist is implemented")

	expiresAt := time.Now().Add(24 * time.Hour)
	payload := models.AddAllowlistRequest{
		Type:       models.AllowlistTypeUserID,
		Identifier: "user-123",
		Reason:     "VIP user bypass",
		ExpiresAt:  &expiresAt,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/admin/rate-limit/allowlist",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	require.Equal(s.T(), http.StatusOK, rec.Code)

	var resp models.AllowlistEntryResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(s.T(), err)
	assert.NotNil(s.T(), resp.ExpiresAt, "expected expires_at in response")
}

// =============================================================================
// HandleRemoveAllowlist Tests
// Per PRD-017 FR-4: DELETE /admin/rate-limit/allowlist
// =============================================================================

func (s *HandlerSuite) TestRemoveAllowlist_InvalidJSON() {
	req := httptest.NewRequest(http.MethodDelete, "/admin/rate-limit/allowlist",
		bytes.NewReader([]byte("not valid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	assert.Equal(s.T(), http.StatusBadRequest, rec.Code,
		"expected 400 for invalid JSON")
}

func (s *HandlerSuite) TestRemoveAllowlist_ValidRequest() {
	s.T().Skip("TODO: Enable after HandleRemoveAllowlist is implemented")

	payload := models.RemoveAllowlistRequest{
		Type:       models.AllowlistTypeIP,
		Identifier: "192.168.1.100",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodDelete, "/admin/rate-limit/allowlist",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	assert.Equal(s.T(), http.StatusNoContent, rec.Code,
		"expected 204 for successful removal")
}

func (s *HandlerSuite) TestRemoveAllowlist_NotFound() {
	s.T().Skip("TODO: Enable after HandleRemoveAllowlist is implemented")

	payload := models.RemoveAllowlistRequest{
		Type:       models.AllowlistTypeIP,
		Identifier: "1.2.3.4", // Never added
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodDelete, "/admin/rate-limit/allowlist",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	// Per PRD-017 FR-4: 404 when removing non-existent entry
	assert.Equal(s.T(), http.StatusNotFound, rec.Code,
		"expected 404 for non-existent entry")
}

// =============================================================================
// HandleListAllowlist Tests
// Per PRD-017 FR-4: GET /admin/rate-limit/allowlist
// =============================================================================

func (s *HandlerSuite) TestListAllowlist_Empty() {
	s.T().Skip("TODO: Enable after HandleListAllowlist is implemented")

	req := httptest.NewRequest(http.MethodGet, "/admin/rate-limit/allowlist", nil)
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	require.Equal(s.T(), http.StatusOK, rec.Code)

	var entries []models.AllowlistEntry
	err := json.NewDecoder(rec.Body).Decode(&entries)
	require.NoError(s.T(), err)
	assert.Empty(s.T(), entries, "expected empty list")
}

func (s *HandlerSuite) TestListAllowlist_WithEntries() {
	s.T().Skip("TODO: Enable after HandleListAllowlist is implemented")

	// First add an entry
	payload := models.AddAllowlistRequest{
		Type:       models.AllowlistTypeIP,
		Identifier: "10.0.0.1",
		Reason:     "Test entry",
	}
	body, _ := json.Marshal(payload)
	addReq := httptest.NewRequest(http.MethodPost, "/admin/rate-limit/allowlist",
		bytes.NewReader(body))
	addReq.Header.Set("Content-Type", "application/json")
	addRec := httptest.NewRecorder()
	s.router.ServeHTTP(addRec, addReq)
	require.Equal(s.T(), http.StatusOK, addRec.Code)

	// Then list
	listReq := httptest.NewRequest(http.MethodGet, "/admin/rate-limit/allowlist", nil)
	listRec := httptest.NewRecorder()
	s.router.ServeHTTP(listRec, listReq)

	require.Equal(s.T(), http.StatusOK, listRec.Code)

	var entries []*models.AllowlistEntry
	err := json.NewDecoder(listRec.Body).Decode(&entries)
	require.NoError(s.T(), err)
	assert.Len(s.T(), entries, 1, "expected one entry")
}

// =============================================================================
// HandleResetRateLimit Tests
// Per PRD-017 TR-1: POST /admin/rate-limit/reset
// =============================================================================

func (s *HandlerSuite) TestResetRateLimit_InvalidJSON() {
	req := httptest.NewRequest(http.MethodPost, "/admin/rate-limit/reset",
		bytes.NewReader([]byte("not valid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	assert.Equal(s.T(), http.StatusBadRequest, rec.Code,
		"expected 400 for invalid JSON")
}

func (s *HandlerSuite) TestResetRateLimit_ValidRequest() {
	s.T().Skip("TODO: Enable after HandleResetRateLimit is implemented")

	payload := models.ResetRateLimitRequest{
		Type:       models.AllowlistTypeIP,
		Identifier: "192.168.1.100",
		Class:      models.ClassAuth,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/admin/rate-limit/reset",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	assert.Equal(s.T(), http.StatusNoContent, rec.Code,
		"expected 204 for successful reset")
}

func (s *HandlerSuite) TestResetRateLimit_AllClasses() {
	s.T().Skip("TODO: Enable after HandleResetRateLimit is implemented")

	// Reset all classes for an identifier (class omitted)
	payload := models.ResetRateLimitRequest{
		Type:       models.AllowlistTypeIP,
		Identifier: "192.168.1.100",
		// Class omitted = reset all classes
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/admin/rate-limit/reset",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	assert.Equal(s.T(), http.StatusNoContent, rec.Code,
		"expected 204 for successful reset of all classes")
}
