package store

import (
	"context"
	"sort"
	"testing"
	"time"
)

const (
	directToFirstStudent = "Direct to the first student"
	sectionBroadcast     = "Section A students"
	crBroadcast          = "Section A class representatives"
	batchBroadcast       = "Whole batch"
	teacherBroadcast     = "All teachers"
)

func seedNotifications(t *testing.T, ctx context.Context, store *Store, firstStudent int64) {
	t.Helper()
	err := store.CreateDirectNotification(ctx, firstStudent, "notice", directToFirstStudent, "only for this user",
		NotificationLink{Kind: "notice", Ref: 4})
	if err != nil {
		t.Fatalf("CreateDirectNotification() returned an unexpected error: %v", err)
	}
	broadcasts := []struct {
		role    string
		batch   string
		section string
		title   string
	}{
		{role: "student", batch: "CSE-23", section: "A", title: sectionBroadcast},
		{role: "cr", batch: "CSE-23", section: "A", title: crBroadcast},
		{role: "student", batch: "CSE-23", title: batchBroadcast},
		{role: "teacher", title: teacherBroadcast},
	}
	for _, broadcast := range broadcasts {
		err := store.CreateBroadcastNotification(ctx, broadcast.role, broadcast.batch, broadcast.section,
			"notice", broadcast.title, "", NotificationLink{Kind: "routine", Date: "2026-10-22"})
		if err != nil {
			t.Fatalf("CreateBroadcastNotification(%q) returned an unexpected error: %v", broadcast.title, err)
		}
	}
}

func TestNotificationAudience(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	firstStudent := insertAccount(t, ctx, pool, "student", "2023331001", "CSE-23", "A", false)
	sectionCR := insertAccount(t, ctx, pool, "student", "2023331002", "CSE-23", "A", true)
	otherSection := insertAccount(t, ctx, pool, "student", "2023331003", "CSE-23", "B", false)
	otherBatch := insertAccount(t, ctx, pool, "student", "2022331004", "CSE-22", "A", false)
	teacher := insertAccount(t, ctx, pool, "teacher", "T-001", "", "", false)
	seedNotifications(t, ctx, store, firstStudent)
	tests := []struct {
		name       string
		userID     int64
		wantTitles []string
	}{
		{
			name:       "student of section A",
			userID:     firstStudent,
			wantTitles: []string{directToFirstStudent, sectionBroadcast, batchBroadcast},
		},
		{
			name:       "class representative of section A",
			userID:     sectionCR,
			wantTitles: []string{crBroadcast, sectionBroadcast, batchBroadcast},
		},
		{
			name:       "student of section B",
			userID:     otherSection,
			wantTitles: []string{batchBroadcast},
		},
		{
			name:       "student of another batch",
			userID:     otherBatch,
			wantTitles: []string{},
		},
		{
			name:       "teacher",
			userID:     teacher,
			wantTitles: []string{teacherBroadcast},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			notifications, err := store.ListNotifications(ctx, test.userID, 30)
			if err != nil {
				t.Fatalf("ListNotifications() returned an unexpected error: %v", err)
			}
			titles := make([]string, 0, len(notifications))
			for _, notification := range notifications {
				titles = append(titles, notification.Title)
			}
			sort.Strings(titles)
			if len(titles) != len(test.wantTitles) {
				t.Fatalf("titles = %v, want %v", titles, test.wantTitles)
			}
			for i, title := range test.wantTitles {
				if titles[i] != title {
					t.Fatalf("titles = %v, want %v", titles, test.wantTitles)
				}
			}
		})
	}
}

func TestNotificationLimitAndCount(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	student := insertAccount(t, ctx, pool, "student", "2023331005", "CSE-23", "A", false)
	seedNotifications(t, ctx, store, student)
	limited, err := store.ListNotifications(ctx, student, 1)
	if err != nil {
		t.Fatalf("ListNotifications() returned an unexpected error: %v", err)
	}
	if len(limited) != 1 {
		t.Fatalf("got %d notifications, want 1 because of the limit", len(limited))
	}
	all, err := store.ListNotifications(ctx, student, 30)
	if err != nil {
		t.Fatalf("ListNotifications() returned an unexpected error: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("got %d notifications, want 3", len(all))
	}
	for i := 1; i < len(all); i++ {
		if all[i].CreatedAt.After(all[i-1].CreatedAt) {
			t.Errorf("notification %d is newer than the one before it, want the newest first", i)
		}
	}
	tests := []struct {
		name  string
		since time.Time
		want  int
	}{
		{name: "from the epoch", since: time.Unix(0, 0).UTC(), want: 3},
		{name: "from an hour ago", since: time.Now().Add(-time.Hour), want: 3},
		{name: "from an hour ahead", since: time.Now().Add(time.Hour), want: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			count, err := store.CountNotificationsSince(ctx, student, test.since)
			if err != nil {
				t.Fatalf("CountNotificationsSince() returned an unexpected error: %v", err)
			}
			if count != test.want {
				t.Errorf("count = %d, want %d", count, test.want)
			}
		})
	}
}

func TestNotificationReadState(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	student := insertAccount(t, ctx, pool, "student", "2023331006", "CSE-23", "A", false)
	other := insertAccount(t, ctx, pool, "student", "2023331007", "CSE-22", "A", false)
	seedNotifications(t, ctx, store, student)
	all, err := store.ListNotifications(ctx, student, 30)
	if err != nil || len(all) != 3 {
		t.Fatalf("ListNotifications() = %d, %v, want 3", len(all), err)
	}
	for _, entry := range all {
		if entry.Read {
			t.Fatalf("the notification %d started as read", entry.ID)
		}
	}
	unread, err := store.CountUnreadNotifications(ctx, student)
	if err != nil || unread != 3 {
		t.Fatalf("CountUnreadNotifications() = %d, %v, want 3", unread, err)
	}
	if err := store.MarkNotificationsRead(ctx, student, []int64{all[0].ID}); err != nil {
		t.Fatalf("MarkNotificationsRead() returned an unexpected error: %v", err)
	}
	if err := store.MarkNotificationsRead(ctx, student, []int64{all[0].ID}); err != nil {
		t.Fatalf("marking the same notification twice failed: %v", err)
	}
	unread, err = store.CountUnreadNotifications(ctx, student)
	if err != nil || unread != 2 {
		t.Fatalf("after one read CountUnreadNotifications() = %d, %v, want 2", unread, err)
	}
	after, err := store.ListNotifications(ctx, student, 30)
	if err != nil || !after[0].Read || after[1].Read {
		t.Fatalf("the read flags are wrong: %+v, %v", after, err)
	}
	if err := store.MarkNotificationsRead(ctx, other, []int64{all[1].ID, all[2].ID}); err != nil {
		t.Fatalf("MarkNotificationsRead() for a stranger returned an error: %v", err)
	}
	var strangers int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM notification_reads WHERE user_id = $1`, other).
		Scan(&strangers); err != nil {
		t.Fatalf("could not count the reads: %v", err)
	}
	if strangers > 1 {
		t.Fatalf("a stranger marked %d notifications that were never theirs", strangers)
	}
	if err := store.MarkAllNotificationsRead(ctx, student, time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("MarkAllNotificationsRead() returned an unexpected error: %v", err)
	}
	unread, err = store.CountUnreadNotifications(ctx, student)
	if err != nil || unread != 0 {
		t.Fatalf("after reading all CountUnreadNotifications() = %d, %v, want 0", unread, err)
	}
	if err := store.MarkAllNotificationsRead(ctx, student, time.Unix(0, 0)); err != nil {
		t.Fatalf("an older mark returned an error: %v", err)
	}
	unread, err = store.CountUnreadNotifications(ctx, student)
	if err != nil || unread != 0 {
		t.Fatalf("an older mark brought %d notifications back", unread)
	}
}
