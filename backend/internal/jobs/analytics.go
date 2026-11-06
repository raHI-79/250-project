package jobs

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/settings"
	"log"
	"time"
)

const (
	AnalyticsInterval = time.Hour
	rollUpDays        = 3
	defaultViewDays   = 90
	defaultEventDays  = 365
	defaultNoticeDays = 120
	closedSessionDays = 30
	// pushUnusedDays is how long a web push subscription may sit unused. A browser that
	// has not taken a notification in two months has thrown the subscription away, and a
	// dead row fails every send after that.
	pushUnusedDays = 60
)

type AnalyticsStore interface {
	RollUpAnalytics(ctx context.Context, from, to time.Time) error
	DeleteOldViews(ctx context.Context, before time.Time) (int64, error)
	DeleteOldEvents(ctx context.Context, before time.Time) (int64, error)
	DeleteOldNotifications(ctx context.Context, before time.Time) (int64, error)
	DeleteClosedSessions(ctx context.Context, before time.Time) (int64, error)
	CleanPushSubscriptions(ctx context.Context, before time.Time) (int, error)
}

type Analytics struct {
	cfg    *config.Config
	views  AnalyticsStore
	values *settings.Cache
}

func (a *Analytics) WithSettings(values *settings.Cache) *Analytics {
	a.values = values
	return a
}

func (a *Analytics) days(ctx context.Context, key string, fallback int) int {
	if a.values == nil {
		return fallback
	}
	return a.values.Number(ctx, key, fallback)
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
	a.sweep(ctx, now)
}

func (a *Analytics) sweep(ctx context.Context, now time.Time) {
	viewDays := a.days(ctx, settings.KeyRetainViews, defaultViewDays)
	if dropped, err := a.views.DeleteOldViews(ctx, now.AddDate(0, 0, -viewDays)); err != nil {
		log.Printf("jobs: could not delete the old page views: %v", err)
	} else if dropped > 0 {
		log.Printf("jobs: deleted %d page views older than %d days", dropped, viewDays)
	}
	eventDays := a.days(ctx, settings.KeyRetainEvents, defaultEventDays)
	if dropped, err := a.views.DeleteOldEvents(ctx, now.AddDate(0, 0, -eventDays)); err != nil {
		log.Printf("jobs: could not delete the old activity events: %v", err)
	} else if dropped > 0 {
		log.Printf("jobs: deleted %d activity events older than %d days", dropped, eventDays)
	}
	noticeDays := a.days(ctx, settings.KeyRetainNotices, defaultNoticeDays)
	if dropped, err := a.views.DeleteOldNotifications(ctx, now.AddDate(0, 0, -noticeDays)); err != nil {
		log.Printf("jobs: could not delete the old notifications: %v", err)
	} else if dropped > 0 {
		log.Printf("jobs: deleted %d notifications older than %d days", dropped, noticeDays)
	}
	if dropped, err := a.views.DeleteClosedSessions(ctx, now.AddDate(0, 0, -closedSessionDays)); err != nil {
		log.Printf("jobs: could not delete the closed sessions: %v", err)
	} else if dropped > 0 {
		log.Printf("jobs: deleted %d closed sessions", dropped)
	}
	if dropped, err := a.views.CleanPushSubscriptions(ctx,
		now.AddDate(0, 0, -pushUnusedDays)); err != nil {
		log.Printf("jobs: could not clean the push subscriptions: %v", err)
	} else if dropped > 0 {
		log.Printf("jobs: dropped %d push subscriptions unused for %d days", dropped, pushUnusedDays)
	}
}
