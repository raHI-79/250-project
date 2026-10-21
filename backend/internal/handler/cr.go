package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	MessageNotACR            = "No class representative profile found for this account."
	MessageDateRequired      = "date is required."
	MessageItemCreated       = "Added to your batch schedule."
	MessageItemUpdated       = "Schedule item updated."
	MessageItemRemoved       = "Schedule item deleted."
	MessageAskedTeacher      = "The teacher was asked for the new slot."
	MessageAddedToFinal      = "Added to the final routine."
	MessageRemovedFromFinal  = "Removed from the final routine."
	MessageRoutinePublished  = "Routine published."
	MessageTimesTogether     = "Send the start time and the end time together."
	MessageRoomForRoomMode   = "Room is required for a class in a room."
	MessageLinkForOnlineMode = "Meeting link is required for an online class."
	MessageIDsRequired       = "item_ids is required."

	titleRoutinePublished = "Routine published"
	titleRoutineChanged   = "Routine changed"
	titleCRAsked          = "The class representative asked for another slot"
	titleCRChanged        = "The class representative changed a class"
	titleCRCancelled      = "The class representative cancelled a class"
	availabilityDays      = 7
	LabelYours            = "yours"
	LabelUnavailable      = "unavailable"
)

type CRStore interface {
	GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error)
	ListSectionRoutines(ctx context.Context, batch, section string) ([]store.RoutineRow, error)
	ListBatchCourseSlots(ctx context.Context, batch, section string) ([]store.CourseSlotRow, error)
	ListRoomAvailability(ctx context.Context, from, to string) ([]store.RoomSlot, error)
	TeacherOfCourse(ctx context.Context, batch, section string, courseID int64) (int64, error)
	CreateCRItem(ctx context.Context, item store.NewScheduleItem) (store.ScheduleItem, error)
	ListBatchItems(ctx context.Context, batch, from string) ([]store.ScheduleItem, error)
	CRRequestReschedule(ctx context.Context, itemID int64, batch string,
		proposal store.Proposal) (store.ScheduleItem, error)
	UpdateItemAsCR(ctx context.Context, itemID int64, batch string, patch store.ItemPatch) (store.ScheduleItem, error)
	CancelItemAsCR(ctx context.Context, itemID int64, batch, reason string) (store.ScheduleItem, error)
	DeleteCRItem(ctx context.Context, itemID int64, batch string) (store.ScheduleItem, error)
	SetItemFinal(ctx context.Context, itemID int64, batch string, inFinal bool) (store.ScheduleItem, error)
	ListFinalItems(ctx context.Context, batch string) ([]store.ScheduleItem, error)
	ListItemsByIDs(ctx context.Context, batch string, ids []int64) ([]store.ScheduleItem, error)
	PublishFinal(ctx context.Context, batch string, publishedBy int64,
		ids []int64) (store.Publication, []store.ScheduleItem, error)
}

type CR struct {
	cfg     *config.Config
	classes CRStore
	alerts  BroadcastNotifier
}

func NewCR(cfg *config.Config, classes CRStore, alerts BroadcastNotifier) *CR {
	return &CR{cfg: cfg, classes: classes, alerts: alerts}
}

type roomSlotBody struct {
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	Batch      string `json:"batch"`
	Section    string `json:"section"`
	CourseCode string `json:"course_code"`
	Status     string `json:"status"`
	Label      string `json:"label"`
}

type roomDayBody struct {
	Date    string         `json:"date"`
	Weekday string         `json:"weekday"`
	Slots   []roomSlotBody `json:"slots"`
}

type roomBody struct {
	Room string        `json:"room"`
	Days []roomDayBody `json:"days"`
}

type availabilityResponse struct {
	From  string     `json:"from"`
	To    string     `json:"to"`
	Rooms []roomBody `json:"rooms"`
}

type crItemRequest struct {
	Kind         string `json:"kind"`
	CourseID     int64  `json:"course_id"`
	Section      string `json:"section"`
	ItemDate     string `json:"item_date"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
	Mode         string `json:"mode"`
	Room         string `json:"room"`
	MeetingLink  string `json:"meeting_link"`
	Syllabus     string `json:"syllabus"`
	Instructions string `json:"instructions"`
	Note         string `json:"note"`
}

type crPatchRequest struct {
	ItemDate     *string `json:"item_date"`
	StartTime    *string `json:"start_time"`
	EndTime      *string `json:"end_time"`
	Mode         *string `json:"mode"`
	Room         *string `json:"room"`
	MeetingLink  *string `json:"meeting_link"`
	Note         *string `json:"note"`
	Syllabus     *string `json:"syllabus"`
	Instructions *string `json:"instructions"`
}

type proposalRequest struct {
	ItemDate    string `json:"date"`
	StartTime   string `json:"start"`
	EndTime     string `json:"end"`
	Mode        string `json:"mode"`
	Room        string `json:"room"`
	MeetingLink string `json:"meeting_link"`
}

type publishRequest struct {
	ItemIDs []int64 `json:"item_ids"`
}

type finalDayBody struct {
	Date    string     `json:"date"`
	Weekday string     `json:"weekday"`
	Items   []itemBody `json:"items"`
}

type finalResponse struct {
	Days                []finalDayBody `json:"days"`
	NeedsTime           int            `json:"needs_time"`
	ChangedSincePublish int            `json:"changed_since_publish"`
}

type publishResponse struct {
	PublicationID int64  `json:"publication_id"`
	Published     int    `json:"published"`
	Changed       int    `json:"changed"`
	Message       string `json:"message"`
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

func (c *CR) Courses() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := c.profile(w, r)
		if !ok {
			return
		}
		slots, err := c.classes.ListBatchCourseSlots(r.Context(), profile.Batch, profile.Section)
		if err != nil {
			log.Printf("handler: could not list the courses of a class representative: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, crCoursesResponse{Items: groupCourseSlots(slots)})
	}
}

func (c *CR) RoomAvailability() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := c.profile(w, r)
		if !ok {
			return
		}
		from := c.cfg.Now()
		if raw := strings.TrimSpace(r.URL.Query().Get("from")); raw != "" {
			parsed, err := validate.Date("from", raw, c.cfg.Location)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			from = parsed
		}
		to := from.AddDate(0, 0, availabilityDays-1)
		slots, err := c.classes.ListRoomAvailability(r.Context(),
			validate.DateString(from), validate.DateString(to))
		if err != nil {
			log.Printf("handler: could not list the room availability: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, availabilityResponse{
			From:  validate.DateString(from),
			To:    validate.DateString(to),
			Rooms: groupRoomSlots(slots, profile.Batch, c.cfg.Location),
		})
	}
}

func (c *CR) CreateItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := c.profile(w, r)
		if !ok {
			return
		}
		var request crItemRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		item, err := c.itemFrom(request, profile)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		teacherID, err := c.classes.TeacherOfCourse(r.Context(), item.Batch, item.Section, item.CourseID)
		if err != nil {
			log.Printf("handler: could not find the teacher of a course: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		item.TeacherID = teacherID
		created, err := c.classes.CreateCRItem(r.Context(), item)
		if scheduleError(w, err) {
			return
		}
		c.tellTeacher(r, created, titleCRChanged)
		httpx.WriteJSON(w, http.StatusCreated, itemResponse{
			Item:    newItemBody(created),
			Message: MessageItemCreated,
		})
	}
}

func (c *CR) itemFrom(request crItemRequest, profile store.StudentProfile) (store.NewScheduleItem, error) {
	kind, err := validate.ClassOrExam("kind", request.Kind)
	if err != nil {
		return store.NewScheduleItem{}, err
	}
	if request.CourseID <= 0 {
		return store.NewScheduleItem{}, errors.New("course_id must be a positive number.")
	}
	section := profile.Section
	if strings.TrimSpace(request.Section) != "" {
		section, err = validate.RoutineSection("Section", request.Section)
		if err != nil {
			return store.NewScheduleItem{}, err
		}
	}
	date, err := validate.Date("Date", request.ItemDate, c.cfg.Location)
	if err != nil {
		return store.NewScheduleItem{}, err
	}
	start, end, err := clockRange(request.StartTime, request.EndTime)
	if err != nil {
		return store.NewScheduleItem{}, err
	}
	mode, room, link, err := slotFields(request.Mode, request.Room, request.MeetingLink)
	if err != nil {
		return store.NewScheduleItem{}, err
	}
	syllabus, err := validate.ShortText("Syllabus", request.Syllabus, validate.MaxSyllabus)
	if err != nil {
		return store.NewScheduleItem{}, err
	}
	instructions, err := validate.ShortText("Instructions", request.Instructions, validate.MaxSyllabus)
	if err != nil {
		return store.NewScheduleItem{}, err
	}
	note, err := validate.ShortText("Note", request.Note, validate.MaxNote)
	if err != nil {
		return store.NewScheduleItem{}, err
	}
	return store.NewScheduleItem{
		Batch:        profile.Batch,
		Section:      section,
		CourseID:     request.CourseID,
		Kind:         kind,
		ItemDate:     validate.DateString(date),
		StartTime:    start,
		EndTime:      end,
		Mode:         mode,
		Room:         room,
		MeetingLink:  link,
		Syllabus:     syllabus,
		Instructions: instructions,
		Note:         note,
		CreatedBy:    profile.UserID,
	}, nil
}

func (c *CR) Upcoming() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := c.profile(w, r)
		if !ok {
			return
		}
		from := c.cfg.Now()
		if raw := strings.TrimSpace(r.URL.Query().Get("from")); raw != "" {
			parsed, err := validate.Date("from", raw, c.cfg.Location)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			from = parsed
		}
		items, err := c.classes.ListBatchItems(r.Context(), profile.Batch, validate.DateString(from))
		if err != nil {
			log.Printf("handler: could not list the upcoming items of a class representative: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		bodies := newItemBodies(items)
		for index := range bodies {
			bodies[index].AllowedActions = crActions(items[index])
		}
		httpx.WriteJSON(w, http.StatusOK, itemsResponse{Items: bodies})
	}
}

func (c *CR) RequestReschedule() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := c.profile(w, r)
		if !ok {
			return
		}
		itemID, ok := pathID(w, r)
		if !ok {
			return
		}
		var request proposalRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		proposal, err := c.proposalFrom(request)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		item, err := c.classes.CRRequestReschedule(r.Context(), itemID, profile.Batch, proposal)
		if scheduleError(w, err) {
			return
		}
		c.tellTeacher(r, item, titleCRAsked)
		httpx.WriteJSON(w, http.StatusOK, itemResponse{Item: newItemBody(item), Message: MessageAskedTeacher})
	}
}

func (c *CR) proposalFrom(request proposalRequest) (store.Proposal, error) {
	date, err := validate.Date("Date", request.ItemDate, c.cfg.Location)
	if err != nil {
		return store.Proposal{}, err
	}
	start, end, err := clockRange(request.StartTime, request.EndTime)
	if err != nil {
		return store.Proposal{}, err
	}
	if start == "" {
		return store.Proposal{}, errors.New("Start time is required.")
	}
	mode, room, link, err := slotFields(request.Mode, request.Room, request.MeetingLink)
	if err != nil {
		return store.Proposal{}, err
	}
	return store.Proposal{
		ItemDate:    validate.DateString(date),
		StartTime:   start,
		EndTime:     end,
		Mode:        mode,
		Room:        room,
		MeetingLink: link,
	}, nil
}

func (c *CR) UpdateItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := c.profile(w, r)
		if !ok {
			return
		}
		itemID, ok := pathID(w, r)
		if !ok {
			return
		}
		var request crPatchRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		patch, err := c.patchFrom(request)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		item, err := c.classes.UpdateItemAsCR(r.Context(), itemID, profile.Batch, patch)
		if scheduleError(w, err) {
			return
		}
		c.tellTeacher(r, item, titleCRChanged)
		c.tellStudents(r, item, titleRoutineChanged)
		httpx.WriteJSON(w, http.StatusOK, itemResponse{Item: newItemBody(item), Message: MessageItemUpdated})
	}
}

func (c *CR) patchFrom(request crPatchRequest) (store.ItemPatch, error) {
	var patch store.ItemPatch
	if request.ItemDate != nil {
		date, err := validate.Date("Date", *request.ItemDate, c.cfg.Location)
		if err != nil {
			return store.ItemPatch{}, err
		}
		value := validate.DateString(date)
		patch.ItemDate = &value
	}
	if (request.StartTime == nil) != (request.EndTime == nil) {
		return store.ItemPatch{}, errors.New(MessageTimesTogether)
	}
	if request.StartTime != nil {
		start, end, err := clockRange(*request.StartTime, *request.EndTime)
		if err != nil {
			return store.ItemPatch{}, err
		}
		patch.StartTime = &start
		patch.EndTime = &end
	}
	if request.Mode != nil {
		mode, room, link, err := slotFields(*request.Mode, text(request.Room), text(request.MeetingLink))
		if err != nil {
			return store.ItemPatch{}, err
		}
		patch.Mode = &mode
		patch.Room = &room
		if mode == validate.ModeOnline {
			patch.MeetingLink = &link
		}
	} else if request.Room != nil {
		room, err := validate.Room("Room", *request.Room)
		if err != nil {
			return store.ItemPatch{}, err
		}
		patch.Room = &room
	}
	if request.MeetingLink != nil && patch.MeetingLink == nil {
		link, err := validate.OptionalLink("Meeting link", *request.MeetingLink)
		if err != nil {
			return store.ItemPatch{}, err
		}
		patch.MeetingLink = &link
	}
	if request.Note != nil {
		note, err := validate.ShortText("Note", *request.Note, validate.MaxNote)
		if err != nil {
			return store.ItemPatch{}, err
		}
		patch.Note = &note
	}
	if request.Syllabus != nil {
		syllabus, err := validate.ShortText("Syllabus", *request.Syllabus, validate.MaxSyllabus)
		if err != nil {
			return store.ItemPatch{}, err
		}
		patch.Syllabus = &syllabus
	}
	if request.Instructions != nil {
		instructions, err := validate.ShortText("Instructions", *request.Instructions, validate.MaxSyllabus)
		if err != nil {
			return store.ItemPatch{}, err
		}
		patch.Instructions = &instructions
	}
	if patch == (store.ItemPatch{}) {
		return store.ItemPatch{}, errors.New(MessageNothingToUpdate)
	}
	return patch, nil
}

func (c *CR) CancelItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := c.profile(w, r)
		if !ok {
			return
		}
		itemID, ok := pathID(w, r)
		if !ok {
			return
		}
		var request cancelRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		reason, err := validate.ShortText("Reason", request.Reason, validate.MaxMessageLength)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		item, err := c.classes.CancelItemAsCR(r.Context(), itemID, profile.Batch, reason)
		if scheduleError(w, err) {
			return
		}
		c.tellTeacher(r, item, titleCRCancelled)
		c.tellStudents(r, item, titleStudentCancelled)
		httpx.WriteJSON(w, http.StatusOK, itemResponse{Item: newItemBody(item), Message: MessageItemCancelled})
	}
}

func (c *CR) DeleteItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := c.profile(w, r)
		if !ok {
			return
		}
		itemID, ok := pathID(w, r)
		if !ok {
			return
		}
		item, err := c.classes.DeleteCRItem(r.Context(), itemID, profile.Batch)
		if scheduleError(w, err) {
			return
		}
		c.tellTeacher(r, item, titleCRCancelled)
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageItemRemoved})
	}
}

func (c *CR) AddToFinal() http.HandlerFunc {
	return c.setFinal(true, MessageAddedToFinal)
}

func (c *CR) RemoveFromFinal() http.HandlerFunc {
	return c.setFinal(false, MessageRemovedFromFinal)
}

func (c *CR) setFinal(inFinal bool, message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := c.profile(w, r)
		if !ok {
			return
		}
		itemID, ok := pathID(w, r)
		if !ok {
			return
		}
		item, err := c.classes.SetItemFinal(r.Context(), itemID, profile.Batch, inFinal)
		if scheduleError(w, err) {
			return
		}
		httpx.WriteJSON(w, http.StatusOK, itemResponse{Item: newItemBody(item), Message: message})
	}
}

func (c *CR) Final() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := c.profile(w, r)
		if !ok {
			return
		}
		items, err := c.classes.ListFinalItems(r.Context(), profile.Batch)
		if err != nil {
			log.Printf("handler: could not list the final routine of a class representative: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, groupFinal(items, c.cfg))
	}
}

func (c *CR) FinalPreview() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := c.profile(w, r)
		if !ok {
			return
		}
		ids, err := parseIDs(r.URL.Query().Get("ids"))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		items, err := c.classes.ListItemsByIDs(r.Context(), profile.Batch, ids)
		if err != nil {
			log.Printf("handler: could not preview the final routine: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, groupFinal(items, c.cfg))
	}
}

func (c *CR) Publish() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := c.profile(w, r)
		if !ok {
			return
		}
		var request publishRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		if len(request.ItemIDs) == 0 {
			httpx.WriteError(w, http.StatusBadRequest, MessageIDsRequired)
			return
		}
		publication, changed, err := c.classes.PublishFinal(r.Context(), profile.Batch,
			profile.UserID, request.ItemIDs)
		if scheduleError(w, err) {
			return
		}
		c.announcePublication(r, profile.Batch, changed)
		httpx.WriteJSON(w, http.StatusOK, publishResponse{
			PublicationID: publication.ID,
			Published:     len(request.ItemIDs),
			Changed:       len(changed),
			Message:       MessageRoutinePublished,
		})
	}
}

func (c *CR) announcePublication(r *http.Request, batch string, changed []store.ScheduleItem) {
	err := c.alerts.NotifyBroadcast(r.Context(), auth.RoleStudent, batch, "", notify.TypeRoutine,
		titleRoutinePublished, "The class representative published the routine of "+batch+".",
		notify.Page(notify.LinkRoutine))
	if err != nil {
		log.Printf("handler: could not announce a publication: %v", err)
	}
	for _, item := range changed {
		c.tellStudents(r, item, titleRoutineChanged)
	}
}

func (c *CR) tellTeacher(r *http.Request, item store.ScheduleItem, title string) {
	if item.TeacherUserID == 0 {
		return
	}
	link := notify.Page(notify.LinkTeacherFinal)
	if item.Status == store.ItemCRAsk {
		link = notify.Page(notify.LinkTeacherNextDay)
	}
	err := c.alerts.NotifyUser(r.Context(), item.TeacherUserID, notify.TypeRoutine, title,
		itemSentence(item), link)
	if err != nil {
		log.Printf("handler: could not notify the teacher of a change: %v", err)
	}
}

func (c *CR) tellStudents(r *http.Request, item store.ScheduleItem, title string) {
	if !item.Published() {
		return
	}
	err := c.alerts.NotifyBroadcast(r.Context(), auth.RoleStudent, item.Batch, broadcastSection(item.Section),
		notify.TypeRoutine, title, itemSentence(item), notify.OnDate(notify.LinkRoutine, item.ItemDate))
	if err != nil {
		log.Printf("handler: could not announce a change to the students: %v", err)
	}
}

func text(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func clockRange(rawStart, rawEnd string) (string, string, error) {
	if strings.TrimSpace(rawStart) == "" && strings.TrimSpace(rawEnd) == "" {
		return "", "", nil
	}
	start, err := validate.ClockTime("Start time", rawStart)
	if err != nil {
		return "", "", err
	}
	end, err := validate.ClockTime("End time", rawEnd)
	if err != nil {
		return "", "", err
	}
	if err := validate.ClassWindow(start, end); err != nil {
		return "", "", err
	}
	return start, end, nil
}

func slotFields(rawMode, rawRoom, rawLink string) (string, string, string, error) {
	mode, err := validate.ItemMode("mode", rawMode)
	if err != nil {
		return "", "", "", err
	}
	link, err := validate.OptionalLink("Meeting link", rawLink)
	if err != nil {
		return "", "", "", err
	}
	if mode == validate.ModeOnline {
		if link == "" {
			return "", "", "", errors.New(MessageLinkForOnlineMode)
		}
		return mode, "", link, nil
	}
	room, err := validate.Room("Room", rawRoom)
	if err != nil {
		return "", "", "", errors.New(MessageRoomForRoomMode)
	}
	return mode, room, link, nil
}

func parseIDs(raw string) ([]int64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New(MessageIDsRequired)
	}
	parts := strings.Split(trimmed, ",")
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || value <= 0 {
			return nil, errors.New("ids must be a comma separated list of positive numbers.")
		}
		ids = append(ids, value)
	}
	return ids, nil
}

func groupRoomSlots(slots []store.RoomSlot, batch string, location *time.Location) []roomBody {
	rooms := make([]roomBody, 0)
	roomIndex := make(map[string]int)
	dayIndex := make(map[string]int)
	for _, slot := range slots {
		position, seen := roomIndex[slot.Room]
		if !seen {
			position = len(rooms)
			roomIndex[slot.Room] = position
			rooms = append(rooms, roomBody{Room: slot.Room, Days: make([]roomDayBody, 0)})
		}
		key := slot.Room + ":" + slot.ItemDate
		dayPosition, hasDay := dayIndex[key]
		if !hasDay {
			dayPosition = len(rooms[position].Days)
			dayIndex[key] = dayPosition
			weekday := ""
			if parsed, err := time.ParseInLocation(validate.DateLayout, slot.ItemDate, location); err == nil {
				weekday = validate.DayName(int(parsed.Weekday()))
			}
			rooms[position].Days = append(rooms[position].Days, roomDayBody{
				Date:    slot.ItemDate,
				Weekday: weekday,
				Slots:   make([]roomSlotBody, 0),
			})
		}
		if slot.StartTime == "" {
			continue
		}
		label := LabelUnavailable
		if slot.Batch == batch {
			label = LabelYours
		}
		rooms[position].Days[dayPosition].Slots = append(rooms[position].Days[dayPosition].Slots, roomSlotBody{
			StartTime:  slot.StartTime,
			EndTime:    slot.EndTime,
			Batch:      slot.Batch,
			Section:    slot.Section,
			CourseCode: slot.CourseCode,
			Status:     slot.Status,
			Label:      label,
		})
	}
	return rooms
}

func groupFinal(items []store.ScheduleItem, cfg *config.Config) finalResponse {
	response := finalResponse{Days: make([]finalDayBody, 0)}
	index := make(map[string]int)
	for _, item := range items {
		position, seen := index[item.ItemDate]
		if !seen {
			position = len(response.Days)
			index[item.ItemDate] = position
			weekday := ""
			if parsed, err := time.ParseInLocation(validate.DateLayout, item.ItemDate, cfg.Location); err == nil {
				weekday = validate.DayName(int(parsed.Weekday()))
			}
			response.Days = append(response.Days, finalDayBody{
				Date:    item.ItemDate,
				Weekday: weekday,
				Items:   make([]itemBody, 0),
			})
		}
		body := newItemBody(item)
		body.AllowedActions = crActions(item)
		response.Days[position].Items = append(response.Days[position].Items, body)
		if !item.HasTimes() {
			response.NeedsTime++
		}
		if item.ChangedSincePublish() {
			response.ChangedSincePublish++
		}
	}
	return response
}
