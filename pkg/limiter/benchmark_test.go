package limiter

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// BenchmarkAlgorithms compares the memory allocations and latency of the 3 algorithms.

func BenchmarkTokenBucket_Sequential(b *testing.B) {
	lim := NewTokenBucket(1000000, 1000000)
	ctx := context.Background()
	key := "client-seq"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = lim.Allow(ctx, key)
	}
}

func BenchmarkTokenBucket_Parallel(b *testing.B) {
	lim := NewTokenBucket(1000000, 1000000)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		id := 0
		for pb.Next() {
			id++
			key := fmt.Sprintf("client-%d", id%100)
			_, _ = lim.Allow(ctx, key)
		}
	})
}

func BenchmarkSlidingWindowCounter_Sequential(b *testing.B) {
	lim := NewSlidingWindowCounter(1000000, time.Minute)
	ctx := context.Background()
	key := "client-seq"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = lim.Allow(ctx, key)
	}
}

func BenchmarkSlidingWindowCounter_Parallel(b *testing.B) {
	lim := NewSlidingWindowCounter(1000000, time.Minute)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		id := 0
		for pb.Next() {
			id++
			key := fmt.Sprintf("client-%d", id%100)
			_, _ = lim.Allow(ctx, key)
		}
	})
}

func BenchmarkFixedWindow_Sequential(b *testing.B) {
	lim := NewFixedWindow(1000000, time.Minute)
	ctx := context.Background()
	key := "client-seq"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = lim.Allow(ctx, key)
	}
}

func BenchmarkFixedWindow_Parallel(b *testing.B) {
	lim := NewFixedWindow(1000000, time.Minute)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		id := 0
		for pb.Next() {
			id++
			key := fmt.Sprintf("client-%d", id%100)
			_, _ = lim.Allow(ctx, key)
		}
	})
}
