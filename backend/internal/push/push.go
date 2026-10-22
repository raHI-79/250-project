package push

import (
	"context"
	"encoding/json"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"net/http"
	"time"
)

const (
	QueueSize   = 256
	Workers     = 4
	TTLSeconds  = 24 * 60 * 60
	SendTimeout = 10 * time.Second
)

type Payload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
	Tag   string `json:"tag"`
}

type Store interface {
	ListPushSubscriptions(ctx context.Context, userIDs []int64) ([]store.PushSubscription, error)
	DropPushSubscription(ctx context.Context, endpoint string) error
	TouchPushSubscription(ctx context.Context, endpoint string, at time.Time) error
}

type Sender interface {
	Send(ctx context.Context, subscription store.PushSubscription, message []byte) (int, error)
}

type job struct {
	userIDs []int64
	payload Payload
}

type Worker struct {
	subscriptions Store
	sender        Sender
	jobs          chan job
	now           func() time.Time
}

func NewWorker(subscriptions Store, sender Sender, queueSize int) *Worker {
	if queueSize <= 0 {
		queueSize = QueueSize
	}
	return &Worker{
		subscriptions: subscriptions,
		sender:        sender,
		jobs:          make(chan job, queueSize),
		now:           time.Now,
	}
}

func (w *Worker) Run(ctx context.Context, workers int) {
	if workers <= 0 {
		workers = Workers
	}
	for index := 0; index < workers; index++ {
		go w.loop(ctx)
	}
}

func (w *Worker) Queue(userIDs []int64, payload Payload) {
	if len(userIDs) == 0 || payload.Title == "" {
		return
	}
	select {
	case w.jobs <- job{userIDs: userIDs, payload: payload}:
	default:
		log.Printf("push: the queue is full, dropped a notification for %d accounts", len(userIDs))
	}
}

func (w *Worker) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case next := <-w.jobs:
			w.deliver(ctx, next)
		}
	}
}

func (w *Worker) deliver(ctx context.Context, next job) {
	message, err := json.Marshal(next.payload)
	if err != nil {
		log.Printf("push: could not encode a notification: %v", err)
		return
	}
	subscriptions, err := w.subscriptions.ListPushSubscriptions(ctx, next.userIDs)
	if err != nil {
		log.Printf("push: could not load the subscriptions: %v", err)
		return
	}
	for _, subscription := range subscriptions {
		sendContext, cancel := context.WithTimeout(ctx, SendTimeout)
		status, err := w.sender.Send(sendContext, subscription, message)
		cancel()
		if err != nil {
			log.Printf("push: could not send a notification: %v", err)
			continue
		}
		if status == http.StatusNotFound || status == http.StatusGone {
			if err := w.subscriptions.DropPushSubscription(ctx, subscription.Endpoint); err != nil {
				log.Printf("push: could not drop a dead subscription: %v", err)
			}
			continue
		}
		if status >= http.StatusBadRequest {
			log.Printf("push: a push service answered %d", status)
			continue
		}
		if err := w.subscriptions.TouchPushSubscription(ctx, subscription.Endpoint, w.now()); err != nil {
			log.Printf("push: could not mark a subscription used: %v", err)
		}
	}
}
