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
	SettleWindow      = time.Minute
	FlushWindow       = time.Minute
	MaintenanceHeader = "X-Maintenance"
	maxDetail         = 300
)

var probePaths = map[string]bool{
	"/api/health":  true,
	"/healthz":     true,
	"/api/healthz": true,
	"/health":      true,
}

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
