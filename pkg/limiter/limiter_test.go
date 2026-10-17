package limiter

import (
	"context"
	"testing"
	"time"
)

func TestTokenBucket_RefillAndBurst(t *testing.T) {
	// Capacity 5, refill rate 2 tokens per second
	tb := NewTokenBucket(5, 2.0)
	ctx := context.Background()

	// 1. Consume all 5 tokens
	for i := 0; i < 5; i++ {
		res, err := tb.Allow(ctx, "user-1")
		if err != nil || !res.Allowed {
			t.Fatalf("request %d should be allowed, got: %+v, err: %v", i+1, res, err)
		}
	}

	// 2. 6th request should fail
	res, _ := tb.Allow(ctx, "user-1")
	if res.Allowed {
		t.Errorf("6th request should be blocked, but was allowed")
	}
	if res.RetryAfter <= 0 {
		t.Errorf("expected positive RetryAfter, got %v", res.RetryAfter)
	}

	// 3. Wait 550ms -> should refill ~1 token
	time.Sleep(550 * time.Millisecond)
	resAfter, _ := tb.Allow(ctx, "user-1")
	if !resAfter.Allowed {
		t.Errorf("request after refill should be allowed, got: %+v", resAfter)
	}
}

func TestSlidingWindowCounter_SmoothTransition(t *testing.T) {
	// 5 requests per 200ms window
	sw := NewSlidingWindowCounter(5, 200*time.Millisecond)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		res, _ := sw.Allow(ctx, "user-2")
		if !res.Allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	// 6th request fails
	res, _ := sw.Allow(ctx, "user-2")
	if res.Allowed {
		t.Errorf("expected 6th request to be denied")
	}
}

func TestFixedWindow_BoundaryReset(t *testing.T) {
	// Limit 3 per 100ms
	fw := NewFixedWindow(3, 100*time.Millisecond)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		res, _ := fw.Allow(ctx, "user-3")
		if !res.Allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	// Exceeded
	res, _ := fw.Allow(ctx, "user-3")
	if res.Allowed {
		t.Errorf("expected 4th request to be denied")
	}

	// Wait for window reset
	time.Sleep(110 * time.Millisecond)
	resReset, _ := fw.Allow(ctx, "user-3")
	if !resReset.Allowed {
		t.Errorf("expected request after window reset to be allowed")
	}
}
