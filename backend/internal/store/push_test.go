package store

import (
	"testing"
	"time"
)

func TestPushSubscriptionsRoundTrip(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	student := insertUser(t, ctx, pool, "student", "2023331300", StatusActive, 0)
	insertStudent(t, ctx, pool, student, "2023331300", false)
	teacher := insertUser(t, ctx, pool, "teacher", "pushteacher", StatusActive, 0)
	insertPublicTeacher(t, ctx, pool, teacher, "T-300")
	first := PushSubscription{
		UserID:    student,
		Endpoint:  "https://push.example/one",
		P256dh:    "key-one",
		Auth:      "secret-one",
		UserAgent: "Chrome on Android",
	}
	id, err := store.SavePushSubscription(ctx, first)
	if err != nil {
		t.Fatalf("SavePushSubscription() returned an unexpected error: %v", err)
	}
	if id == 0 {
		t.Fatal("SavePushSubscription() returned no id")
	}
	first.P256dh = "key-one-rotated"
	again, err := store.SavePushSubscription(ctx, first)
	if err != nil {
		t.Fatalf("SavePushSubscription() returned an unexpected error: %v", err)
	}
	if again != id {
		t.Errorf("the same endpoint made id %d, want %d", again, id)
	}
	if _, err := store.SavePushSubscription(ctx, PushSubscription{
		UserID: teacher, Endpoint: "https://push.example/two", P256dh: "key-two", Auth: "secret-two",
	}); err != nil {
		t.Fatalf("SavePushSubscription() returned an unexpected error: %v", err)
	}
	rows, err := store.ListPushSubscriptions(ctx, []int64{student, teacher})
	if err != nil {
		t.Fatalf("ListPushSubscriptions() returned an unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d subscriptions, want 2", len(rows))
	}
	for _, row := range rows {
		if row.UserID == student && row.P256dh != "key-one-rotated" {
			t.Errorf("student key = %q, want the rotated key", row.P256dh)
		}
	}
	if err := store.TouchPushSubscription(ctx, "https://push.example/one", time.Now()); err != nil {
		t.Fatalf("TouchPushSubscription() returned an unexpected error: %v", err)
	}
	var used int
	query := "SELECT COUNT(*) FROM push_subscriptions WHERE last_used_at IS NOT NULL"
	if err := pool.QueryRow(ctx, query).Scan(&used); err != nil {
		t.Fatalf("could not read back the subscriptions: %v", err)
	}
	if used != 1 {
		t.Errorf("%d subscriptions carry a last use, want 1", used)
	}
	if err := store.DeletePushSubscription(ctx, teacher, "https://push.example/one"); err != nil {
		t.Fatalf("DeletePushSubscription() returned an unexpected error: %v", err)
	}
	if rows, err := store.ListPushSubscriptions(ctx, []int64{student}); err != nil || len(rows) != 1 {
		t.Errorf("another account deleted the subscription: %d rows, %v", len(rows), err)
	}
	if err := store.DeletePushSubscription(ctx, student, "https://push.example/one"); err != nil {
		t.Fatalf("DeletePushSubscription() returned an unexpected error: %v", err)
	}
	if err := store.DropPushSubscription(ctx, "https://push.example/two"); err != nil {
		t.Fatalf("DropPushSubscription() returned an unexpected error: %v", err)
	}
	left, err := store.ListPushSubscriptions(ctx, []int64{student, teacher})
	if err != nil {
		t.Fatalf("ListPushSubscriptions() returned an unexpected error: %v", err)
	}
	if len(left) != 0 {
		t.Errorf("%d subscriptions stayed, want none", len(left))
	}
}

func TestNotificationRecipientsSkipTheOffice(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	student := insertUser(t, ctx, pool, "student", "2023331310", StatusActive, 0)
	insertStudent(t, ctx, pool, student, "2023331310", false)
	cr := insertUser(t, ctx, pool, "student", "2023331311", StatusActive, 0)
	insertStudent(t, ctx, pool, cr, "2023331311", true)
	teacher := insertUser(t, ctx, pool, "teacher", "recipientteacher", StatusActive, 0)
	insertPublicTeacher(t, ctx, pool, teacher, "T-310")
	admin := insertUser(t, ctx, pool, "admin", "office", StatusActive, 0)
	removed := insertUser(t, ctx, pool, "student", "2023331312", StatusRemoved, 0)
	insertStudent(t, ctx, pool, removed, "2023331312", false)
	tests := []struct {
		name    string
		role    string
		batch   string
		section string
		want    []int64
	}{
		{name: "every account but the office", want: []int64{student, cr, teacher}},
		{name: "the students of a batch", role: "student", batch: "CSE-23", want: []int64{student, cr}},
		{name: "one section", role: "student", batch: "CSE-23", section: "A", want: []int64{student, cr}},
		{name: "another batch", role: "student", batch: "CSE-19"},
		{name: "the teachers", role: "teacher", want: []int64{teacher}},
		{name: "the class representatives", role: "cr", want: []int64{cr}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := store.ListNotificationRecipients(ctx, test.role, test.batch, test.section)
			if err != nil {
				t.Fatalf("ListNotificationRecipients() returned an unexpected error: %v", err)
			}
			if len(got) != len(test.want) {
				t.Fatalf("recipients = %v, want %v", got, test.want)
			}
			for index, want := range test.want {
				if got[index] != want {
					t.Errorf("recipients[%d] = %d, want %d", index, got[index], want)
				}
			}
			for _, id := range got {
				if id == admin || id == removed {
					t.Errorf("recipients hold %d, want the office and removed accounts left out", id)
				}
			}
		})
	}
}
