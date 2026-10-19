package server

import (
	"encoding/json"
	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/jobs"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

const (
	scheduleCourse  = `{"code":"cse233","title":"Digital Logic Design"}`
	scheduleTeacher = `{"name":"Schedule Teacher","teacher_id":"T-900","email":"scheduleteacher@sust.edu",
"phone":"01712345900","password":"sust-250-project","confirm_password":"sust-250-project"}`
	scheduleCR = `{"full_name":"Schedule CR","reg_no":"2023331900","batch":"CSE-23","section":"A",
"email":"schedulecr@student.sust.edu","phone":"01712345901","blood_group":"O+",
"password":"sust-250-project","confirm_password":"sust-250-project"}`
	scheduleStudent = `{"full_name":"Schedule Student","reg_no":"2023331901","batch":"CSE-23","section":"A",
"email":"schedulestudent@student.sust.edu","phone":"01712345902","blood_group":"A+",
"password":"sust-250-project","confirm_password":"sust-250-project","has_drop_course":false}`
)

func groupsOf(t *testing.T, rec *httptest.ResponseRecorder) []map[string]any {
	t.Helper()
	var payload struct {
		Groups []map[string]any `json:"groups"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	return payload.Groups
}

func groupItems(t *testing.T, group map[string]any) []map[string]any {
	t.Helper()
	raw, ok := group["items"].([]any)
	if !ok {
		t.Fatalf("the group has no items: %v", group)
	}
	items := make([]map[string]any, 0, len(raw))
	for _, entry := range raw {
		item, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("an item is not an object: %v", entry)
		}
		items = append(items, item)
	}
	return items
}

func TestScheduleFlow(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	adminToken := seedAdmin(t, ctx, pool, router)
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load the timezone: %v", err)
	}
	now := time.Now().In(loc)
	tomorrow := now.AddDate(0, 0, 1)
	date := tomorrow.Format(validate.DateLayout)
	course := call(t, router, http.MethodPost, "/api/admin/courses", scheduleCourse, adminToken)
	if course.Code != http.StatusCreated {
		t.Fatalf("create course status = %d, want 201 (body = %q)", course.Code, course.Body.String())
	}
	courseID := int64(number(t, course, "id"))
	createdTeacher := teacherInviteCall(t, router, adminToken, scheduleTeacher)
	if createdTeacher.Code != http.StatusCreated {
		t.Fatalf("create teacher status = %d, want 201 (body = %q)", createdTeacher.Code, createdTeacher.Body.String())
	}
	teacherUserID := int64(number(t, createdTeacher, "id"))
	var teacherID int64
	for _, item := range listItems(t, call(t, router, http.MethodGet, "/api/lookups/teachers", "", adminToken)) {
		if int64(item["user_id"].(float64)) == teacherUserID {
			teacherID = int64(item["id"].(float64))
		}
	}
	if teacherID == 0 {
		t.Fatal("the new teacher is missing from the teacher lookup")
	}
	onboardCR(t, router, adminToken, scheduleCR)
	registeredStudent := registerStudent(t, router, scheduleStudent)
	if registeredStudent.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want 201 (body = %q)", registeredStudent.Code, registeredStudent.Body.String())
	}
	verifyStudentEmail(t, router, registeredStudent, "2023331901")
	approveStudent(t, router, adminToken, "2023331901")
	crToken := loginToken(t, router, "2023331900")
	teacherToken := loginToken(t, router, "T-900")
	studentToken := loginToken(t, router, "2023331901")
	requested := call(t, router, http.MethodPost, "/api/cr/class-requests",
		classRequestBody("class", date, "10:30", "12:45", "1101", courseID, teacherID), crToken)
	if requested.Code != http.StatusCreated {
		t.Fatalf("class request status = %d, want 201 (body = %q)", requested.Code, requested.Body.String())
	}
	pending := call(t, router, http.MethodGet, "/api/teacher/requests", "", teacherToken)
	if pending.Code != http.StatusOK {
		t.Fatalf("teacher requests status = %d, want 200 (body = %q)", pending.Code, pending.Body.String())
	}
	waiting := listItems(t, pending)
	if len(waiting) != 1 {
		t.Fatalf("the teacher has %d requests, want 1 (body = %q)", len(waiting), pending.Body.String())
	}
	if waiting[0]["requested_by"] != "Schedule CR" || waiting[0]["requested_by_reg_no"] != "2023331900" {
		t.Errorf("request = %v, want the class representative who asked", waiting[0])
	}
	requestID := int64(waiting[0]["id"].(float64))
	requestPath := "/api/teacher/requests/" + strconv.FormatInt(requestID, 10)
	approved := call(t, router, http.MethodPost, requestPath+"/approve", "", teacherToken)
	if approved.Code != http.StatusOK {
		t.Fatalf("approve status = %d, want 200 (body = %q)", approved.Code, approved.Body.String())
	}
	again := call(t, router, http.MethodPost, requestPath+"/approve", "", teacherToken)
	if again.Code != http.StatusConflict {
		t.Errorf("second approve status = %d, want 409 (body = %q)", again.Code, again.Body.String())
	}
	schedule := call(t, router, http.MethodGet, "/api/teacher/schedule?date="+date, "", teacherToken)
	if schedule.Code != http.StatusOK {
		t.Fatalf("schedule status = %d, want 200 (body = %q)", schedule.Code, schedule.Body.String())
	}
	entries := listItems(t, schedule)
	if len(entries) != 1 || entries[0]["kind"] != "class" || entries[0]["cancelled"] != false {
		t.Fatalf("schedule = %v, want the approved class", entries)
	}
	upcoming := call(t, router, http.MethodGet, "/api/teacher/classes/upcoming", "", teacherToken)
	classes := listItems(t, upcoming)
	if len(classes) != 1 || classes[0]["event_date"] != date {
		t.Fatalf("upcoming classes = %v, want the class of tomorrow", classes)
	}
	classID := int64(classes[0]["id"].(float64))
	routinePath := "/api/students/upcoming-routine?date=" + date
	routine := call(t, router, http.MethodGet, routinePath, "", studentToken)
	if routine.Code != http.StatusOK {
		t.Fatalf("upcoming routine status = %d, want 200 (body = %q)", routine.Code, routine.Body.String())
	}
	if got := field(t, routine, "date"); got != date {
		t.Errorf("date = %q, want tomorrow %q", got, date)
	}
	groups := groupsOf(t, routine)
	if len(groups) != 1 || groups[0]["key"] != "A" || groups[0]["section_label"] != "Section A" {
		t.Fatalf("groups = %v, want one for section A", groups)
	}
	planned := groupItems(t, groups[0])
	if len(planned) != 1 || planned[0]["status"] != handler.StatusScheduled || planned[0]["course_code"] != "CSE233" {
		t.Fatalf("items = %v, want the scheduled class", planned)
	}
	if planned[0]["teacher_name"] != "Schedule Teacher" || planned[0]["room"] != "1101" {
		t.Errorf("item = %v, want the teacher and room", planned[0])
	}
	crDay := call(t, router, http.MethodGet, "/api/cr/day?date="+date, "", crToken)
	if crDay.Code != http.StatusOK {
		t.Fatalf("cr day status = %d, want 200 (body = %q)", crDay.Code, crDay.Body.String())
	}
	onThatDay := listItems(t, crDay)
	if len(onThatDay) != 1 || onThatDay[0]["source"] != "event" || onThatDay[0]["status"] != store.EventStatusApproved {
		t.Fatalf("the day panel shows %v, want the approved event", onThatDay)
	}
	cancelled := call(t, router, http.MethodPost,
		"/api/teacher/classes/"+strconv.FormatInt(classID, 10)+"/cancel", "", teacherToken)
	if cancelled.Code != http.StatusOK {
		t.Fatalf("cancel status = %d, want 200 (body = %q)", cancelled.Code, cancelled.Body.String())
	}
	today := call(t, router, http.MethodGet, "/api/students/today", "", studentToken)
	if today.Code != http.StatusOK {
		t.Fatalf("today status = %d, want 200 (body = %q)", today.Code, today.Body.String())
	}
	if got := field(t, today, "date"); got != now.Format(validate.DateLayout) {
		t.Errorf("today date = %q, want %q", got, now.Format(validate.DateLayout))
	}
	afterCancel := call(t, router, http.MethodGet, routinePath, "", studentToken)
	cancelledGroups := groupsOf(t, afterCancel)
	if len(cancelledGroups) != 1 {
		t.Fatalf("groups after the cancellation = %v, want one", cancelledGroups)
	}
	shown := groupItems(t, cancelledGroups[0])
	if len(shown) != 1 || shown[0]["status"] != handler.StatusCancelled {
		t.Fatalf("items after the cancellation = %v, want the class marked as cancelled", shown)
	}
	alerts := listItems(t, call(t, router, http.MethodGet, "/api/notifications", "", studentToken))
	if len(alerts) != 2 {
		t.Fatalf("the student has %d notifications, want the approval and the cancellation", len(alerts))
	}
	stale := call(t, router, http.MethodPost, "/api/cr/class-requests",
		classRequestBody("exam", date, "08:00", "09:00", "1102", courseID, teacherID), crToken)
	if stale.Code != http.StatusCreated {
		t.Fatalf("second class request status = %d, want 201 (body = %q)", stale.Code, stale.Body.String())
	}
	userStore := store.New(pool)
	count, err := jobs.NewAutoCanceller(userStore, notify.New(userStore)).AutoCancel(ctx, tomorrow)
	if err != nil {
		t.Fatalf("AutoCancel() returned an unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("%d requests were auto cancelled, want 1", count)
	}
	empty := call(t, router, http.MethodGet, "/api/teacher/requests", "", teacherToken)
	if left := listItems(t, empty); len(left) != 0 {
		t.Errorf("the teacher still has %v, want no pending request", left)
	}
	mine := listItems(t, call(t, router, http.MethodGet, "/api/cr/requests?status=auto_cancelled", "", crToken))
	if len(mine) != 1 || mine[0]["room"] != "1102" {
		t.Fatalf("auto cancelled requests = %v, want the exam in room 1102", mine)
	}
	crAlerts := listItems(t, call(t, router, http.MethodGet, "/api/notifications", "", crToken))
	if len(crAlerts) != 6 {
		t.Fatalf("the class representative has %d notifications, want the promotion, the three decisions and the two batch broadcasts", len(crAlerts))
	}
	if crAlerts[0]["title"] != jobs.NotificationTitle {
		t.Errorf("the newest notification = %v, want %q", crAlerts[0], jobs.NotificationTitle)
	}
}
