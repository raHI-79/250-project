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
	"strings"
	"time"
)

const (
	MessageNotATeacher     = "No teacher profile found for this account."
	MessageItemConfirmed   = "Class confirmed."
	MessageItemCancelled   = "Class cancelled."
	MessageRescheduleAsked = "Reschedule requested. The class representative picks a new slot."
	MessageProposalTaken   = "New slot accepted."
	MessageProposalLeft    = "The original slot stays."
	MessageExamCreated     = "Exam added. The class representative sets the time and the room."
	MessageExamUpdated     = "Exam updated."
	MessageExamCancelled   = "Exam cancelled."
	MessageExamDeleted     = "Exam deleted."
	MessagePastDate        = "Date must be today or later."

	titleTeacherConfirmed = "Class confirmed by the teacher"
	titleTeacherCancelled = "Class cancelled by the teacher"
	titleTeacherAsked     = "The teacher asked for another slot"
	titleProposalTaken    = "The teacher accepted the new slot"
	titleProposalLeft     = "The teacher kept the original slot"
	titleExamAdded        = "New exam from the teacher"
	titleExamChanged      = "Exam changed by the teacher"
	titleExamCancelled    = "Exam cancelled by the teacher"
	titleStudentCancelled = "Class cancelled"
	titleStudentChanged   = "Class changed"
)

type TeacherStore interface {
	GetTeacherByUser(ctx context.Context, userID int64) (store.Teacher, error)
	ListTeacherRoutine(ctx context.Context, teacherID int64) ([]store.RoutineRow, error)
	ListTeacherDayItems(ctx context.Context, teacherID int64, date string) ([]store.ScheduleItem, error)
	ListTeacherFinalItems(ctx context.Context, teacherID int64, from string) ([]store.ScheduleItem, error)
	ListTeacherExams(ctx context.Context, teacherID int64, from string) ([]store.ScheduleItem, error)
	ConfirmItem(ctx context.Context, itemID, teacherID int64, mode, room, link string,
		now time.Time) (store.ScheduleItem, error)
	CancelItemAsTeacher(ctx context.Context, itemID, teacherID int64, reason string,
		now time.Time) (store.ScheduleItem, error)
	TeacherRequestReschedule(ctx context.Context, itemID, teacherID int64, message string,
		now time.Time) (store.ScheduleItem, error)
	AcceptProposal(ctx context.Context, itemID, teacherID int64) (store.ScheduleItem, error)
	DeclineProposal(ctx context.Context, itemID, teacherID int64) (store.ScheduleItem, error)
	CreateTeacherExam(ctx context.Context, exam store.NewScheduleItem) (store.ScheduleItem, error)
	UpdateTeacherExam(ctx context.Context, itemID, teacherID int64, patch store.ItemPatch) (store.ScheduleItem, error)
	CancelTeacherExam(ctx context.Context, itemID, teacherID int64, reason string) (store.ScheduleItem, error)
	DeleteTeacherExam(ctx context.Context, itemID, teacherID int64) (store.ScheduleItem, error)
	ListActiveCRs(ctx context.Context, batch, section string) ([]int64, error)
}

type Teacher struct {
	cfg      *config.Config
	teachers TeacherStore
	alerts   BroadcastNotifier
}

func NewTeacher(cfg *config.Config, teachers TeacherStore, alerts BroadcastNotifier) *Teacher {
	return &Teacher{cfg: cfg, teachers: teachers, alerts: alerts}
}

type nextDayResponse struct {
	Date    string     `json:"date"`
	Weekday string     `json:"weekday"`
	Items   []itemBody `json:"items"`
}

type confirmRequest struct {
	Mode        string `json:"mode"`
	Room        string `json:"room"`
	MeetingLink string `json:"meeting_link"`
}

type cancelRequest struct {
	Reason string `json:"reason"`
}

type messageRequest struct {
	Message string `json:"message"`
}

type examRequest struct {
	CourseID     int64  `json:"course_id"`
	Batch        string `json:"batch"`
	Section      string `json:"section"`
	ItemDate     string `json:"item_date"`
	Syllabus     string `json:"syllabus"`
	Instructions string `json:"instructions"`
}

type examPatchRequest struct {
	ItemDate     *string `json:"item_date"`
	Syllabus     *string `json:"syllabus"`
	Instructions *string `json:"instructions"`
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

func (t *Teacher) FixedRoutine() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := t.profile(w, r)
		if !ok {
			return
		}
		entries, err := t.teachers.ListTeacherRoutine(r.Context(), teacher.ID)
		if err != nil {
			log.Printf("handler: could not list the routine of a teacher: %v", err)
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

func (t *Teacher) NextDay() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := t.profile(w, r)
		if !ok {
			return
		}
		now := t.cfg.Now()
		tomorrow := now.AddDate(0, 0, 1)
		day := validate.DateString(tomorrow)
		items, err := t.teachers.ListTeacherDayItems(r.Context(), teacher.ID, day)
		if err != nil {
			log.Printf("handler: could not list the next day of a teacher: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, nextDayResponse{
			Date:    day,
			Weekday: validate.DayName(int(tomorrow.Weekday())),
			Items:   newItemBodies(items),
		})
	}
}

func (t *Teacher) FinalRoutine() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := t.profile(w, r)
		if !ok {
			return
		}
		from := validate.DateString(t.cfg.Now())
		items, err := t.teachers.ListTeacherFinalItems(r.Context(), teacher.ID, from)
		if err != nil {
			log.Printf("handler: could not list the final routine of a teacher: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, itemsResponse{Items: newItemBodies(items)})
	}
}

func (t *Teacher) Confirm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := t.profile(w, r)
		if !ok {
			return
		}
		itemID, ok := pathID(w, r)
		if !ok {
			return
		}
		var request confirmRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		mode, err := validate.ItemMode("mode", request.Mode)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		link, err := validate.OptionalLink("Meeting link", request.MeetingLink)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		room, link, err := validate.Slot(mode, request.Room, link)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		now := t.cfg.Now()
		item, err := t.teachers.ConfirmItem(r.Context(), itemID, teacher.ID, mode, room, link, now)
		if scheduleError(w, err) {
			return
		}
		t.tellCRs(r, item, notify.TypeRoutine, titleTeacherConfirmed)
		t.tellStudents(r, item, titleStudentChanged)
		httpx.WriteJSON(w, http.StatusOK, itemResponse{Item: newItemBody(item), Message: MessageItemConfirmed})
	}
}

func (t *Teacher) Cancel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := t.profile(w, r)
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
		now := t.cfg.Now()
		item, err := t.teachers.CancelItemAsTeacher(r.Context(), itemID, teacher.ID, reason, now)
		if scheduleError(w, err) {
			return
		}
		t.tellCRs(r, item, notify.TypeRoutine, titleTeacherCancelled)
		t.tellStudents(r, item, titleStudentCancelled)
		httpx.WriteJSON(w, http.StatusOK, itemResponse{Item: newItemBody(item), Message: MessageItemCancelled})
	}
}

func (t *Teacher) RequestReschedule() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := t.profile(w, r)
		if !ok {
			return
		}
		itemID, ok := pathID(w, r)
		if !ok {
			return
		}
		var request messageRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		message, err := validate.Message("Message", request.Message)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		now := t.cfg.Now()
		item, err := t.teachers.TeacherRequestReschedule(r.Context(), itemID, teacher.ID, message, now)
		if scheduleError(w, err) {
			return
		}
		t.tellCRs(r, item, notify.TypeRoutine, titleTeacherAsked)
		httpx.WriteJSON(w, http.StatusOK, itemResponse{Item: newItemBody(item), Message: MessageRescheduleAsked})
	}
}

func (t *Teacher) AcceptReschedule() http.HandlerFunc {
	return t.decideProposal(true)
}

func (t *Teacher) DeclineReschedule() http.HandlerFunc {
	return t.decideProposal(false)
}

func (t *Teacher) decideProposal(accept bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := t.profile(w, r)
		if !ok {
			return
		}
		itemID, ok := pathID(w, r)
		if !ok {
			return
		}
		var item store.ScheduleItem
		var err error
		if accept {
			item, err = t.teachers.AcceptProposal(r.Context(), itemID, teacher.ID)
		} else {
			item, err = t.teachers.DeclineProposal(r.Context(), itemID, teacher.ID)
		}
		if scheduleError(w, err) {
			return
		}
		title, message := titleProposalLeft, MessageProposalLeft
		if accept {
			title, message = titleProposalTaken, MessageProposalTaken
		}
		t.tellCRs(r, item, notify.TypeRoutine, title)
		if accept {
			t.tellStudents(r, item, titleStudentChanged)
		}
		httpx.WriteJSON(w, http.StatusOK, itemResponse{Item: newItemBody(item), Message: message})
	}
}

func (t *Teacher) Exams() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := t.profile(w, r)
		if !ok {
			return
		}
		from := validate.DateString(t.cfg.Now())
		items, err := t.teachers.ListTeacherExams(r.Context(), teacher.ID, from)
		if err != nil {
			log.Printf("handler: could not list the exams of a teacher: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, itemsResponse{Items: newItemBodies(items)})
	}
}

func (t *Teacher) CreateExam() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := t.profile(w, r)
		if !ok {
			return
		}
		var request examRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		exam, err := t.examFrom(request, teacher)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		item, err := t.teachers.CreateTeacherExam(r.Context(), exam)
		if scheduleError(w, err) {
			return
		}
		t.tellCRs(r, item, notify.TypeExam, titleExamAdded)
		httpx.WriteJSON(w, http.StatusCreated, itemResponse{Item: newItemBody(item), Message: MessageExamCreated})
	}
}

func (t *Teacher) examFrom(request examRequest, teacher store.Teacher) (store.NewScheduleItem, error) {
	if request.CourseID <= 0 {
		return store.NewScheduleItem{}, errors.New("course_id must be a positive number.")
	}
	batch, err := validate.Batch("Batch", request.Batch)
	if err != nil {
		return store.NewScheduleItem{}, err
	}
	section := validate.SectionAll
	if strings.TrimSpace(request.Section) != "" {
		section, err = validate.RoutineSection("Section", request.Section)
		if err != nil {
			return store.NewScheduleItem{}, err
		}
	}
	date, err := validate.Date("Date", request.ItemDate, t.cfg.Location)
	if err != nil {
		return store.NewScheduleItem{}, err
	}
	if validate.DateString(date) < validate.DateString(t.cfg.Now()) {
		return store.NewScheduleItem{}, errors.New(MessagePastDate)
	}
	syllabus, err := validate.ShortText("Syllabus", request.Syllabus, validate.MaxSyllabus)
	if err != nil {
		return store.NewScheduleItem{}, err
	}
	instructions, err := validate.ShortText("Instructions", request.Instructions, validate.MaxSyllabus)
	if err != nil {
		return store.NewScheduleItem{}, err
	}
	return store.NewScheduleItem{
		Batch:        batch,
		Section:      section,
		CourseID:     request.CourseID,
		TeacherID:    teacher.ID,
		ItemDate:     validate.DateString(date),
		Syllabus:     syllabus,
		Instructions: instructions,
		CreatedBy:    teacher.UserID,
	}, nil
}

func (t *Teacher) UpdateExam() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := t.profile(w, r)
		if !ok {
			return
		}
		itemID, ok := pathID(w, r)
		if !ok {
			return
		}
		var request examPatchRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		patch, err := t.examPatch(request)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		item, err := t.teachers.UpdateTeacherExam(r.Context(), itemID, teacher.ID, patch)
		if scheduleError(w, err) {
			return
		}
		t.tellCRs(r, item, notify.TypeExam, titleExamChanged)
		t.tellStudents(r, item, titleStudentChanged)
		httpx.WriteJSON(w, http.StatusOK, itemResponse{Item: newItemBody(item), Message: MessageExamUpdated})
	}
}

func (t *Teacher) examPatch(request examPatchRequest) (store.ItemPatch, error) {
	var patch store.ItemPatch
	if request.ItemDate != nil {
		date, err := validate.Date("Date", *request.ItemDate, t.cfg.Location)
		if err != nil {
			return store.ItemPatch{}, err
		}
		if validate.DateString(date) < validate.DateString(t.cfg.Now()) {
			return store.ItemPatch{}, errors.New(MessagePastDate)
		}
		value := validate.DateString(date)
		patch.ItemDate = &value
	}
	if request.Syllabus != nil {
		value, err := validate.ShortText("Syllabus", *request.Syllabus, validate.MaxSyllabus)
		if err != nil {
			return store.ItemPatch{}, err
		}
		patch.Syllabus = &value
	}
	if request.Instructions != nil {
		value, err := validate.ShortText("Instructions", *request.Instructions, validate.MaxSyllabus)
		if err != nil {
			return store.ItemPatch{}, err
		}
		patch.Instructions = &value
	}
	if patch.ItemDate == nil && patch.Syllabus == nil && patch.Instructions == nil {
		return store.ItemPatch{}, errors.New(MessageNothingToUpdate)
	}
	return patch, nil
}

func (t *Teacher) CancelExam() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := t.profile(w, r)
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
		item, err := t.teachers.CancelTeacherExam(r.Context(), itemID, teacher.ID, reason)
		if scheduleError(w, err) {
			return
		}
		t.tellCRs(r, item, notify.TypeExam, titleExamCancelled)
		t.tellStudents(r, item, titleStudentCancelled)
		httpx.WriteJSON(w, http.StatusOK, itemResponse{Item: newItemBody(item), Message: MessageExamCancelled})
	}
}

func (t *Teacher) DeleteExam() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := t.profile(w, r)
		if !ok {
			return
		}
		itemID, ok := pathID(w, r)
		if !ok {
			return
		}
		item, err := t.teachers.DeleteTeacherExam(r.Context(), itemID, teacher.ID)
		if scheduleError(w, err) {
			return
		}
		t.tellCRs(r, item, notify.TypeExam, titleExamCancelled)
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageExamDeleted})
	}
}

func (t *Teacher) tellCRs(r *http.Request, item store.ScheduleItem, kind, title string) {
	crs, err := t.teachers.ListActiveCRs(r.Context(), item.Batch, item.Section)
	if err != nil {
		log.Printf("handler: could not list the class representatives of %s: %v", item.Batch, err)
		return
	}
	link := notify.Ref(notify.LinkCRUpcoming, item.ID)
	if kind == notify.TypeExam {
		link = notify.Page(notify.LinkCRFinal)
	}
	for _, userID := range crs {
		if err := t.alerts.NotifyUser(r.Context(), userID, kind, title, itemSentence(item), link); err != nil {
			log.Printf("handler: could not notify the class representative %d: %v", userID, err)
		}
	}
}

func (t *Teacher) tellStudents(r *http.Request, item store.ScheduleItem, title string) {
	if !item.Published() {
		return
	}
	err := t.alerts.NotifyBroadcast(r.Context(), auth.RoleStudent, item.Batch, broadcastSection(item.Section),
		notify.TypeRoutine, title, itemSentence(item), notify.OnDate(notify.LinkRoutine, item.ItemDate))
	if err != nil {
		log.Printf("handler: could not announce a change to the students: %v", err)
	}
}
