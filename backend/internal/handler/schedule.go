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
	"strings"
)

const (
	MessageItemNotFound    = "No schedule item found with this ID."
	MessageItemState       = "This schedule item cannot be changed this way right now."
	MessageItemStarted     = "This class has already started."
	MessageNoProposal      = "This schedule item carries no proposed slot."
	MessageNothingSelected = "Select at least one schedule item."
	MessageLinkChanged     = "Meeting link updated."

	ActionCancel          = "cancel"
	ActionDelete          = "delete"
	ActionEdit            = "edit"
	ActionSetSlot         = "set_slot"
	ActionSetTime         = "set_time"
	ActionReschedule      = "request_reschedule"
	ActionAddToFinal      = "add_to_final"
	ActionRemoveFromFinal = "remove_from_final"
)

type proposedBody struct {
	ItemDate    string `json:"item_date"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Mode        string `json:"mode,omitempty"`
	Room        string `json:"room,omitempty"`
	MeetingLink string `json:"meeting_link,omitempty"`
}

type itemBody struct {
	ID                  int64         `json:"id"`
	Batch               string        `json:"batch"`
	Section             string        `json:"section"`
	CourseID            int64         `json:"course_id"`
	CourseCode          string        `json:"course_code"`
	CourseTitle         string        `json:"course_title"`
	TeacherID           int64         `json:"teacher_id,omitempty"`
	TeacherUserID       int64         `json:"teacher_user_id,omitempty"`
	TeacherName         string        `json:"teacher_name,omitempty"`
	Kind                string        `json:"kind"`
	Origin              string        `json:"origin"`
	ItemDate            string        `json:"item_date"`
	StartTime           string        `json:"start_time,omitempty"`
	EndTime             string        `json:"end_time,omitempty"`
	Mode                string        `json:"mode"`
	Room                string        `json:"room,omitempty"`
	MeetingLink         string        `json:"meeting_link,omitempty"`
	Status              string        `json:"status"`
	TeacherMessage      string        `json:"teacher_message,omitempty"`
	CancelReason        string        `json:"cancel_reason,omitempty"`
	Syllabus            string        `json:"syllabus,omitempty"`
	Instructions        string        `json:"instructions,omitempty"`
	Note                string        `json:"note,omitempty"`
	Proposed            *proposedBody `json:"proposed,omitempty"`
	InFinal             bool          `json:"in_final"`
	Published           bool          `json:"published"`
	ChangedSincePublish bool          `json:"changed_since_publish"`
	NeedsTime           bool          `json:"needs_time"`
	AllowedActions      []string      `json:"allowed_actions,omitempty"`
}

type itemsResponse struct {
	Items []itemBody `json:"items"`
}

type itemResponse struct {
	Item    itemBody `json:"item"`
	Message string   `json:"message"`
}

func newItemBody(item store.ScheduleItem) itemBody {
	body := itemBody{
		ID:                  item.ID,
		Batch:               item.Batch,
		Section:             item.Section,
		CourseID:            item.CourseID,
		CourseCode:          item.CourseCode,
		CourseTitle:         item.CourseTitle,
		TeacherID:           item.TeacherID,
		TeacherUserID:       item.TeacherUserID,
		Kind:                item.Kind,
		Origin:              item.Origin,
		ItemDate:            item.ItemDate,
		StartTime:           item.StartTime,
		EndTime:             item.EndTime,
		Mode:                item.Mode,
		Room:                item.Room,
		MeetingLink:         item.MeetingLink,
		Status:              item.Status,
		TeacherMessage:      item.TeacherMessage,
		CancelReason:        item.CancelReason,
		Syllabus:            item.Syllabus,
		Instructions:        item.Instructions,
		Note:                item.Note,
		InFinal:             item.InFinal,
		Published:           item.Published(),
		ChangedSincePublish: item.ChangedSincePublish(),
		NeedsTime:           !item.HasTimes(),
	}
	if item.TeacherID > 0 {
		body.TeacherName = item.TeacherName
	}
	if item.ProposedDate != "" {
		body.Proposed = &proposedBody{
			ItemDate:    item.ProposedDate,
			StartTime:   item.ProposedStart,
			EndTime:     item.ProposedEnd,
			Mode:        item.ProposedMode,
			Room:        item.ProposedRoom,
			MeetingLink: item.ProposedLink,
		}
	}
	return body
}

func newItemBodies(items []store.ScheduleItem) []itemBody {
	bodies := make([]itemBody, 0, len(items))
	for _, item := range items {
		bodies = append(bodies, newItemBody(item))
	}
	return bodies
}

func crActions(item store.ScheduleItem) []string {
	if item.Status == store.ItemCancelled {
		return []string{}
	}
	actions := make([]string, 0, 5)
	switch {
	case item.Origin == store.OriginCR:
		actions = append(actions, ActionEdit, ActionCancel, ActionDelete)
	case item.Status == store.ItemTeacherAsk:
		actions = append(actions, ActionSetSlot, ActionCancel)
	case item.Origin == store.OriginTeacherExam:
		actions = append(actions, ActionSetTime)
	}
	if item.Origin == store.OriginFixed {
		switch item.Status {
		case store.ItemAwaiting, store.ItemNoResponse, store.ItemConfirmed, store.ItemDeclined:
			actions = append(actions, ActionReschedule)
		}
		switch item.Status {
		case store.ItemNoResponse, store.ItemDeclined:
			actions = append(actions, ActionCancel)
		}
	}
	if item.InFinal {
		actions = append(actions, ActionRemoveFromFinal)
	} else {
		actions = append(actions, ActionAddToFinal)
	}
	return actions
}

func conflictMessage(conflict *store.SlotConflict) string {
	if conflict.Reason == store.ConflictRoom {
		return fmt.Sprintf("Room %s is already booked for %s section %s from %s to %s.",
			conflict.Room, conflict.Batch, conflict.Section, conflict.StartTime, conflict.EndTime)
	}
	return fmt.Sprintf("%s section %s already has %s at %s.",
		conflict.Batch, conflict.Section, conflict.CourseCode, conflict.StartTime)
}

func scheduleError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	var conflict *store.SlotConflict
	var incomplete *store.IncompleteItemError
	switch {
	case errors.As(err, &conflict):
		httpx.WriteError(w, http.StatusConflict, conflictMessage(conflict))
	case errors.As(err, &incomplete):
		httpx.WriteError(w, http.StatusBadRequest, fmt.Sprintf(
			"%s on %s still needs a date, a time and a room or a meeting link.",
			incomplete.Item.CourseName(), incomplete.Item.ItemDate))
	case errors.Is(err, store.ErrItemNotFound), errors.Is(err, store.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, MessageItemNotFound)
	case errors.Is(err, store.ErrItemState):
		httpx.WriteError(w, http.StatusConflict, MessageItemState)
	case errors.Is(err, store.ErrItemStarted):
		httpx.WriteError(w, http.StatusConflict, MessageItemStarted)
	case errors.Is(err, store.ErrNoProposal):
		httpx.WriteError(w, http.StatusConflict, MessageNoProposal)
	case errors.Is(err, store.ErrNothingToPublish):
		httpx.WriteError(w, http.StatusBadRequest, MessageNothingSelected)
	case errors.Is(err, store.ErrCourseNotFound):
		httpx.WriteError(w, http.StatusBadRequest, MessageCourseUnknown)
	default:
		log.Printf("handler: could not change a schedule item: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
	}
	return true
}

func itemSentence(item store.ScheduleItem) string {
	where := "online"
	if item.Mode == store.ModeRoom && item.Room != "" {
		where = "room " + item.Room
	}
	when := item.ItemDate
	if item.HasTimes() {
		when += " " + item.StartTime + " to " + item.EndTime
	}
	return fmt.Sprintf("%s for %s section %s on %s in %s.",
		item.CourseName(), item.Batch, item.Section, when, where)
}

func broadcastSection(section string) string {
	if section == validate.SectionAll {
		return ""
	}
	return section
}

type linkRequest struct {
	MeetingLink string `json:"meeting_link"`
}

type Notifier interface {
	NotifyUser(ctx context.Context, userID int64, kind, title, body string, link store.NotificationLink) error
}

type BroadcastNotifier interface {
	Notifier
	PushUser(ctx context.Context, userID int64, kind, title, body string,
		link store.NotificationLink) error
	NotifyBroadcast(ctx context.Context, role, batch, section, kind, title, body string,
		link store.NotificationLink) error
}

type ScheduleStore interface {
	UpdateItemLink(ctx context.Context, itemID, userID int64, link string) (store.ScheduleItem, error)
	ListActiveCRs(ctx context.Context, batch, section string) ([]int64, error)
}

type Schedule struct {
	cfg    *config.Config
	items  ScheduleStore
	alerts BroadcastNotifier
}

func NewSchedule(cfg *config.Config, items ScheduleStore, alerts BroadcastNotifier) *Schedule {
	return &Schedule{cfg: cfg, items: items, alerts: alerts}
}

func (s *Schedule) Link() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		itemID, ok := pathID(w, r)
		if !ok {
			return
		}
		var request linkRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		link, err := validate.Link("Meeting link", strings.TrimSpace(request.MeetingLink))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		item, err := s.items.UpdateItemLink(r.Context(), itemID, current.ID, link)
		if scheduleError(w, err) {
			return
		}
		if item.Published() {
			title := "Online link updated"
			err := s.alerts.NotifyBroadcast(r.Context(), auth.RoleStudent, item.Batch,
				broadcastSection(item.Section), notify.TypeRoutine, title, itemSentence(item),
				notify.OnDate(notify.LinkRoutine, item.ItemDate))
			if err != nil {
				log.Printf("handler: could not announce an updated link: %v", err)
			}
		}
		httpx.WriteJSON(w, http.StatusOK, itemResponse{Item: newItemBody(item), Message: MessageLinkChanged})
	}
}
