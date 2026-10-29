package handler

import (
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/turnstile"
	"log"
	"net/http"
	"time"
)

const (
	MessageHumanCheckFailed = "The human check did not pass. Please try again."
	MessageHumanCheckNeeded = "Please complete the human check."
	MessageHumanCheckOff    = "The human check is not set up on this server."
	HumanPassHeader         = "X-Human-Pass"
)

type HumanPassIssuer interface {
	Issue(ttl time.Duration) (string, time.Time, error)
	Check(token string) error
}

type Human struct {
	cfg    *config.Config
	passes HumanPassIssuer
	checks turnstile.Checker
}

func NewHuman(cfg *config.Config, passes HumanPassIssuer, checks turnstile.Checker) *Human {
	return &Human{cfg: cfg, passes: passes, checks: checks}
}

type humanVerifyRequest struct {
	Token      string `json:"token"`
	Standalone bool   `json:"standalone"`
}

type humanVerifyResponse struct {
	Pass      string `json:"pass"`
	ExpiresAt string `json:"expires_at"`
}

func (h *Human) Verify() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.cfg.HumanCheckEnabled() || h.checks == nil || h.passes == nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, MessageHumanCheckOff)
			return
		}
		var request humanVerifyRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		err := h.checks.Verify(r.Context(), request.Token, middleware.ClientIP(r))
		if errors.Is(err, turnstile.ErrNoToken) || errors.Is(err, turnstile.ErrRefused) {
			httpx.WriteError(w, http.StatusBadRequest, MessageHumanCheckFailed)
			return
		}
		if err != nil {
			log.Printf("handler: could not check a human token: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		ttl := auth.HumanPassBrowser
		if request.Standalone {
			ttl = auth.HumanPassApp
		}
		pass, expiresAt, err := h.passes.Issue(ttl)
		if err != nil {
			log.Printf("handler: could not issue a human pass: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, humanVerifyResponse{
			Pass:      pass,
			ExpiresAt: expiresAt.Format(time.RFC3339),
		})
	}
}

func RequireHuman(cfg *config.Config, passes HumanPassIssuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.HumanCheckEnabled() || passes == nil {
				next.ServeHTTP(w, r)
				return
			}
			if middleware.FromAndroidApp(r) {
				next.ServeHTTP(w, r)
				return
			}
			if err := passes.Check(r.Header.Get(HumanPassHeader)); err != nil {
				httpx.WriteError(w, http.StatusForbidden, MessageHumanCheckNeeded)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
