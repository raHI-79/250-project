package store

import (
	"errors"
	"testing"
	"time"
)

func TestPeopleAreSearchedAndFiltered(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	first := insertUser(t, ctx, pool, RoleStudent, "2023331001", StatusActive, 0)
	insertStudentRow(t, ctx, pool, first, "2023331001", "CSE-23", "A")
	second := insertUser(t, ctx, pool, RoleStudent, "2022331002", StatusRemoved, 0)
	insertStudentRow(t, ctx, pool, second, "2022331002", "CSE-22", "B")
	teacher := insertUser(t, ctx, pool, RoleTeacher, "t.islam", StatusActive, 0)
	insertTeacherRow(t, ctx, pool, teacher, "T-501", "Ratul Islam")
	all, total, err := store.ListPeople(ctx, PersonFilter{})
	if err != nil {
		t.Fatalf("ListPeople() returned an unexpected error: %v", err)
	}
	if len(all) != 3 || total != 3 {
		t.Fatalf("got %d of %d accounts, want 3 of 3", len(all), total)
	}
	students, _, err := store.ListPeople(ctx, PersonFilter{Role: RoleStudent})
	if err != nil {
		t.Fatalf("ListPeople(role) returned an unexpected error: %v", err)
	}
	if len(students) != 2 {
		t.Errorf("got %d students, want 2", len(students))
	}
	active, _, err := store.ListPeople(ctx, PersonFilter{Status: StatusActive})
	if err != nil {
		t.Fatalf("ListPeople(status) returned an unexpected error: %v", err)
	}
	if len(active) != 2 {
		t.Errorf("got %d active accounts, want 2", len(active))
	}
	batch, _, err := store.ListPeople(ctx, PersonFilter{Batch: "CSE-23"})
	if err != nil {
		t.Fatalf("ListPeople(batch) returned an unexpected error: %v", err)
	}
	if len(batch) != 1 {
		t.Errorf("got %d accounts in CSE-23, want 1", len(batch))
	}
	byName, _, err := store.ListPeople(ctx, PersonFilter{Search: "Ratul"})
	if err != nil {
		t.Fatalf("ListPeople(search) returned an unexpected error: %v", err)
	}
	if len(byName) != 1 || byName[0].Name != "Ratul Islam" {
		t.Errorf("search = %+v, want the teacher", byName)
	}
	byID, _, err := store.ListPeople(ctx, PersonFilter{Search: "2023331001"})
	if err != nil {
		t.Fatalf("ListPeople(search) returned an unexpected error: %v", err)
	}
	if len(byID) != 1 {
		t.Errorf("searching a registration number found %d accounts, want 1", len(byID))
	}
}

func TestPeopleActionsChangeOneAccount(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	userID := insertUser(t, ctx, pool, RoleStudent, "2023331001", StatusActive, 0)
	insertStudentRow(t, ctx, pool, userID, "2023331001", "CSE-23", "A")
	if _, err := store.CreateSession(ctx, NewSession{
		UserID:    userID,
		Client:    ClientWeb,
		ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateSession() returned an unexpected error: %v", err)
	}
	bind := `INSERT INTO device_bindings (user_id, device_id) VALUES ($1, 'device-1')`
	if _, err := pool.Exec(ctx, bind, userID); err != nil {
		t.Fatalf("could not bind a device: %v", err)
	}
	detail, err := store.FindPersonDetail(ctx, userID)
	if err != nil {
		t.Fatalf("FindPersonDetail() returned an unexpected error: %v", err)
	}
	if detail.Sessions != 1 || detail.DeviceID != "device-1" {
		t.Fatalf("detail = %+v, want one session and a bound device", detail)
	}
	if err := store.ResetPersonDevice(ctx, userID); err != nil {
		t.Fatalf("ResetPersonDevice() returned an unexpected error: %v", err)
	}
	if err := store.SetCR(ctx, userID, true); err != nil {
		t.Fatalf("SetCR() returned an unexpected error: %v", err)
	}
	detail, err = store.FindPersonDetail(ctx, userID)
	if err != nil {
		t.Fatalf("FindPersonDetail() returned an unexpected error: %v", err)
	}
	if detail.DeviceID != "" {
		t.Error("the attendance phone was not reset")
	}
	if !detail.IsCR {
		t.Error("the student was not made a CR")
	}
	if _, err := store.SuspendPerson(ctx, userID); err != nil {
		t.Fatalf("SuspendPerson() returned an unexpected error: %v", err)
	}
	detail, err = store.FindPersonDetail(ctx, userID)
	if err != nil {
		t.Fatalf("FindPersonDetail() returned an unexpected error: %v", err)
	}
	if detail.Status != StatusRemoved || detail.Sessions != 0 {
		t.Errorf("detail = %+v, want a removed account with no sessions", detail)
	}
	if _, err := store.RestorePerson(ctx, userID); err != nil {
		t.Fatalf("RestorePerson() returned an unexpected error: %v", err)
	}
	if err := store.SignOutPerson(ctx, userID); err != nil {
		t.Fatalf("SignOutPerson() returned an unexpected error: %v", err)
	}
	teacher := insertUser(t, ctx, pool, RoleTeacher, "t.islam", StatusActive, 0)
	insertTeacherRow(t, ctx, pool, teacher, "T-502", "Ratul Islam")
	if err := store.SetCR(ctx, teacher, true); !errors.Is(err, ErrNotAStudent) {
		t.Errorf("SetCR() on a teacher returned %v, want ErrNotAStudent", err)
	}
}
