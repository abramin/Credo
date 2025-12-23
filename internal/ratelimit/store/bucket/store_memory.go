package bucket

import (
	"container/list"
	"context"
	"hash/fnv"
	"sync"
	"time"

	"credo/internal/ratelimit/models"
)

const (
	defaultShardCount   = 32
	defaultMaxBuckets   = 100000 // Max buckets per shard before LRU eviction
	circularBufferSize  = 256    // Fixed size for circular buffer (covers most limits)
)

// circularWindow is a fixed-size circular buffer for sliding window rate limiting.
// Provides O(1) operations and bounded memory.
type circularWindow struct {
	timestamps [circularBufferSize]int64 // Unix nano timestamps
	head       int                       // Next write position
	count      int                       // Current number of valid entries
	window     time.Duration
}

func (cw *circularWindow) tryConsume(cost, limit int, now time.Time) (allowed bool, remaining int, resetAt time.Time) {
	nowNano := now.UnixNano()
	cutoffNano := nowNano - cw.window.Nanoseconds()

	// Count valid (non-expired) entries
	validCount := 0
	oldestValid := nowNano
	for i := 0; i < cw.count; i++ {
		idx := (cw.head - cw.count + i + circularBufferSize) % circularBufferSize
		ts := cw.timestamps[idx]
		if ts > cutoffNano {
			validCount++
			if ts < oldestValid {
				oldestValid = ts
			}
		}
	}

	// Update count to reflect cleanup
	cw.count = validCount

	if validCount+cost > limit {
		resetAt = time.Unix(0, oldestValid).Add(cw.window)
		return false, 0, resetAt
	}

	// Record new timestamps
	for i := 0; i < cost; i++ {
		cw.timestamps[cw.head] = nowNano
		cw.head = (cw.head + 1) % circularBufferSize
		if cw.count < circularBufferSize {
			cw.count++
		}
	}

	remaining = limit - (validCount + cost)
	resetAt = now.Add(cw.window)
	return true, remaining, resetAt
}

func (cw *circularWindow) currentCount(now time.Time) int {
	cutoffNano := now.UnixNano() - cw.window.Nanoseconds()
	validCount := 0
	for i := 0; i < cw.count; i++ {
		idx := (cw.head - cw.count + i + circularBufferSize) % circularBufferSize
		if cw.timestamps[idx] > cutoffNano {
			validCount++
		}
	}
	return validCount
}

// lruEntry wraps a bucket with LRU tracking.
type lruEntry struct {
	key    string
	bucket *circularWindow
}

// shard is a partition of the bucket store with its own lock and LRU list.
type shard struct {
	mu       sync.RWMutex
	buckets  map[string]*list.Element
	lruList  *list.List
	maxSize  int
}

func newShard(maxSize int) *shard {
	return &shard{
		buckets: make(map[string]*list.Element),
		lruList: list.New(),
		maxSize: maxSize,
	}
}

func (s *shard) get(key string) (*circularWindow, bool) {
	elem, ok := s.buckets[key]
	if !ok {
		return nil, false
	}
	// Move to front (most recently used)
	s.lruList.MoveToFront(elem)
	return elem.Value.(*lruEntry).bucket, true
}

func (s *shard) set(key string, bucket *circularWindow) {
	if elem, ok := s.buckets[key]; ok {
		s.lruList.MoveToFront(elem)
		elem.Value.(*lruEntry).bucket = bucket
		return
	}

	// Evict if at capacity
	if s.lruList.Len() >= s.maxSize {
		oldest := s.lruList.Back()
		if oldest != nil {
			entry := oldest.Value.(*lruEntry)
			delete(s.buckets, entry.key)
			s.lruList.Remove(oldest)
		}
	}

	entry := &lruEntry{key: key, bucket: bucket}
	elem := s.lruList.PushFront(entry)
	s.buckets[key] = elem
}

func (s *shard) delete(key string) {
	if elem, ok := s.buckets[key]; ok {
		s.lruList.Remove(elem)
		delete(s.buckets, key)
	}
}

// InMemoryBucketStore implements a sharded, LRU-evicting rate limit store
// with circular buffer sliding windows for bounded memory and O(1) operations.
// For production at scale, use RedisStore instead.
type InMemoryBucketStore struct {
	shards     []*shard
	shardCount uint32
}

// Option configures the bucket store.
type Option func(*InMemoryBucketStore)

// WithShardCount sets the number of shards (default 32).
func WithShardCount(count int) Option {
	return func(s *InMemoryBucketStore) {
		if count > 0 {
			s.shardCount = uint32(count)
		}
	}
}

// WithMaxBucketsPerShard sets max buckets per shard before LRU eviction.
func WithMaxBucketsPerShard(max int) Option {
	return func(s *InMemoryBucketStore) {
		for _, sh := range s.shards {
			sh.maxSize = max
		}
	}
}

func New(opts ...Option) *InMemoryBucketStore {
	store := &InMemoryBucketStore{
		shardCount: defaultShardCount,
	}

	// Apply options that affect shard count first
	for _, opt := range opts {
		opt(store)
	}

	// Initialize shards
	store.shards = make([]*shard, store.shardCount)
	for i := range store.shards {
		store.shards[i] = newShard(defaultMaxBuckets)
	}

	// Apply remaining options
	for _, opt := range opts {
		opt(store)
	}

	return store
}

func (s *InMemoryBucketStore) getShard(key string) *shard {
	h := fnv.New32a()
	h.Write([]byte(key))
	return s.shards[h.Sum32()%s.shardCount]
}

func (s *InMemoryBucketStore) Allow(ctx context.Context, key string, limit int, window time.Duration) (*models.RateLimitResult, error) {
	return s.AllowN(ctx, key, 1, limit, window)
}

func (s *InMemoryBucketStore) AllowN(ctx context.Context, key string, cost, limit int, window time.Duration) (*models.RateLimitResult, error) {
	sh := s.getShard(key)

	sh.mu.Lock()
	defer sh.mu.Unlock()

	bucket, ok := sh.get(key)
	if !ok {
		bucket = &circularWindow{
			window: window,
		}
		sh.set(key, bucket)
	}

	allowed, remaining, resetAt := bucket.tryConsume(cost, limit, time.Now())

	return &models.RateLimitResult{
		Allowed:    allowed,
		Limit:      limit,
		Remaining:  remaining,
		ResetAt:    resetAt,
		RetryAfter: retryAfterSeconds(allowed, resetAt),
	}, nil
}

func (s *InMemoryBucketStore) Reset(ctx context.Context, key string) error {
	sh := s.getShard(key)

	sh.mu.Lock()
	defer sh.mu.Unlock()

	sh.delete(key)
	return nil
}

func (s *InMemoryBucketStore) GetCurrentCount(ctx context.Context, key string) (int, error) {
	sh := s.getShard(key)

	sh.mu.RLock()
	defer sh.mu.RUnlock()

	elem, ok := sh.buckets[key]
	if !ok {
		return 0, nil
	}

	return elem.Value.(*lruEntry).bucket.currentCount(time.Now()), nil
}

// Stats returns store statistics for monitoring.
func (s *InMemoryBucketStore) Stats() (totalBuckets int, bucketsPerShard []int) {
	bucketsPerShard = make([]int, s.shardCount)
	for i, sh := range s.shards {
		sh.mu.RLock()
		bucketsPerShard[i] = len(sh.buckets)
		totalBuckets += bucketsPerShard[i]
		sh.mu.RUnlock()
	}
	return
}

func retryAfterSeconds(allowed bool, resetAt time.Time) int {
	if allowed {
		return 0
	}
	seconds := int(time.Until(resetAt).Seconds())
	if seconds < 0 {
		return 0
	}
	return seconds
}
