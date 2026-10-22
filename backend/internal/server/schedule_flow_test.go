package server

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/jobs"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	engineCourse  = `{"code":"cse 2243 7637d","title":"Compilers","credit":3}`
	engineSecond  = `{"code":"CSE-2211","title":"Operating Systems","credit":3}`
	engineTeacher = `{"name":"Engine Teacher","user_id":"T-500","email":"engineteacher@sust.edu",
"phone":"01712345500"}`
	engineCR = `{"full_name":"Engine CR","reg_no":"2023331500","batch":"CSE-23","section":"A",
"email":"enginecr@student.sust.edu","phone":"01712345501","blood_group":"O+",
"password":"sust-250-project","confirm_password":"sust-250-project"}`
	engineStudent = `{"full_name":"Engine Student","reg_no":"2023331502","batch":"CSE-23","section":"A",
"email":"enginestudent@student.sust.edu","phone":"01712345502","blood_group":"A+",
"password":"sust-250-project","confirm_password":"sust-250-project"}`
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

func generateItems(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	userStore := store.New(pool)
	scheduler := jobs.NewScheduler(flowConfig(t), userStore, notify.New(userStore))
	if _, err := scheduler.Generate(ctx, flowConfig(t).Now()); err != nil {
		t.Fatalf("Generate() returned an unexpected error: %v", err)
	}
}

func itemsOn(t *testing.T, router http.Handler, token, path string) []map[string]any {
	t.Helper()
	rec := call(t, router, http.MethodGet, path, "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200 (body = %q)", path, rec.Code, rec.Body.String())
	}
	return listItems(t, rec)
}

func itemOfCourse(t *testing.T, items []map[string]any, code string) map[string]any {
	t.Helper()
	for _, item := range items {
		if item["course_code"] == code {
			return item
		}
	}
	t.Fatalf("no item for the course %q in %v", code, items)
	return nil
}

func itemID(t *testing.T, item map[string]any) int64 {
	t.Helper()
	value, ok := item["id"].(float64)
	if !ok {
		t.Fatalf("the item has no id: %v", item)
	}
	return int64(value)
}

func addToFinal(t *testing.T, router http.Handler, crToken string, id int64) {
	t.Helper()
	path := "/api/cr/items/" + strconv.FormatInt(id, 10) + "/final"
	rec := call(t, router, http.MethodPost, path, "", crToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("add to final status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
}

func publishItems(t *testing.T, router http.Handler, crToken string, ids ...int64) *httptest.ResponseRecorder {
	t.Helper()
	texts := make([]string, 0, len(ids))
	for _, id := range ids {
		texts = append(texts, strconv.FormatInt(id, 10))
	}
	body := `{"item_ids":[` + strings.Join(texts, ",") + `]}`
	return call(t, router, http.MethodPost, "/api/cr/final/publish", body, crToken)
}

func seedEngine(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	router http.Handler) (string, string, string, int64, int64) {
	t.Helper()
	adminToken := seedAdmin(t, ctx, pool, router)
	first := call(t, router, http.MethodPost, "/api/admin/courses", engineCourse, adminToken)
	if first.Code != http.StatusCreated {
		t.Fatalf("create course status = %d, want 201 (body = %q)", first.Code, first.Body.String())
	}
	firstID := int64(number(t, first, "id"))
	second := call(t, router, http.MethodPost, "/api/admin/courses", engineSecond, adminToken)
	if second.Code != http.StatusCreated {
		t.Fatalf("create second course status = %d, want 201 (body = %q)", second.Code, second.Body.String())
	}
	secondID := int64(number(t, second, "id"))
	invited := teacherInviteCall(t, router, adminToken, engineTeacher)
	if invited.Code != http.StatusCreated {
		t.Fatalf("invite teacher status = %d, want 201 (body = %q)", invited.Code, invited.Body.String())
	}
	teacherRowID := lookupTeacherID(t, router, adminToken, int64(number(t, invited, "id")))
	tomorrow := flowConfig(t).Now().AddDate(0, 0, 1)
	weekday := int(tomorrow.Weekday())
	for _, routine := range []string{
		routineBody("A", firstID, teacherRowID, weekday, "09:00", "10:30", "1003", "class"),
		routineBody("A", secondID, teacherRowID, weekday, "10:30", "12:00", "1004", "class"),
	} {
		rec := call(t, router, http.MethodPost, "/api/admin/routines", routine, adminToken)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create routine status = %d, want 201 (body = %q)", rec.Code, rec.Body.String())
		}
	}
	onboardCR(t, router, adminToken, engineCR)
	onboardStudent(t, router, adminToken, engineStudent)
	return adminToken, loginToken(t, router, "T-500"), loginToken(t, router, "2023331500"), firstID, secondID
}

func TestScheduleEngineFullDay(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	_, teacherToken, crToken, _, _ := seedEngine(t, ctx, pool, router)
	studentToken := loginToken(t, router, "2023331502")
	generateItems(t, ctx, pool)
	tomorrow := validate.DateString(flowConfig(t).Now().AddDate(0, 0, 1))
	nextDay := call(t, router, http.MethodGet, "/api/teacher/upcoming/next-day", "", teacherToken)
	if nextDay.Code != http.StatusOK {
		t.Fatalf("next day status = %d, want 200 (body = %q)", nextDay.Code, nextDay.Body.String())
	}
	if got := field(t, nextDay, "date"); got != tomorrow {
		t.Errorf("next day date = %q, want %q", got, tomorrow)
	}
	if !boolean(t, nextDay, "decision_open") {
		t.Fatal("the decision window is closed in the test configuration")
	}
	items := listItems(t, nextDay)
	if len(items) != 2 {
		t.Fatalf("the teacher sees %d items, want 2 (body = %q)", len(items), nextDay.Body.String())
	}
	compilers := itemID(t, itemOfCourse(t, items, "CSE 2243 7637D"))
	systems := itemID(t, itemOfCourse(t, items, "CSE 2211"))
	confirmed := call(t, router, http.MethodPost,
		"/api/teacher/items/"+strconv.FormatInt(compilers, 10)+"/confirm",
		`{"mode":"online","meeting_link":"https://meet.example.com/compilers"}`, teacherToken)
	if confirmed.Code != http.StatusOK {
		t.Fatalf("confirm status = %d, want 200 (body = %q)", confirmed.Code, confirmed.Body.String())
	}
	cancelled := call(t, router, http.MethodPost,
		"/api/teacher/items/"+strconv.FormatInt(systems, 10)+"/cancel",
		`{"reason":"I am at a conference."}`, teacherToken)
	if cancelled.Code != http.StatusOK {
		t.Fatalf("cancel status = %d, want 200 (body = %q)", cancelled.Code, cancelled.Body.String())
	}
	upcoming := itemsOn(t, router, crToken, "/api/cr/upcoming")
	if len(upcoming) != 2 {
		t.Fatalf("the class representative sees %d items, want 2", len(upcoming))
	}
	addToFinal(t, router, crToken, compilers)
	published := publishItems(t, router, crToken, compilers)
	if published.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want 200 (body = %q)", published.Code, published.Body.String())
	}
	routine := call(t, router, http.MethodGet, "/api/students/upcoming-routine?date="+tomorrow, "", studentToken)
	if routine.Code != http.StatusOK {
		t.Fatalf("student routine status = %d, want 200 (body = %q)", routine.Code, routine.Body.String())
	}
	groups := groupsOf(t, routine)
	if len(groups) != 1 {
		t.Fatalf("the student sees %d groups, want only section A (body = %q)", len(groups), routine.Body.String())
	}
	studentItems := groupItems(t, groups[0])
	if len(studentItems) != 1 {
		t.Fatalf("the student sees %d items, want only the published one", len(studentItems))
	}
	if studentItems[0]["course_code"] != "CSE 2243 7637D" {
		t.Errorf("the student sees %v, want the published class", studentItems[0])
	}
	if studentItems[0]["mode"] != "online" {
		t.Errorf("mode = %v, want online", studentItems[0]["mode"])
	}
	if studentItems[0]["meeting_link"] != "https://meet.example.com/compilers" {
		t.Errorf("meeting_link = %v, want the online link", studentItems[0]["meeting_link"])
	}
}

func TestScheduleEngineRescheduleAndCancellationReachStudents(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	_, teacherToken, crToken, _, _ := seedEngine(t, ctx, pool, router)
	studentToken := loginToken(t, router, "2023331502")
	generateItems(t, ctx, pool)
	tomorrow := validate.DateString(flowConfig(t).Now().AddDate(0, 0, 1))
	items := itemsOn(t, router, teacherToken, "/api/teacher/upcoming/next-day")
	compilers := itemID(t, itemOfCourse(t, items, "CSE 2243 7637D"))
	asked := call(t, router, http.MethodPost,
		"/api/teacher/items/"+strconv.FormatInt(compilers, 10)+"/request-reschedule",
		`{"message":"Please move this class."}`, teacherToken)
	if asked.Code != http.StatusOK {
		t.Fatalf("request reschedule status = %d, want 200 (body = %q)", asked.Code, asked.Body.String())
	}
	moved := call(t, router, http.MethodPatch, "/api/cr/items/"+strconv.FormatInt(compilers, 10),
		`{"start_time":"14:00","end_time":"15:30","mode":"room","room":"1003"}`, crToken)
	if moved.Code != http.StatusOK {
		t.Fatalf("patch status = %d, want 200 (body = %q)", moved.Code, moved.Body.String())
	}
	addToFinal(t, router, crToken, compilers)
	if rec := publishItems(t, router, crToken, compilers); rec.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	published := groupItems(t, groupsOf(t, call(t, router, http.MethodGet,
		"/api/students/upcoming-routine?date="+tomorrow, "", studentToken))[0])
	if len(published) != 1 || published[0]["start_time"] != "14:00" {
		t.Fatalf("the student sees %v, want the moved class at 14:00", published)
	}
	if published[0]["status"] != store.ItemRescheduled {
		t.Errorf("status = %v, want %q", published[0]["status"], store.ItemRescheduled)
	}
	later := call(t, router, http.MethodPost,
		"/api/teacher/items/"+strconv.FormatInt(compilers, 10)+"/cancel", `{}`, teacherToken)
	if later.Code != http.StatusOK {
		t.Fatalf("cancel status = %d, want 200 (body = %q)", later.Code, later.Body.String())
	}
	afterCancel := groupItems(t, groupsOf(t, call(t, router, http.MethodGet,
		"/api/students/upcoming-routine?date="+tomorrow, "", studentToken))[0])
	if len(afterCancel) != 1 || afterCancel[0]["status"] != store.ItemCancelled {
		t.Fatalf("the student sees %v, want the cancellation straight away", afterCancel)
	}
}

func TestScheduleEngineSnapshotHidesLaterEdits(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	_, teacherToken, crToken, _, _ := seedEngine(t, ctx, pool, router)
	studentToken := loginToken(t, router, "2023331502")
	generateItems(t, ctx, pool)
	tomorrow := validate.DateString(flowConfig(t).Now().AddDate(0, 0, 1))
	items := itemsOn(t, router, teacherToken, "/api/teacher/upcoming/next-day")
	compilers := itemID(t, itemOfCourse(t, items, "CSE 2243 7637D"))
	confirm := call(t, router, http.MethodPost,
		"/api/teacher/items/"+strconv.FormatInt(compilers, 10)+"/confirm",
		`{"mode":"room","room":"1003"}`, teacherToken)
	if confirm.Code != http.StatusOK {
		t.Fatalf("confirm status = %d, want 200 (body = %q)", confirm.Code, confirm.Body.String())
	}
	addToFinal(t, router, crToken, compilers)
	if rec := publishItems(t, router, crToken, compilers); rec.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	changed := call(t, router, http.MethodPost,
		"/api/teacher/items/"+strconv.FormatInt(compilers, 10)+"/confirm",
		`{"mode":"room","room":"2104"}`, teacherToken)
	if changed.Code != http.StatusOK {
		t.Fatalf("second confirm status = %d, want 200 (body = %q)", changed.Code, changed.Body.String())
	}
	studentItems := groupItems(t, groupsOf(t, call(t, router, http.MethodGet,
		"/api/students/upcoming-routine?date="+tomorrow, "", studentToken))[0])
	if len(studentItems) != 1 || studentItems[0]["room"] != "1003" {
		t.Fatalf("the student sees %v, want the published room 1003", studentItems)
	}
	final := call(t, router, http.MethodGet, "/api/cr/final", "", crToken)
	if final.Code != http.StatusOK {
		t.Fatalf("final status = %d, want 200 (body = %q)", final.Code, final.Body.String())
	}
	if got := number(t, final, "changed_since_publish"); got != 1 {
		t.Errorf("changed_since_publish = %v, want 1 (body = %q)", got, final.Body.String())
	}
	if rec := publishItems(t, router, crToken, compilers); rec.Code != http.StatusOK {
		t.Fatalf("republish status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	after := groupItems(t, groupsOf(t, call(t, router, http.MethodGet,
		"/api/students/upcoming-routine?date="+tomorrow, "", studentToken))[0])
	if len(after) != 1 || after[0]["room"] != "2104" {
		t.Fatalf("the student sees %v, want room 2104 after the republish", after)
	}
}

func TestScheduleEngineExamFlow(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	_, teacherToken, crToken, courseID, _ := seedEngine(t, ctx, pool, router)
	studentToken := loginToken(t, router, "2023331502")
	tomorrow := validate.DateString(flowConfig(t).Now().AddDate(0, 0, 1))
	body := `{"course_id":` + strconv.FormatInt(courseID, 10) +
		`,"batch":"CSE-23","section":"A","item_date":"` + tomorrow +
		`","syllabus":"Chapters 1 to 4","instructions":"Bring a calculator."}`
	created := call(t, router, http.MethodPost, "/api/teacher/exams", body, teacherToken)
	if created.Code != http.StatusCreated {
		t.Fatalf("create exam status = %d, want 201 (body = %q)", created.Code, created.Body.String())
	}
	var payload struct {
		Item map[string]any `json:"item"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	examID := itemID(t, payload.Item)
	if payload.Item["needs_time"] != true {
		t.Errorf("needs_time = %v, want true for a fresh teacher exam", payload.Item["needs_time"])
	}
	exams := itemsOn(t, router, studentToken, "/api/students/exams")
	if len(exams) != 1 {
		t.Fatalf("the student sees %d exams, want the teacher exam straight away", len(exams))
	}
	if exams[0]["syllabus"] != "Chapters 1 to 4" || exams[0]["start_time"] != nil {
		t.Errorf("exam = %v, want the syllabus without a time", exams[0])
	}
	timed := call(t, router, http.MethodPatch, "/api/cr/items/"+strconv.FormatInt(examID, 10),
		`{"start_time":"09:00","end_time":"11:00","mode":"room","room":"1003"}`, crToken)
	if timed.Code != http.StatusOK {
		t.Fatalf("set the exam time status = %d, want 200 (body = %q)", timed.Code, timed.Body.String())
	}
	addToFinal(t, router, crToken, examID)
	if rec := publishItems(t, router, crToken, examID); rec.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	afterPublish := itemsOn(t, router, studentToken, "/api/students/exams")
	if len(afterPublish) != 1 || afterPublish[0]["start_time"] != "09:00" {
		t.Fatalf("the student sees %v, want the published exam time", afterPublish)
	}
	if afterPublish[0]["room"] != "1003" {
		t.Errorf("room = %v, want 1003", afterPublish[0]["room"])
	}
}

func TestScheduleEngineRoomAndBatchRules(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	_, teacherToken, crToken, courseID, _ := seedEngine(t, ctx, pool, router)
	generateItems(t, ctx, pool)
	tomorrow := validate.DateString(flowConfig(t).Now().AddDate(0, 0, 1))
	items := itemsOn(t, router, teacherToken, "/api/teacher/upcoming/next-day")
	if len(items) != 2 {
		t.Fatalf("the teacher sees %d items, want 2", len(items))
	}
	clash := `{"kind":"class","course_id":` + strconv.FormatInt(courseID, 10) +
		`,"section":"A","item_date":"` + tomorrow +
		`","start_time":"09:45","end_time":"10:30","mode":"room","room":"2104"}`
	rec := call(t, router, http.MethodPost, "/api/cr/items", clash, crToken)
	if rec.Code != http.StatusConflict {
		t.Fatalf("batch clash status = %d, want 409 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := field(t, rec, "error"); !strings.Contains(got, "CSE-23 section A already has") {
		t.Errorf("error = %q, want the batch clash sentence", got)
	}
	roomClash := `{"kind":"class","course_id":` + strconv.FormatInt(courseID, 10) +
		`,"section":"B","item_date":"` + tomorrow +
		`","start_time":"09:45","end_time":"10:30","mode":"room","room":"1003"}`
	roomRec := call(t, router, http.MethodPost, "/api/cr/items", roomClash, crToken)
	if roomRec.Code != http.StatusConflict {
		t.Fatalf("room clash status = %d, want 409 (body = %q)", roomRec.Code, roomRec.Body.String())
	}
	if got := field(t, roomRec, "error"); !strings.Contains(got, "Room 1003 is already booked") {
		t.Errorf("error = %q, want the room clash sentence", got)
	}
	unmanaged := `{"kind":"class","course_id":` + strconv.FormatInt(courseID, 10) +
		`,"section":"B","item_date":"` + tomorrow +
		`","start_time":"09:45","end_time":"10:30","mode":"room","room":"Library Hall"}`
	free := call(t, router, http.MethodPost, "/api/cr/items", unmanaged, crToken)
	if free.Code != http.StatusCreated {
		t.Fatalf("unmanaged room status = %d, want 201 (body = %q)", free.Code, free.Body.String())
	}
	online := `{"kind":"class","course_id":` + strconv.FormatInt(courseID, 10) +
		`,"section":"B","item_date":"` + tomorrow +
		`","start_time":"10:30","end_time":"11:15","mode":"online","meeting_link":"https://meet.example.com/extra"}`
	onlineRec := call(t, router, http.MethodPost, "/api/cr/items", online, crToken)
	if onlineRec.Code != http.StatusCreated {
		t.Fatalf("online class status = %d, want 201 (body = %q)", onlineRec.Code, onlineRec.Body.String())
	}
}

func TestScheduleEngineDecisionDeadline(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	seedEngine(t, ctx, pool, router)
	userStore := store.New(pool)
	closedConfig := flowConfig(t)
	closedConfig.DecisionFrom = 17 * 60
	closedConfig.DecisionTo = 23 * 60
	scheduler := jobs.NewScheduler(closedConfig, userStore, notify.New(userStore))
	now := closedConfig.Now()
	if _, err := scheduler.Generate(ctx, now); err != nil {
		t.Fatalf("Generate() returned an unexpected error: %v", err)
	}
	again, err := scheduler.Generate(ctx, now)
	if err != nil {
		t.Fatalf("second Generate() returned an unexpected error: %v", err)
	}
	if again != 0 {
		t.Errorf("the second generation created %d items, want none", again)
	}
	deadline := time.Date(now.Year(), now.Month(), now.Day(), 23, 30, 0, 0, now.Location())
	closed, err := scheduler.CloseDecisions(ctx, deadline)
	if err != nil {
		t.Fatalf("CloseDecisions() returned an unexpected error: %v", err)
	}
	if closed != 2 {
		t.Fatalf("CloseDecisions() closed %d items, want 2", closed)
	}
	repeat, err := scheduler.CloseDecisions(ctx, deadline)
	if err != nil || repeat != 0 {
		t.Errorf("the second CloseDecisions() closed %d items (err = %v), want none", repeat, err)
	}
	early := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
	if count, err := scheduler.CloseDecisions(ctx, early); err != nil || count != 0 {
		t.Errorf("CloseDecisions() inside the window closed %d items (err = %v), want none", count, err)
	}
	tomorrow := validate.DateString(now.AddDate(0, 0, 1))
	crToken := loginToken(t, router, "2023331500")
	items := itemsOn(t, router, crToken, "/api/cr/upcoming?from="+tomorrow)
	for _, item := range items {
		if item["status"] != store.ItemNoResponse {
			t.Errorf("status = %v, want %q", item["status"], store.ItemNoResponse)
		}
	}
	notifications := itemsOn(t, router, crToken, "/api/notifications")
	found := false
	for _, entry := range notifications {
		if entry["title"] == jobs.TitleNoResponse {
			found = true
			link, _ := entry["link"].(map[string]any)
			if link == nil || link["kind"] != notify.LinkCRUpcoming {
				t.Errorf("link = %v, want the class representative upcoming page", link)
			}
		}
	}
	if !found {
		t.Error("the class representative got no notification about the missing answer")
	}
}

func TestScheduleEngineClosedWindowRefusesTheFirstDecision(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	seedEngine(t, ctx, pool, router)
	generateItems(t, ctx, pool)
	userStore := store.New(pool)
	teacher, err := userStore.GetTeacherByUser(ctx, teacherUserOf(t, ctx, pool, "T-500"))
	if err != nil {
		t.Fatalf("GetTeacherByUser() returned an unexpected error: %v", err)
	}
	tomorrow := validate.DateString(flowConfig(t).Now().AddDate(0, 0, 1))
	items, err := userStore.ListTeacherDayItems(ctx, teacher.ID, tomorrow)
	if err != nil {
		t.Fatalf("ListTeacherDayItems() returned an unexpected error: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("the teacher has no generated items")
	}
	_, err = userStore.ConfirmItem(ctx, items[0].ID, teacher.ID, store.ModeRoom, "1003", "",
		flowConfig(t).Now(), false)
	if err != store.ErrDecisionClosed {
		t.Errorf("ConfirmItem() outside the window error = %v, want ErrDecisionClosed", err)
	}
	_ = router
}

func teacherUserOf(t *testing.T, ctx context.Context, pool *pgxpool.Pool, identifier string) int64 {
	t.Helper()
	var userID int64
	query := "SELECT id FROM users WHERE lower(identifier) = lower($1)"
	if err := pool.QueryRow(ctx, query, identifier).Scan(&userID); err != nil {
		t.Fatalf("could not find the user %q: %v", identifier, err)
	}
	return userID
}

func TestScheduleEnginePublishNeedsACompleteItem(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	_, teacherToken, crToken, courseID, _ := seedEngine(t, ctx, pool, router)
	tomorrow := validate.DateString(flowConfig(t).Now().AddDate(0, 0, 1))
	body := `{"course_id":` + strconv.FormatInt(courseID, 10) +
		`,"batch":"CSE-23","section":"A","item_date":"` + tomorrow + `","syllabus":"Everything"}`
	created := call(t, router, http.MethodPost, "/api/teacher/exams", body, teacherToken)
	if created.Code != http.StatusCreated {
		t.Fatalf("create exam status = %d, want 201 (body = %q)", created.Code, created.Body.String())
	}
	var payload struct {
		Item map[string]any `json:"item"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	examID := itemID(t, payload.Item)
	addToFinal(t, router, crToken, examID)
	rec := publishItems(t, router, crToken, examID)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("publish without a time status = %d, want 400 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := field(t, rec, "error"); !strings.Contains(got, "still needs a date") {
		t.Errorf("error = %q, want the incomplete item sentence", got)
	}
	final := call(t, router, http.MethodGet, "/api/cr/final", "", crToken)
	if got := number(t, final, "needs_time"); got != 1 {
		t.Errorf("needs_time = %v, want 1", got)
	}
}

func TestScheduleEngineLinkUpdateReachesStudents(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	_, teacherToken, crToken, _, _ := seedEngine(t, ctx, pool, router)
	studentToken := loginToken(t, router, "2023331502")
	generateItems(t, ctx, pool)
	tomorrow := validate.DateString(flowConfig(t).Now().AddDate(0, 0, 1))
	items := itemsOn(t, router, teacherToken, "/api/teacher/upcoming/next-day")
	compilers := itemID(t, itemOfCourse(t, items, "CSE 2243 7637D"))
	confirm := call(t, router, http.MethodPost,
		"/api/teacher/items/"+strconv.FormatInt(compilers, 10)+"/confirm",
		`{"mode":"online","meeting_link":"https://meet.example.com/first"}`, teacherToken)
	if confirm.Code != http.StatusOK {
		t.Fatalf("confirm status = %d, want 200 (body = %q)", confirm.Code, confirm.Body.String())
	}
	addToFinal(t, router, crToken, compilers)
	if rec := publishItems(t, router, crToken, compilers); rec.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	path := "/api/schedule/items/" + strconv.FormatInt(compilers, 10) + "/link"
	updated := call(t, router, http.MethodPatch, path,
		`{"meeting_link":"https://meet.example.com/second"}`, crToken)
	if updated.Code != http.StatusOK {
		t.Fatalf("link update status = %d, want 200 (body = %q)", updated.Code, updated.Body.String())
	}
	studentItems := groupItems(t, groupsOf(t, call(t, router, http.MethodGet,
		"/api/students/upcoming-routine?date="+tomorrow, "", studentToken))[0])
	if len(studentItems) != 1 || studentItems[0]["meeting_link"] != "https://meet.example.com/second" {
		t.Fatalf("the student sees %v, want the new link", studentItems)
	}
	refused := call(t, router, http.MethodPatch, path,
		`{"meeting_link":"https://meet.example.com/third"}`, studentToken)
	if refused.Code != http.StatusNotFound {
		t.Errorf("a plain student changing the link status = %d, want 404", refused.Code)
	}
	_ = handler.MessageLinkChanged
}
