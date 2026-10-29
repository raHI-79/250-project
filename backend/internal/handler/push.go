package handler

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"net/http"
	"strings"
)

const (
	MessagePushSaved      = "This device will receive notifications."
	MessagePushRemoved    = "This device will not receive notifications."
	MessagePushOff        = "Push notifications are not set up on this server."
	MessagePushIncomplete = "The browser did not send a complete subscription."
	MessagePushNoDevice   = "No device on this account is set up for notifications."
	MessagePushTestSent   = "Test notification sent."
	MessagePushLogged     = "Reported."
	MessagePushLogEmpty   = "The report is empty."
	MessageTokenSaved     = "This device will receive notifications."
	MessageTokenRemoved   = "This device will not receive notifications."
	MessageTokenMissing   = "The app did not send a device token."
	MaxTokenLength        = 4096
	MaxPlatformLength     = 20
	MaxDeviceLength       = 100
	TestPushTitle         = "SUST CSE"
	TestPushBody          = "Notifications are working on this device."
	MaxEndpointLength     = 1000
	MaxKeyLength          = 400
	MaxAgentLength        = 300
	MaxLogFieldLength     = 200
)

type PushStore interface {
	SavePushSubscription(ctx context.Context, subscription store.PushSubscription) (int64, error)
	DeletePushSubscription(ctx context.Context, userID int64, endpoint string) error
	CountPushSubscriptions(ctx context.Context, userID int64) (int, error)
	SaveFCMToken(ctx context.Context, token store.FCMToken) (int64, error)
	DeleteFCMToken(ctx context.Context, userID int64, token string) error
	CountFCMTokens(ctx context.Context, userID int64) (int, error)
}

type PushTester interface {
	SendTest(ctx context.Context, userID int64, title, body, target string) (int, error)
}

type Push struct {
	cfg           *config.Config
	subscriptions PushStore
	tester        PushTester
}

func NewPush(cfg *config.Config, subscriptions PushStore, tester PushTester) *Push {
	return &Push{cfg: cfg, subscriptions: subscriptions, tester: tester}
}

type pushKeys struct {
	P256dh string `json:"p256dh"`
	Auth   string `json:"auth"`
}

type pushSubscriptionRequest struct {
	Endpoint  string   `json:"endpoint"`
	Keys      pushKeys `json:"keys"`
	UserAgent string   `json:"user_agent"`
}

type pushClientLogRequest struct {
	Stage   string `json:"stage"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

type fcmTokenRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
	DeviceID string `json:"device_id"`
}

type publicKeyResponse struct {
	Enabled   bool   `json:"enabled"`
	PublicKey string `json:"public_key,omitempty"`
}

type testPushResponse struct {
	Sent    int    `json:"sent"`
	Message string `json:"message"`
}

func (p *Push) PublicKey() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, publicKeyResponse{
			Enabled:   p.cfg.PushEnabled(),
			PublicKey: p.cfg.VAPIDPublicKey,
		})
	}
}

func (p *Push) Subscribe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		if !p.cfg.PushEnabled() {
			httpx.WriteError(w, http.StatusServiceUnavailable, MessagePushOff)
			return
		}
		var request pushSubscriptionRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		endpoint := strings.TrimSpace(request.Endpoint)
		p256dh := strings.TrimSpace(request.Keys.P256dh)
		auth := strings.TrimSpace(request.Keys.Auth)
		if endpoint == "" || p256dh == "" || auth == "" {
			httpx.WriteError(w, http.StatusBadRequest, MessagePushIncomplete)
			return
		}
		if len(endpoint) > MaxEndpointLength || len(p256dh) > MaxKeyLength || len(auth) > MaxKeyLength {
			httpx.WriteError(w, http.StatusBadRequest, MessagePushIncomplete)
			return
		}
		agent := strings.TrimSpace(request.UserAgent)
		if agent == "" {
			agent = strings.TrimSpace(r.UserAgent())
		}
		if len(agent) > MaxAgentLength {
			agent = agent[:MaxAgentLength]
		}
		_, err := p.subscriptions.SavePushSubscription(r.Context(), store.PushSubscription{
			UserID:    current.ID,
			Endpoint:  endpoint,
			P256dh:    p256dh,
			Auth:      auth,
			UserAgent: agent,
		})
		if err != nil {
			log.Printf("handler: could not save a push subscription: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		count, err := p.subscriptions.CountPushSubscriptions(r.Context(), current.ID)
		if err != nil {
			log.Printf("handler: could not count the push subscriptions of an account: %v", err)
		} else {
			log.Printf("push: account %d now has %d subscriptions", current.ID, count)
		}
		httpx.WriteJSON(w, http.StatusCreated, messageResponse{Message: MessagePushSaved})
	}
}

func (p *Push) SaveToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		var request fcmTokenRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		token := strings.TrimSpace(request.Token)
		if token == "" || len(token) > MaxTokenLength {
			httpx.WriteError(w, http.StatusBadRequest, MessageTokenMissing)
			return
		}
		platform := clip(strings.ToLower(strings.TrimSpace(request.Platform)), MaxPlatformLength)
		if platform == "" {
			platform = store.PlatformAndroid
		}
		_, err := p.subscriptions.SaveFCMToken(r.Context(), store.FCMToken{
			UserID:   current.ID,
			Token:    token,
			Platform: platform,
			DeviceID: clip(strings.TrimSpace(request.DeviceID), MaxDeviceLength),
		})
		if err != nil {
			log.Printf("handler: could not save a device token: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if count, err := p.subscriptions.CountFCMTokens(r.Context(), current.ID); err == nil {
			log.Printf("push: account %d now has %d device tokens", current.ID, count)
		}
		httpx.WriteJSON(w, http.StatusCreated, messageResponse{Message: MessageTokenSaved})
	}
}

func (p *Push) DeleteToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		var request fcmTokenRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		token := strings.TrimSpace(request.Token)
		if token == "" {
			httpx.WriteError(w, http.StatusBadRequest, MessageTokenMissing)
			return
		}
		if err := p.subscriptions.DeleteFCMToken(r.Context(), current.ID, token); err != nil {
			log.Printf("handler: could not delete a device token: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageTokenRemoved})
	}
}

func clip(value string, limit int) string {
	if len(value) > limit {
		return value[:limit]
	}
	return value
}

func (p *Push) ClientLog() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		var request pushClientLogRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		stage := clipField(request.Stage)
		name := clipField(request.Name)
		message := clipField(request.Message)
		if stage == "" && name == "" && message == "" {
			httpx.WriteError(w, http.StatusBadRequest, MessagePushLogEmpty)
			return
		}
		log.Printf("push: account %d could not set up notifications at stage %q: %s: %s",
			current.ID, stage, name, message)
		httpx.WriteJSON(w, http.StatusAccepted, messageResponse{Message: MessagePushLogged})
	}
}

func clipField(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) > MaxLogFieldLength {
		return trimmed[:MaxLogFieldLength]
	}
	return trimmed
}

func (p *Push) Unsubscribe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		var request pushSubscriptionRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		endpoint := strings.TrimSpace(request.Endpoint)
		if endpoint == "" {
			httpx.WriteError(w, http.StatusBadRequest, MessagePushIncomplete)
			return
		}
		if err := p.subscriptions.DeletePushSubscription(r.Context(), current.ID, endpoint); err != nil {
			log.Printf("handler: could not delete a push subscription: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessagePushRemoved})
	}
}

func (p *Push) Test() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		if !p.cfg.PushEnabled() || p.tester == nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, MessagePushOff)
			return
		}
		sent, err := p.tester.SendTest(r.Context(), current.ID, TestPushTitle, TestPushBody, p.cfg.FrontendOrigin)
		if err != nil {
			log.Printf("handler: could not send a test notification: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if sent == 0 {
			httpx.WriteError(w, http.StatusNotFound, MessagePushNoDevice)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, testPushResponse{Sent: sent, Message: MessagePushTestSent})
	}
}
