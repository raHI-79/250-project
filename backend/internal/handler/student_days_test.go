package handler

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeDayStore struct {
	fakeCourseStore
	askedDays  []string
	askedExams []string
	exams      []store.StudentExam
}

func (f *fakeDayStore) ListPublishedDay(ctx context.Context, batch, date string,
	dropCourseCodes []string) ([]store.PlanItem, error) {
	f.askedDays = append(f.askedDays, date)
	return []store.PlanItem{{
		Batch:       batch,
		Section:     "A",
		StartTime:   "09:00",
		EndTime:     "09:45",
		CourseCode:  "CSE245",
		CourseTitle: "Databases",
		Kind:        "class",
		Mode:        store.ModeRoom,
		Room:        "2104",
		Status:      store.ItemConfirmed,
	}}, nil
}

func (f *fakeDayStore) ListStudentExamsOnDate(ctx context.Context, batch, section, date string,
	dropCourseCodes []string) ([]store.StudentExam, error) {
	f.askedExams = append(f.askedExams, date)
	return f.exams, nil
}

func studentDayHandler(t *testing.T, records *fakeDayStore) *Student {
	t.Helper()
	records.profile = store.StudentProfile{UserID: 11, StudentID: 7, Batch: "CSE-23", Section: "A"}
	return NewStudent(accountConfig(t, "test"), records)
}

func TestTheStudentRoutineOpensOnToday(t *testing.T) {
	silenceLog(t)
	records := &fakeDayStore{}
	student := studentDayHandler(t, records)
	request := httptest.NewRequest(http.MethodGet, "/api/students/upcoming-routine", nil)
	rec := serve(student.UpcomingRoutine(), withCurrentUser(request, 11, auth.RoleStudent))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	today := validate.DateString(accountConfig(t, "test").Now())
	if len(records.askedDays) != 1 || records.askedDays[0] != today {
		t.Fatalf("the store was asked for %v, want only today", records.askedDays)
	}
	if got := decodeBody(t, rec)["date"]; got != today {
		t.Errorf("date = %v, want today", got)
	}
}

func TestTheStudentRoutineReadsAnyDay(t *testing.T) {
	silenceLog(t)
	records := &fakeDayStore{}
	student := studentDayHandler(t, records)
	for _, day := range []string{"2020-03-04", "2030-03-04"} {
		request := httptest.NewRequest(http.MethodGet,
			"/api/students/upcoming-routine?date="+day, nil)
		rec := serve(student.UpcomingRoutine(), withCurrentUser(request, 11, auth.RoleStudent))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d for %s, want 200 (body = %q)", rec.Code, day, rec.Body.String())
		}
		if got := decodeBody(t, rec)["date"]; got != day {
			t.Errorf("date = %v, want %q", got, day)
		}
	}
}

func TestTheStudentExamsReadADay(t *testing.T) {
	silenceLog(t)
	records := &fakeDayStore{exams: []store.StudentExam{{
		ID:          4,
		ItemDate:    "2020-03-04",
		CourseCode:  "CSE245",
		CourseTitle: "Databases",
		Batch:       "CSE-23",
		Section:     "A",
		Origin:      store.OriginTeacherExam,
		Status:      store.ItemConfirmed,
		Published:   true,
	}}}
	student := studentDayHandler(t, records)
	request := httptest.NewRequest(http.MethodGet, "/api/students/exams?date=2020-03-04", nil)
	rec := serve(student.Exams(), withCurrentUser(request, 11, auth.RoleStudent))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	body := decodeBody(t, rec)
	if body["past"] != true || body["read_only"] != true {
		t.Errorf("a past day is not read only: %q", rec.Body.String())
	}
	items := dayItems(t, rec)
	if len(items) != 1 {
		t.Fatalf("items = %d, want the exam of that day", len(items))
	}
	if items[0]["final_status"] != StatusExamHeld {
		t.Errorf("final status = %v, want %q", items[0]["final_status"], StatusExamHeld)
	}
	if len(records.askedExams) != 1 || records.askedExams[0] != "2020-03-04" {
		t.Errorf("the store was asked for %v, want the one day", records.askedExams)
	}
}

func TestTheStudentExamsKeepTheWholeListWithoutADate(t *testing.T) {
	silenceLog(t)
	records := &fakeDayStore{}
	student := studentDayHandler(t, records)
	request := httptest.NewRequest(http.MethodGet, "/api/students/exams", nil)
	rec := serve(student.Exams(), withCurrentUser(request, 11, auth.RoleStudent))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if len(records.askedExams) != 0 {
		t.Errorf("the day query ran for %v, want the open list instead", records.askedExams)
	}
	if _, ok := decodeBody(t, rec)["date"]; ok {
		t.Error("the open list carries a date")
	}
}
