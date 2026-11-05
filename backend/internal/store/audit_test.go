package store

import (
	"testing"
	"time"
)

func TestAuditEventsCanBeFilteredAndPaged(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	adminID := insertUser(t, ctx, pool, RoleAdmin, "office.one", StatusActive, 0)
	teacherUser := insertUser(t, ctx, pool, RoleTeacher, "t.islam", StatusActive, 0)
	insertTeacherRow(t, ctx, pool, teacherUser, "T-201", "Ratul Islam")
	events := []AuditEvent{
		{ActorID: adminID, ActorRole: RoleAdmin, Action: ActionCreated, Summary: "Added a course", Client: ClientWeb},
		{ActorID: adminID, ActorRole: RoleAdmin, Action: ActionDeleted, Summary: "Removed a notice", Client: ClientWeb},
		{ActorID: teacherUser, ActorRole: RoleTeacher, Action: ActionSignIn, Summary: "Ratul signed in", Client: ClientAndroid},
		{ActorRole: "", Action: ActionSignInFailed, Summary: "Failed sign-in for ghost", Client: ClientWeb},
	}
	for _, event := range events {
		if err := store.WriteEvent(ctx, event); err != nil {
			t.Fatalf("WriteEvent() returned an unexpected error: %v", err)
		}
	}
	all, err := store.ListEvents(ctx, AuditFilter{})
	if err != nil {
		t.Fatalf("ListEvents() returned an unexpected error: %v", err)
	}
	if len(all) != 4 {
		t.Fatalf("got %d events, want 4", len(all))
	}
	if all[0].Action != ActionSignInFailed {
		t.Errorf("the newest event is %q, want the failed sign-in", all[0].Action)
	}
	if all[0].ActorLabel != "System" {
		t.Errorf("the actorless event is labelled %q, want System", all[0].ActorLabel)
	}
	byPerson, err := store.ListEvents(ctx, AuditFilter{ActorID: adminID})
	if err != nil {
		t.Fatalf("ListEvents(person) returned an unexpected error: %v", err)
	}
	if len(byPerson) != 2 {
		t.Errorf("got %d events for the admin, want 2", len(byPerson))
	}
	byRole, err := store.ListEvents(ctx, AuditFilter{Role: RoleTeacher})
	if err != nil {
		t.Fatalf("ListEvents(role) returned an unexpected error: %v", err)
	}
	if len(byRole) != 1 || byRole[0].ActorLabel != "Ratul Islam" {
		t.Errorf("teacher events = %+v, want the teacher sign-in named", byRole)
	}
	byClient, err := store.ListEvents(ctx, AuditFilter{Client: ClientAndroid})
	if err != nil {
		t.Fatalf("ListEvents(client) returned an unexpected error: %v", err)
	}
	if len(byClient) != 1 {
		t.Errorf("got %d app events, want 1", len(byClient))
	}
	bySearch, err := store.ListEvents(ctx, AuditFilter{Search: "notice"})
	if err != nil {
		t.Fatalf("ListEvents(search) returned an unexpected error: %v", err)
	}
	if len(bySearch) != 1 {
		t.Errorf("got %d events for the search, want 1", len(bySearch))
	}
	firstPage, err := store.ListEvents(ctx, AuditFilter{Limit: 2})
	if err != nil {
		t.Fatalf("ListEvents(limit) returned an unexpected error: %v", err)
	}
	if len(firstPage) != 2 {
		t.Fatalf("got %d events on the first page, want 2", len(firstPage))
	}
	nextPage, err := store.ListEvents(ctx, AuditFilter{Limit: 2, Before: firstPage[1].ID})
	if err != nil {
		t.Fatalf("ListEvents(before) returned an unexpected error: %v", err)
	}
	if len(nextPage) != 2 || nextPage[0].ID >= firstPage[1].ID {
		t.Errorf("the second page is %+v, want the two older events", nextPage)
	}
}

func TestAuditEventsAreFilteredByDate(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	adminID := insertUser(t, ctx, pool, RoleAdmin, "office.one", StatusActive, 0)
	event := AuditEvent{ActorID: adminID, ActorRole: RoleAdmin, Action: ActionCreated, Client: ClientWeb}
	if err := store.WriteEvent(ctx, event); err != nil {
		t.Fatalf("WriteEvent() returned an unexpected error: %v", err)
	}
	past := time.Now().Add(-48 * time.Hour)
	future := time.Now().Add(48 * time.Hour)
	inside, err := store.ListEvents(ctx, AuditFilter{From: &past, To: &future})
	if err != nil {
		t.Fatalf("ListEvents(range) returned an unexpected error: %v", err)
	}
	if len(inside) != 1 {
		t.Errorf("got %d events inside the range, want 1", len(inside))
	}
	outside, err := store.ListEvents(ctx, AuditFilter{From: &future})
	if err != nil {
		t.Fatalf("ListEvents(from) returned an unexpected error: %v", err)
	}
	if len(outside) != 0 {
		t.Errorf("got %d events after the range, want 0", len(outside))
	}
}

func TestActionCountsAndPeopleDescribeTheTimeline(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	adminID := insertUser(t, ctx, pool, RoleAdmin, "office.one", StatusActive, 0)
	for i := 0; i < 2; i++ {
		event := AuditEvent{ActorID: adminID, ActorRole: RoleAdmin, Action: ActionCreated, Client: ClientWeb}
		if err := store.WriteEvent(ctx, event); err != nil {
			t.Fatalf("WriteEvent() returned an unexpected error: %v", err)
		}
	}
	update := AuditEvent{ActorID: adminID, ActorRole: RoleAdmin, Action: ActionUpdated, Client: ClientWeb}
	if err := store.WriteEvent(ctx, update); err != nil {
		t.Fatalf("WriteEvent() returned an unexpected error: %v", err)
	}
	counts, err := store.CountActions(ctx, adminID, nil, nil)
	if err != nil {
		t.Fatalf("CountActions() returned an unexpected error: %v", err)
	}
	if len(counts) != 2 || counts[0].Action != ActionCreated || counts[0].Total != 2 {
		t.Errorf("counts = %+v, want two creates first", counts)
	}
	people, err := store.ListAuditPeople(ctx)
	if err != nil {
		t.Fatalf("ListAuditPeople() returned an unexpected error: %v", err)
	}
	if len(people) != 1 || people[0].ID != adminID || people[0].Total != 3 {
		t.Errorf("people = %+v, want the admin with three actions", people)
	}
}

func TestOldEventsCanBeDeleted(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	adminID := insertUser(t, ctx, pool, RoleAdmin, "office.one", StatusActive, 0)
	event := AuditEvent{ActorID: adminID, ActorRole: RoleAdmin, Action: ActionCreated, Client: ClientWeb}
	if err := store.WriteEvent(ctx, event); err != nil {
		t.Fatalf("WriteEvent() returned an unexpected error: %v", err)
	}
	dropped, err := store.DeleteOldEvents(ctx, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("DeleteOldEvents() returned an unexpected error: %v", err)
	}
	if dropped != 1 {
		t.Errorf("deleted %d events, want 1", dropped)
	}
}
