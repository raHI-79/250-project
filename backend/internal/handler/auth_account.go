package handler

import (
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	MessageGoogleDisabled   = "Google sign-in is not configured."
	MessageGoogleCredential = "credential is required."
)

type authConfigResponse struct {
	StudentEmailDomain string `json:"student_email_domain"`
	GoogleEnabled      bool   `json:"google_enabled"`
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
		httpx.WriteJSON(w, http.StatusOK, authConfigResponse{
			StudentEmailDomain: a.cfg.StudentDomain,
			GoogleEnabled:      a.cfg.GoogleEnabled(),
		})
	}
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
			httpx.WriteError(w, http.StatusBadRequest, a.googleDomainMessage())
			return
		}
		email := strings.ToLower(strings.TrimSpace(identity.Email))
		if !identity.EmailVerified || identity.HostedDomain != a.cfg.StudentDomain ||
			!strings.HasSuffix(email, a.cfg.StudentEmailSuffix()) {
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

func (a *Auth) googleDomainMessage() string {
	return "Use your " + a.cfg.StudentEmailSuffix() + " Google account."
}
