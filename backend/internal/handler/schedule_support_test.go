package handler

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type notifyCall struct {
	userID  int64
	role    string
	batch   string
	section string
	kind    string
	title   string
	body    string
	link    store.NotificationLink
}

type fakeNotifier struct {
	userID     int64
	role       string
	batch      string
	section    string
	title      string
	link       store.NotificationLink
	calls      int
	broadcasts int
	direct     []notifyCall
	sent       []notifyCall
	err        error
}

func (f *fakeNotifier) NotifyUser(ctx context.Context, userID int64, kind, title, body string,
	link store.NotificationLink) error {
	if f.err != nil {
		return f.err
	}
	f.userID, f.title, f.link = userID, title, link
	f.calls++
	call := notifyCall{userID: userID, kind: kind, title: title, body: body, link: link}
	f.direct = append(f.direct, call)
	f.sent = append(f.sent, call)
	return nil
}

func (f *fakeNotifier) PushUser(ctx context.Context, userID int64, kind, title, body string,
	link store.NotificationLink) error {
	return nil
}

func (f *fakeNotifier) NotifyBroadcast(ctx context.Context, role, batch, section, kind, title, body string,
	link store.NotificationLink) error {
	if f.err != nil {
		return f.err
	}
	f.role, f.batch, f.section, f.title, f.link = role, batch, section, title, link
	f.broadcasts++
	f.sent = append(f.sent, notifyCall{
		role: role, batch: batch, section: section, kind: kind, title: title, body: body, link: link,
	})
	return nil
}

func plannedItem() store.ScheduleItem {
	return store.ScheduleItem{
		ID:            4,
		Batch:         "CSE-23",
		Section:       "A",
		CourseID:      7,
		CourseCode:    "CSE 2243 7637D",
		CourseTitle:   "Compilers",
		TeacherID:     3,
		TeacherUserID: 44,
		TeacherName:   "Engine Teacher",
		Kind:          "class",
		Origin:        store.OriginFixed,
		RoutineID:     9,
		ItemDate:      "2026-10-23",
		StartTime:     "08:00",
		EndTime:       "09:30",
		Mode:          store.ModeRoom,
		Room:          "1003",
		Status:        store.ItemAwaiting,
	}
}

type fakeScheduleStore struct {
	item        store.ScheduleItem
	items       []store.ScheduleItem
	routines    []store.RoutineRow
	slots       []store.RoomSlot
	courseSlots []store.CourseSlotRow
	publication store.Publication
	changed     []store.ScheduleItem
	crs         []int64
	teacherID   int64
	err         error
	confirmed   []string
	cancelled   []string
	patches     []store.ItemPatch
	proposals   []store.Proposal
	created     []store.NewScheduleItem
	finals      map[int64]bool
	publishIDs  []int64
	deleted     []int64
	periods     []store.Period
	linkUser    int64
	links       []string
}

func newFakeScheduleStore() *fakeScheduleStore {
	return &fakeScheduleStore{item: plannedItem(), finals: make(map[int64]bool)}
}

func (f *fakeScheduleStore) GetTeacherByUser(ctx context.Context, userID int64) (store.Teacher, error) {
	return store.Teacher{ID: 3, UserID: userID, Name: "Engine Teacher"}, nil
}

func (f *fakeScheduleStore) GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error) {
	return store.StudentProfile{UserID: userID, StudentID: 2, Batch: "CSE-23", Section: "A", IsCR: true}, nil
}

func (f *fakeScheduleStore) ListTeacherRoutine(ctx context.Context, teacherID int64) ([]store.RoutineRow, error) {
	return f.routines, f.err
}

func (f *fakeScheduleStore) ListSectionRoutines(ctx context.Context, batch, section,
	semester string) ([]store.RoutineRow, error) {
	return f.routines, f.err
}

func (f *fakeScheduleStore) ListBatchCourseSlots(ctx context.Context, batch, section string) ([]store.CourseSlotRow, error) {
	return f.courseSlots, f.err
}

func (f *fakeScheduleStore) ListRoomAvailability(ctx context.Context, from, to string) ([]store.RoomSlot, error) {
	return f.slots, f.err
}

func (f *fakeScheduleStore) TeacherOfCourse(ctx context.Context, batch, section string, courseID int64) (int64, error) {
	return f.teacherID, nil
}

func (f *fakeScheduleStore) ListTeacherDayItems(ctx context.Context, teacherID int64,
	date string) ([]store.ScheduleItem, error) {
	return f.items, f.err
}

func (f *fakeScheduleStore) ListTeacherFinalItems(ctx context.Context, teacherID int64,
	from string) ([]store.ScheduleItem, error) {
	return f.items, f.err
}

func (f *fakeScheduleStore) ListTeacherExams(ctx context.Context, teacherID int64,
	from string) ([]store.ScheduleItem, error) {
	return f.items, f.err
}

func (f *fakeScheduleStore) ConfirmItem(ctx context.Context, itemID, teacherID int64, mode, room, link string,
	now time.Time) (store.ScheduleItem, error) {
	if f.err != nil {
		return store.ScheduleItem{}, f.err
	}
	f.confirmed = append(f.confirmed, mode+":"+room+":"+link)
	item := f.item
	item.Status = store.ItemConfirmed
	item.Mode, item.Room, item.MeetingLink = mode, room, link
	return item, nil
}

func (f *fakeScheduleStore) ListPeriods(ctx context.Context) ([]store.Period, error) {
	if len(f.periods) > 0 {
		return f.periods, nil
	}
	return testPeriods(), nil
}

func (f *fakeScheduleStore) CancelItemAsTeacher(ctx context.Context, itemID, teacherID int64,
	reason string, now time.Time) (store.ScheduleItem, error) {
	if f.err != nil {
		return store.ScheduleItem{}, f.err
	}
	f.cancelled = append(f.cancelled, reason)
	item := f.item
	item.Status = store.ItemCancelled
	item.CancelReason = reason
	return item, nil
}

func (f *fakeScheduleStore) TeacherRequestReschedule(ctx context.Context, itemID, teacherID int64,
	message string, now time.Time) (store.ScheduleItem, error) {
	if f.err != nil {
		return store.ScheduleItem{}, f.err
	}
	item := f.item
	item.Status = store.ItemTeacherAsk
	item.TeacherMessage = message
	return item, nil
}

func (f *fakeScheduleStore) AcceptProposal(ctx context.Context, itemID, teacherID int64) (store.ScheduleItem, error) {
	if f.err != nil {
		return store.ScheduleItem{}, f.err
	}
	item := f.item
	item.Status = store.ItemRescheduled
	return item, nil
}

func (f *fakeScheduleStore) DeclineProposal(ctx context.Context, itemID, teacherID int64) (store.ScheduleItem, error) {
	if f.err != nil {
		return store.ScheduleItem{}, f.err
	}
	item := f.item
	item.Status = store.ItemDeclined
	return item, nil
}

func (f *fakeScheduleStore) CreateTeacherExam(ctx context.Context,
	exam store.NewScheduleItem) (store.ScheduleItem, error) {
	if f.err != nil {
		return store.ScheduleItem{}, f.err
	}
	f.created = append(f.created, exam)
	item := f.item
	item.Origin = store.OriginTeacherExam
	item.Kind = store.KindExam
	item.Status = store.ItemScheduled
	item.StartTime, item.EndTime = "", ""
	item.ItemDate = exam.ItemDate
	item.Syllabus, item.Instructions = exam.Syllabus, exam.Instructions
	return item, nil
}

func (f *fakeScheduleStore) UpdateTeacherExam(ctx context.Context, itemID, teacherID int64,
	patch store.ItemPatch) (store.ScheduleItem, error) {
	if f.err != nil {
		return store.ScheduleItem{}, f.err
	}
	f.patches = append(f.patches, patch)
	return f.item, nil
}

func (f *fakeScheduleStore) CancelTeacherExam(ctx context.Context, itemID, teacherID int64,
	reason string) (store.ScheduleItem, error) {
	if f.err != nil {
		return store.ScheduleItem{}, f.err
	}
	f.cancelled = append(f.cancelled, reason)
	item := f.item
	item.Status = store.ItemCancelled
	return item, nil
}

func (f *fakeScheduleStore) DeleteTeacherExam(ctx context.Context, itemID, teacherID int64) (store.ScheduleItem, error) {
	if f.err != nil {
		return store.ScheduleItem{}, f.err
	}
	f.deleted = append(f.deleted, itemID)
	return f.item, nil
}

func (f *fakeScheduleStore) CreateCRItem(ctx context.Context, item store.NewScheduleItem) (store.ScheduleItem, error) {
	if f.err != nil {
		return store.ScheduleItem{}, f.err
	}
	f.created = append(f.created, item)
	created := f.item
	created.Origin = store.OriginCR
	created.Status = store.ItemScheduled
	created.Kind = item.Kind
	created.Mode, created.Room, created.MeetingLink = item.Mode, item.Room, item.MeetingLink
	created.StartTime, created.EndTime = item.StartTime, item.EndTime
	return created, nil
}

func (f *fakeScheduleStore) ListBatchItems(ctx context.Context, batch, from string) ([]store.ScheduleItem, error) {
	return f.items, f.err
}

func (f *fakeScheduleStore) CRRequestReschedule(ctx context.Context, itemID int64, batch string,
	proposal store.Proposal) (store.ScheduleItem, error) {
	if f.err != nil {
		return store.ScheduleItem{}, f.err
	}
	f.proposals = append(f.proposals, proposal)
	item := f.item
	item.Status = store.ItemCRAsk
	item.ProposedDate, item.ProposedStart, item.ProposedEnd = proposal.ItemDate, proposal.StartTime, proposal.EndTime
	item.ProposedMode, item.ProposedRoom, item.ProposedLink = proposal.Mode, proposal.Room, proposal.MeetingLink
	return item, nil
}

func (f *fakeScheduleStore) UpdateItemAsCR(ctx context.Context, itemID int64, batch string,
	patch store.ItemPatch) (store.ScheduleItem, error) {
	if f.err != nil {
		return store.ScheduleItem{}, f.err
	}
	f.patches = append(f.patches, patch)
	return f.item, nil
}

func (f *fakeScheduleStore) CancelItemAsCR(ctx context.Context, itemID int64, batch,
	reason string) (store.ScheduleItem, error) {
	if f.err != nil {
		return store.ScheduleItem{}, f.err
	}
	f.cancelled = append(f.cancelled, reason)
	item := f.item
	item.Status = store.ItemCancelled
	return item, nil
}

func (f *fakeScheduleStore) DeleteCRItem(ctx context.Context, itemID int64, batch string) (store.ScheduleItem, error) {
	if f.err != nil {
		return store.ScheduleItem{}, f.err
	}
	f.deleted = append(f.deleted, itemID)
	return f.item, nil
}

func (f *fakeScheduleStore) SetItemFinal(ctx context.Context, itemID int64, batch string,
	inFinal bool) (store.ScheduleItem, error) {
	if f.err != nil {
		return store.ScheduleItem{}, f.err
	}
	f.finals[itemID] = inFinal
	item := f.item
	item.InFinal = inFinal
	return item, nil
}

func (f *fakeScheduleStore) ListFinalItems(ctx context.Context, batch string) ([]store.ScheduleItem, error) {
	return f.items, f.err
}

func (f *fakeScheduleStore) ListItemsByIDs(ctx context.Context, batch string,
	ids []int64) ([]store.ScheduleItem, error) {
	return f.items, f.err
}

func (f *fakeScheduleStore) PublishFinal(ctx context.Context, batch string, publishedBy int64,
	ids []int64) (store.Publication, []store.ScheduleItem, error) {
	if f.err != nil {
		return store.Publication{}, nil, f.err
	}
	f.publishIDs = ids
	return f.publication, f.changed, nil
}

func (f *fakeScheduleStore) UpdateItemLink(ctx context.Context, itemID, userID int64,
	link string) (store.ScheduleItem, error) {
	if f.err != nil {
		return store.ScheduleItem{}, f.err
	}
	f.linkUser = userID
	f.links = append(f.links, link)
	item := f.item
	item.MeetingLink = link
	return item, nil
}

func (f *fakeScheduleStore) ListActiveCRs(ctx context.Context, batch, section string) ([]int64, error) {
	return f.crs, nil
}

func scheduleCall(t *testing.T, handlerFunc http.HandlerFunc, method, target, id, body string,
	userID int64, role string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, target, nil)
	if body != "" {
		request = httptest.NewRequest(method, target, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
	}
	if id != "" {
		request.SetPathValue("id", id)
	}
	return serve(handlerFunc, withCurrentUser(request, userID, role))
}
