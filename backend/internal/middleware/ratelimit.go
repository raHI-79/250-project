package middleware

import (
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	MessageTooManyAttempts = "Too many attempts. Please try again in a minute."
	DefaultRateLimit       = 10
	DefaultRateWindow      = time.Minute
)

type RateLimiter struct {
	mu        sync.Mutex
	limit     int
	window    time.Duration
	hits      map[string][]time.Time
	lastSweep time.Time
	now       func() time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	now := time.Now
	return &RateLimiter{
		limit:     limit,
		window:    window,
		hits:      make(map[string][]time.Time),
		lastSweep: now(),
		now:       now,
	}
}

func (l *RateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if now.Sub(l.lastSweep) >= l.window {
		l.sweep(now)
	}
	recent := keepRecent(l.hits[key], now.Add(-l.window))
	if len(recent) >= l.limit {
		l.hits[key] = recent
		return false
	}
	l.hits[key] = append(recent, now)
	return true
}

func (l *RateLimiter) Tracked() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.hits)
}

func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.Allow(clientIP(r)) {
			w.Header().Set("Retry-After", strconv.Itoa(int(l.window.Seconds())))
			httpx.WriteError(w, http.StatusTooManyRequests, MessageTooManyAttempts)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *RateLimiter) sweep(now time.Time) {
	cutoff := now.Add(-l.window)
	for key, times := range l.hits {
		recent := keepRecent(times, cutoff)
		if len(recent) == 0 {
			delete(l.hits, key)
			continue
		}
		l.hits[key] = recent
	}
	l.lastSweep = now
}

func keepRecent(times []time.Time, cutoff time.Time) []time.Time {
	recent := times[:0]
	for _, moment := range times {
		if moment.After(cutoff) {
			recent = append(recent, moment)
		}
	}
	return recent
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
