package handler

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"net/http"
	"sync"
	"time"
)

const StatsCacheTTL = 5 * time.Minute

type PublicStore interface {
	PublicStats(ctx context.Context) (store.PublicStats, error)
}

type publicStatsResponse struct {
	Students int `json:"students"`
	Teachers int `json:"teachers"`
	Courses  int `json:"courses"`
	Batches  int `json:"batches"`
}

type Public struct {
	numbers PublicStore
	now     func() time.Time
	guard   sync.Mutex
	cached  publicStatsResponse
	freshAt time.Time
}

func NewPublic(numbers PublicStore, now func() time.Time) *Public {
	if now == nil {
		now = time.Now
	}
	return &Public{numbers: numbers, now: now}
}

func (p *Public) Stats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := p.stats(r.Context())
		if err != nil {
			log.Printf("handler: could not load the public stats: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=300")
		httpx.WriteJSON(w, http.StatusOK, body)
	}
}

func (p *Public) stats(ctx context.Context) (publicStatsResponse, error) {
	p.guard.Lock()
	defer p.guard.Unlock()
	now := p.now()
	if !p.freshAt.IsZero() && now.Sub(p.freshAt) < StatsCacheTTL {
		return p.cached, nil
	}
	stats, err := p.numbers.PublicStats(ctx)
	if err != nil {
		if !p.freshAt.IsZero() {
			return p.cached, nil
		}
		return publicStatsResponse{}, err
	}
	p.cached = publicStatsResponse{
		Students: stats.Students,
		Teachers: stats.Teachers,
		Courses:  stats.Courses,
		Batches:  stats.Batches,
	}
	p.freshAt = now
	return p.cached, nil
}
