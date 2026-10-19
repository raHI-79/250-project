package handler

import (
	"context"
	"errors"
	"fmt"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
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
	MessageNotATeacher      = "No teacher profile found for this account."
	MessageRequestNotFound  = "No request found with this ID."
	MessageRequestDecided   = "This request has already been decided."
	MessageTooLateToApprove = "This request can no longer be approved."
	MessageRequestApproved  = "Request approved."
	MessageRequestRejected  = "Request rejected."
)

type TeacherStore interface {
	GetTeacherByUser(ctx context.Context, userID int64) (store.Teacher, error)
	ListTeacherRequests(ctx context.Context, teacherID int64) ([]store.RequestRow, error)
	DecideRequest(ctx context.Context, eventID, teacherID int64, status string, now time.Time) (store.EventDecision, error)
	CancelClassEvent(ctx context.Context, eventID, teacherID int64, now time.Time) (store.EventDecision, error)
	ListTeacherSchedule(ctx context.Context, teacherID int64, date string) ([]store.ScheduleItem, error)
	ListTeacherClasses(ctx context.Context, teacherID int64, date string, now time.Time) ([]store.ClassEventRow, error)
	GetTeacherRoutine(ctx context.Context, routineID, teacherID int64) (store.RoutineRow, error)
	CancelRoutineOccurrence(ctx context.Context, routineID, cancelledBy int64, date string) error
	ListActiveCRs(ctx context.Context, batch, section string) ([]int64, error)
}

type BroadcastNotifier interface {
	NotifyUser(ctx context.Context, userID int64, kind, title, body string) error
	NotifyBroadcast(ctx context.Context, role, batch, section, kind, title, body string) error
}

type Teacher struct {
	cfg      *config.Config
	teachers TeacherStore
	alerts   BroadcastNotifier
}

func NewTeacher(cfg *config.Config, teachers TeacherStore, alerts BroadcastNotifier) *Teacher {
	return &Teacher{cfg: cfg, teachers: teachers, alerts: alerts}
}

type requestItem struct {
	ID               int64  `json:"id"`
	EventType        string `json:"event_type"`
	Batch            string `json:"batch"`
	Section          string `json:"section"`
	CourseCode       string `json:"course_code"`
	CourseTitle      string `json:"course_title"`
	EventDate        string `json:"event_date"`
	StartTime        string `json:"start_time"`
	EndTime          string `json:"end_time"`
	Room             string `json:"room"`
	RequestedBy      string `json:"requested_by"`
	RequestedByRegNo string `json:"requested_by_reg_no"`
	Note             string `json:"note,omitempty"`
	CreatedAt        string `json:"created_at"`
}

type requestsResponse struct {
	Items []requestItem `json:"items"`
}

func (t *Teacher) profile(w http.ResponseWriter, r *http.Request) (store.Teacher, bool) {
	current, ok := middleware.CurrentUser(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
		return store.Teacher{}, false
	}
	teacher, err := t.teachers.GetTeacherByUser(r.Context(), current.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusForbidden, MessageNotATeacher)
			return store.Teacher{}, false
		}
		log.Printf("handler: could not load the profile of a teacher: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return store.Teacher{}, false
	}
	return teacher, true
}

func (t *Teacher) Requests() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := t.profile(w, r)
		if !ok {
			return
		}
		requests, err := t.teachers.ListTeacherRequests(r.Context(), teacher.ID)
		if err != nil {
			log.Printf("handler: could not list the requests of a teacher: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]requestItem, 0, len(requests))
		for _, request := range requests {
			items = append(items, requestItem{
				ID:               request.ID,
				EventType:        request.EventType,
				Batch:            request.Batch,
				Section:          request.Section,
				CourseCode:       request.CourseCode,
				CourseTitle:      request.CourseTitle,
				EventDate:        request.EventDate,
				StartTime:        request.StartTime,
				EndTime:          request.EndTime,
				Room:             request.Room,
				RequestedBy:      request.RequestedBy,
				RequestedByRegNo: request.RequestedByRegNo,
				Note:             request.Note,
				CreatedAt:        timestamp(request.CreatedAt),
			})
		}
		httpx.WriteJSON(w, http.StatusOK, requestsResponse{Items: items})
	}
}

func (t *Teacher) Approve() http.HandlerFunc {
	return t.decide(store.EventStatusApproved, MessageRequestApproved)
}

func (t *Teacher) Reject() http.HandlerFunc {
	return t.decide(store.EventStatusRejected, MessageRequestRejected)
}

func (t *Teacher) decide(status, message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := t.profile(w, r)
		if !ok {
			return
		}
		eventID, ok := pathID(w, r)
		if !ok {
			return
		}
		decision, err := t.teachers.DecideRequest(r.Context(), eventID, teacher.ID, status, t.cfg.Now())
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageRequestNotFound)
			return
		case errors.Is(err, store.ErrNotPending):
			httpx.WriteError(w, http.StatusConflict, MessageRequestDecided)
			return
		case errors.Is(err, store.ErrTooLateToApprove):
			httpx.WriteError(w, http.StatusConflict, MessageTooLateToApprove)
			return
		case err != nil:
			log.Printf("handler: could not decide a request: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		t.announceDecision(r.Context(), decision, status)
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: message})
	}
}

func (t *Teacher) announceDecision(ctx context.Context, decision store.EventDecision, status string) {
	title := MessageRequestRejected
	if status == store.EventStatusApproved {
		title = MessageRequestApproved
	}
	err := t.alerts.NotifyUser(ctx, decision.CreatedBy, notify.TypeDecision, title, eventSentence(decision))
	if err != nil {
		log.Printf("handler: could not notify the class representative of a decision: %v", err)
	}
	if status != store.EventStatusApproved {
		return
	}
	broadcastTitle := fmt.Sprintf("New %s scheduled", eventTypeWords(decision.EventType))
	err = t.alerts.NotifyBroadcast(ctx, auth.RoleStudent, decision.Batch, broadcastSection(decision.Section),
		notify.TypeClass, broadcastTitle, eventSentence(decision))
	if err != nil {
		log.Printf("handler: could not announce a scheduled class: %v", err)
	}
}

func eventSentence(decision store.EventDecision) string {
	return fmt.Sprintf("%s %s on %s from %s to %s in room %s for %s section %s.",
		decision.CourseCode,
		eventTypeWords(decision.EventType),
		decision.EventDate,
		decision.StartTime,
		decision.EndTime,
		decision.Room,
		decision.Batch,
		decision.Section,
	)
}

func broadcastSection(section string) string {
	if section == validate.SectionAll {
		return ""
	}
	return section
}
