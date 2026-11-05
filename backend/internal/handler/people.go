package handler

import (
	"context"
	"encoding/csv"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	MessagePersonNotFound  = "Account not found."
	MessagePersonSuspended = "Account suspended."
	MessagePersonRestored  = "Account restored."
	MessagePersonSignedOut = "Signed out everywhere."
	MessagePhoneReset      = "The attendance phone was reset."
	MessageResetSent       = "A password reset email is on its way."
	MessageRoleChanged     = "Role changed."
	MessageBulkDone        = "Done."
	MessageNoSelection     = "Select at least one account."
	MessageUnknownRole     = "The role must be student or cr."
	MessageProtectedChief  = "The super admin account cannot be changed here."
	peopleExportName       = "users.csv"
	maxBulkPeople          = 100
)

type PeopleStore interface {
	ListPeople(ctx context.Context, filter store.PersonFilter) ([]store.PersonRow, int, error)
	FindPersonDetail(ctx context.Context, userID int64) (store.PersonDetail, error)
	SuspendPerson(ctx context.Context, userID int64) (string, error)
	RestorePerson(ctx context.Context, userID int64) (string, error)
	SignOutPerson(ctx context.Context, userID int64) error
	ResetPersonDevice(ctx context.Context, userID int64) error
	SetCR(ctx context.Context, userID int64, isCR bool) error
	ListSessions(ctx context.Context, userID int64) ([]store.Session, error)
	ListEvents(ctx context.Context, filter store.AuditFilter) ([]store.AuditEntry, error)
	FindResetTarget(ctx context.Context, identifier string) (store.ResetTarget, error)
}

type People struct {
	cfg    *config.Config
	people PeopleStore
	reset  func(w http.ResponseWriter, r *http.Request, identifier string) bool
	events *Recorder
}

func NewPeople(cfg *config.Config, people PeopleStore, events *Recorder) *People {
	return &People{cfg: cfg, people: people, events: events}
}

type accountItem struct {
	ID         int64  `json:"id"`
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
	Email      string `json:"email,omitempty"`
	Phone      string `json:"phone,omitempty"`
	Role       string `json:"role"`
	Status     string `json:"status"`
	Batch      string `json:"batch,omitempty"`
	Section    string `json:"section,omitempty"`
	LastSignIn string `json:"last_sign_in,omitempty"`
	CreatedAt  string `json:"created_at"`
}

type accountsResponse struct {
	Items []accountItem `json:"items"`
	Total int           `json:"total"`
}

type accountDetailResponse struct {
	Person   accountItem    `json:"person"`
	Sessions []sessionItem  `json:"sessions"`
	Activity []activityItem `json:"activity"`
	Device   string         `json:"device,omitempty"`
	TwoStep  bool           `json:"two_step"`
	Counts   map[string]int `json:"counts,omitempty"`
}

type bulkRequest struct {
	IDs    []int64 `json:"ids"`
	Action string  `json:"action"`
}

type roleRequest struct {
	Role string `json:"role"`
}

func personRole(row store.PersonRow) string {
	return auth.EffectiveRole(row.Role, row.IsCR)
}

func accountItemOf(row store.PersonRow) accountItem {
	item := accountItem{
		ID:         row.ID,
		Identifier: row.Identifier,
		Name:       row.Name,
		Email:      row.Email,
		Phone:      row.Phone,
		Role:       personRole(row),
		Status:     row.Status,
		Batch:      row.Batch,
		Section:    row.Section,
		CreatedAt:  timestamp(row.CreatedAt),
	}
	if row.LastSignIn != nil {
		item.LastSignIn = row.LastSignIn.Format(time.RFC3339)
	}
	return item
}

func (p *People) filterFrom(r *http.Request) store.PersonFilter {
	query := r.URL.Query()
	filter := store.PersonFilter{
		Search: strings.TrimSpace(query.Get("search")),
		Role:   strings.TrimSpace(query.Get("role")),
		Status: strings.TrimSpace(query.Get("status")),
		Batch:  strings.TrimSpace(query.Get("batch")),
	}
	if limit, err := strconv.Atoi(strings.TrimSpace(query.Get("limit"))); err == nil {
		filter.Limit = limit
	}
	if offset, err := strconv.Atoi(strings.TrimSpace(query.Get("offset"))); err == nil && offset > 0 {
		filter.Offset = offset
	}
	return filter
}

func (p *People) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, total, err := p.people.ListPeople(r.Context(), p.filterFrom(r))
		if err != nil {
			log.Printf("handler: could not list the accounts: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]accountItem, 0, len(rows))
		for _, row := range rows {
			items = append(items, accountItemOf(row))
		}
		httpx.WriteJSON(w, http.StatusOK, accountsResponse{Items: items, Total: total})
	}
}

func (p *People) Export() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter := p.filterFrom(r)
		filter.Limit = store.MaxStudentPage
		rows := make([][]string, 0, 64)
		rows = append(rows, []string{"Name", "Identifier", "Role", "Status", "Batch", "Section",
			"Email", "Phone", "Last sign-in", "Created"})
		for page := 0; page < 20; page++ {
			people, _, err := p.people.ListPeople(r.Context(), filter)
			if err != nil {
				log.Printf("handler: could not export the accounts: %v", err)
				httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
				return
			}
			for _, row := range people {
				last := ""
				if row.LastSignIn != nil {
					last = row.LastSignIn.In(p.cfg.Location).Format(time.RFC3339)
				}
				rows = append(rows, []string{
					row.Name, row.Identifier, personRole(row), row.Status, row.Batch, row.Section,
					row.Email, row.Phone, last, row.CreatedAt.In(p.cfg.Location).Format(time.RFC3339),
				})
			}
			if len(people) < filter.Limit {
				break
			}
			filter.Offset += len(people)
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+peopleExportName+`"`)
		writer := csv.NewWriter(w)
		if err := writer.WriteAll(rows); err != nil {
			log.Printf("handler: could not write the account export: %v", err)
		}
	}
}

func (p *People) Detail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := pathID(w, r)
		if !ok {
			return
		}
		detail, err := p.people.FindPersonDetail(r.Context(), userID)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, MessagePersonNotFound)
			return
		}
		if err != nil {
			log.Printf("handler: could not load an account: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		sessions, err := p.people.ListSessions(r.Context(), userID)
		if err != nil {
			log.Printf("handler: could not list the sessions of an account: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		events, err := p.people.ListEvents(r.Context(), store.AuditFilter{ActorID: userID, Limit: 20})
		if err != nil {
			log.Printf("handler: could not list the activity of an account: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]sessionItem, 0, len(sessions))
		for _, session := range sessions {
			country := session.Country
			if country == "" {
				country = "Unknown"
			}
			items = append(items, sessionItem{
				ID:         session.ID,
				Device:     session.Device,
				Client:     session.Client,
				Country:    country,
				LastSeenAt: session.LastSeenAt.Format(time.RFC3339),
				CreatedAt:  session.CreatedAt.Format(time.RFC3339),
			})
		}
		body := accountDetailResponse{
			Person:   accountItemOf(detail.PersonRow),
			Sessions: items,
			Activity: activityItems(events),
			TwoStep:  detail.TwoStep,
		}
		if detail.DeviceID != "" {
			body.Device = "Bound"
		}
		httpx.WriteJSON(w, http.StatusOK, body)
	}
}

func (p *People) Suspend() http.HandlerFunc {
	return p.change(store.ActionSuspendUser, MessagePersonSuspended,
		func(ctx context.Context, userID int64) (string, error) {
			return p.people.SuspendPerson(ctx, userID)
		})
}

func (p *People) Restore() http.HandlerFunc {
	return p.change(store.ActionRestoreUser, MessagePersonRestored,
		func(ctx context.Context, userID int64) (string, error) {
			return p.people.RestorePerson(ctx, userID)
		})
}

func (p *People) SignOut() http.HandlerFunc {
	return p.change(store.ActionSessionOutAll, MessagePersonSignedOut,
		func(ctx context.Context, userID int64) (string, error) {
			return "", p.people.SignOutPerson(ctx, userID)
		})
}

func (p *People) ResetPhone() http.HandlerFunc {
	return p.change(store.ActionResetPhone, MessagePhoneReset,
		func(ctx context.Context, userID int64) (string, error) {
			return "", p.people.ResetPersonDevice(ctx, userID)
		})
}

func (p *People) change(action, message string,
	apply func(ctx context.Context, userID int64) (string, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := pathID(w, r)
		if !ok {
			return
		}
		detail, err := p.people.FindPersonDetail(r.Context(), userID)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, MessagePersonNotFound)
			return
		}
		if err != nil {
			log.Printf("handler: could not load an account: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if detail.Role == auth.RoleSuperAdmin {
			httpx.WriteError(w, http.StatusForbidden, MessageProtectedChief)
			return
		}
		if _, err := apply(r.Context(), userID); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				httpx.WriteError(w, http.StatusConflict, MessagePersonNotFound)
				return
			}
			log.Printf("handler: could not change an account: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		p.events.Record(r, action, "account", idText(userID), message+" "+detail.Identifier)
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: message})
	}
}

func (p *People) SetRole() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := pathID(w, r)
		if !ok {
			return
		}
		var request roleRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		role := strings.ToLower(strings.TrimSpace(request.Role))
		if role != auth.RoleStudent && role != auth.RoleCR {
			httpx.WriteError(w, http.StatusBadRequest, MessageUnknownRole)
			return
		}
		err := p.people.SetCR(r.Context(), userID, role == auth.RoleCR)
		if errors.Is(err, store.ErrNotAStudent) {
			httpx.WriteError(w, http.StatusConflict, MessageUnknownRole)
			return
		}
		if err != nil {
			log.Printf("handler: could not change the role of an account: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		p.events.Record(r, store.ActionRoleChange, "account", idText(userID),
			"Role set to "+role+" for account "+idText(userID))
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageRoleChanged})
	}
}

func (p *People) Bulk() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request bulkRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		if len(request.IDs) == 0 {
			httpx.WriteError(w, http.StatusBadRequest, MessageNoSelection)
			return
		}
		if len(request.IDs) > maxBulkPeople {
			httpx.WriteError(w, http.StatusBadRequest, "Select at most 100 accounts.")
			return
		}
		action := strings.ToLower(strings.TrimSpace(request.Action))
		if action != "suspend" && action != "sign_out" {
			httpx.WriteError(w, http.StatusBadRequest, "The action must be suspend or sign_out.")
			return
		}
		done := 0
		for _, userID := range request.IDs {
			detail, err := p.people.FindPersonDetail(r.Context(), userID)
			if err != nil || detail.Role == auth.RoleSuperAdmin {
				continue
			}
			if action == "suspend" {
				if _, err := p.people.SuspendPerson(r.Context(), userID); err != nil {
					continue
				}
			} else if err := p.people.SignOutPerson(r.Context(), userID); err != nil {
				continue
			}
			done++
		}
		event := store.ActionSuspendUser
		if action == "sign_out" {
			event = store.ActionSessionOutAll
		}
		p.events.Record(r, event, "account", "",
			"Bulk "+action+" on "+strconv.Itoa(done)+" accounts")
		httpx.WriteJSON(w, http.StatusOK, messageResponse{
			Message: MessageBulkDone + " " + strconv.Itoa(done) + " accounts changed.",
		})
	}
}
