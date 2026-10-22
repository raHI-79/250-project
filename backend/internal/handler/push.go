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
	MaxEndpointLength     = 1000
	MaxKeyLength          = 400
	MaxAgentLength        = 300
)

type PushStore interface {
	SavePushSubscription(ctx context.Context, subscription store.PushSubscription) (int64, error)
	DeletePushSubscription(ctx context.Context, userID int64, endpoint string) error
}

type Push struct {
	cfg           *config.Config
	subscriptions PushStore
}

func NewPush(cfg *config.Config, subscriptions PushStore) *Push {
	return &Push{cfg: cfg, subscriptions: subscriptions}
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

type publicKeyResponse struct {
	Enabled   bool   `json:"enabled"`
	PublicKey string `json:"public_key,omitempty"`
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
		httpx.WriteJSON(w, http.StatusCreated, messageResponse{Message: MessagePushSaved})
	}
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
