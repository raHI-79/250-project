package handler

import (
	"context"
	"encoding/csv"
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
	MessageBadDate   = "A date must look like 2026-01-31."
	exportFileName   = "activity-log.csv"
	exportLimit      = store.MaxEventPage
	exportPages      = 20
	monitorPageLimit = 100
)

type AuditStore interface {
	ListEvents(ctx context.Context, filter store.AuditFilter) ([]store.AuditEntry, error)
	CountActions(ctx context.Context, actorID int64, from, to *time.Time) ([]store.ActionCount, error)
	ListAuditPeople(ctx context.Context) ([]store.AuditPerson, error)
}

type Activity struct {
	cfg   *config.Config
	trail AuditStore
}

func NewActivity(cfg *config.Config, trail AuditStore) *Activity {
	return &Activity{cfg: cfg, trail: trail}
}

type activityItem struct {
	ID         int64  `json:"id"`
	ActorID    int64  `json:"actor_id"`
	Actor      string `json:"actor"`
	Role       string `json:"role"`
	Action     string `json:"action"`
	TargetType string `json:"target_type,omitempty"`
	TargetID   string `json:"target_id,omitempty"`
	Summary    string `json:"summary"`
	Client     string `json:"client"`
	CreatedAt  string `json:"created_at"`
}

type activityResponse struct {
	Items  []activityItem `json:"items"`
	Cursor int64          `json:"cursor,omitempty"`
}

type actionCountItem struct {
	Action string `json:"action"`
	Total  int    `json:"total"`
}

type monitorResponse struct {
	Items  []activityItem    `json:"items"`
	Counts []actionCountItem `json:"counts"`
}

type personItem struct {
	ID    int64  `json:"id"`
	Label string `json:"label"`
	Role  string `json:"role"`
	Total int    `json:"total"`
}

type peopleResponse struct {
	Items []personItem `json:"items"`
}

func (a *Activity) filterFrom(r *http.Request) (store.AuditFilter, error) {
	query := r.URL.Query()
	filter := store.AuditFilter{
		Role:   strings.TrimSpace(query.Get("role")),
		Action: strings.TrimSpace(query.Get("action")),
		Client: strings.TrimSpace(query.Get("client")),
		Search: strings.TrimSpace(query.Get("search")),
	}
	if person := strings.TrimSpace(query.Get("person")); person != "" {
		id, err := strconv.ParseInt(person, 10, 64)
		if err != nil || id < 0 {
			return store.AuditFilter{}, errInvalidID
		}
		filter.ActorID = id
	}
	if before := strings.TrimSpace(query.Get("before")); before != "" {
		id, err := strconv.ParseInt(before, 10, 64)
		if err != nil || id < 0 {
			return store.AuditFilter{}, errInvalidID
		}
		filter.Before = id
	}
	from, err := a.dayFrom(query.Get("from"), false)
	if err != nil {
		return store.AuditFilter{}, err
	}
	to, err := a.dayFrom(query.Get("to"), true)
	if err != nil {
		return store.AuditFilter{}, err
	}
	filter.From = from
	filter.To = to
	if limit := strings.TrimSpace(query.Get("limit")); limit != "" {
		size, err := strconv.Atoi(limit)
		if err != nil || size <= 0 {
			return store.AuditFilter{}, errInvalidID
		}
		filter.Limit = size
	}
	return filter, nil
}

func (a *Activity) dayFrom(value string, exclusive bool) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	day, err := time.ParseInLocation("2006-01-02", trimmed, a.cfg.Location)
	if err != nil {
		return nil, errInvalidDate
	}
	if exclusive {
		day = day.AddDate(0, 0, 1)
	}
	return &day, nil
}

func (a *Activity) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter, err := a.filterFrom(r)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		entries, err := a.trail.ListEvents(r.Context(), filter)
		if err != nil {
			log.Printf("handler: could not list the activity log: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := activityItems(entries)
		body := activityResponse{Items: items}
		if len(items) > 0 {
			body.Cursor = items[len(items)-1].ID
		}
		httpx.WriteJSON(w, http.StatusOK, body)
	}
}

func (a *Activity) People() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		people, err := a.trail.ListAuditPeople(r.Context())
		if err != nil {
			log.Printf("handler: could not list the activity log people: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]personItem, 0, len(people))
		for _, person := range people {
			items = append(items, personItem{
				ID:    person.ID,
				Label: person.Label,
				Role:  person.Role,
				Total: person.Total,
			})
		}
		httpx.WriteJSON(w, http.StatusOK, peopleResponse{Items: items})
	}
}

func (a *Activity) Monitor() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := pathID(w, r)
		if !ok {
			return
		}
		filter, err := a.filterFrom(r)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		filter.ActorID = userID
		filter.Limit = monitorPageLimit
		entries, err := a.trail.ListEvents(r.Context(), filter)
		if err != nil {
			log.Printf("handler: could not list an admin timeline: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		counts, err := a.trail.CountActions(r.Context(), userID, filter.From, filter.To)
		if err != nil {
			log.Printf("handler: could not count the actions of an admin: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		totals := make([]actionCountItem, 0, len(counts))
		for _, count := range counts {
			totals = append(totals, actionCountItem{Action: count.Action, Total: count.Total})
		}
		httpx.WriteJSON(w, http.StatusOK, monitorResponse{
			Items:  activityItems(entries),
			Counts: totals,
		})
	}
}

func (a *Activity) Export() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter, err := a.filterFrom(r)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		filter.Limit = exportLimit
		filter.Before = 0
		rows := make([][]string, 0, exportLimit)
		rows = append(rows, []string{"When", "Person", "Role", "Action", "Target", "Client", "Summary"})
		for page := 0; page < exportPages; page++ {
			entries, err := a.trail.ListEvents(r.Context(), filter)
			if err != nil {
				log.Printf("handler: could not export the activity log: %v", err)
				httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
				return
			}
			for _, entry := range entries {
				rows = append(rows, []string{
					entry.CreatedAt.In(a.cfg.Location).Format(time.RFC3339),
					entry.ActorLabel,
					entry.ActorRole,
					entry.Action,
					strings.TrimSpace(entry.TargetType + " " + entry.TargetID),
					entry.Client,
					entry.Summary,
				})
			}
			if len(entries) < filter.Limit {
				break
			}
			filter.Before = entries[len(entries)-1].ID
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+exportFileName+`"`)
		writer := csv.NewWriter(w)
		if err := writer.WriteAll(rows); err != nil {
			log.Printf("handler: could not write the activity log export: %v", err)
		}
	}
}

func activityItems(entries []store.AuditEntry) []activityItem {
	items := make([]activityItem, 0, len(entries))
	for _, entry := range entries {
		items = append(items, activityItem{
			ID:         entry.ID,
			ActorID:    entry.ActorID,
			Actor:      entry.ActorLabel,
			Role:       entry.ActorRole,
			Action:     entry.Action,
			TargetType: entry.TargetType,
			TargetID:   entry.TargetID,
			Summary:    entry.Summary,
			Client:     entry.Client,
			CreatedAt:  entry.CreatedAt.Format(time.RFC3339),
		})
	}
	return items
}
