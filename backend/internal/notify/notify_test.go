package notify

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/store"
	"testing"
	"time"
)

type directCall struct {
	userID int64
	kind   string
	title  string
	body   string
	link   store.NotificationLink
}

type broadcastCall struct {
	role    string
	batch   string
	section string
	kind    string
	title   string
	body    string
	link    store.NotificationLink
}

type fakeStore struct {
	direct     []directCall
	broadcasts []broadcastCall
	listed     []store.Notification
	count      int
	err        error
	askedLimit int
	askedUser  int64
	askedSince time.Time
	recipients []int64
}

func (f *fakeStore) ListNotificationRecipients(ctx context.Context, role, batch,
	section string) ([]int64, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.recipients, nil
}

func (f *fakeStore) CreateDirectNotification(ctx context.Context, userID int64, kind, title, body string,
	link store.NotificationLink) error {
	if f.err != nil {
		return f.err
	}
	f.direct = append(f.direct, directCall{userID: userID, kind: kind, title: title, body: body, link: link})
	return nil
}

func (f *fakeStore) CreateBroadcastNotification(ctx context.Context, role, batch, section, kind, title, body string,
	link store.NotificationLink) error {
	if f.err != nil {
		return f.err
	}
	f.broadcasts = append(f.broadcasts, broadcastCall{
		role: role, batch: batch, section: section, kind: kind, title: title, body: body, link: link,
	})
	return nil
}

func (f *fakeStore) ListNotifications(ctx context.Context, userID int64, limit int) ([]store.Notification, error) {
	f.askedUser, f.askedLimit = userID, limit
	if f.err != nil {
		return nil, f.err
	}
	return f.listed, nil
}

func (f *fakeStore) CountNotificationsSince(ctx context.Context, userID int64, since time.Time) (int, error) {
	f.askedUser, f.askedSince = userID, since
	if f.err != nil {
		return 0, f.err
	}
	return f.count, nil
}

func TestNotifyUser(t *testing.T) {
	tests := []struct {
		name     string
		userID   int64
		kind     string
		title    string
		body     string
		storeErr error
		wantErr  error
		wantCall directCall
	}{
		{
			name:     "stored",
			userID:   7,
			kind:     TypeDecision,
			title:    "  Request approved  ",
			body:     " Monday 10:30 ",
			wantCall: directCall{userID: 7, kind: TypeDecision, title: "Request approved", body: "Monday 10:30"},
		},
		{name: "zero user", userID: 0, kind: TypeNotice, title: "Hello", wantErr: ErrInvalidRecipient},
		{name: "negative user", userID: -3, kind: TypeNotice, title: "Hello", wantErr: ErrInvalidRecipient},
		{name: "missing type", userID: 7, kind: " ", title: "Hello", wantErr: ErrInvalidContent},
		{name: "missing title", userID: 7, kind: TypeNotice, title: "  ", wantErr: ErrInvalidContent},
		{name: "store failure", userID: 7, kind: TypeNotice, title: "Hello", storeErr: errors.New("database is down")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			notifications := &fakeStore{err: test.storeErr}
			err := New(notifications).NotifyUser(context.Background(), test.userID, test.kind, test.title,
				test.body, Page(LinkCRUpcoming))
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("error = %v, want %v", err, test.wantErr)
				}
				if len(notifications.direct) != 0 {
					t.Error("the store was written to although the call was rejected")
				}
				return
			}
			if test.storeErr != nil {
				if err == nil {
					t.Fatal("the store failure was swallowed")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			want := test.wantCall
			want.link = Page(LinkCRUpcoming)
			if len(notifications.direct) != 1 || notifications.direct[0] != want {
				t.Errorf("call = %+v, want %+v", notifications.direct, want)
			}
		})
	}
}

func TestNotifyBroadcast(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		batch    string
		section  string
		kind     string
		title    string
		wantErr  error
		wantCall broadcastCall
	}{
		{
			name:     "role and batch",
			role:     "Student",
			batch:    "cse-23",
			section:  "a",
			kind:     TypeClass,
			title:    "Class approved",
			wantCall: broadcastCall{role: "student", batch: "CSE-23", section: "A", kind: TypeClass, title: "Class approved"},
		},
		{
			name:     "whole batch",
			role:     "student",
			batch:    "CSE-23",
			kind:     TypeNotice,
			title:    "New notice",
			wantCall: broadcastCall{role: "student", batch: "CSE-23", kind: TypeNotice, title: "New notice"},
		},
		{
			name:     "role only",
			role:     "teacher",
			kind:     TypeRequest,
			title:    "New request",
			wantCall: broadcastCall{role: "teacher", kind: TypeRequest, title: "New request"},
		},
		{
			name:     "every batch for the students",
			role:     "student",
			kind:     TypeNotice,
			title:    "New notice",
			wantCall: broadcastCall{role: "student", kind: TypeNotice, title: "New notice"},
		},
		{
			name:     "batch only",
			batch:    "CSE-23",
			kind:     TypeNotice,
			title:    "New notice",
			wantCall: broadcastCall{batch: "CSE-23", kind: TypeNotice, title: "New notice"},
		},
		{name: "no target", kind: TypeNotice, title: "New notice", wantErr: ErrInvalidRecipient},
		{name: "section without a batch", role: "student", section: "A", kind: TypeNotice, title: "New notice", wantErr: ErrInvalidRecipient},
		{name: "unknown role", role: "guest", batch: "CSE-23", kind: TypeNotice, title: "New notice", wantErr: ErrInvalidRole},
		{name: "missing title", role: "student", batch: "CSE-23", kind: TypeNotice, wantErr: ErrInvalidContent},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			notifications := &fakeStore{}
			err := New(notifications).NotifyBroadcast(context.Background(), test.role, test.batch,
				test.section, test.kind, test.title, "", Page(LinkRoutine))
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("error = %v, want %v", err, test.wantErr)
				}
				if len(notifications.broadcasts) != 0 {
					t.Error("the store was written to although the call was rejected")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			want := test.wantCall
			want.link = Page(LinkRoutine)
			if len(notifications.broadcasts) != 1 || notifications.broadcasts[0] != want {
				t.Errorf("call = %+v, want %+v", notifications.broadcasts, want)
			}
		})
	}
}

func TestRecentClampsTheLimit(t *testing.T) {
	tests := []struct {
		name      string
		limit     int
		wantLimit int
	}{
		{name: "zero", limit: 0, wantLimit: DefaultLimit},
		{name: "negative", limit: -5, wantLimit: DefaultLimit},
		{name: "small", limit: 5, wantLimit: 5},
		{name: "maximum", limit: MaxLimit, wantLimit: MaxLimit},
		{name: "over the maximum", limit: 500, wantLimit: MaxLimit},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			notifications := &fakeStore{listed: []store.Notification{{ID: 1, Type: TypeNotice, Title: "New notice"}}}
			got, err := New(notifications).Recent(context.Background(), 9, test.limit)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != 1 {
				t.Fatalf("got %d notifications, want 1", len(got))
			}
			if notifications.askedLimit != test.wantLimit {
				t.Errorf("the store was asked for %d rows, want %d", notifications.askedLimit, test.wantLimit)
			}
			if notifications.askedUser != 9 {
				t.Errorf("the store was asked for user %d, want 9", notifications.askedUser)
			}
		})
	}
}

func TestUnreadCount(t *testing.T) {
	since := time.Date(2026, 10, 16, 9, 30, 0, 0, time.UTC)
	notifications := &fakeStore{count: 4}
	count, err := New(notifications).UnreadCount(context.Background(), 9, since)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 4 {
		t.Errorf("count = %d, want 4", count)
	}
	if !notifications.askedSince.Equal(since) {
		t.Errorf("the store was asked for %v, want %v", notifications.askedSince, since)
	}
	failing := &fakeStore{err: errors.New("database is down")}
	if _, err := New(failing).UnreadCount(context.Background(), 9, since); err == nil {
		t.Error("the store failure was swallowed")
	}
}
