package handler

import (
	"errors"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"net/http"
	"strings"
)

type permanentDeleteRequest struct {
	Confirm string `json:"confirm"`
}

type auditItem struct {
	ID            int64  `json:"id"`
	AdminID       int64  `json:"admin_id,omitempty"`
	AdminName     string `json:"admin_name"`
	Action        string `json:"action"`
	TargetSummary string `json:"target_summary"`
	CreatedAt     string `json:"created_at"`
}

type auditResponse struct {
	Items []auditItem `json:"items"`
}

func (a *Admin) Restore() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := pathID(w, r)
		if !ok {
			return
		}
		account, err := a.people.FindAccount(r.Context(), userID)
		if !a.accountLoaded(w, account, err) {
			return
		}
		status, err := a.people.RestoreAccount(r.Context(), userID)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageAccountNotFound)
			return
		case errors.Is(err, store.ErrNotRestorable):
			httpx.WriteError(w, http.StatusConflict, MessageNotRestorable)
			return
		case err != nil:
			log.Printf("handler: could not restore an account: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		a.audit(r, store.ActionRestore, accountSummary(account)+" is now "+status)
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageAccountRestored})
	}
}

func (a *Admin) PermanentDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := pathID(w, r)
		if !ok {
			return
		}
		var request permanentDeleteRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		account, err := a.people.FindAccount(r.Context(), userID)
		if !a.accountLoaded(w, account, err) {
			return
		}
		if account.Role == store.RoleAdmin || account.ID == current.ID {
			httpx.WriteError(w, http.StatusForbidden, MessageProtectedAccount)
			return
		}
		if account.Status != store.StatusRemoved && account.Status != store.StatusRejected {
			httpx.WriteError(w, http.StatusConflict, MessageNotDeletable)
			return
		}
		if !confirmsAccount(request.Confirm, account) {
			httpx.WriteError(w, http.StatusBadRequest, MessageConfirmMismatch)
			return
		}
		summary := accountSummary(account)
		photoPath, err := a.people.DeleteAccount(r.Context(), userID)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageAccountNotFound)
			return
		case errors.Is(err, store.ErrProtected):
			httpx.WriteError(w, http.StatusForbidden, MessageProtectedAccount)
			return
		case errors.Is(err, store.ErrNotDeletable):
			httpx.WriteError(w, http.StatusConflict, MessageNotDeletable)
			return
		case err != nil:
			log.Printf("handler: could not delete an account: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		a.removePhoto(photoPath)
		a.audit(r, store.ActionDelete, summary)
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageAccountDeleted})
	}
}

func (a *Admin) Audit() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries, err := a.people.ListAudit(r.Context(), store.DefaultAuditLimit)
		if err != nil {
			log.Printf("handler: could not list the audit entries: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]auditItem, 0, len(entries))
		for _, entry := range entries {
			items = append(items, auditItem{
				ID:            entry.ID,
				AdminID:       entry.AdminID,
				AdminName:     entry.AdminName,
				Action:        entry.Action,
				TargetSummary: entry.TargetSummary,
				CreatedAt:     timestamp(entry.CreatedAt),
			})
		}
		httpx.WriteJSON(w, http.StatusOK, auditResponse{Items: items})
	}
}

func (a *Admin) accountLoaded(w http.ResponseWriter, account store.AccountRef, err error) bool {
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, MessageAccountNotFound)
		return false
	}
	if err != nil {
		log.Printf("handler: could not load an account: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return false
	}
	return account.ID > 0
}

func (a *Admin) audit(r *http.Request, action, targetSummary string) {
	current, ok := middleware.CurrentUser(r.Context())
	if !ok {
		return
	}
	if err := a.people.WriteAudit(r.Context(), current.ID, action, targetSummary); err != nil {
		log.Printf("handler: could not write an audit entry: %v", err)
	}
}

func (a *Admin) auditRemoval(r *http.Request, userID int64) {
	account, err := a.people.FindAccount(r.Context(), userID)
	if err != nil {
		log.Printf("handler: could not load the removed account %d: %v", userID, err)
		return
	}
	a.audit(r, store.ActionRemove, accountSummary(account))
}

func accountSummary(account store.AccountRef) string {
	label := account.Identifier
	if label == "" {
		label = account.Email
	}
	name := account.Name
	if name == "" {
		name = label
	}
	role := account.Role
	if account.IsCR {
		role = "class representative"
	}
	if label == "" {
		return role + " " + name
	}
	return role + " " + name + " (" + label + ")"
}

func confirmsAccount(confirm string, account store.AccountRef) bool {
	typed := strings.ToLower(strings.TrimSpace(confirm))
	if typed == "" {
		return false
	}
	if account.Identifier != "" && typed == strings.ToLower(account.Identifier) {
		return true
	}
	return account.Email != "" && typed == strings.ToLower(account.Email)
}
