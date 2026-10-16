package handler

import (
	"context"
	"errors"
	"fmt"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"time"
)

const (
	MessageClassNotFound       = "No class found with this ID."
	MessageClassNotCancellable = "This class can no longer be cancelled."
	MessageClassCancelled      = "Class cancelled."
	MessageOccurrenceCancelled = "This occurrence is already cancelled."
	MessagePastDate            = "date must be today or later."
)

type cancelRoutineRequest struct {
	Date string `json:"date"`
}

func (t *Teacher) CancelClass() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := t.profile(w, r)
		if !ok {
			return
		}
		eventID, ok := pathID(w, r)
		if !ok {
			return
		}
		decision, err := t.teachers.CancelClassEvent(r.Context(), eventID, teacher.ID, t.cfg.Now())
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageClassNotFound)
			return
		case errors.Is(err, store.ErrNotCancellable):
			httpx.WriteError(w, http.StatusConflict, MessageClassNotCancellable)
			return
		case err != nil:
			log.Printf("handler: could not cancel a class: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		title := fmt.Sprintf("%s cancelled", cancelledWords(decision.EventType))
		body := eventSentence(decision)
		if err := t.alerts.NotifyUser(r.Context(), decision.CreatedBy, notify.TypeClass, title, body); err != nil {
			log.Printf("handler: could not notify the class representative of a cancellation: %v", err)
		}
		err = t.alerts.NotifyBroadcast(r.Context(), auth.RoleStudent, decision.Batch, broadcastSection(decision.Section),
			notify.TypeClass, title, body)
		if err != nil {
			log.Printf("handler: could not announce a cancellation: %v", err)
		}
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageClassCancelled})
	}
}

func (t *Teacher) CancelRoutine() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := t.profile(w, r)
		if !ok {
			return
		}
		routineID, ok := pathID(w, r)
		if !ok {
			return
		}
		var request cancelRoutineRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		date, err := validate.Date("date", request.Date, t.cfg.Location)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		now := t.cfg.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, t.cfg.Location)
		if date.Before(today) {
			httpx.WriteError(w, http.StatusBadRequest, MessagePastDate)
			return
		}
		entry, err := t.teachers.GetTeacherRoutine(r.Context(), routineID, teacher.ID)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, MessageRoutineNotFound)
			return
		}
		if err != nil {
			log.Printf("handler: could not load a routine entry of a teacher: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if int(date.Weekday()) != entry.DayOfWeek {
			httpx.WriteError(w, http.StatusBadRequest, weekdayMismatchMessage(entry.DayOfWeek))
			return
		}
		day := validate.DateString(date)
		err = t.teachers.CancelRoutineOccurrence(r.Context(), routineID, teacher.UserID, day)
		switch {
		case errors.Is(err, store.ErrAlreadyCancelled):
			httpx.WriteError(w, http.StatusConflict, MessageOccurrenceCancelled)
			return
		case err != nil:
			log.Printf("handler: could not cancel a routine occurrence: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		t.announceRoutineCancellation(r.Context(), entry, day)
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageClassCancelled})
	}
}

func (t *Teacher) announceRoutineCancellation(ctx context.Context, entry store.RoutineRow, day string) {
	title := "Class cancelled"
	body := fmt.Sprintf("%s class on %s from %s to %s in room %s for %s section %s.",
		entry.CourseCode, day, entry.StartTime, entry.EndTime, entry.Room, entry.Batch, entry.Section)
	err := t.alerts.NotifyBroadcast(ctx, auth.RoleStudent, entry.Batch, broadcastSection(entry.Section),
		notify.TypeClass, title, body)
	if err != nil {
		log.Printf("handler: could not announce a routine cancellation: %v", err)
	}
	representatives, err := t.teachers.ListActiveCRs(ctx, entry.Batch, entry.Section)
	if err != nil {
		log.Printf("handler: could not list the class representatives of a cancelled class: %v", err)
		return
	}
	for _, userID := range representatives {
		if err := t.alerts.NotifyUser(ctx, userID, notify.TypeClass, title, body); err != nil {
			log.Printf("handler: could not notify a class representative of a cancellation: %v", err)
		}
	}
}

func cancelledWords(eventType string) string {
	switch eventType {
	case validate.EventExam:
		return "Exam"
	case validate.EventExtra:
		return "Extra class"
	default:
		return "Class"
	}
}

func weekdayMismatchMessage(day int) string {
	return fmt.Sprintf("date must fall on a %s, the weekday of this routine entry.", validate.DayName(day))
}
