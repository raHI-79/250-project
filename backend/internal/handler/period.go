package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strings"
)

const (
	MessagePeriodCreated = "Class period added."
	MessagePeriodUpdated = "Class period updated."
	MessagePeriodDeleted = "Class period deleted."
	MessagePeriodUnknown = "No class period found with this ID."
	MessagePeriodLast    = "The day keeps at least one class period."
	MessagePeriodOrder   = "Send every class period exactly once."
)

type PeriodStore interface {
	ListPeriods(ctx context.Context) ([]store.Period, error)
	CreatePeriod(ctx context.Context, period store.NewPeriod) (store.Period, error)
	UpdatePeriod(ctx context.Context, periodID int64, patch store.PeriodPatch) (store.Period, error)
	DeletePeriod(ctx context.Context, periodID int64) error
	ReorderPeriods(ctx context.Context, ids []int64) ([]store.Period, error)
}

type Periods struct {
	periods PeriodStore
}

func NewPeriods(periods PeriodStore) *Periods {
	return &Periods{periods: periods}
}

type periodRequest struct {
	Label     string `json:"label"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	IsBreak   bool   `json:"is_break"`
}

type periodPatchRequest struct {
	Label     *string `json:"label"`
	StartTime *string `json:"start_time"`
	EndTime   *string `json:"end_time"`
	IsBreak   *bool   `json:"is_break"`
}

type reorderRequest struct {
	IDs []int64 `json:"ids"`
}

type periodItem struct {
	ID        int64  `json:"id"`
	Label     string `json:"label"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	IsBreak   bool   `json:"is_break"`
	Position  int    `json:"position"`
}

type periodsResponse struct {
	Items []periodItem `json:"items"`
}

type periodResponse struct {
	periodItem
	Message string `json:"message"`
}

func (p *Periods) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		periods, err := p.periods.ListPeriods(r.Context())
		if err != nil {
			log.Printf("handler: could not list the class periods: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, periodsResponse{Items: periodItems(periods)})
	}
}

func (p *Periods) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request periodRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		label, start, end, ok := periodFields(w, request.Label, request.StartTime, request.EndTime)
		if !ok {
			return
		}
		created, err := p.periods.CreatePeriod(r.Context(), store.NewPeriod{
			Label:   label,
			Start:   start,
			End:     end,
			IsBreak: request.IsBreak,
		})
		if errors.Is(err, store.ErrPeriodOverlap) {
			httpx.WriteError(w, http.StatusConflict, overlapMessage(err))
			return
		}
		if err != nil {
			log.Printf("handler: could not create a class period: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, periodResponse{
			periodItem: newPeriodItem(created),
			Message:    MessagePeriodCreated,
		})
	}
}

func (p *Periods) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		periodID, ok := pathID(w, r)
		if !ok {
			return
		}
		var request periodPatchRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		patch, ok := periodPatch(w, request)
		if !ok {
			return
		}
		updated, err := p.periods.UpdatePeriod(r.Context(), periodID, patch)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessagePeriodUnknown)
		case errors.Is(err, store.ErrPeriodOverlap):
			httpx.WriteError(w, http.StatusConflict, overlapMessage(err))
		case err != nil:
			log.Printf("handler: could not update a class period: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusOK, periodResponse{
				periodItem: newPeriodItem(updated),
				Message:    MessagePeriodUpdated,
			})
		}
	}
}

func (p *Periods) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		periodID, ok := pathID(w, r)
		if !ok {
			return
		}
		err := p.periods.DeletePeriod(r.Context(), periodID)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessagePeriodUnknown)
		case errors.Is(err, store.ErrPeriodInUse):
			httpx.WriteError(w, http.StatusConflict, MessagePeriodLast)
		case err != nil:
			log.Printf("handler: could not delete a class period: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessagePeriodDeleted})
		}
	}
}

func (p *Periods) Reorder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request reorderRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		if !allDistinct(request.IDs) {
			httpx.WriteError(w, http.StatusBadRequest, MessagePeriodOrder)
			return
		}
		periods, err := p.periods.ReorderPeriods(r.Context(), request.IDs)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusBadRequest, MessagePeriodOrder)
		case err != nil:
			log.Printf("handler: could not reorder the class periods: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusOK, periodsResponse{Items: periodItems(periods)})
		}
	}
}

func periodPatch(w http.ResponseWriter, request periodPatchRequest) (store.PeriodPatch, bool) {
	patch := store.PeriodPatch{IsBreak: request.IsBreak}
	if request.Label != nil {
		label, err := validate.PeriodLabel("Label", *request.Label)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return store.PeriodPatch{}, false
		}
		patch.Label = &label
	}
	if request.StartTime != nil {
		start, err := validate.ClockTime("Start time", *request.StartTime)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return store.PeriodPatch{}, false
		}
		patch.Start = &start
	}
	if request.EndTime != nil {
		end, err := validate.ClockTime("End time", *request.EndTime)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return store.PeriodPatch{}, false
		}
		patch.End = &end
	}
	return patch, true
}

func allDistinct(ids []int64) bool {
	if len(ids) == 0 {
		return false
	}
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if id <= 0 || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}

func periodFields(w http.ResponseWriter, rawLabel, rawStart, rawEnd string) (string, string, string, bool) {
	label, err := validate.PeriodLabel("Label", rawLabel)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return "", "", "", false
	}
	start, err := validate.ClockTime("Start time", rawStart)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return "", "", "", false
	}
	end, err := validate.ClockTime("End time", rawEnd)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return "", "", "", false
	}
	if err := validate.PeriodWindow(start, end); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return "", "", "", false
	}
	return label, start, end, true
}

func overlapMessage(err error) string {
	parts := strings.SplitN(err.Error(), ": ", 3)
	return parts[len(parts)-1]
}

func periodItems(periods []store.Period) []periodItem {
	items := make([]periodItem, 0, len(periods))
	for _, period := range periods {
		items = append(items, newPeriodItem(period))
	}
	return items
}

func newPeriodItem(period store.Period) periodItem {
	return periodItem{
		ID:        period.ID,
		Label:     period.Label,
		StartTime: period.Start,
		EndTime:   period.End,
		IsBreak:   period.IsBreak,
		Position:  period.Position,
	}
}
