package handler

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/metrics"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/settings"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	MessageBlocked     = "This address cannot use the portal."
	MessageFeatureOff  = "This part of the portal is switched off right now."
	blockCacheWindow   = 30 * time.Second
	maintenanceAllowed = "/api/health,/api/auth,/api/office,/api/super-admin,/api/admin,/api/system,/api/public,/api/analytics,/api/me"
)

type Gates struct {
	values *settings.Cache
}

func NewGates(values *settings.Cache) *Gates {
	return &Gates{values: values}
}

func (g *Gates) Maintenance(next http.Handler) http.Handler {
	allowed := strings.Split(maintenanceAllowed, ",")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if g.values == nil || !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		for _, prefix := range allowed {
			if strings.HasPrefix(r.URL.Path, prefix) {
				next.ServeHTTP(w, r)
				return
			}
		}
		closed, message := g.values.Maintenance(r.Context())
		if !closed {
			next.ServeHTTP(w, r)
			return
		}
		// The header tells the request counters that this answer was meant, so that a
		// maintenance window does not look like a wave of server errors.
		w.Header().Set(metrics.MaintenanceHeader, settings.On)
		httpx.WriteError(w, http.StatusServiceUnavailable, message)
	})
}

func (g *Gates) Feature(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if g.values == nil || g.values.On(r.Context(), key) {
				next.ServeHTTP(w, r)
				return
			}
			httpx.WriteError(w, http.StatusServiceUnavailable, MessageFeatureOff)
		})
	}
}

func (g *Gates) WebAttendance(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if g.values == nil || middleware.FromAndroidApp(r) ||
			g.values.On(r.Context(), settings.KeyWebAttendance) {
			next.ServeHTTP(w, r)
			return
		}
		httpx.WriteError(w, http.StatusServiceUnavailable,
			"Attendance can only be marked from the Android app right now.")
	})
}

type BlockStore interface {
	ActiveIPBlocks(ctx context.Context) ([]string, error)
}

type Blocks struct {
	store  BlockStore
	events *Recorder
	mu     sync.RWMutex
	ranges []*net.IPNet
	plain  map[string]bool
	readAt time.Time
}

func NewBlocks(store BlockStore, events *Recorder) *Blocks {
	return &Blocks{store: store, events: events}
}

func (b *Blocks) Forget() {
	b.mu.Lock()
	b.readAt = time.Time{}
	b.mu.Unlock()
}

func (b *Blocks) load(ctx context.Context) ([]*net.IPNet, map[string]bool) {
	b.mu.RLock()
	fresh := !b.readAt.IsZero() && time.Since(b.readAt) < blockCacheWindow
	ranges, plain := b.ranges, b.plain
	b.mu.RUnlock()
	if fresh {
		return ranges, plain
	}
	entries, err := b.store.ActiveIPBlocks(ctx)
	if err != nil {
		log.Printf("handler: could not read the blocked addresses: %v", err)
		return ranges, plain
	}
	nets := make([]*net.IPNet, 0, len(entries))
	exact := make(map[string]bool, len(entries))
	for _, entry := range entries {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" {
			continue
		}
		if _, network, err := net.ParseCIDR(trimmed); err == nil {
			nets = append(nets, network)
			continue
		}
		exact[trimmed] = true
	}
	b.mu.Lock()
	b.ranges = nets
	b.plain = exact
	b.readAt = time.Now()
	b.mu.Unlock()
	return nets, exact
}

func (b *Blocks) Blocked(ctx context.Context, address string) bool {
	if b == nil || b.store == nil || address == "" {
		return false
	}
	ranges, plain := b.load(ctx)
	if plain[address] {
		return true
	}
	parsed := net.ParseIP(address)
	if parsed == nil {
		return false
	}
	for _, network := range ranges {
		if network.Contains(parsed) {
			return true
		}
	}
	return false
}

func (b *Blocks) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		address := RequestAddress(r)
		if !b.Blocked(r.Context(), address) {
			next.ServeHTTP(w, r)
			return
		}
		b.events.RecordAs(r, 0, "", "blocked", "blocked_request", "request", r.URL.Path,
			"A blocked address asked for "+r.URL.Path)
		httpx.WriteError(w, http.StatusForbidden, MessageBlocked)
	})
}
