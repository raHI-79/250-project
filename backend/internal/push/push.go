package push

import (
	"context"
	"encoding/json"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"net/http"
	"net/url"
	"time"
)

const (
	QueueSize   = 256
	Workers     = 4
	TTLSeconds  = 24 * 60 * 60
	SendTimeout = 10 * time.Second
	TestTag     = "sustcse-test"
	unknownHost = "a push service"
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
		w.send(ctx, subscription, message)
	}
}

func (w *Worker) SendTest(ctx context.Context, userID int64, title, body, target string) (int, error) {
	message, err := json.Marshal(Payload{Title: title, Body: body, URL: target, Tag: TestTag})
	if err != nil {
		return 0, err
	}
	subscriptions, err := w.subscriptions.ListPushSubscriptions(ctx, []int64{userID})
	if err != nil {
		return 0, err
	}
	sent := 0
	for _, subscription := range subscriptions {
		if w.send(ctx, subscription, message) {
			sent++
		}
	}
	return sent, nil
}

func (w *Worker) send(ctx context.Context, subscription store.PushSubscription, message []byte) bool {
	sendContext, cancel := context.WithTimeout(ctx, SendTimeout)
	status, err := w.sender.Send(sendContext, subscription, message)
	cancel()
	host := endpointHost(subscription.Endpoint)
	if err != nil {
		log.Printf("push: %s could not be reached", host)
		return false
	}
	if status == http.StatusNotFound || status == http.StatusGone {
		log.Printf("push: %s answered %d, dropping the subscription", host, status)
		if err := w.subscriptions.DropPushSubscription(ctx, subscription.Endpoint); err != nil {
			log.Printf("push: could not drop a dead subscription: %v", err)
		}
		return false
	}
	if status >= http.StatusBadRequest {
		log.Printf("push: %s answered %d", host, status)
		return false
	}
	log.Printf("push: %s answered %d", host, status)
	if err := w.subscriptions.TouchPushSubscription(ctx, subscription.Endpoint, w.now()); err != nil {
		log.Printf("push: could not mark a subscription used: %v", err)
	}
	return true
}

func endpointHost(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" {
		return unknownHost
	}
	return parsed.Host
}
