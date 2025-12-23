package globalthrottle

import (
	"context"
	"testing"
)

// BenchmarkIncrementGlobal measures single-threaded throughput
func BenchmarkIncrementGlobal(b *testing.B) {
	store := New(WithPerSecondLimit(1000000), WithPerHourLimit(100000000))
	ctx := context.Background()

	for b.Loop() {
		_, _, _ = store.IncrementGlobal(ctx)
	}
}

// BenchmarkIncrementGlobal_Parallel measures concurrent throughput with atomics
func BenchmarkIncrementGlobal_Parallel(b *testing.B) {
	store := New(WithPerSecondLimit(1000000), WithPerHourLimit(100000000))
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, _ = store.IncrementGlobal(ctx)
		}
	})
}

// BenchmarkGetGlobalCount measures read performance
func BenchmarkGetGlobalCount(b *testing.B) {
	store := New()
	ctx := context.Background()

	// Warm up
	for range 100 {
		_, _, _ = store.IncrementGlobal(ctx)
	}

	for b.Loop() {
		_, _ = store.GetGlobalCount(ctx)
	}
}

// BenchmarkGetGlobalCount_Parallel measures concurrent read performance
func BenchmarkGetGlobalCount_Parallel(b *testing.B) {
	store := New()
	ctx := context.Background()

	// Warm up
	for range 100 {
		_, _, _ = store.IncrementGlobal(ctx)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = store.GetGlobalCount(ctx)
		}
	})
}
