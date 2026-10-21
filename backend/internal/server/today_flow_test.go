package server

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raHI-79/250-project/backend/internal/db"
	"github.com/raHI-79/250-project/backend/internal/dbtest"
	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"net/http"
	"strconv"
	"testing"
	"time"
)

const (
	todayCourse  = `{"code":"cse233","title":"Digital Logic Design"}`
	todayDropped = `{"code":"cse211","title":"Data Structures"}`
	todayTeacher = `{"name":"Today Teacher","user_id":"T-600","email":"todayteacher@sust.edu",
"phone":"01712345600","password":"sust-250-project","confirm_password":"sust-250-project"}`
	todayStudent = `{"full_name":"Today Student","reg_no":"2023331600","batch":"CSE-23","section":"A",
"email":"todaystudent@student.sust.edu","phone":"01712345601","blood_group":"O+",
"password":"sust-250-project","confirm_password":"sust-250-project","has_drop_course":true}`
	todayCR = `{"full_name":"Today CR","reg_no":"2022331601","batch":"CSE-22","section":"A",
"email":"todaycr@student.sust.edu","phone":"01712345602","blood_group":"A+",
"password":"sust-250-project","confirm_password":"sust-250-project"}`
)

func lookupTeacherID(t *testing.T, router http.Handler, adminToken string, teacherUserID int64) int64 {
	t.Helper()
	for _, item := range listItems(t, call(t, router, http.MethodGet, "/api/lookups/teachers", "", adminToken)) {
		if int64(item["user_id"].(float64)) == teacherUserID {
			return int64(item["id"].(float64))
		}
	}
	t.Fatal("the new teacher is missing from the teacher lookup")
	return 0
}

func otherBatchRoutineBody(section string, courseID, teacherID int64, day int, start, end, room string) string {
	return `{"batch":"CSE-22","section":"` + section +
		`","course_id":` + strconv.FormatInt(courseID, 10) +
		`,"teacher_id":` + strconv.FormatInt(teacherID, 10) +
		`,"day_of_week":` + strconv.Itoa(day) +
		`,"start_time":"` + start + `","end_time":"` + end +
		`","room":"` + room + `","type":"class"}`
}

func TestTodayAndDropCourseRoutineFlow(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	adminToken := seedAdmin(t, ctx, pool, router)
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load the timezone: %v", err)
	}
	now := time.Now().In(loc)
	date := now.Format(validate.DateLayout)
	weekday := int(now.Weekday())
	course := call(t, router, http.MethodPost, "/api/admin/courses", todayCourse, adminToken)
	if course.Code != http.StatusCreated {
		t.Fatalf("create course status = %d, want 201 (body = %q)", course.Code, course.Body.String())
	}
	courseID := int64(number(t, course, "id"))
	dropped := call(t, router, http.MethodPost, "/api/admin/courses", todayDropped, adminToken)
	if dropped.Code != http.StatusCreated {
		t.Fatalf("create drop course status = %d, want 201 (body = %q)", dropped.Code, dropped.Body.String())
	}
	droppedID := int64(number(t, dropped, "id"))
	createdTeacher := teacherInviteCall(t, router, adminToken, todayTeacher)
	if createdTeacher.Code != http.StatusCreated {
		t.Fatalf("create teacher status = %d, want 201 (body = %q)", createdTeacher.Code, createdTeacher.Body.String())
	}
	teacherID := lookupTeacherID(t, router, adminToken, int64(number(t, createdTeacher, "id")))
	own := call(t, router, http.MethodPost, "/api/admin/routines",
		routineBody("A", courseID, teacherID, weekday, "08:00", "09:00", "1003", "class"), adminToken)
	if own.Code != http.StatusCreated {
		t.Fatalf("own routine status = %d, want 201 (body = %q)", own.Code, own.Body.String())
	}
	junior := call(t, router, http.MethodPost, "/api/admin/routines",
		otherBatchRoutineBody("A", droppedID, teacherID, weekday, "10:30", "11:30", "1004"), adminToken)
	if junior.Code != http.StatusCreated {
		t.Fatalf("junior routine status = %d, want 201 (body = %q)", junior.Code, junior.Body.String())
	}
	onboardStudent(t, router, adminToken, todayStudent, droppedID)
	onboardCR(t, router, adminToken, todayCR)
	studentToken := loginToken(t, router, "2023331600")
	crToken := loginToken(t, router, "2022331601")
	routine := call(t, router, http.MethodGet, "/api/students/upcoming-routine", "", studentToken)
	if routine.Code != http.StatusOK {
		t.Fatalf("upcoming routine status = %d, want 200 (body = %q)", routine.Code, routine.Body.String())
	}
	if got := field(t, routine, "date"); got != date {
		t.Errorf("date = %q, want today %q", got, date)
	}
	groups := groupsOf(t, routine)
	if len(groups) != 2 {
		t.Fatalf("groups = %v, want the own section and the drop courses", groups)
	}
	if groups[0]["key"] != "A" || groups[0]["section_label"] != "Section A" {
		t.Errorf("first group = %v, want section A", groups[0])
	}
	last := groups[1]
	if last["key"] != handler.GroupDropCourses || last["drop_course"] != true {
		t.Fatalf("last group = %v, want the drop course group", last)
	}
	droppedItems := groupItems(t, last)
	if len(droppedItems) != 1 || droppedItems[0]["batch"] != "CSE-22" || droppedItems[0]["course_code"] != "CSE211" {
		t.Errorf("the drop course items are %v, want the CSE-22 class", droppedItems)
	}
	summary := call(t, router, http.MethodGet, "/api/students/today", "", studentToken)
	if summary.Code != http.StatusOK {
		t.Fatalf("today status = %d, want 200 (body = %q)", summary.Code, summary.Body.String())
	}
	if got := number(t, summary, "classes_today"); got != 2 {
		t.Errorf("classes_today = %v, want the own class and the drop course one", got)
	}
	if got := number(t, summary, "upcoming_exams"); got != 0 {
		t.Errorf("upcoming_exams = %v, want none", got)
	}
	noDropCourse := groupsOf(t, call(t, router, http.MethodGet, "/api/students/upcoming-routine", "", crToken))
	for _, group := range noDropCourse {
		if group["key"] == handler.GroupDropCourses {
			t.Errorf("the class representative without a drop course sees %v", group)
		}
	}
}

func TestDriveLinkMigrationKeepsTheLatestMaterial(t *testing.T) {
	ctx, pool := dbtest.Pool(t)
	applyUpTo(t, ctx, pool, "0009_class_event_notes.sql")
	if _, err := pool.Exec(ctx, `INSERT INTO users (id, role, identifier, password_hash, status)
VALUES (901, 'student', '2023331901', 'hash', 'active')`); err != nil {
		t.Fatalf("could not seed a user: %v", err)
	}
	seed := []struct {
		batch string
		link  string
		day   string
	}{
		{batch: "CSE-23", link: "https://drive.google.com/old", day: "2026-01-01"},
		{batch: "CSE-23", link: "https://drive.google.com/new", day: "2026-02-01"},
		{batch: "CSE-22", link: "https://drive.google.com/other", day: "2026-01-15"},
		{batch: "ALL", link: "https://drive.google.com/everyone", day: "2026-03-01"},
	}
	for _, row := range seed {
		_, err := pool.Exec(ctx, `INSERT INTO materials (batch, title, link, created_by, created_at)
VALUES ($1, $2, $3, 901, $4::date)`, row.batch, row.batch+" material", row.link, row.day)
		if err != nil {
			t.Fatalf("could not seed a material: %v", err)
		}
	}
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate() returned an unexpected error: %v", err)
	}
	rows, err := pool.Query(ctx, "SELECT batch, title, link FROM batch_drive_links ORDER BY batch")
	if err != nil {
		t.Fatalf("could not read the migrated drive links: %v", err)
	}
	defer rows.Close()
	type link struct{ batch, title, address string }
	migrated := make([]link, 0)
	for rows.Next() {
		var row link
		if err := rows.Scan(&row.batch, &row.title, &row.address); err != nil {
			t.Fatalf("could not scan a migrated drive link: %v", err)
		}
		migrated = append(migrated, row)
	}
	want := []link{
		{batch: "CSE-22", title: "CSE-22", address: "https://drive.google.com/other"},
		{batch: "CSE-23", title: "CSE-23", address: "https://drive.google.com/new"},
	}
	if len(migrated) != len(want) {
		t.Fatalf("the migration kept %v, want %v", migrated, want)
	}
	for i, row := range want {
		if migrated[i] != row {
			t.Errorf("link %d = %v, want %v", i, migrated[i], row)
		}
	}
	var leftovers int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'materials'").Scan(&leftovers)
	if err != nil {
		t.Fatalf("could not look for the old table: %v", err)
	}
	if leftovers != 0 {
		t.Error("the materials table is still there")
	}
}

func applyUpTo(t *testing.T, ctx context.Context, pool *pgxpool.Pool, last string) {
	t.Helper()
	if err := db.MigrateUpTo(ctx, pool, last); err != nil {
		t.Fatalf("MigrateUpTo(%q) returned an unexpected error: %v", last, err)
	}
}
