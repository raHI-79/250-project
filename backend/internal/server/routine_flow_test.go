package server

import (
	"encoding/json"
	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

const (
	flowCourse  = `{"code":"cse233","title":"Digital Logic Design"}`
	flowTeacher = `{"name":"Routine Teacher","teacher_id":"T-700","email":"routineteacher@sust.edu",
"phone":"01712345700","password":"sust-250-project","confirm_password":"sust-250-project"}`
	flowStudentOne = `{"full_name":"Routine Student","reg_no":"2023331700","batch":"CSE-23","section":"A",
"email":"routinestudent@student.sust.edu","phone":"01712345701","blood_group":"O+",
"password":"sust-250-project","confirm_password":"sust-250-project","has_drop_course":false}`
	flowStudentTwo = `{"full_name":"Other Batch Student","reg_no":"2022331701","batch":"CSE-22","section":"A",
"email":"otherbatch@student.sust.edu","phone":"01712345702","blood_group":"A+",
"password":"sust-250-project","confirm_password":"sust-250-project","has_drop_course":false}`
	flowPassword = "sust-250-project"
)

func listItems(t *testing.T, rec *httptest.ResponseRecorder) []map[string]any {
	t.Helper()
	var payload struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	return payload.Items
}

func routineBody(section string, courseID, teacherID int64, day int, start, end, room, kind string) string {
	return `{"batch":"CSE-23","section":"` + section +
		`","course_id":` + strconv.FormatInt(courseID, 10) +
		`,"teacher_id":` + strconv.FormatInt(teacherID, 10) +
		`,"day_of_week":` + strconv.Itoa(day) +
		`,"start_time":"` + start + `","end_time":"` + end +
		`","room":"` + room + `","type":"` + kind + `"}`
}

func approveStudent(t *testing.T, router http.Handler, adminToken, regNo string) int64 {
	t.Helper()
	pending := call(t, router, http.MethodGet, "/api/admin/registrations", "", adminToken)
	if pending.Code != http.StatusOK {
		t.Fatalf("registrations status = %d, want 200 (body = %q)", pending.Code, pending.Body.String())
	}
	for _, item := range listItems(t, pending) {
		if item["reg_no"] != regNo {
			continue
		}
		userID := int64(item["id"].(float64))
		approved := call(t, router, http.MethodPost,
			"/api/admin/registrations/"+strconv.FormatInt(userID, 10)+"/approve", "", adminToken)
		if approved.Code != http.StatusOK {
			t.Fatalf("approve status = %d, want 200 (body = %q)", approved.Code, approved.Body.String())
		}
		return userID
	}
	t.Fatalf("no pending registration has the registration number %q", regNo)
	return 0
}

func loginToken(t *testing.T, router http.Handler, identifier string) string {
	t.Helper()
	body := `{"identifier":"` + identifier + `","password":"` + flowPassword + `"}`
	rec := call(t, router, http.MethodPost, "/api/auth/login", body, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	return field(t, rec, "token")
}

func TestRoutineFlow(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	adminToken := seedAdmin(t, ctx, pool, router)
	course := call(t, router, http.MethodPost, "/api/admin/courses", flowCourse, adminToken)
	if course.Code != http.StatusCreated {
		t.Fatalf("create course status = %d, want 201 (body = %q)", course.Code, course.Body.String())
	}
	if got := field(t, course, "code"); got != "CSE233" {
		t.Errorf("course code = %q, want the uppercase code", got)
	}
	courseID := int64(number(t, course, "id"))
	duplicate := call(t, router, http.MethodPost, "/api/admin/courses", flowCourse, adminToken)
	if duplicate.Code != http.StatusConflict {
		t.Errorf("duplicate course status = %d, want 409 (body = %q)", duplicate.Code, duplicate.Body.String())
	}
	createdTeacher := teacherInviteCall(t, router, adminToken, flowTeacher)
	if createdTeacher.Code != http.StatusCreated {
		t.Fatalf("create teacher status = %d, want 201 (body = %q)", createdTeacher.Code, createdTeacher.Body.String())
	}
	teacherUserID := int64(number(t, createdTeacher, "id"))
	teachers := call(t, router, http.MethodGet, "/api/lookups/teachers", "", adminToken)
	if teachers.Code != http.StatusOK {
		t.Fatalf("teacher lookup status = %d, want 200 (body = %q)", teachers.Code, teachers.Body.String())
	}
	var teacherID int64
	for _, item := range listItems(t, teachers) {
		if int64(item["user_id"].(float64)) == teacherUserID {
			teacherID = int64(item["id"].(float64))
		}
	}
	if teacherID == 0 {
		t.Fatalf("the teacher lookup does not carry the new teacher (body = %q)", teachers.Body.String())
	}
	first := call(t, router, http.MethodPost, "/api/admin/routines",
		routineBody("A", courseID, teacherID, 1, "10:30", "12:45", "1003", "lab"), adminToken)
	if first.Code != http.StatusCreated {
		t.Fatalf("first routine status = %d, want 201 (body = %q)", first.Code, first.Body.String())
	}
	if got := field(t, first, "day"); got != "Monday" {
		t.Errorf("day = %q, want Monday", got)
	}
	overlapping := call(t, router, http.MethodPost, "/api/admin/routines",
		routineBody("A", courseID, teacherID, 1, "11:00", "12:00", "1003", "class"), adminToken)
	if overlapping.Code != http.StatusConflict {
		t.Fatalf("overlapping routine status = %d, want 409 (body = %q)", overlapping.Code, overlapping.Body.String())
	}
	wantConflict := "Room 1003 is already booked on Monday 10:30–12:45 by CSE-23 section A (CSE233)."
	if got := field(t, overlapping, "error"); got != wantConflict {
		t.Errorf("conflict error = %q, want %q", got, wantConflict)
	}
	tests := []struct {
		name string
		body string
	}{
		{name: "same time in another room", body: routineBody("A", courseID, teacherID, 1, "11:00", "12:00", "1004", "class")},
		{name: "right after the first class", body: routineBody("A", courseID, teacherID, 1, "12:45", "14:00", "1003", "class")},
		{name: "same room on another day", body: routineBody("A", courseID, teacherID, 2, "10:30", "12:45", "1003", "lab")},
		{name: "both sections", body: routineBody("ALL", courseID, teacherID, 3, "09:00", "10:00", "2001", "class")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rec := call(t, router, http.MethodPost, "/api/admin/routines", test.body, adminToken)
			if rec.Code != http.StatusCreated {
				t.Fatalf("status = %d, want 201 (body = %q)", rec.Code, rec.Body.String())
			}
		})
	}
	listed := call(t, router, http.MethodGet, "/api/admin/routines?batch=CSE-23", "", adminToken)
	if listed.Code != http.StatusOK {
		t.Fatalf("routine list status = %d, want 200 (body = %q)", listed.Code, listed.Body.String())
	}
	entries := listItems(t, listed)
	if len(entries) != 5 {
		t.Fatalf("got %d routine entries, want 5 (body = %q)", len(entries), listed.Body.String())
	}
	wantOrder := []struct {
		section string
		day     float64
		start   string
	}{
		{section: "A", day: 1, start: "10:30"},
		{section: "A", day: 1, start: "11:00"},
		{section: "A", day: 1, start: "12:45"},
		{section: "A", day: 2, start: "10:30"},
		{section: "ALL", day: 3, start: "09:00"},
	}
	for i, want := range wantOrder {
		entry := entries[i]
		if entry["section"] != want.section || entry["day_of_week"] != want.day || entry["start_time"] != want.start {
			t.Errorf("entry %d = %v, want section %s on day %v at %s", i, entry, want.section, want.day, want.start)
		}
		if entry["course_code"] != "CSE233" || entry["teacher_name"] != "Routine Teacher" {
			t.Errorf("entry %d hides the course or the teacher: %v", i, entry)
		}
	}
	sectionB := call(t, router, http.MethodGet, "/api/admin/routines?batch=CSE-23&section=B", "", adminToken)
	if sectionB.Code != http.StatusOK {
		t.Fatalf("section filter status = %d, want 200 (body = %q)", sectionB.Code, sectionB.Body.String())
	}
	bothSections := listItems(t, sectionB)
	if len(bothSections) != 1 || bothSections[0]["section"] != "ALL" {
		t.Fatalf("section B sees %v, want only the entry for both sections", bothSections)
	}
	removableID := int64(bothSections[0]["id"].(float64))
	removePath := "/api/admin/routines/" + strconv.FormatInt(removableID, 10)
	removed := call(t, router, http.MethodDelete, removePath, "", adminToken)
	if removed.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204 (body = %q)", removed.Code, removed.Body.String())
	}
	removedAgain := call(t, router, http.MethodDelete, removePath, "", adminToken)
	if removedAgain.Code != http.StatusNotFound {
		t.Errorf("second delete status = %d, want 404 (body = %q)", removedAgain.Code, removedAgain.Body.String())
	}
	batches := call(t, router, http.MethodGet, "/api/lookups/batches", "", adminToken)
	if batches.Code != http.StatusOK {
		t.Fatalf("batch lookup status = %d, want 200 (body = %q)", batches.Code, batches.Body.String())
	}
	var batchPayload struct {
		Items []string `json:"items"`
	}
	if err := json.Unmarshal(batches.Body.Bytes(), &batchPayload); err != nil {
		t.Fatalf("the batch lookup body is not valid JSON: %v", err)
	}
	if len(batchPayload.Items) != 1 || batchPayload.Items[0] != "CSE-23" {
		t.Errorf("batches = %v, want only CSE-23", batchPayload.Items)
	}
	sections := call(t, router, http.MethodGet, "/api/lookups/sections?batch=CSE-23", "", adminToken)
	var sectionPayload struct {
		Items []string `json:"items"`
	}
	if err := json.Unmarshal(sections.Body.Bytes(), &sectionPayload); err != nil {
		t.Fatalf("the section lookup body is not valid JSON: %v", err)
	}
	if len(sectionPayload.Items) != 1 || sectionPayload.Items[0] != "A" {
		t.Errorf("sections = %v, want only A because ALL is not a real section", sectionPayload.Items)
	}
	courses := call(t, router, http.MethodGet, "/api/lookups/courses", "", adminToken)
	if list := listItems(t, courses); len(list) != 1 || list[0]["code"] != "CSE233" {
		t.Errorf("course lookup = %v, want the one course", list)
	}
}

func TestNotificationFlow(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	adminToken := seedAdmin(t, ctx, pool, router)
	for _, body := range []string{flowStudentOne, flowStudentTwo} {
		rec := registerStudent(t, router, body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("register status = %d, want 201 (body = %q)", rec.Code, rec.Body.String())
		}
		verifyStudentEmail(t, router, rec, regNoOf(t, body))
	}
	firstID := approveStudent(t, router, adminToken, "2023331700")
	approveStudent(t, router, adminToken, "2022331701")
	firstToken := loginToken(t, router, "2023331700")
	secondToken := loginToken(t, router, "2022331701")
	notifier := notify.New(store.New(pool))
	if err := notifier.NotifyUser(ctx, firstID, notify.TypeDecision, "Your request was approved", "Monday 10:30"); err != nil {
		t.Fatalf("NotifyUser() returned an unexpected error: %v", err)
	}
	if err := notifier.NotifyBroadcast(ctx, "student", "CSE-23", "A", notify.TypeClass, "Extra class added", ""); err != nil {
		t.Fatalf("NotifyBroadcast() returned an unexpected error: %v", err)
	}
	if err := notifier.NotifyBroadcast(ctx, "student", "CSE-22", "", notify.TypeNotice, "Other batch notice", ""); err != nil {
		t.Fatalf("NotifyBroadcast() returned an unexpected error: %v", err)
	}
	tests := []struct {
		name       string
		token      string
		wantTitles map[string]bool
	}{
		{
			name:       "the student of CSE-23 section A",
			token:      firstToken,
			wantTitles: map[string]bool{"Your request was approved": true, "Extra class added": true},
		},
		{
			name:       "the student of CSE-22",
			token:      secondToken,
			wantTitles: map[string]bool{"Other batch notice": true},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rec := call(t, router, http.MethodGet, "/api/notifications", "", test.token)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
			}
			list := listItems(t, rec)
			if len(list) != len(test.wantTitles) {
				t.Fatalf("got %d notifications, want %d (body = %q)", len(list), len(test.wantTitles), rec.Body.String())
			}
			for _, item := range list {
				title, _ := item["title"].(string)
				if !test.wantTitles[title] {
					t.Errorf("the list carries %q, want only %v", title, test.wantTitles)
				}
			}
		})
	}
	counted := call(t, router, http.MethodGet, "/api/notifications/unread-count", "", firstToken)
	if counted.Code != http.StatusOK {
		t.Fatalf("unread count status = %d, want 200 (body = %q)", counted.Code, counted.Body.String())
	}
	if got := number(t, counted, "count"); got != 2 {
		t.Errorf("count = %v, want 2", got)
	}
	ahead := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	fresh := call(t, router, http.MethodGet, "/api/notifications/unread-count?since="+ahead, "", firstToken)
	if got := number(t, fresh, "count"); got != 0 {
		t.Errorf("count since the future = %v, want 0", got)
	}
	badLimit := call(t, router, http.MethodGet, "/api/notifications?limit=0", "", firstToken)
	if badLimit.Code != http.StatusBadRequest {
		t.Errorf("limit 0 status = %d, want 400", badLimit.Code)
	}
	if got := field(t, badLimit, "error"); got != handler.MessageInvalidNotificationLimit {
		t.Errorf("limit error = %q, want %q", got, handler.MessageInvalidNotificationLimit)
	}
	asAdmin := call(t, router, http.MethodGet, "/api/notifications", "", adminToken)
	if asAdmin.Code != http.StatusForbidden {
		t.Fatalf("admin status = %d, want 403 (body = %q)", asAdmin.Code, asAdmin.Body.String())
	}
	if got := field(t, asAdmin, "error"); got != middleware.MessageAccessDenied {
		t.Errorf("admin error = %q, want %q", got, middleware.MessageAccessDenied)
	}
}
