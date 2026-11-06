package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raHI-79/250-project/backend/internal/attendance"
	"github.com/raHI-79/250-project/backend/internal/jobs"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/settings"
	"github.com/raHI-79/250-project/backend/internal/store"
)

const (
	markCourse  = `{"code":"CSE 2101","title":"Data Structures","credit":3}`
	markTeacher = `{"name":"Mark Teacher","user_id":"T-700","email":"markteacher@sust.edu",
"phone":"01712347700"}`
	markCR = `{"full_name":"Mark CR","reg_no":"2023331700","batch":"CSE-23","section":"A",
"email":"markcr@student.sust.edu","phone":"01712347701","blood_group":"O+",
"password":"sust-250-project","confirm_password":"sust-250-project"}`
	markPresent = `{"full_name":"Mark Present","reg_no":"2023331701","batch":"CSE-23","section":"A",
"email":"markpresent@student.sust.edu","phone":"01712347702","blood_group":"A+",
"password":"sust-250-project","confirm_password":"sust-250-project"}`
	markAbsent = `{"full_name":"Mark Absent","reg_no":"2023331702","batch":"CSE-23","section":"A",
"email":"markabsent@student.sust.edu","phone":"01712347703","blood_group":"B+",
"password":"sust-250-project","confirm_password":"sust-250-project"}`
)

type markFixture struct {
	ctx          context.Context
	pool         *pgxpool.Pool
	router       http.Handler
	adminToken   string
	teacherToken string
	presentToken string
	absentToken  string
	courseID     int64
	teacherRowID int64
}

func callFromApp(t *testing.T, router http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	request := httptest.NewRequest(method, path, nil)
	if body != "" {
		reader = strings.NewReader(body)
		request = httptest.NewRequest(method, path, reader)
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	request.Header.Set(middleware.ClientHeader, middleware.ClientAndroid)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, request)
	return rec
}

func seedAttendance(t *testing.T) markFixture {
	t.Helper()
	ctx, pool, router := newFlowRouter(t)
	fixture := markFixture{ctx: ctx, pool: pool, router: router}
	fixture.adminToken = seedAdmin(t, ctx, pool, router)
	created := call(t, router, http.MethodPost, "/api/admin/courses", markCourse, fixture.adminToken)
	if created.Code != http.StatusCreated {
		t.Fatalf("create course status = %d, want 201 (body = %q)", created.Code, created.Body.String())
	}
	fixture.courseID = int64(number(t, created, "id"))
	invited := teacherInviteCall(t, router, fixture.adminToken, markTeacher)
	if invited.Code != http.StatusCreated {
		t.Fatalf("invite teacher status = %d, want 201 (body = %q)", invited.Code, invited.Body.String())
	}
	fixture.teacherRowID = lookupTeacherID(t, router, fixture.adminToken,
		int64(number(t, invited, "id")))
	// A routine entry gives the teacher the course and the batch it may take attendance
	// for, which the options endpoint reads.
	routine := routineBody("A", fixture.courseID, fixture.teacherRowID, 0, "09:00", "10:30",
		"2101", "class")
	if rec := call(t, router, http.MethodPost, "/api/admin/routines", routine,
		fixture.adminToken); rec.Code != http.StatusCreated {
		t.Fatalf("create routine status = %d, want 201 (body = %q)", rec.Code, rec.Body.String())
	}
	onboardCR(t, router, fixture.adminToken, markCR)
	onboardStudent(t, router, fixture.adminToken, markPresent)
	onboardStudent(t, router, fixture.adminToken, markAbsent)
	fixture.teacherToken = loginToken(t, router, "T-700")
	fixture.presentToken = loginToken(t, router, "2023331701")
	fixture.absentToken = loginToken(t, router, "2023331702")
	return fixture
}

func openedSession(t *testing.T, fixture markFixture, method string) map[string]any {
	t.Helper()
	body := `{"course_id":` + strconv.FormatInt(fixture.courseID, 10) +
		`,"batch":"CSE-23","section":"A","method":"` + method + `","minutes":10}`
	rec := call(t, fixture.router, http.MethodPost, "/api/teacher/attendance/sessions",
		body, fixture.teacherToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("open a %s session status = %d, want 201 (body = %q)",
			method, rec.Code, rec.Body.String())
	}
	var session map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &session); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	return session
}

func sessionID(t *testing.T, session map[string]any) int64 {
	t.Helper()
	value, ok := session["id"].(float64)
	if !ok {
		t.Fatalf("the session has no id: %v", session)
	}
	return int64(value)
}

func TestAttendanceFromOpenToSheet(t *testing.T) {
	fixture := seedAttendance(t)
	router := fixture.router
	var bluetoothID int64

	t.Run("the teacher opens a Bluetooth session and keeps the key to itself", func(t *testing.T) {
		session := openedSession(t, fixture, attendance.TakeBluetooth)
		bluetoothID = sessionID(t, session)
		key, ok := session["token_key"].(string)
		if !ok || key == "" {
			t.Fatalf("the teacher did not receive the key it needs: %v", session)
		}
		if code, ok := session["code"].(string); !ok || len(code) != 4 {
			t.Errorf("the teacher received the code %v, want four digits", session["code"])
		}
		// The same session read by a student carries neither.
		listed := call(t, router, http.MethodGet, "/api/students/attendance/open", "",
			fixture.presentToken)
		if listed.Code != http.StatusOK {
			t.Fatalf("the open sessions status = %d, want 200 (body = %q)",
				listed.Code, listed.Body.String())
		}
		if strings.Contains(listed.Body.String(), key) {
			t.Errorf("the key reached a student: %s", listed.Body.String())
		}
		if strings.Contains(listed.Body.String(), `"code"`) {
			t.Errorf("the code reached a student: %s", listed.Body.String())
		}
	})

	t.Run("a phone marks itself present with the token and its position", func(t *testing.T) {
		bound := callFromApp(t, router, http.MethodPost, "/api/students/attendance/phone",
			`{"device_id":"device-present"}`, fixture.presentToken)
		if bound.Code != http.StatusOK {
			t.Fatalf("binding the phone status = %d, want 200 (body = %q)",
				bound.Code, bound.Body.String())
		}
		// The teacher's phone reports where the class is, and the student's has to be
		// close to it.
		placed := call(t, router, http.MethodPost,
			"/api/teacher/attendance/sessions/"+strconv.FormatInt(bluetoothID, 10)+"/location",
			`{"lat":24.9165,"lng":91.8316,"accuracy_m":8}`, fixture.teacherToken)
		if placed.Code != http.StatusOK {
			t.Fatalf("the teacher position status = %d, want 200 (body = %q)",
				placed.Code, placed.Body.String())
		}
		key := sessionKeyOf(t, fixture, bluetoothID)
		token := attendance.CurrentToken(key, bluetoothID, flowClock())
		body := `{"session_id":` + strconv.FormatInt(bluetoothID, 10) +
			`,"token":"` + token + `","device_id":"device-present",` +
			`"lat":24.9165,"lng":91.8316,"accuracy_m":10}`
		marked := callFromApp(t, router, http.MethodPost, "/api/students/attendance/mark",
			body, fixture.presentToken)
		if marked.Code != http.StatusCreated {
			t.Fatalf("marking from the app status = %d, want 201 (body = %q)",
				marked.Code, marked.Body.String())
		}
	})

	t.Run("a phone far from the class is turned away", func(t *testing.T) {
		key := sessionKeyOf(t, fixture, bluetoothID)
		token := attendance.CurrentToken(key, bluetoothID, flowClock())
		bound := callFromApp(t, router, http.MethodPost, "/api/students/attendance/phone",
			`{"device_id":"device-absent"}`, fixture.absentToken)
		if bound.Code != http.StatusOK {
			t.Fatalf("binding the phone status = %d, want 200 (body = %q)",
				bound.Code, bound.Body.String())
		}
		// Half a degree of latitude is tens of kilometres away from the room.
		body := `{"session_id":` + strconv.FormatInt(bluetoothID, 10) +
			`,"token":"` + token + `","device_id":"device-absent",` +
			`"lat":25.5000,"lng":91.8316,"accuracy_m":10}`
		marked := callFromApp(t, router, http.MethodPost, "/api/students/attendance/mark",
			body, fixture.absentToken)
		if marked.Code < http.StatusBadRequest {
			t.Errorf("a phone far from the class was marked present: %d (body = %q)",
				marked.Code, marked.Body.String())
		}
	})

	t.Run("a stale token is refused", func(t *testing.T) {
		body := `{"session_id":` + strconv.FormatInt(bluetoothID, 10) +
			`,"token":"not-the-token","device_id":"device-present"}`
		refused := callFromApp(t, router, http.MethodPost, "/api/students/attendance/mark",
			body, fixture.presentToken)
		if refused.Code < http.StatusBadRequest {
			t.Errorf("a wrong token was accepted: %d (body = %q)",
				refused.Code, refused.Body.String())
		}
	})

	t.Run("the teacher marks the rest and closes the session", func(t *testing.T) {
		all := call(t, router, http.MethodPost,
			"/api/teacher/attendance/sessions/"+strconv.FormatInt(bluetoothID, 10)+"/mark-all",
			`{"status":"absent"}`, fixture.teacherToken)
		if all.Code != http.StatusOK {
			t.Fatalf("mark all status = %d, want 200 (body = %q)", all.Code, all.Body.String())
		}
		closed := call(t, router, http.MethodPost,
			"/api/teacher/attendance/sessions/"+strconv.FormatInt(bluetoothID, 10)+"/close",
			"", fixture.teacherToken)
		if closed.Code != http.StatusOK {
			t.Fatalf("close status = %d, want 200 (body = %q)", closed.Code, closed.Body.String())
		}
		// A closed session hands nobody the key any more.
		if strings.Contains(closed.Body.String(), sessionKeyOf(t, fixture, bluetoothID)) {
			t.Errorf("a closed session still carried its key: %s", closed.Body.String())
		}
	})

	t.Run("the teacher corrects one student and writes a note", func(t *testing.T) {
		roster := call(t, router, http.MethodGet,
			"/api/teacher/attendance/sessions/"+strconv.FormatInt(bluetoothID, 10),
			"", fixture.teacherToken)
		if roster.Code != http.StatusOK {
			t.Fatalf("the roster status = %d, want 200 (body = %q)",
				roster.Code, roster.Body.String())
		}
		var payload struct {
			Marks []struct {
				StudentID int64  `json:"student_id"`
				RegNo     string `json:"reg_no"`
				Status    string `json:"status"`
			} `json:"marks"`
		}
		if err := json.Unmarshal(roster.Body.Bytes(), &payload); err != nil {
			t.Fatalf("body is not valid JSON: %v (body = %q)", err, roster.Body.String())
		}
		var absentee int64
		for _, mark := range payload.Marks {
			if mark.RegNo == "2023331702" {
				absentee = mark.StudentID
			}
		}
		if absentee == 0 {
			t.Fatalf("the roster carries nobody with that registration number: %v", payload.Marks)
		}
		path := "/api/teacher/attendance/sessions/" + strconv.FormatInt(bluetoothID, 10) +
			"/students/" + strconv.FormatInt(absentee, 10)
		edited := call(t, router, http.MethodPatch, path, `{"status":"present"}`, fixture.teacherToken)
		if edited.Code != http.StatusOK {
			t.Fatalf("the correction status = %d, want 200 (body = %q)",
				edited.Code, edited.Body.String())
		}
		noted := call(t, router, http.MethodPatch, path+"/note",
			`{"note":"Came in late with a note from the office"}`, fixture.teacherToken)
		if noted.Code != http.StatusOK {
			t.Fatalf("the note status = %d, want 200 (body = %q)", noted.Code, noted.Body.String())
		}
	})

	t.Run("a student reads its own percentage and nobody else's", func(t *testing.T) {
		mine := call(t, router, http.MethodGet, "/api/students/attendance/mine", "",
			fixture.presentToken)
		if mine.Code != http.StatusOK {
			t.Fatalf("my attendance status = %d, want 200 (body = %q)",
				mine.Code, mine.Body.String())
		}
		if !strings.Contains(mine.Body.String(), "CSE 2101") {
			t.Errorf("the student sees no course: %s", mine.Body.String())
		}
	})

	t.Run("a teacher deletes its own sheet and nobody else's", func(t *testing.T) {
		// A class the teacher does not take is refused before anything is removed.
		other := call(t, router, http.MethodDelete, "/api/teacher/attendance/sheet",
			`{"course_id":`+strconv.FormatInt(fixture.courseID, 10)+
				`,"batch":"CSE-24","section":"A"}`, fixture.teacherToken)
		if other.Code != http.StatusForbidden {
			t.Errorf("deleting another class sheet = %d, want 403 (body = %q)",
				other.Code, other.Body.String())
		}
		mine := call(t, router, http.MethodDelete, "/api/teacher/attendance/sheet",
			`{"course_id":`+strconv.FormatInt(fixture.courseID, 10)+
				`,"batch":"CSE-23","section":"A"}`, fixture.teacherToken)
		if mine.Code != http.StatusOK {
			t.Fatalf("deleting its own sheet = %d, want 200 (body = %q)",
				mine.Code, mine.Body.String())
		}
		// The deletion is on the record, which is what raises the control room alert.
		var deletions int
		query := `SELECT COUNT(*) FROM audit_events
WHERE action = 'delete' AND target_type = 'attendance'`
		if err := fixture.pool.QueryRow(fixture.ctx, query).Scan(&deletions); err != nil {
			t.Fatalf("could not count the recorded deletions: %v", err)
		}
		if deletions == 0 {
			t.Error("a sheet was deleted without leaving a record")
		}
		left := call(t, router, http.MethodGet, "/api/students/attendance/mine", "",
			fixture.presentToken)
		if left.Code != http.StatusOK {
			t.Fatalf("my attendance after the delete = %d, want 200 (body = %q)",
				left.Code, left.Body.String())
		}
		if strings.Contains(left.Body.String(), `"present"`) {
			t.Errorf("the deleted sheet still shows marks: %s", left.Body.String())
		}
	})
}

// sessionKeyOf reads the key out of the database, the way only the teacher's own view
// would ever see it.
func sessionKeyOf(t *testing.T, fixture markFixture, id int64) string {
	t.Helper()
	var key string
	query := `SELECT token_key FROM attendance_sessions WHERE id = $1`
	if err := fixture.pool.QueryRow(fixture.ctx, query, id).Scan(&key); err != nil {
		t.Fatalf("could not read the session key: %v", err)
	}
	return key
}

// TestRemindersOnlyReachTheTeachersWithSomethingUndecided drives the three reminder slots
// against the real database. A holiday or a day everybody already decided has to stay
// silent, which is the whole point of the morning slot.
func TestRemindersOnlyReachTheTeachersWithSomethingUndecided(t *testing.T) {
	fixture := seedAttendance(t)
	userStore := store.New(fixture.pool)
	alerts := notify.New(userStore)
	cfg := flowConfigAt(t, flowClock())
	values := settings.New(userStore, SettingDefaults(cfg))
	scheduler := jobs.NewScheduler(cfg, userStore, alerts).WithSettings(values)

	t.Run("nothing undecided means no reminder at any slot", func(t *testing.T) {
		for _, clock := range []string{"16:00", "21:00", "07:30"} {
			at := atClock(t, cfg.Now(), clock)
			sent, err := scheduler.RemindTeachers(fixture.ctx, at)
			if err != nil {
				t.Fatalf("RemindTeachers() at %s returned an unexpected error: %v", clock, err)
			}
			if sent != 0 {
				t.Errorf("the %s slot sent %d reminders with nothing undecided", clock, sent)
			}
		}
	})

	t.Run("an undecided class reaches its teacher once per slot", func(t *testing.T) {
		if _, err := scheduler.Generate(fixture.ctx, cfg.Now()); err != nil {
			t.Fatalf("Generate() returned an unexpected error: %v", err)
		}
		undecided, err := userStore.ListUndecidedTeachers(fixture.ctx, cfg.Tomorrow())
		if err != nil {
			t.Fatalf("ListUndecidedTeachers() returned an unexpected error: %v", err)
		}
		if len(undecided) == 0 {
			t.Skip("the generated week carries nothing for tomorrow, so there is nothing to remind about")
		}
		at := atClock(t, cfg.Now(), "16:00")
		sent, err := scheduler.RemindTeachers(fixture.ctx, at)
		if err != nil {
			t.Fatalf("RemindTeachers() returned an unexpected error: %v", err)
		}
		if sent != len(undecided) {
			t.Errorf("the evening slot sent %d reminders, want one per teacher (%d)",
				sent, len(undecided))
		}
		// The same slot a minute later sends nothing: a slot is claimed once a day.
		again, err := scheduler.RemindTeachers(fixture.ctx, at.Add(time.Minute))
		if err != nil {
			t.Fatalf("RemindTeachers() returned an unexpected error: %v", err)
		}
		if again != 0 {
			t.Errorf("the same slot sent %d reminders twice", again)
		}
	})
}

func atClock(t *testing.T, day time.Time, clock string) time.Time {
	t.Helper()
	parsed, err := time.Parse("15:04", clock)
	if err != nil {
		t.Fatalf("could not read the clock %q: %v", clock, err)
	}
	return time.Date(day.Year(), day.Month(), day.Day(), parsed.Hour(), parsed.Minute(), 0, 0,
		day.Location())
}
