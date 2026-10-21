package handler

import (
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strings"
)

const (
	MessageInviteInvalid    = "This invitation link is invalid or has expired."
	MessageInviteMismatch   = "The date of birth does not match this invitation."
	MessageInviteLocked     = "Too many wrong answers. Ask the administration for a new invitation link."
	MessageTeacherActivated = "Account activated. You can log in with your User ID and new password."
	MessageUserIDTaken      = "This User ID is already taken."
	ActivationEmailSubject  = "Your User ID"
	fieldToken              = "token"
	fieldUserID             = "user_id"
	maxInviteAttempts       = 5
)

type inviteResponse struct {
	Name        string `json:"name"`
	Designation string `json:"designation"`
	MaskedEmail string `json:"masked_email"`
}

type userIDAvailableResponse struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

type activatedResponse struct {
	UserID  string `json:"user_id"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

func (a *Auth) Invite() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(r.URL.Query().Get(fieldToken))
		if token == "" {
			httpx.WriteError(w, http.StatusBadRequest, MessageInviteInvalid)
			return
		}
		invite, err := a.users.FindInvite(r.Context(), hashResetSecret(token))
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusBadRequest, MessageInviteInvalid)
			return
		}
		if err != nil {
			log.Printf("handler: could not load a teacher invitation: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, inviteResponse{
			Name:        invite.Name,
			Designation: invite.Designation,
			MaskedEmail: emailHint(invite.Email),
		})
	}
}

func (a *Auth) UserIDAvailable() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		chosenID, err := validate.UserID(r.URL.Query().Get(fieldUserID))
		if err != nil {
			httpx.WriteJSON(w, http.StatusOK, userIDAvailableResponse{Reason: err.Error()})
			return
		}
		taken, err := a.users.IdentifierTaken(r.Context(), chosenID)
		if err != nil {
			log.Printf("handler: could not check a user id: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if taken {
			httpx.WriteJSON(w, http.StatusOK, userIDAvailableResponse{Reason: MessageUserIDTaken})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, userIDAvailableResponse{Available: true})
	}
}

func (a *Auth) Activate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := a.photos.Parse(r); err != nil {
			status, message := photoError(err)
			httpx.WriteError(w, status, message)
			return
		}
		token := strings.TrimSpace(r.FormValue(fieldToken))
		if token == "" {
			httpx.WriteError(w, http.StatusBadRequest, MessageInviteInvalid)
			return
		}
		invite, err := a.users.FindInvite(r.Context(), hashResetSecret(token))
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusBadRequest, MessageInviteInvalid)
			return
		}
		if err != nil {
			log.Printf("handler: could not load a teacher invitation: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		birthDate := strings.TrimSpace(r.FormValue(fieldDateOfBirth))
		if birthDate == "" || birthDate != invite.DateOfBirth {
			a.failedInvite(w, r, invite.UserID)
			return
		}
		chosenID, err := validate.UserID(r.FormValue(fieldUserID))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		password := r.FormValue(fieldPassword)
		if err := validate.Password("Password", password, r.FormValue(fieldConfirmPassword)); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		hash, err := auth.HashPassword(password)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, MessagePasswordTooLong)
			return
		}
		photoPath, err := a.photos.Save(r)
		if err != nil {
			status, message := photoError(err)
			httpx.WriteError(w, status, message)
			return
		}
		err = a.users.ActivateTeacher(r.Context(), invite.UserID, chosenID, hash, photoPath)
		if err != nil {
			a.removePhoto(photoPath)
			switch {
			case errors.Is(err, store.ErrDuplicateIdentifier):
				httpx.WriteError(w, http.StatusConflict, MessageUserIDTaken)
			case errors.Is(err, store.ErrNotFound):
				httpx.WriteError(w, http.StatusBadRequest, MessageInviteInvalid)
			default:
				log.Printf("handler: could not activate a teacher: %v", err)
				httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			}
			return
		}
		if photoPath != "" && invite.PhotoPath != "" && invite.PhotoPath != photoPath {
			a.removePhoto(invite.PhotoPath)
		}
		a.sendUserID(r, invite.Email, chosenID)
		httpx.WriteJSON(w, http.StatusOK, activatedResponse{
			UserID:  chosenID,
			Name:    invite.Name,
			Message: MessageTeacherActivated,
		})
	}
}

func (a *Auth) failedInvite(w http.ResponseWriter, r *http.Request, userID int64) {
	attempts, err := a.users.CountInviteAttempt(r.Context(), userID)
	if err != nil {
		log.Printf("handler: could not count an invitation attempt: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return
	}
	if attempts >= maxInviteAttempts {
		if err := a.users.ClearInvite(r.Context(), userID); err != nil {
			log.Printf("handler: could not clear the invitation of user %d: %v", userID, err)
		}
		httpx.WriteError(w, http.StatusBadRequest, MessageInviteLocked)
		return
	}
	httpx.WriteError(w, http.StatusBadRequest, MessageInviteMismatch)
}

func (a *Auth) sendUserID(r *http.Request, email, chosenID string) {
	if !a.mailerReady() {
		return
	}
	if err := a.mail.Send(r.Context(), email, ActivationEmailSubject, userIDEmailBody(chosenID)); err != nil {
		log.Printf("handler: could not email a user id: %v", err)
	}
}

func userIDEmailBody(chosenID string) string {
	return "Your account is ready. Your User ID is " + chosenID + ".\n\n" +
		"Sign in with this User ID or your official email address and the password you chose."
}
