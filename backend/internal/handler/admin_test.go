package handler

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var adminNow = time.Date(2026, 10, 16, 9, 30, 0, 0, time.UTC)

type fakeAdminStore struct {
	pending             []store.PendingRegistration
	students            []store.StudentRow
	total               int
	crs                 []store.StudentRow
	teachers            []store.TeacherRow
	stats               store.Stats
	err                 error
	nextID              int64
	statuses            map[int64]string
	removed             []int64
	demoted             []int64
	deactivatedCRs      []int64
	deactivatedTeachers []int64
	createdCRs          []store.NewCR
	createdTeachers     []store.NewTeacher
	studentFilter       store.StudentFilter
	crFilter            store.CRFilter
	teacherFilter       store.TeacherFilter
}

func (f *fakeAdminStore) ListPendingRegistrations(ctx context.Context) ([]store.PendingRegistration, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.pending, nil
}

func (f *fakeAdminStore) SetRegistrationStatus(ctx context.Context, userID int64, status string) error {
	if f.err != nil {
		return f.err
	}
	if f.statuses == nil {
		f.statuses = make(map[int64]string)
	}
	f.statuses[userID] = status
	return nil
}

func (f *fakeAdminStore) ListStudents(ctx context.Context, filter store.StudentFilter) ([]store.StudentRow, int, error) {
	f.studentFilter = filter
	if f.err != nil {
		return nil, 0, f.err
	}
	return f.students, f.total, nil
}

func (f *fakeAdminStore) RemoveStudent(ctx context.Context, userID int64) error {
	if f.err != nil {
		return f.err
	}
	f.removed = append(f.removed, userID)
	return nil
}

func (f *fakeAdminStore) CreateCR(ctx context.Context, cr store.NewCR) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	f.createdCRs = append(f.createdCRs, cr)
	f.nextID++
	return f.nextID, nil
}

func (f *fakeAdminStore) ListCRs(ctx context.Context, filter store.CRFilter) ([]store.StudentRow, error) {
	f.crFilter = filter
	if f.err != nil {
		return nil, f.err
	}
	return f.crs, nil
}

func (f *fakeAdminStore) DemoteCR(ctx context.Context, userID int64) error {
	if f.err != nil {
		return f.err
	}
	f.demoted = append(f.demoted, userID)
	return nil
}

func (f *fakeAdminStore) DeactivateCR(ctx context.Context, userID int64) error {
	if f.err != nil {
		return f.err
	}
	f.deactivatedCRs = append(f.deactivatedCRs, userID)
	return nil
}

func (f *fakeAdminStore) CreateTeacher(ctx context.Context, teacher store.NewTeacher) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	f.createdTeachers = append(f.createdTeachers, teacher)
	f.nextID++
	return f.nextID, nil
}

func (f *fakeAdminStore) ListTeachers(ctx context.Context, filter store.TeacherFilter) ([]store.TeacherRow, error) {
	f.teacherFilter = filter
	if f.err != nil {
		return nil, f.err
	}
	return f.teachers, nil
}

func (f *fakeAdminStore) DeactivateTeacher(ctx context.Context, userID int64) error {
	if f.err != nil {
		return f.err
	}
	f.deactivatedTeachers = append(f.deactivatedTeachers, userID)
	return nil
}

func (f *fakeAdminStore) Stats(ctx context.Context) (store.Stats, error) {
	if f.err != nil {
		return store.Stats{}, f.err
	}
	return f.stats, nil
}

func adminCall(t *testing.T, handlerFunc http.HandlerFunc, method, target, id, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	if body != "" {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if id != "" {
		req.SetPathValue("id", id)
	}
	rec := httptest.NewRecorder()
	handlerFunc.ServeHTTP(rec, req)
	return rec
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	return payload
}

func items(t *testing.T, rec *httptest.ResponseRecorder) []map[string]any {
	t.Helper()
	raw, ok := decodeBody(t, rec)["items"]
	if !ok {
		t.Fatalf("the response has no items field (body = %q)", rec.Body.String())
	}
	list, ok := raw.([]any)
	if !ok {
		t.Fatalf("items is not an array (body = %q)", rec.Body.String())
	}
	out := make([]map[string]any, 0, len(list))
	for _, entry := range list {
		item, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("an item is not an object (body = %q)", rec.Body.String())
		}
		out = append(out, item)
	}
	return out
}

func TestAdminRegistrations(t *testing.T) {
	silenceLog(t)
	people := &fakeAdminStore{pending: []store.PendingRegistration{
		{
			ID:             7,
			FullName:       "Pending Student",
			RegNo:          "2023331100",
			Batch:          "CSE-23",
			Section:        "A",
			Email:          "pending@student.sust.edu",
			Phone:          "01712345678",
			BloodGroup:     "O+",
			HasDropCourse:  true,
			DropCourseCode: "CSE-233",
			CreatedAt:      adminNow,
		},
	}}
	admin := NewAdmin(people)
	rec := adminCall(t, admin.Registrations(), http.MethodGet, "/api/admin/registrations", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	list := items(t, rec)
	if len(list) != 1 {
		t.Fatalf("got %d items, want 1", len(list))
	}
	wantFields := map[string]any{
		"id":               float64(7),
		"full_name":        "Pending Student",
		"reg_no":           "2023331100",
		"batch":            "CSE-23",
		"section":          "A",
		"email":            "pending@student.sust.edu",
		"phone":            "01712345678",
		"blood_group":      "O+",
		"has_drop_course":  true,
		"drop_course_code": "CSE-233",
		"created_at":       adminNow.Format(time.RFC3339),
	}
	for name, want := range wantFields {
		if got := list[0][name]; got != want {
			t.Errorf("item %s = %v, want %v", name, got, want)
		}
	}
	if strings.Contains(rec.Body.String(), "password") {
		t.Error("the registrations response mentions a password field")
	}
	failing := NewAdmin(&fakeAdminStore{err: errors.New("database is down")})
	broken := adminCall(t, failing.Registrations(), http.MethodGet, "/api/admin/registrations", "", "")
	if broken.Code != http.StatusInternalServerError {
		t.Errorf("failing store status = %d, want 500", broken.Code)
	}
}

func TestAdminRegistrationDecisions(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name        string
		approve     bool
		id          string
		storeErr    error
		wantStatus  int
		wantMessage string
		wantError   string
		wantStored  string
	}{
		{name: "approve", approve: true, id: "7", wantStatus: http.StatusOK, wantMessage: MessageRegistrationApproved, wantStored: store.StatusActive},
		{name: "reject", id: "7", wantStatus: http.StatusOK, wantMessage: MessageRegistrationRejected, wantStored: store.StatusRejected},
		{name: "not a number", approve: true, id: "abc", wantStatus: http.StatusBadRequest, wantError: MessageInvalidID},
		{name: "zero id", approve: true, id: "0", wantStatus: http.StatusBadRequest, wantError: MessageInvalidID},
		{name: "unknown student", approve: true, id: "7", storeErr: store.ErrNotFound, wantStatus: http.StatusNotFound, wantError: MessageStudentNotFound},
		{name: "already decided", approve: true, id: "7", storeErr: store.ErrNotPending, wantStatus: http.StatusConflict, wantError: MessageRegistrationNotPending},
		{name: "store failure", approve: true, id: "7", storeErr: errors.New("database is down"), wantStatus: http.StatusInternalServerError, wantError: "Internal server error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			people := &fakeAdminStore{err: test.storeErr}
			admin := NewAdmin(people)
			handlerFunc := admin.RejectRegistration()
			path := "/api/admin/registrations/" + test.id + "/reject"
			if test.approve {
				handlerFunc = admin.ApproveRegistration()
				path = "/api/admin/registrations/" + test.id + "/approve"
			}
			rec := adminCall(t, handlerFunc, http.MethodPost, path, test.id, "")
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantMessage != "" && successMessage(t, rec) != test.wantMessage {
				t.Errorf("message = %q, want %q", successMessage(t, rec), test.wantMessage)
			}
			if test.wantError != "" && errorMessage(t, rec) != test.wantError {
				t.Errorf("error = %q, want %q", errorMessage(t, rec), test.wantError)
			}
			if test.wantStored != "" && people.statuses[7] != test.wantStored {
				t.Errorf("stored status = %q, want %q", people.statuses[7], test.wantStored)
			}
		})
	}
}

func TestAdminStudentsFilters(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantError  string
		wantFilter store.StudentFilter
	}{
		{
			name:       "defaults",
			query:      "",
			wantStatus: http.StatusOK,
			wantFilter: store.StudentFilter{Limit: store.DefaultStudentPage},
		},
		{
			name:       "all filters",
			query:      "?search=ratul&batch=cse-23&section=a&status=active&limit=10&offset=20",
			wantStatus: http.StatusOK,
			wantFilter: store.StudentFilter{Search: "ratul", Batch: "CSE-23", Section: "A", Status: store.StatusActive, Limit: 10, Offset: 20},
		},
		{
			name:       "status all",
			query:      "?status=all",
			wantStatus: http.StatusOK,
			wantFilter: store.StudentFilter{Status: store.StatusFilterAll, Limit: store.DefaultStudentPage},
		},
		{name: "bad batch", query: "?batch=CSE23", wantStatus: http.StatusBadRequest, wantError: "batch must look like CSE-23."},
		{name: "bad section", query: "?section=AB", wantStatus: http.StatusBadRequest, wantError: "section must be a single letter from A to Z."},
		{name: "bad status", query: "?status=deleted", wantStatus: http.StatusBadRequest, wantError: MessageInvalidStatus},
		{name: "zero limit", query: "?limit=0", wantStatus: http.StatusBadRequest, wantError: MessageInvalidLimit},
		{name: "limit over the maximum", query: "?limit=201", wantStatus: http.StatusBadRequest, wantError: MessageInvalidLimit},
		{name: "limit not a number", query: "?limit=ten", wantStatus: http.StatusBadRequest, wantError: MessageInvalidLimit},
		{name: "negative offset", query: "?offset=-1", wantStatus: http.StatusBadRequest, wantError: MessageInvalidOffset},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			people := &fakeAdminStore{}
			admin := NewAdmin(people)
			rec := adminCall(t, admin.Students(), http.MethodGet, "/api/admin/students"+test.query, "", "")
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			if people.studentFilter != test.wantFilter {
				t.Errorf("filter = %+v, want %+v", people.studentFilter, test.wantFilter)
			}
		})
	}
}

func TestAdminStudentsResponse(t *testing.T) {
	silenceLog(t)
	people := &fakeAdminStore{
		total: 3,
		students: []store.StudentRow{
			{ID: 2, FullName: "Class Representative", RegNo: "2023331002", Batch: "CSE-23", Section: "B", Email: "cr@student.sust.edu", Phone: "01712345679", BloodGroup: "B+", IsCR: true, Status: store.StatusActive, CreatedAt: adminNow},
		},
	}
	admin := NewAdmin(people)
	rec := adminCall(t, admin.Students(), http.MethodGet, "/api/admin/students", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	payload := decodeBody(t, rec)
	if got := payload["total"]; got != float64(3) {
		t.Errorf("total = %v, want 3", got)
	}
	list := items(t, rec)
	if len(list) != 1 {
		t.Fatalf("got %d items, want 1", len(list))
	}
	if got := list[0]["is_cr"]; got != true {
		t.Errorf("is_cr = %v, want true", got)
	}
	if got := list[0]["status"]; got != store.StatusActive {
		t.Errorf("status = %v, want %q", got, store.StatusActive)
	}
	if strings.Contains(rec.Body.String(), "password") {
		t.Error("the students response mentions a password field")
	}
	failing := NewAdmin(&fakeAdminStore{err: errors.New("database is down")})
	broken := adminCall(t, failing.Students(), http.MethodGet, "/api/admin/students", "", "")
	if broken.Code != http.StatusInternalServerError {
		t.Errorf("failing store status = %d, want 500", broken.Code)
	}
}

func TestAdminRemoveStudent(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name        string
		id          string
		storeErr    error
		wantStatus  int
		wantMessage string
		wantError   string
	}{
		{name: "removed", id: "7", wantStatus: http.StatusOK, wantMessage: MessageStudentRemoved},
		{name: "bad id", id: "x", wantStatus: http.StatusBadRequest, wantError: MessageInvalidID},
		{name: "unknown student", id: "7", storeErr: store.ErrNotFound, wantStatus: http.StatusNotFound, wantError: MessageStudentNotFound},
		{name: "already removed", id: "7", storeErr: store.ErrAlreadyRemoved, wantStatus: http.StatusConflict, wantError: MessageAlreadyRemoved},
		{name: "store failure", id: "7", storeErr: errors.New("database is down"), wantStatus: http.StatusInternalServerError, wantError: "Internal server error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			people := &fakeAdminStore{err: test.storeErr}
			admin := NewAdmin(people)
			rec := adminCall(t, admin.RemoveStudent(), http.MethodPost, "/api/admin/students/"+test.id+"/remove", test.id, "")
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantMessage != "" {
				if got := successMessage(t, rec); got != test.wantMessage {
					t.Errorf("message = %q, want %q", got, test.wantMessage)
				}
				if len(people.removed) != 1 || people.removed[0] != 7 {
					t.Errorf("removed = %v, want [7]", people.removed)
				}
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
			}
		})
	}
}

const validCR = `{"full_name":"New CR","reg_no":"2023331200","batch":"CSE-23","section":"A",
"email":"newcr@student.sust.edu","phone":"01712345670","blood_group":"A+",
"password":"sust-250-project","confirm_password":"sust-250-project"}`

func TestAdminCreateCR(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		body       string
		storeErr   error
		wantStatus int
		wantError  string
	}{
		{name: "created", body: validCR, wantStatus: http.StatusCreated},
		{name: "invalid json", body: `{`, wantStatus: http.StatusBadRequest, wantError: "Invalid request body"},
		{name: "unknown field", body: `{"full_name":"A","is_cr":true}`, wantStatus: http.StatusBadRequest, wantError: "Invalid request body"},
		{
			name:       "missing full name",
			body:       strings.Replace(validCR, `"full_name":"New CR"`, `"full_name":" "`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Full name is required.",
		},
		{
			name:       "bad batch",
			body:       strings.Replace(validCR, `"batch":"CSE-23"`, `"batch":"CSE23"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Batch must look like CSE-23.",
		},
		{
			name:       "bad blood group",
			body:       strings.Replace(validCR, `"blood_group":"A+"`, `"blood_group":"C+"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Blood group must be one of A+, A-, B+, B-, AB+, AB-, O+, O-.",
		},
		{
			name:       "passwords do not match",
			body:       strings.Replace(validCR, `"confirm_password":"sust-250-project"`, `"confirm_password":"other-password"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Password and the confirmation do not match.",
		},
		{name: "duplicate", body: validCR, storeErr: store.ErrDuplicateAccount, wantStatus: http.StatusConflict, wantError: MessageDuplicateAccount},
		{name: "store failure", body: validCR, storeErr: errors.New("database is down"), wantStatus: http.StatusInternalServerError, wantError: "Internal server error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			people := &fakeAdminStore{err: test.storeErr}
			admin := NewAdmin(people)
			rec := adminCall(t, admin.CreateCR(), http.MethodPost, "/api/admin/crs", "", test.body)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			if len(people.createdCRs) != 1 {
				t.Fatalf("the store received %d class representatives, want 1", len(people.createdCRs))
			}
			created := people.createdCRs[0]
			if created.RegNo != "2023331200" || created.Batch != "CSE-23" || created.Section != "A" {
				t.Errorf("created = %+v, want the submitted registration number, batch and section", created)
			}
			if created.PasswordHash == "" || created.PasswordHash == "sust-250-project" {
				t.Error("the store did not receive a hashed password")
			}
			if strings.Contains(rec.Body.String(), created.PasswordHash) || strings.Contains(rec.Body.String(), "sust-250-project") {
				t.Error("the response body leaks the password or its hash")
			}
			if got := decodeBody(t, rec)["id"]; got != float64(1) {
				t.Errorf("id = %v, want 1", got)
			}
		})
	}
}

func TestAdminCRs(t *testing.T) {
	silenceLog(t)
	people := &fakeAdminStore{crs: []store.StudentRow{
		{ID: 2, FullName: "Class Representative", RegNo: "2023331002", Batch: "CSE-23", Section: "B", IsCR: true, Status: store.StatusActive, CreatedAt: adminNow},
	}}
	admin := NewAdmin(people)
	rec := adminCall(t, admin.CRs(), http.MethodGet, "/api/admin/crs?search=class&batch=cse-23", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if people.crFilter != (store.CRFilter{Search: "class", Batch: "CSE-23"}) {
		t.Errorf("filter = %+v, want search class and batch CSE-23", people.crFilter)
	}
	if list := items(t, rec); len(list) != 1 || list[0]["is_cr"] != true {
		t.Errorf("items = %v, want one class representative", list)
	}
	bad := adminCall(t, admin.CRs(), http.MethodGet, "/api/admin/crs?batch=23", "", "")
	if bad.Code != http.StatusBadRequest {
		t.Errorf("bad batch status = %d, want 400", bad.Code)
	}
	failing := NewAdmin(&fakeAdminStore{err: errors.New("database is down")})
	broken := adminCall(t, failing.CRs(), http.MethodGet, "/api/admin/crs", "", "")
	if broken.Code != http.StatusInternalServerError {
		t.Errorf("failing store status = %d, want 500", broken.Code)
	}
}

func TestAdminDemoteAndDeactivateCR(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name        string
		deactivate  bool
		id          string
		storeErr    error
		wantStatus  int
		wantMessage string
		wantError   string
	}{
		{name: "demoted", id: "2", wantStatus: http.StatusOK, wantMessage: MessageCRDemoted},
		{name: "demote bad id", id: "-3", wantStatus: http.StatusBadRequest, wantError: MessageInvalidID},
		{name: "demote unknown", id: "2", storeErr: store.ErrNotFound, wantStatus: http.StatusNotFound, wantError: MessageStudentNotFound},
		{name: "demote plain student", id: "2", storeErr: store.ErrNotCR, wantStatus: http.StatusConflict, wantError: MessageNotCR},
		{name: "deactivated", deactivate: true, id: "2", wantStatus: http.StatusOK, wantMessage: MessageCRDeactivated},
		{name: "deactivate bad id", deactivate: true, id: "", wantStatus: http.StatusBadRequest, wantError: MessageInvalidID},
		{name: "deactivate unknown", deactivate: true, id: "2", storeErr: store.ErrNotFound, wantStatus: http.StatusNotFound, wantError: MessageCRNotFound},
		{name: "deactivate already removed", deactivate: true, id: "2", storeErr: store.ErrAlreadyRemoved, wantStatus: http.StatusConflict, wantError: MessageAlreadyRemoved},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			people := &fakeAdminStore{err: test.storeErr}
			admin := NewAdmin(people)
			handlerFunc := admin.DemoteCR()
			path := "/api/admin/crs/" + test.id + "/demote"
			if test.deactivate {
				handlerFunc = admin.DeactivateCR()
				path = "/api/admin/crs/" + test.id + "/deactivate"
			}
			rec := adminCall(t, handlerFunc, http.MethodPost, path, test.id, "")
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantMessage != "" && successMessage(t, rec) != test.wantMessage {
				t.Errorf("message = %q, want %q", successMessage(t, rec), test.wantMessage)
			}
			if test.wantError != "" && errorMessage(t, rec) != test.wantError {
				t.Errorf("error = %q, want %q", errorMessage(t, rec), test.wantError)
			}
		})
	}
}

const validTeacher = `{"name":"New Teacher","teacher_id":"T-100","email":"teacher@sust.edu",
"phone":"01712345671","password":"sust-250-project","confirm_password":"sust-250-project"}`

func TestAdminCreateTeacher(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		body       string
		storeErr   error
		wantStatus int
		wantError  string
	}{
		{name: "created", body: validTeacher, wantStatus: http.StatusCreated},
		{name: "invalid json", body: `{`, wantStatus: http.StatusBadRequest, wantError: "Invalid request body"},
		{
			name:       "missing name",
			body:       strings.Replace(validTeacher, `"name":"New Teacher"`, `"name":""`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Name is required.",
		},
		{
			name:       "short teacher id",
			body:       strings.Replace(validTeacher, `"teacher_id":"T-100"`, `"teacher_id":"T1"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Teacher ID must be 3 to 20 letters, digits or hyphens.",
		},
		{
			name:       "teacher id with spaces",
			body:       strings.Replace(validTeacher, `"teacher_id":"T-100"`, `"teacher_id":"T 100"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Teacher ID must be 3 to 20 letters, digits or hyphens.",
		},
		{
			name:       "bad email",
			body:       strings.Replace(validTeacher, `"email":"teacher@sust.edu"`, `"email":"teacher.sust.edu"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Email must be a valid email address.",
		},
		{
			name:       "short password",
			body:       strings.ReplaceAll(validTeacher, `"sust-250-project"`, `"short"`),
			wantStatus: http.StatusBadRequest,
			wantError:  "Password must be at least 8 characters long.",
		},
		{name: "duplicate", body: validTeacher, storeErr: store.ErrDuplicateAccount, wantStatus: http.StatusConflict, wantError: MessageDuplicateTeacher},
		{name: "store failure", body: validTeacher, storeErr: errors.New("database is down"), wantStatus: http.StatusInternalServerError, wantError: "Internal server error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			people := &fakeAdminStore{err: test.storeErr}
			admin := NewAdmin(people)
			rec := adminCall(t, admin.CreateTeacher(), http.MethodPost, "/api/admin/teachers", "", test.body)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			if len(people.createdTeachers) != 1 {
				t.Fatalf("the store received %d teachers, want 1", len(people.createdTeachers))
			}
			created := people.createdTeachers[0]
			if created.TeacherID != "T-100" || created.Email != "teacher@sust.edu" {
				t.Errorf("created = %+v, want the submitted teacher ID and email", created)
			}
			if created.PasswordHash == "" || created.PasswordHash == "sust-250-project" {
				t.Error("the store did not receive a hashed password")
			}
			if strings.Contains(rec.Body.String(), created.PasswordHash) || strings.Contains(rec.Body.String(), "sust-250-project") {
				t.Error("the response body leaks the password or its hash")
			}
		})
	}
}

func TestAdminTeachers(t *testing.T) {
	silenceLog(t)
	people := &fakeAdminStore{teachers: []store.TeacherRow{
		{ID: 6, TeacherID: "T-001", Name: "Active Teacher", Email: "teacher@sust.edu", Phone: "01712345672", Status: store.StatusActive, CreatedAt: adminNow},
	}}
	admin := NewAdmin(people)
	rec := adminCall(t, admin.Teachers(), http.MethodGet, "/api/admin/teachers?search=active&status=active", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if people.teacherFilter != (store.TeacherFilter{Search: "active", Status: store.StatusActive}) {
		t.Errorf("filter = %+v, want search active and status active", people.teacherFilter)
	}
	list := items(t, rec)
	if len(list) != 1 {
		t.Fatalf("got %d items, want 1", len(list))
	}
	if got := list[0]["teacher_id"]; got != "T-001" {
		t.Errorf("teacher_id = %v, want T-001", got)
	}
	if strings.Contains(rec.Body.String(), "password") {
		t.Error("the teachers response mentions a password field")
	}
	bad := adminCall(t, admin.Teachers(), http.MethodGet, "/api/admin/teachers?status=gone", "", "")
	if bad.Code != http.StatusBadRequest {
		t.Errorf("bad status filter = %d, want 400", bad.Code)
	}
	failing := NewAdmin(&fakeAdminStore{err: errors.New("database is down")})
	broken := adminCall(t, failing.Teachers(), http.MethodGet, "/api/admin/teachers", "", "")
	if broken.Code != http.StatusInternalServerError {
		t.Errorf("failing store status = %d, want 500", broken.Code)
	}
}

func TestAdminDeactivateTeacher(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name        string
		id          string
		storeErr    error
		wantStatus  int
		wantMessage string
		wantError   string
	}{
		{name: "deactivated", id: "6", wantStatus: http.StatusOK, wantMessage: MessageTeacherDeactivated},
		{name: "bad id", id: "six", wantStatus: http.StatusBadRequest, wantError: MessageInvalidID},
		{name: "unknown teacher", id: "6", storeErr: store.ErrNotFound, wantStatus: http.StatusNotFound, wantError: MessageTeacherNotFound},
		{name: "already removed", id: "6", storeErr: store.ErrAlreadyRemoved, wantStatus: http.StatusConflict, wantError: MessageAlreadyRemoved},
		{name: "store failure", id: "6", storeErr: errors.New("database is down"), wantStatus: http.StatusInternalServerError, wantError: "Internal server error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			people := &fakeAdminStore{err: test.storeErr}
			admin := NewAdmin(people)
			rec := adminCall(t, admin.DeactivateTeacher(), http.MethodPost, "/api/admin/teachers/"+test.id+"/deactivate", test.id, "")
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantMessage != "" {
				if got := successMessage(t, rec); got != test.wantMessage {
					t.Errorf("message = %q, want %q", got, test.wantMessage)
				}
				if len(people.deactivatedTeachers) != 1 || people.deactivatedTeachers[0] != 6 {
					t.Errorf("deactivated = %v, want [6]", people.deactivatedTeachers)
				}
			}
			if test.wantError != "" && errorMessage(t, rec) != test.wantError {
				t.Errorf("error = %q, want %q", errorMessage(t, rec), test.wantError)
			}
		})
	}
}

func TestAdminStats(t *testing.T) {
	silenceLog(t)
	people := &fakeAdminStore{stats: store.Stats{TotalStudents: 42, TotalCRs: 4, TotalTeachers: 7, PendingApprovals: 3}}
	admin := NewAdmin(people)
	rec := adminCall(t, admin.Stats(), http.MethodGet, "/api/admin/stats", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	payload := decodeBody(t, rec)
	want := map[string]any{
		"total_students":    float64(42),
		"total_crs":         float64(4),
		"total_teachers":    float64(7),
		"pending_approvals": float64(3),
	}
	for name, value := range want {
		if got := payload[name]; got != value {
			t.Errorf("%s = %v, want %v", name, got, value)
		}
	}
	failing := NewAdmin(&fakeAdminStore{err: errors.New("database is down")})
	broken := adminCall(t, failing.Stats(), http.MethodGet, "/api/admin/stats", "", "")
	if broken.Code != http.StatusInternalServerError {
		t.Errorf("failing store status = %d, want 500", broken.Code)
	}
}
