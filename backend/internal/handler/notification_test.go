package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
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
