package jobs

import (
	"context"
	"fmt"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"time"
)

const (
	NotificationTitle = "Request auto-cancelled"
	NotificationBody  = "not approved in time"
)

type Store interface {
	AutoCancelStaleRequests(ctx context.Context, now time.Time) ([]store.EventDecision, error)
}

type Notifier interface {
	NotifyUser(ctx context.Context, userID int64, kind, title, body string) error
}

type AutoCanceller struct {
	events Store
	alerts Notifier
}

func NewAutoCanceller(events Store, alerts Notifier) *AutoCanceller {
	return &AutoCanceller{events: events, alerts: alerts}
}

func (a *AutoCanceller) AutoCancel(ctx context.Context, now time.Time) (int, error) {
	cancelled, err := a.events.AutoCancelStaleRequests(ctx, now)
	if err != nil {
		return 0, err
	}
	for _, request := range cancelled {
		body := fmt.Sprintf("The %s for %s on %s from %s to %s in room %s was %s.",
			request.EventType,
			request.CourseCode,
			request.EventDate,
			request.StartTime,
			request.EndTime,
			request.Room,
			NotificationBody,
		)
		err := a.alerts.NotifyUser(ctx, request.CreatedBy, notify.TypeDecision, NotificationTitle, body)
		if err != nil {
			log.Printf("jobs: could not notify a class representative of an auto cancelled request: %v", err)
		}
	}
	return len(cancelled), nil
}

func (a *AutoCanceller) Run(ctx context.Context, interval time.Duration, now func() time.Time) {
	a.runOnce(ctx, now())
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.runOnce(ctx, now())
		}
	}
}

func (a *AutoCanceller) runOnce(ctx context.Context, now time.Time) {
	count, err := a.AutoCancel(ctx, now)
	if err != nil {
		log.Printf("jobs: could not auto cancel the stale requests: %v", err)
		return
	}
	if count > 0 {
		log.Printf("jobs: auto cancelled %d requests", count)
	}
}
