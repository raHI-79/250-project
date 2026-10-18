package handler

import (
	"errors"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strings"
)

type promoteCRRequest struct {
	UserID int64 `json:"user_id"`
}

type createdResponse struct {
	ID      int64  `json:"id"`
	Message string `json:"message"`
}

type crsResponse struct {
	Items []studentItem `json:"items"`
}

func (a *Admin) PromoteCR() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request promoteCRRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		if request.UserID <= 0 {
			httpx.WriteError(w, http.StatusBadRequest, MessageInvalidUserID)
			return
		}
		promoted, err := a.people.PromoteCR(r.Context(), request.UserID)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageActiveStudentNotFound)
			return
		case errors.Is(err, store.ErrAlreadyCR):
			httpx.WriteError(w, http.StatusConflict, MessageAlreadyCR)
			return
		case err != nil:
			log.Printf("handler: could not promote a class representative: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		title := "You are now the Class Representative of " + promoted.Batch + " section " + promoted.Section
		if err := a.alerts.NotifyUser(r.Context(), promoted.UserID, notify.TypeDecision, title, ""); err != nil {
			log.Printf("handler: could not notify a new class representative: %v", err)
		}
		httpx.WriteJSON(w, http.StatusOK, createdResponse{ID: promoted.UserID, Message: MessageCRPromoted})
	}
}

func (a *Admin) CRs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		filter := store.CRFilter{Search: strings.TrimSpace(query.Get("search"))}
		if batch := strings.TrimSpace(query.Get("batch")); batch != "" {
			value, err := validate.Batch("batch", batch)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			filter.Batch = value
		}
		crs, err := a.people.ListCRs(r.Context(), filter)
		if err != nil {
			log.Printf("handler: could not list the class representatives: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, crsResponse{Items: studentItems(crs)})
	}
}

func (a *Admin) DemoteCR() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := pathID(w, r)
		if !ok {
			return
		}
		err := a.people.DemoteCR(r.Context(), userID)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageStudentNotFound)
		case errors.Is(err, store.ErrNotCR):
			httpx.WriteError(w, http.StatusConflict, MessageNotCR)
		case err != nil:
			log.Printf("handler: could not demote a class representative: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageCRDemoted})
		}
	}
}

func (a *Admin) DeactivateCR() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := pathID(w, r)
		if !ok {
			return
		}
		err := a.people.DeactivateCR(r.Context(), userID)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageCRNotFound)
		case errors.Is(err, store.ErrAlreadyRemoved):
			httpx.WriteError(w, http.StatusConflict, MessageAlreadyRemoved)
		case err != nil:
			log.Printf("handler: could not deactivate a class representative: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageCRDeactivated})
		}
	}
}
