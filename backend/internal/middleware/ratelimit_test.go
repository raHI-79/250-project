package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func newTestLimiter(limit int, window time.Duration, clock *time.Time) *RateLimiter {
	limiter := NewRateLimiter(limit, window)
	limiter.now = func() time.Time { return *clock }
	limiter.lastSweep = *clock
	return limiter
}

func TestRateLimiterAllow(t *testing.T) {
	tests := []struct {
		name     string
		limit    int
		requests int
		wantLast bool
	}{
		{name: "under the limit", limit: 10, requests: 9, wantLast: true},
		{name: "exactly at the limit", limit: 10, requests: 10, wantLast: true},
		{name: "one over the limit", limit: 10, requests: 11, wantLast: false},
		{name: "far over the limit", limit: 10, requests: 25, wantLast: false},
		{name: "limit of one", limit: 1, requests: 2, wantLast: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clock := time.Now()
			limiter := newTestLimiter(tc.limit, time.Minute, &clock)
			var allowed bool
			for i := 0; i < tc.requests; i++ {
				allowed = limiter.Allow("10.0.0.1")
			}
			if allowed != tc.wantLast {
				t.Errorf("request %d allowed = %t, want %t", tc.requests, allowed, tc.wantLast)
			}
		})
	}
}

func TestRateLimiterCountsPerKey(t *testing.T) {
	clock := time.Now()
	limiter := newTestLimiter(2, time.Minute, &clock)
	for i := 0; i < 2; i++ {
		if !limiter.Allow("10.0.0.1") {
			t.Fatalf("request %d from the first client was blocked too early", i+1)
		}
	}
	if limiter.Allow("10.0.0.1") {
		t.Error("the first client was allowed past its limit")
	}
	if !limiter.Allow("10.0.0.2") {
		t.Error("a second client was blocked by the first client's requests")
	}
}

func TestRateLimiterWindowMovesOn(t *testing.T) {
	clock := time.Now()
	limiter := newTestLimiter(3, time.Minute, &clock)
	for i := 0; i < 3; i++ {
		if !limiter.Allow("10.0.0.1") {
			t.Fatalf("request %d was blocked too early", i+1)
		}
	}
	if limiter.Allow("10.0.0.1") {
		t.Fatal("the fourth request inside the window was allowed")
	}
	clock = clock.Add(30 * time.Second)
	if limiter.Allow("10.0.0.1") {
		t.Error("a request half way through the window was allowed")
	}
	clock = clock.Add(31 * time.Second)
	if !limiter.Allow("10.0.0.1") {
		t.Error("a request after the window passed was blocked")
	}
}

func TestRateLimiterForgetsIdleClients(t *testing.T) {
	clock := time.Now()
	limiter := newTestLimiter(5, time.Minute, &clock)
	for _, key := range []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"} {
		limiter.Allow(key)
	}
	if got := limiter.Tracked(); got != 3 {
		t.Fatalf("tracked clients = %d, want 3", got)
	}
	clock = clock.Add(2 * time.Minute)
	limiter.Allow("10.0.0.4")
	if got := limiter.Tracked(); got != 1 {
		t.Errorf("tracked clients = %d after the window passed, want 1", got)
	}
}

func TestRateLimiterIsConcurrencySafe(t *testing.T) {
	clock := time.Now()
	limiter := newTestLimiter(50, time.Minute, &clock)
	var waitGroup sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for i := 0; i < 200; i++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			if limiter.Allow("10.0.0.1") {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	waitGroup.Wait()
	if allowed != 50 {
		t.Errorf("%d of 200 concurrent requests were allowed, want exactly 50", allowed)
	}
}

func TestRateLimiterMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		requests   int
		wantStatus int
	}{
		{name: "first request", remoteAddr: "10.0.0.1:54321", requests: 1, wantStatus: http.StatusOK},
		{name: "last allowed request", remoteAddr: "10.0.0.1:54321", requests: 10, wantStatus: http.StatusOK},
		{name: "one request too many", remoteAddr: "10.0.0.1:54321", requests: 11, wantStatus: http.StatusTooManyRequests},
		{name: "address without a port", remoteAddr: "10.0.0.1", requests: 11, wantStatus: http.StatusTooManyRequests},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clock := time.Now()
			limiter := newTestLimiter(DefaultRateLimit, DefaultRateWindow, &clock)
			handlerRuns := 0
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerRuns++
				w.WriteHeader(http.StatusOK)
			})
			var rec *httptest.ResponseRecorder
			for i := 0; i < tc.requests; i++ {
				req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
				req.RemoteAddr = tc.remoteAddr
				rec = httptest.NewRecorder()
				limiter.Middleware(next).ServeHTTP(rec, req)
			}
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantStatus != http.StatusTooManyRequests {
				if handlerRuns != tc.requests {
					t.Errorf("the handler ran %d times, want %d", handlerRuns, tc.requests)
				}
				return
			}
			if handlerRuns != DefaultRateLimit {
				t.Errorf("the handler ran %d times, want %d", handlerRuns, DefaultRateLimit)
			}
			if got := rec.Header().Get("Retry-After"); got != "60" {
				t.Errorf("Retry-After = %q, want %q", got, "60")
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
			var payload struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
			}
			if payload.Error != MessageTooManyAttempts {
				t.Errorf("error = %q, want %q", payload.Error, MessageTooManyAttempts)
			}
		})
	}
}

func TestRateLimiterMiddlewareSeparatesClients(t *testing.T) {
	clock := time.Now()
	limiter := newTestLimiter(2, time.Minute, &clock)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	send := func(remoteAddr string) int {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
		req.RemoteAddr = remoteAddr
		rec := httptest.NewRecorder()
		limiter.Middleware(next).ServeHTTP(rec, req)
		return rec.Code
	}
	for i := 0; i < 2; i++ {
		if got := send("10.0.0.1:1111"); got != http.StatusOK {
			t.Fatalf("request %d from the first client = %d, want 200", i+1, got)
		}
	}
	if got := send("10.0.0.1:2222"); got != http.StatusTooManyRequests {
		t.Errorf("a third request from the same address on a new port = %d, want 429", got)
	}
	if got := send("10.0.0.9:1111"); got != http.StatusOK {
		t.Errorf("a request from another address = %d, want 200", got)
	}
}
