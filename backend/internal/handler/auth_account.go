package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/googleauth"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/settings"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	MessageGoogleDisabled   = "Google sign-in is not configured."
	MessageGoogleCredential = "credential is required."
	MessageGoogleAudience   = "Google issued this sign-in for a different client id. The app and the server have to use the same Google web client id."
	MessageGoogleExpired    = "The Google sign-in expired before it reached the server. Try again."
	MessageGoogleRefused    = "Google could not verify this sign-in. Try again."
)

type authConfigResponse struct {
	Maintenance         bool            `json:"maintenance"`
	MaintenanceMessage  string          `json:"maintenance_message,omitempty"`
	RegistrationOpen    bool            `json:"registration_open"`
	RegistrationMessage string          `json:"registration_message,omitempty"`
	Announcement        string          `json:"announcement,omitempty"`
	AnnouncementTone    string          `json:"announcement_tone,omitempty"`
	AnnouncementRoles   []string        `json:"announcement_roles,omitempty"`
	MinAppVersion       string          `json:"min_app_version,omitempty"`
	Features            map[string]bool `json:"features,omitempty"`
	StudentEmailDomain  string          `json:"student_email_domain"`
	GoogleEnabled       bool            `json:"google_enabled"`
	GoogleClientID      string          `json:"google_client_id,omitempty"`
	HumanCheckEnabled   bool            `json:"human_check_enabled"`
	TurnstileSiteKey    string          `json:"turnstile_site_key,omitempty"`
}

type dropCourseOption struct {
	ID    int64  `json:"id"`
	Code  string `json:"code"`
	Title string `json:"title"`
}

type dropCourseOptionsResponse struct {
	Items []dropCourseOption `json:"items"`
}

type googleVerifyRequest struct {
	Credential string `json:"credential"`
}

type googleVerifyResponse struct {
	Email      string `json:"email"`
	EmailProof string `json:"email_proof"`
	ExpiresAt  string `json:"expires_at"`
}

func (a *Auth) Config() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body := authConfigResponse{
			StudentEmailDomain: a.cfg.StudentDomain,
			GoogleEnabled:      a.cfg.GoogleEnabled(),
			GoogleClientID:     strings.TrimSpace(a.cfg.GoogleClientID),
			HumanCheckEnabled:  a.cfg.HumanCheckEnabled(),
			TurnstileSiteKey:   a.cfg.TurnstileSiteKey,
			RegistrationOpen:   true,
		}
		if a.values != nil {
			closed, message := a.values.Maintenance(r.Context())
			body.Maintenance = closed
			body.MaintenanceMessage = message
			open, note := a.values.Registration(r.Context())
			body.RegistrationOpen = open
			body.RegistrationMessage = note
			body.MinAppVersion = a.values.Get(r.Context(), settings.KeyMinAppVersion)
			body.Features = featureState(r.Context(), a.values)
			text, tone := announcementFor(r.Context(), a.values, "")
			body.Announcement = text
			body.AnnouncementTone = tone
			body.AnnouncementRoles = a.values.List(r.Context(), settings.KeyAnnouncementRoles)
		}
		httpx.WriteJSON(w, http.StatusOK, body)
	}
}

func featureState(ctx context.Context, values *settings.Cache) map[string]bool {
	state := make(map[string]bool, len(settings.Features()))
	for _, key := range settings.Features() {
		state[strings.TrimPrefix(key, "feature_")] = values.On(ctx, key)
	}
	return state
}

func (a *Auth) DropCourseOptions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		search, err := validate.Search("search", r.URL.Query().Get("search"))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		courses, err := a.users.SearchDropCourseOptions(r.Context(), search)
		if err != nil {
			log.Printf("handler: could not list the drop course options: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]dropCourseOption, 0, len(courses))
		for _, course := range courses {
			items = append(items, dropCourseOption{ID: course.ID, Code: course.Code, Title: course.Title})
		}
		httpx.WriteJSON(w, http.StatusOK, dropCourseOptionsResponse{Items: items})
	}
}

func (a *Auth) Sections() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		batch, err := validate.Batch("batch", r.URL.Query().Get("batch"))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		sections, err := a.users.ListSections(r.Context(), batch)
		if err != nil {
			log.Printf("handler: could not list the sections of a batch: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, sectionsResponse{Batch: batch, Items: sections})
	}
}

func (a *Auth) GoogleVerifyEmail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request googleVerifyRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		if !a.cfg.GoogleEnabled() || a.google == nil {
			httpx.WriteError(w, http.StatusBadRequest, MessageGoogleDisabled)
			return
		}
		credential := strings.TrimSpace(request.Credential)
		if credential == "" {
			httpx.WriteError(w, http.StatusBadRequest, MessageGoogleCredential)
			return
		}
		identity, err := a.google.Validate(r.Context(), credential, a.cfg.GoogleClientID)
		if err != nil {
			log.Printf("handler: google refused a sign-in credential: %v", err)
			httpx.WriteError(w, http.StatusBadRequest, googleRefusalMessage(err))
			return
		}
		email := strings.ToLower(strings.TrimSpace(identity.Email))
		if !identity.EmailVerified || identity.HostedDomain != a.cfg.StudentDomain ||
			!strings.HasSuffix(email, a.cfg.StudentEmailSuffix()) {
			log.Printf("handler: google returned an address outside the student domain (verified = %t, hosted domain = %q)",
				identity.EmailVerified, identity.HostedDomain)
			httpx.WriteError(w, http.StatusBadRequest, a.googleDomainMessage())
			return
		}
		proof, expiresAt, err := a.proofs.IssueEmailProof(email)
		if err != nil {
			log.Printf("handler: could not issue an email proof: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, googleVerifyResponse{
			Email:      email,
			EmailProof: proof,
			ExpiresAt:  expiresAt.Format(time.RFC3339),
		})
	}
}

func googleRefusalMessage(err error) string {
	switch {
	case errors.Is(err, googleauth.ErrWrongAudience):
		return MessageGoogleAudience
	case errors.Is(err, googleauth.ErrExpiredToken):
		return MessageGoogleExpired
	case errors.Is(err, googleauth.ErrNotConfigured):
		return MessageGoogleDisabled
	default:
		return MessageGoogleRefused
	}
}

func (a *Auth) googleDomainMessage() string {
	return "Use your " + a.cfg.StudentEmailSuffix() + " Google account."
}
