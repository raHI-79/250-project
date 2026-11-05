package store

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
	"time"
)

type dashboardSeed struct {
	window   Window
	today    time.Time
	tomorrow time.Time
	adminID  int64
	courseID int64
}

func openSession(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	courseID, teacherID int64, batch, section string) int64 {
	t.Helper()
	var id int64
	query := `INSERT INTO attendance_sessions
(course_id, batch, section, teacher_id, ends_at, token_key, status)
VALUES ($1, $2, $3, $4, now() + INTERVAL '1 hour', 'token', 'open')
RETURNING id`
	if err := pool.QueryRow(ctx, query, courseID, batch, section, teacherID).Scan(&id); err != nil {
		t.Fatalf("could not open the attendance session: %v", err)
	}
	return id
}

func markStudent(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	sessionID, studentID int64, status string) {
	t.Helper()
	query := `INSERT INTO attendance_records (session_id, student_id, status, method)
VALUES ($1, $2, $3, 'manual')`
	if _, err := pool.Exec(ctx, query, sessionID, studentID, status); err != nil {
		t.Fatalf("could not mark the student: %v", err)
	}
}

func seedDashboard(t *testing.T, ctx context.Context, store *Store, pool *pgxpool.Pool) dashboardSeed {
	t.Helper()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	seed := dashboardSeed{
		today:    today,
		tomorrow: today.AddDate(0, 0, 1),
		window:   Window{From: today.AddDate(0, 0, -6), To: today.AddDate(0, 0, 1)},
	}
	seed.adminID = insertUser(t, ctx, pool, RoleAdmin, "office.one", StatusActive, 0)
	teacherUser := insertUser(t, ctx, pool, RoleTeacher, "t.islam", StatusActive, 0)
	teacherID := insertTeacherRow(t, ctx, pool, teacherUser, "T-301", "Ratul Islam")
	presentUser := insertUser(t, ctx, pool, RoleStudent, "2023331101", StatusActive, 0)
	absentUser := insertUser(t, ctx, pool, RoleStudent, "2023331102", StatusActive, 0)
	insertUser(t, ctx, pool, RoleStudent, "2023331103", StatusPending, 0)
	presentID := insertStudentRow(t, ctx, pool, presentUser, "2023331101", "CSE-23", "A")
	absentID := insertStudentRow(t, ctx, pool, absentUser, "2023331102", "CSE-23", "A")
	seed.courseID = insertCourseRow(t, ctx, pool, "CSE 133", "Discrete Mathematics")
	sessionID := openSession(t, ctx, pool, seed.courseID, teacherID, "CSE-23", "A")
	markStudent(t, ctx, pool, sessionID, presentID, "present")
	markStudent(t, ctx, pool, sessionID, absentID, "absent")
	items := []struct {
		status string
		day    string
	}{
		{status: "confirmed", day: "CURRENT_DATE"},
		{status: "cancelled", day: "CURRENT_DATE"},
		{status: "rescheduled", day: "CURRENT_DATE"},
	}
	for _, item := range items {
		query := `INSERT INTO schedule_items
(batch, section, course_id, teacher_id, kind, origin, item_date, start_time, end_time, mode, room, status)
VALUES ('CSE-23', 'A', $1, $2, 'class', 'cr', ` + item.day + `, '08:00', '09:00', 'room', '1003', $3)`
		if _, err := pool.Exec(ctx, query, seed.courseID, teacherID, item.status); err != nil {
			t.Fatalf("could not insert the %s schedule item: %v", item.status, err)
		}
	}
	waiting := `INSERT INTO schedule_items
(batch, section, course_id, teacher_id, kind, origin, item_date, start_time, end_time, mode, room, status)
VALUES ('CSE-23', 'A', $1, $2, 'class', 'cr', CURRENT_DATE + 1, '10:00', '11:00', 'room', '1003', 'awaiting_teacher')`
	if _, err := pool.Exec(ctx, waiting, seed.courseID, teacherID); err != nil {
		t.Fatalf("could not insert tomorrow's undecided class: %v", err)
	}
	notice := `INSERT INTO notices (source, title, created_by) VALUES ('admin', 'Holiday', $1)`
	if _, err := pool.Exec(ctx, notice, seed.adminID); err != nil {
		t.Fatalf("could not insert the notice: %v", err)
	}
	poll := `INSERT INTO polls (question, batch, section, scope, created_by)
VALUES ('Lab day?', 'CSE-23', 'ALL', 'batch', $1)`
	if _, err := pool.Exec(ctx, poll, presentUser); err != nil {
		t.Fatalf("could not insert the poll: %v", err)
	}
	var conversationID int64
	chat := `INSERT INTO conversations (kind, created_by) VALUES ('direct', $1) RETURNING id`
	if err := pool.QueryRow(ctx, chat, presentUser).Scan(&conversationID); err != nil {
		t.Fatalf("could not insert the conversation: %v", err)
	}
	message := `INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, 'Hello')`
	if _, err := pool.Exec(ctx, message, conversationID, presentUser); err != nil {
		t.Fatalf("could not insert the message: %v", err)
	}
	phone := `INSERT INTO device_change_requests (user_id, new_device_id, reason)
VALUES ($1, 'device-2', 'New phone')`
	if _, err := pool.Exec(ctx, phone, presentUser); err != nil {
		t.Fatalf("could not insert the phone change request: %v", err)
	}
	views := []PageView{
		{VisitorID: "v1", SessionID: "s1", Path: "/student/today", Client: ClientWeb, Device: "desktop"},
		{VisitorID: "v1", SessionID: "s1", Path: "/student/today", Client: ClientWeb, Device: "desktop"},
		{VisitorID: "v2", SessionID: "s2", Path: "/student/exams", Client: ClientAndroid, Device: "phone", AppVersion: "2.5.0"},
	}
	if err := store.RecordViews(ctx, views); err != nil {
		t.Fatalf("RecordViews() returned an unexpected error: %v", err)
	}
	events := []AuditEvent{
		{ActorID: presentUser, ActorRole: RoleStudent, Action: ActionSignIn, Client: ClientAndroid},
		{ActorID: seed.adminID, ActorRole: RoleAdmin, Action: ActionSignIn, Client: ClientWeb},
		{ActorID: teacherUser, ActorRole: RoleTeacher, Action: ActionDeleted, TargetType: "attendance", Client: ClientWeb},
		{ActorRole: "", Action: ActionSignInFailed, IPHash: "hashed-ip", Client: ClientWeb},
		{ActorRole: "", Action: ActionSignInFailed, IPHash: "hashed-ip", Client: ClientWeb},
	}
	for _, event := range events {
		if err := store.WriteEvent(ctx, event); err != nil {
			t.Fatalf("WriteEvent() returned an unexpected error: %v", err)
		}
	}
	for _, kind := range []string{KindEmail, KindPush, KindFCM} {
		if err := store.WriteSystemEvent(ctx, kind, true, "ok"); err != nil {
			t.Fatalf("WriteSystemEvent() returned an unexpected error: %v", err)
		}
	}
	if err := store.WriteSystemEvent(ctx, KindEmail, false, "failed"); err != nil {
		t.Fatalf("WriteSystemEvent() returned an unexpected error: %v", err)
	}
	if err := store.WriteSystemEvent(ctx, KindClient5x, false, "GET /api/boom"); err != nil {
		t.Fatalf("WriteSystemEvent() returned an unexpected error: %v", err)
	}
	if err := store.WriteSystemEvent(ctx, KindClient4x, false, "GET /api/nope"); err != nil {
		t.Fatalf("WriteSystemEvent() returned an unexpected error: %v", err)
	}
	return seed
}

func TestDashboardOverviewReadsTheSeededData(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	seed := seedDashboard(t, ctx, store, pool)
	totals, err := store.OverviewTotals(ctx, seed.window, seed.today)
	if err != nil {
		t.Fatalf("OverviewTotals() returned an unexpected error: %v", err)
	}
	if totals.Visitors != 2 || totals.Views != 3 {
		t.Errorf("visitors = %d and views = %d, want 2 and 3", totals.Visitors, totals.Views)
	}
	if totals.SignIns != 2 {
		t.Errorf("sign-ins = %d, want 2", totals.SignIns)
	}
	if totals.Pending != 1 {
		t.Errorf("pending approvals = %d, want 1", totals.Pending)
	}
	if totals.ActiveToday < 3 {
		t.Errorf("active users today = %d, want at least 3", totals.ActiveToday)
	}
	series, err := store.VisitorSeries(ctx, seed.window)
	if err != nil {
		t.Fatalf("VisitorSeries() returned an unexpected error: %v", err)
	}
	if len(series) != 7 {
		t.Fatalf("got %d points, want one per day of the window", len(series))
	}
	last := series[len(series)-1]
	if last.Visitors != 2 || last.SignIns != 2 {
		t.Errorf("today = %+v, want 2 visitors and 2 sign-ins", last)
	}
	roles, err := store.SignInsByRole(ctx, seed.window)
	if err != nil {
		t.Fatalf("SignInsByRole() returned an unexpected error: %v", err)
	}
	if len(roles) != 2 {
		t.Errorf("roles = %+v, want a student row and an admin row", roles)
	}
	clients, err := store.ClientSplit(ctx, seed.window)
	if err != nil {
		t.Fatalf("ClientSplit() returned an unexpected error: %v", err)
	}
	if len(clients) != 2 {
		t.Errorf("clients = %+v, want the website and the app", clients)
	}
	versions, err := store.AppVersions(ctx, seed.window)
	if err != nil {
		t.Fatalf("AppVersions() returned an unexpected error: %v", err)
	}
	if len(versions) != 1 || versions[0].Label != "2.5.0" {
		t.Errorf("versions = %+v, want one 2.5.0 row", versions)
	}
	devices, err := store.Devices(ctx, seed.window)
	if err != nil {
		t.Fatalf("Devices() returned an unexpected error: %v", err)
	}
	if len(devices) != 2 {
		t.Errorf("devices = %+v, want a desktop row and a phone row", devices)
	}
	pages, err := store.TopPages(ctx, seed.window)
	if err != nil {
		t.Fatalf("TopPages() returned an unexpected error: %v", err)
	}
	if len(pages) != 2 || pages[0].Path != "/student/today" || pages[0].Views != 2 {
		t.Errorf("pages = %+v, want today's page on top with 2 views", pages)
	}
}

func TestDashboardAcademicReadsTheSeededData(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	seed := seedDashboard(t, ctx, store, pool)
	averages, err := store.BatchAttendance(ctx, seed.window)
	if err != nil {
		t.Fatalf("BatchAttendance() returned an unexpected error: %v", err)
	}
	if len(averages) != 1 {
		t.Fatalf("averages = %+v, want one batch row", averages)
	}
	if averages[0].Batch != "CSE-23" || averages[0].Average < 49 || averages[0].Average > 51 {
		t.Errorf("average = %+v, want half of CSE-23 present", averages[0])
	}
	courses, err := store.CourseClasses(ctx, seed.window)
	if err != nil {
		t.Fatalf("CourseClasses() returned an unexpected error: %v", err)
	}
	if len(courses) != 1 {
		t.Fatalf("courses = %+v, want one course row", courses)
	}
	if courses[0].Held != 1 || courses[0].Cancelled != 1 || courses[0].Rescheduled != 1 {
		t.Errorf("course row = %+v, want one of each status", courses[0])
	}
	undecided, err := store.UndecidedTeachers(ctx, seed.tomorrow)
	if err != nil {
		t.Fatalf("UndecidedTeachers() returned an unexpected error: %v", err)
	}
	if len(undecided) != 1 || undecided[0].Total != 1 {
		t.Errorf("undecided = %+v, want one class for one teacher", undecided)
	}
	totals, err := store.AcademicTotals(ctx, seed.window, seed.tomorrow)
	if err != nil {
		t.Fatalf("AcademicTotals() returned an unexpected error: %v", err)
	}
	if totals.BelowHalf != 1 {
		t.Errorf("students below half = %d, want 1", totals.BelowHalf)
	}
	if totals.PhonePending != 1 {
		t.Errorf("pending phone requests = %d, want 1", totals.PhonePending)
	}
	if totals.Notices != 1 || totals.Polls != 1 || totals.Messages != 1 {
		t.Errorf("notices, polls and messages = %d, %d, %d, want one each",
			totals.Notices, totals.Polls, totals.Messages)
	}
	if totals.ClassesHeld != 1 || totals.Cancelled != 1 || totals.Rescheduled != 1 {
		t.Errorf("class counts = %d held, %d cancelled, %d rescheduled, want one each",
			totals.ClassesHeld, totals.Cancelled, totals.Rescheduled)
	}
	if totals.Undecided != 1 {
		t.Errorf("undecided tomorrow = %d, want 1", totals.Undecided)
	}
}

func TestDashboardSystemHealthReadsTheSeededData(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	seed := seedDashboard(t, ctx, store, pool)
	health, err := store.SystemTotals(ctx, seed.window)
	if err != nil {
		t.Fatalf("SystemTotals() returned an unexpected error: %v", err)
	}
	if health.DatabaseBytes <= 0 {
		t.Error("the database size is not positive")
	}
	if health.EmailsSent != 1 || health.EmailsFailed != 1 {
		t.Errorf("emails = %d sent and %d failed, want one each", health.EmailsSent, health.EmailsFailed)
	}
	if health.PushSent != 1 || health.FCMSent != 1 {
		t.Errorf("push = %d and fcm = %d, want one each", health.PushSent, health.FCMSent)
	}
	if health.Server5xx != 1 || health.Client4xx != 1 {
		t.Errorf("errors = %d 5xx and %d 4xx, want one each", health.Server5xx, health.Client4xx)
	}
	failures, err := store.LatestServerErrors(ctx)
	if err != nil {
		t.Fatalf("LatestServerErrors() returned an unexpected error: %v", err)
	}
	if len(failures) != 1 || failures[0].Path != "GET /api/boom" {
		t.Errorf("failures = %+v, want the seeded 5xx path", failures)
	}
	spike, err := store.FailedSignInSpike(ctx, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("FailedSignInSpike() returned an unexpected error: %v", err)
	}
	if spike != 2 {
		t.Errorf("spike = %d, want 2 failures from one address", spike)
	}
	deletions, err := store.SheetDeletions(ctx, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("SheetDeletions() returned an unexpected error: %v", err)
	}
	if deletions != 1 {
		t.Errorf("sheet deletions = %d, want 1", deletions)
	}
}
