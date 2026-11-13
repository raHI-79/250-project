package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/raHI-79/250-project/backend/internal/attendance"
)

type ownMark struct {
	RegNo    string `json:"reg_no"`
	Status   string `json:"status"`
	Method   string `json:"method"`
	MarkedAt string `json:"marked_at"`
}

func sessionMarks(t *testing.T, fixture markFixture, id int64) map[string]ownMark {
	t.Helper()
	rec := call(t, fixture.router, http.MethodGet,
		"/api/teacher/attendance/sessions/"+strconv.FormatInt(id, 10), "", fixture.teacherToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("the session %d status = %d, want 200 (body = %q)", id, rec.Code, rec.Body.String())
	}
	var payload struct {
		Marks []ownMark `json:"marks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	marks := make(map[string]ownMark, len(payload.Marks))
	for _, mark := range payload.Marks {
		marks[mark.RegNo] = mark
	}
	return marks
}

func startedSession(t *testing.T, fixture markFixture, extra string) int64 {
	t.Helper()
	body := `{"course_id":` + strconv.FormatInt(fixture.courseID, 10) +
		`,"batch":"CSE-23","section":"A","method":"bluetooth","minutes":240` + extra + `}`
	rec := call(t, fixture.router, http.MethodPost, "/api/teacher/attendance/sessions",
		body, fixture.teacherToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("start status = %d, want 201 (body = %q)", rec.Code, rec.Body.String())
	}
	var session map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &session); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	return sessionID(t, session)
}

func placeTeacher(t *testing.T, fixture markFixture, id int64) {
	t.Helper()
	rec := call(t, fixture.router, http.MethodPost,
		"/api/teacher/attendance/sessions/"+strconv.FormatInt(id, 10)+"/location",
		`{"lat":24.9165,"lng":91.8316,"accuracy_m":8}`, fixture.teacherToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("the teacher position status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
}

func webMark(t *testing.T, fixture markFixture, id int64, code string, located bool, token string) (int, string) {
	t.Helper()
	body := `{"session_id":` + strconv.FormatInt(id, 10) + `,"code":"` + code + `"`
	if located {
		body += `,"lat":24.9165,"lng":91.8316,"accuracy_m":20`
	}
	body += `}`
	rec := call(t, fixture.router, http.MethodPost, "/api/students/attendance/mark", body, token)
	var payload struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &payload)
	return rec.Code, payload.Error
}

func currentCode(t *testing.T, fixture markFixture, id int64) string {
	t.Helper()
	return attendance.CurrentCode(sessionKeyOf(t, fixture, id), id, flowClock())
}

func otherCode(t *testing.T, fixture markFixture, target int64, key string, source int64, back int64) string {
	t.Helper()
	now := flowClock()
	slice := attendance.Slice(now, attendance.CodeSliceSeconds)
	targetKey := sessionKeyOf(t, fixture, target)
	for step := back; step < back+50; step++ {
		code := attendance.Code(key, source, slice-step)
		if !attendance.CodeMatches(targetKey, target, now, code) {
			return code
		}
	}
	t.Fatal("could not draw a code that differs from the current one")
	return ""
}

func closeSession(t *testing.T, fixture markFixture, id int64) {
	t.Helper()
	rec := call(t, fixture.router, http.MethodPost,
		"/api/teacher/attendance/sessions/"+strconv.FormatInt(id, 10)+"/close", "", fixture.teacherToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("close status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
}

func studentPresent(t *testing.T, fixture markFixture, token string) bool {
	t.Helper()
	rec := call(t, fixture.router, http.MethodGet, "/api/students/attendance/open", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("the open session status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	var payload struct {
		Session *struct {
			Present bool `json:"present"`
		} `json:"session"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	if payload.Session == nil {
		t.Fatalf("the student sees no open session: %s", rec.Body.String())
	}
	return payload.Session.Present
}

func TestAMarkBelongsToItsOwnSessionAndItsOwnStudent(t *testing.T) {
	fixture := seedAttendance(t)
	first := startedSession(t, fixture, "")

	t.Run("starting a session marks nobody", func(t *testing.T) {
		for regNo, mark := range sessionMarks(t, fixture, first) {
			if mark.Status != "" || mark.MarkedAt != "" {
				t.Errorf("%s is %q by %q right after the start, want unmarked", regNo, mark.Status, mark.Method)
			}
		}
		if studentPresent(t, fixture, fixture.presentToken) {
			t.Error("the student is present before doing anything")
		}
	})

	t.Run("a code without a location is refused", func(t *testing.T) {
		placeTeacher(t, fixture, first)
		code, message := webMark(t, fixture, first, currentCode(t, fixture, first), false, fixture.presentToken)
		if code != http.StatusForbidden || message != attendance.MessageNoPosition {
			t.Errorf("a code without a position = %d %q, want 403 %q", code, message, attendance.MessageNoPosition)
		}
	})

	t.Run("an expired code is refused", func(t *testing.T) {
		old := otherCode(t, fixture, first, sessionKeyOf(t, fixture, first), first, 5)
		code, message := webMark(t, fixture, first, old, true, fixture.presentToken)
		if code != http.StatusForbidden || message != attendance.MessageWrongToken {
			t.Errorf("an expired code = %d %q, want 403 %q", code, message, attendance.MessageWrongToken)
		}
		if studentPresent(t, fixture, fixture.presentToken) {
			t.Error("the student is present after a refused code")
		}
	})

	t.Run("the current code with a location marks the student", func(t *testing.T) {
		code, message := webMark(t, fixture, first, currentCode(t, fixture, first), true, fixture.presentToken)
		if code != http.StatusCreated {
			t.Fatalf("the mark = %d %q, want 201", code, message)
		}
		marks := sessionMarks(t, fixture, first)
		if got := marks["2023331701"]; got.Status != attendance.Present || got.Method != attendance.MethodWeb {
			t.Errorf("the student = %+v, want present by the website code", got)
		}
		if got := marks["2023331702"]; got.Status != "" {
			t.Errorf("the other student = %+v, want unmarked while the session is open", got)
		}
	})

	firstKey := sessionKeyOf(t, fixture, first)

	t.Run("finishing makes the unmarked students absent", func(t *testing.T) {
		closeSession(t, fixture, first)
		marks := sessionMarks(t, fixture, first)
		if got := marks["2023331702"]; got.Status != attendance.Absent || got.Method != "system" {
			t.Errorf("the unmarked student = %+v, want absent by the system", got)
		}
		if got := marks["2023331701"]; got.Status != attendance.Present {
			t.Errorf("the marked student = %+v, want present kept", got)
		}
	})

	var second int64

	t.Run("a second session the same day starts empty", func(t *testing.T) {
		second = startedSession(t, fixture, "")
		if second == first {
			t.Fatalf("the second start reopened the session %d", first)
		}
		if sessionKeyOf(t, fixture, second) == firstKey {
			t.Error("the second session reuses the key of the first")
		}
		for regNo, mark := range sessionMarks(t, fixture, second) {
			if mark.Status != "" {
				t.Errorf("%s is %q by %q in the new session, want unmarked", regNo, mark.Status, mark.Method)
			}
		}
		if studentPresent(t, fixture, fixture.presentToken) {
			t.Error("the student is present in the new session without acting")
		}
		if got := sessionMarks(t, fixture, first)["2023331701"]; got.Status != attendance.Present {
			t.Errorf("the first session lost its mark: %+v", got)
		}
	})

	t.Run("a code of another session is refused", func(t *testing.T) {
		placeTeacher(t, fixture, second)
		foreign := otherCode(t, fixture, second, firstKey, first, 0)
		code, message := webMark(t, fixture, second, foreign, true, fixture.presentToken)
		if code != http.StatusForbidden || message != attendance.MessageWrongToken {
			t.Errorf("a code of the first session = %d %q, want 403 %q", code, message, attendance.MessageWrongToken)
		}
		code, message = webMark(t, fixture, first, attendance.CurrentCode(firstKey, first, flowClock()), true,
			fixture.presentToken)
		if code != http.StatusConflict || message != attendance.MessageSessionClosed {
			t.Errorf("a code for the ended session = %d %q, want 409 %q", code, message,
				attendance.MessageSessionClosed)
		}
		if studentPresent(t, fixture, fixture.presentToken) {
			t.Error("the student is present after two refused codes")
		}
	})

	t.Run("the two sessions keep separate marks", func(t *testing.T) {
		code, message := webMark(t, fixture, second, currentCode(t, fixture, second), true, fixture.absentToken)
		if code != http.StatusCreated {
			t.Fatalf("the mark = %d %q, want 201", code, message)
		}
		later := sessionMarks(t, fixture, second)
		if later["2023331702"].Status != attendance.Present || later["2023331701"].Status != "" {
			t.Errorf("the second session = %+v, want only the second student present", later)
		}
		earlier := sessionMarks(t, fixture, first)
		if earlier["2023331701"].Status != attendance.Present || earlier["2023331702"].Status != attendance.Absent {
			t.Errorf("the first session = %+v, want it unchanged", earlier)
		}
	})
}

func TestOnePhoneIsNeverATeacherAndAStudent(t *testing.T) {
	fixture := seedAttendance(t)
	router := fixture.router
	bound := callFromApp(t, router, http.MethodPost, "/api/students/attendance/phone",
		`{"device_id":"device-student"}`, fixture.presentToken)
	if bound.Code != http.StatusOK {
		t.Fatalf("binding the phone status = %d, want 200 (body = %q)", bound.Code, bound.Body.String())
	}
	start := `{"course_id":` + strconv.FormatInt(fixture.courseID, 10) +
		`,"batch":"CSE-23","section":"A","method":"bluetooth","minutes":240,"device_id":"%s"}`

	refused := callFromApp(t, router, http.MethodPost, "/api/teacher/attendance/sessions",
		strings.Replace(start, "%s", "device-student", 1), fixture.teacherToken)
	if refused.Code != http.StatusForbidden || !strings.Contains(refused.Body.String(), attendance.MessageStudentPhone) {
		t.Fatalf("starting from a student's phone = %d, want 403 (body = %q)", refused.Code, refused.Body.String())
	}

	opened := callFromApp(t, router, http.MethodPost, "/api/teacher/attendance/sessions",
		strings.Replace(start, "%s", "device-teacher", 1), fixture.teacherToken)
	if opened.Code != http.StatusCreated {
		t.Fatalf("starting from the teacher's phone = %d, want 201 (body = %q)", opened.Code, opened.Body.String())
	}
	id := int64(number(t, opened, "id"))
	placeTeacher(t, fixture, id)
	if _, err := fixture.pool.Exec(fixture.ctx,
		`UPDATE device_bindings SET device_id = 'device-teacher' WHERE device_id = 'device-student'`); err != nil {
		t.Fatalf("could not hand the teacher's phone to the student: %v", err)
	}
	token := attendance.CurrentToken(sessionKeyOf(t, fixture, id), id, flowClock())
	body := `{"session_id":` + strconv.FormatInt(id, 10) + `,"token":"` + token +
		`","device_id":"device-teacher","lat":24.9165,"lng":91.8316,"accuracy_m":10}`
	marked := callFromApp(t, router, http.MethodPost, "/api/students/attendance/mark", body, fixture.presentToken)
	if marked.Code != http.StatusForbidden || !strings.Contains(marked.Body.String(), attendance.MessageTeacherPhone) {
		t.Errorf("marking from the teacher's phone = %d, want 403 (body = %q)", marked.Code, marked.Body.String())
	}
	for regNo, mark := range sessionMarks(t, fixture, id) {
		if mark.Status != "" {
			t.Errorf("%s is %q after the refused mark, want unmarked", regNo, mark.Status)
		}
	}
}
