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
	teacherUserID = int64(51)
	teacherRowID  = int64(7)
)

type broadcastCall struct {
	role    string
	batch   string
	section string
	kind    string
	title   string
	body    string
}

type directCall struct {
	userID int64
	kind   string
	title  string
	body   string
}

type fakeAlerts struct {
	direct     []directCall
	broadcasts []broadcastCall
	err        error
}

func (f *fakeAlerts) NotifyUser(ctx context.Context, userID int64, kind, title, body string) error {
	f.direct = append(f.direct, directCall{userID: userID, kind: kind, title: title, body: body})
	return f.err
}

func (f *fakeAlerts) NotifyBroadcast(ctx context.Context, role, batch, section, kind, title, body string) error {
	f.broadcasts = append(f.broadcasts, broadcastCall{role: role, batch: batch, section: section, kind: kind, title: title, body: body})
	return f.err
}

type fakeTeacherStore struct {
	teacher      store.Teacher
	teacherErr   error
	requests     []store.RequestRow
	decision     store.EventDecision
	schedule     []store.ScheduleItem
	classes      []store.ClassEventRow
	routine      store.RoutineRow
	routineErr   error
	crs          []int64
	err          error
	cancelErr    error
	askedDate    string
	askedStatus  string
	askedEvent   int64
	askedTeacher int64
	cancelled    []string
	crBatch      string
	crSection    string
}

func (f *fakeTeacherStore) GetTeacherByUser(ctx context.Context, userID int64) (store.Teacher, error) {
	if f.teacherErr != nil {
		return store.Teacher{}, f.teacherErr
	}
	return f.teacher, nil
}

func (f *fakeTeacherStore) ListTeacherRequests(ctx context.Context, teacherID int64) ([]store.RequestRow, error) {
	f.askedTeacher = teacherID
	if f.err != nil {
		return nil, f.err
	}
	return f.requests, nil
}

func (f *fakeTeacherStore) DecideRequest(ctx context.Context, eventID, teacherID int64, status string, now time.Time) (store.EventDecision, error) {
	f.askedEvent, f.askedTeacher, f.askedStatus = eventID, teacherID, status
	if f.err != nil {
		return store.EventDecision{}, f.err
	}
	decision := f.decision
	decision.ID = eventID
	decision.Status = status
	return decision, nil
}

func (f *fakeTeacherStore) CancelClassEvent(ctx context.Context, eventID, teacherID int64, now time.Time) (store.EventDecision, error) {
	f.askedEvent, f.askedTeacher = eventID, teacherID
	if f.err != nil {
		return store.EventDecision{}, f.err
	}
	decision := f.decision
	decision.ID = eventID
	decision.Status = store.EventStatusCancelled
	return decision, nil
}

func (f *fakeTeacherStore) ListTeacherSchedule(ctx context.Context, teacherID int64, date string) ([]store.ScheduleItem, error) {
	f.askedTeacher, f.askedDate = teacherID, date
	if f.err != nil {
		return nil, f.err
	}
	return f.schedule, nil
}

func (f *fakeTeacherStore) ListTeacherClasses(ctx context.Context, teacherID int64, date string, now time.Time) ([]store.ClassEventRow, error) {
	f.askedTeacher, f.askedDate = teacherID, date
	if f.err != nil {
		return nil, f.err
	}
	return f.classes, nil
}

func (f *fakeTeacherStore) GetTeacherRoutine(ctx context.Context, routineID, teacherID int64) (store.RoutineRow, error) {
	f.askedEvent, f.askedTeacher = routineID, teacherID
	if f.routineErr != nil {
		return store.RoutineRow{}, f.routineErr
	}
	return f.routine, nil
}

func (f *fakeTeacherStore) CancelRoutineOccurrence(ctx context.Context, routineID, cancelledBy int64, date string) error {
	if f.cancelErr != nil {
		return f.cancelErr
	}
	f.cancelled = append(f.cancelled, date)
	return nil
}

func (f *fakeTeacherStore) ListActiveCRs(ctx context.Context, batch, section string) ([]int64, error) {
	f.crBatch, f.crSection = batch, section
	return f.crs, nil
}

func newTeacherStore() *fakeTeacherStore {
	return &fakeTeacherStore{
		teacher: store.Teacher{ID: teacherRowID, UserID: teacherUserID, Name: "Active Teacher"},
		decision: store.EventDecision{
			EventType:   "class",
			Batch:       "CSE-23",
			Section:     "A",
			CourseCode:  "CSE-233",
			CourseTitle: "Digital Logic Design",
			EventDate:   "2026-10-18",
			StartTime:   "10:30",
			EndTime:     "12:45",
			Room:        "1003",
			CreatedBy:   31,
		},
	}
}

func newTeacherHandler(t *testing.T, teachers TeacherStore, alerts BroadcastNotifier) *Teacher {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load the timezone: %v", err)
	}
	cfg := &config.Config{AppEnv: "test", Timezone: "Asia/Dhaka", Location: loc}
	return NewTeacher(cfg, teachers, alerts)
}

func roleCall(t *testing.T, handlerFunc http.HandlerFunc, user middleware.User, method, target, id, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	if body != "" {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if id != "" {
		req.SetPathValue("id", id)
	}
	req = req.WithContext(middleware.WithUser(req.Context(), user))
	rec := httptest.NewRecorder()
	handlerFunc.ServeHTTP(rec, req)
	return rec
}

func teacherCall(t *testing.T, handlerFunc http.HandlerFunc, method, target, id, body string) *httptest.ResponseRecorder {
	t.Helper()
	user := middleware.User{ID: teacherUserID, Identifier: "T-001", Role: "teacher"}
	return roleCall(t, handlerFunc, user, method, target, id, body)
}

func teacherDates(t *testing.T) (string, string, int) {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load the timezone: %v", err)
	}
	now := time.Now().In(loc)
	tomorrow := now.AddDate(0, 0, 1)
	return tomorrow.Format(validate.DateLayout), now.AddDate(0, 0, -1).Format(validate.DateLayout), int(tomorrow.Weekday())
}

func TestTeacherRequests(t *testing.T) {
	silenceLog(t)
	teachers := newTeacherStore()
	teachers.requests = []store.RequestRow{
		{
			ID:               12,
			EventType:        "extra",
			Batch:            "CSE-23",
			Section:          "A",
			CourseCode:       "CSE-233",
			CourseTitle:      "Digital Logic Design",
			EventDate:        "2026-10-18",
			StartTime:        "10:30",
			EndTime:          "12:45",
			Room:             "1004",
			RequestedBy:      "First CR",
			RequestedByRegNo: "2023331800",
			CreatedBy:        31,
			CreatedAt:        time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC),
		},
	}
	rec := teacherCall(t, newTeacherHandler(t, teachers, &fakeAlerts{}).Requests(), http.MethodGet, "/api/teacher/requests", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if teachers.askedTeacher != teacherRowID {
		t.Errorf("the store was asked for teacher %d, want %d", teachers.askedTeacher, teacherRowID)
	}
	item := items(t, rec)[0]
	if item["requested_by"] != "First CR" || item["requested_by_reg_no"] != "2023331800" {
		t.Errorf("item = %v, want the class representative name and registration number", item)
	}
	if item["event_type"] != "extra" || item["room"] != "1004" {
		t.Errorf("item = %v, want the type and room", item)
	}
	missing := newTeacherStore()
	missing.teacherErr = store.ErrNotFound
	denied := teacherCall(t, newTeacherHandler(t, missing, &fakeAlerts{}).Requests(), http.MethodGet, "/api/teacher/requests", "", "")
	if denied.Code != http.StatusForbidden {
		t.Errorf("status without a teacher profile = %d, want 403", denied.Code)
	}
	if got := errorMessage(t, denied); got != MessageNotATeacher {
		t.Errorf("error = %q, want %q", got, MessageNotATeacher)
	}
	failing := newTeacherStore()
	failing.err = errors.New("database is down")
	broken := teacherCall(t, newTeacherHandler(t, failing, &fakeAlerts{}).Requests(), http.MethodGet, "/api/teacher/requests", "", "")
	if broken.Code != http.StatusInternalServerError {
		t.Errorf("failing store status = %d, want 500", broken.Code)
	}
}

func TestTeacherDecisions(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name          string
		reject        bool
		id            string
		storeErr      error
		wantStatus    int
		wantMessage   string
		wantError     string
		wantBroadcast bool
	}{
		{name: "approved", id: "12", wantStatus: http.StatusOK, wantMessage: MessageRequestApproved, wantBroadcast: true},
		{name: "rejected", reject: true, id: "12", wantStatus: http.StatusOK, wantMessage: MessageRequestRejected},
		{name: "bad id", id: "twelve", wantStatus: http.StatusBadRequest, wantError: MessageInvalidID},
		{name: "another teacher", id: "12", storeErr: store.ErrNotFound, wantStatus: http.StatusNotFound, wantError: MessageRequestNotFound},
		{name: "already decided", id: "12", storeErr: store.ErrNotPending, wantStatus: http.StatusConflict, wantError: MessageRequestDecided},
		{name: "too late", id: "12", storeErr: store.ErrTooLateToApprove, wantStatus: http.StatusConflict, wantError: MessageTooLateToApprove},
		{
			name:       "store failure",
			id:         "12",
			storeErr:   errors.New("database is down"),
			wantStatus: http.StatusInternalServerError,
			wantError:  "Internal server error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			teachers := newTeacherStore()
			teachers.err = test.storeErr
			alerts := &fakeAlerts{}
			handler := newTeacherHandler(t, teachers, alerts)
			handlerFunc := handler.Approve()
			path := "/api/teacher/requests/" + test.id + "/approve"
			if test.reject {
				handlerFunc = handler.Reject()
				path = "/api/teacher/requests/" + test.id + "/reject"
			}
			rec := teacherCall(t, handlerFunc, http.MethodPost, path, test.id, "")
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				if len(alerts.direct) != 0 || len(alerts.broadcasts) != 0 {
					t.Error("a notification was sent although the decision failed")
				}
				return
			}
			if got := successMessage(t, rec); got != test.wantMessage {
				t.Errorf("message = %q, want %q", got, test.wantMessage)
			}
			wantStored := store.EventStatusApproved
			if test.reject {
				wantStored = store.EventStatusRejected
			}
			if teachers.askedStatus != wantStored || teachers.askedEvent != 12 {
				t.Errorf("the store was asked to set %q on event %d, want %q on 12", teachers.askedStatus, teachers.askedEvent, wantStored)
			}
			if len(alerts.direct) != 1 || alerts.direct[0].userID != 31 {
				t.Fatalf("direct notifications = %v, want one for the class representative", alerts.direct)
			}
			if !strings.Contains(alerts.direct[0].body, "CSE-233") || !strings.Contains(alerts.direct[0].body, "1003") {
				t.Errorf("notification body = %q, want the course and room", alerts.direct[0].body)
			}
			if !test.wantBroadcast {
				if len(alerts.broadcasts) != 0 {
					t.Errorf("broadcasts = %v, want none for a rejection", alerts.broadcasts)
				}
				return
			}
			if len(alerts.broadcasts) != 1 {
				t.Fatalf("broadcasts = %v, want one for the students", alerts.broadcasts)
			}
			broadcast := alerts.broadcasts[0]
			if broadcast.role != "student" || broadcast.batch != "CSE-23" || broadcast.section != "A" {
				t.Errorf("broadcast = %+v, want the students of CSE-23 section A", broadcast)
			}
			if !strings.Contains(broadcast.title, "class") {
				t.Errorf("broadcast title = %q, want it to name the type", broadcast.title)
			}
		})
	}
}

func TestTeacherApprovalForBothSectionsReachesTheWholeBatch(t *testing.T) {
	silenceLog(t)
	teachers := newTeacherStore()
	teachers.decision.Section = validate.SectionAll
	alerts := &fakeAlerts{}
	rec := teacherCall(t, newTeacherHandler(t, teachers, alerts).Approve(), http.MethodPost, "/api/teacher/requests/12/approve", "12", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if len(alerts.broadcasts) != 1 || alerts.broadcasts[0].section != "" {
		t.Errorf("broadcasts = %v, want one without a section", alerts.broadcasts)
	}
}

func TestTeacherSchedule(t *testing.T) {
	silenceLog(t)
	tomorrow, _, _ := teacherDates(t)
	teachers := newTeacherStore()
	teachers.schedule = []store.ScheduleItem{
		{Kind: "fixed", ID: 3, Batch: "CSE-23", Section: "A", CourseCode: "CSE-250", CourseTitle: "Software Engineering", StartTime: "08:00", EndTime: "09:00", Room: "1003", Cancelled: true},
		{Kind: "exam", ID: 9, Batch: "CSE-23", Section: "ALL", CourseCode: "CSE-233", CourseTitle: "Digital Logic Design", StartTime: "10:30", EndTime: "12:45", Room: "1004"},
	}
	rec := teacherCall(t, newTeacherHandler(t, teachers, &fakeAlerts{}).Schedule(), http.MethodGet,
		"/api/teacher/schedule?date="+tomorrow, "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if teachers.askedDate != tomorrow {
		t.Errorf("the store was asked for %q, want %q", teachers.askedDate, tomorrow)
	}
	list := items(t, rec)
	if len(list) != 2 || list[0]["kind"] != "fixed" || list[0]["cancelled"] != true {
		t.Errorf("items = %v, want the cancelled fixed entry first", list)
	}
	if list[1]["kind"] != "exam" || list[1]["cancelled"] != false {
		t.Errorf("second item = %v, want the exam marked as not cancelled", list[1])
	}
	if got := decodeBody(t, rec)["weekday"]; got == "" {
		t.Error("the response has no weekday")
	}
	tests := []struct {
		name      string
		query     string
		wantError string
	}{
		{name: "missing date", query: "", wantError: MessageDateRequired},
		{name: "bad date", query: "?date=tomorrow", wantError: "date must look like 2026-10-17."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rejected := teacherCall(t, newTeacherHandler(t, newTeacherStore(), &fakeAlerts{}).Schedule(),
				http.MethodGet, "/api/teacher/schedule"+test.query, "", "")
			if rejected.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body = %q)", rejected.Code, rejected.Body.String())
			}
			if got := errorMessage(t, rejected); got != test.wantError {
				t.Errorf("error = %q, want %q", got, test.wantError)
			}
		})
	}
}

func TestTeacherUpcomingClasses(t *testing.T) {
	silenceLog(t)
	tomorrow, _, _ := teacherDates(t)
	teachers := newTeacherStore()
	teachers.classes = []store.ClassEventRow{
		{ID: 9, EventType: "class", Batch: "CSE-23", Section: "A", CourseCode: "CSE-233", CourseTitle: "Digital Logic Design", EventDate: tomorrow, StartTime: "10:30", EndTime: "12:45", Room: "1004", Status: store.EventStatusApproved},
	}
	rec := teacherCall(t, newTeacherHandler(t, teachers, &fakeAlerts{}).UpcomingClasses(), http.MethodGet,
		"/api/teacher/classes/upcoming", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if teachers.askedDate != "" {
		t.Errorf("the store was asked for date %q, want every upcoming date", teachers.askedDate)
	}
	if item := items(t, rec)[0]; item["event_date"] != tomorrow || item["status"] != store.EventStatusApproved {
		t.Errorf("item = %v, want the approved class of tomorrow", item)
	}
	filtered := newTeacherStore()
	one := teacherCall(t, newTeacherHandler(t, filtered, &fakeAlerts{}).UpcomingClasses(), http.MethodGet,
		"/api/teacher/classes/upcoming?date="+tomorrow, "", "")
	if one.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", one.Code, one.Body.String())
	}
	if filtered.askedDate != tomorrow {
		t.Errorf("the store was asked for %q, want %q", filtered.askedDate, tomorrow)
	}
	bad := teacherCall(t, newTeacherHandler(t, newTeacherStore(), &fakeAlerts{}).UpcomingClasses(), http.MethodGet,
		"/api/teacher/classes/upcoming?date=soon", "", "")
	if bad.Code != http.StatusBadRequest {
		t.Errorf("bad date status = %d, want 400", bad.Code)
	}
}

func TestTeacherCancelClass(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		id         string
		storeErr   error
		wantStatus int
		wantError  string
	}{
		{name: "cancelled", id: "9", wantStatus: http.StatusOK},
		{name: "bad id", id: "nine", wantStatus: http.StatusBadRequest, wantError: MessageInvalidID},
		{name: "another teacher", id: "9", storeErr: store.ErrNotFound, wantStatus: http.StatusNotFound, wantError: MessageClassNotFound},
		{name: "not approved", id: "9", storeErr: store.ErrNotCancellable, wantStatus: http.StatusConflict, wantError: MessageClassNotCancellable},
		{
			name:       "store failure",
			id:         "9",
			storeErr:   errors.New("database is down"),
			wantStatus: http.StatusInternalServerError,
			wantError:  "Internal server error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			teachers := newTeacherStore()
			teachers.err = test.storeErr
			alerts := &fakeAlerts{}
			rec := teacherCall(t, newTeacherHandler(t, teachers, alerts).CancelClass(), http.MethodPost,
				"/api/teacher/classes/"+test.id+"/cancel", test.id, "")
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			if got := successMessage(t, rec); got != MessageClassCancelled {
				t.Errorf("message = %q, want %q", got, MessageClassCancelled)
			}
			if len(alerts.direct) != 1 || alerts.direct[0].userID != 31 {
				t.Errorf("direct notifications = %v, want one for the class representative", alerts.direct)
			}
			if len(alerts.broadcasts) != 1 || alerts.broadcasts[0].batch != "CSE-23" {
				t.Errorf("broadcasts = %v, want one for the batch", alerts.broadcasts)
			}
		})
	}
}

func TestTeacherCancelRoutineOccurrence(t *testing.T) {
	silenceLog(t)
	tomorrow, yesterday, weekday := teacherDates(t)
	tests := []struct {
		name       string
		id         string
		date       string
		day        int
		routineErr error
		cancelErr  error
		wantStatus int
		wantError  string
	}{
		{name: "cancelled", id: "3", date: tomorrow, day: weekday, wantStatus: http.StatusOK},
		{name: "bad id", id: "three", date: tomorrow, day: weekday, wantStatus: http.StatusBadRequest, wantError: MessageInvalidID},
		{name: "missing date", id: "3", date: "", day: weekday, wantStatus: http.StatusBadRequest, wantError: "date is required."},
		{name: "bad date", id: "3", date: "next week", day: weekday, wantStatus: http.StatusBadRequest, wantError: "date must look like 2026-10-17."},
		{name: "past date", id: "3", date: yesterday, day: weekday, wantStatus: http.StatusBadRequest, wantError: MessagePastDate},
		{
			name:       "wrong weekday",
			id:         "3",
			date:       tomorrow,
			day:        (weekday + 1) % 7,
			wantStatus: http.StatusBadRequest,
			wantError:  weekdayMismatchMessage((weekday + 1) % 7),
		},
		{
			name:       "another teacher",
			id:         "3",
			date:       tomorrow,
			day:        weekday,
			routineErr: store.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantError:  MessageRoutineNotFound,
		},
		{
			name:       "already cancelled",
			id:         "3",
			date:       tomorrow,
			day:        weekday,
			cancelErr:  store.ErrAlreadyCancelled,
			wantStatus: http.StatusConflict,
			wantError:  MessageOccurrenceCancelled,
		},
		{
			name:       "store failure",
			id:         "3",
			date:       tomorrow,
			day:        weekday,
			cancelErr:  errors.New("database is down"),
			wantStatus: http.StatusInternalServerError,
			wantError:  "Internal server error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			teachers := newTeacherStore()
			teachers.routine = store.RoutineRow{
				ID:          3,
				Batch:       "CSE-23",
				Section:     "A",
				CourseCode:  "CSE-250",
				CourseTitle: "Software Engineering",
				TeacherID:   teacherRowID,
				DayOfWeek:   test.day,
				StartTime:   "08:00",
				EndTime:     "09:00",
				Room:        "1003",
				Type:        "class",
			}
			teachers.routineErr = test.routineErr
			teachers.cancelErr = test.cancelErr
			teachers.crs = []int64{31, 32}
			alerts := &fakeAlerts{}
			body := `{"date":"` + test.date + `"}`
			rec := teacherCall(t, newTeacherHandler(t, teachers, alerts).CancelRoutine(), http.MethodPost,
				"/api/teacher/routines/"+test.id+"/cancel", test.id, body)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				if len(teachers.cancelled) != 0 {
					t.Error("an occurrence was cancelled although the call was rejected")
				}
				return
			}
			if len(teachers.cancelled) != 1 || teachers.cancelled[0] != tomorrow {
				t.Errorf("cancelled = %v, want [%s]", teachers.cancelled, tomorrow)
			}
			if len(alerts.broadcasts) != 1 || alerts.broadcasts[0].batch != "CSE-23" || alerts.broadcasts[0].section != "A" {
				t.Errorf("broadcasts = %v, want one for CSE-23 section A", alerts.broadcasts)
			}
			if len(alerts.direct) != 2 {
				t.Fatalf("direct notifications = %v, want one per class representative", alerts.direct)
			}
			for i, call := range alerts.direct {
				if call.userID != teachers.crs[i] {
					t.Errorf("notification %d went to %d, want %d", i, call.userID, teachers.crs[i])
				}
				if !strings.Contains(call.body, tomorrow) {
					t.Errorf("notification body = %q, want the cancelled date", call.body)
				}
			}
			if teachers.crBatch != "CSE-23" || teachers.crSection != "A" {
				t.Errorf("the class representatives of %q and %q were asked for, want CSE-23 and A", teachers.crBatch, teachers.crSection)
			}
		})
	}
}

func TestTeacherCancelRoutineKeepsGoingWithoutNotifications(t *testing.T) {
	silenceLog(t)
	tomorrow, _, weekday := teacherDates(t)
	teachers := newTeacherStore()
	teachers.routine = store.RoutineRow{ID: 3, Batch: "CSE-23", Section: "A", DayOfWeek: weekday, StartTime: "08:00", EndTime: "09:00", Room: "1003", CourseCode: "CSE-250"}
	teachers.crs = []int64{31}
	alerts := &fakeAlerts{err: errors.New("notification store is down")}
	rec := teacherCall(t, newTeacherHandler(t, teachers, alerts).CancelRoutine(), http.MethodPost,
		"/api/teacher/routines/3/cancel", "3", `{"date":"`+tomorrow+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if len(teachers.cancelled) != 1 {
		t.Errorf("cancelled = %v, want the occurrence stored", teachers.cancelled)
	}
	if len(alerts.direct) != 1 {
		t.Errorf("direct notifications = %v, want one attempt", alerts.direct)
	}
}
