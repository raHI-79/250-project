package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"net/http"
	"strconv"
	"testing"
)

func testPeriods() []store.Period {
	return []store.Period{
		{ID: 1, Position: 1, Period: validate.Period{Label: "Period 1", Start: "09:00", End: "09:45"}},
		{ID: 2, Position: 2, Period: validate.Period{Label: "Period 2", Start: "09:45", End: "10:30"}},
		{ID: 3, Position: 3, Period: validate.Period{Label: "Period 3", Start: "10:30", End: "11:15"}},
		{ID: 4, Position: 4, Period: validate.Period{Label: "Period 4", Start: "11:15", End: "12:00"}},
		{ID: 5, Position: 5, Period: validate.Period{Label: "Period 5", Start: "12:00", End: "12:45"}},
		{ID: 6, Position: 6, Period: validate.Period{Label: "Break", Start: "12:45", End: "13:45", IsBreak: true}},
		{ID: 7, Position: 7, Period: validate.Period{Label: "Period 6", Start: "13:45", End: "14:30"}},
		{ID: 8, Position: 8, Period: validate.Period{Label: "Period 7", Start: "14:30", End: "15:15"}},
		{ID: 9, Position: 9, Period: validate.Period{Label: "Period 8", Start: "15:15", End: "16:00"}},
		{ID: 10, Position: 10, Period: validate.Period{Label: "Period 9", Start: "16:00", End: "16:45"}},
	}
}

type fakePeriodStore struct {
	periods   []store.Period
	created   store.NewPeriod
	patched   store.PeriodPatch
	order     []int64
	deleted   int64
	createErr error
	updateErr error
	deleteErr error
	orderErr  error
}

func (f *fakePeriodStore) ListPeriods(ctx context.Context) ([]store.Period, error) {
	if f.periods == nil {
		return testPeriods(), nil
	}
	return f.periods, nil
}

func (f *fakePeriodStore) CreatePeriod(ctx context.Context, period store.NewPeriod) (store.Period, error) {
	if f.createErr != nil {
		return store.Period{}, f.createErr
	}
	f.created = period
	return store.Period{
		ID:       9,
		Position: 9,
		Period: validate.Period{
			Label:   period.Label,
			Start:   period.Start,
			End:     period.End,
			IsBreak: period.IsBreak,
		},
	}, nil
}

func (f *fakePeriodStore) UpdatePeriod(ctx context.Context, periodID int64,
	patch store.PeriodPatch) (store.Period, error) {
	if f.updateErr != nil {
		return store.Period{}, f.updateErr
	}
	f.patched = patch
	return testPeriods()[0], nil
}

func (f *fakePeriodStore) DeletePeriod(ctx context.Context, periodID int64) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = periodID
	return nil
}

func (f *fakePeriodStore) ReorderPeriods(ctx context.Context, ids []int64) ([]store.Period, error) {
	if f.orderErr != nil {
		return nil, f.orderErr
	}
	f.order = ids
	return testPeriods(), nil
}

func TestPeriodsList(t *testing.T) {
	rec := adminCall(t, NewPeriods(&fakePeriodStore{}).List(), http.MethodGet, "/api/admin/periods", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	payload := decodeBody(t, rec)
	items, ok := payload["items"].([]any)
	if !ok || len(items) != len(testPeriods()) {
		t.Fatalf("items = %v, want the whole teaching day", payload["items"])
	}
}

func TestPeriodsCreate(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		storeErr   error
		wantStatus int
		wantError  string
	}{
		{name: "added", body: `{"label":"Period 4","start_time":"14:30","end_time":"15:15"}`,
			wantStatus: http.StatusCreated},
		{name: "break added", body: `{"label":"Tea","start_time":"16:00","end_time":"16:30","is_break":true}`,
			wantStatus: http.StatusCreated},
		{name: "no label", body: `{"label":"  ","start_time":"14:30","end_time":"15:15"}`,
			wantStatus: http.StatusBadRequest, wantError: "Label is required."},
		{name: "bad clock", body: `{"label":"Period 4","start_time":"25:00","end_time":"15:15"}`,
			wantStatus: http.StatusBadRequest, wantError: "Start time must look like 09:30."},
		{name: "end before start", body: `{"label":"Period 4","start_time":"15:15","end_time":"14:30"}`,
			wantStatus: http.StatusBadRequest, wantError: "Start time must be earlier than end time."},
		{name: "overlap", body: `{"label":"Period 4","start_time":"09:30","end_time":"10:00"}`,
			storeErr:   store.ErrPeriodOverlap,
			wantStatus: http.StatusConflict},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			periods := &fakePeriodStore{createErr: test.storeErr}
			rec := adminCall(t, NewPeriods(periods).Create(), http.MethodPost, "/api/admin/periods", "", test.body)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %s)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := decodeBody(t, rec)["error"]; got != test.wantError {
					t.Errorf("error = %v, want %q", got, test.wantError)
				}
			}
		})
	}
}

func TestPeriodsDelete(t *testing.T) {
	tests := []struct {
		name       string
		storeErr   error
		wantStatus int
		wantError  string
	}{
		{name: "deleted", wantStatus: http.StatusOK},
		{name: "unknown", storeErr: store.ErrNotFound, wantStatus: http.StatusNotFound,
			wantError: MessagePeriodUnknown},
		{name: "last one", storeErr: store.ErrPeriodInUse, wantStatus: http.StatusConflict,
			wantError: MessagePeriodLast},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			periods := &fakePeriodStore{deleteErr: test.storeErr}
			rec := adminCall(t, NewPeriods(periods).Delete(), http.MethodDelete,
				"/api/admin/periods/2", "2", "")
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, test.wantStatus)
			}
			if test.wantError != "" {
				if got := decodeBody(t, rec)["error"]; got != test.wantError {
					t.Errorf("error = %v, want %q", got, test.wantError)
				}
			}
		})
	}
}

func TestPeriodsReorder(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		storeErr   error
		wantStatus int
	}{
		{name: "reordered", body: `{"ids":[2,1,3,4,5,6,7,8,9,10]}`, wantStatus: http.StatusOK},
		{name: "empty", body: `{"ids":[]}`, wantStatus: http.StatusBadRequest},
		{name: "repeated", body: `{"ids":[1,1]}`, wantStatus: http.StatusBadRequest},
		{name: "not a period", body: `{"ids":[7,8]}`, storeErr: store.ErrNotFound,
			wantStatus: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			periods := &fakePeriodStore{orderErr: test.storeErr}
			rec := adminCall(t, NewPeriods(periods).Reorder(), http.MethodPost,
				"/api/admin/periods/reorder", "", test.body)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %s)", rec.Code, test.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestPeriodsUpdate(t *testing.T) {
	periods := &fakePeriodStore{}
	rec := adminCall(t, NewPeriods(periods).Update(), http.MethodPatch, "/api/admin/periods/1", "1",
		`{"label":"Morning","is_break":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body = %s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	if periods.patched.Label == nil || *periods.patched.Label != "Morning" {
		t.Errorf("label = %v, want Morning", periods.patched.Label)
	}
	if periods.patched.IsBreak == nil || !*periods.patched.IsBreak {
		t.Errorf("is_break = %v, want true", periods.patched.IsBreak)
	}
}

func TestPeriodsUpdateRejectsOverlap(t *testing.T) {
	periods := &fakePeriodStore{updateErr: errors.New("store: this period overlaps another one: boom")}
	rec := adminCall(t, NewPeriods(periods).Update(), http.MethodPatch, "/api/admin/periods/"+
		strconv.Itoa(1), "1", `{"start_time":"09:30"}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
