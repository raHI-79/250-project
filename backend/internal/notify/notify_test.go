package notify

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/push"
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
	audience   []store.NotificationRecipient
	roles      map[int64]string
}

func (f *fakeStore) ListNotificationRecipients(ctx context.Context, role, batch,
	section string) ([]int64, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.recipients, nil
}

func (f *fakeStore) ListNotificationAudience(ctx context.Context, role, batch,
	section string) ([]store.NotificationRecipient, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.audience != nil {
		return f.audience, nil
	}
	audience := make([]store.NotificationRecipient, 0, len(f.recipients))
	for _, userID := range f.recipients {
		audience = append(audience, store.NotificationRecipient{UserID: userID, Role: auth.RoleStudent})
	}
	return audience, nil
}

func (f *fakeStore) NotificationRole(ctx context.Context, userID int64) (string, error) {
	if role, ok := f.roles[userID]; ok {
		return role, nil
	}
	return auth.RoleStudent, nil
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

type recordedPush struct {
	userIDs []int64
	payload push.Payload
}

type fakePusher struct {
	calls []recordedPush
}

func (f *fakePusher) Queue(userIDs []int64, payload push.Payload) {
	f.calls = append(f.calls, recordedPush{userIDs: append([]int64(nil), userIDs...), payload: payload})
}

func TestPageURLPointsAtTheRoutesTheAppKnows(t *testing.T) {
	const origin = "https://portal.example/"
	tests := []struct {
		name string
		role string
		link store.NotificationLink
		want string
	}{
		{
			name: "a teacher decision",
			role: auth.RoleTeacher,
			link: store.NotificationLink{Kind: LinkTeacherNextDay, Ref: 12},
			want: "https://portal.example/teacher/upcoming-routine?item=12&tab=next-day",
		},
		{
			name: "a final routine for the representative",
			role: auth.RoleCR,
			link: store.NotificationLink{Kind: LinkCRFinal, Ref: 7},
			want: "https://portal.example/cr/create-class?item=7&tab=final",
		},
		{
			name: "a notice for a student",
			role: auth.RoleStudent,
			link: store.NotificationLink{Kind: LinkNotice, Ref: 3},
			want: "https://portal.example/student/notices?highlight=3",
		},
		{
			name: "the same notice for a teacher",
			role: auth.RoleTeacher,
			link: store.NotificationLink{Kind: LinkNotice, Ref: 3},
			want: "https://portal.example/teacher/notices?highlight=3",
		},
		{
			name: "a message for a teacher",
			role: auth.RoleTeacher,
			link: store.NotificationLink{Kind: LinkMessage, Ref: 9},
			want: "https://portal.example/teacher/messages?c=9",
		},
		{
			name: "a poll for the representative",
			role: auth.RoleCR,
			link: store.NotificationLink{Kind: LinkPoll, Ref: 4},
			want: "https://portal.example/cr/polls?highlight=4",
		},
		{
			name: "a published routine",
			role: auth.RoleStudent,
			link: store.NotificationLink{Kind: LinkRoutine, Date: "2026-03-09"},
			want: "https://portal.example/student/upcoming-routine?date=2026-03-09",
		},
		{
			name: "an unknown kind falls back to the home page",
			role: auth.RoleStudent,
			link: store.NotificationLink{Kind: "nothing"},
			want: "https://portal.example",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := PageURL(origin, test.role, test.link); got != test.want {
				t.Errorf("PageURL = %q, want %q", got, test.want)
			}
		})
	}
}

func TestNotifyBroadcastResolvesTheAudienceToEachRole(t *testing.T) {
	notifications := &fakeStore{audience: []store.NotificationRecipient{
		{UserID: 4, Role: auth.RoleStudent},
		{UserID: 5, Role: auth.RoleCR},
		{UserID: 6, Role: auth.RoleStudent},
		{UserID: 7, Role: auth.RoleTeacher},
	}}
	pusher := &fakePusher{}
	service := New(notifications).WithPush(pusher, "https://portal.example")
	err := service.NotifyBroadcast(context.Background(), "", "CSE-23", "", TypeNotice,
		"New notice", "Quiz on Monday", Ref(LinkNotice, 11))
	if err != nil {
		t.Fatalf("NotifyBroadcast returned %v", err)
	}
	if len(pusher.calls) != 2 {
		t.Fatalf("queued %d payloads, want one per page", len(pusher.calls))
	}
	byURL := make(map[string][]int64)
	for _, call := range pusher.calls {
		byURL[call.payload.URL] = append(byURL[call.payload.URL], call.userIDs...)
		if call.payload.Title != "New notice" || call.payload.Tag != TypeNotice {
			t.Errorf("payload = %+v, want the broadcast title and tag", call.payload)
		}
	}
	student := "https://portal.example/student/notices?highlight=11"
	teacher := "https://portal.example/teacher/notices?highlight=11"
	if got := byURL[student]; len(got) != 3 {
		t.Errorf("students = %v, want both students and the representative on %s", got, student)
	}
	if got := byURL[teacher]; len(got) != 1 || got[0] != 7 {
		t.Errorf("teachers = %v, want only account 7 on %s", got, teacher)
	}
	if len(byURL) != 2 {
		t.Errorf("urls = %v, want one payload per distinct page", byURL)
	}
}

func TestGroupByRoleSkipsEmptyRecipients(t *testing.T) {
	grouped := GroupByRole([]store.NotificationRecipient{
		{UserID: 0, Role: auth.RoleStudent},
		{UserID: 3, Role: auth.RoleStudent},
		{UserID: 4, Role: auth.RoleTeacher},
	})
	if len(grouped[auth.RoleStudent]) != 1 || grouped[auth.RoleStudent][0] != 3 {
		t.Errorf("students = %v, want only account 3", grouped[auth.RoleStudent])
	}
	if len(grouped[auth.RoleTeacher]) != 1 {
		t.Errorf("teachers = %v, want one account", grouped[auth.RoleTeacher])
	}
}
