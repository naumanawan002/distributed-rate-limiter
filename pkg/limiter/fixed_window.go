package limiter

import (
	"context"
	"sync"
	"time"
)

type fixedWindowState struct {
	windowStart time.Time
	count       int64
}

// FixedWindow implements the classic fixed window rate limiting algorithm.
type FixedWindow struct {
	mu         sync.RWMutex
	limit      int64
	windowSize time.Duration
	windows    map[string]*fixedWindowState
}

// NewFixedWindow creates a fixed window limiter.
func NewFixedWindow(limit int64, windowSize time.Duration) *FixedWindow {
	return &FixedWindow{
		limit:      limit,
		windowSize: windowSize,
		windows:    make(map[string]*fixedWindowState),
	}
}

func (fw *FixedWindow) Algorithm() string {
	return "fixed_window"
}

func (fw *FixedWindow) Allow(ctx context.Context, key string) (Result, error) {
	return fw.AllowN(ctx, key, 1)
}

func (fw *FixedWindow) AllowN(ctx context.Context, key string, n int64) (Result, error) {
	if key == "" {
		return Result{}, ErrKeyRequired
	}

	fw.mu.Lock()
	defer fw.mu.Unlock()

	now := time.Now()
	currentStart := now.Truncate(fw.windowSize)

	w, exists := fw.windows[key]
	if !exists || currentStart.After(w.windowStart) {
		w = &fixedWindowState{
			windowStart: currentStart,
			count:       0,
		}
		fw.windows[key] = w
	}

	resetAfter := fw.windowSize - now.Sub(currentStart)

	if w.count+n <= fw.limit {
		w.count += n
		remaining := fw.limit - w.count
		return Result{
			Allowed:    true,
			Limit:      fw.limit,
			Remaining:  remaining,
			ResetAfter: resetAfter,
			Algorithm:  fw.Algorithm(),
		}, nil
	}

	return Result{
		Allowed:    false,
		Limit:      fw.limit,
		Remaining:  0,
		ResetAfter: resetAfter,
		RetryAfter: resetAfter,
		Algorithm:  fw.Algorithm(),
	}, nil
}

func (fw *FixedWindow) Reset(key string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	delete(fw.windows, key)
}
