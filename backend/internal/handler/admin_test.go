package handler

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
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
	promoted            []int64
	promotion           store.PromotedCR
	createdTeachers     []store.NewTeacher
	teacherUpdates      []store.TeacherUpdate
	resentInvites       []int64
	invite              store.TeacherInvite
	studentFilter       store.StudentFilter
	crFilter            store.CRFilter
	teacherFilter       store.TeacherFilter
	accounts            map[int64]store.AccountRef
	accountErr          error
	restoreErr          error
	deleteErr           error
	restored            []int64
	deleted             []int64
	auditEntries        []store.AuditRow
	auditLimit          int
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

func (f *fakeAdminStore) PromoteCR(ctx context.Context, userID int64) (store.PromotedCR, error) {
	if f.err != nil {
		return store.PromotedCR{}, f.err
	}
	f.promoted = append(f.promoted, userID)
	promotion := f.promotion
	promotion.UserID = userID
	return promotion, nil
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

func (f *fakeAdminStore) UpdateTeacher(ctx context.Context, userID int64, update store.TeacherUpdate) error {
	if f.err != nil {
		return f.err
	}
	f.teacherUpdates = append(f.teacherUpdates, update)
	return nil
}

func (f *fakeAdminStore) ResetTeacherInvite(ctx context.Context, userID int64, inviteHash string, expiresAt time.Time) (store.TeacherInvite, error) {
	if f.err != nil {
		return store.TeacherInvite{}, f.err
	}
	f.resentInvites = append(f.resentInvites, userID)
	invite := f.invite
	invite.UserID = userID
	return invite, nil
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

func (f *fakeAdminStore) FindAccount(ctx context.Context, userID int64) (store.AccountRef, error) {
	if f.accountErr != nil {
		return store.AccountRef{}, f.accountErr
	}
	account, ok := f.accounts[userID]
	if !ok {
		return store.AccountRef{}, store.ErrNotFound
	}
	return account, nil
}

func (f *fakeAdminStore) RestoreAccount(ctx context.Context, userID int64) (string, error) {
	if f.restoreErr != nil {
		return "", f.restoreErr
	}
	account, ok := f.accounts[userID]
	if !ok {
		return "", store.ErrNotFound
	}
	switch account.Status {
	case store.StatusRemoved:
		account.Status = store.StatusActive
	case store.StatusRejected:
		account.Status = store.StatusPending
	default:
		return "", store.ErrNotRestorable
	}
	f.accounts[userID] = account
	f.restored = append(f.restored, userID)
	return account.Status, nil
}

func (f *fakeAdminStore) DeleteAccount(ctx context.Context, userID int64) (string, error) {
	if f.deleteErr != nil {
		return "", f.deleteErr
	}
	account, ok := f.accounts[userID]
	if !ok {
		return "", store.ErrNotFound
	}
	delete(f.accounts, userID)
	f.deleted = append(f.deleted, userID)
	return account.PhotoPath, nil
}

func (f *fakeAdminStore) WriteAudit(ctx context.Context, adminID int64, action, targetSummary string) error {
	f.auditEntries = append(f.auditEntries, store.AuditRow{
		AdminID:       adminID,
		Action:        action,
		TargetSummary: targetSummary,
		CreatedAt:     adminNow,
	})
	return nil
}

func (f *fakeAdminStore) ListAudit(ctx context.Context, limit int) ([]store.AuditRow, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.auditLimit = limit
	return f.auditEntries, nil
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

func newAdminHandler(t *testing.T, people AdminStore) *Admin {
	t.Helper()
	return NewAdmin(accountConfig(t, "test"), people, &fakeNotifier{}, &fakeMailer{}, testPhotos(t))
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
			ID:          7,
			StudentID:   3,
			FullName:    "Pending Student",
			RegNo:       "2023331100",
			Batch:       "CSE-23",
			Section:     "A",
			Email:       "pending@student.sust.edu",
			Phone:       "01712345678",
			BloodGroup:  "O+",
			DateOfBirth: "2003-05-14",
			PhotoPath:   "/uploads/photos/aaaa.jpg",
			CreatedAt:   adminNow,
		},
	}}
	admin := newAdminHandler(t, people)
	rec := adminCall(t, admin.Registrations(), http.MethodGet, "/api/admin/registrations", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	list := items(t, rec)
	if len(list) != 1 {
		t.Fatalf("got %d items, want 1", len(list))
	}
	wantFields := map[string]any{
		"id":            float64(7),
		"user_id":       float64(7),
		"full_name":     "Pending Student",
		"reg_no":        "2023331100",
		"batch":         "CSE-23",
		"section":       "A",
		"email":         "pending@student.sust.edu",
		"phone":         "01712345678",
		"blood_group":   "O+",
		"date_of_birth": "2003-05-14",
		"photo_url":     "/uploads/photos/aaaa.jpg",
		"created_at":    adminNow.Format(time.RFC3339),
	}
	for name, want := range wantFields {
		if got := list[0][name]; got != want {
			t.Errorf("item %s = %v, want %v", name, got, want)
		}
	}
	if _, present := list[0]["drop_courses"]; present {
		t.Error("the registrations response still carries drop courses")
	}
	if strings.Contains(rec.Body.String(), "password") {
		t.Error("the registrations response mentions a password field")
	}
	failing := newAdminHandler(t, &fakeAdminStore{err: errors.New("database is down")})
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
			admin := newAdminHandler(t, people)
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
		{name: "bad section", query: "?section=AB", wantStatus: http.StatusBadRequest, wantError: "section must be A or B."},
		{name: "bad status", query: "?status=deleted", wantStatus: http.StatusBadRequest, wantError: MessageInvalidStatus},
		{name: "zero limit", query: "?limit=0", wantStatus: http.StatusBadRequest, wantError: MessageInvalidLimit},
		{name: "limit over the maximum", query: "?limit=201", wantStatus: http.StatusBadRequest, wantError: MessageInvalidLimit},
		{name: "limit not a number", query: "?limit=ten", wantStatus: http.StatusBadRequest, wantError: MessageInvalidLimit},
		{name: "negative offset", query: "?offset=-1", wantStatus: http.StatusBadRequest, wantError: MessageInvalidOffset},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			people := &fakeAdminStore{}
			admin := newAdminHandler(t, people)
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
	admin := newAdminHandler(t, people)
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
	failing := newAdminHandler(t, &fakeAdminStore{err: errors.New("database is down")})
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
			admin := newAdminHandler(t, people)
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

func TestAdminPromoteCR(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name        string
		body        string
		storeErr    error
		wantStatus  int
		wantMessage string
		wantError   string
	}{
		{name: "promoted", body: `{"user_id":11}`, wantStatus: http.StatusOK, wantMessage: MessageCRPromoted},
		{name: "invalid json", body: `{`, wantStatus: http.StatusBadRequest, wantError: "Invalid request body"},
		{name: "unknown field", body: `{"user_id":11,"is_cr":true}`, wantStatus: http.StatusBadRequest, wantError: "Invalid request body"},
		{name: "missing user id", body: `{}`, wantStatus: http.StatusBadRequest, wantError: MessageInvalidUserID},
		{name: "negative user id", body: `{"user_id":-3}`, wantStatus: http.StatusBadRequest, wantError: MessageInvalidUserID},
		{
			name:       "not an active student",
			body:       `{"user_id":11}`,
			storeErr:   store.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantError:  MessageActiveStudentNotFound,
		},
		{
			name:       "already a class representative",
			body:       `{"user_id":11}`,
			storeErr:   store.ErrAlreadyCR,
			wantStatus: http.StatusConflict,
			wantError:  MessageAlreadyCR,
		},
		{
			name:       "store failure",
			body:       `{"user_id":11}`,
			storeErr:   errors.New("database is down"),
			wantStatus: http.StatusInternalServerError,
			wantError:  "Internal server error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			people := &fakeAdminStore{
				err:       test.storeErr,
				promotion: store.PromotedCR{FullName: "New CR", Batch: "CSE-23", Section: "B"},
			}
			alerts := &fakeNotifier{}
			admin := NewAdmin(accountConfig(t, "test"), people, alerts, &fakeMailer{}, testPhotos(t))
			rec := adminCall(t, admin.PromoteCR(), http.MethodPost, "/api/admin/crs/promote", "", test.body)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				if alerts.calls != 0 {
					t.Errorf("notifications = %d, want none", alerts.calls)
				}
				return
			}
			if got := successMessage(t, rec); got != test.wantMessage {
				t.Errorf("message = %q, want %q", got, test.wantMessage)
			}
			if len(people.promoted) != 1 || people.promoted[0] != 11 {
				t.Errorf("promoted = %v, want [11]", people.promoted)
			}
			if alerts.calls != 1 {
				t.Fatalf("notifications = %d, want 1", alerts.calls)
			}
			want := "You are now the Class Representative of CSE-23 section B"
			if alerts.title != want {
				t.Errorf("notification title = %q, want %q", alerts.title, want)
			}
			if alerts.userID != 11 {
				t.Errorf("notification user = %d, want 11", alerts.userID)
			}
		})
	}
}

func TestAdminCRs(t *testing.T) {
	silenceLog(t)
	people := &fakeAdminStore{crs: []store.StudentRow{
		{ID: 2, FullName: "Class Representative", RegNo: "2023331002", Batch: "CSE-23", Section: "B", IsCR: true, Status: store.StatusActive, CreatedAt: adminNow},
	}}
	admin := newAdminHandler(t, people)
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
	filtered := adminCall(t, admin.CRs(), http.MethodGet, "/api/admin/crs?status=removed", "", "")
	if filtered.Code != http.StatusOK {
		t.Fatalf("status filter code = %d, want 200 (body = %q)", filtered.Code, filtered.Body.String())
	}
	if people.crFilter != (store.CRFilter{Status: store.StatusRemoved}) {
		t.Errorf("filter = %+v, want status removed", people.crFilter)
	}
	bad := adminCall(t, admin.CRs(), http.MethodGet, "/api/admin/crs?batch=23", "", "")
	if bad.Code != http.StatusBadRequest {
		t.Errorf("bad batch status = %d, want 400", bad.Code)
	}
	badStatus := adminCall(t, admin.CRs(), http.MethodGet, "/api/admin/crs?status=gone", "", "")
	if badStatus.Code != http.StatusBadRequest {
		t.Errorf("bad status code = %d, want 400", badStatus.Code)
	}
	failing := newAdminHandler(t, &fakeAdminStore{err: errors.New("database is down")})
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
			admin := newAdminHandler(t, people)
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

func teacherForm() map[string][]string {
	return map[string][]string{
		"name":          {"New Teacher"},
		"designation":   {"Assistant Professor"},
		"email":         {"teacher@sust.edu"},
		"phone":         {"01712345671"},
		"blood_group":   {"A+"},
		"date_of_birth": {"1988-03-09"},
		"office_room":   {"AB-401"},
		"joining_date":  {"2015-01-15"},
	}
}

func withoutField(values map[string][]string, name string, replacement ...string) map[string][]string {
	copied := make(map[string][]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	if len(replacement) == 0 {
		delete(copied, name)
		return copied
	}
	copied[name] = replacement
	return copied
}

func TestAdminInviteTeacher(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		values     map[string][]string
		withPhoto  bool
		storeErr   error
		wantStatus int
		wantError  string
		wantEmail  string
	}{
		{name: "invited", values: teacherForm(), wantStatus: http.StatusCreated},
		{name: "invited with a photo", values: teacherForm(), withPhoto: true, wantStatus: http.StatusCreated},
		{
			name:       "missing designation",
			values:     withoutField(teacherForm(), "designation"),
			wantStatus: http.StatusBadRequest,
			wantError:  "Designation is required.",
		},
		{
			name:       "any email domain is accepted",
			values:     withoutField(teacherForm(), "email", "teacher@gmail.com"),
			wantStatus: http.StatusCreated,
			wantEmail:  "teacher@gmail.com",
		},
		{
			name:       "a personal domain is accepted",
			values:     withoutField(teacherForm(), "email", "Teacher@Example.ORG"),
			wantStatus: http.StatusCreated,
			wantEmail:  "teacher@example.org",
		},
		{
			name:       "a broken email is refused",
			values:     withoutField(teacherForm(), "email", "teacher@localhost"),
			wantStatus: http.StatusBadRequest,
			wantError:  "Email must be a valid email address.",
		},
		{
			name:       "bad phone",
			values:     withoutField(teacherForm(), "phone", "0171234567"),
			wantStatus: http.StatusBadRequest,
			wantError:  validate.MessagePhone,
		},
		{
			name:       "missing birth date",
			values:     withoutField(teacherForm(), "date_of_birth"),
			wantStatus: http.StatusBadRequest,
			wantError:  "Date of birth is required.",
		},
		{
			name:       "duplicate",
			values:     teacherForm(),
			storeErr:   store.ErrDuplicateAccount,
			wantStatus: http.StatusConflict,
			wantError:  MessageDuplicateTeacher,
		},
		{
			name:       "store failure",
			values:     teacherForm(),
			storeErr:   errors.New("database is down"),
			wantStatus: http.StatusInternalServerError,
			wantError:  "Internal server error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			people := &fakeAdminStore{err: test.storeErr}
			mail := &fakeMailer{}
			photos := testPhotos(t)
			admin := NewAdmin(accountConfig(t, "test"), people, &fakeNotifier{}, mail, photos)
			files := []formFile{}
			if test.withPhoto {
				files = append(files, formFile{field: "photo", name: "teacher.png", data: pngPhoto(t, 240, 240)})
			}
			request := multipartRequest(t, http.MethodPost, "/api/admin/teachers", test.values, files...)
			rec := serve(admin.CreateTeacher(), request)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				if savedPhotos(t, photos) != 0 {
					t.Error("a photo stayed on disk although the invitation failed")
				}
				return
			}
			if len(people.createdTeachers) != 1 {
				t.Fatalf("the store received %d teachers, want 1", len(people.createdTeachers))
			}
			created := people.createdTeachers[0]
			wantEmail := test.wantEmail
			if wantEmail == "" {
				wantEmail = "teacher@sust.edu"
			}
			if created.Email != wantEmail {
				t.Errorf("created email = %q, want %q", created.Email, wantEmail)
			}
			if created.Designation != "Assistant Professor" || created.OfficeRoom != "AB-401" {
				t.Errorf("created = %+v, want the submitted designation and office room", created)
			}
			if created.DateOfBirth != "1988-03-09" || created.JoiningDate != "2015-01-15" {
				t.Errorf("created = %+v, want the submitted dates", created)
			}
			if created.InviteHash == "" || created.InviteExpiresAt.IsZero() {
				t.Error("the store did not receive an invitation token and expiry")
			}
			if test.withPhoto && created.PhotoPath == "" {
				t.Error("the store did not receive the uploaded photo")
			}
			payload := decodeBody(t, rec)
			link, _ := payload["invite_link"].(string)
			if !strings.HasPrefix(link, "http://localhost:5173/teacher/activate?token=") {
				t.Errorf("invite_link = %q, want an activation link", link)
			}
			if strings.Contains(link, created.InviteHash) {
				t.Error("the invitation link leaks the stored hash")
			}
			if strings.Contains(rec.Body.String(), "password") {
				t.Error("the invitation response mentions a password field")
			}
		})
	}
}

func TestAdminInviteTeacherMailedInProduction(t *testing.T) {
	silenceLog(t)
	people := &fakeAdminStore{}
	mail := &fakeMailer{configured: true}
	admin := NewAdmin(accountConfig(t, "production"), people, &fakeNotifier{}, mail, testPhotos(t))
	request := multipartRequest(t, http.MethodPost, "/api/admin/teachers", teacherForm())
	rec := serve(admin.CreateTeacher(), request)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body = %q)", rec.Code, rec.Body.String())
	}
	if _, present := decodeBody(t, rec)["invite_link"]; present {
		t.Error("the response carries an invite_link although the mailer is configured")
	}
	if len(mail.sent) != 1 || mail.sent[0].to != "teacher@sust.edu" {
		t.Fatalf("sent mail = %+v, want one message to the teacher", mail.sent)
	}
	if !strings.Contains(mail.sent[0].body, "/teacher/activate?token=") {
		t.Errorf("mail body = %q, want the activation link", mail.sent[0].body)
	}
}

func TestAdminUpdateTeacher(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		id         string
		body       string
		storeErr   error
		wantStatus int
		wantError  string
		wantUpdate bool
	}{
		{name: "updated", id: "6", body: `{"designation":"Professor"}`, wantStatus: http.StatusOK, wantUpdate: true},
		{name: "bad id", id: "six", body: `{"designation":"Professor"}`, wantStatus: http.StatusBadRequest, wantError: MessageInvalidID},
		{name: "nothing to change", id: "6", body: `{}`, wantStatus: http.StatusBadRequest, wantError: MessageNothingToUpdate},
		{
			name:       "phone is not an admin field",
			id:         "6",
			body:       `{"phone":"01712345678"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request body",
		},
		{
			name:       "bio is not an admin field",
			id:         "6",
			body:       `{"bio":"hello"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request body",
		},
		{
			name:       "the teacher id is not an admin field any more",
			id:         "6",
			body:       `{"teacher_id":"T-200"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request body",
		},
		{
			name:       "a broken email is refused",
			id:         "6",
			body:       `{"email":"teacher@localhost"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Email must be a valid email address.",
		},
		{
			name:       "unknown teacher",
			id:         "6",
			body:       `{"designation":"Professor"}`,
			storeErr:   store.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantError:  MessageTeacherNotFound,
		},
		{
			name:       "duplicate email",
			id:         "6",
			body:       `{"email":"taken@sust.edu"}`,
			storeErr:   store.ErrDuplicateAccount,
			wantStatus: http.StatusConflict,
			wantError:  MessageDuplicateTeacher,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			people := &fakeAdminStore{err: test.storeErr}
			admin := newAdminHandler(t, people)
			rec := adminCall(t, admin.UpdateTeacher(), http.MethodPatch, "/api/admin/teachers/"+test.id, test.id, test.body)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			if got := successMessage(t, rec); got != MessageTeacherUpdated {
				t.Errorf("message = %q, want %q", got, MessageTeacherUpdated)
			}
			if !test.wantUpdate {
				return
			}
			if len(people.teacherUpdates) != 1 {
				t.Fatalf("updates = %+v, want one update", people.teacherUpdates)
			}
			if people.teacherUpdates[0].Designation == nil || *people.teacherUpdates[0].Designation != "Professor" {
				t.Errorf("updates = %+v, want one designation change", people.teacherUpdates)
			}
		})
	}
}

func TestAdminResendInvite(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		id         string
		storeErr   error
		wantStatus int
		wantError  string
	}{
		{name: "resent", id: "6", wantStatus: http.StatusOK},
		{name: "bad id", id: "six", wantStatus: http.StatusBadRequest, wantError: MessageInvalidID},
		{name: "unknown teacher", id: "6", storeErr: store.ErrNotFound, wantStatus: http.StatusNotFound, wantError: MessageTeacherNotFound},
		{
			name:       "already activated",
			id:         "6",
			storeErr:   store.ErrNotInvited,
			wantStatus: http.StatusConflict,
			wantError:  MessageTeacherNotInvited,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			people := &fakeAdminStore{
				err:    test.storeErr,
				invite: store.TeacherInvite{Email: "teacher@sust.edu", Name: "Active Teacher"},
			}
			admin := newAdminHandler(t, people)
			path := "/api/admin/teachers/" + test.id + "/resend-invite"
			rec := adminCall(t, admin.ResendInvite(), http.MethodPost, path, test.id, "")
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			if len(people.resentInvites) != 1 || people.resentInvites[0] != 6 {
				t.Errorf("resent = %v, want [6]", people.resentInvites)
			}
			link, _ := decodeBody(t, rec)["invite_link"].(string)
			if !strings.HasPrefix(link, "http://localhost:5173/teacher/activate?token=") {
				t.Errorf("invite_link = %q, want an activation link", link)
			}
		})
	}
}

func TestAdminTeachers(t *testing.T) {
	silenceLog(t)
	people := &fakeAdminStore{teachers: []store.TeacherRow{
		{
			ID: 6, TeacherID: "T-001", Name: "Active Teacher", Designation: "Lecturer",
			Email: "teacher@sust.edu", Phone: "01712345672", OfficeRoom: "AB-401",
			PhotoPath: "/uploads/photos/bbbb.png", Status: store.StatusActive, CreatedAt: adminNow,
		},
	}}
	admin := newAdminHandler(t, people)
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
	if got := list[0]["identifier"]; got != "T-001" {
		t.Errorf("identifier = %v, want T-001", got)
	}
	if got := list[0]["photo_url"]; got != "/uploads/photos/bbbb.png" {
		t.Errorf("photo_url = %v, want the stored photo", got)
	}
	if got := list[0]["designation"]; got != "Lecturer" {
		t.Errorf("designation = %v, want Lecturer", got)
	}
	if strings.Contains(rec.Body.String(), "password") {
		t.Error("the teachers response mentions a password field")
	}
	bad := adminCall(t, admin.Teachers(), http.MethodGet, "/api/admin/teachers?status=gone", "", "")
	if bad.Code != http.StatusBadRequest {
		t.Errorf("bad status filter = %d, want 400", bad.Code)
	}
	failing := newAdminHandler(t, &fakeAdminStore{err: errors.New("database is down")})
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
			admin := newAdminHandler(t, people)
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
	admin := newAdminHandler(t, people)
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
	failing := newAdminHandler(t, &fakeAdminStore{err: errors.New("database is down")})
	broken := adminCall(t, failing.Stats(), http.MethodGet, "/api/admin/stats", "", "")
	if broken.Code != http.StatusInternalServerError {
		t.Errorf("failing store status = %d, want 500", broken.Code)
	}
}
