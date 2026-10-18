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
	MessageInviteMismatch   = "The teacher ID and date of birth do not match this invitation."
	MessageInviteLocked     = "Too many wrong answers. Ask the administration for a new invitation link."
	MessageTeacherActivated = "Account activated. You can log in with your teacher ID and new password."
	fieldToken              = "token"
	fieldTeacherID          = "teacher_id"
	maxInviteAttempts       = 5
)

type inviteResponse struct {
	Name        string `json:"name"`
	Designation string `json:"designation"`
	MaskedEmail string `json:"masked_email"`
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
		teacherID := strings.TrimSpace(r.FormValue(fieldTeacherID))
		birthDate := strings.TrimSpace(r.FormValue(fieldDateOfBirth))
		if teacherID != invite.TeacherID || birthDate == "" || birthDate != invite.DateOfBirth {
			attempts, err := a.users.CountInviteAttempt(r.Context(), invite.UserID)
			if err != nil {
				log.Printf("handler: could not count an invitation attempt: %v", err)
				httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
				return
			}
			if attempts >= maxInviteAttempts {
				if err := a.users.ClearInvite(r.Context(), invite.UserID); err != nil {
					log.Printf("handler: could not clear the invitation of user %d: %v", invite.UserID, err)
				}
				httpx.WriteError(w, http.StatusBadRequest, MessageInviteLocked)
				return
			}
			httpx.WriteError(w, http.StatusBadRequest, MessageInviteMismatch)
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
		if err := a.users.ActivateTeacher(r.Context(), invite.UserID, hash, photoPath); err != nil {
			a.removePhoto(photoPath)
			if errors.Is(err, store.ErrNotFound) {
				httpx.WriteError(w, http.StatusBadRequest, MessageInviteInvalid)
				return
			}
			log.Printf("handler: could not activate a teacher: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if photoPath != "" && invite.PhotoPath != "" && invite.PhotoPath != photoPath {
			a.removePhoto(invite.PhotoPath)
		}
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageTeacherActivated})
	}
}
