package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
	"time"
)

func insertClassEvent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, courseID, teacherID, createdBy int64, date, status string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(ctx,
		`INSERT INTO class_events
(batch, section, course_id, teacher_id, event_date, start_time, end_time, room, event_type, status, created_by)
VALUES ('CSE-23', 'A', $1, $2, $3::date, '10:30', '12:45', '1003', 'class', $4, $5)
RETURNING id`,
		courseID, teacherID, date, status, createdBy).Scan(&id)
	if err != nil {
		t.Fatalf("could not insert a class event: %v", err)
	}
	return id
}

func eventStatus(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id int64) (string, *time.Time) {
	t.Helper()
	var status string
	var decidedAt *time.Time
	if err := pool.QueryRow(ctx, "SELECT status, decided_at FROM class_events WHERE id = $1", id).Scan(&status, &decidedAt); err != nil {
		t.Fatalf("could not read the status of the event %d: %v", id, err)
	}
	return status, decidedAt
}

func TestAutoCancelStaleRequests(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	teacherUser := insertAccount(t, ctx, pool, "teacher", "T-001", "", "", false)
	courseID, teacherID := seedCourseAndTeacher(t, ctx, store, pool)
	representative := insertAccount(t, ctx, pool, "student", "2023331001", "CSE-23", "A", true)
	now := time.Now()
	today := now.Format(dateLayout)
	yesterday := now.AddDate(0, 0, -1).Format(dateLayout)
	tomorrow := now.AddDate(0, 0, 1).Format(dateLayout)
	pendingToday := insertClassEvent(t, ctx, pool, courseID, teacherID, representative, today, EventStatusPending)
	pendingYesterday := insertClassEvent(t, ctx, pool, courseID, teacherID, representative, yesterday, EventStatusPending)
	pendingTomorrow := insertClassEvent(t, ctx, pool, courseID, teacherID, representative, tomorrow, EventStatusPending)
	approvedToday := insertClassEvent(t, ctx, pool, courseID, teacherID, representative, today, EventStatusApproved)
	cancelled, err := store.AutoCancelStaleRequests(ctx, now)
	if err != nil {
		t.Fatalf("AutoCancelStaleRequests() returned an unexpected error: %v", err)
	}
	if len(cancelled) != 2 {
		t.Fatalf("%d requests were cancelled, want 2", len(cancelled))
	}
	for _, request := range cancelled {
		if request.CreatedBy != representative || request.CourseCode != "CSE-233" {
			t.Errorf("cancelled request = %+v, want the class representative and the course code", request)
		}
	}
	tests := []struct {
		name        string
		id          int64
		wantStatus  string
		wantDecided bool
	}{
		{name: "pending today", id: pendingToday, wantStatus: EventStatusAutoCancelled, wantDecided: true},
		{name: "pending yesterday", id: pendingYesterday, wantStatus: EventStatusAutoCancelled, wantDecided: true},
		{name: "pending tomorrow", id: pendingTomorrow, wantStatus: EventStatusPending},
		{name: "approved today", id: approvedToday, wantStatus: EventStatusApproved},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status, decidedAt := eventStatus(t, ctx, pool, test.id)
			if status != test.wantStatus {
				t.Errorf("status = %q, want %q", status, test.wantStatus)
			}
			if test.wantDecided && decidedAt == nil {
				t.Error("decided_at is empty, want the cancellation time")
			}
			if !test.wantDecided && decidedAt != nil {
				t.Error("decided_at is set, want it empty")
			}
		})
	}
	again, err := store.AutoCancelStaleRequests(ctx, now)
	if err != nil {
		t.Fatalf("AutoCancelStaleRequests() returned an unexpected error: %v", err)
	}
	if len(again) != 0 {
		t.Errorf("%d requests were cancelled twice, want none", len(again))
	}
	teacher, err := store.GetTeacherByUser(ctx, teacherUser)
	if err != nil {
		t.Fatalf("GetTeacherByUser() returned an unexpected error: %v", err)
	}
	if teacher.ID != teacherID {
		t.Errorf("teacher id = %d, want %d", teacher.ID, teacherID)
	}
	if _, err := store.GetTeacherByUser(ctx, representative); !errors.Is(err, ErrNotFound) {
		t.Errorf("error for a student = %v, want %v", err, ErrNotFound)
	}
}

func TestDecideAndCancelClassEvents(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	insertAccount(t, ctx, pool, "teacher", "T-001", "", "", false)
	courseID, teacherID := seedCourseAndTeacher(t, ctx, store, pool)
	otherTeacherUser := insertAccount(t, ctx, pool, "teacher", "T-002", "", "", false)
	var otherTeacherID int64
	if err := pool.QueryRow(ctx, "SELECT id FROM teachers WHERE user_id = $1", otherTeacherUser).Scan(&otherTeacherID); err != nil {
		t.Fatalf("could not read the second teacher: %v", err)
	}
	representative := insertAccount(t, ctx, pool, "student", "2023331001", "CSE-23", "A", true)
	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1).Format(dateLayout)
	yesterday := now.AddDate(0, 0, -1).Format(dateLayout)
	request := insertClassEvent(t, ctx, pool, courseID, teacherID, representative, tomorrow, EventStatusPending)
	if _, err := store.DecideRequest(ctx, request, otherTeacherID, EventStatusApproved, now); !errors.Is(err, ErrNotFound) {
		t.Errorf("another teacher's decision error = %v, want %v", err, ErrNotFound)
	}
	decision, err := store.DecideRequest(ctx, request, teacherID, EventStatusApproved, now)
	if err != nil {
		t.Fatalf("DecideRequest() returned an unexpected error: %v", err)
	}
	if decision.CreatedBy != representative || decision.EventDate != tomorrow {
		t.Errorf("decision = %+v, want the class representative and the date", decision)
	}
	if _, err := store.DecideRequest(ctx, request, teacherID, EventStatusRejected, now); !errors.Is(err, ErrNotPending) {
		t.Errorf("second decision error = %v, want %v", err, ErrNotPending)
	}
	stale := insertClassEvent(t, ctx, pool, courseID, teacherID, representative, yesterday, EventStatusPending)
	if _, err := store.DecideRequest(ctx, stale, teacherID, EventStatusApproved, now); !errors.Is(err, ErrTooLateToApprove) {
		t.Errorf("late approval error = %v, want %v", err, ErrTooLateToApprove)
	}
	if _, err := store.DecideRequest(ctx, stale, teacherID, EventStatusRejected, now); err != nil {
		t.Errorf("late rejection returned an unexpected error: %v", err)
	}
	if _, err := store.CancelClassEvent(ctx, request, teacherID, now); err != nil {
		t.Fatalf("CancelClassEvent() returned an unexpected error: %v", err)
	}
	status, decidedAt := eventStatus(t, ctx, pool, request)
	if status != EventStatusCancelled || decidedAt == nil {
		t.Errorf("status = %q with decided_at %v, want cancelled with a time", status, decidedAt)
	}
	if _, err := store.CancelClassEvent(ctx, request, teacherID, now); !errors.Is(err, ErrNotCancellable) {
		t.Errorf("second cancellation error = %v, want %v", err, ErrNotCancellable)
	}
	past := insertClassEvent(t, ctx, pool, courseID, teacherID, representative, yesterday, EventStatusApproved)
	if _, err := store.CancelClassEvent(ctx, past, teacherID, now); !errors.Is(err, ErrNotCancellable) {
		t.Errorf("past cancellation error = %v, want %v", err, ErrNotCancellable)
	}
}

func TestRoutineCancellationsAndSchedule(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	teacherUser := insertAccount(t, ctx, pool, "teacher", "T-001", "", "", false)
	courseID, teacherID := seedCourseAndTeacher(t, ctx, store, pool)
	representative := insertAccount(t, ctx, pool, "student", "2023331001", "CSE-23", "A", true)
	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1)
	date := tomorrow.Format(dateLayout)
	entry, err := store.CreateRoutine(ctx, NewRoutine{
		Batch:     "CSE-23",
		Section:   "A",
		CourseID:  courseID,
		TeacherID: teacherID,
		DayOfWeek: int(tomorrow.Weekday()),
		StartTime: "08:00",
		EndTime:   "09:00",
		Room:      "1003",
		Type:      "class",
	})
	if err != nil {
		t.Fatalf("CreateRoutine() returned an unexpected error: %v", err)
	}
	schedule, err := store.ListTeacherSchedule(ctx, teacherID, date)
	if err != nil {
		t.Fatalf("ListTeacherSchedule() returned an unexpected error: %v", err)
	}
	if len(schedule) != 1 || schedule[0].Kind != BookingSourceFixed || schedule[0].Cancelled {
		t.Fatalf("schedule = %+v, want one fixed entry that is not cancelled", schedule)
	}
	if err := store.CancelRoutineOccurrence(ctx, entry.ID, teacherUser, date); err != nil {
		t.Fatalf("CancelRoutineOccurrence() returned an unexpected error: %v", err)
	}
	if err := store.CancelRoutineOccurrence(ctx, entry.ID, teacherUser, date); !errors.Is(err, ErrAlreadyCancelled) {
		t.Errorf("second cancellation error = %v, want %v", err, ErrAlreadyCancelled)
	}
	schedule, err = store.ListTeacherSchedule(ctx, teacherID, date)
	if err != nil {
		t.Fatalf("ListTeacherSchedule() returned an unexpected error: %v", err)
	}
	if len(schedule) != 1 || !schedule[0].Cancelled {
		t.Errorf("schedule = %+v, want the entry marked as cancelled", schedule)
	}
	plan, err := store.ListDayPlan(ctx, "CSE-23", date)
	if err != nil {
		t.Fatalf("ListDayPlan() returned an unexpected error: %v", err)
	}
	if len(plan) != 1 || !plan[0].Cancelled || plan[0].TeacherName == "" {
		t.Errorf("plan = %+v, want the cancelled entry with its teacher", plan)
	}
	crs, err := store.ListActiveCRs(ctx, "CSE-23", "A")
	if err != nil {
		t.Fatalf("ListActiveCRs() returned an unexpected error: %v", err)
	}
	if len(crs) != 1 || crs[0] != representative {
		t.Errorf("class representatives = %v, want [%d]", crs, representative)
	}
	if _, err := store.GetTeacherRoutine(ctx, entry.ID, teacherID); err != nil {
		t.Errorf("GetTeacherRoutine() returned an unexpected error: %v", err)
	}
	if _, err := store.GetTeacherRoutine(ctx, entry.ID, teacherID+999); !errors.Is(err, ErrNotFound) {
		t.Errorf("error for another teacher = %v, want %v", err, ErrNotFound)
	}
}
