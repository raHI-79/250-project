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
	"strconv"
	"strings"
)

const MessageAdminActivated = "Your office account is ready. Sign in with your username and password."

type adminInviteResponse struct {
	MaskedEmail string `json:"masked_email"`
}

type acceptAdminInviteRequest struct {
	Token           string `json:"token"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

type adminActivatedResponse struct {
	Username string `json:"username"`
	Message  string `json:"message"`
}

func (a *Auth) AdminInvite() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a.office == nil {
			httpx.WriteError(w, http.StatusBadRequest, MessageInviteInvalid)
			return
		}
		token := strings.TrimSpace(r.URL.Query().Get(fieldToken))
		if token == "" {
			httpx.WriteError(w, http.StatusBadRequest, MessageInviteInvalid)
			return
		}
		invite, err := a.office.FindAdminInvite(r.Context(), hashResetSecret(token))
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusBadRequest, MessageInviteInvalid)
			return
		}
		if err != nil {
			log.Printf("handler: could not load an office admin invitation: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, adminInviteResponse{MaskedEmail: emailHint(invite.Email)})
	}
}

func (a *Auth) AcceptAdminInvite() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if middleware.FromAndroidApp(r) {
			httpx.WriteError(w, http.StatusForbidden, middleware.MessageAppForbidden)
			return
		}
		if a.office == nil {
			httpx.WriteError(w, http.StatusBadRequest, MessageInviteInvalid)
			return
		}
		var request acceptAdminInviteRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		token := strings.TrimSpace(request.Token)
		if token == "" {
			httpx.WriteError(w, http.StatusBadRequest, MessageInviteInvalid)
			return
		}
		invite, err := a.office.FindAdminInvite(r.Context(), hashResetSecret(token))
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusBadRequest, MessageInviteInvalid)
			return
		}
		if err != nil {
			log.Printf("handler: could not load an office admin invitation: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		username, err := validate.UserID(request.Username)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := validate.Password("Password", request.Password, request.ConfirmPassword); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		hash, err := auth.HashPassword(request.Password)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, MessagePasswordTooLong)
			return
		}
		err = a.office.ActivateAdmin(r.Context(), invite.UserID, username, hash)
		switch {
		case errors.Is(err, store.ErrDuplicateIdentifier):
			httpx.WriteError(w, http.StatusConflict, MessageUserIDTaken)
			return
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusBadRequest, MessageInviteInvalid)
			return
		case err != nil:
			log.Printf("handler: could not activate an office admin: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if err := a.office.SetPermissions(r.Context(), invite.UserID,
			auth.DefaultPermissions(), 0); err != nil {
			log.Printf("handler: could not grant the default permissions: %v", err)
		}
		a.events.RecordAs(r, invite.UserID, auth.RoleAdmin, username, store.ActionActivateAdmin,
			"office_admin", idText(invite.UserID), username+" accepted the office admin invitation")
		a.sendUserID(r, invite.Email, username)
		httpx.WriteJSON(w, http.StatusOK, adminActivatedResponse{
			Username: username,
			Message:  MessageAdminActivated,
		})
	}
}

func idText(id int64) string {
	return strconv.FormatInt(id, 10)
}
