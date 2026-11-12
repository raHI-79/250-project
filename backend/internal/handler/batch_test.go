package handler

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeBatchStore struct {
	rows     map[string]store.Batch
	statuses []string
	deleted  []string
	audits   []string
	watchers []int64
}

func newFakeBatchStore() *fakeBatchStore {
	return &fakeBatchStore{rows: map[string]store.Batch{
		"CSE-23": {ID: 1, Name: "CSE-23", RunningSemester: "3-1", Sections: []string{"A", "B"},
			Status: store.BatchActive, Students: 42},
	}}
}

func (f *fakeBatchStore) ListBatchRows(ctx context.Context) ([]store.Batch, error) {
	rows := make([]store.Batch, 0, len(f.rows))
	for _, row := range f.rows {
		rows = append(rows, row)
	}
	return rows, nil
}

func (f *fakeBatchStore) FindBatch(ctx context.Context, name string) (store.Batch, error) {
	row, ok := f.rows[name]
	if !ok {
		return store.Batch{}, store.ErrNotFound
	}
	return row, nil
}

func (f *fakeBatchStore) CreateBatch(ctx context.Context, name, semester string, by int64) (store.Batch, error) {
	if _, ok := f.rows[name]; ok {
		return store.Batch{}, store.ErrDuplicateBatch
	}
	row := store.Batch{Name: name, RunningSemester: semester, Sections: []string{"A", "B"},
		Status: store.BatchActive}
	f.rows[name] = row
	return row, nil
}

func (f *fakeBatchStore) SetBatchSemester(ctx context.Context, name, semester string, by int64) error {
	row, ok := f.rows[name]
	if !ok {
		return store.ErrNotFound
	}
	row.RunningSemester = semester
	f.rows[name] = row
	return nil
}

func (f *fakeBatchStore) SetBatchStatus(ctx context.Context, name, status string, by int64) (int, error) {
	row, ok := f.rows[name]
	if !ok {
		return 0, store.ErrNotFound
	}
	if row.Status == status {
		return 0, store.ErrBatchUnchanged
	}
	row.Status = status
	f.rows[name] = row
	f.statuses = append(f.statuses, name+":"+status)
	return row.Students, nil
}

func (f *fakeBatchStore) DeleteBatch(ctx context.Context, name string) (store.BatchRemoval, error) {
	row, ok := f.rows[name]
	if !ok {
		return store.BatchRemoval{}, store.ErrNotFound
	}
	delete(f.rows, name)
	f.deleted = append(f.deleted, name)
	return store.BatchRemoval{Students: row.Students}, nil
}

func (f *fakeBatchStore) BatchWatchers(ctx context.Context, exceptUserID int64) ([]int64, error) {
	return f.watchers, nil
}

func (f *fakeBatchStore) WriteAudit(ctx context.Context, adminID int64, action, targetSummary string) error {
	f.audits = append(f.audits, action)
	return nil
}

func (f *fakeBatchStore) BatchOpen(ctx context.Context, name string) (bool, error) {
	row, ok := f.rows[name]
	return ok && row.Status == store.BatchActive, nil
}

func batchCall(t *testing.T, handlerFunc http.HandlerFunc, method, batch, body string,
	current middleware.User) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "/api/admin/batches/"+batch, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("batch", batch)
	req = req.WithContext(middleware.WithUser(req.Context(), current))
	rec := httptest.NewRecorder()
	handlerFunc.ServeHTTP(rec, req)
	return rec
}

func officeAdmin(confirmed bool) middleware.User {
	current := middleware.User{ID: 9, Identifier: "office", Role: "admin"}
	if confirmed {
		until := time.Now().Add(time.Minute)
		current.SudoUntil = &until
	}
	return current
}

func TestDeactivateAndActivateABatch(t *testing.T) {
	silenceLog(t)
	batches := newFakeBatchStore()
	batches.watchers = []int64{4}
	alerts := &fakeNotifier{}
	handler := NewBatches(accountConfig(t, "test"), batches, alerts)
	rec := batchCall(t, handler.SetStatus(store.BatchInactive), http.MethodPost, "CSE-23", "", officeAdmin(false))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %s)", rec.Code, rec.Body.String())
	}
	if got := decodeBody(t, rec)["status"]; got != store.BatchInactive {
		t.Errorf("status = %v, want inactive", got)
	}
	if alerts.userID != 4 || alerts.title != "CSE-23 was deactivated" {
		t.Errorf("notified %d with %q, want admin 4 told about the deactivation", alerts.userID, alerts.title)
	}
	if len(batches.audits) != 1 || batches.audits[0] != ActionBatchDeactivated {
		t.Errorf("audits = %v, want one deactivation", batches.audits)
	}
	again := batchCall(t, handler.SetStatus(store.BatchInactive), http.MethodPost, "CSE-23", "", officeAdmin(false))
	if again.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409 for a batch that is already inactive", again.Code)
	}
	back := batchCall(t, handler.SetStatus(store.BatchActive), http.MethodPost, "CSE-23", "", officeAdmin(false))
	if back.Code != http.StatusOK || decodeBody(t, back)["status"] != store.BatchActive {
		t.Errorf("status = %d, body = %s, want the batch active again", back.Code, back.Body.String())
	}
	missing := batchCall(t, handler.SetStatus(store.BatchInactive), http.MethodPost, "CSE-77", "", officeAdmin(false))
	if missing.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 for an unknown batch", missing.Code)
	}
}

func TestDeleteABatchNeedsTheNameAndThePassword(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name        string
		body        string
		confirmed   bool
		wantStatus  int
		wantError   string
		wantDeleted bool
	}{
		{name: "wrong name", body: `{"confirm":"CSE-24"}`, confirmed: true,
			wantStatus: http.StatusBadRequest, wantError: MessageBatchConfirm},
		{name: "no name", body: `{}`, confirmed: true,
			wantStatus: http.StatusBadRequest, wantError: MessageBatchConfirm},
		{name: "password not confirmed", body: `{"confirm":"CSE-23"}`,
			wantStatus: http.StatusPreconditionRequired, wantError: MessageConfirmNeeded},
		{name: "confirmed", body: `{"confirm":"cse-23"}`, confirmed: true,
			wantStatus: http.StatusOK, wantDeleted: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			batches := newFakeBatchStore()
			handler := NewBatches(accountConfig(t, "test"), batches, &fakeNotifier{})
			rec := batchCall(t, handler.Delete(), http.MethodDelete, "CSE-23", test.body,
				officeAdmin(test.confirmed))
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %s)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := decodeBody(t, rec)["error"]; got != test.wantError {
					t.Errorf("error = %v, want %q", got, test.wantError)
				}
			}
			if deleted := len(batches.deleted) == 1; deleted != test.wantDeleted {
				t.Errorf("deleted = %v, want %v", batches.deleted, test.wantDeleted)
			}
			if test.wantDeleted && (len(batches.audits) != 1 || batches.audits[0] != ActionBatchDeleted) {
				t.Errorf("audits = %v, want the deletion audited", batches.audits)
			}
		})
	}
}

func TestCreateABatch(t *testing.T) {
	silenceLog(t)
	batches := newFakeBatchStore()
	handler := NewBatches(accountConfig(t, "test"), batches, &fakeNotifier{})
	rec := batchCall(t, handler.Create(), http.MethodPost, "", `{"name":"cse-25","running_semester":"1-1"}`,
		officeAdmin(false))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body = %s)", rec.Code, rec.Body.String())
	}
	if _, ok := batches.rows["CSE-25"]; !ok {
		t.Errorf("rows = %v, want CSE-25 stored", batches.rows)
	}
	twice := batchCall(t, handler.Create(), http.MethodPost, "", `{"name":"CSE-25","running_semester":"1-1"}`,
		officeAdmin(false))
	if twice.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409 for a duplicate", twice.Code)
	}
	bad := batchCall(t, handler.Create(), http.MethodPost, "", `{"name":"25","running_semester":"1-1"}`,
		officeAdmin(false))
	if bad.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for a bad name", bad.Code)
	}
}

func TestInactiveBatchIsNotATarget(t *testing.T) {
	silenceLog(t)
	batches := newFakeBatchStore()
	UseBatchGate(batches)
	t.Cleanup(func() { UseBatchGate(nil) })
	if _, err := openBatch(context.Background(), "Batch", "CSE-23"); err != nil {
		t.Fatalf("openBatch() refused an active batch: %v", err)
	}
	if _, err := openBatch(context.Background(), "Batch", "CSE-25"); err == nil ||
		err.Error() != "CSE-25 is not an active batch." {
		t.Errorf("error = %v, want the unknown batch refused", err)
	}
	if _, err := batches.SetBatchStatus(context.Background(), "CSE-23", store.BatchInactive, 9); err != nil {
		t.Fatalf("SetBatchStatus() returned an unexpected error: %v", err)
	}
	catalog := &fakeCatalogStore{}
	rec := adminCall(t, NewCatalog(catalog).CreateRoutine(), http.MethodPost, "/api/admin/routines", "",
		validRoutine)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body = %s)", rec.Code, rec.Body.String())
	}
	if got := decodeBody(t, rec)["error"]; got != "CSE-23 is not an active batch." {
		t.Errorf("error = %v, want the inactive batch refused", got)
	}
	if len(catalog.routines) != 0 {
		t.Errorf("routines = %+v, want nothing stored", catalog.routines)
	}
}

func TestLoginRefusesAStudentOfAnInactiveBatch(t *testing.T) {
	silenceLog(t)
	users := newAuthStore(t)
	student := users.users["2023331001"]
	student.BatchClosed = true
	users.users["2023331001"] = student
	handler := newAuthHandler(t, users, &fakeIssuer{token: issuedToken})
	rec := post(t, handler.Login(), "/api/auth/login", `{"identifier":"2023331001","password":"student-pass"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body = %s)", rec.Code, rec.Body.String())
	}
	if got := errorMessage(t, rec); got != "Your batch is not active. Please contact the department office." {
		t.Errorf("error = %q, want the inactive batch message", got)
	}
}
