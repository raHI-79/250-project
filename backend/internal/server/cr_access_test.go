package server

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type emptyCRStore struct{}

func (emptyCRStore) GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error) {
	return store.StudentProfile{UserID: userID, Batch: "CSE-23", Section: "A", IsCR: true}, nil
}

func (emptyCRStore) ListSectionRoutines(ctx context.Context, batch, section string) ([]store.RoutineRow, error) {
	return nil, nil
}

func (emptyCRStore) ListBatchCourseSlots(ctx context.Context, batch, section string) ([]store.CourseSlotRow, error) {
	return nil, nil
}

func (emptyCRStore) ListRoomAvailability(ctx context.Context, from, to string) ([]store.RoomSlot, error) {
	return nil, nil
}

func (emptyCRStore) TeacherOfCourse(ctx context.Context, batch, section string, courseID int64) (int64, error) {
	return 0, nil
}

func (emptyCRStore) CreateCRItem(ctx context.Context, item store.NewScheduleItem) (store.ScheduleItem, error) {
	return store.ScheduleItem{}, store.ErrCourseNotFound
}

func (emptyCRStore) ListBatchItems(ctx context.Context, batch, from string) ([]store.ScheduleItem, error) {
	return nil, nil
}

func (emptyCRStore) CRRequestReschedule(ctx context.Context, itemID int64, batch string,
	proposal store.Proposal) (store.ScheduleItem, error) {
	return store.ScheduleItem{}, store.ErrItemNotFound
}

func (emptyCRStore) UpdateItemAsCR(ctx context.Context, itemID int64, batch string,
	patch store.ItemPatch) (store.ScheduleItem, error) {
	return store.ScheduleItem{}, store.ErrItemNotFound
}

func (emptyCRStore) CancelItemAsCR(ctx context.Context, itemID int64, batch, reason string) (store.ScheduleItem, error) {
	return store.ScheduleItem{}, store.ErrItemNotFound
}

func (emptyCRStore) DeleteCRItem(ctx context.Context, itemID int64, batch string) (store.ScheduleItem, error) {
	return store.ScheduleItem{}, store.ErrItemNotFound
}

func (emptyCRStore) SetItemFinal(ctx context.Context, itemID int64, batch string,
	inFinal bool) (store.ScheduleItem, error) {
	return store.ScheduleItem{}, store.ErrItemNotFound
}

func (emptyCRStore) ListFinalItems(ctx context.Context, batch string) ([]store.ScheduleItem, error) {
	return nil, nil
}

func (emptyCRStore) ListItemsByIDs(ctx context.Context, batch string, ids []int64) ([]store.ScheduleItem, error) {
	return nil, nil
}

func (emptyCRStore) PublishFinal(ctx context.Context, batch string, publishedBy int64,
	ids []int64) (store.Publication, []store.ScheduleItem, error) {
	return store.Publication{}, nil, store.ErrItemNotFound
}

type emptyItemStore struct{}

func (emptyItemStore) UpdateItemLink(ctx context.Context, itemID, userID int64, link string) (store.ScheduleItem, error) {
	return store.ScheduleItem{}, store.ErrItemNotFound
}

func (emptyItemStore) ListActiveCRs(ctx context.Context, batch, section string) ([]int64, error) {
	return nil, nil
}

type silentNotifier struct{}

func (silentNotifier) NotifyUser(ctx context.Context, userID int64, kind, title, body string,
	link store.NotificationLink) error {
	return nil
}

func (silentNotifier) NotifyBroadcast(ctx context.Context, role, batch, section, kind, title, body string,
	link store.NotificationLink) error {
	return nil
}

type crRoute struct {
	method    string
	path      string
	body      string
	wantForCR int
}

func crRoutes() []crRoute {
	return []crRoute{
		{method: http.MethodGet, path: "/api/cr/routine", wantForCR: http.StatusOK},
		{method: http.MethodGet, path: "/api/cr/courses", wantForCR: http.StatusOK},
		{method: http.MethodGet, path: "/api/cr/room-availability", wantForCR: http.StatusOK},
		{method: http.MethodGet, path: "/api/cr/upcoming", wantForCR: http.StatusOK},
		{method: http.MethodGet, path: "/api/cr/final", wantForCR: http.StatusOK},
		{method: http.MethodGet, path: "/api/cr/final/preview?ids=1", wantForCR: http.StatusOK},
		{method: http.MethodPost, path: "/api/cr/items", body: `{`, wantForCR: http.StatusBadRequest},
		{method: http.MethodPatch, path: "/api/cr/items/1", body: `{`, wantForCR: http.StatusBadRequest},
		{method: http.MethodDelete, path: "/api/cr/items/1", wantForCR: http.StatusNotFound},
		{method: http.MethodPost, path: "/api/cr/items/1/cancel", body: `{}`, wantForCR: http.StatusNotFound},
		{method: http.MethodPost, path: "/api/cr/items/1/final", wantForCR: http.StatusNotFound},
		{method: http.MethodDelete, path: "/api/cr/items/1/final", wantForCR: http.StatusNotFound},
		{method: http.MethodPost, path: "/api/cr/final/publish", body: `{"item_ids":[1]}`, wantForCR: http.StatusNotFound},
	}
}

func TestCRRoutesNeedACRToken(t *testing.T) {
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
			name:       "with a teacher token",
			token:      issue(t, deps, accessTeacherID, auth.RoleTeacher),
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
	crToken := issue(t, deps, accessCRID, auth.RoleCR)
	for _, route := range crRoutes() {
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
		t.Run(route.method+" "+route.path+" with a class representative token", func(t *testing.T) {
			rec := serve(t, router, route.method, route.path, route.body, crToken)
			if rec.Code != route.wantForCR {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, route.wantForCR, rec.Body.String())
			}
		})
	}
}

func serve(t *testing.T, router http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	request := httptest.NewRequest(method, path, nil)
	if body != "" {
		reader = strings.NewReader(body)
		request = httptest.NewRequest(method, path, reader)
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, request)
	return rec
}
