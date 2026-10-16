package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	crBatch   = "CSE-23"
	crSection = "A"
	crUserID  = int64(31)
)

func crDates(t *testing.T) (string, string, string) {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load the timezone: %v", err)
	}
	now := time.Now().In(loc)
	return now.AddDate(0, 0, 1).Format(validate.DateLayout),
		now.Format(validate.DateLayout),
		now.AddDate(0, 0, -7).Format(validate.DateLayout)
}

func crRequestBody(date string) string {
	return `{"event_type":"class","course_id":4,"teacher_id":6,"event_date":"` + date + `",` +
		`"start_time":"10:30","end_time":"12:45","room":"1003"}`
}

type fakeCRStore struct {
	profile      store.StudentProfile
	profileErr   error
	entries      []store.RoutineRow
	bookings     []store.Booking
	requests     []store.ClassEventRow
	created      store.ClassEventRow
	err          error
	events       []store.NewClassEvent
	askedBatch   string
	askedSection string
	askedDate    string
	askedRoom    string
	askedUser    int64
	askedStatus  string
}

func (f *fakeCRStore) GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error) {
	if f.profileErr != nil {
		return store.StudentProfile{}, f.profileErr
	}
	return f.profile, nil
}

func (f *fakeCRStore) ListSectionRoutines(ctx context.Context, batch, section string) ([]store.RoutineRow, error) {
	f.askedBatch, f.askedSection = batch, section
	if f.err != nil {
		return nil, f.err
	}
	return f.entries, nil
}

func (f *fakeCRStore) CreateClassRequest(ctx context.Context, event store.NewClassEvent) (store.ClassEventRow, error) {
	f.events = append(f.events, event)
	if f.err != nil {
		return store.ClassEventRow{}, f.err
	}
	created := f.created
	created.Batch = event.Batch
	created.Section = event.Section
	created.EventType = event.EventType
	created.EventDate = event.EventDate
	created.StartTime = event.StartTime
	created.EndTime = event.EndTime
	created.Room = event.Room
	return created, nil
}

func (f *fakeCRStore) ListRoomBookings(ctx context.Context, date, room string) ([]store.Booking, error) {
	f.askedDate, f.askedRoom = date, room
	if f.err != nil {
		return nil, f.err
	}
	return f.bookings, nil
}

func (f *fakeCRStore) ListClassRequests(ctx context.Context, createdBy int64, status string) ([]store.ClassEventRow, error) {
	f.askedUser, f.askedStatus = createdBy, status
	if f.err != nil {
		return nil, f.err
	}
	return f.requests, nil
}

type fakeNotifier struct {
	userID int64
	kind   string
	title  string
	body   string
	calls  int
	err    error
}

func (f *fakeNotifier) NotifyUser(ctx context.Context, userID int64, kind, title, body string) error {
	f.calls++
	f.userID, f.kind, f.title, f.body = userID, kind, title, body
	return f.err
}

func newCRStore() *fakeCRStore {
	return &fakeCRStore{
		profile: store.StudentProfile{UserID: crUserID, Batch: crBatch, Section: crSection, IsCR: true},
		created: store.ClassEventRow{
			ID:            12,
			CourseID:      4,
			CourseCode:    "CSE-233",
			CourseTitle:   "Digital Logic Design",
			TeacherID:     6,
			TeacherUserID: 44,
			TeacherName:   "Active Teacher",
			Status:        store.EventStatusPending,
			CreatedAt:     time.Date(2026, 10, 16, 9, 30, 0, 0, time.UTC),
		},
	}
}

func newCRHandler(t *testing.T, classes CRStore, alerts Notifier) *CR {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load the timezone: %v", err)
	}
	cfg := &config.Config{AppEnv: "test", Timezone: "Asia/Dhaka", Location: loc}
	return NewCR(cfg, classes, alerts)
}

func crCall(t *testing.T, handlerFunc http.HandlerFunc, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	if body != "" {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	user := middleware.User{ID: crUserID, Identifier: "2023331002", Role: "cr"}
	req = req.WithContext(middleware.WithUser(req.Context(), user))
	rec := httptest.NewRecorder()
	handlerFunc.ServeHTTP(rec, req)
	return rec
}

func TestCRRoutineUsesTheOwnProfile(t *testing.T) {
	silenceLog(t)
	classes := newCRStore()
	classes.entries = []store.RoutineRow{
		{ID: 1, Batch: crBatch, Section: "ALL", CourseCode: "CSE-250", CourseTitle: "Software Engineering", TeacherName: "First Teacher", DayOfWeek: 1, StartTime: "08:00", EndTime: "09:00", Room: "1003", Type: "class"},
		{ID: 2, Batch: crBatch, Section: crSection, CourseCode: "CSE-233", CourseTitle: "Digital Logic Design", TeacherName: "Second Teacher", DayOfWeek: 2, StartTime: "10:30", EndTime: "12:45", Room: "1004", Type: "lab"},
	}
	rec := crCall(t, newCRHandler(t, classes, &fakeNotifier{}).Routine(), http.MethodGet, "/api/cr/routine", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if classes.askedBatch != crBatch || classes.askedSection != crSection {
		t.Errorf("the store was asked for %q and %q, want %q and %q", classes.askedBatch, classes.askedSection, crBatch, crSection)
	}
	list := items(t, rec)
	if len(list) != 2 || list[0]["day"] != "Monday" || list[1]["day"] != "Tuesday" {
		t.Errorf("items = %v, want the store order kept", list)
	}
	if list[0]["course_title"] != "Software Engineering" || list[0]["teacher_name"] != "First Teacher" || list[0]["room"] != "1003" {
		t.Errorf("the first entry hides the course, teacher or room: %v", list[0])
	}
	notACR := newCRStore()
	notACR.profile = store.StudentProfile{UserID: crUserID, Batch: crBatch, Section: crSection}
	denied := crCall(t, newCRHandler(t, notACR, &fakeNotifier{}).Routine(), http.MethodGet, "/api/cr/routine", "")
	if denied.Code != http.StatusForbidden {
		t.Errorf("status without the class representative flag = %d, want 403", denied.Code)
	}
	if got := errorMessage(t, denied); got != MessageNotACR {
		t.Errorf("error = %q, want %q", got, MessageNotACR)
	}
	missing := newCRStore()
	missing.profileErr = store.ErrNotFound
	unknown := crCall(t, newCRHandler(t, missing, &fakeNotifier{}).Routine(), http.MethodGet, "/api/cr/routine", "")
	if unknown.Code != http.StatusForbidden {
		t.Errorf("status without a profile = %d, want 403", unknown.Code)
	}
	failing := newCRStore()
	failing.err = errors.New("database is down")
	broken := crCall(t, newCRHandler(t, failing, &fakeNotifier{}).Routine(), http.MethodGet, "/api/cr/routine", "")
	if broken.Code != http.StatusInternalServerError {
		t.Errorf("failing store status = %d, want 500", broken.Code)
	}
}

func TestCRCreateClassRequestValidation(t *testing.T) {
	silenceLog(t)
	tomorrow, today, past := crDates(t)
	validRequest := crRequestBody(tomorrow)
	tests := []struct {
		name       string
		body       string
		storeErr   error
		wantStatus int
		wantError  string
	}{
		{name: "created", body: validRequest, wantStatus: http.StatusCreated},
		{name: "exam", body: strings.Replace(validRequest, `"event_type":"class"`, `"event_type":"exam"`, 1), wantStatus: http.StatusCreated},
		{name: "invalid json", body: `{`, wantStatus: http.StatusBadRequest, wantError: "Invalid request body"},
		{
			name:       "batch in the body",
			body:       strings.Replace(validRequest, `{"event_type":"class"`, `{"batch":"CSE-22","section":"B","event_type":"class"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request body",
		},
		{
			name:       "bad type",
			body:       strings.Replace(validRequest, `"event_type":"class"`, `"event_type":"lab"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Type must be one of class, exam, extra.",
		},
		{
			name:       "missing course",
			body:       strings.Replace(validRequest, `"course_id":4`, `"course_id":0`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  MessageCourseRequired,
		},
		{
			name:       "missing teacher",
			body:       strings.Replace(validRequest, `"teacher_id":6`, `"teacher_id":0`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  MessageTeacherRequired,
		},
		{
			name:       "today",
			body:       crRequestBody(today),
			wantStatus: http.StatusBadRequest,
			wantError:  "Date must be tomorrow or later.",
		},
		{
			name:       "in the past",
			body:       crRequestBody(past),
			wantStatus: http.StatusBadRequest,
			wantError:  "Date must be tomorrow or later.",
		},
		{
			name:       "bad date",
			body:       crRequestBody("17/10/2026"),
			wantStatus: http.StatusBadRequest,
			wantError:  "Date must look like 2026-10-17.",
		},
		{
			name:       "before opening",
			body:       strings.Replace(validRequest, `"start_time":"10:30"`, `"start_time":"07:30"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Classes must be between 08:00 and 17:00.",
		},
		{
			name:       "after closing",
			body:       strings.Replace(validRequest, `"end_time":"12:45"`, `"end_time":"17:30"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Classes must be between 08:00 and 17:00.",
		},
		{
			name:       "start after end",
			body:       strings.Replace(validRequest, `"start_time":"10:30"`, `"start_time":"13:00"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Start time must be earlier than end time.",
		},
		{
			name:       "start equal to end",
			body:       strings.Replace(validRequest, `"end_time":"12:45"`, `"end_time":"10:30"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Start time must be earlier than end time.",
		},
		{
			name:       "missing room",
			body:       strings.Replace(validRequest, `"room":"1003"`, `"room":"  "`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Room is required.",
		},
		{name: "room taken", body: validRequest, storeErr: store.ErrRoomUnavailable, wantStatus: http.StatusConflict, wantError: MessageRoomUnavailable},
		{name: "unknown course", body: validRequest, storeErr: store.ErrCourseNotFound, wantStatus: http.StatusBadRequest, wantError: MessageCourseUnknown},
		{name: "unknown teacher", body: validRequest, storeErr: store.ErrTeacherNotFound, wantStatus: http.StatusBadRequest, wantError: MessageTeacherNotFound},
		{name: "inactive teacher", body: validRequest, storeErr: store.ErrTeacherInactive, wantStatus: http.StatusBadRequest, wantError: MessageTeacherNotActive},
		{
			name:       "store failure",
			body:       validRequest,
			storeErr:   errors.New("database is down"),
			wantStatus: http.StatusInternalServerError,
			wantError:  "Internal server error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			classes := newCRStore()
			classes.err = test.storeErr
			alerts := &fakeNotifier{}
			handlerFunc := newCRHandler(t, classes, alerts).CreateClassRequest()
			rec := crCall(t, handlerFunc, http.MethodPost, "/api/cr/class-requests", test.body)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				if alerts.calls != 0 {
					t.Error("a notification was sent although the request was rejected")
				}
				return
			}
			if len(classes.events) != 1 {
				t.Fatalf("the store received %d requests, want 1", len(classes.events))
			}
			event := classes.events[0]
			if event.Batch != crBatch || event.Section != crSection || event.CreatedBy != crUserID {
				t.Errorf("event = %+v, want the batch, section and user id from the profile", event)
			}
			payload := decodeBody(t, rec)
			if payload["status"] != store.EventStatusPending || payload["batch"] != crBatch || payload["section"] != crSection {
				t.Errorf("payload = %v, want a pending request for the profile batch and section", payload)
			}
			if _, leaked := payload["teacher_user_id"]; leaked {
				t.Error("the response carries the teacher user id")
			}
			if alerts.calls != 1 {
				t.Fatalf("%d notifications were sent, want 1", alerts.calls)
			}
			if alerts.userID != 44 || alerts.kind != NotificationTypeClassRequest {
				t.Errorf("notification went to user %d as %q, want user 44 as %q", alerts.userID, alerts.kind, NotificationTypeClassRequest)
			}
			if !strings.Contains(alerts.body, crBatch) || !strings.Contains(alerts.body, "CSE-233") ||
				!strings.Contains(alerts.body, tomorrow) || !strings.Contains(alerts.body, "1003") {
				t.Errorf("notification body = %q, want the batch, course, date and room", alerts.body)
			}
		})
	}
}

func TestCRCreateClassRequestFixedSlotConflict(t *testing.T) {
	silenceLog(t)
	tomorrow, _, _ := crDates(t)
	validRequest := crRequestBody(tomorrow)
	tests := []struct {
		name    string
		booking store.RoomBooking
		want    string
	}{
		{
			name: "one section",
			booking: store.RoomBooking{
				Room:       "1003",
				DayOfWeek:  1,
				StartTime:  "10:30",
				EndTime:    "12:45",
				Batch:      "CSE-22",
				Section:    "B",
				CourseCode: "CSE-211",
			},
			want: "Room 1003 is reserved for CSE-22 section B (CSE-211) every Monday 10:30–12:45.",
		},
		{
			name: "both sections",
			booking: store.RoomBooking{
				Room:       "2001",
				DayOfWeek:  4,
				StartTime:  "09:00",
				EndTime:    "10:00",
				Batch:      "CSE-21",
				Section:    "ALL",
				CourseCode: "CSE100",
			},
			want: "Room 2001 is reserved for CSE-21 both sections (CSE100) every Thursday 09:00–10:00.",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			classes := newCRStore()
			classes.err = &store.FixedSlotError{Booking: test.booking}
			alerts := &fakeNotifier{}
			rec := crCall(t, newCRHandler(t, classes, alerts).CreateClassRequest(), http.MethodPost, "/api/cr/class-requests", validRequest)
			if rec.Code != http.StatusConflict {
				t.Fatalf("status = %d, want 409 (body = %q)", rec.Code, rec.Body.String())
			}
			if got := errorMessage(t, rec); got != test.want {
				t.Errorf("error = %q, want %q", got, test.want)
			}
			if alerts.calls != 0 {
				t.Error("a notification was sent although the room was reserved")
			}
		})
	}
}

func TestCRCreateClassRequestSurvivesANotificationFailure(t *testing.T) {
	silenceLog(t)
	tomorrow, _, _ := crDates(t)
	validRequest := crRequestBody(tomorrow)
	classes := newCRStore()
	alerts := &fakeNotifier{err: errors.New("notification store is down")}
	rec := crCall(t, newCRHandler(t, classes, alerts).CreateClassRequest(), http.MethodPost, "/api/cr/class-requests", validRequest)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body = %q)", rec.Code, rec.Body.String())
	}
	if len(classes.events) != 1 {
		t.Errorf("the store received %d requests, want 1", len(classes.events))
	}
}

func TestCRRoomAvailability(t *testing.T) {
	silenceLog(t)
	bookings := []store.Booking{
		{Room: "1003", StartTime: "08:00", EndTime: "09:00", Batch: crBatch, Section: "A", CourseCode: "CSE-250", Source: store.BookingSourceFixed, Status: store.BookingSourceFixed},
		{Room: "1003", StartTime: "10:30", EndTime: "12:45", Batch: "CSE-22", Section: "B", CourseCode: "CSE-211", Source: store.BookingSourceFixed, Status: store.BookingSourceFixed},
		{Room: "1004", StartTime: "09:00", EndTime: "10:00", Batch: crBatch, Section: "A", CourseCode: "CSE-233", Source: "exam", Status: store.EventStatusApproved},
		{Room: "1004", StartTime: "13:00", EndTime: "14:00", Batch: "CSE-22", Section: "B", CourseCode: "CSE-211", Source: "extra", Status: store.EventStatusPending},
	}
	wantLabels := []string{LabelYours, LabelUnavailable, LabelUnavailable, LabelUnavailable}
	classes := newCRStore()
	classes.bookings = bookings
	rec := crCall(t, newCRHandler(t, classes, &fakeNotifier{}).RoomAvailability(), http.MethodGet,
		"/api/cr/room-availability?date=2026-10-17&room=", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if classes.askedDate != "2026-10-17" || classes.askedRoom != "" {
		t.Errorf("the store was asked for %q and %q, want the date and no room", classes.askedDate, classes.askedRoom)
	}
	list := items(t, rec)
	if len(list) != len(bookings) {
		t.Fatalf("got %d items, want %d", len(list), len(bookings))
	}
	for i, want := range wantLabels {
		if list[i]["label"] != want {
			t.Errorf("item %d label = %v, want %q", i, list[i]["label"], want)
		}
		if list[i]["room"] != bookings[i].Room || list[i]["start_time"] != bookings[i].StartTime {
			t.Errorf("item %d = %v, want the store order kept", i, list[i])
		}
	}
	if list[0]["source"] != store.BookingSourceFixed || list[2]["source"] != "exam" || list[3]["status"] != store.EventStatusPending {
		t.Errorf("the sources or statuses are wrong: %v", list)
	}
	filtered := newCRStore()
	filtered.bookings = bookings[:2]
	roomRec := crCall(t, newCRHandler(t, filtered, &fakeNotifier{}).RoomAvailability(), http.MethodGet,
		"/api/cr/room-availability?date=2026-10-17&room=1003", "")
	if roomRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", roomRec.Code, roomRec.Body.String())
	}
	if filtered.askedRoom != "1003" {
		t.Errorf("the store was asked for room %q, want 1003", filtered.askedRoom)
	}
	tests := []struct {
		name      string
		query     string
		wantError string
	}{
		{name: "missing date", query: "", wantError: MessageDateRequired},
		{name: "blank date", query: "?date=%20", wantError: MessageDateRequired},
		{name: "bad date", query: "?date=2026-13-40", wantError: "date must look like 2026-10-17."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rejected := crCall(t, newCRHandler(t, newCRStore(), &fakeNotifier{}).RoomAvailability(), http.MethodGet,
				"/api/cr/room-availability"+test.query, "")
			if rejected.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body = %q)", rejected.Code, rejected.Body.String())
			}
			if got := errorMessage(t, rejected); got != test.wantError {
				t.Errorf("error = %q, want %q", got, test.wantError)
			}
		})
	}
}

func TestCRRequests(t *testing.T) {
	silenceLog(t)
	decided := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	classes := newCRStore()
	classes.requests = []store.ClassEventRow{
		{
			ID:          12,
			Batch:       crBatch,
			Section:     crSection,
			EventType:   "extra",
			CourseCode:  "CSE-233",
			CourseTitle: "Digital Logic Design",
			TeacherID:   6,
			TeacherName: "Active Teacher",
			EventDate:   "2026-10-18",
			StartTime:   "10:30",
			EndTime:     "12:45",
			Room:        "1004",
			Status:      store.EventStatusPending,
			CreatedAt:   time.Date(2026, 10, 16, 11, 0, 0, 0, time.UTC),
		},
		{
			ID:          11,
			Batch:       crBatch,
			Section:     crSection,
			EventType:   "class",
			CourseCode:  "CSE-250",
			CourseTitle: "Software Engineering",
			TeacherID:   7,
			TeacherName: "Other Teacher",
			EventDate:   "2026-10-17",
			StartTime:   "08:00",
			EndTime:     "09:00",
			Room:        "1003",
			Status:      store.EventStatusApproved,
			CreatedAt:   time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC),
			DecidedAt:   &decided,
		},
	}
	rec := crCall(t, newCRHandler(t, classes, &fakeNotifier{}).Requests(), http.MethodGet, "/api/cr/requests", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if classes.askedUser != crUserID || classes.askedStatus != "" {
		t.Errorf("the store was asked for user %d and status %q, want %d and no status", classes.askedUser, classes.askedStatus, crUserID)
	}
	list := items(t, rec)
	if len(list) != 2 || list[0]["id"] != float64(12) || list[1]["id"] != float64(11) {
		t.Fatalf("items = %v, want the newest request first", list)
	}
	if list[0]["decided_at"] != nil {
		t.Errorf("the pending request carries decided_at = %v, want none", list[0]["decided_at"])
	}
	if list[1]["decided_at"] != decided.Format(time.RFC3339) {
		t.Errorf("decided_at = %v, want %q", list[1]["decided_at"], decided.Format(time.RFC3339))
	}
	if list[0]["course_title"] != "Digital Logic Design" || list[0]["teacher_name"] != "Active Teacher" {
		t.Errorf("the first request hides the course or teacher: %v", list[0])
	}
	statuses := []struct {
		name       string
		query      string
		wantStatus int
		wantAsked  string
		wantError  string
	}{
		{name: "pending", query: "?status=pending", wantStatus: http.StatusOK, wantAsked: store.EventStatusPending},
		{name: "auto cancelled", query: "?status=auto_cancelled", wantStatus: http.StatusOK, wantAsked: store.EventStatusAutoCancelled},
		{name: "unknown", query: "?status=waiting", wantStatus: http.StatusBadRequest, wantError: MessageInvalidEventStatus},
	}
	for _, test := range statuses {
		t.Run(test.name, func(t *testing.T) {
			filtered := newCRStore()
			filteredRec := crCall(t, newCRHandler(t, filtered, &fakeNotifier{}).Requests(), http.MethodGet, "/api/cr/requests"+test.query, "")
			if filteredRec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", filteredRec.Code, test.wantStatus, filteredRec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, filteredRec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			if filtered.askedStatus != test.wantAsked {
				t.Errorf("the store was asked for status %q, want %q", filtered.askedStatus, test.wantAsked)
			}
		})
	}
	failing := newCRStore()
	failing.err = errors.New("database is down")
	broken := crCall(t, newCRHandler(t, failing, &fakeNotifier{}).Requests(), http.MethodGet, "/api/cr/requests", "")
	if broken.Code != http.StatusInternalServerError {
		t.Errorf("failing store status = %d, want 500", broken.Code)
	}
}
