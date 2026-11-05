package handler

import (
	"errors"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	MessageTwoStepSent    = "We sent a 6-digit code to the email address on your account."
	MessageTwoStepInvalid = "Invalid or expired code."
	MessageTwoStepLocked  = "Too many wrong codes. Please sign in again."
	MessageTwoStepUnknown = "Please sign in again."
	TwoStepEmailSubject   = "Your sign-in code"
	twoStepLifetime       = 10 * time.Minute
	maxTwoStepAttempts    = 5
)

type twoStepLoginRequest struct {
	Identifier string `json:"identifier"`
	Code       string `json:"code"`
	Pending    string `json:"pending"`
}

func (a *Auth) twoStepStart(w http.ResponseWriter, r *http.Request, user store.LoginUser,
	email string) bool {
	code, err := newResetCode()
	if err != nil {
		log.Printf("handler: could not generate a two-step code: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return false
	}
	now := a.cfg.Now()
	if a.office == nil {
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return false
	}
	err = a.office.StartTwoStep(r.Context(), user.ID, hashResetSecret(code), now.Add(twoStepLifetime), now)
	if err != nil {
		log.Printf("handler: could not store a two-step code: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return false
	}
	pending, _, err := a.proofs.IssueEmailProof(strings.ToLower(user.Identifier))
	if err != nil {
		log.Printf("handler: could not issue a two-step pass: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return false
	}
	body := loginResponse{
		TwoStep:   true,
		Pending:   pending,
		Message:   MessageTwoStepSent,
		EmailHint: emailHint(email),
	}
	if a.mailerReady() {
		if err := a.mail.Send(r.Context(), email, TwoStepEmailSubject, twoStepEmailBody(code)); err != nil {
			log.Printf("handler: could not email a two-step code: %v", err)
		}
	} else if !a.cfg.IsProduction() {
		body.DevCode = code
	}
	a.events.RecordAs(r, user.ID, user.Role, user.Identifier, store.ActionTwoStepSent, "session",
		idText(user.ID), user.Identifier+" was sent a sign-in code")
	httpx.WriteJSON(w, http.StatusOK, body)
	return true
}

func (a *Auth) TwoStepLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if middleware.FromAndroidApp(r) {
			httpx.WriteError(w, http.StatusForbidden, middleware.MessageAppForbidden)
			return
		}
		if a.office == nil {
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		var request twoStepLoginRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		identifier, err := validate.Required("ID", request.Identifier)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if a.proofs.CheckEmailProof(strings.TrimSpace(request.Pending),
			strings.ToLower(identifier)) != nil {
			httpx.WriteError(w, http.StatusBadRequest, MessageTwoStepUnknown)
			return
		}
		code := strings.TrimSpace(request.Code)
		if !isSixDigitCode(code) {
			httpx.WriteError(w, http.StatusBadRequest, MessageTwoStepInvalid)
			return
		}
		challenge, err := a.office.TwoStepChallenge(r.Context(), identifier)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusBadRequest, MessageTwoStepUnknown)
			return
		}
		if err != nil {
			log.Printf("handler: could not load a two-step challenge: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		now := a.cfg.Now()
		if !challenge.Enabled || challenge.CodeHash == "" || challenge.ExpiresAt == nil ||
			!challenge.ExpiresAt.After(now) {
			httpx.WriteError(w, http.StatusBadRequest, MessageTwoStepInvalid)
			return
		}
		if challenge.CodeHash != hashResetSecret(code) {
			attempts, err := a.office.CountTwoStepAttempt(r.Context(), challenge.UserID)
			if err != nil {
				log.Printf("handler: could not count a two-step attempt: %v", err)
				httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
				return
			}
			a.events.RecordAs(r, challenge.UserID, challenge.Role, identifier,
				store.ActionTwoStepFailed, "session", idText(challenge.UserID),
				"Wrong sign-in code for "+identifier)
			if attempts >= maxTwoStepAttempts {
				if err := a.office.ClearTwoStep(r.Context(), challenge.UserID); err != nil {
					log.Printf("handler: could not clear a two-step code: %v", err)
				}
				httpx.WriteError(w, http.StatusBadRequest, MessageTwoStepLocked)
				return
			}
			httpx.WriteError(w, http.StatusBadRequest, MessageTwoStepInvalid)
			return
		}
		if err := a.office.ClearTwoStep(r.Context(), challenge.UserID); err != nil {
			log.Printf("handler: could not clear a two-step code: %v", err)
		}
		user, err := a.users.FindLoginUser(r.Context(), identifier)
		if err != nil {
			log.Printf("handler: could not load a user after a two-step code: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if !user.IsActive() {
			httpx.WriteError(w, http.StatusUnauthorized, MessageInvalidLogin)
			return
		}
		a.finishLogin(w, r, user, loginOfficeRole(user))
	}
}

func loginOfficeRole(user store.LoginUser) string {
	if user.Role == store.RoleSuperAdmin {
		return store.RoleSuperAdmin
	}
	return store.RoleAdmin
}

func twoStepEmailBody(code string) string {
	return "Your sign-in code is " + code + ". It expires in 10 minutes.\n\n" +
		"If you did not try to sign in, change your password now."
}
