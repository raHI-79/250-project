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
