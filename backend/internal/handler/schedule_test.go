package handler

import (
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"net/http"
	"testing"
)

const (
	teacherUserID = int64(44)
	crUserID      = int64(11)
)

func newTeacherHandler(t *testing.T, items *fakeScheduleStore, alerts *fakeNotifier) *Teacher {
	t.Helper()
	return NewTeacher(accountConfig(t, "test"), items, alerts)
}

func newCRHandler(t *testing.T, items *fakeScheduleStore, alerts *fakeNotifier) *CR {
	t.Helper()
	return NewCR(accountConfig(t, "test"), items, alerts)
}

func TestTeacherConfirm(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		body       string
		storeErr   error
		wantStatus int
		wantError  string
		wantSlot   string
	}{
		{name: "a room", body: `{"mode":"room","room":"1003"}`, wantStatus: http.StatusOK, wantSlot: "room:1003:"},
		{
			name:       "online with a link",
			body:       `{"mode":"online","meeting_link":"https://meet.example.com/a"}`,
			wantStatus: http.StatusOK,
			wantSlot:   "online::https://meet.example.com/a",
		},
		{name: "an unknown mode", body: `{"mode":"hall","room":"1003"}`, wantStatus: http.StatusBadRequest,
			wantError: "mode must be one of room, online."},
		{name: "a room mode without a room", body: `{"mode":"room"}`, wantStatus: http.StatusBadRequest,
			wantError: "Room is required."},
		{name: "a broken link", body: `{"mode":"online","meeting_link":"meet.example.com"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Meeting link must be a full link that starts with http:// or https://."},
		{name: "outside the window", body: `{"mode":"room","room":"1003"}`, storeErr: store.ErrDecisionClosed,
			wantStatus: http.StatusConflict, wantError: MessageDecisionClosed},
		{name: "already started", body: `{"mode":"room","room":"1003"}`, storeErr: store.ErrItemStarted,
			wantStatus: http.StatusConflict, wantError: MessageItemStarted},
		{name: "not this teacher", body: `{"mode":"room","room":"1003"}`, storeErr: store.ErrItemNotFound,
			wantStatus: http.StatusNotFound, wantError: MessageItemNotFound},
		{name: "a wrong state", body: `{"mode":"room","room":"1003"}`, storeErr: store.ErrItemState,
			wantStatus: http.StatusConflict, wantError: MessageItemState},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items := newFakeScheduleStore()
			items.err = test.storeErr
			items.crs = []int64{crUserID}
			alerts := &fakeNotifier{}
			teacher := newTeacherHandler(t, items, alerts)
			rec := scheduleCall(t, teacher.Confirm(), http.MethodPost, "/api/teacher/items/4/confirm",
				"4", test.body, teacherUserID, auth.RoleTeacher)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				if alerts.calls != 0 {
					t.Errorf("notifications = %d, want none after a refusal", alerts.calls)
				}
				return
			}
			if len(items.confirmed) != 1 || items.confirmed[0] != test.wantSlot {
				t.Errorf("the store received %v, want %q", items.confirmed, test.wantSlot)
			}
			if alerts.calls != 1 || alerts.userID != crUserID {
				t.Fatalf("notifications = %d to user %d, want one to the class representative",
					alerts.calls, alerts.userID)
			}
			if alerts.link.Kind != notify.LinkCRUpcoming || alerts.link.Ref != 4 {
				t.Errorf("link = %+v, want the class representative upcoming item", alerts.link)
			}
		})
	}
}

func TestTeacherConfirmPassesTheWindow(t *testing.T) {
	silenceLog(t)
	items := newFakeScheduleStore()
	teacher := newTeacherHandler(t, items, &fakeNotifier{})
	rec := scheduleCall(t, teacher.Confirm(), http.MethodPost, "/api/teacher/items/4/confirm",
		"4", `{"mode":"room","room":"1003"}`, teacherUserID, auth.RoleTeacher)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if items.windowOpen {
		t.Error("the handler reported an open window although the test clock is outside 17:00 to 23:00")
	}
}

func TestTeacherCancelTellsEverybody(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name          string
		published     bool
		wantBroadcast int
	}{
		{name: "an unpublished class stays between the teacher and the class representative", published: false},
		{name: "a published class reaches the students at once", published: true, wantBroadcast: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items := newFakeScheduleStore()
			items.crs = []int64{crUserID}
			if test.published {
				items.item.Snapshot = store.ItemSnapshot{Exists: true, Status: store.ItemConfirmed}
			}
			alerts := &fakeNotifier{}
			teacher := newTeacherHandler(t, items, alerts)
			rec := scheduleCall(t, teacher.Cancel(), http.MethodPost, "/api/teacher/items/4/cancel",
				"4", `{"reason":"I am ill."}`, teacherUserID, auth.RoleTeacher)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
			}
			if len(items.cancelled) != 1 || items.cancelled[0] != "I am ill." {
				t.Errorf("the store received %v, want the reason", items.cancelled)
			}
			if alerts.calls != 1 {
				t.Errorf("direct notifications = %d, want one to the class representative", alerts.calls)
			}
			if alerts.broadcasts != test.wantBroadcast {
				t.Errorf("broadcasts = %d, want %d", alerts.broadcasts, test.wantBroadcast)
			}
			if test.wantBroadcast > 0 && alerts.link.Kind != notify.LinkRoutine {
				t.Errorf("link = %+v, want the routine page", alerts.link)
			}
		})
	}
}

func TestTeacherRescheduleRequests(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{name: "with a message", body: `{"message":"I am at a conference."}`, wantStatus: http.StatusOK},
		{name: "without a message", body: `{}`, wantStatus: http.StatusBadRequest,
			wantError: "Message is required."},
		{name: "with a blank message", body: `{"message":"   "}`, wantStatus: http.StatusBadRequest,
			wantError: "Message is required."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items := newFakeScheduleStore()
			items.crs = []int64{crUserID}
			alerts := &fakeNotifier{}
			teacher := newTeacherHandler(t, items, alerts)
			rec := scheduleCall(t, teacher.RequestReschedule(), http.MethodPost,
				"/api/teacher/items/4/request-reschedule", "4", test.body, teacherUserID, auth.RoleTeacher)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			if got := decodeBody(t, rec)["message"]; got != MessageRescheduleAsked {
				t.Errorf("message = %v, want %q", got, MessageRescheduleAsked)
			}
		})
	}
}

func TestTeacherDecidesAProposal(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name         string
		accept       bool
		storeErr     error
		wantStatus   int
		wantError    string
		wantMessage  string
		wantStatuses string
	}{
		{name: "accepted", accept: true, wantStatus: http.StatusOK, wantMessage: MessageProposalTaken,
			wantStatuses: store.ItemRescheduled},
		{name: "declined", accept: false, wantStatus: http.StatusOK, wantMessage: MessageProposalLeft,
			wantStatuses: store.ItemDeclined},
		{name: "no proposal", accept: true, storeErr: store.ErrNoProposal, wantStatus: http.StatusConflict,
			wantError: MessageNoProposal},
		{name: "a wrong state", accept: false, storeErr: store.ErrItemState, wantStatus: http.StatusConflict,
			wantError: MessageItemState},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items := newFakeScheduleStore()
			items.err = test.storeErr
			items.crs = []int64{crUserID}
			alerts := &fakeNotifier{}
			teacher := newTeacherHandler(t, items, alerts)
			handlerFunc := teacher.DeclineReschedule()
			path := "/api/teacher/items/4/reschedule-request/decline"
			if test.accept {
				handlerFunc = teacher.AcceptReschedule()
				path = "/api/teacher/items/4/reschedule-request/accept"
			}
			rec := scheduleCall(t, handlerFunc, http.MethodPost, path, "4", "", teacherUserID, auth.RoleTeacher)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			payload := decodeBody(t, rec)
			if payload["message"] != test.wantMessage {
				t.Errorf("message = %v, want %q", payload["message"], test.wantMessage)
			}
			item, _ := payload["item"].(map[string]any)
			if item == nil || item["status"] != test.wantStatuses {
				t.Errorf("item = %v, want the status %q", item, test.wantStatuses)
			}
		})
	}
}

func TestTeacherExamValidation(t *testing.T) {
	silenceLog(t)
	tomorrow := validate.DateString(accountConfig(t, "test").Now().AddDate(0, 0, 1))
	yesterday := validate.DateString(accountConfig(t, "test").Now().AddDate(0, 0, -1))
	tests := []struct {
		name        string
		body        string
		wantStatus  int
		wantError   string
		wantSection string
	}{
		{
			name:        "a section is optional",
			body:        `{"course_id":7,"batch":"CSE-23","item_date":"` + tomorrow + `","syllabus":"All"}`,
			wantStatus:  http.StatusCreated,
			wantSection: validate.SectionAll,
		},
		{
			name: "one section",
			body: `{"course_id":7,"batch":"cse-23","section":"b","item_date":"` + tomorrow +
				`","syllabus":"All","instructions":"Bring a pen."}`,
			wantStatus:  http.StatusCreated,
			wantSection: "B",
		},
		{
			name:       "without a course",
			body:       `{"batch":"CSE-23","item_date":"` + tomorrow + `"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "course_id must be a positive number.",
		},
		{
			name:       "a broken batch",
			body:       `{"course_id":7,"batch":"CSE","item_date":"` + tomorrow + `"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Batch must look like CSE-23.",
		},
		{
			name:       "a date in the past",
			body:       `{"course_id":7,"batch":"CSE-23","item_date":"` + yesterday + `"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessagePastDate,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items := newFakeScheduleStore()
			items.crs = []int64{crUserID}
			alerts := &fakeNotifier{}
			teacher := newTeacherHandler(t, items, alerts)
			rec := scheduleCall(t, teacher.CreateExam(), http.MethodPost, "/api/teacher/exams",
				"", test.body, teacherUserID, auth.RoleTeacher)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			if len(items.created) != 1 {
				t.Fatalf("the store received %d exams, want 1", len(items.created))
			}
			created := items.created[0]
			if created.Section != test.wantSection {
				t.Errorf("section = %q, want %q", created.Section, test.wantSection)
			}
			if created.Batch != "CSE-23" {
				t.Errorf("batch = %q, want CSE-23", created.Batch)
			}
			if created.StartTime != "" || created.EndTime != "" {
				t.Errorf("times = %q to %q, want none until the class representative sets them",
					created.StartTime, created.EndTime)
			}
			if alerts.link.Kind != notify.LinkCRFinal {
				t.Errorf("link = %+v, want the final routine page", alerts.link)
			}
		})
	}
}

func TestCRCreateItem(t *testing.T) {
	silenceLog(t)
	tomorrow := validate.DateString(accountConfig(t, "test").Now().AddDate(0, 0, 1))
	base := func(extra string) string {
		return `{"kind":"class","course_id":7,"section":"A","item_date":"` + tomorrow + `"` + extra + `}`
	}
	tests := []struct {
		name       string
		body       string
		storeErr   error
		wantStatus int
		wantError  string
	}{
		{
			name:       "a class in a room",
			body:       base(`,"start_time":"10:00","end_time":"11:30","mode":"room","room":"1003"`),
			wantStatus: http.StatusCreated,
		},
		{
			name:       "an online class",
			body:       base(`,"start_time":"10:00","end_time":"11:30","mode":"online","meeting_link":"https://meet.example.com/x"`),
			wantStatus: http.StatusCreated,
		},
		{
			name:       "an exam",
			body:       `{"kind":"exam","course_id":7,"section":"ALL","item_date":"` + tomorrow + `","start_time":"09:00","end_time":"11:00","mode":"room","room":"1003"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "a lab is not a class representative kind",
			body:       base(`,"start_time":"10:00","end_time":"11:30","mode":"room","room":"1003"`),
			wantStatus: http.StatusCreated,
		},
		{
			name:       "outside the class hours",
			body:       base(`,"start_time":"07:00","end_time":"08:00","mode":"room","room":"1003"`),
			wantStatus: http.StatusBadRequest,
			wantError:  "Classes must be between 08:00 and 17:00.",
		},
		{
			name:       "the end before the start",
			body:       base(`,"start_time":"11:00","end_time":"10:00","mode":"room","room":"1003"`),
			wantStatus: http.StatusBadRequest,
			wantError:  "Start time must be earlier than end time.",
		},
		{
			name:       "online without a link",
			body:       base(`,"start_time":"10:00","end_time":"11:30","mode":"online"`),
			wantStatus: http.StatusBadRequest,
			wantError:  MessageLinkForOnlineMode,
		},
		{
			name:       "a room mode without a room",
			body:       base(`,"start_time":"10:00","end_time":"11:30","mode":"room"`),
			wantStatus: http.StatusBadRequest,
			wantError:  MessageRoomForRoomMode,
		},
		{
			name:       "a batch clash",
			body:       base(`,"start_time":"10:00","end_time":"11:30","mode":"room","room":"1003"`),
			storeErr:   &store.SlotConflict{Reason: store.ConflictBatch, Batch: "CSE-23", Section: "A", CourseCode: "CSE 2211", StartTime: "10:00", EndTime: "11:30"},
			wantStatus: http.StatusConflict,
			wantError:  "CSE-23 section A already has CSE 2211 at 10:00.",
		},
		{
			name:       "a room clash",
			body:       base(`,"start_time":"10:00","end_time":"11:30","mode":"room","room":"1003"`),
			storeErr:   &store.SlotConflict{Reason: store.ConflictRoom, Batch: "CSE-22", Section: "B", Room: "1003", StartTime: "10:30", EndTime: "12:00"},
			wantStatus: http.StatusConflict,
			wantError:  "Room 1003 is already booked for CSE-22 section B from 10:30 to 12:00.",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items := newFakeScheduleStore()
			items.err = test.storeErr
			items.teacherID = 3
			items.item.TeacherUserID = teacherUserID
			alerts := &fakeNotifier{}
			cr := newCRHandler(t, items, alerts)
			rec := scheduleCall(t, cr.CreateItem(), http.MethodPost, "/api/cr/items",
				"", test.body, crUserID, auth.RoleCR)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			if len(items.created) != 1 {
				t.Fatalf("the store received %d items, want 1", len(items.created))
			}
			if items.created[0].TeacherID != 3 {
				t.Errorf("teacher = %d, want the routine teacher 3", items.created[0].TeacherID)
			}
			if alerts.calls != 1 || alerts.userID != teacherUserID {
				t.Errorf("notifications = %d to user %d, want one to the teacher", alerts.calls, alerts.userID)
			}
		})
	}
}

func TestCRPatchValidation(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{name: "a note only", body: `{"note":"Bring the lab sheet."}`, wantStatus: http.StatusOK},
		{name: "a new slot", body: `{"start_time":"14:00","end_time":"15:30","mode":"room","room":"1003"}`,
			wantStatus: http.StatusOK},
		{name: "only a start time", body: `{"start_time":"14:00"}`, wantStatus: http.StatusBadRequest,
			wantError: MessageTimesTogether},
		{name: "only an end time", body: `{"end_time":"15:30"}`, wantStatus: http.StatusBadRequest,
			wantError: MessageTimesTogether},
		{name: "nothing", body: `{}`, wantStatus: http.StatusBadRequest, wantError: MessageNothingToUpdate},
		{name: "online without a link", body: `{"mode":"online"}`, wantStatus: http.StatusBadRequest,
			wantError: MessageLinkForOnlineMode},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items := newFakeScheduleStore()
			alerts := &fakeNotifier{}
			cr := newCRHandler(t, items, alerts)
			rec := scheduleCall(t, cr.UpdateItem(), http.MethodPatch, "/api/cr/items/4",
				"4", test.body, crUserID, auth.RoleCR)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			if len(items.patches) != 1 {
				t.Fatalf("the store received %d patches, want 1", len(items.patches))
			}
		})
	}
}

func TestCRPublish(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name           string
		body           string
		changed        []store.ScheduleItem
		storeErr       error
		wantStatus     int
		wantError      string
		wantBroadcasts int
	}{
		{name: "one item", body: `{"item_ids":[4]}`, wantStatus: http.StatusOK, wantBroadcasts: 1},
		{
			name:           "a changed item is announced too",
			body:           `{"item_ids":[4]}`,
			changed:        []store.ScheduleItem{publishedItem()},
			wantStatus:     http.StatusOK,
			wantBroadcasts: 2,
		},
		{name: "no ids", body: `{"item_ids":[]}`, wantStatus: http.StatusBadRequest, wantError: MessageIDsRequired},
		{
			name:       "an incomplete item",
			body:       `{"item_ids":[4]}`,
			storeErr:   &store.IncompleteItemError{Item: store.ScheduleItem{CourseCode: "CSE 2243", ItemDate: "2026-10-23"}},
			wantStatus: http.StatusBadRequest,
			wantError:  "CSE 2243 on 2026-10-23 still needs a date, a time and a room or a meeting link.",
		},
		{
			name:       "an item outside the final routine",
			body:       `{"item_ids":[4]}`,
			storeErr:   store.ErrItemState,
			wantStatus: http.StatusConflict,
			wantError:  MessageItemState,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items := newFakeScheduleStore()
			items.err = test.storeErr
			items.changed = test.changed
			items.publication = store.Publication{ID: 2, Batch: "CSE-23"}
			alerts := &fakeNotifier{}
			cr := newCRHandler(t, items, alerts)
			rec := scheduleCall(t, cr.Publish(), http.MethodPost, "/api/cr/final/publish",
				"", test.body, crUserID, auth.RoleCR)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			payload := decodeBody(t, rec)
			if payload["message"] != MessageRoutinePublished {
				t.Errorf("message = %v, want %q", payload["message"], MessageRoutinePublished)
			}
			if alerts.broadcasts != test.wantBroadcasts {
				t.Errorf("broadcasts = %d, want %d", alerts.broadcasts, test.wantBroadcasts)
			}
			if len(items.publishIDs) != 1 || items.publishIDs[0] != 4 {
				t.Errorf("the store received %v, want [4]", items.publishIDs)
			}
		})
	}
}

func publishedItem() store.ScheduleItem {
	item := plannedItem()
	item.Status = store.ItemConfirmed
	item.Snapshot = store.ItemSnapshot{
		Exists:    true,
		Status:    store.ItemConfirmed,
		ItemDate:  item.ItemDate,
		StartTime: "08:00",
		EndTime:   "09:30",
		Mode:      store.ModeRoom,
		Room:      "2104",
	}
	return item
}

func TestCRAllowedActions(t *testing.T) {
	tests := []struct {
		name    string
		item    store.ScheduleItem
		wantAny []string
	}{
		{
			name:    "a fixed item waiting for the teacher",
			item:    plannedItem(),
			wantAny: []string{ActionReschedule, ActionAddToFinal},
		},
		{
			name: "an item with no answer",
			item: func() store.ScheduleItem {
				item := plannedItem()
				item.Status = store.ItemNoResponse
				return item
			}(),
			wantAny: []string{ActionReschedule, ActionCancel, ActionAddToFinal},
		},
		{
			name: "an item the teacher wants moved",
			item: func() store.ScheduleItem {
				item := plannedItem()
				item.Status = store.ItemTeacherAsk
				return item
			}(),
			wantAny: []string{ActionSetSlot, ActionCancel},
		},
		{
			name: "an item of the class representative",
			item: func() store.ScheduleItem {
				item := plannedItem()
				item.Origin = store.OriginCR
				item.Status = store.ItemScheduled
				return item
			}(),
			wantAny: []string{ActionEdit, ActionCancel, ActionDelete},
		},
		{
			name: "a teacher exam",
			item: func() store.ScheduleItem {
				item := plannedItem()
				item.Origin = store.OriginTeacherExam
				item.Status = store.ItemScheduled
				return item
			}(),
			wantAny: []string{ActionSetTime},
		},
		{
			name: "an item already in the final routine",
			item: func() store.ScheduleItem {
				item := plannedItem()
				item.InFinal = true
				return item
			}(),
			wantAny: []string{ActionRemoveFromFinal},
		},
		{
			name: "a cancelled item",
			item: func() store.ScheduleItem {
				item := plannedItem()
				item.Status = store.ItemCancelled
				return item
			}(),
			wantAny: []string{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actions := crActions(test.item)
			if len(test.wantAny) == 0 {
				if len(actions) != 0 {
					t.Fatalf("actions = %v, want none", actions)
				}
				return
			}
			for _, want := range test.wantAny {
				found := false
				for _, action := range actions {
					if action == want {
						found = true
					}
				}
				if !found {
					t.Errorf("actions = %v, want %q among them", actions, want)
				}
			}
		})
	}
}

func TestScheduleLink(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name           string
		body           string
		published      bool
		storeErr       error
		wantStatus     int
		wantError      string
		wantBroadcasts int
	}{
		{name: "a new link", body: `{"meeting_link":"https://meet.example.com/new"}`, wantStatus: http.StatusOK},
		{
			name:           "a published item reaches the students",
			body:           `{"meeting_link":"https://meet.example.com/new"}`,
			published:      true,
			wantStatus:     http.StatusOK,
			wantBroadcasts: 1,
		},
		{name: "a broken link", body: `{"meeting_link":"meet"}`, wantStatus: http.StatusBadRequest,
			wantError: "Meeting link must be a full link that starts with http:// or https://."},
		{name: "an empty link", body: `{"meeting_link":""}`, wantStatus: http.StatusBadRequest,
			wantError: "Meeting link is required."},
		{name: "somebody else", body: `{"meeting_link":"https://meet.example.com/new"}`,
			storeErr: store.ErrItemNotFound, wantStatus: http.StatusNotFound, wantError: MessageItemNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items := newFakeScheduleStore()
			items.err = test.storeErr
			if test.published {
				items.item.Snapshot = store.ItemSnapshot{Exists: true, Status: store.ItemConfirmed}
			}
			alerts := &fakeNotifier{}
			schedule := NewSchedule(accountConfig(t, "test"), items, alerts)
			rec := scheduleCall(t, schedule.Link(), http.MethodPatch, "/api/schedule/items/4/link",
				"4", test.body, teacherUserID, auth.RoleTeacher)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			if items.linkUser != teacherUserID {
				t.Errorf("the store checked user %d, want %d", items.linkUser, teacherUserID)
			}
			if alerts.broadcasts != test.wantBroadcasts {
				t.Errorf("broadcasts = %d, want %d", alerts.broadcasts, test.wantBroadcasts)
			}
		})
	}
}

func TestNextDayCarriesTheWindow(t *testing.T) {
	silenceLog(t)
	items := newFakeScheduleStore()
	items.items = []store.ScheduleItem{plannedItem()}
	teacher := newTeacherHandler(t, items, &fakeNotifier{})
	rec := scheduleCall(t, teacher.NextDay(), http.MethodGet, "/api/teacher/upcoming/next-day",
		"", "", teacherUserID, auth.RoleTeacher)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	payload := decodeBody(t, rec)
	tomorrow := validate.DateString(accountConfig(t, "test").Now().AddDate(0, 0, 1))
	if payload["date"] != tomorrow {
		t.Errorf("date = %v, want tomorrow %q", payload["date"], tomorrow)
	}
	if payload["window_start"] != "" && payload["window_end"] != "" {
		if payload["decision_open"] == nil {
			t.Error("the response carries no decision_open flag")
		}
	}
	if len(items.items) != 1 {
		t.Fatalf("the fake store lost its items")
	}
}
