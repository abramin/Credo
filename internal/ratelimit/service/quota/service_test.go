package quota

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"credo/internal/ratelimit/config"
	"credo/internal/ratelimit/models"
	quotaStore "credo/internal/ratelimit/store/quota"
	id "credo/pkg/domain"
)

// =============================================================================
// Quota Service Test Suite
// =============================================================================
// Justification for unit tests: The quota service contains tier-specific behavior,
// overage handling, and audit emission logic that is difficult to exercise precisely
// through E2E tests which require full API key provisioning flows.

type QuotaServiceSuite struct {
	suite.Suite
	store   *quotaStore.InMemoryQuotaStore
	service *Service
}

func TestQuotaServiceSuite(t *testing.T) {
	suite.Run(t, new(QuotaServiceSuite))
}

func (s *QuotaServiceSuite) SetupTest() {
	cfg := config.DefaultConfig()
	s.store = quotaStore.New(cfg)

	var err error
	s.service, err = New(s.store)
	s.Require().NoError(err)
}

// =============================================================================
// Constructor Tests (Invariant Enforcement)
// =============================================================================

func (s *QuotaServiceSuite) TestNew() {
	s.Run("nil store returns error", func() {
		_, err := New(nil)
		s.Error(err)
		s.Contains(err.Error(), "quota store is required")
	})

	s.Run("valid store returns configured service", func() {
		svc, err := New(s.store)
		s.NoError(err)
		s.NotNil(svc)
	})
}

// =============================================================================
// Check Tests
// =============================================================================

func (s *QuotaServiceSuite) TestCheck() {
	ctx := context.Background()

	s.Run("missing quota returns not found error", func() {
		apiKeyID := id.APIKeyID("missing-key")
		_, err := s.service.Check(ctx, apiKeyID)
		s.Error(err)
		s.Contains(err.Error(), "not found")
	})

	s.Run("existing quota returns quota record", func() {
		apiKeyID := id.APIKeyID("existing-key")
		// Create quota by incrementing
		_, err := s.store.IncrementUsage(ctx, apiKeyID, 1)
		s.Require().NoError(err)

		quota, err := s.service.Check(ctx, apiKeyID)
		s.NoError(err)
		s.Equal(apiKeyID, quota.APIKeyID)
		s.Equal(1, quota.CurrentUsage)
	})
}

// =============================================================================
// Increment Tests
// =============================================================================

func (s *QuotaServiceSuite) TestIncrement() {
	ctx := context.Background()

	s.Run("increments usage counter", func() {
		apiKeyID := id.APIKeyID("increment-key")

		quota, err := s.service.Increment(ctx, apiKeyID, 5)
		s.NoError(err)
		s.Equal(5, quota.CurrentUsage)

		quota, err = s.service.Increment(ctx, apiKeyID, 3)
		s.NoError(err)
		s.Equal(8, quota.CurrentUsage)
	})

	s.Run("new key starts with free tier defaults", func() {
		apiKeyID := id.APIKeyID("free-tier-key")

		quota, err := s.service.Increment(ctx, apiKeyID, 1)
		s.NoError(err)
		s.Equal(models.QuotaTierFree, quota.Tier)
		s.Equal(1000, quota.MonthlyLimit) // Free tier default
		s.False(quota.OverageAllowed)
	})
}

// =============================================================================
// Reset Tests
// =============================================================================

func (s *QuotaServiceSuite) TestReset() {
	ctx := context.Background()

	s.Run("empty api_key_id returns bad request", func() {
		err := s.service.Reset(ctx, id.APIKeyID(""))
		s.Error(err)
		s.Contains(err.Error(), "api_key_id is required")
	})

	s.Run("clears usage counter", func() {
		apiKeyID := id.APIKeyID("reset-key")
		_, _ = s.service.Increment(ctx, apiKeyID, 100)

		err := s.service.Reset(ctx, apiKeyID)
		s.NoError(err)

		quota, err := s.service.Check(ctx, apiKeyID)
		s.NoError(err)
		s.Equal(0, quota.CurrentUsage)
	})
}

// =============================================================================
// UpdateTier Tests
// =============================================================================

func (s *QuotaServiceSuite) TestUpdateTier() {
	ctx := context.Background()

	s.Run("empty api_key_id returns bad request", func() {
		err := s.service.UpdateTier(ctx, id.APIKeyID(""), models.QuotaTierStarter)
		s.Error(err)
		s.Contains(err.Error(), "api_key_id is required")
	})

	s.Run("invalid tier returns bad request", func() {
		apiKeyID := id.APIKeyID("invalid-tier-key")
		err := s.service.UpdateTier(ctx, apiKeyID, models.QuotaTier("invalid"))
		s.Error(err)
		s.Contains(err.Error(), "invalid tier")
	})

	s.Run("updates tier and limits", func() {
		apiKeyID := id.APIKeyID("update-tier-key")
		// Create initial quota
		_, _ = s.service.Increment(ctx, apiKeyID, 1)

		err := s.service.UpdateTier(ctx, apiKeyID, models.QuotaTierStarter)
		s.NoError(err)

		quota, err := s.service.Check(ctx, apiKeyID)
		s.NoError(err)
		s.Equal(models.QuotaTierStarter, quota.Tier)
		s.Equal(10000, quota.MonthlyLimit)
		s.True(quota.OverageAllowed)
	})

	s.Run("preserves existing usage on tier change", func() {
		apiKeyID := id.APIKeyID("preserve-usage-key")
		_, _ = s.service.Increment(ctx, apiKeyID, 500)

		err := s.service.UpdateTier(ctx, apiKeyID, models.QuotaTierBusiness)
		s.NoError(err)

		quota, err := s.service.Check(ctx, apiKeyID)
		s.NoError(err)
		s.Equal(500, quota.CurrentUsage) // Usage preserved
		s.Equal(100000, quota.MonthlyLimit)
	})
}

// =============================================================================
// Tier Boundary Tests
// =============================================================================
// Justification: Tier-specific behavior verification

func (s *QuotaServiceSuite) TestTierBoundaries() {
	ctx := context.Background()

	s.Run("free tier blocks at limit without overage", func() {
		apiKeyID := id.APIKeyID("free-at-limit-key")
		// Increment to limit
		_, _ = s.service.Increment(ctx, apiKeyID, 1000)

		quota, err := s.service.Check(ctx, apiKeyID)
		s.NoError(err)
		s.True(quota.IsOverQuota())
		s.False(quota.OverageAllowed)
	})

	s.Run("starter tier allows overage", func() {
		apiKeyID := id.APIKeyID("starter-overage-key")
		_ = s.store.UpdateTier(ctx, apiKeyID, models.QuotaTierStarter)
		_, _ = s.service.Increment(ctx, apiKeyID, 10001)

		quota, err := s.service.Check(ctx, apiKeyID)
		s.NoError(err)
		s.True(quota.IsOverQuota())
		s.True(quota.OverageAllowed) // Can continue with overage billing
	})

	s.Run("enterprise tier has unlimited quota with overage", func() {
		apiKeyID := id.APIKeyID("enterprise-key")
		_ = s.store.UpdateTier(ctx, apiKeyID, models.QuotaTierEnterprise)
		_, _ = s.service.Increment(ctx, apiKeyID, 1)

		quota, err := s.service.Check(ctx, apiKeyID)
		s.NoError(err)
		s.Equal(-1, quota.MonthlyLimit) // -1 = unlimited
		s.True(quota.OverageAllowed)
	})
}

// =============================================================================
// List Tests
// =============================================================================

func (s *QuotaServiceSuite) TestList() {
	ctx := context.Background()

	s.Run("returns all quota records", func() {
		// Create multiple quotas
		key1 := id.APIKeyID("list-key-1")
		key2 := id.APIKeyID("list-key-2")
		_, _ = s.service.Increment(ctx, key1, 10)
		_, _ = s.service.Increment(ctx, key2, 20)

		quotas, err := s.service.List(ctx)
		s.NoError(err)
		s.GreaterOrEqual(len(quotas), 2)
	})
}
