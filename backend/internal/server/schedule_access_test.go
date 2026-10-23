package server

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"net/http"
	"testing"
	"time"
)

type emptyTeacherStore struct{}

func (emptyTeacherStore) GetTeacherByUser(ctx context.Context, userID int64) (store.Teacher, error) {
	return store.Teacher{ID: 7, UserID: userID, Name: "Active Teacher"}, nil
}

func (emptyTeacherStore) ListTeacherRoutine(ctx context.Context, teacherID int64) ([]store.RoutineRow, error) {
	return nil, nil
}

func (emptyTeacherStore) ListTeacherDayItems(ctx context.Context, teacherID int64,
	date string) ([]store.ScheduleItem, error) {
	return nil, nil
}

func (emptyTeacherStore) ListTeacherFinalItems(ctx context.Context, teacherID int64,
	from string) ([]store.ScheduleItem, error) {
	return nil, nil
}

func (emptyTeacherStore) ListTeacherExams(ctx context.Context, teacherID int64,
	from string) ([]store.ScheduleItem, error) {
	return nil, nil
}

func (emptyTeacherStore) ConfirmItem(ctx context.Context, itemID, teacherID int64, mode, room, link string,
	now time.Time, windowOpen bool) (store.ScheduleItem, error) {
	return store.ScheduleItem{}, store.ErrItemNotFound
}

func (emptyTeacherStore) CancelItemAsTeacher(ctx context.Context, itemID, teacherID int64,
	reason string, windowOpen bool) (store.ScheduleItem, error) {
	return store.ScheduleItem{}, store.ErrItemNotFound
}

func (emptyTeacherStore) TeacherRequestReschedule(ctx context.Context, itemID, teacherID int64,
	message string, windowOpen bool) (store.ScheduleItem, error) {
	return store.ScheduleItem{}, store.ErrItemNotFound
}

func (emptyTeacherStore) AcceptProposal(ctx context.Context, itemID, teacherID int64) (store.ScheduleItem, error) {
	return store.ScheduleItem{}, store.ErrItemNotFound
}

func (emptyTeacherStore) DeclineProposal(ctx context.Context, itemID, teacherID int64) (store.ScheduleItem, error) {
	return store.ScheduleItem{}, store.ErrItemNotFound
}

func (emptyTeacherStore) CreateTeacherExam(ctx context.Context,
	exam store.NewScheduleItem) (store.ScheduleItem, error) {
	return store.ScheduleItem{}, store.ErrCourseNotFound
}

func (emptyTeacherStore) UpdateTeacherExam(ctx context.Context, itemID, teacherID int64,
	patch store.ItemPatch) (store.ScheduleItem, error) {
	return store.ScheduleItem{}, store.ErrItemNotFound
}

func (emptyTeacherStore) CancelTeacherExam(ctx context.Context, itemID, teacherID int64,
	reason string) (store.ScheduleItem, error) {
	return store.ScheduleItem{}, store.ErrItemNotFound
}

func (emptyTeacherStore) DeleteTeacherExam(ctx context.Context, itemID, teacherID int64) (store.ScheduleItem, error) {
	return store.ScheduleItem{}, store.ErrItemNotFound
}

func (emptyTeacherStore) ListActiveCRs(ctx context.Context, batch, section string) ([]int64, error) {
	return nil, nil
}

type emptyStudentStore struct{}

func (emptyStudentStore) GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error) {
	return store.StudentProfile{UserID: userID, Batch: "CSE-23", Section: "A"}, nil
}

func (emptyStudentStore) ListStudentDropCourses(ctx context.Context, studentID int64) ([]store.Course, error) {
	return nil, nil
}

func (emptyStudentStore) ListPublishedDay(ctx context.Context, batch, date string,
	dropCourseCodes []string) ([]store.PlanItem, error) {
	return nil, nil
}

func (emptyStudentStore) ListStudentExams(ctx context.Context, batch, section string,
	now time.Time) ([]store.StudentExam, error) {
	return nil, nil
}

func (emptyStudentStore) RunningSemester(ctx context.Context, batch string) (string, error) {
	return "1-1", nil
}

func (emptyStudentStore) RunningSemesterSince(ctx context.Context,
	batch string) (string, time.Time, error) {
	return "1-1", time.Time{}, nil
}

func (emptyStudentStore) ListRunningCourseSlots(ctx context.Context, batch string) ([]store.RunningCourseSlot, error) {
	return nil, nil
}

func (emptyStudentStore) ListFilteredRoutines(ctx context.Context, filter store.RoutineFilter) ([]store.RoutineRow, error) {
	return nil, nil
}

func (emptyStudentStore) ListStudentNotices(ctx context.Context, batch, section, source, scope string) ([]store.NoticeRow, error) {
	return nil, nil
}

func (emptyStudentStore) ListStudentPolls(ctx context.Context, viewerUserID int64, batch, section string) ([]store.PollRow, error) {
	return nil, nil
}

func (emptyStudentStore) GetMessageViewer(ctx context.Context, userID int64) (store.MessageViewer, error) {
	return store.MessageViewer{UserID: userID, Role: "student", Batch: "CSE-23", Section: "A"}, nil
}

func (emptyStudentStore) CountUnreadMessages(ctx context.Context, viewer store.MessageViewer) (int, error) {
	return 0, nil
}

type silentAlerts struct{}

func (silentAlerts) NotifyUser(ctx context.Context, userID int64, kind, title, body string,
	link store.NotificationLink) error {
	return nil
}

func (silentAlerts) NotifyBroadcast(ctx context.Context, role, batch, section, kind, title, body string,
	link store.NotificationLink) error {
	return nil
}

type scheduleRoute struct {
	method     string
	path       string
	body       string
	wantForOwn int
}

func accessDate(t *testing.T) string {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load the timezone: %v", err)
	}
	return time.Now().In(loc).AddDate(0, 0, 1).Format(validate.DateLayout)
}

func teacherRoutes(t *testing.T) []scheduleRoute {
	t.Helper()
	date := accessDate(t)
	return []scheduleRoute{
		{method: http.MethodGet, path: "/api/teacher/fixed-routine", wantForOwn: http.StatusOK},
		{method: http.MethodGet, path: "/api/teacher/upcoming/next-day", wantForOwn: http.StatusOK},
		{method: http.MethodGet, path: "/api/teacher/upcoming/final", wantForOwn: http.StatusOK},
		{method: http.MethodGet, path: "/api/teacher/exams", wantForOwn: http.StatusOK},
		{
			method: http.MethodPost, path: "/api/teacher/items/1/confirm",
			body: `{"mode":"room","room":"1003"}`, wantForOwn: http.StatusNotFound,
		},
		{method: http.MethodPost, path: "/api/teacher/items/1/cancel", body: `{}`, wantForOwn: http.StatusNotFound},
		{
			method: http.MethodPost, path: "/api/teacher/items/1/request-reschedule",
			body: `{"message":"I am away that day."}`, wantForOwn: http.StatusNotFound,
		},
		{
			method: http.MethodPost, path: "/api/teacher/items/1/reschedule-request/accept",
			wantForOwn: http.StatusNotFound,
		},
		{
			method: http.MethodPost, path: "/api/teacher/items/1/reschedule-request/decline",
			wantForOwn: http.StatusNotFound,
		},
		{
			method: http.MethodPost, path: "/api/teacher/exams",
			body:       `{"course_id":1,"batch":"CSE-23","item_date":"` + date + `"}`,
			wantForOwn: http.StatusBadRequest,
		},
		{method: http.MethodPatch, path: "/api/teacher/exams/1", body: `{"syllabus":"All"}`, wantForOwn: http.StatusNotFound},
		{method: http.MethodPost, path: "/api/teacher/exams/1/cancel", body: `{}`, wantForOwn: http.StatusNotFound},
		{method: http.MethodDelete, path: "/api/teacher/exams/1", wantForOwn: http.StatusNotFound},
	}
}

func studentRoutes() []scheduleRoute {
	return []scheduleRoute{
		{method: http.MethodGet, path: "/api/students/today", wantForOwn: http.StatusOK},
		{method: http.MethodGet, path: "/api/students/upcoming-routine", wantForOwn: http.StatusOK},
		{method: http.MethodGet, path: "/api/students/exams", wantForOwn: http.StatusOK},
		{method: http.MethodGet, path: "/api/students/routine", wantForOwn: http.StatusOK},
	}
}

func TestTeacherRoutesNeedATeacherToken(t *testing.T) {
	deps, router := newAccessRouter(t)
	callers := []struct {
		name       string
		token      string
		wantStatus int
		wantError  string
	}{
		{name: "without a token", wantStatus: http.StatusUnauthorized, wantError: middleware.MessageUnauthorized},
		{
			name:       "with a student token",
			token:      issue(t, deps, accessStudentID, auth.RoleStudent),
			wantStatus: http.StatusForbidden,
			wantError:  middleware.MessageAccessDenied,
		},
		{
			name:       "with a class representative token",
			token:      issue(t, deps, accessCRID, auth.RoleCR),
			wantStatus: http.StatusForbidden,
			wantError:  middleware.MessageAccessDenied,
		},
		{
			name:       "with an admin token",
			token:      issue(t, deps, accessAdminID, auth.RoleAdmin),
			wantStatus: http.StatusForbidden,
			wantError:  middleware.MessageAccessDenied,
		},
	}
	teacherToken := issue(t, deps, accessTeacherID, auth.RoleTeacher)
	for _, route := range teacherRoutes(t) {
		for _, caller := range callers {
			t.Run(route.method+" "+route.path+" "+caller.name, func(t *testing.T) {
				rec := serve(t, router, route.method, route.path, route.body, caller.token)
				if rec.Code != caller.wantStatus {
					t.Fatalf("status = %d, want %d (body = %q)", rec.Code, caller.wantStatus, rec.Body.String())
				}
				if got := errorField(t, rec); got != caller.wantError {
					t.Errorf("error = %q, want %q", got, caller.wantError)
				}
			})
		}
		t.Run(route.method+" "+route.path+" with a teacher token", func(t *testing.T) {
			rec := serve(t, router, route.method, route.path, route.body, teacherToken)
			if rec.Code != route.wantForOwn {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, route.wantForOwn, rec.Body.String())
			}
		})
	}
}

func TestStudentRoutesAllowStudentsAndCRs(t *testing.T) {
	deps, router := newAccessRouter(t)
	allowed := map[string]string{
		"student":              issue(t, deps, accessStudentID, auth.RoleStudent),
		"class representative": issue(t, deps, accessCRID, auth.RoleCR),
	}
	refused := map[string]string{
		"teacher": issue(t, deps, accessTeacherID, auth.RoleTeacher),
		"admin":   issue(t, deps, accessAdminID, auth.RoleAdmin),
	}
	for _, route := range studentRoutes() {
		t.Run(route.path+" without a token", func(t *testing.T) {
			rec := serve(t, router, route.method, route.path, route.body, "")
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401 (body = %q)", rec.Code, rec.Body.String())
			}
			if got := errorField(t, rec); got != middleware.MessageUnauthorized {
				t.Errorf("error = %q, want %q", got, middleware.MessageUnauthorized)
			}
		})
		for role, token := range allowed {
			t.Run(route.path+" as a "+role, func(t *testing.T) {
				rec := serve(t, router, route.method, route.path, route.body, token)
				if rec.Code != route.wantForOwn {
					t.Fatalf("status = %d, want %d (body = %q)", rec.Code, route.wantForOwn, rec.Body.String())
				}
			})
		}
		for role, token := range refused {
			t.Run(route.path+" as a "+role, func(t *testing.T) {
				rec := serve(t, router, route.method, route.path, route.body, token)
				if rec.Code != http.StatusForbidden {
					t.Fatalf("status = %d, want 403 (body = %q)", rec.Code, rec.Body.String())
				}
				if got := errorField(t, rec); got != middleware.MessageAccessDenied {
					t.Errorf("error = %q, want %q", got, middleware.MessageAccessDenied)
				}
			})
		}
	}
}
