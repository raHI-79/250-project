package push

import (
	"context"
	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/raHI-79/250-project/backend/internal/store"
)

type VAPIDSender struct {
	publicKey  string
	privateKey string
	subject    string
}

func NewSender(publicKey, privateKey, subject string) *VAPIDSender {
	return &VAPIDSender{publicKey: publicKey, privateKey: privateKey, subject: subject}
}

func (s *VAPIDSender) Send(ctx context.Context, subscription store.PushSubscription,
	message []byte) (int, error) {
	target := &webpush.Subscription{
		Endpoint: subscription.Endpoint,
		Keys:     webpush.Keys{P256dh: subscription.P256dh, Auth: subscription.Auth},
	}
	response, err := webpush.SendNotificationWithContext(ctx, message, target, &webpush.Options{
		Subscriber:      s.subject,
		VAPIDPublicKey:  s.publicKey,
		VAPIDPrivateKey: s.privateKey,
		TTL:             TTLSeconds,
		Urgency:         webpush.UrgencyHigh,
	})
	if err != nil {
		return 0, err
	}
	defer func() { _ = response.Body.Close() }()
	return response.StatusCode, nil
}
