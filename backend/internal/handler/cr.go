package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"net/http"
)

const (
	MessageNotACR                = "No class representative profile found for this account."
	MessageRoomUnavailable       = "This room is not available at the selected time."
	MessageDateRequired          = "date is required."
	NotificationTypeClassRequest = "class_request"
)

type CRStore interface {
	GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error)
	ListSectionRoutines(ctx context.Context, batch, section string) ([]store.RoutineRow, error)
	CreateClassRequest(ctx context.Context, event store.NewClassEvent) (store.ClassEventRow, error)
	ListRoomBookings(ctx context.Context, date, room string) ([]store.Booking, error)
	ListClassRequests(ctx context.Context, createdBy int64, status string) ([]store.ClassEventRow, error)
	ListBatchCourseSlots(ctx context.Context, batch, section string) ([]store.CourseSlotRow, error)
	ListBatchDay(ctx context.Context, batch, date string) ([]store.DayEntryRow, error)
}

type Notifier interface {
	NotifyUser(ctx context.Context, userID int64, kind, title, body string) error
}

type CR struct {
	cfg     *config.Config
	classes CRStore
	alerts  Notifier
}

func NewCR(cfg *config.Config, classes CRStore, alerts Notifier) *CR {
	return &CR{cfg: cfg, classes: classes, alerts: alerts}
}

func (c *CR) profile(w http.ResponseWriter, r *http.Request) (store.StudentProfile, bool) {
	current, ok := middleware.CurrentUser(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
		return store.StudentProfile{}, false
	}
	profile, err := c.classes.GetStudentProfile(r.Context(), current.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusForbidden, MessageNotACR)
			return store.StudentProfile{}, false
		}
		log.Printf("handler: could not load the profile of a class representative: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return store.StudentProfile{}, false
	}
	if !profile.IsCR {
		httpx.WriteError(w, http.StatusForbidden, MessageNotACR)
		return store.StudentProfile{}, false
	}
	return profile, true
}

func (c *CR) Routine() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := c.profile(w, r)
		if !ok {
			return
		}
		entries, err := c.classes.ListSectionRoutines(r.Context(), profile.Batch, profile.Section)
		if err != nil {
			log.Printf("handler: could not list the routine of a class representative: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]routineItem, 0, len(entries))
		for _, entry := range entries {
			items = append(items, newRoutineItem(entry))
		}
		httpx.WriteJSON(w, http.StatusOK, routinesResponse{Items: items})
	}
}
