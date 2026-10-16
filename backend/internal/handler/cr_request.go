package handler

import (
	"errors"
	"fmt"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strings"
	"time"
)

const MessageInvalidEventStatus = "status must be one of pending, approved, rejected, cancelled, auto_cancelled."

type createClassRequest struct {
	EventType string `json:"event_type"`
	CourseID  int64  `json:"course_id"`
	TeacherID int64  `json:"teacher_id"`
	EventDate string `json:"event_date"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Room      string `json:"room"`
}

type classRequestItem struct {
	ID          int64  `json:"id"`
	EventType   string `json:"event_type"`
	Batch       string `json:"batch"`
	Section     string `json:"section"`
	CourseID    int64  `json:"course_id"`
	CourseCode  string `json:"course_code"`
	CourseTitle string `json:"course_title"`
	TeacherID   int64  `json:"teacher_id"`
	TeacherName string `json:"teacher_name"`
	EventDate   string `json:"event_date"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Room        string `json:"room"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	DecidedAt   string `json:"decided_at,omitempty"`
}

type classRequestsResponse struct {
	Items []classRequestItem `json:"items"`
}

func (c *CR) CreateClassRequest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := c.profile(w, r)
		if !ok {
			return
		}
		var request createClassRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		event, err := request.toNewClassEvent(c.cfg.Now(), c.cfg.Location)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		event.Batch = profile.Batch
		event.Section = profile.Section
		event.CreatedBy = profile.UserID
		created, err := c.classes.CreateClassRequest(r.Context(), event)
		var reserved *store.FixedSlotError
		switch {
		case errors.Is(err, store.ErrRoomUnavailable):
			httpx.WriteError(w, http.StatusConflict, MessageRoomUnavailable)
			return
		case errors.As(err, &reserved):
			httpx.WriteError(w, http.StatusConflict, fixedSlotMessage(reserved.Booking))
			return
		case errors.Is(err, store.ErrCourseNotFound):
			httpx.WriteError(w, http.StatusBadRequest, MessageCourseUnknown)
			return
		case errors.Is(err, store.ErrTeacherNotFound):
			httpx.WriteError(w, http.StatusBadRequest, MessageTeacherNotFound)
			return
		case errors.Is(err, store.ErrTeacherInactive):
			httpx.WriteError(w, http.StatusBadRequest, MessageTeacherNotActive)
			return
		case err != nil:
			log.Printf("handler: could not create a class request: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		title, body := requestNotification(created)
		if err := c.alerts.NotifyUser(r.Context(), created.TeacherUserID, NotificationTypeClassRequest, title, body); err != nil {
			log.Printf("handler: could not notify the teacher of a class request: %v", err)
		}
		httpx.WriteJSON(w, http.StatusCreated, newClassRequestItem(created))
	}
}

func (request createClassRequest) toNewClassEvent(now time.Time, loc *time.Location) (store.NewClassEvent, error) {
	var event store.NewClassEvent
	eventType, err := validate.EventType("Type", request.EventType)
	if err != nil {
		return store.NewClassEvent{}, err
	}
	event.EventType = eventType
	if request.CourseID <= 0 {
		return store.NewClassEvent{}, errors.New(MessageCourseRequired)
	}
	event.CourseID = request.CourseID
	if request.TeacherID <= 0 {
		return store.NewClassEvent{}, errors.New(MessageTeacherRequired)
	}
	event.TeacherID = request.TeacherID
	date, err := validate.FutureDate("Date", request.EventDate, now, loc)
	if err != nil {
		return store.NewClassEvent{}, err
	}
	event.EventDate = validate.DateString(date)
	start, err := validate.ClockTime("Start time", request.StartTime)
	if err != nil {
		return store.NewClassEvent{}, err
	}
	end, err := validate.ClockTime("End time", request.EndTime)
	if err != nil {
		return store.NewClassEvent{}, err
	}
	if err := validate.ClassWindow(start, end); err != nil {
		return store.NewClassEvent{}, err
	}
	event.StartTime, event.EndTime = start, end
	room, err := validate.Required("Room", request.Room)
	if err != nil {
		return store.NewClassEvent{}, err
	}
	event.Room = room
	return event, nil
}

func fixedSlotMessage(booking store.RoomBooking) string {
	audience := "section " + booking.Section
	if booking.Section == validate.SectionAll {
		audience = "both sections"
	}
	return fmt.Sprintf("Room %s is reserved for %s %s (%s) every %s %s–%s.",
		booking.Room,
		booking.Batch,
		audience,
		booking.CourseCode,
		validate.DayName(booking.DayOfWeek),
		booking.StartTime,
		booking.EndTime,
	)
}

func requestNotification(event store.ClassEventRow) (string, string) {
	title := fmt.Sprintf("New %s request", eventTypeWords(event.EventType))
	body := fmt.Sprintf("%s section %s asked for %s on %s from %s to %s in room %s.",
		event.Batch,
		event.Section,
		event.CourseCode,
		event.EventDate,
		event.StartTime,
		event.EndTime,
		event.Room,
	)
	return title, body
}

func eventTypeWords(eventType string) string {
	if eventType == validate.EventExtra {
		return "extra class"
	}
	return eventType
}

func newClassRequestItem(event store.ClassEventRow) classRequestItem {
	item := classRequestItem{
		ID:          event.ID,
		EventType:   event.EventType,
		Batch:       event.Batch,
		Section:     event.Section,
		CourseID:    event.CourseID,
		CourseCode:  event.CourseCode,
		CourseTitle: event.CourseTitle,
		TeacherID:   event.TeacherID,
		TeacherName: event.TeacherName,
		EventDate:   event.EventDate,
		StartTime:   event.StartTime,
		EndTime:     event.EndTime,
		Room:        event.Room,
		Status:      event.Status,
		CreatedAt:   timestamp(event.CreatedAt),
	}
	if event.DecidedAt != nil {
		item.DecidedAt = timestamp(*event.DecidedAt)
	}
	return item
}

func (c *CR) Requests() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := c.profile(w, r)
		if !ok {
			return
		}
		status, err := eventStatusFilter(r.URL.Query().Get("status"))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		requests, err := c.classes.ListClassRequests(r.Context(), profile.UserID, status)
		if err != nil {
			log.Printf("handler: could not list the class requests: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]classRequestItem, 0, len(requests))
		for _, request := range requests {
			items = append(items, newClassRequestItem(request))
		}
		httpx.WriteJSON(w, http.StatusOK, classRequestsResponse{Items: items})
	}
}

func eventStatusFilter(raw string) (string, error) {
	status := strings.ToLower(strings.TrimSpace(raw))
	switch status {
	case "",
		store.EventStatusPending,
		store.EventStatusApproved,
		store.EventStatusRejected,
		store.EventStatusCancelled,
		store.EventStatusAutoCancelled:
		return status, nil
	default:
		return "", errors.New(MessageInvalidEventStatus)
	}
}
