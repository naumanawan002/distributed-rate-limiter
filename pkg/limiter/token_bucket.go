package limiter

import (
	"context"
	"math"
	"sync"
	"time"
)

type bucketState struct {
	tokens     float64
	lastRefill time.Time
}

// TokenBucket implements the Token Bucket rate limiting algorithm.
type TokenBucket struct {
	mu         sync.RWMutex
	capacity   float64
	refillRate float64 // tokens per second
	buckets    map[string]*bucketState
}

// NewTokenBucket creates a token bucket limiter.
func NewTokenBucket(capacity int64, refillRatePerSec float64) *TokenBucket {
	return &TokenBucket{
		capacity:   float64(capacity),
		refillRate: refillRatePerSec,
		buckets:    make(map[string]*bucketState),
	}
}

func (tb *TokenBucket) Algorithm() string {
	return "token_bucket"
}

func (tb *TokenBucket) Allow(ctx context.Context, key string) (Result, error) {
	return tb.AllowN(ctx, key, 1)
}

func (tb *TokenBucket) AllowN(ctx context.Context, key string, n int64) (Result, error) {
	if key == "" {
		return Result{}, ErrKeyRequired
	}

	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	b, exists := tb.buckets[key]
	if !exists {
		b = &bucketState{
			tokens:     tb.capacity,
			lastRefill: now,
		}
		tb.buckets[key] = b
	}

	// Refill tokens based on elapsed duration
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens = math.Min(tb.capacity, b.tokens+(elapsed*tb.refillRate))
	b.lastRefill = now

	needed := float64(n)
	if b.tokens >= needed {
		b.tokens -= needed
		remaining := int64(math.Floor(b.tokens))
		return Result{
			Allowed:    true,
			Limit:      int64(tb.capacity),
			Remaining:  remaining,
			ResetAfter: time.Duration((tb.capacity-b.tokens)/tb.refillRate) * time.Second,
			Algorithm:  tb.Algorithm(),
		}, nil
	}

	// Rate limit exceeded
	deficit := needed - b.tokens
	retryAfter := time.Duration(math.Ceil(deficit/tb.refillRate)) * time.Second

	return Result{
		Allowed:    false,
		Limit:      int64(tb.capacity),
		Remaining:  0,
		ResetAfter: time.Duration(tb.capacity/tb.refillRate) * time.Second,
		RetryAfter: retryAfter,
		Algorithm:  tb.Algorithm(),
	}, nil
}

func (tb *TokenBucket) Reset(key string) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	delete(tb.buckets, key)
}
