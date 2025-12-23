package bucket

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// BenchmarkAllowN measures single-threaded throughput
func BenchmarkAllowN(b *testing.B) {
	store := New()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.AllowN(ctx, "bench-key", 1, 1000, time.Minute)
	}
}

// BenchmarkAllowN_Parallel measures concurrent throughput
func BenchmarkAllowN_Parallel(b *testing.B) {
	store := New()
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = store.AllowN(ctx, "bench-key", 1, 1000, time.Minute)
		}
	})
}

// BenchmarkAllowN_HighCardinality measures performance with many unique keys
func BenchmarkAllowN_HighCardinality(b *testing.B) {
	store := New()
	ctx := context.Background()

	for i := 0; b.Loop(); i++ {
		key := fmt.Sprintf("ip:10.0.%d.%d", (i/256)%256, i%256)
		_, _ = store.AllowN(ctx, key, 1, 100, time.Minute)
	}
}

// BenchmarkAllowN_HighCardinality_Parallel measures concurrent high-cardinality performance
func BenchmarkAllowN_HighCardinality_Parallel(b *testing.B) {
	store := New()
	ctx := context.Background()
	var counter int64
	var mu sync.Mutex

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			counter++
			i := counter
			mu.Unlock()
			key := fmt.Sprintf("ip:10.0.%d.%d", (i/256)%256, i%256)
			_, _ = store.AllowN(ctx, key, 1, 100, time.Minute)
		}
	})
}

// BenchmarkShardDistribution verifies even shard distribution
func BenchmarkShardDistribution(b *testing.B) {
	store := New()
	ctx := context.Background()

	// Warm up with many keys
	for i := range 10000 {
		key := fmt.Sprintf("user:%d", i)
		_, _ = store.AllowN(ctx, key, 1, 100, time.Minute)
	}

	total, perShard := store.Stats()
	b.Logf("Total buckets: %d", total)

	// Check distribution
	var min, max int
	for i, count := range perShard {
		if i == 0 || count < min {
			min = count
		}
		if count > max {
			max = count
		}
	}
	b.Logf("Shard distribution: min=%d, max=%d, spread=%.2f%%",
		min, max, float64(max-min)/float64(total)*100)
}

// BenchmarkLRUEviction measures eviction overhead
func BenchmarkLRUEviction(b *testing.B) {
	// Create store with small max size to force evictions
	store := New(WithMaxBucketsPerShard(100))
	ctx := context.Background()

	for i := 0; b.Loop(); i++ {
		// Use many unique keys to trigger evictions
		key := fmt.Sprintf("evict:%d", i)
		_, _ = store.AllowN(ctx, key, 1, 100, time.Minute)
	}
}
