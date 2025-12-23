package quota

import (
	"context"
	"credo/internal/ratelimit/config"
	"credo/pkg/domain"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: IncrementUsage behavior is covered by E2E FR-5 quota scenarios.
// Only edge cases and invariants not covered by E2E are tested here.

func TestInMemoryQuotaStore(t *testing.T) {
	store := New(config.DefaultConfig())
	ctx := context.Background()

	t.Run("GetQuota for missing key returns nil", func(t *testing.T) {
		res, err := store.GetQuota(ctx, domain.APIKeyID("missing"))
		require.NoError(t, err)
		assert.Nil(t, res)
	})

	t.Run("GetQuota does not mutate quota", func(t *testing.T) {
		// Setup: create a quota via increment
		res, err := store.IncrementUsage(ctx, domain.APIKeyID("immutability-test"), 5)
		require.NoError(t, err)
		assert.NotNil(t, res)

		got, err := store.GetQuota(ctx, domain.APIKeyID("immutability-test"))
		require.NoError(t, err)

		// Verify GetQuota doesn't mutate the quota
		usageBefore := got.CurrentUsage
		periodStartBefore := got.PeriodStart
		periodEndBefore := got.PeriodEnd

		got2, err := store.GetQuota(ctx, domain.APIKeyID("immutability-test"))
		require.NoError(t, err)
		assert.Equal(t, usageBefore, got2.CurrentUsage, "GetQuota should not mutate usage")
		assert.Equal(t, periodStartBefore, got2.PeriodStart, "GetQuota should not mutate period start")
		assert.Equal(t, periodEndBefore, got2.PeriodEnd, "GetQuota should not mutate period end")
	})
}

func TestInMemoryQuotaStore_Concurrent(t *testing.T) {
	store := New(config.DefaultConfig())
	ctx := context.Background()
	apiKeyID := domain.APIKeyID("concurrent-test")

	const goroutines = 100
	const incrementsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				_, err := store.IncrementUsage(ctx, apiKeyID, 1)
				assert.NoError(t, err)
			}
		}()
	}

	wg.Wait()

	quota, err := store.GetQuota(ctx, apiKeyID)
	require.NoError(t, err)
	assert.Equal(t, goroutines*incrementsPerGoroutine, quota.CurrentUsage,
		"concurrent increments should result in exact total")
}
