package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
	"time"
)

type engineFixture struct {
	courseID  int64
	teacherID int64
	teacherDB int64
	crUserID  int64
	routineID int64
}

func seedEngine(t *testing.T, ctx context.Context, store *Store, pool *pgxpool.Pool, day int) engineFixture {
	t.Helper()
	course, err := store.CreateCourse(ctx, "CSE 2243 7637D", "Compilers", 3.0)
	if err != nil {
		t.Fatalf("CreateCourse() returned an unexpected error: %v", err)
	}
	teacherUserID := insertUser(t, ctx, pool, "teacher", "karim.cse", StatusActive, 0)
	var teacherID int64
	query := `INSERT INTO teachers (user_id, teacher_id, name, email, phone)
VALUES ($1, 'karim.cse', 'Engine Teacher', 'engine@sust.edu', '01800000000') RETURNING id`
	if err := pool.QueryRow(ctx, query, teacherUserID).Scan(&teacherID); err != nil {
		t.Fatalf("could not insert the teacher: %v", err)
	}
	crUserID := insertUser(t, ctx, pool, "student", "2023331500", StatusActive, 0)
	insertStudent(t, ctx, pool, crUserID, "2023331500", true)
	setRunningSemester(t, ctx, pool, "CSE-23", "1-1")
	routine := `INSERT INTO routines (batch, semester, section, course_id, teacher_id, day_of_week, start_time, end_time, room, type)
VALUES ('CSE-23', '1-1', 'A', $1, $2, $3, TIME '08:00', TIME '09:30', '1003', 'class') RETURNING id`
	var routineID int64
	if err := pool.QueryRow(ctx, routine, course.ID, teacherID, day).Scan(&routineID); err != nil {
		t.Fatalf("could not insert the routine entry: %v", err)
	}
	return engineFixture{
		courseID:  course.ID,
		teacherID: teacherID,
		teacherDB: teacherUserID,
		crUserID:  crUserID,
		routineID: routineID,
	}
}

func newItem(fixture engineFixture, date, start, end, section string) NewScheduleItem {
	return NewScheduleItem{
		Batch:     "CSE-23",
		Section:   section,
		CourseID:  fixture.courseID,
		TeacherID: fixture.teacherID,
		Kind:      "class",
		ItemDate:  date,
		StartTime: start,
		EndTime:   end,
		Mode:      ModeRoom,
		Room:      "1003",
		CreatedBy: fixture.crUserID,
	}
}

func TestGenerateFixedItemsIsIdempotent(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	now := time.Now()
	fixture := seedEngine(t, ctx, store, pool, int(now.Weekday()))
	today := now.Format(dateLayout)
	week := now.AddDate(0, 0, 6).Format(dateLayout)
	created, err := store.GenerateFixedItems(ctx, today, week)
	if err != nil {
		t.Fatalf("GenerateFixedItems() returned an unexpected error: %v", err)
	}
	if created != 1 {
		t.Fatalf("created = %d, want one item for the only matching weekday", created)
	}
	again, err := store.GenerateFixedItems(ctx, today, week)
	if err != nil {
		t.Fatalf("second GenerateFixedItems() returned an unexpected error: %v", err)
	}
	if again != 0 {
		t.Errorf("the second run created %d items, want none", again)
	}
	items, err := store.ListTeacherDayItems(ctx, fixture.teacherID, today)
	if err != nil {
		t.Fatalf("ListTeacherDayItems() returned an unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	item := items[0]
	if item.Status != ItemAwaiting || item.Origin != OriginFixed {
		t.Errorf("item = %+v, want an awaiting fixed item", item)
	}
	if item.Room != "1003" || item.Mode != ModeRoom {
		t.Errorf("item room = %q mode = %q, want the routine room", item.Room, item.Mode)
	}
	if item.RoutineID != fixture.routineID {
		t.Errorf("routine = %d, want %d", item.RoutineID, fixture.routineID)
	}
}

func TestSlotRulesOnCreate(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	fixture := seedEngine(t, ctx, store, pool, 1)
	date := "2026-11-02"
	if _, err := store.CreateCRItem(ctx, newItem(fixture, date, "10:00", "11:30", "A")); err != nil {
		t.Fatalf("the first item was refused: %v", err)
	}
	tests := []struct {
		name       string
		section    string
		start      string
		end        string
		mode       string
		room       string
		wantReason string
	}{
		{name: "the same section overlaps", section: "A", start: "11:00", end: "12:00",
			mode: ModeRoom, room: "2104", wantReason: ConflictBatch},
		{name: "ALL overlaps every section", section: "ALL", start: "11:00", end: "12:00",
			mode: ModeRoom, room: "2104", wantReason: ConflictBatch},
		{name: "another section shares the room", section: "B", start: "11:00", end: "12:00",
			mode: ModeRoom, room: "1003", wantReason: ConflictRoom},
		{name: "another section at another time is fine", section: "B", start: "12:00", end: "13:00",
			mode: ModeRoom, room: "1003"},
		{name: "an unmanaged room is never checked", section: "B", start: "11:00", end: "12:00",
			mode: ModeRoom, room: "Library Hall"},
		{name: "online skips the room check", section: "B", start: "14:00", end: "15:00",
			mode: ModeOnline, room: ""},
		{name: "touching slots do not overlap", section: "ALL", start: "09:00", end: "10:00",
			mode: ModeRoom, room: "1003"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item := newItem(fixture, date, test.start, test.end, test.section)
			item.Mode, item.Room = test.mode, test.room
			if test.mode == ModeOnline {
				item.MeetingLink = "https://meet.example.com/x"
			}
			_, err := store.CreateCRItem(ctx, item)
			if test.wantReason == "" {
				if err != nil {
					t.Fatalf("the item was refused: %v", err)
				}
				return
			}
			var conflict *SlotConflict
			if !errors.As(err, &conflict) {
				t.Fatalf("error = %v, want a slot conflict", err)
			}
			if conflict.Reason != test.wantReason {
				t.Errorf("reason = %q, want %q", conflict.Reason, test.wantReason)
			}
		})
	}
}

func TestCancelledItemsReleaseTheirSlot(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	fixture := seedEngine(t, ctx, store, pool, 1)
	date := "2026-11-03"
	first, err := store.CreateCRItem(ctx, newItem(fixture, date, "10:00", "11:30", "A"))
	if err != nil {
		t.Fatalf("the first item was refused: %v", err)
	}
	if _, err := store.CreateCRItem(ctx, newItem(fixture, date, "10:00", "11:30", "A")); err == nil {
		t.Fatal("the clashing item was accepted")
	}
	if _, err := store.CancelItemAsCR(ctx, first.ID, "CSE-23", "Room flooded"); err != nil {
		t.Fatalf("CancelItemAsCR() returned an unexpected error: %v", err)
	}
	if _, err := store.CreateCRItem(ctx, newItem(fixture, date, "10:00", "11:30", "A")); err != nil {
		t.Fatalf("the slot was not released by the cancellation: %v", err)
	}
}

func TestTeacherDecisionsOnAProposal(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	now := time.Now()
	fixture := seedEngine(t, ctx, store, pool, int(now.AddDate(0, 0, 1).Weekday()))
	tomorrow := now.AddDate(0, 0, 1).Format(dateLayout)
	later := now.AddDate(0, 0, 3).Format(dateLayout)
	tests := []struct {
		name       string
		confirm    bool
		accept     bool
		wantStatus string
	}{
		{name: "declined before any confirmation", confirm: false, accept: false, wantStatus: ItemDeclined},
		{name: "declined after a confirmation", confirm: true, accept: false, wantStatus: ItemConfirmed},
		{name: "accepted before any confirmation", confirm: false, accept: true, wantStatus: ItemRescheduled},
		{name: "accepted after a confirmation", confirm: true, accept: true, wantStatus: ItemRescheduled},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := pool.Exec(ctx, "DELETE FROM schedule_items"); err != nil {
				t.Fatalf("could not clear the items: %v", err)
			}
			if _, err := store.GenerateFixedItems(ctx, tomorrow, tomorrow); err != nil {
				t.Fatalf("GenerateFixedItems() returned an unexpected error: %v", err)
			}
			items, err := store.ListTeacherDayItems(ctx, fixture.teacherID, tomorrow)
			if err != nil || len(items) != 1 {
				t.Fatalf("items = %v, err = %v, want one generated item", items, err)
			}
			item := items[0]
			if test.confirm {
				item, err = store.ConfirmItem(ctx, item.ID, fixture.teacherID, ModeRoom, "1003", "", now)
				if err != nil {
					t.Fatalf("ConfirmItem() returned an unexpected error: %v", err)
				}
				if item.Status != ItemConfirmed {
					t.Fatalf("status = %q, want %q", item.Status, ItemConfirmed)
				}
			}
			proposal := Proposal{
				ItemDate:  later,
				StartTime: "14:00",
				EndTime:   "15:30",
				Mode:      ModeRoom,
				Room:      "1003",
			}
			asked, err := store.CRRequestReschedule(ctx, item.ID, "CSE-23", proposal)
			if err != nil {
				t.Fatalf("CRRequestReschedule() returned an unexpected error: %v", err)
			}
			if asked.Status != ItemCRAsk {
				t.Fatalf("status = %q, want %q", asked.Status, ItemCRAsk)
			}
			var decided ScheduleItem
			if test.accept {
				decided, err = store.AcceptProposal(ctx, item.ID, fixture.teacherID)
			} else {
				decided, err = store.DeclineProposal(ctx, item.ID, fixture.teacherID)
			}
			if err != nil {
				t.Fatalf("the decision returned an unexpected error: %v", err)
			}
			if decided.Status != test.wantStatus {
				t.Errorf("status = %q, want %q", decided.Status, test.wantStatus)
			}
			if test.accept {
				if decided.ItemDate != later || decided.StartTime != "14:00" {
					t.Errorf("slot = %s %s, want the proposed slot", decided.ItemDate, decided.StartTime)
				}
			} else if decided.ItemDate != tomorrow {
				t.Errorf("date = %s, want the original %s", decided.ItemDate, tomorrow)
			}
			if decided.ProposedDate != "" {
				t.Errorf("the proposal survived the decision: %+v", decided)
			}
			_ = index
		})
	}
}

func TestConfirmWindowAndLaterChanges(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	now := time.Now()
	fixture := seedEngine(t, ctx, store, pool, int(now.AddDate(0, 0, 1).Weekday()))
	tomorrow := now.AddDate(0, 0, 1).Format(dateLayout)
	if _, err := store.GenerateFixedItems(ctx, tomorrow, tomorrow); err != nil {
		t.Fatalf("GenerateFixedItems() returned an unexpected error: %v", err)
	}
	items, err := store.ListTeacherDayItems(ctx, fixture.teacherID, tomorrow)
	if err != nil || len(items) != 1 {
		t.Fatalf("items = %v, err = %v, want one item", items, err)
	}
	item := items[0]
	confirmed, err := store.ConfirmItem(ctx, item.ID, fixture.teacherID, ModeRoom, "1003", "", now)
	if err != nil {
		t.Fatalf("ConfirmItem() returned an unexpected error: %v", err)
	}
	if confirmed.Status != ItemConfirmed {
		t.Fatalf("status = %q, want %q", confirmed.Status, ItemConfirmed)
	}
	changed, err := store.ConfirmItem(ctx, item.ID, fixture.teacherID, ModeOnline, "",
		"https://meet.example.com/x", now)
	if err != nil {
		t.Fatalf("a later change was refused: %v", err)
	}
	if changed.Mode != ModeOnline || changed.MeetingLink == "" {
		t.Errorf("item = %+v, want the online slot", changed)
	}
	past := now.AddDate(0, 0, -2).Format(dateLayout)
	if _, err := pool.Exec(ctx, "UPDATE schedule_items SET item_date = $2::date WHERE id = $1",
		item.ID, past); err != nil {
		t.Fatalf("could not move the item into the past: %v", err)
	}
	_, err = store.ConfirmItem(ctx, item.ID, fixture.teacherID, ModeRoom, "1003", "", now)
	if !errors.Is(err, ErrItemStarted) {
		t.Errorf("changing a started class error = %v, want ErrItemStarted", err)
	}
}

func TestOwnershipChecks(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	fixture := seedEngine(t, ctx, store, pool, 1)
	item, err := store.CreateCRItem(ctx, newItem(fixture, "2026-11-04", "10:00", "11:30", "A"))
	if err != nil {
		t.Fatalf("CreateCRItem() returned an unexpected error: %v", err)
	}
	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "another teacher cannot confirm",
			call: func() error {
				_, err := store.ConfirmItem(ctx, item.ID, fixture.teacherID+99, ModeRoom, "1003", "",
					time.Now())
				return err
			},
		},
		{
			name: "another batch cannot patch",
			call: func() error {
				note := "Hello"
				_, err := store.UpdateItemAsCR(ctx, item.ID, "CSE-22", ItemPatch{Note: &note})
				return err
			},
		},
		{
			name: "another batch cannot cancel",
			call: func() error {
				_, err := store.CancelItemAsCR(ctx, item.ID, "CSE-22", "")
				return err
			},
		},
		{
			name: "another batch cannot delete",
			call: func() error {
				_, err := store.DeleteCRItem(ctx, item.ID, "CSE-22")
				return err
			},
		},
		{
			name: "another batch cannot put it in the final routine",
			call: func() error {
				_, err := store.SetItemFinal(ctx, item.ID, "CSE-22", true)
				return err
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(); !errors.Is(err, ErrItemNotFound) {
				t.Errorf("error = %v, want ErrItemNotFound", err)
			}
		})
	}
}

func TestPatchAllowed(t *testing.T) {
	date := "2026-11-05"
	start := "10:00"
	syllabus := "Everything"
	note := "Bring the lab sheet"
	tests := []struct {
		name    string
		item    ScheduleItem
		patch   ItemPatch
		wantErr bool
	}{
		{
			name:  "a class representative item takes any change",
			item:  ScheduleItem{Origin: OriginCR, Status: ItemScheduled},
			patch: ItemPatch{ItemDate: &date, Syllabus: &syllabus},
		},
		{
			name:  "a teacher reschedule request takes a new slot",
			item:  ScheduleItem{Origin: OriginFixed, Status: ItemTeacherAsk},
			patch: ItemPatch{ItemDate: &date, StartTime: &start},
		},
		{
			name:    "a teacher reschedule request refuses the syllabus",
			item:    ScheduleItem{Origin: OriginFixed, Status: ItemTeacherAsk},
			patch:   ItemPatch{Syllabus: &syllabus},
			wantErr: true,
		},
		{
			name:  "a teacher exam takes a time",
			item:  ScheduleItem{Origin: OriginTeacherExam, Status: ItemScheduled},
			patch: ItemPatch{StartTime: &start},
		},
		{
			name:    "a teacher exam refuses a new date",
			item:    ScheduleItem{Origin: OriginTeacherExam, Status: ItemScheduled},
			patch:   ItemPatch{ItemDate: &date},
			wantErr: true,
		},
		{
			name:    "a teacher exam refuses the syllabus",
			item:    ScheduleItem{Origin: OriginTeacherExam, Status: ItemScheduled},
			patch:   ItemPatch{Syllabus: &syllabus},
			wantErr: true,
		},
		{
			name:  "any item takes a note",
			item:  ScheduleItem{Origin: OriginFixed, Status: ItemConfirmed},
			patch: ItemPatch{Note: &note},
		},
		{
			name:    "a confirmed fixed item refuses a new slot",
			item:    ScheduleItem{Origin: OriginFixed, Status: ItemConfirmed},
			patch:   ItemPatch{ItemDate: &date},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := patchAllowed(test.item, test.patch)
			if test.wantErr && err == nil {
				t.Error("patchAllowed() accepted the change, want a refusal")
			}
			if !test.wantErr && err != nil {
				t.Errorf("patchAllowed() refused the change: %v", err)
			}
		})
	}
}

func TestFinalAndPublishing(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	fixture := seedEngine(t, ctx, store, pool, 1)
	item, err := store.CreateCRItem(ctx, newItem(fixture, "2026-11-06", "10:00", "11:30", "A"))
	if err != nil {
		t.Fatalf("CreateCRItem() returned an unexpected error: %v", err)
	}
	if _, _, err := store.PublishFinal(ctx, "CSE-23", fixture.crUserID, []int64{item.ID}); !errors.Is(err, ErrItemState) {
		t.Errorf("publishing an item outside the final routine error = %v, want ErrItemState", err)
	}
	inFinal, err := store.SetItemFinal(ctx, item.ID, "CSE-23", true)
	if err != nil {
		t.Fatalf("SetItemFinal() returned an unexpected error: %v", err)
	}
	if !inFinal.InFinal {
		t.Fatal("the item is not in the final routine")
	}
	publication, changed, err := store.PublishFinal(ctx, "CSE-23", fixture.crUserID, []int64{item.ID})
	if err != nil {
		t.Fatalf("PublishFinal() returned an unexpected error: %v", err)
	}
	if publication.ID == 0 || publication.Batch != "CSE-23" {
		t.Errorf("publication = %+v, want a stored publication for CSE-23", publication)
	}
	if len(changed) != 0 {
		t.Errorf("changed = %v, want none on the first publication", changed)
	}
	stored, err := store.GetScheduleItem(ctx, item.ID)
	if err != nil {
		t.Fatalf("GetScheduleItem() returned an unexpected error: %v", err)
	}
	if !stored.Published() || stored.ChangedSincePublish() {
		t.Errorf("item = %+v, want a published item with no pending change", stored)
	}
	room := "2104"
	if _, err := store.UpdateItemAsCR(ctx, item.ID, "CSE-23", ItemPatch{Room: &room}); err != nil {
		t.Fatalf("UpdateItemAsCR() returned an unexpected error: %v", err)
	}
	afterEdit, err := store.GetScheduleItem(ctx, item.ID)
	if err != nil {
		t.Fatalf("GetScheduleItem() returned an unexpected error: %v", err)
	}
	if !afterEdit.ChangedSincePublish() {
		t.Error("the edit is not reported as a pending change")
	}
	if afterEdit.Snapshot.Room != "1003" {
		t.Errorf("the published snapshot moved to %q, want the published room 1003", afterEdit.Snapshot.Room)
	}
	_, republished, err := store.PublishFinal(ctx, "CSE-23", fixture.crUserID, []int64{item.ID})
	if err != nil {
		t.Fatalf("the second PublishFinal() returned an unexpected error: %v", err)
	}
	if len(republished) != 1 {
		t.Fatalf("changed = %v, want the edited item", republished)
	}
	final, err := store.GetScheduleItem(ctx, item.ID)
	if err != nil {
		t.Fatalf("GetScheduleItem() returned an unexpected error: %v", err)
	}
	if final.Snapshot.Room != "2104" || final.ChangedSincePublish() {
		t.Errorf("item = %+v, want the republished room", final)
	}
	removed, err := store.SetItemFinal(ctx, item.ID, "CSE-23", false)
	if err != nil {
		t.Fatalf("SetItemFinal(false) returned an unexpected error: %v", err)
	}
	if removed.InFinal {
		t.Error("the item stayed in the final routine")
	}
}

func TestPublishNeedsACompleteItem(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	fixture := seedEngine(t, ctx, store, pool, 1)
	exam := NewScheduleItem{
		Batch:     "CSE-23",
		Section:   "ALL",
		CourseID:  fixture.courseID,
		TeacherID: fixture.teacherID,
		ItemDate:  "2026-11-07",
		Syllabus:  "Everything",
		CreatedBy: fixture.teacherDB,
	}
	item, err := store.CreateTeacherExam(ctx, exam)
	if err != nil {
		t.Fatalf("CreateTeacherExam() returned an unexpected error: %v", err)
	}
	if item.HasTimes() {
		t.Fatal("a fresh teacher exam already carries times")
	}
	if _, err := store.SetItemFinal(ctx, item.ID, "CSE-23", true); err != nil {
		t.Fatalf("SetItemFinal() returned an unexpected error: %v", err)
	}
	_, _, err = store.PublishFinal(ctx, "CSE-23", fixture.crUserID, []int64{item.ID})
	var incomplete *IncompleteItemError
	if !errors.As(err, &incomplete) {
		t.Fatalf("error = %v, want an incomplete item error", err)
	}
	if incomplete.Item.ID != item.ID {
		t.Errorf("the error names item %d, want %d", incomplete.Item.ID, item.ID)
	}
	start, end, mode, room := "09:00", "11:00", ModeRoom, "1003"
	patch := ItemPatch{StartTime: &start, EndTime: &end, Mode: &mode, Room: &room}
	if _, err := store.UpdateItemAsCR(ctx, item.ID, "CSE-23", patch); err != nil {
		t.Fatalf("UpdateItemAsCR() returned an unexpected error: %v", err)
	}
	if _, _, err := store.PublishFinal(ctx, "CSE-23", fixture.crUserID, []int64{item.ID}); err != nil {
		t.Fatalf("PublishFinal() after the time was set returned an unexpected error: %v", err)
	}
}

func TestPublishedStudentViews(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	fixture := seedEngine(t, ctx, store, pool, 1)
	date := "2026-11-09"
	item, err := store.CreateCRItem(ctx, newItem(fixture, date, "10:00", "11:30", "A"))
	if err != nil {
		t.Fatalf("CreateCRItem() returned an unexpected error: %v", err)
	}
	plan, err := store.ListPublishedDay(ctx, "CSE-23", date, []string{})
	if err != nil {
		t.Fatalf("ListPublishedDay() returned an unexpected error: %v", err)
	}
	if len(plan) != 0 {
		t.Fatalf("the student sees %v before the publication, want nothing", plan)
	}
	if _, err := store.SetItemFinal(ctx, item.ID, "CSE-23", true); err != nil {
		t.Fatalf("SetItemFinal() returned an unexpected error: %v", err)
	}
	if _, _, err := store.PublishFinal(ctx, "CSE-23", fixture.crUserID, []int64{item.ID}); err != nil {
		t.Fatalf("PublishFinal() returned an unexpected error: %v", err)
	}
	plan, err = store.ListPublishedDay(ctx, "CSE-23", date, []string{})
	if err != nil {
		t.Fatalf("ListPublishedDay() returned an unexpected error: %v", err)
	}
	if len(plan) != 1 || plan[0].CourseCode != "CSE 2243 7637D" {
		t.Fatalf("the student sees %v, want the published class", plan)
	}
	if _, err := store.CancelItemAsCR(ctx, item.ID, "CSE-23", "Teacher is ill"); err != nil {
		t.Fatalf("CancelItemAsCR() returned an unexpected error: %v", err)
	}
	plan, err = store.ListPublishedDay(ctx, "CSE-23", date, []string{})
	if err != nil {
		t.Fatalf("ListPublishedDay() returned an unexpected error: %v", err)
	}
	if len(plan) != 1 || plan[0].Status != ItemCancelled {
		t.Fatalf("the student sees %v, want the cancellation straight away", plan)
	}
}

func TestUpdateItemLinkPermissions(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	fixture := seedEngine(t, ctx, store, pool, 1)
	item, err := store.CreateCRItem(ctx, newItem(fixture, "2026-11-10", "10:00", "11:30", "A"))
	if err != nil {
		t.Fatalf("CreateCRItem() returned an unexpected error: %v", err)
	}
	strangerID := insertUser(t, ctx, pool, "student", "2023331999", StatusActive, 0)
	insertStudent(t, ctx, pool, strangerID, "2023331999", false)
	tests := []struct {
		name    string
		userID  int64
		wantErr bool
	}{
		{name: "the teacher of the item", userID: fixture.teacherDB},
		{name: "the class representative of the batch", userID: fixture.crUserID},
		{name: "a plain student", userID: strangerID, wantErr: true},
		{name: "nobody", userID: 0, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			link := "https://meet.example.com/" + test.name
			updated, err := store.UpdateItemLink(ctx, item.ID, test.userID, link)
			if test.wantErr {
				if !errors.Is(err, ErrItemNotFound) {
					t.Fatalf("error = %v, want ErrItemNotFound", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateItemLink() returned an unexpected error: %v", err)
			}
			if updated.MeetingLink != link {
				t.Errorf("link = %q, want %q", updated.MeetingLink, link)
			}
		})
	}
}

func TestCourseSearchIgnoresSpaces(t *testing.T) {
	ctx, store, _ := newTestStore(t)
	if _, err := store.CreateCourse(ctx, "cse 2243 7637d", "Compilers", 3.0); err != nil {
		t.Fatalf("CreateCourse() returned an unexpected error: %v", err)
	}
	if _, err := store.CreateCourse(ctx, "CSE-1111", "Introduction", 3.0); err != nil {
		t.Fatalf("CreateCourse() returned an unexpected error: %v", err)
	}
	tests := []struct {
		name   string
		search string
		want   int
	}{
		{name: "the stored spelling", search: "CSE 2243 7637D", want: 1},
		{name: "without spaces", search: "CSE22437637D", want: 1},
		{name: "with hyphens", search: "cse-2243-7637d", want: 1},
		{name: "a prefix", search: "cse2243", want: 1},
		{name: "by title", search: "compil", want: 1},
		{name: "everything", search: "", want: 2},
		{name: "nothing matches", search: "MAT 101", want: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			courses, err := store.SearchCourses(ctx, test.search)
			if err != nil {
				t.Fatalf("SearchCourses() returned an unexpected error: %v", err)
			}
			if len(courses) != test.want {
				t.Errorf("SearchCourses(%q) returned %v, want %d course(s)", test.search, courses, test.want)
			}
		})
	}
}
