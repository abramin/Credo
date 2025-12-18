package bucket

import (
	"context"
	"credo/internal/ratelimit/models"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	testLimit  = 10
	testWindow = time.Minute
)

type InMemoryBucketStoreSuite struct {
	suite.Suite
	store *InMemoryBucketStore
	ctx   context.Context
}

func TestInMemoryBucketStoreSuite(t *testing.T) {
	suite.Run(t, new(InMemoryBucketStoreSuite))
}

func (s *InMemoryBucketStoreSuite) SetupTest() {
	s.store = NewInMemoryBucketStore()
	s.ctx = context.Background()
}

// Per PRD-017 FR-1, FR-3: Sliding window rate limiting.
func (s *InMemoryBucketStoreSuite) TestAllow() {
	s.Run("first request allowed", func() {
		result, err := s.store.Allow(s.ctx, "test:key:allow:first", testLimit, testWindow)
		s.Require().NoError(err)
		s.True(result.Allowed)
		s.Equal(testLimit, result.Limit)
		s.Equal(testLimit-1, result.Remaining)
	})

	s.Run("requests up to limit allowed", func() {
		var result *models.RateLimitResult
		var err error
		for range testLimit {
			result, err = s.store.Allow(s.ctx, "test:key:allow:limit", testLimit, testWindow)
		}
		s.Require().NoError(err)
		s.True(result.Allowed)
		s.Equal(testLimit, result.Limit)
		s.Equal(0, result.Remaining)
	})

	s.Run("request over limit denied", func() {
		for range testLimit {
			_, err := s.store.Allow(s.ctx, "test:key:allow:over", testLimit, testWindow)
			require.NoError(s.T(), err)
		}
		result, err := s.store.Allow(s.ctx, "test:key:allow:over", testLimit, testWindow)
		s.Require().NoError(err)
		s.False(result.Allowed)
		s.Equal(0, result.Remaining)
	})

	s.Run("after window expires requests allowed", func() {
		_, err := s.store.Allow(s.ctx, "test:key:allow:reset", testLimit, testWindow)
		s.Require().NoError(err)

		s.store.mu.Lock()
		if sw, exists := s.store.buckets["test:key:allow:reset"]; exists {
			sw.timestamps = []time.Time{}
		}
		s.store.mu.Unlock()

		result, err := s.store.Allow(s.ctx, "test:key:allow:reset", testLimit, testWindow)
		s.Require().NoError(err)
		s.True(result.Allowed)
		s.Equal(testLimit, result.Limit)
		s.Equal(testLimit-1, result.Remaining)
	})
}

// Per PRD-017 TR-1: AllowN for operations consuming multiple tokens.
func (s *InMemoryBucketStoreSuite) TestAllowN() {
	s.Run("cost of 1 behaves like Allow", func() {
		result, err := s.store.AllowN(s.ctx, "test:key:allown:one", 1, testLimit, testWindow)
		s.Require().NoError(err)
		s.True(result.Allowed)
		s.Equal(testLimit, result.Limit)
		s.Equal(testLimit-1, result.Remaining)
	})

	s.Run("cost of 5 consumes 5 tokens", func() {
		result, err := s.store.AllowN(s.ctx, "test:key:allown:five", 5, testLimit, testWindow)
		s.Require().NoError(err)
		s.True(result.Allowed)
		s.Equal(testLimit, result.Limit)
		s.Equal(5, result.Remaining)
	})

	s.Run("cost greater than remaining denied", func() {
		firstResult, err := s.store.AllowN(s.ctx, "test:key:allown:deny", 7, testLimit, testWindow)
		s.Require().NoError(err)
		s.Require().True(firstResult.Allowed)

		result, err := s.store.AllowN(s.ctx, "test:key:allown:deny", 4, testLimit, testWindow)
		s.Require().NoError(err)
		s.False(result.Allowed)
		s.Equal(0, result.Remaining)
	})
}

// Per PRD-017 TR-1: Admin reset operation.
func (s *InMemoryBucketStoreSuite) TestReset() {
	_, err := s.store.AllowN(s.ctx, "test:key:reset", 5, testLimit, testWindow)
	s.Require().NoError(err)

	err = s.store.Reset(s.ctx, "test:key:reset")
	s.Require().NoError(err)

	result, err := s.store.AllowN(s.ctx, "test:key:reset", testLimit, testLimit, testWindow)
	s.Require().NoError(err)
	s.True(result.Allowed)
	s.Equal(testLimit, result.Limit)
	s.Equal(0, result.Remaining)
}

func (s *InMemoryBucketStoreSuite) TestConcurrent() {
	limit := 100 // Different from testLimit for concurrency testing
	key := "test:key:concurrent"
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowedCount := 0

	for range 200 {
		wg.Go(func() {
			result, err := s.store.Allow(s.ctx, key, limit, testWindow)
			s.Require().NoError(err)
			if result.Allowed {
				mu.Lock()
				allowedCount++
				mu.Unlock()
			}
		})
	}

	wg.Wait()
	s.Equal(limit, allowedCount)
}
