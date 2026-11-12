package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeNotificationService struct {
	notifications []store.Notification
	count         int
	err           error
	askedUser     int64
	askedLimit    int
	askedSince    time.Time
	readIDs       []int64
	readPage      string
	readBefore    time.Time
}

func (f *fakeNotificationService) Recent(ctx context.Context, userID int64, limit int) ([]store.Notification, error) {
	f.askedUser, f.askedLimit = userID, limit
	if f.err != nil {
		return nil, f.err
	}
	return f.notifications, nil
}

func (f *fakeNotificationService) UnreadCount(ctx context.Context, userID int64, since time.Time) (int, error) {
	f.askedUser, f.askedSince = userID, since
	if f.err != nil {
		return 0, f.err
	}
	return f.count, nil
}

func (f *fakeNotificationService) Unread(ctx context.Context, userID int64) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.count, nil
}

func (f *fakeNotificationService) Read(ctx context.Context, userID int64, ids []int64) error {
	f.askedUser = userID
	f.readIDs = append(f.readIDs, ids...)
	if f.err != nil {
		return f.err
	}
	f.count -= len(ids)
	return nil
}

func (f *fakeNotificationService) ReadPage(ctx context.Context, userID int64, page string) (int, error) {
	f.askedUser, f.readPage = userID, page
	if f.err != nil {
		return 0, f.err
	}
	f.count -= 2
	return 2, nil
}

func (f *fakeNotificationService) ReadAll(ctx context.Context, userID int64, before time.Time) error {
	f.askedUser, f.readBefore = userID, before
	if f.err != nil {
		return f.err
	}
	f.count = 0
	return nil
}

func notificationCall(t *testing.T, handlerFunc http.HandlerFunc, target string, user *middleware.User) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if user != nil {
		req = req.WithContext(middleware.WithUser(req.Context(), *user))
	}
	rec := httptest.NewRecorder()
	handlerFunc.ServeHTTP(rec, req)
	return rec
}

func TestNotificationsList(t *testing.T) {
	silenceLog(t)
	viewer := &middleware.User{ID: 21, Identifier: "2023331002", Role: "cr"}
	created := time.Date(2026, 10, 16, 9, 30, 0, 0, time.UTC)
	tests := []struct {
		name       string
		query      string
		user       *middleware.User
		storeErr   error
		wantStatus int
		wantError  string
		wantLimit  int
	}{
		{name: "default limit", query: "", user: viewer, wantStatus: http.StatusOK, wantLimit: notify.DefaultLimit},
		{name: "explicit limit", query: "?limit=5", user: viewer, wantStatus: http.StatusOK, wantLimit: 5},
		{name: "maximum limit", query: "?limit=100", user: viewer, wantStatus: http.StatusOK, wantLimit: notify.MaxLimit},
		{name: "zero limit", query: "?limit=0", user: viewer, wantStatus: http.StatusBadRequest, wantError: MessageInvalidNotificationLimit},
		{name: "limit over the maximum", query: "?limit=101", user: viewer, wantStatus: http.StatusBadRequest, wantError: MessageInvalidNotificationLimit},
		{name: "limit not a number", query: "?limit=many", user: viewer, wantStatus: http.StatusBadRequest, wantError: MessageInvalidNotificationLimit},
		{name: "without a signed in user", query: "", wantStatus: http.StatusUnauthorized, wantError: middleware.MessageUnauthorized},
		{
			name:       "store failure",
			query:      "",
			user:       viewer,
			storeErr:   errors.New("database is down"),
			wantStatus: http.StatusInternalServerError,
			wantError:  "Internal server error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeNotificationService{
				err: test.storeErr,
				notifications: []store.Notification{
					{ID: 4, Type: notify.TypeNotice, Title: "New notice", Body: "Check the board", CreatedAt: created},
				},
			}
			rec := notificationCall(t, NewNotifications(service).List(), "/api/notifications"+test.query, test.user)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			if service.askedUser != viewer.ID || service.askedLimit != test.wantLimit {
				t.Errorf("the service was asked for user %d with limit %d, want %d and %d",
					service.askedUser, service.askedLimit, viewer.ID, test.wantLimit)
			}
			item := items(t, rec)[0]
			if item["id"] != float64(4) || item["type"] != notify.TypeNotice || item["title"] != "New notice" {
				t.Errorf("item = %v, want the id, type and title", item)
			}
			if item["created_at"] != created.Format(time.RFC3339) {
				t.Errorf("created_at = %v, want %q", item["created_at"], created.Format(time.RFC3339))
			}
		})
	}
}

func TestNotificationsUnreadCount(t *testing.T) {
	silenceLog(t)
	viewer := &middleware.User{ID: 21, Identifier: "2023331002", Role: "cr"}
	since := time.Date(2026, 10, 16, 9, 30, 0, 0, time.UTC)
	tests := []struct {
		name       string
		query      string
		user       *middleware.User
		storeErr   error
		wantStatus int
		wantError  string
		wantSince  time.Time
	}{
		{name: "with a since value", query: "?since=2026-10-16T09:30:00Z", user: viewer, wantStatus: http.StatusOK, wantSince: since},
		{name: "without a since value", query: "", user: viewer, wantStatus: http.StatusOK, wantSince: time.Unix(0, 0).UTC()},
		{name: "bad since value", query: "?since=yesterday", user: viewer, wantStatus: http.StatusBadRequest, wantError: MessageInvalidSince},
		{name: "date only", query: "?since=2026-10-16", user: viewer, wantStatus: http.StatusBadRequest, wantError: MessageInvalidSince},
		{name: "without a signed in user", query: "", wantStatus: http.StatusUnauthorized, wantError: middleware.MessageUnauthorized},
		{
			name:       "store failure",
			query:      "",
			user:       viewer,
			storeErr:   errors.New("database is down"),
			wantStatus: http.StatusInternalServerError,
			wantError:  "Internal server error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeNotificationService{count: 3, err: test.storeErr}
			rec := notificationCall(t, NewNotifications(service).UnreadCount(), "/api/notifications/unread-count"+test.query, test.user)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			if got := decodeBody(t, rec)["count"]; got != float64(3) {
				t.Errorf("count = %v, want 3", got)
			}
			if !service.askedSince.Equal(test.wantSince) {
				t.Errorf("the service was asked for %v, want %v", service.askedSince, test.wantSince)
			}
			if service.askedUser != viewer.ID {
				t.Errorf("the service was asked for user %d, want %d", service.askedUser, viewer.ID)
			}
		})
	}
}

func notificationPost(t *testing.T, handlerFunc http.HandlerFunc, target, body string, user *middleware.User) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if user != nil {
		req = req.WithContext(middleware.WithUser(req.Context(), *user))
	}
	rec := httptest.NewRecorder()
	handlerFunc.ServeHTTP(rec, req)
	return rec
}

func TestNotificationsRead(t *testing.T) {
	silenceLog(t)
	viewer := &middleware.User{ID: 21, Identifier: "2023331002", Role: "cr"}
	tests := []struct {
		name       string
		body       string
		user       *middleware.User
		wantStatus int
		wantBody   string
		wantIDs    int
		wantPage   string
	}{
		{name: "one notification", body: `{"id":5}`, user: viewer, wantStatus: http.StatusOK,
			wantBody: `"unread":3`, wantIDs: 1},
		{name: "the notifications of a page", body: `{"page":"/student/notices/7"}`, user: viewer,
			wantStatus: http.StatusOK, wantBody: `"unread":2`, wantPage: "/student/notices/7"},
		{name: "nothing named", body: `{}`, user: viewer, wantStatus: http.StatusBadRequest,
			wantBody: MessageReadNothing},
		{name: "signed out", body: `{"id":5}`, wantStatus: http.StatusUnauthorized,
			wantBody: middleware.MessageUnauthorized},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeNotificationService{count: 4}
			rec := notificationPost(t, NewNotifications(service).Read(), "/api/notifications/read", test.body, test.user)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), test.wantBody) {
				t.Fatalf("body = %q, want it to contain %q", rec.Body.String(), test.wantBody)
			}
			if len(service.readIDs) != test.wantIDs || service.readPage != test.wantPage {
				t.Fatalf("marked ids %v and page %q", service.readIDs, service.readPage)
			}
		})
	}
}

func TestNotificationsReadAll(t *testing.T) {
	silenceLog(t)
	viewer := &middleware.User{ID: 21, Identifier: "2023331002", Role: "cr"}
	service := &fakeNotificationService{count: 4}
	rec := notificationPost(t, NewNotifications(service).ReadAll(), "/api/notifications/read-all",
		`{"before":"2026-10-16T09:30:00+06:00"}`, viewer)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"unread":0`) {
		t.Fatalf("status = %d, body = %q", rec.Code, rec.Body.String())
	}
	if service.readBefore.Year() != 2026 || service.readBefore.Month() != time.October {
		t.Fatalf("read before %v, want the time the phone remembered", service.readBefore)
	}
	rec = notificationPost(t, NewNotifications(service).ReadAll(), "/api/notifications/read-all",
		`{"before":"yesterday"}`, viewer)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("a bad time answered %d", rec.Code)
	}
}
