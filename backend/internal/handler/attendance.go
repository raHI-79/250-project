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
	"strconv"
	"strings"
	"time"
)

const (
	MessageAttendanceOpened  = "Attendance is open."
	MessageAttendanceClosed  = "Attendance closed."
	MessageAttendanceLonger  = "Attendance stays open longer."
	MessageMarkSaved         = "You are marked present."
	MessageMarkEdited        = "The attendance of this student is updated."
	MessageNoSession         = "No attendance is open for your classes right now."
	MessageNotYourClass      = "This class is not yours in the running semester."
	MessageSessionGone       = "This attendance session was not found."
	MessageAttendanceOver    = "This attendance is already closed."
	MessageWrongMarkStatus   = "A student is either present or absent."
	MessageDeviceIDRequired  = "The app did not send the phone id. Update the app and try again."
	MessageAttendanceAndroid = attendance.MessageWebsiteOnly

	titleAttendanceOpen = "Attendance open · "
	bodyAttendanceOpen  = "Tap to mark yourself present from your registered phone."
	defaultHistoryDays  = 120
)

type AttendanceStore interface {
	GetTeacherByUser(ctx context.Context, userID int64) (store.Teacher, error)
	ListTeacherRoutine(ctx context.Context, teacherID int64) ([]store.RoutineRow, error)
	ListTeacherDayItems(ctx context.Context, teacherID int64, date string) ([]store.ScheduleItem, error)
	OpenAttendanceSession(ctx context.Context, fresh store.NewAttendanceSession) (store.AttendanceSession, error)
	AttendanceSession(ctx context.Context, sessionID int64) (store.AttendanceSession, error)
	SetAttendanceLocation(ctx context.Context, sessionID, teacherID int64, lat, lng, accuracy float64) error
	ExtendAttendanceSession(ctx context.Context, sessionID, teacherID int64,
		minutes int) (store.AttendanceSession, error)
	CloseAttendanceSession(ctx context.Context, sessionID, teacherID int64) (store.AttendanceSession, error)
	ListTeacherAttendance(ctx context.Context, teacherID int64, from, to time.Time) ([]store.AttendanceSession, error)
	ListSessionRecords(ctx context.Context, sessionID int64) ([]store.AttendanceRecord, error)
	ListRoster(ctx context.Context, sessionID int64) ([]store.RosterStudent, error)
	EditAttendanceRecord(ctx context.Context, sessionID, studentID, editorUserID int64,
		status, note string) error
	ListTeacherClasses(ctx context.Context, teacherID int64, from, to time.Time) ([]store.ClassSummary, error)
	ClassRates(ctx context.Context, teacherID, courseID int64, batch, section string,
		from, to time.Time) ([]store.StudentRate, error)
	TeacherExportRows(ctx context.Context, teacherID, courseID int64, batch, section string,
		from, to time.Time) ([]store.ExportRow, error)
	WriteAudit(ctx context.Context, adminID int64, action, targetSummary string) error
}

type Attendance struct {
	cfg    *config.Config
	store  AttendanceStore
	alerts BroadcastNotifier
}

func NewAttendance(cfg *config.Config, records AttendanceStore, alerts BroadcastNotifier) *Attendance {
	return &Attendance{cfg: cfg, store: records, alerts: alerts}
}

type sessionBody struct {
	ID          int64  `json:"id"`
	CourseID    int64  `json:"course_id"`
	CourseCode  string `json:"course_code"`
	CourseTitle string `json:"course_title"`
	Batch       string `json:"batch"`
	Section     string `json:"section"`
	TeacherName string `json:"teacher_name"`
	StartedAt   string `json:"started_at"`
	EndsAt      string `json:"ends_at"`
	Status      string `json:"status"`
	Present     int    `json:"present"`
	Roster      int    `json:"roster"`
	Code        string `json:"code,omitempty"`
	CodeSeconds int    `json:"code_seconds,omitempty"`
	TokenKey    string `json:"token_key,omitempty"`
	HasLocation bool   `json:"has_location"`
	LocationAt  string `json:"location_updated_at,omitempty"`
	AccuracyM   int    `json:"accuracy_m,omitempty"`
}

type markBody struct {
	StudentID int64  `json:"student_id"`
	UserID    int64  `json:"user_id"`
	FullName  string `json:"full_name"`
	RegNo     string `json:"reg_no"`
	Section   string `json:"section"`
	Status    string `json:"status"`
	Method    string `json:"method"`
	DistanceM int    `json:"distance_m,omitempty"`
	AccuracyM int    `json:"accuracy_m,omitempty"`
	MarkedAt  string `json:"marked_at,omitempty"`
	Note      string `json:"note,omitempty"`
	EditedBy  string `json:"edited_by,omitempty"`
	EditedAt  string `json:"edited_at,omitempty"`
}

type classBody struct {
	CourseID    int64  `json:"course_id"`
	CourseCode  string `json:"course_code"`
	CourseTitle string `json:"course_title"`
	Batch       string `json:"batch"`
	Section     string `json:"section"`
	Sessions    int    `json:"sessions"`
	AverageRate int    `json:"average_rate"`
	BelowFifty  int    `json:"below_fifty"`
}

type sessionsResponse struct {
	Items []sessionBody `json:"items"`
}

type sessionDetailResponse struct {
	Session sessionBody `json:"session"`
	Marks   []markBody  `json:"marks"`
}

type classesResponse struct {
	Items []classBody `json:"items"`
}

func newSessionBody(session store.AttendanceSession, now time.Time, forTeacher bool) sessionBody {
	body := sessionBody{
		ID:          session.ID,
		CourseID:    session.CourseID,
		CourseCode:  session.CourseCode,
		CourseTitle: session.CourseTitle,
		Batch:       session.Batch,
		Section:     session.Section,
		TeacherName: session.TeacherName,
		StartedAt:   timestamp(session.StartedAt),
		EndsAt:      timestamp(session.EndsAt),
		Status:      session.Status,
		Present:     session.Present,
		Roster:      session.Roster,
		HasLocation: session.HasLocation,
	}
	if !session.LocationAt.IsZero() {
		body.LocationAt = timestamp(session.LocationAt)
	}
	if forTeacher && session.Status == attendance.StatusOpen {
		body.Code = attendance.CurrentCode(session.TokenKey, session.ID, now)
		body.CodeSeconds = attendance.CodeSliceSeconds -
			int(now.Unix()%int64(attendance.CodeSliceSeconds))
		body.TokenKey = session.TokenKey
		if session.HasLocation {
			body.AccuracyM = int(math.Round(session.TeacherAccuracy))
		}
	}
	return body
}

func newMarkBody(record store.AttendanceRecord) markBody {
	body := markBody{
		StudentID: record.StudentID,
		UserID:    record.UserID,
		FullName:  record.FullName,
		RegNo:     record.RegNo,
		Section:   record.Section,
		Status:    record.Status,
		Method:    record.Method,
		Note:      record.Note,
		EditedBy:  record.EditedBy,
	}
	if record.HasPosition {
		body.DistanceM = int(math.Round(record.DistanceM))
		body.AccuracyM = int(math.Round(record.AccuracyM))
	}
	if record.HasMark {
		body.MarkedAt = timestamp(record.MarkedAt)
	}
	if !record.EditedAt.IsZero() {
		body.EditedAt = timestamp(record.EditedAt)
	}
	return body
}

func newMarkBodies(records []store.AttendanceRecord) []markBody {
	marks := make([]markBody, 0, len(records))
	for _, record := range records {
		marks = append(marks, newMarkBody(record))
	}
	return marks
}

func (a *Attendance) teacher(w http.ResponseWriter, r *http.Request) (store.Teacher, bool) {
	current, ok := middleware.CurrentUser(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
		return store.Teacher{}, false
	}
	teacher, err := a.store.GetTeacherByUser(r.Context(), current.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusForbidden, MessageNotATeacher)
			return store.Teacher{}, false
		}
		log.Printf("handler: could not load the teacher of an attendance request: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return store.Teacher{}, false
	}
	return teacher, true
}

func (a *Attendance) ownSession(w http.ResponseWriter, r *http.Request,
	teacher store.Teacher) (store.AttendanceSession, bool) {
	sessionID, ok := pathID(w, r)
	if !ok {
		return store.AttendanceSession{}, false
	}
	session, err := a.store.AttendanceSession(r.Context(), sessionID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, MessageSessionGone)
		return store.AttendanceSession{}, false
	}
	if err != nil {
		log.Printf("handler: could not load an attendance session: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return store.AttendanceSession{}, false
	}
	if session.TeacherID != teacher.ID {
		httpx.WriteError(w, http.StatusForbidden, middleware.MessageAccessDenied)
		return store.AttendanceSession{}, false
	}
	return session, true
}

func (a *Attendance) window(r *http.Request) (time.Time, time.Time) {
	now := a.cfg.Now()
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).
		AddDate(0, 0, -defaultHistoryDays)
	to := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
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

func (a *Attendance) teaches(ctx context.Context, teacherID, courseID int64,
	batch, section string) (bool, error) {
	rows, err := a.store.ListTeacherRoutine(ctx, teacherID)
	if err != nil {
		return false, err
	}
	for _, row := range rows {
		if row.CourseID != courseID || row.Batch != batch {
			continue
		}
		if row.Section == section || row.Section == validate.SectionAll ||
			section == validate.SectionAll {
			return true, nil
		}
	}
	return false, nil
}

func (a *Attendance) announce(ctx context.Context, session store.AttendanceSession) {
	roster, err := a.store.ListRoster(ctx, session.ID)
	if err != nil {
		log.Printf("handler: could not list the roster of a new attendance session: %v", err)
		return
	}
	title := titleAttendanceOpen + session.CourseTitle
	link := notify.Ref(notify.LinkAttendMark, session.ID)
	for _, entry := range roster {
		err := a.alerts.PushUser(ctx, entry.UserID, notify.TypeAttend, title, bodyAttendanceOpen, link)
		if err != nil {
			log.Printf("handler: could not tell the student %d about an open attendance: %v",
				entry.UserID, err)
		}
	}
}

func percentText(present, held int) int {
	return int(math.Round(attendance.Percentage(present, held)))
}

func sessionIDQuery(r *http.Request, key string) int64 {
	value, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get(key)), 10, 64)
	if err != nil || value <= 0 {
		return 0
	}
	return value
}
