package limiter

import (
	"context"
	"errors"
	"time"
)

var (
	ErrKeyRequired = errors.New("rate limit key cannot be empty")
)

// Result contains the rate limiting decision and telemetry.
type Result struct {
	Allowed    bool          `json:"allowed"`
	Limit      int64         `json:"limit"`
	Remaining  int64         `json:"remaining"`
	ResetAfter time.Duration `json:"reset_after"`
	RetryAfter time.Duration `json:"retry_after,omitempty"`
	Algorithm  string        `json:"algorithm"`
}

// RateLimiter is the universal interface implemented by all algorithms.
type RateLimiter interface {
	// Allow checks if 1 request from key is allowed.
	Allow(ctx context.Context, key string) (Result, error)

	// AllowN checks if n tokens/requests from key are allowed.
	AllowN(ctx context.Context, key string, n int64) (Result, error)

	// Algorithm returns the human-readable algorithm name.
	Algorithm() string

	// Reset clears rate limit state for a key (useful for tests and manual resets).
	Reset(key string)
}
