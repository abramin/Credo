package admin

//go:generate mockgen -source=admin.go -destination=mocks/mocks.go -package=mocks AllowlistStore,BucketStore,AuditPublisher

import (
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"credo/internal/ratelimit/admin/mocks"
)

// =============================================================================
// Admin Service Test Suite
// =============================================================================
// Justification for unit tests: The admin service manages allowlist entries
// and rate limit resets. Tests verify constructor invariants, input validation,
// error propagation, and audit event emission.

type AdminServiceSuite struct {
	suite.Suite
	ctrl               *gomock.Controller
	mockAllowlist      *mocks.MockAllowlistStore
	mockBuckets        *mocks.MockBucketStore
	mockAuditPublisher *mocks.MockAuditPublisher
	service            *Service
}

func TestAdminServiceSuite(t *testing.T) {
	suite.Run(t, new(AdminServiceSuite))
}

func (s *AdminServiceSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockAllowlist = mocks.NewMockAllowlistStore(s.ctrl)
	s.mockBuckets = mocks.NewMockBucketStore(s.ctrl)
	s.mockAuditPublisher = mocks.NewMockAuditPublisher(s.ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	s.service, _ = New(
		s.mockAllowlist,
		s.mockBuckets,
		WithLogger(logger),
		WithAuditPublisher(s.mockAuditPublisher),
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
			WithAuditPublisher(s.mockAuditPublisher),
		)
		s.NoError(err)
		s.Equal(logger, svc.logger)
		s.Equal(s.mockAuditPublisher, svc.auditPublisher)
	})
}
