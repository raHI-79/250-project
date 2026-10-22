package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"net/http"
	"strings"
	"testing"
)

type fakeCatalogStore struct {
	search         string
	courses        []store.Course
	created        store.RoutineRow
	entries        []store.RoutineRow
	err            error
	createdCourse  []store.Course
	routines       []store.NewRoutine
	deleted        []int64
	batchFilter    string
	sectionFilter  string
	usage          store.CourseUsage
	usageErr       error
	deletedCourse  []int64
	periods        []store.Period
	semesterFilter string
	movedSemester  string
	running        map[string]string
	runningSet     map[string]string
}

func (f *fakeCatalogStore) UpdateRoutineSemester(ctx context.Context, routineID int64, semester string) error {
	if f.err != nil {
		return f.err
	}
	f.movedSemester = semester
	return nil
}

func (f *fakeCatalogStore) ListRunningSemesters(ctx context.Context) (map[string]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.running == nil {
		return map[string]string{"CSE-23": "2-1"}, nil
	}
	return f.running, nil
}

func (f *fakeCatalogStore) SetRunningSemester(ctx context.Context, batch, semester string) error {
	if f.err != nil {
		return f.err
	}
	if f.runningSet == nil {
		f.runningSet = make(map[string]string)
	}
	f.runningSet[batch] = semester
	return nil
}

func (f *fakeCatalogStore) CreateCourse(ctx context.Context, code, title string,
	credit float64) (store.Course, error) {
	if f.err != nil {
		return store.Course{}, f.err
	}
	course := store.Course{ID: int64(len(f.createdCourse) + 1), Code: code, Title: title, Credit: credit}
	f.createdCourse = append(f.createdCourse, course)
	return course, nil
}

func (f *fakeCatalogStore) ListPeriods(ctx context.Context) ([]store.Period, error) {
	if len(f.periods) > 0 {
		return f.periods, nil
	}
	return testPeriods(), nil
}

func (f *fakeCatalogStore) ListCourses(ctx context.Context) ([]store.Course, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.courses, nil
}

func (f *fakeCatalogStore) SearchCourses(ctx context.Context, search string) ([]store.Course, error) {
	f.search = search
	if f.err != nil {
		return nil, f.err
	}
	return f.courses, nil
}

func (f *fakeCatalogStore) CreateRoutine(ctx context.Context, routine store.NewRoutine) (store.RoutineRow, error) {
	f.routines = append(f.routines, routine)
	if f.err != nil {
		return store.RoutineRow{}, f.err
	}
	created := f.created
	created.Batch = routine.Batch
	created.Section = routine.Section
	created.DayOfWeek = routine.DayOfWeek
	created.StartTime = routine.StartTime
	created.EndTime = routine.EndTime
	created.Room = routine.Room
	created.Type = routine.Type
	return created, nil
}

func (f *fakeCatalogStore) ListRoutines(ctx context.Context, batch, section,
	semester string) ([]store.RoutineRow, error) {
	f.batchFilter, f.sectionFilter, f.semesterFilter = batch, section, semester
	if f.err != nil {
		return nil, f.err
	}
	return f.entries, nil
}

func (f *fakeCatalogStore) DeleteCourse(ctx context.Context, courseID int64) (store.CourseUsage, error) {
	if f.usageErr != nil {
		return f.usage, f.usageErr
	}
	f.deletedCourse = append(f.deletedCourse, courseID)
	return store.CourseUsage{}, nil
}

func (f *fakeCatalogStore) DeleteRoutine(ctx context.Context, routineID int64) error {
	if f.err != nil {
		return f.err
	}
	f.deleted = append(f.deleted, routineID)
	return nil
}

const validRoutine = `{"batch":"CSE-23","semester":"2-1","section":"A","course_id":4,"teacher_id":6,
"day_of_week":1,"start_time":"10:30","end_time":"12:45","room":"1003","type":"lab"}`

func TestCreateCourse(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		body       string
		storeErr   error
		wantStatus int
		wantError  string
		wantCode   string
	}{
		{name: "created", body: `{"code":"cse-233","title":"Digital Logic Design","credit":3}`, wantStatus: http.StatusCreated, wantCode: "CSE 233"},
		{name: "created without a hyphen", body: `{"code":"cse233","title":"Digital Logic Design","credit":3}`, wantStatus: http.StatusCreated, wantCode: "CSE233"},
		{name: "no credit", body: `{"code":"CSE-233","title":"Digital Logic Design"}`,
			wantStatus: http.StatusBadRequest, wantError: "Credit is required."},
		{name: "a credit off the quarter step", body: `{"code":"CSE-233","title":"Digital Logic Design","credit":1.3}`,
			wantStatus: http.StatusBadRequest, wantError: "Credit must move in steps of 0.25."},
		{name: "a credit below the floor", body: `{"code":"CSE-233","title":"Digital Logic Design","credit":0.25}`,
			wantStatus: http.StatusBadRequest, wantError: "Credit must be between 0.5 and 6.0."},
		{name: "a credit above the ceiling", body: `{"code":"CSE-233","title":"Digital Logic Design","credit":6.5}`,
			wantStatus: http.StatusBadRequest, wantError: "Credit must be between 0.5 and 6.0."},
		{name: "a half credit lab", body: `{"code":"CSE-234","title":"Digital Logic Lab","credit":1.5}`,
			wantStatus: http.StatusCreated, wantCode: "CSE 234"},
		{name: "invalid json", body: `{`, wantStatus: http.StatusBadRequest, wantError: "Invalid request body"},
		{name: "unknown field", body: `{"code":"CSE-233","title":"A","credits":3}`, wantStatus: http.StatusBadRequest, wantError: "Invalid request body"},
		{name: "missing code", body: `{"code":"","title":"Digital Logic Design","credit":3}`, wantStatus: http.StatusBadRequest, wantError: "Course code is required."},
		{
			name:       "a code without a digit",
			body:       `{"code":"CSE CODE","title":"Digital Logic Design","credit":3}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Course code must contain at least one digit.",
		},
		{
			name:       "a code with an underscore",
			body:       `{"code":"CSE_2243","title":"Digital Logic Design","credit":3}`,
			wantStatus: http.StatusBadRequest,
			wantError:  validate.MessageCourseCode,
		},
		{
			name:       "a long code with spaces",
			body:       `{"code":"cse 2243 7637d","title":"Compilers","credit":3}`,
			wantStatus: http.StatusCreated,
			wantCode:   "CSE 2243 7637D",
		},
		{
			name:       "a code without the letter prefix",
			body:       `{"code":"2243 7637D","title":"Compilers","credit":3}`,
			wantStatus: http.StatusCreated,
			wantCode:   "2243 7637D",
		},
		{name: "missing title", body: `{"code":"CSE-233","title":"  ","credit":3}`, wantStatus: http.StatusBadRequest, wantError: "Title is required."},
		{
			name:       "duplicate",
			body:       `{"code":"CSE-233","title":"Digital Logic Design","credit":3}`,
			storeErr:   store.ErrDuplicateCourse,
			wantStatus: http.StatusConflict,
			wantError:  MessageDuplicateCourse,
		},
		{
			name:       "store failure",
			body:       `{"code":"CSE-233","title":"Digital Logic Design","credit":3}`,
			storeErr:   errors.New("database is down"),
			wantStatus: http.StatusInternalServerError,
			wantError:  "Internal server error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			catalog := &fakeCatalogStore{err: test.storeErr}
			rec := adminCall(t, NewCatalog(catalog).CreateCourse(), http.MethodPost, "/api/admin/courses", "", test.body)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			if got := decodeBody(t, rec)["code"]; got != test.wantCode {
				t.Errorf("code = %v, want %q", got, test.wantCode)
			}
			if len(catalog.createdCourse) != 1 || catalog.createdCourse[0].Code != test.wantCode {
				t.Errorf("the store received %v, want the uppercase code %q", catalog.createdCourse, test.wantCode)
			}
		})
	}
}

func TestCoursesList(t *testing.T) {
	silenceLog(t)
	catalog := &fakeCatalogStore{courses: []store.Course{
		{ID: 1, Code: "CSE-233", Title: "Digital Logic Design"},
		{ID: 2, Code: "CSE-250", Title: "Software Engineering"},
	}}
	rec := adminCall(t, NewCatalog(catalog).Courses(), http.MethodGet, "/api/admin/courses", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	list := items(t, rec)
	if len(list) != 2 || list[0]["code"] != "CSE-233" || list[1]["code"] != "CSE-250" {
		t.Errorf("items = %v, want the two courses in code order", list)
	}
	failing := NewCatalog(&fakeCatalogStore{err: errors.New("database is down")})
	broken := adminCall(t, failing.Courses(), http.MethodGet, "/api/admin/courses", "", "")
	if broken.Code != http.StatusInternalServerError {
		t.Errorf("failing store status = %d, want 500", broken.Code)
	}
}

func TestCreateRoutineValidation(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		body       string
		storeErr   error
		wantStatus int
		wantError  string
	}{
		{name: "created", body: validRoutine, wantStatus: http.StatusCreated},
		{name: "created for both sections", body: strings.Replace(validRoutine, `"section":"A"`, `"section":"all"`, 1), wantStatus: http.StatusCreated},
		{name: "invalid json", body: `{`, wantStatus: http.StatusBadRequest, wantError: "Invalid request body"},
		{
			name:       "bad batch",
			body:       strings.Replace(validRoutine, `"batch":"CSE-23"`, `"batch":"CSE23"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Batch must look like CSE-23.",
		},
		{
			name:       "bad section",
			body:       strings.Replace(validRoutine, `"section":"A"`, `"section":"AB"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Section must be A, B or ALL.",
		},
		{
			name:       "missing course",
			body:       strings.Replace(validRoutine, `"course_id":4`, `"course_id":0`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  MessageCourseRequired,
		},
		{
			name:       "missing teacher",
			body:       strings.Replace(validRoutine, `"teacher_id":6`, `"teacher_id":-1`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  MessageTeacherRequired,
		},
		{
			name:       "bad day",
			body:       strings.Replace(validRoutine, `"day_of_week":1`, `"day_of_week":7`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Day of week must be a number from 0 (Sunday) to 6 (Saturday).",
		},
		{
			name:       "before opening",
			body:       strings.Replace(validRoutine, `"start_time":"10:30"`, `"start_time":"07:00"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Classes must be between 09:00 and 16:45.",
		},
		{
			name:       "after closing",
			body:       strings.Replace(validRoutine, `"end_time":"12:45"`, `"end_time":"17:30"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Classes must be between 09:00 and 16:45.",
		},
		{
			name:       "start after end",
			body:       strings.Replace(validRoutine, `"start_time":"10:30"`, `"start_time":"13:00"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Start time must be earlier than end time.",
		},
		{
			name:       "start equal to end",
			body:       strings.Replace(validRoutine, `"end_time":"12:45"`, `"end_time":"10:30"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Start time must be earlier than end time.",
		},
		{
			name:       "missing room",
			body:       strings.Replace(validRoutine, `"room":"1003"`, `"room":" "`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Room is required.",
		},
		{
			name:       "bad type",
			body:       strings.Replace(validRoutine, `"type":"lab"`, `"type":"exam"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Type must be class or lab.",
		},
		{name: "unknown course", body: validRoutine, storeErr: store.ErrCourseNotFound, wantStatus: http.StatusBadRequest, wantError: MessageCourseUnknown},
		{name: "unknown teacher", body: validRoutine, storeErr: store.ErrTeacherNotFound, wantStatus: http.StatusBadRequest, wantError: MessageTeacherNotFound},
		{name: "inactive teacher", body: validRoutine, storeErr: store.ErrTeacherInactive, wantStatus: http.StatusBadRequest, wantError: MessageTeacherNotActive},
		{name: "store failure", body: validRoutine, storeErr: errors.New("database is down"), wantStatus: http.StatusInternalServerError, wantError: "Internal server error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			catalog := &fakeCatalogStore{err: test.storeErr, created: store.RoutineRow{ID: 9, CourseCode: "CSE-233", TeacherName: "Flow Teacher"}}
			rec := adminCall(t, NewCatalog(catalog).CreateRoutine(), http.MethodPost, "/api/admin/routines", "", test.body)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			payload := decodeBody(t, rec)
			if payload["day"] != "Monday" || payload["start_time"] != "10:30" || payload["room"] != "1003" {
				t.Errorf("payload = %v, want Monday 10:30 in room 1003", payload)
			}
			if len(catalog.routines) != 1 || catalog.routines[0].Type != "lab" {
				t.Errorf("the store received %v, want one lab entry", catalog.routines)
			}
		})
	}
}

func TestCreateRoutineConflictMessage(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name    string
		booking store.RoomBooking
		want    string
	}{
		{
			name: "one section",
			booking: store.RoomBooking{
				Room:       "1003",
				DayOfWeek:  1,
				StartTime:  "10:30",
				EndTime:    "12:45",
				Batch:      "CSE-23",
				Section:    "A",
				CourseCode: "CSE-233",
			},
			want: "Room 1003 is already booked on Monday 10:30–12:45 by CSE-23 section A (CSE-233).",
		},
		{
			name: "both sections",
			booking: store.RoomBooking{
				Room:       "2001",
				DayOfWeek:  3,
				StartTime:  "09:00",
				EndTime:    "10:00",
				Batch:      "CSE-22",
				Section:    "ALL",
				CourseCode: "CSE250",
			},
			want: "Room 2001 is already booked on Wednesday 09:00–10:00 by CSE-22 both sections (CSE250).",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			catalog := &fakeCatalogStore{err: &store.RoomBookedError{Booking: test.booking}}
			rec := adminCall(t, NewCatalog(catalog).CreateRoutine(), http.MethodPost, "/api/admin/routines", "", validRoutine)
			if rec.Code != http.StatusConflict {
				t.Fatalf("status = %d, want 409 (body = %q)", rec.Code, rec.Body.String())
			}
			if got := errorMessage(t, rec); got != test.want {
				t.Errorf("error = %q, want %q", got, test.want)
			}
		})
	}
}

func TestRoutinesListFilters(t *testing.T) {
	silenceLog(t)
	entries := []store.RoutineRow{
		{ID: 1, Batch: "CSE-23", Section: "A", CourseCode: "CSE-233", CourseTitle: "Digital Logic Design", TeacherName: "First Teacher", DayOfWeek: 1, StartTime: "10:30", EndTime: "12:45", Room: "1003", Type: "lab"},
		{ID: 2, Batch: "CSE-23", Section: "B", CourseCode: "CSE-250", CourseTitle: "Software Engineering", TeacherName: "Second Teacher", DayOfWeek: 2, StartTime: "09:00", EndTime: "10:00", Room: "1004", Type: "class"},
	}
	tests := []struct {
		name        string
		query       string
		wantStatus  int
		wantBatch   string
		wantSection string
		wantError   string
	}{
		{name: "no filters", query: "", wantStatus: http.StatusOK},
		{name: "batch filter", query: "?batch=cse-23", wantStatus: http.StatusOK, wantBatch: "CSE-23"},
		{name: "section filter", query: "?section=a", wantStatus: http.StatusOK, wantSection: "A"},
		{name: "both filters", query: "?batch=CSE-23&section=ALL", wantStatus: http.StatusOK, wantBatch: "CSE-23", wantSection: "ALL"},
		{name: "bad batch", query: "?batch=23", wantStatus: http.StatusBadRequest, wantError: "batch must look like CSE-23."},
		{name: "bad section", query: "?section=AB", wantStatus: http.StatusBadRequest, wantError: "section must be A, B or ALL."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			catalog := &fakeCatalogStore{entries: entries}
			rec := adminCall(t, NewCatalog(catalog).Routines(), http.MethodGet, "/api/admin/routines"+test.query, "", "")
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			if catalog.batchFilter != test.wantBatch || catalog.sectionFilter != test.wantSection {
				t.Errorf("filters = %q and %q, want %q and %q", catalog.batchFilter, catalog.sectionFilter, test.wantBatch, test.wantSection)
			}
			list := items(t, rec)
			if len(list) != 2 {
				t.Fatalf("got %d items, want 2", len(list))
			}
			if list[0]["section"] != "A" || list[1]["section"] != "B" {
				t.Errorf("items = %v, want the store order kept", list)
			}
			if list[0]["day"] != "Monday" || list[1]["day"] != "Tuesday" {
				t.Errorf("day names = %v and %v, want Monday and Tuesday", list[0]["day"], list[1]["day"])
			}
			if list[0]["course_title"] != "Digital Logic Design" || list[0]["teacher_name"] != "First Teacher" {
				t.Errorf("the first item hides the course or teacher: %v", list[0])
			}
		})
	}
}

func TestDeleteRoutine(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		id         string
		storeErr   error
		wantStatus int
		wantError  string
	}{
		{name: "deleted", id: "9", wantStatus: http.StatusNoContent},
		{name: "bad id", id: "nine", wantStatus: http.StatusBadRequest, wantError: MessageInvalidID},
		{name: "unknown entry", id: "9", storeErr: store.ErrNotFound, wantStatus: http.StatusNotFound, wantError: MessageRoutineNotFound},
		{name: "store failure", id: "9", storeErr: errors.New("database is down"), wantStatus: http.StatusInternalServerError, wantError: "Internal server error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			catalog := &fakeCatalogStore{err: test.storeErr}
			rec := adminCall(t, NewCatalog(catalog).DeleteRoutine(), http.MethodDelete, "/api/admin/routines/"+test.id, test.id, "")
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			if rec.Body.Len() != 0 {
				t.Errorf("body = %q, want it empty", rec.Body.String())
			}
			if len(catalog.deleted) != 1 || catalog.deleted[0] != 9 {
				t.Errorf("deleted = %v, want [9]", catalog.deleted)
			}
		})
	}
}

func TestDeleteCourseRules(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		id         string
		usage      store.CourseUsage
		storeErr   error
		wantStatus int
		wantError  string
	}{
		{name: "deletes an unused course", id: "7", wantStatus: http.StatusOK},
		{
			name:       "an unknown course",
			id:         "9",
			storeErr:   store.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantError:  MessageCourseUnknown,
		},
		{
			name:       "one routine entry",
			id:         "4",
			usage:      store.CourseUsage{Routines: 1},
			storeErr:   store.ErrCourseInUse,
			wantStatus: http.StatusConflict,
			wantError:  "This course is still in use by 1 routine entry. Remove those first.",
		},
		{
			name:       "routines, classes and drop courses",
			id:         "4",
			usage:      store.CourseUsage{Routines: 3, Items: 2, DropCourses: 1},
			storeErr:   store.ErrCourseInUse,
			wantStatus: http.StatusConflict,
			wantError: "This course is still in use by 3 routine entries, 2 classes and exams and " +
				"1 student with it as a drop course. Remove those first.",
		},
		{
			name:       "a broken store",
			id:         "4",
			storeErr:   errors.New("database is down"),
			wantStatus: http.StatusInternalServerError,
			wantError:  "Internal server error",
		},
		{name: "an invalid id", id: "abc", wantStatus: http.StatusBadRequest, wantError: MessageInvalidID},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			catalog := &fakeCatalogStore{usage: test.usage, usageErr: test.storeErr}
			rec := adminCall(t, NewCatalog(catalog).DeleteCourse(), http.MethodDelete,
				"/api/admin/courses/"+test.id, test.id, "")
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			if len(catalog.deletedCourse) != 1 {
				t.Errorf("deleted courses = %v, want one id", catalog.deletedCourse)
			}
		})
	}
}
