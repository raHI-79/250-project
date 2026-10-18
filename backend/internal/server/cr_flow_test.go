package server

import (
	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"net/http"
	"strconv"
	"testing"
	"time"
)

const (
	crFlowCourse  = `{"code":"cse233","title":"Digital Logic Design"}`
	crFlowTeacher = `{"name":"Request Teacher","teacher_id":"T-800","email":"requestteacher@sust.edu",
"phone":"01712345800","password":"sust-250-project","confirm_password":"sust-250-project"}`
	crFlowFirstCR = `{"full_name":"First CR","reg_no":"2023331800","batch":"CSE-23","section":"A",
"email":"firstcr@student.sust.edu","phone":"01712345801","blood_group":"O+",
"password":"sust-250-project","confirm_password":"sust-250-project"}`
	crFlowSecondCR = `{"full_name":"Second CR","reg_no":"2022331801","batch":"CSE-22","section":"B",
"email":"secondcr@student.sust.edu","phone":"01712345802","blood_group":"A+",
"password":"sust-250-project","confirm_password":"sust-250-project"}`
)

func classRequestBody(eventType, date, start, end, room string, courseID, teacherID int64) string {
	return `{"event_type":"` + eventType +
		`","course_id":` + strconv.FormatInt(courseID, 10) +
		`,"teacher_id":` + strconv.FormatInt(teacherID, 10) +
		`,"event_date":"` + date +
		`","start_time":"` + start + `","end_time":"` + end +
		`","room":"` + room + `"}`
}

func TestCRClassRequestFlow(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	adminToken := seedAdmin(t, ctx, pool, router)
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load the timezone: %v", err)
	}
	tomorrow := time.Now().In(loc).AddDate(0, 0, 1)
	date := tomorrow.Format(validate.DateLayout)
	weekday := int(tomorrow.Weekday())
	course := call(t, router, http.MethodPost, "/api/admin/courses", crFlowCourse, adminToken)
	if course.Code != http.StatusCreated {
		t.Fatalf("create course status = %d, want 201 (body = %q)", course.Code, course.Body.String())
	}
	courseID := int64(number(t, course, "id"))
	createdTeacher := teacherInviteCall(t, router, adminToken, crFlowTeacher)
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
	for _, body := range []string{crFlowFirstCR, crFlowSecondCR} {
		onboardCR(t, router, adminToken, body)
	}
	fixed := call(t, router, http.MethodPost, "/api/admin/routines",
		routineBody("A", courseID, teacherID, weekday, "08:00", "09:00", "1003", "class"), adminToken)
	if fixed.Code != http.StatusCreated {
		t.Fatalf("create routine status = %d, want 201 (body = %q)", fixed.Code, fixed.Body.String())
	}
	firstToken := loginToken(t, router, "2023331800")
	secondToken := loginToken(t, router, "2022331801")
	routine := call(t, router, http.MethodGet, "/api/cr/routine", "", firstToken)
	if routine.Code != http.StatusOK {
		t.Fatalf("routine status = %d, want 200 (body = %q)", routine.Code, routine.Body.String())
	}
	if entries := listItems(t, routine); len(entries) != 1 || entries[0]["room"] != "1003" {
		t.Fatalf("routine = %v, want the one fixed entry of CSE-23 section A", entries)
	}
	otherRoutine := call(t, router, http.MethodGet, "/api/cr/routine", "", secondToken)
	if entries := listItems(t, otherRoutine); len(entries) != 0 {
		t.Errorf("the class representative of CSE-22 sees %v, want nothing", entries)
	}
	booked := call(t, router, http.MethodPost, "/api/cr/class-requests",
		classRequestBody("extra", date, "10:30", "12:45", "1004", courseID, teacherID), firstToken)
	if booked.Code != http.StatusCreated {
		t.Fatalf("first class request status = %d, want 201 (body = %q)", booked.Code, booked.Body.String())
	}
	if got := field(t, booked, "batch"); got != "CSE-23" {
		t.Errorf("batch = %q, want CSE-23 from the profile", got)
	}
	if got := field(t, booked, "status"); got != "pending" {
		t.Errorf("status = %q, want pending", got)
	}
	taken := call(t, router, http.MethodPost, "/api/cr/class-requests",
		classRequestBody("class", date, "11:00", "12:00", "1004", courseID, teacherID), secondToken)
	if taken.Code != http.StatusConflict {
		t.Fatalf("second class request status = %d, want 409 (body = %q)", taken.Code, taken.Body.String())
	}
	if got := field(t, taken, "error"); got != handler.MessageRoomUnavailable {
		t.Errorf("error = %q, want %q", got, handler.MessageRoomUnavailable)
	}
	reserved := call(t, router, http.MethodPost, "/api/cr/class-requests",
		classRequestBody("class", date, "08:00", "09:00", "1003", courseID, teacherID), secondToken)
	if reserved.Code != http.StatusConflict {
		t.Fatalf("reserved room status = %d, want 409 (body = %q)", reserved.Code, reserved.Body.String())
	}
	wantReserved := "Room 1003 is reserved for CSE-23 section A (CSE233) every " +
		validate.DayName(weekday) + " 08:00–09:00."
	if got := field(t, reserved, "error"); got != wantReserved {
		t.Errorf("error = %q, want %q", got, wantReserved)
	}
	ownSlot := call(t, router, http.MethodPost, "/api/cr/class-requests",
		classRequestBody("class", date, "08:00", "09:00", "1003", courseID, teacherID), firstToken)
	if ownSlot.Code != http.StatusCreated {
		t.Fatalf("own fixed slot status = %d, want 201 (body = %q)", ownSlot.Code, ownSlot.Body.String())
	}
	availability := call(t, router, http.MethodGet, "/api/cr/room-availability?date="+date, "", firstToken)
	if availability.Code != http.StatusOK {
		t.Fatalf("availability status = %d, want 200 (body = %q)", availability.Code, availability.Body.String())
	}
	bookings := listItems(t, availability)
	if len(bookings) != 3 {
		t.Fatalf("got %d bookings, want 3 (body = %q)", len(bookings), availability.Body.String())
	}
	wantBookings := []struct {
		room   string
		start  string
		source string
		status string
		label  string
	}{
		{room: "1003", start: "08:00", source: "fixed", status: "fixed", label: handler.LabelYours},
		{room: "1003", start: "08:00", source: "class", status: "pending", label: handler.LabelUnavailable},
		{room: "1004", start: "10:30", source: "extra", status: "pending", label: handler.LabelUnavailable},
	}
	for i, want := range wantBookings {
		booking := bookings[i]
		if booking["room"] != want.room || booking["start_time"] != want.start ||
			booking["source"] != want.source || booking["status"] != want.status || booking["label"] != want.label {
			t.Errorf("booking %d = %v, want %+v", i, booking, want)
		}
	}
	forSecondCR := call(t, router, http.MethodGet, "/api/cr/room-availability?date="+date+"&room=1003", "", secondToken)
	otherBookings := listItems(t, forSecondCR)
	if len(otherBookings) != 2 {
		t.Fatalf("got %d bookings for room 1003, want 2 (body = %q)", len(otherBookings), forSecondCR.Body.String())
	}
	for i, booking := range otherBookings {
		if booking["label"] != handler.LabelUnavailable {
			t.Errorf("booking %d label = %v, want %q for another batch", i, booking["label"], handler.LabelUnavailable)
		}
	}
	requests := call(t, router, http.MethodGet, "/api/cr/requests", "", firstToken)
	if requests.Code != http.StatusOK {
		t.Fatalf("requests status = %d, want 200 (body = %q)", requests.Code, requests.Body.String())
	}
	mine := listItems(t, requests)
	if len(mine) != 2 {
		t.Fatalf("got %d requests, want 2 (body = %q)", len(mine), requests.Body.String())
	}
	if mine[0]["room"] != "1003" || mine[1]["room"] != "1004" {
		t.Errorf("requests = %v, want the newest one first", mine)
	}
	if mine[0]["course_code"] != "CSE233" || mine[0]["teacher_name"] != "Request Teacher" {
		t.Errorf("the first request hides the course or teacher: %v", mine[0])
	}
	pending := call(t, router, http.MethodGet, "/api/cr/requests?status=approved", "", firstToken)
	if entries := listItems(t, pending); len(entries) != 0 {
		t.Errorf("approved requests = %v, want none", entries)
	}
	secondRequests := call(t, router, http.MethodGet, "/api/cr/requests", "", secondToken)
	if entries := listItems(t, secondRequests); len(entries) != 0 {
		t.Errorf("the other class representative sees %v, want nothing", entries)
	}
	teacherToken := loginToken(t, router, "T-800")
	notifications := call(t, router, http.MethodGet, "/api/notifications", "", teacherToken)
	if notifications.Code != http.StatusOK {
		t.Fatalf("notifications status = %d, want 200 (body = %q)", notifications.Code, notifications.Body.String())
	}
	alerts := listItems(t, notifications)
	if len(alerts) != 2 {
		t.Fatalf("the teacher has %d notifications, want 2 (body = %q)", len(alerts), notifications.Body.String())
	}
	for _, alert := range alerts {
		if alert["type"] != handler.NotificationTypeClassRequest {
			t.Errorf("notification type = %v, want %q", alert["type"], handler.NotificationTypeClassRequest)
		}
	}
}
