package gateway

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/nauman/distributed-rate-limiter/pkg/limiter"
)

func TestGateway_MiddlewareHeadersAndStatus(t *testing.T) {
	// Upstream mock
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"upstream":"success"}`))
	}))
	defer upstream.Close()

	uURL, _ := url.Parse(upstream.URL)
	tb := limiter.NewTokenBucket(2, 1.0)
	sw := limiter.NewSlidingWindowCounter(5, time.Minute)
	fw := limiter.NewFixedWindow(5, time.Minute)

	gw := NewGateway(uURL, tb, sw, fw)

	// 1. Request 1 - Allowed
	req1 := httptest.NewRequest("GET", "/api/data", nil)
	req1.Header.Set("X-API-Key", "test-key-123")
	rr1 := httptest.NewRecorder()
	gw.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr1.Code)
	}
	if rr1.Header().Get("RateLimit-Remaining") != "1" {
		t.Errorf("expected Remaining 1, got %s", rr1.Header().Get("RateLimit-Remaining"))
	}

	// 2. Request 2 - Allowed (remaining 0)
	req2 := httptest.NewRequest("GET", "/api/data", nil)
	req2.Header.Set("X-API-Key", "test-key-123")
	rr2 := httptest.NewRecorder()
	gw.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr2.Code)
	}

	// 3. Request 3 - Rate limited (429)
	req3 := httptest.NewRequest("GET", "/api/data", nil)
	req3.Header.Set("X-API-Key", "test-key-123")
	rr3 := httptest.NewRecorder()
	gw.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 Too Many Requests, got %d", rr3.Code)
	}
	if rr3.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header on 429")
	}
}
