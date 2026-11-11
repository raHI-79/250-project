package metrics

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// SettleWindow is how long after a start the counters ignore a server error. A
	// deploy or a restart answers a few requests before the database pool is warm, and
	// those errors say nothing about the health of the running server.
	SettleWindow = time.Minute
	// FlushWindow is how often the in-process request counter reaches the database. One
	// row per minute keeps the volume of a busy hour to sixty rows.
	FlushWindow = time.Minute
	// MaintenanceHeader marks the answers the maintenance gate wrote, so that the
	// counters can leave them out of the server errors.
	MaintenanceHeader = "X-Maintenance"
	maxDetail         = 300
)

// probePaths are the addresses a platform health check calls. Render asks for one of
// them every few seconds, and a 5xx from a probe during a restart is not a bug in a
// handler.
var probePaths = map[string]bool{
	"/api/health":  true,
	"/healthz":     true,
	"/api/healthz": true,
	"/health":      true,
}

// probeAgents are the user agents of the platform and cluster probes.
var probeAgents = []string{"kube-probe", "render", "elb-healthchecker", "googlehc", "uptime"}

func IsProbe(r *http.Request) bool {
	if probePaths[strings.ToLower(r.URL.Path)] {
		return true
	}
	agent := strings.ToLower(r.Header.Get("User-Agent"))
	if agent == "" {
		return false
	}
	for _, probe := range probeAgents {
		if strings.Contains(agent, probe) {
			return true
		}
	}
	return false
}

type CounterStore interface {
	AddRequestCounts(ctx context.Context, bucket time.Time, requests, failures int) error
}

// Counter keeps the request volume in memory and writes it to one row a minute. The
// alerts need the volume to tell a handful of errors in a quiet hour from the same
// handful in a busy one, and a row per request would cost more than it is worth.
type Counter struct {
	store  CounterStore
	now    func() time.Time
	window time.Duration

	mu       sync.Mutex
	bucket   time.Time
	requests int
	failures int
}

func NewCounter(store CounterStore) *Counter {
	return &Counter{store: store, now: time.Now, window: FlushWindow}
}

func (c *Counter) Add(failed bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.bucket.IsZero() {
		c.bucket = c.now().Truncate(c.window)
	}
	c.requests++
	if failed {
		c.failures++
	}
}

// Flush writes what has piled up. It returns the number of requests it wrote, so that a
// test can see the write happen.
func (c *Counter) Flush(ctx context.Context) int {
	if c == nil || c.store == nil {
		return 0
	}
	c.mu.Lock()
	bucket, requests, failures := c.bucket, c.requests, c.failures
	c.bucket, c.requests, c.failures = c.now().Truncate(c.window), 0, 0
	c.mu.Unlock()
	if requests == 0 || bucket.IsZero() {
		return 0
	}
	if err := c.store.AddRequestCounts(ctx, bucket, requests, failures); err != nil {
		log.Printf("metrics: could not write the request counters: %v", err)
		return 0
	}
	return requests
}

func (c *Counter) Run(ctx context.Context, every time.Duration) {
	if c == nil {
		return
	}
	if every <= 0 {
		every = FlushWindow
	}
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			c.Flush(context.WithoutCancel(ctx))
			return
		case <-ticker.C:
			c.Flush(ctx)
		}
	}
}

func clip(text string) string {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) <= maxDetail {
		return trimmed
	}
	return trimmed[:maxDetail]
}
