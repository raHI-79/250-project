package handler

import (
	"context"
	"errors"
	"fmt"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/daydate"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/settings"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	MessageItemNotFound    = "No schedule item found with this ID."
	MessageItemState       = "This schedule item cannot be changed this way right now."
	MessageItemStarted     = "This class has already started."
	MessageNoProposal      = "This schedule item carries no proposed slot."
	MessageNothingSelected = "Select at least one schedule item."
	MessageLinkChanged     = "Meeting link updated."

	ActionCancel          = "cancel"
	ActionConfirm         = "confirm"
	ActionAccept          = "accept_reschedule"
	ActionDecline         = "decline_reschedule"
	ActionDelete          = "delete"
	ActionEdit            = "edit"
	ActionSetSlot         = "set_slot"
	ActionSetTime         = "set_time"
	ActionReschedule      = "request_reschedule"
	ActionAddToFinal      = "add_to_final"
	ActionRemoveFromFinal = "remove_from_final"
	ActionEditLink        = "edit_link"
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
	FinalStatus         string        `json:"final_status,omitempty"`
	AllowedActions      []string      `json:"allowed_actions,omitempty"`
}

type itemsResponse struct {
	Items []itemBody `json:"items"`
}

// dayKindBody is the chip every routine view puts at the top of a day. A weekly off day
// carries the name of its weekday, a holiday its own name, and a working day the weekday
// whose routine the office told it to follow.
type dayKindBody struct {
	Holiday     bool   `json:"holiday"`
	WorkingDay  bool   `json:"working_day"`
	Name        string `json:"name,omitempty"`
	FollowsName string `json:"follows_weekday_name,omitempty"`
	Label       string `json:"label"`
}

type dayBody struct {
	Date    string       `json:"date"`
	Weekday string       `json:"weekday"`
	Kind    *dayKindBody `json:"day_kind,omitempty"`
	Items   []itemBody   `json:"items"`
}

type dateItemsResponse struct {
	Date      string       `json:"date"`
	DateLabel string       `json:"date_label"`
	Weekday   string       `json:"weekday"`
	Kind      *dayKindBody `json:"day_kind,omitempty"`
	Past      bool         `json:"past"`
	ReadOnly  bool         `json:"read_only"`
	Items     []itemBody   `json:"items"`
}

type finalItemsResponse struct {
	Items []itemBody `json:"items"`
	Days  []dayBody  `json:"days"`
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

func groupItemDays(items []store.ScheduleItem, cfg *config.Config, role string,
	kinds map[string]store.DayKind) []dayBody {
	days := make([]dayBody, 0)
	index := make(map[string]int)
	for _, item := range items {
		position, seen := index[item.ItemDate]
		if !seen {
			position = len(days)
			index[item.ItemDate] = position
			days = append(days, dayBody{
				Date:    item.ItemDate,
				Weekday: weekdayOf(item.ItemDate, cfg),
				Kind:    kindOf(kinds, item.ItemDate),
				Items:   make([]itemBody, 0),
			})
		}
		body := newItemBody(item)
		body.AllowedActions = AllowedActions(role, item, false)
		days[position].Items = append(days[position].Items, body)
	}
	return days
}

const (
	StatusHeld        = "Held"
	StatusCancelledAt = "Cancelled"
	StatusMoved       = "Rescheduled"
	StatusExamHeld    = "Exam held"
)

func finalStatusOf(item store.ScheduleItem) string {
	switch {
	case item.Status == store.ItemCancelled:
		return StatusCancelledAt
	case item.Kind == store.KindExam:
		return StatusExamHeld
	case item.Status == store.ItemRescheduled:
		return StatusMoved
	default:
		return StatusHeld
	}
}

func newDayKindBody(kind store.DayKind) *dayKindBody {
	body := &dayKindBody{
		Holiday:    kind.Holiday,
		WorkingDay: kind.WorkingDay,
		Name:       kind.Name,
	}
	if kind.WorkingDay {
		body.FollowsName = kind.FollowsName()
		body.Label = "Working day · follows " + body.FollowsName
		return body
	}
	name := kind.Name
	if name == "" {
		name = daydate.WeekdayName(kind.Date)
	}
	body.Label = "Holiday · " + name
	return body
}

// DayCalendar answers what each date in a range is. Every routine view takes its chips
// from it, so the chip a class representative sees is the chip the teacher and the
// students see.
type DayCalendar interface {
	DayKinds(ctx context.Context, from, to string, offDays []int) (map[string]store.DayKind, error)
}

// dayKinds reads the calendar for a range and never fails a request over it: a view that
// cannot read the calendar simply shows no chip.
func dayKinds(r *http.Request, calendar DayCalendar, values *settings.Cache,
	from, to string) map[string]store.DayKind {
	if calendar == nil {
		return nil
	}
	offDays := settings.OffDayNumbers()
	if values != nil {
		offDays = values.Numbers(r.Context(), settings.KeyOffDays, offDays)
	}
	kinds, err := calendar.DayKinds(r.Context(), from, to, offDays)
	if err != nil {
		log.Printf("handler: could not read the day calendar from %s to %s: %v", from, to, err)
		return nil
	}
	return kinds
}

func kindOf(kinds map[string]store.DayKind, date string) *dayKindBody {
	kind, ok := kinds[date]
	if !ok {
		return nil
	}
	return newDayKindBody(kind)
}

func weekdayOf(date string, cfg *config.Config) string {
	parsed, err := time.ParseInLocation(validate.DateLayout, date, cfg.Location)
	if err != nil {
		return ""
	}
	return validate.DayName(int(parsed.Weekday()))
}

func dateItems(items []store.ScheduleItem, date string, past bool, cfg *config.Config,
	role string, kind *dayKindBody) dateItemsResponse {
	bodies := make([]itemBody, 0, len(items))
	for _, item := range items {
		body := newItemBody(item)
		if past {
			body.FinalStatus = finalStatusOf(item)
		}
		body.AllowedActions = AllowedActions(role, item, past)
		bodies = append(bodies, body)
	}
	return dateItemsResponse{
		Date:      date,
		DateLabel: daydate.Readable(date),
		Weekday:   weekdayOf(date, cfg),
		Kind:      kind,
		Past:      past,
		ReadOnly:  past,
		Items:     bodies,
	}
}

func firstDays(days []dayBody, limit int) ([]dayBody, bool) {
	if limit <= 0 || len(days) <= limit {
		return days, false
	}
	return days[:limit], true
}

func liveItems(items []store.ScheduleItem) []store.ScheduleItem {
	kept := make([]store.ScheduleItem, 0, len(items))
	for _, item := range items {
		if item.Status == store.ItemCancelled {
			continue
		}
		kept = append(kept, item)
	}
	return kept
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

type DropCourseStore interface {
	ListDropCourseStudents(ctx context.Context, courseID int64, batch string) ([]int64, error)
}

func tellDropCourseStudents(r *http.Request, students DropCourseStore, alerts Notifier,
	item store.ScheduleItem, title string) {
	users, err := students.ListDropCourseStudents(r.Context(), item.CourseID, item.Batch)
	if err != nil {
		log.Printf("handler: could not list the drop course students of %d: %v", item.CourseID, err)
		return
	}
	link := notify.OnDate(notify.LinkRoutine, item.ItemDate)
	for _, userID := range users {
		if err := alerts.NotifyUser(r.Context(), userID, notify.TypeRoutine, title,
			itemSentence(item), link); err != nil {
			log.Printf("handler: could not notify the drop course student %d: %v", userID, err)
		}
	}
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

func requestedDay(w http.ResponseWriter, r *http.Request, cfg *config.Config) (string, bool, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("date"))
	if raw == "" {
		return "", false, true
	}
	parsed, err := validate.Date("date", raw, cfg.Location)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return "", true, false
	}
	return validate.DateString(parsed), true, true
}

func requestedDayLimit(r *http.Request) int {
	limit, err := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit_days")))
	if err != nil || limit < 0 {
		return 0
	}
	return limit
}
