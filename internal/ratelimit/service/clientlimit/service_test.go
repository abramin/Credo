package clientlimit

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"credo/internal/ratelimit/config"
	bucketStore "credo/internal/ratelimit/store/bucket"
)

// =============================================================================
// Client Limit Service Test Suite
// =============================================================================
// Justification for unit tests: The client limit service contains type-based
// limit selection and fallback behavior on lookup failures that are difficult
// to exercise precisely through middleware/E2E tests.

type ClientLimitServiceSuite struct {
	suite.Suite
	buckets      *bucketStore.InMemoryBucketStore
	clientLookup *mockClientLookup
	service      *Service
}

func TestClientLimitServiceSuite(t *testing.T) {
	suite.Run(t, new(ClientLimitServiceSuite))
}

func (s *ClientLimitServiceSuite) SetupTest() {
	s.buckets = bucketStore.New()
	s.clientLookup = &mockClientLookup{
		clients: make(map[string]bool),
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	var err error
	s.service, err = New(
		s.buckets,
		s.clientLookup,
		WithLogger(logger),
	)
	s.Require().NoError(err)
}

// =============================================================================
// Mock Client Lookup
// =============================================================================

type mockClientLookup struct {
	clients   map[string]bool // clientID -> isConfidential
	shouldErr bool
}

func (m *mockClientLookup) IsConfidentialClient(_ context.Context, clientID string) (bool, error) {
	if m.shouldErr {
		return false, errors.New("lookup failed")
	}
	isConfidential, exists := m.clients[clientID]
	if !exists {
		return false, nil // Unknown clients are public
	}
	return isConfidential, nil
}

// =============================================================================
// Constructor Tests (Invariant Enforcement)
// =============================================================================

func (s *ClientLimitServiceSuite) TestNew() {
	s.Run("nil buckets store returns error", func() {
		_, err := New(nil, s.clientLookup)
		s.Error(err)
		s.Contains(err.Error(), "buckets store is required")
	})

	s.Run("nil client lookup returns error", func() {
		_, err := New(s.buckets, nil)
		s.Error(err)
		s.Contains(err.Error(), "client lookup is required")
	})

	s.Run("valid dependencies returns configured service", func() {
		svc, err := New(s.buckets, s.clientLookup)
		s.NoError(err)
		s.NotNil(svc)
	})
}

// =============================================================================
// Check Tests - Client Type Selection
// =============================================================================

func (s *ClientLimitServiceSuite) TestClientTypeSelection() {
	ctx := context.Background()
	cfg := config.DefaultConfig()

	s.Run("empty client_id bypasses rate limiting", func() {
		result, err := s.service.Check(ctx, "", "/auth/token")
		s.NoError(err)
		s.True(result.Allowed)
		s.Equal(0, result.Limit) // No limit applied
	})

	s.Run("confidential client uses higher limit", func() {
		s.clientLookup.clients["confidential-app"] = true

		result, err := s.service.Check(ctx, "confidential-app", "/auth/token")
		s.NoError(err)
		s.True(result.Allowed)
		s.Equal(cfg.ClientLimits.ConfidentialLimit.RequestsPerWindow, result.Limit)
	})

	s.Run("public client uses lower limit", func() {
		s.clientLookup.clients["public-spa"] = false

		result, err := s.service.Check(ctx, "public-spa", "/auth/token")
		s.NoError(err)
		s.True(result.Allowed)
		s.Equal(cfg.ClientLimits.PublicLimit.RequestsPerWindow, result.Limit)
	})

	s.Run("unknown client defaults to public limits", func() {
		// Not registered in mock - should default to public
		result, err := s.service.Check(ctx, "unknown-client", "/auth/token")
		s.NoError(err)
		s.True(result.Allowed)
		s.Equal(cfg.ClientLimits.PublicLimit.RequestsPerWindow, result.Limit)
	})
}

// =============================================================================
// Check Tests - Fallback Behavior
// =============================================================================

func (s *ClientLimitServiceSuite) TestLookupFailureFallback() {
	ctx := context.Background()
	cfg := config.DefaultConfig()

	s.Run("lookup failure defaults to public limits", func() {
		s.clientLookup.shouldErr = true
		defer func() { s.clientLookup.shouldErr = false }()

		result, err := s.service.Check(ctx, "any-client", "/auth/token")
		s.NoError(err) // Should NOT propagate error
		s.True(result.Allowed)
		s.Equal(cfg.ClientLimits.PublicLimit.RequestsPerWindow, result.Limit)
	})
}

// =============================================================================
// Check Tests - Rate Limit Enforcement
// =============================================================================

func (s *ClientLimitServiceSuite) TestRateLimitEnforcement() {
	ctx := context.Background()
	cfg := config.DefaultConfig()

	s.Run("confidential client blocked after exceeding limit", func() {
		s.clientLookup.clients["high-volume-app"] = true
		limit := cfg.ClientLimits.ConfidentialLimit.RequestsPerWindow

		// Exhaust the limit
		for i := 0; i < limit; i++ {
			result, err := s.service.Check(ctx, "high-volume-app", "/auth/token")
			s.Require().NoError(err)
			s.True(result.Allowed, "request %d should be allowed", i+1)
		}

		// Next request should be blocked
		result, err := s.service.Check(ctx, "high-volume-app", "/auth/token")
		s.NoError(err)
		s.False(result.Allowed)
		s.Greater(result.RetryAfter, 0)
	})

	s.Run("public client blocked after exceeding lower limit", func() {
		s.clientLookup.clients["mobile-app"] = false
		limit := cfg.ClientLimits.PublicLimit.RequestsPerWindow

		// Exhaust the limit
		for i := 0; i < limit; i++ {
			result, err := s.service.Check(ctx, "mobile-app", "/auth/token")
			s.Require().NoError(err)
			s.True(result.Allowed)
		}

		// Next request should be blocked
		result, err := s.service.Check(ctx, "mobile-app", "/auth/token")
		s.NoError(err)
		s.False(result.Allowed)
	})
}

// =============================================================================
// Check Tests - Endpoint Isolation
// =============================================================================

func (s *ClientLimitServiceSuite) TestEndpointIsolation() {
	ctx := context.Background()

	s.Run("different endpoints have separate limits", func() {
		s.clientLookup.clients["multi-endpoint-app"] = false
		cfg := config.DefaultConfig()
		limit := cfg.ClientLimits.PublicLimit.RequestsPerWindow

		// Exhaust limit on /auth/token
		for i := 0; i < limit; i++ {
			_, _ = s.service.Check(ctx, "multi-endpoint-app", "/auth/token")
		}

		// /auth/authorize should still have quota
		result, err := s.service.Check(ctx, "multi-endpoint-app", "/auth/authorize")
		s.NoError(err)
		s.True(result.Allowed)
	})
}

// =============================================================================
// Anonymization Tests
// =============================================================================

func (s *ClientLimitServiceSuite) TestAnonymizeClientID() {
	s.Run("short client_id gets partial masking", func() {
		result := anonymizeClientID("abc")
		s.Equal("a***", result)
	})

	s.Run("normal client_id shows prefix and suffix", func() {
		result := anonymizeClientID("my-client-id-12345")
		s.Equal("my-c***2345", result)
	})
}

// =============================================================================
// Configuration Tests
// =============================================================================

func (s *ClientLimitServiceSuite) TestWithConfig() {
	ctx := context.Background()

	customConfig := &config.ClientLimitConfig{
		ConfidentialLimit: config.Limit{
			RequestsPerWindow: 5,
			Window:            time.Minute,
		},
		PublicLimit: config.Limit{
			RequestsPerWindow: 2,
			Window:            time.Minute,
		},
	}

	svc, err := New(
		s.buckets,
		s.clientLookup,
		WithConfig(customConfig),
	)
	s.Require().NoError(err)

	s.clientLookup.clients["custom-public"] = false

	// Should use custom public limit (2)
	result, err := svc.Check(ctx, "custom-public", "/test")
	s.NoError(err)
	s.Equal(2, result.Limit)
}
