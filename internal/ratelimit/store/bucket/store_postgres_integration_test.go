//go:build integration

package bucket_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"credo/internal/ratelimit/store/bucket"
	"credo/pkg/testutil/containers"
)

type PostgresStoreSuite struct {
	suite.Suite
	postgres *containers.PostgresContainer
	store    *bucket.PostgresBucketStore
}

func TestPostgresStoreSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite.Run(t, new(PostgresStoreSuite))
}

func (s *PostgresStoreSuite) SetupSuite() {
	mgr := containers.GetManager()
	s.postgres = mgr.GetPostgres(s.T())
	s.store = bucket.NewPostgres(s.postgres.DB)
}

func (s *PostgresStoreSuite) SetupTest() {
	ctx := context.Background()
	err := s.postgres.TruncateTables(ctx, "rate_limit_events")
	s.Require().NoError(err)
}

// TestConcurrentAllowNRequests verifies that concurrent AllowN requests
// correctly enforce the rate limit (sum of allowed <= limit).
func (s *PostgresStoreSuite) TestConcurrentAllowNRequests() {
	ctx := context.Background()
	key := "concurrent-test"
	limit := 10
	window := 1 * time.Minute
	const goroutines = 50

	var wg sync.WaitGroup
	var allowedCount atomic.Int32
	var deniedCount atomic.Int32

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			result, err := s.store.Allow(ctx, key, limit, window)
			s.Require().NoError(err)

			if result.Allowed {
				allowedCount.Add(1)
			} else {
				deniedCount.Add(1)
			}
		}()
	}

	wg.Wait()

	// Exactly 'limit' requests should be allowed
	s.Equal(int32(limit), allowedCount.Load(), "exactly %d requests should be allowed", limit)
	s.Equal(int32(goroutines-limit), deniedCount.Load(), "remaining requests should be denied")

	// Verify current count
	count, err := s.store.GetCurrentCount(ctx, key)
	s.Require().NoError(err)
	s.Equal(limit, count)
}

// TestAdvisoryLockContention verifies correct blocking behavior under high contention.
func (s *PostgresStoreSuite) TestAdvisoryLockContention() {
	ctx := context.Background()
	key := "contention-test"
	limit := 100
	window := 1 * time.Minute
	const goroutines = 200

	var wg sync.WaitGroup
	var errors atomic.Int32

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, err := s.store.Allow(ctx, key, limit, window)
			if err != nil {
				errors.Add(1)
			}
		}()
	}

	wg.Wait()

	// No errors should occur (advisory locks should serialize properly)
	s.Equal(int32(0), errors.Load(), "no errors expected under contention")

	// Verify count is exactly 'limit' (100 allowed, rest denied)
	count, err := s.store.GetCurrentCount(ctx, key)
	s.Require().NoError(err)
	s.Equal(limit, count)
}

// TestWindowCleanupUnderLoad verifies expired events are cleaned correctly.
func (s *PostgresStoreSuite) TestWindowCleanupUnderLoad() {
	ctx := context.Background()
	key := "cleanup-test"
	limit := 100
	window := 1 * time.Second

	// First: fill up to limit
	for i := 0; i < limit; i++ {
		_, err := s.store.Allow(ctx, key, limit, window)
		s.Require().NoError(err)
	}

	// Verify at limit
	result, err := s.store.Allow(ctx, key, limit, window)
	s.Require().NoError(err)
	s.False(result.Allowed, "should be at limit")

	// Wait for window to expire
	time.Sleep(1500 * time.Millisecond)

	// Now requests should be allowed again (old events cleaned up)
	result, err = s.store.Allow(ctx, key, limit, window)
	s.Require().NoError(err)
	s.True(result.Allowed, "should be allowed after window expires")
}

// TestMultipleKeysConcurrently verifies independent rate limiting per key.
func (s *PostgresStoreSuite) TestMultipleKeysConcurrently() {
	ctx := context.Background()
	limit := 5
	window := 1 * time.Minute
	const keys = 10
	const requestsPerKey = 20

	var wg sync.WaitGroup
	allowedPerKey := make([]atomic.Int32, keys)

	for k := 0; k < keys; k++ {
		for r := 0; r < requestsPerKey; r++ {
			wg.Add(1)
			go func(keyIdx int) {
				defer wg.Done()

				key := "key-" + string(rune('A'+keyIdx))
				result, err := s.store.Allow(ctx, key, limit, window)
				s.Require().NoError(err)

				if result.Allowed {
					allowedPerKey[keyIdx].Add(1)
				}
			}(k)
		}
	}

	wg.Wait()

	// Each key should have exactly 'limit' allowed
	for k := 0; k < keys; k++ {
		s.Equal(int32(limit), allowedPerKey[k].Load(),
			"key %d should have %d allowed", k, limit)
	}
}

// TestAllowNCost verifies correct cost accounting.
func (s *PostgresStoreSuite) TestAllowNCost() {
	ctx := context.Background()
	key := "cost-test"
	limit := 10
	window := 1 * time.Minute

	// Cost 3 -> should allow (3 <= 10)
	result, err := s.store.AllowN(ctx, key, 3, limit, window)
	s.Require().NoError(err)
	s.True(result.Allowed)
	s.Equal(7, result.Remaining)

	// Cost 5 -> should allow (3+5=8 <= 10)
	result, err = s.store.AllowN(ctx, key, 5, limit, window)
	s.Require().NoError(err)
	s.True(result.Allowed)
	s.Equal(2, result.Remaining)

	// Cost 3 -> should deny (8+3=11 > 10)
	result, err = s.store.AllowN(ctx, key, 3, limit, window)
	s.Require().NoError(err)
	s.False(result.Allowed)
}

// TestReset verifies reset clears the rate limit.
func (s *PostgresStoreSuite) TestReset() {
	ctx := context.Background()
	key := "reset-test"
	limit := 5
	window := 1 * time.Minute

	// Fill up
	for i := 0; i < limit; i++ {
		_, err := s.store.Allow(ctx, key, limit, window)
		s.Require().NoError(err)
	}

	// Should be at limit
	result, err := s.store.Allow(ctx, key, limit, window)
	s.Require().NoError(err)
	s.False(result.Allowed)

	// Reset
	err = s.store.Reset(ctx, key)
	s.Require().NoError(err)

	// Should be allowed again
	result, err = s.store.Allow(ctx, key, limit, window)
	s.Require().NoError(err)
	s.True(result.Allowed)
}
