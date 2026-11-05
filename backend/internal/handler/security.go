package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/mailer"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	MessageConfirmNeeded    = "Please confirm your password first."
	MessageConfirmOpen      = "Password confirmed. Sensitive actions are open for five minutes."
	MessagePasswordChanged  = "Password changed. Every other session was signed out."
	MessageContactChanged   = "Contact details updated."
	MessageTwoStepOn        = "Two-step sign-in is on."
	MessageTwoStepOff       = "Two-step sign-in is off."
	MessageSessionClosed    = "Session signed out."
	MessageOtherSessionsOut = "Every other session was signed out."
	MessageSessionNotFound  = "That session is already closed."
	SudoWindow              = 5 * time.Minute
)

type SecurityStore interface {
	GetPasswordHash(ctx context.Context, userID int64) (string, error)
	SetPassword(ctx context.Context, userID int64, passwordHash string) (int, error)
	ListSessions(ctx context.Context, userID int64) ([]store.Session, error)
	RevokeSession(ctx context.Context, userID, sessionID int64) error
	RevokeOtherSessions(ctx context.Context, userID, keepID int64) (int64, error)
	GrantSudo(ctx context.Context, sessionID int64, until time.Time) error
	DropSudo(ctx context.Context, userID int64) error
	SetTwoStep(ctx context.Context, userID int64, enabled bool) error
	UpdateOfficeContact(ctx context.Context, userID int64, email, phone string) error
	TwoStepChallenge(ctx context.Context, identifier string) (store.TwoStepChallenge, error)
}

type Security struct {
	cfg    *config.Config
	people SecurityStore
	mail   mailer.Sender
	events *Recorder
}

func NewSecurity(cfg *config.Config, people SecurityStore, mail mailer.Sender,
	events *Recorder) *Security {
	return &Security{cfg: cfg, people: people, mail: mail, events: events}
}

type confirmPasswordRequest struct {
	Password string `json:"password"`
}

type securityPasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

type contactChangeRequest struct {
	Password string `json:"password"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

type twoStepRequest struct {
	Password string `json:"password"`
	Enabled  bool   `json:"enabled"`
}

type sessionItem struct {
	ID         int64  `json:"id"`
	Device     string `json:"device"`
	Client     string `json:"client"`
	Country    string `json:"country"`
	LastSeenAt string `json:"last_seen_at"`
	CreatedAt  string `json:"created_at"`
	Current    bool   `json:"current"`
}

type signedInResponse struct {
	Items []sessionItem `json:"items"`
}

type sudoResponse struct {
	Message string `json:"message"`
	Until   string `json:"until"`
}

type securityStateResponse struct {
	Identifier string `json:"identifier"`
	Role       string `json:"role"`
	TwoStep    bool   `json:"two_step"`
	SudoUntil  string `json:"sudo_until,omitempty"`
}

func (s *Security) checkPassword(w http.ResponseWriter, r *http.Request, userID int64,
	password string) bool {
	if strings.TrimSpace(password) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Password is required.")
		return false
	}
	hash, err := s.people.GetPasswordHash(r.Context(), userID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageSessionExpired)
		return false
	}
	if err != nil {
		log.Printf("handler: could not load a password: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return false
	}
	if auth.CheckPassword(hash, password) != nil {
		httpx.WriteError(w, http.StatusBadRequest, MessageWrongPassword)
		return false
	}
	return true
}

func (s *Security) State() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		challenge, err := s.people.TwoStepChallenge(r.Context(), current.Identifier)
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			log.Printf("handler: could not read the two-step setting: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		body := securityStateResponse{
			Identifier: current.Identifier,
			Role:       current.Role,
			TwoStep:    challenge.Enabled,
		}
		if current.SudoUntil != nil && current.SudoUntil.After(s.cfg.Now()) {
			body.SudoUntil = current.SudoUntil.Format(time.RFC3339)
		}
		httpx.WriteJSON(w, http.StatusOK, body)
	}
}

func (s *Security) Confirm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		var request confirmPasswordRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		if !s.checkPassword(w, r, current.ID, request.Password) {
			return
		}
		if current.SessionID <= 0 {
			httpx.WriteError(w, http.StatusConflict, middleware.MessageSessionExpired)
			return
		}
		until := s.cfg.Now().Add(SudoWindow)
		if err := s.people.GrantSudo(r.Context(), current.SessionID, until); err != nil {
			if errors.Is(err, store.ErrSessionGone) {
				httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageSessionExpired)
				return
			}
			log.Printf("handler: could not open the confirmation window: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		s.events.Record(r, store.ActionConfirmPassword, "session", idText(current.SessionID),
			current.Identifier+" confirmed their password")
		httpx.WriteJSON(w, http.StatusOK, sudoResponse{
			Message: MessageConfirmOpen,
			Until:   until.Format(time.RFC3339),
		})
	}
}

func (s *Security) ChangePassword() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		var request securityPasswordRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		if err := validate.OfficePassword("New password", request.NewPassword,
			request.ConfirmPassword); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if request.NewPassword == request.CurrentPassword {
			httpx.WriteError(w, http.StatusBadRequest, MessageSamePassword)
			return
		}
		if !s.checkPassword(w, r, current.ID, request.CurrentPassword) {
			return
		}
		hash, err := auth.HashPassword(request.NewPassword)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, MessagePasswordTooLong)
			return
		}
		if _, err := s.people.SetPassword(r.Context(), current.ID, hash); err != nil {
			log.Printf("handler: could not change a password: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if _, err := s.people.RevokeOtherSessions(r.Context(), current.ID, current.SessionID); err != nil {
			log.Printf("handler: could not revoke the other sessions: %v", err)
		}
		s.events.Record(r, store.ActionPasswordChange, "account", idText(current.ID),
			current.Identifier+" changed their password")
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessagePasswordChanged})
	}
}

func (s *Security) ChangeContact() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		var request contactChangeRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		email := strings.TrimSpace(request.Email)
		phone := strings.TrimSpace(request.Phone)
		if email == "" && phone == "" {
			httpx.WriteError(w, http.StatusBadRequest, "Nothing to change.")
			return
		}
		if email != "" {
			checked, err := validate.Email("Email", email)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			email = checked
		}
		if phone != "" {
			checked, err := validate.Phone("Phone", phone)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			phone = checked
		}
		if !s.checkPassword(w, r, current.ID, request.Password) {
			return
		}
		err := s.people.UpdateOfficeContact(r.Context(), current.ID, email, phone)
		if errors.Is(err, store.ErrDuplicateAccount) {
			httpx.WriteError(w, http.StatusConflict, MessageDuplicateAccount)
			return
		}
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusForbidden, middleware.MessageAccessDenied)
			return
		}
		if err != nil {
			log.Printf("handler: could not change the office contact details: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		s.events.Record(r, store.ActionContactChange, "account", idText(current.ID),
			current.Identifier+" changed their contact details")
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageContactChanged})
	}
}

func (s *Security) TwoStep() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		var request twoStepRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		if !s.checkPassword(w, r, current.ID, request.Password) {
			return
		}
		if err := s.people.SetTwoStep(r.Context(), current.ID, request.Enabled); err != nil {
			log.Printf("handler: could not change the two-step setting: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		message := MessageTwoStepOff
		action := store.ActionTwoStepOff
		if request.Enabled {
			message = MessageTwoStepOn
			action = store.ActionTwoStepOn
		}
		s.events.Record(r, action, "account", idText(current.ID), current.Identifier+": "+message)
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: message})
	}
}

func (s *Security) Sessions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		sessions, err := s.people.ListSessions(r.Context(), current.ID)
		if err != nil {
			log.Printf("handler: could not list the sessions: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]sessionItem, 0, len(sessions))
		for _, session := range sessions {
			country := session.Country
			if country == "" {
				country = "Unknown"
			}
			device := session.Device
			if device == "" {
				device = "Unknown device"
			}
			items = append(items, sessionItem{
				ID:         session.ID,
				Device:     device,
				Client:     session.Client,
				Country:    country,
				LastSeenAt: session.LastSeenAt.Format(time.RFC3339),
				CreatedAt:  session.CreatedAt.Format(time.RFC3339),
				Current:    session.ID == current.SessionID,
			})
		}
		httpx.WriteJSON(w, http.StatusOK, signedInResponse{Items: items})
	}
}

func (s *Security) CloseSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		sessionID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
		if err != nil || sessionID <= 0 {
			httpx.WriteError(w, http.StatusBadRequest, MessageInvalidID)
			return
		}
		err = s.people.RevokeSession(r.Context(), current.ID, sessionID)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, MessageSessionNotFound)
			return
		}
		if err != nil {
			log.Printf("handler: could not revoke a session: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		s.events.Record(r, store.ActionSessionOut, "session", idText(sessionID),
			current.Identifier+" signed a session out")
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageSessionClosed})
	}
}

func (s *Security) CloseOtherSessions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		if _, err := s.people.RevokeOtherSessions(r.Context(), current.ID, current.SessionID); err != nil {
			log.Printf("handler: could not revoke the other sessions: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if err := s.people.DropSudo(r.Context(), current.ID); err != nil {
			log.Printf("handler: could not close the confirmation windows: %v", err)
		}
		s.events.Record(r, store.ActionSessionOutAll, "account", idText(current.ID),
			current.Identifier+" signed every other session out")
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageOtherSessionsOut})
	}
}

func RequireSudo(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead {
				next.ServeHTTP(w, r)
				return
			}
			current, ok := middleware.CurrentUser(r.Context())
			if !ok {
				httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
				return
			}
			if !current.HasSudo(cfg.Now()) {
				httpx.WriteError(w, http.StatusPreconditionRequired, MessageConfirmNeeded)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
