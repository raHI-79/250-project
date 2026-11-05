package jobs

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"time"
)

const (
	AnalyticsInterval = time.Hour
	rollUpDays        = 3
)

type AnalyticsStore interface {
	RollUpAnalytics(ctx context.Context, from, to time.Time) error
	DeleteOldViews(ctx context.Context, before time.Time) (int64, error)
}

type Analytics struct {
	cfg   *config.Config
	views AnalyticsStore
}

func NewAnalytics(cfg *config.Config, views AnalyticsStore) *Analytics {
	return &Analytics{cfg: cfg, views: views}
}

func (a *Analytics) Run(ctx context.Context, interval time.Duration) {
	a.RunOnce(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.RunOnce(ctx)
		}
	}
}

func (a *Analytics) RunOnce(ctx context.Context) {
	now := a.cfg.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, a.cfg.Location)
	from := today.AddDate(0, 0, -rollUpDays)
	if err := a.views.RollUpAnalytics(ctx, from, today.AddDate(0, 0, 1)); err != nil {
		log.Printf("jobs: could not roll up the page views: %v", err)
	}
	cutoff := now.Add(-store.ViewRetained)
	if dropped, err := a.views.DeleteOldViews(ctx, cutoff); err != nil {
		log.Printf("jobs: could not delete the old page views: %v", err)
	} else if dropped > 0 {
		log.Printf("jobs: deleted %d page views older than 90 days", dropped)
	}
}
