package fcm

import (
	"context"
	"errors"
	"log"

	"github.com/raHI-79/250-project/backend/internal/store"
)

type Store interface {
	ListFCMTokens(ctx context.Context, userIDs []int64) ([]store.FCMToken, error)
	DropFCMToken(ctx context.Context, token string) error
	TouchFCMTokens(ctx context.Context, tokens []string) error
}

type Worker struct {
	tokens Store
	sender Sender
}

func NewWorker(tokens Store, sender Sender) *Worker {
	return &Worker{tokens: tokens, sender: sender}
}

func (w *Worker) Ready() bool {
	return w != nil && w.sender != nil && w.sender.Configured()
}

func (w *Worker) Deliver(ctx context.Context, userIDs []int64, title, body, url, tag string) int {
	if !w.Ready() || len(userIDs) == 0 || title == "" {
		return 0
	}
	devices, err := w.tokens.ListFCMTokens(ctx, userIDs)
	if err != nil {
		log.Printf("fcm: could not load the device tokens: %v", err)
		return 0
	}
	delivered := make([]string, 0, len(devices))
	for _, device := range devices {
		err := w.sender.Send(ctx, Message{
			Token: device.Token,
			Title: title,
			Body:  body,
			URL:   url,
			Tag:   tag,
		})
		switch {
		case errors.Is(err, ErrGone):
			log.Printf("fcm: a device token is no longer registered, removing it")
			if dropErr := w.tokens.DropFCMToken(ctx, device.Token); dropErr != nil {
				log.Printf("fcm: could not remove a stale device token: %v", dropErr)
			}
		case err != nil:
			log.Printf("fcm: a notification was not delivered: %v", err)
		default:
			delivered = append(delivered, device.Token)
		}
	}
	if len(delivered) > 0 {
		if err := w.tokens.TouchFCMTokens(ctx, delivered); err != nil {
			log.Printf("fcm: could not touch the device tokens: %v", err)
		}
	}
	return len(delivered)
}
