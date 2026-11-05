package handler

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"net/http"
	"time"
)

const (
	MessageApprovalAsked   = "Another office admin has to approve this. The request waits 24 hours."
	MessageApprovalDone    = "Approved and done."
	MessageApprovalNo      = "Request declined."
	MessageApprovalGone    = "That request is no longer waiting."
	MessageApprovalOwn     = "Another admin has to approve your own request."
	MessageApprovalUnknown = "That request cannot be replayed."
	ApprovalWindow         = 24 * time.Hour
	ActionDeleteUser       = "delete_user"
	ActionDeleteSheet      = "delete_sheet"
	ActionRemoveAdmin      = "remove_admin"
)

type ApprovalStore interface {
	CreateApproval(ctx context.Context, approval store.Approval) (int64, error)
	ListApprovals(ctx context.Context) ([]store.Approval, error)
	LoadApproval(ctx context.Context, approvalID int64) (store.Approval, error)
	DecideApproval(ctx context.Context, approvalID, deciderID int64, status string) (store.Approval, error)
	OtherOfficeAdmins(ctx context.Context, userID int64) (int, error)
}

type Approvals struct {
	cfg     *config.Config
	store   ApprovalStore
	events  *Recorder
	runners map[string]func(ctx context.Context, approval store.Approval) error
}

func NewApprovals(cfg *config.Config, people ApprovalStore, events *Recorder) *Approvals {
	return &Approvals{cfg: cfg, store: people, events: events,
		runners: make(map[string]func(ctx context.Context, approval store.Approval) error)}
}

func (a *Approvals) Handle(action string, run func(ctx context.Context, approval store.Approval) error) {
	a.runners[action] = run
}

type approvalItem struct {
	ID        int64  `json:"id"`
	Action    string `json:"action"`
	Summary   string `json:"summary"`
	Requester string `json:"requester"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
	DecidedBy string `json:"decided_by,omitempty"`
	Mine      bool   `json:"mine"`
}

type approvalsResponse struct {
	Items []approvalItem `json:"items"`
}

func (a *Approvals) Needed(w http.ResponseWriter, r *http.Request, action, targetType,
	targetID, summary string, payload any) bool {
	current, ok := middleware.CurrentUser(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
		return true
	}
	if a == nil || a.store == nil {
		return false
	}
	if current.Role == auth.RoleSuperAdmin {
		a.events.Record(r, store.ActionOverride, targetType, targetID,
			current.Identifier+" went ahead without a second admin: "+summary)
		return false
	}
	others, err := a.store.OtherOfficeAdmins(r.Context(), current.ID)
	if err != nil {
		log.Printf("handler: could not count the other office admins: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return true
	}
	if others == 0 {
		return false
	}
	body := ""
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err == nil {
			body = string(encoded)
		}
	}
	request := store.Approval{
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		Summary:     summary,
		Payload:     body,
		RequestedBy: current.ID,
		ExpiresAt:   a.cfg.Now().Add(ApprovalWindow),
	}
	id, err := a.store.CreateApproval(r.Context(), request)
	if err != nil {
		log.Printf("handler: could not record an approval request: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return true
	}
	a.events.Record(r, store.ActionApprovalAsked, targetType, targetID,
		current.Identifier+" asked another admin to approve: "+summary)
	httpx.WriteJSON(w, http.StatusAccepted, approvalAskedResponse{
		Message:    MessageApprovalAsked,
		ApprovalID: id,
	})
	return true
}

type approvalAskedResponse struct {
	Message    string `json:"message"`
	ApprovalID int64  `json:"approval_id"`
}

func (a *Approvals) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, _ := middleware.CurrentUser(r.Context())
		approvals, err := a.store.ListApprovals(r.Context())
		if err != nil {
			log.Printf("handler: could not list the approval requests: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]approvalItem, 0, len(approvals))
		for _, approval := range approvals {
			items = append(items, approvalItem{
				ID:        approval.ID,
				Action:    approval.Action,
				Summary:   approval.Summary,
				Requester: approval.Requester,
				Status:    approval.Status,
				CreatedAt: approval.CreatedAt.Format(time.RFC3339),
				ExpiresAt: approval.ExpiresAt.Format(time.RFC3339),
				DecidedBy: approval.DecidedBy,
				Mine:      approval.RequestedBy == current.ID,
			})
		}
		httpx.WriteJSON(w, http.StatusOK, approvalsResponse{Items: items})
	}
}

func (a *Approvals) Decide(approve bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		approvalID, ok := pathID(w, r)
		if !ok {
			return
		}
		current, _ := middleware.CurrentUser(r.Context())
		status := store.ApprovalDeclined
		if approve {
			status = store.ApprovalApproved
		}
		approval, err := a.store.DecideApproval(r.Context(), approvalID, current.ID, status)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageApprovalGone)
			return
		case errors.Is(err, store.ErrOwnApproval):
			httpx.WriteError(w, http.StatusForbidden, MessageApprovalOwn)
			return
		case errors.Is(err, store.ErrApprovalGone):
			httpx.WriteError(w, http.StatusConflict, MessageApprovalGone)
			return
		case err != nil:
			log.Printf("handler: could not decide an approval request: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if !approve {
			a.events.Record(r, store.ActionApprovalNo, approval.TargetType, approval.TargetID,
				current.Identifier+" declined: "+approval.Summary)
			httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageApprovalNo})
			return
		}
		run, known := a.runners[approval.Action]
		if !known {
			httpx.WriteError(w, http.StatusConflict, MessageApprovalUnknown)
			return
		}
		if err := run(r.Context(), approval); err != nil {
			log.Printf("handler: could not carry out an approved request: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		a.events.Record(r, store.ActionApprovalDone, approval.TargetType, approval.TargetID,
			current.Identifier+" approved: "+approval.Summary)
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageApprovalDone})
	}
}
