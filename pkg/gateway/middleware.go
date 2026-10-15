package gateway

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nauman/distributed-rate-limiter/pkg/limiter"
)

// Gateway wraps a reverse proxy and attaches multi-algorithm rate limiters.
type Gateway struct {
	tokenBucket   limiter.RateLimiter
	slidingWindow limiter.RateLimiter
	fixedWindow   limiter.RateLimiter
	proxy         *httputil.ReverseProxy
}

// NewGateway creates an API Gateway with rate limiting engines.
func NewGateway(upstreamURL *url.URL, tb, sw, fw limiter.RateLimiter) *Gateway {
	proxy := httputil.NewSingleHostReverseProxy(upstreamURL)
	return &Gateway{
		tokenBucket:   tb,
		slidingWindow: sw,
		fixedWindow:   fw,
		proxy:         proxy,
	}
}

// ServeHTTP applies rate limiting and dispatches to upstream.
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/healthz" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
		return
	}

	// 1. Identify Client Key (API Key > Bearer Token > Remote IP)
	clientKey := extractClientKey(r)

	// 2. Select Rate Limiting Algorithm
	lim := g.selectLimiter(r.Header.Get("X-RateLimit-Algorithm"))

	// 3. Evaluate Rate Limit
	res, err := lim.Allow(r.Context(), clientKey)
	if err != nil {
		http.Error(w, "internal rate limiter error", http.StatusInternalServerError)
		return
	}

	// 4. Inject Standard RateLimit-* Response Headers
	w.Header().Set("RateLimit-Limit", strconv.FormatInt(res.Limit, 10))
	w.Header().Set("RateLimit-Remaining", strconv.FormatInt(res.Remaining, 10))
	w.Header().Set("RateLimit-Reset", strconv.FormatInt(int64(res.ResetAfter.Seconds()), 10))
	w.Header().Set("X-RateLimit-Algorithm-Used", res.Algorithm)

	// 5. Handle Rate Limit Exceeded (HTTP 429)
	if !res.Allowed {
		retrySec := int64(res.RetryAfter.Seconds())
		if retrySec <= 0 {
			retrySec = 1
		}
		w.Header().Set("Retry-After", strconv.FormatInt(retrySec, 10))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":       "too_many_requests",
			"message":     fmt.Sprintf("Rate limit exceeded for key %q using %s", clientKey, res.Algorithm),
			"limit":       res.Limit,
			"retry_after": fmt.Sprintf("%ds", retrySec),
		})
		return
	}

	// 6. Forward to Upstream
	g.proxy.ServeHTTP(w, r)
}

func (g *Gateway) selectLimiter(algoHeader string) limiter.RateLimiter {
	switch strings.ToLower(algoHeader) {
	case "sliding_window", "sliding":
		return g.slidingWindow
	case "fixed_window", "fixed":
		return g.fixedWindow
	default:
		return g.tokenBucket
	}
}

func extractClientKey(r *http.Request) string {
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return "apikey:" + apiKey
	}

	authHeader := r.Header.Get("Authorization")
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return "token:" + authHeader[7:min(len(authHeader), 24)]
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return "ip:" + strings.TrimSpace(parts[0])
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "ip:" + r.RemoteAddr
	}
	return "ip:" + host
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
