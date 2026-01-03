package admin

//go:generate mockgen -source=admin.go -destination=mocks/mocks.go -package=mocks AllowlistStore,BucketStore

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"credo/internal/ratelimit/admin/mocks"
	"credo/internal/ratelimit/models"
	"credo/internal/ratelimit/observability"
	"credo/pkg/platform/audit/publishers/security"
	auditmemory "credo/pkg/platform/audit/store/memory"
)

// =============================================================================
// Admin Service Test Suite
// =============================================================================
// Justification for unit tests: The admin service manages allowlist entries
// and rate limit resets. Tests verify constructor invariants, input validation,
// error propagation, and audit event emission.

type AdminServiceSuite struct {
	suite.Suite
	ctrl           *gomock.Controller
	mockAllowlist  *mocks.MockAllowlistStore
	mockBuckets    *mocks.MockBucketStore
	auditPublisher observability.AuditPublisher
	service        *Service
}

func TestAdminServiceSuite(t *testing.T) {
	suite.Run(t, new(AdminServiceSuite))
}

func (s *AdminServiceSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockAllowlist = mocks.NewMockAllowlistStore(s.ctrl)
	s.mockBuckets = mocks.NewMockBucketStore(s.ctrl)
	s.auditPublisher = security.New(auditmemory.NewInMemoryStore())
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	s.service, _ = New(
		s.mockAllowlist,
		s.mockBuckets,
		WithLogger(logger),
		WithAuditPublisher(s.auditPublisher),
	)
}

func (s *AdminServiceSuite) TearDownTest() {
	s.ctrl.Finish()
}

// =============================================================================
// Constructor Tests (Invariant Enforcement)
// =============================================================================
// Justification: Constructor invariants prevent invalid service creation.
// Integration tests cannot easily verify nil-guard behaviors.

func (s *AdminServiceSuite) TestNew() {
	s.Run("nil allowlist store returns error", func() {
		_, err := New(nil, s.mockBuckets)
		s.Error(err)
		s.Contains(err.Error(), "allowlist store is required")
	})

	s.Run("nil buckets store returns error", func() {
		_, err := New(s.mockAllowlist, nil)
		s.Error(err)
		s.Contains(err.Error(), "buckets store is required")
	})

	s.Run("valid stores returns configured service", func() {
		svc, err := New(s.mockAllowlist, s.mockBuckets)
		s.NoError(err)
		s.NotNil(svc)
	})

	s.Run("with options applies options", func() {
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		svc, err := New(
			s.mockAllowlist,
			s.mockBuckets,
			WithLogger(logger),
			WithAuditPublisher(s.auditPublisher),
		)
		s.NoError(err)
		s.Equal(logger, svc.logger)
		s.Equal(s.auditPublisher, svc.auditPublisher)
	})
}

// =============================================================================
// ResetRateLimit Normalization Tests (Security)
// =============================================================================
// Security test: Verify normalization is applied before validation to handle
// mixed case/whitespace input consistently.

func (s *AdminServiceSuite) TestResetRateLimitNormalization() {
	ctx := context.Background()

	s.Run("mixed case type is normalized before validation", func() {
		// Input with mixed case - should be normalized to lowercase
		req := &models.ResetRateLimitRequest{
			Type:       "  IP  ",            // Mixed case with whitespace
			Identifier: "  192.168.1.100  ", // Whitespace around
			Class:      "  AUTH  ",          // Mixed case with whitespace
		}

		// Mock should receive the sanitized key (after normalization)
		s.mockBuckets.EXPECT().
			Reset(ctx, "ip:192.168.1.100:auth").
			Return(nil)

		err := s.service.ResetRateLimit(ctx, req)
		s.NoError(err)
	})

	s.Run("user_id type with whitespace is normalized", func() {
		req := &models.ResetRateLimitRequest{
			Type:       "  USER_ID  ",
			Identifier: "  user-123  ",
		}

		s.mockBuckets.EXPECT().
			Reset(ctx, "user:user-123:auth").
			Return(nil)
		s.mockBuckets.EXPECT().
			Reset(ctx, "user:user-123:sensitive").
			Return(nil)
		s.mockBuckets.EXPECT().
			Reset(ctx, "user:user-123:read").
			Return(nil)
		s.mockBuckets.EXPECT().
			Reset(ctx, "user:user-123:write").
			Return(nil)

		err := s.service.ResetRateLimit(ctx, req)
		s.NoError(err)
	})

	s.Run("invalid type after normalization returns error", func() {
		req := &models.ResetRateLimitRequest{
			Type:       "  INVALID  ",
			Identifier: "192.168.1.100",
		}

		err := s.service.ResetRateLimit(ctx, req)
		s.Error(err)
		s.Contains(err.Error(), "type must be")
	})
}
