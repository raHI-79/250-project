package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/attendance"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"math"
	"net/http"
	"strings"
	"time"
)

const (
	MessagePhoneApproved  = "The phone is approved and the student was told."
	MessagePhoneRejected  = "The request is refused and the student was told."
	MessageNoteForChange  = "This student already used both phone changes this semester, so add a note to approve."
	MessageRequestGone    = "This phone request was not found."
	MessageRequestClosed  = "Another teacher already answered this request."
	MessageNotYourStudent = "This student is not in a batch you teach this semester."

	auditPhoneApproved = "phone_change_approved"
	auditPhoneRejected = "phone_change_rejected"
	historyLimit       = 200
)

type PhoneStore interface {
	GetTeacherByUser(ctx context.Context, userID int64) (store.Teacher, error)
	ListTeacherDeviceRequests(ctx context.Context, teacherUserID int64) ([]store.DeviceChangeRequest, error)
	ListDeviceRequestHistory(ctx context.Context, limit int) ([]store.DeviceChangeRequest, error)
	DeviceRequest(ctx context.Context, requestID int64) (store.DeviceChangeRequest, error)
	DecideDeviceChange(ctx context.Context, requestID, deciderUserID int64, approve bool,
		note string, changeLimit int) (store.DeviceChangeRequest, error)
	ApprovedDeviceChanges(ctx context.Context, userID int64, semester string) (int, error)
	FindAccount(ctx context.Context, userID int64) (store.AccountRef, error)
	WriteAudit(ctx context.Context, adminID int64, action, targetSummary string) error
	ListBatchClasses(ctx context.Context, batch string, from, to time.Time) ([]store.ClassSummary, error)
	BatchClassRates(ctx context.Context, courseID int64, batch, section string,
		from, to time.Time) ([]store.StudentRate, error)
	BatchExportRows(ctx context.Context, courseID int64, batch, section string,
		from, to time.Time) ([]store.ExportRow, error)
	ListBatches(ctx context.Context) ([]string, error)
}

type Phones struct {
	cfg    *config.Config
	store  PhoneStore
	alerts BroadcastNotifier
}

func NewPhones(cfg *config.Config, phones PhoneStore, alerts BroadcastNotifier) *Phones {
	return &Phones{cfg: cfg, store: phones, alerts: alerts}
}

type decideRequest struct {
	Note string `json:"note"`
}

type requestBody struct {
	ID           int64  `json:"id"`
	StudentName  string `json:"student_name"`
	RegNo        string `json:"reg_no"`
	Batch        string `json:"batch"`
	Section      string `json:"section"`
	Reason       string `json:"reason"`
	Status       string `json:"status"`
	Semester     string `json:"semester"`
	Changes      int    `json:"changes"`
	NeedsNote    bool   `json:"needs_note"`
	DeciderName  string `json:"decider_name,omitempty"`
	DecisionNote string `json:"decision_note,omitempty"`
	DecidedAt    string `json:"decided_at,omitempty"`
	CreatedAt    string `json:"created_at"`
}

type requestsResponse struct {
	Items []requestBody `json:"items"`
	Limit int           `json:"limit"`
}

func (p *Phones) body(ctx context.Context, request store.DeviceChangeRequest) requestBody {
	body := requestBody{
		ID:           request.ID,
		StudentName:  request.StudentName,
		RegNo:        request.RegNo,
		Batch:        request.Batch,
		Section:      request.Section,
		Reason:       request.Reason,
		Status:       request.Status,
		Semester:     request.Semester,
		DeciderName:  request.DeciderName,
		DecisionNote: request.DecisionNote,
		CreatedAt:    timestamp(request.CreatedAt),
	}
	if !request.DecidedAt.IsZero() {
		body.DecidedAt = timestamp(request.DecidedAt)
	}
	if changes, err := p.store.ApprovedDeviceChanges(ctx, request.UserID, request.Semester); err == nil {
		body.Changes = changes
		body.NeedsNote = attendance.NoteRequired(changes)
	}
	return body
}

func (p *Phones) TeacherList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		if _, err := p.store.GetTeacherByUser(r.Context(), current.ID); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				httpx.WriteError(w, http.StatusForbidden, MessageNotATeacher)
				return
			}
			log.Printf("handler: could not load the teacher of a phone request list: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		requests, err := p.store.ListTeacherDeviceRequests(r.Context(), current.ID)
		if err != nil {
			log.Printf("handler: could not list the phone requests of a teacher: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]requestBody, 0, len(requests))
		for _, request := range requests {
			items = append(items, p.body(r.Context(), request))
		}
		httpx.WriteJSON(w, http.StatusOK, requestsResponse{
			Items: items,
			Limit: attendance.SemesterChangeLimit,
		})
	}
}

func (p *Phones) Decide(approve bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		requestID, ok := pathID(w, r)
		if !ok {
			return
		}
		var body decideRequest
		if r.ContentLength > 0 && !httpx.DecodeJSON(w, r, &body) {
			return
		}
		note, err := validate.Description("Note", body.Note)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		decided, err := p.store.DecideDeviceChange(r.Context(), requestID, current.ID,
			approve, note, attendance.SemesterChangeLimit)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageRequestGone)
			return
		case errors.Is(err, store.ErrRequestDecided):
			httpx.WriteError(w, http.StatusConflict, MessageRequestClosed)
			return
		case errors.Is(err, store.ErrNotTheirStudent):
			httpx.WriteError(w, http.StatusForbidden, MessageNotYourStudent)
			return
		case errors.Is(err, store.ErrNoteRequired):
			httpx.WriteError(w, http.StatusBadRequest, MessageNoteForChange)
			return
		case err != nil:
			log.Printf("handler: could not decide a phone request: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		p.audit(r.Context(), current.ID, decided, approve)
		p.tellStudent(r.Context(), current.ID, decided, approve)
		message := MessagePhoneRejected
		if approve {
			message = MessagePhoneApproved
		}
		httpx.WriteJSON(w, http.StatusOK, struct {
			Message string      `json:"message"`
			Request requestBody `json:"request"`
		}{Message: message, Request: p.body(r.Context(), decided)})
	}
}

func (p *Phones) audit(ctx context.Context, deciderID int64, request store.DeviceChangeRequest,
	approve bool) {
	action := auditPhoneRejected
	if approve {
		action = auditPhoneApproved
	}
	summary := request.StudentName + " (" + request.RegNo + ")"
	if request.DecisionNote != "" {
		summary += " · " + request.DecisionNote
	}
	if err := p.store.WriteAudit(ctx, deciderID, action, summary); err != nil {
		log.Printf("handler: could not record a phone decision: %v", err)
	}
}

func (p *Phones) tellStudent(ctx context.Context, deciderID int64,
	request store.DeviceChangeRequest, approve bool) {
	name := request.DeciderName
	if name == "" {
		if account, err := p.store.FindAccount(ctx, deciderID); err == nil {
			name = account.Name
		}
	}
	title := titlePhoneRefused
	body := name + " refused your new attendance phone."
	if approve {
		title = titlePhoneAllowed
		body = name + bodyPhoneAllowed
	}
	if request.DecisionNote != "" {
		body += " " + request.DecisionNote
	}
	err := p.alerts.NotifyUser(ctx, request.UserID, notify.TypeDecision, title, body,
		notify.Page(notify.LinkAttendMark))
	if err != nil {
		log.Printf("handler: could not tell the student %d about a phone decision: %v",
			request.UserID, err)
	}
}

func (p *Phones) AdminHistory() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requests, err := p.store.ListDeviceRequestHistory(r.Context(), historyLimit)
		if err != nil {
			log.Printf("handler: could not list the phone change history: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]requestBody, 0, len(requests))
		for _, request := range requests {
			items = append(items, p.body(r.Context(), request))
		}
		httpx.WriteJSON(w, http.StatusOK, requestsResponse{
			Items: items,
			Limit: attendance.SemesterChangeLimit,
		})
	}
}

func (p *Phones) window(r *http.Request) (time.Time, time.Time) {
	now := p.cfg.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	from := midnight.AddDate(0, 0, -defaultHistoryDays)
	to := midnight.AddDate(0, 0, 1)
	if parsed, err := time.ParseInLocation(validate.DateLayout,
		strings.TrimSpace(r.URL.Query().Get("from")), now.Location()); err == nil {
		from = parsed
	}
	if parsed, err := time.ParseInLocation(validate.DateLayout,
		strings.TrimSpace(r.URL.Query().Get("to")), now.Location()); err == nil {
		to = parsed.AddDate(0, 0, 1)
	}
	return from, to
}

func (p *Phones) batches(ctx context.Context, wanted string) ([]string, error) {
	if wanted != "" {
		return []string{wanted}, nil
	}
	return p.store.ListBatches(ctx)
}

func (p *Phones) AdminOverview() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from, to := p.window(r)
		wanted := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("batch")))
		batches, err := p.batches(r.Context(), wanted)
		if err != nil {
			log.Printf("handler: could not list the batches of the attendance overview: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]classBody, 0)
		for _, batch := range batches {
			classes, err := p.store.ListBatchClasses(r.Context(), batch, from, to)
			if err != nil {
				log.Printf("handler: could not list the attendance classes of %s: %v", batch, err)
				httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
				return
			}
			for _, class := range classes {
				rates, err := p.store.BatchClassRates(r.Context(), class.CourseID,
					class.Batch, class.Section, from, to)
				if err != nil {
					log.Printf("handler: could not count the attendance of %s: %v", batch, err)
					httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
					return
				}
				total, below := 0.0, 0
				for _, rate := range rates {
					share := attendance.Percentage(rate.Present, rate.Held)
					total += share
					if share < attendance.AlertPercentage {
						below++
					}
				}
				average := 0
				if len(rates) > 0 {
					average = int(math.Round(total / float64(len(rates))))
				}
				items = append(items, classBody{
					CourseID:    class.CourseID,
					CourseCode:  class.CourseCode,
					CourseTitle: class.CourseTitle,
					Batch:       class.Batch,
					Section:     class.Section,
					Sessions:    class.Sessions,
					AverageRate: average,
					BelowFifty:  below,
				})
			}
		}
		httpx.WriteJSON(w, http.StatusOK, classesResponse{Items: items})
	}
}

func (p *Phones) AdminExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from, to := p.window(r)
		wanted := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("batch")))
		batches, err := p.batches(r.Context(), wanted)
		if err != nil {
			log.Printf("handler: could not list the batches of an attendance export: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		sheets := make([]attendance.ExportClass, 0)
		for _, batch := range batches {
			classes, err := p.store.ListBatchClasses(r.Context(), batch, from, to)
			if err != nil {
				log.Printf("handler: could not list the classes of an attendance export: %v", err)
				httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
				return
			}
			for _, class := range classes {
				rows, err := p.store.BatchExportRows(r.Context(), class.CourseID,
					class.Batch, class.Section, from, to)
				if err != nil {
					log.Printf("handler: could not read the rows of an attendance export: %v", err)
					httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
					return
				}
				sheets = append(sheets, attendance.ExportClass{
					CourseCode:  class.CourseCode,
					CourseTitle: class.CourseTitle,
					Batch:       class.Batch,
					Section:     class.Section,
					Sessions:    class.Sessions,
					Rows:        exportRows(rows),
				})
			}
		}
		if len(sheets) == 0 {
			httpx.WriteError(w, http.StatusNotFound, MessageNoWorkbook)
			return
		}
		writeWorkbook(w, sheets, true, from, to)
	}
}
