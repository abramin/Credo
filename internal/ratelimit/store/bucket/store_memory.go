package bucket

import (
	"context"
	"sync"
	"time"

	"credo/internal/ratelimit/models"
)

// InMemoryBucketStore implements BucketStore using in-memory sliding window.
// Per PRD-017 TR-1: In-memory implementation (MVP, not distributed).
// For production, use RedisStore instead.
type InMemoryBucketStore struct {
	mu      sync.RWMutex
	buckets map[string]*slidingWindow
}

// slidingWindow tracks request timestamps for sliding window rate limiting.
// Per PRD-017 FR-3: Sliding window algorithm prevents boundary attacks.
type slidingWindow struct {
	timestamps []time.Time
	window     time.Duration
}

// NewInMemoryBucketStore creates a new in-memory bucket store.
func NewInMemoryBucketStore() *InMemoryBucketStore {
	return &InMemoryBucketStore{
		buckets: make(map[string]*slidingWindow),
	}
}

// Allow checks if a request is allowed and increments the counter.
func (s *InMemoryBucketStore) Allow(ctx context.Context, key string, limit int, window time.Duration) (*models.RateLimitResult, error) {
	return s.AllowN(ctx, key, 1, limit, window)
}

// AllowN checks if a request with custom cost is allowed.
// Similar to Allow but adds 'cost' number of timestamps instead of 1
func (s *InMemoryBucketStore) AllowN(ctx context.Context, key string, cost int, limit int, window time.Duration) (*models.RateLimitResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cw := s.getOrCreateBucket(key, window)
	cw.cleanup(time.Now())
	count := len(cw.timestamps)

	if count+cost <= limit {
		now := time.Now()
		for range cost {
			cw.timestamps = append(cw.timestamps, now)
		}

		var resetAt time.Time
		if len(cw.timestamps) > 0 {
			resetAt = cw.timestamps[0].Add(window)
		} else {
			resetAt = now.Add(window)
		}

		return &models.RateLimitResult{
			Allowed:   true,
			Remaining: limit - len(cw.timestamps),
			ResetAt:   resetAt,
			Limit:     limit,
		}, nil
	}

	return &models.RateLimitResult{
		Allowed:   false,
		Remaining: 0,
		ResetAt:   time.Now().Add(window),
	}, nil
}

// Reset clears the rate limit counter for a key.
func (s *InMemoryBucketStore) Reset(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.buckets, key)
	return nil
}

// GetCurrentCount returns the current request count for a key.
func (s *InMemoryBucketStore) GetCurrentCount(ctx context.Context, key string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cw := s.buckets[key]
	if cw == nil {
		return 0, nil
	}

	cw.cleanup(time.Now())
	return len(cw.timestamps), nil
}

// cleanup removes expired timestamps from a sliding window.
func (sw *slidingWindow) cleanup(now time.Time) {
	cutoff := now.Add(-sw.window)
	i := 0
	for ; i < len(sw.timestamps); i++ {
		if sw.timestamps[i].After(cutoff) {
			break
		}
	}
	sw.timestamps = sw.timestamps[i:]
}

// getOrCreateBucket returns an existing bucket or creates a new one.
// Must be called while holding s.mu lock.
func (s *InMemoryBucketStore) getOrCreateBucket(key string, window time.Duration) *slidingWindow {
	if cw := s.buckets[key]; cw != nil {
		return cw
	}
	cw := &slidingWindow{timestamps: []time.Time{}, window: window}
	s.buckets[key] = cw
	return cw
}
