package limiter

import (
	"context"
	"math"
	"sync"
	"time"
)

type slidingWindowState struct {
	currWindowStart time.Time
	prevCount       int64
	currCount       int64
}

// SlidingWindowCounter implements memory-efficient sliding window rate limiting.
type SlidingWindowCounter struct {
	mu          sync.RWMutex
	limit       int64
	windowSize  time.Duration
	windows     map[string]*slidingWindowState
}

// NewSlidingWindowCounter creates a sliding window counter limiter.
func NewSlidingWindowCounter(limit int64, windowSize time.Duration) *SlidingWindowCounter {
	return &SlidingWindowCounter{
		limit:      limit,
		windowSize: windowSize,
		windows:    make(map[string]*slidingWindowState),
	}
}

func (sw *SlidingWindowCounter) Algorithm() string {
	return "sliding_window_counter"
}

func (sw *SlidingWindowCounter) Allow(ctx context.Context, key string) (Result, error) {
	return sw.AllowN(ctx, key, 1)
}

func (sw *SlidingWindowCounter) AllowN(ctx context.Context, key string, n int64) (Result, error) {
	if key == "" {
		return Result{}, ErrKeyRequired
	}

	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	w, exists := sw.windows[key]
	if !exists {
		w = &slidingWindowState{
			currWindowStart: now.Truncate(sw.windowSize),
			prevCount:       0,
			currCount:       0,
		}
		sw.windows[key] = w
	}

	currentStart := now.Truncate(sw.windowSize)
	elapsedSinceCurrent := now.Sub(currentStart)

	// Check if we advanced to a new window
	if currentStart.After(w.currWindowStart) {
		windowsPassed := currentStart.Sub(w.currWindowStart) / sw.windowSize
		if windowsPassed == 1 {
			w.prevCount = w.currCount
		} else {
			w.prevCount = 0
		}
		w.currCount = 0
		w.currWindowStart = currentStart
	}

	// Calculate weighted estimate of requests in sliding window
	weight := 1.0 - (float64(elapsedSinceCurrent) / float64(sw.windowSize))
	estimatedCount := float64(w.prevCount)*weight + float64(w.currCount)

	if estimatedCount+float64(n) <= float64(sw.limit) {
		w.currCount += n
		remaining := int64(math.Max(0, float64(sw.limit)-(estimatedCount+float64(n))))
		resetAfter := sw.windowSize - elapsedSinceCurrent

		return Result{
			Allowed:    true,
			Limit:      sw.limit,
			Remaining:  remaining,
			ResetAfter: resetAfter,
			Algorithm:  sw.Algorithm(),
		}, nil
	}

	// Over limit
	resetAfter := sw.windowSize - elapsedSinceCurrent
	return Result{
		Allowed:    false,
		Limit:      sw.limit,
		Remaining:  0,
		ResetAfter: resetAfter,
		RetryAfter: resetAfter,
		Algorithm:  sw.Algorithm(),
	}, nil
}

func (sw *SlidingWindowCounter) Reset(key string) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	delete(sw.windows, key)
}
