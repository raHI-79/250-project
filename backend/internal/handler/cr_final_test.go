package handler

import (
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"testing"
)

func TestCRActionsPerOrigin(t *testing.T) {
	tests := []struct {
		name    string
		item    store.ScheduleItem
		want    []string
		without []string
	}{
		{
			name: "an item of the class representative",
			item: store.ScheduleItem{Origin: store.OriginCR, Status: store.ItemScheduled, InFinal: true},
			want: []string{ActionEdit, ActionCancel, ActionDelete, ActionRemoveFromFinal},
		},
		{
			name:    "an exam of the teacher",
			item:    store.ScheduleItem{Origin: store.OriginTeacherExam, Status: store.ItemScheduled, InFinal: true},
			want:    []string{ActionSetTime, ActionRemoveFromFinal},
			without: []string{ActionDelete},
		},
		{
			name:    "a cancelled item of the final routine",
			item:    store.ScheduleItem{Origin: store.OriginFixed, Status: store.ItemCancelled, InFinal: true},
			want:    []string{ActionRemoveFromFinal},
			without: []string{ActionDelete, ActionAddToFinal},
		},
		{
			name:    "a cancelled item outside the final routine",
			item:    store.ScheduleItem{Origin: store.OriginFixed, Status: store.ItemCancelled},
			without: []string{ActionRemoveFromFinal, ActionAddToFinal, ActionDelete},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actions := crActions(test.item)
			has := make(map[string]bool, len(actions))
			for _, action := range actions {
				has[action] = true
			}
			for _, action := range test.want {
				if !has[action] {
					t.Errorf("actions = %v, want %q among them", actions, action)
				}
			}
			for _, action := range test.without {
				if has[action] {
					t.Errorf("actions = %v, want %q left out", actions, action)
				}
			}
		})
	}
}

func TestCRDeleteOnlyTouchesItsOwnItems(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		storeErr   error
		wantStatus int
		wantError  string
	}{
		{name: "an item of the class representative", wantStatus: http.StatusOK},
		{name: "an exam of the teacher", storeErr: store.ErrItemState,
			wantStatus: http.StatusConflict, wantError: MessageItemState},
		{name: "an item of another batch", storeErr: store.ErrItemNotFound,
			wantStatus: http.StatusNotFound, wantError: MessageItemNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items := newFakeScheduleStore()
			items.err = test.storeErr
			cr := newCRHandler(t, items, &fakeNotifier{})
			rec := scheduleCall(t, cr.DeleteItem(), http.MethodDelete, "/api/cr/items/4",
				"4", "", crUserID, auth.RoleStudent)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				if len(items.deleted) != 0 {
					t.Errorf("deleted = %v, want nothing", items.deleted)
				}
				return
			}
			if len(items.deleted) != 1 || items.deleted[0] != 4 {
				t.Errorf("deleted = %v, want the item 4", items.deleted)
			}
		})
	}
}

func TestCRRemoveFromFinalWithdrawsAPublishedItem(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name          string
		published     bool
		wantBroadcast int
		wantDirect    int
	}{
		{name: "an item that was never published", published: false},
		{name: "a published item", published: true, wantBroadcast: 1, wantDirect: 3},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items := newFakeScheduleStore()
			items.dropUsers = []int64{77, 78}
			items.withdrawn[4] = test.published
			alerts := &fakeNotifier{}
			cr := newCRHandler(t, items, alerts)
			rec := scheduleCall(t, cr.RemoveFromFinal(), http.MethodDelete, "/api/cr/items/4/final",
				"4", "", crUserID, auth.RoleStudent)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
			}
			if len(items.removals) != 1 || items.removals[0] != 4 {
				t.Errorf("removals = %v, want the item 4", items.removals)
			}
			if alerts.broadcasts != test.wantBroadcast {
				t.Errorf("broadcasts = %d, want %d", alerts.broadcasts, test.wantBroadcast)
			}
			if len(alerts.direct) != test.wantDirect {
				t.Errorf("direct notifications = %d, want %d (the teacher and the drop course students)",
					len(alerts.direct), test.wantDirect)
			}
		})
	}
}

func TestCRRemovesManyItemsFromFinal(t *testing.T) {
	silenceLog(t)
	items := newFakeScheduleStore()
	items.withdrawn[5] = true
	alerts := &fakeNotifier{}
	cr := newCRHandler(t, items, alerts)
	rec := scheduleCall(t, cr.RemoveManyFromFinal(), http.MethodPost, "/api/cr/final/remove",
		"", `{"item_ids":[5,6,7]}`, crUserID, auth.RoleStudent)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	payload := decodeBody(t, rec)
	if payload["removed"] != float64(3) || payload["withdrawn"] != float64(1) {
		t.Errorf("payload = %v, want three removed and one withdrawn", payload)
	}
	if len(items.removals) != 3 {
		t.Errorf("removals = %v, want the three picked items", items.removals)
	}
	if alerts.broadcasts != 1 {
		t.Errorf("broadcasts = %d, want one for the published item", alerts.broadcasts)
	}
}

func TestCRRemoveManyNeedsIDs(t *testing.T) {
	silenceLog(t)
	items := newFakeScheduleStore()
	cr := newCRHandler(t, items, &fakeNotifier{})
	rec := scheduleCall(t, cr.RemoveManyFromFinal(), http.MethodPost, "/api/cr/final/remove",
		"", `{"item_ids":[]}`, crUserID, auth.RoleStudent)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := errorMessage(t, rec); got != MessageIDsRequired {
		t.Errorf("error = %q, want %q", got, MessageIDsRequired)
	}
}
