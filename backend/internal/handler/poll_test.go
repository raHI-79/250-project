package handler

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

type fakePollStore struct {
	profile      store.StudentProfile
	created      []store.NewPoll
	polls        []store.PollRow
	closedID     int64
	closedBy     int64
	closeErr     error
	voted        []int64
	voteErr      error
	askedViewer  int64
	askedCreator int64
	askedBatch   string
	askedSection string
	err          error
}

func (f *fakePollStore) GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error) {
	return f.profile, nil
}

func (f *fakePollStore) CreatePoll(ctx context.Context, poll store.NewPoll) (store.PollRow, error) {
	f.created = append(f.created, poll)
	if f.err != nil {
		return store.PollRow{}, f.err
	}
	options := make([]store.PollOptionRow, 0, len(poll.Options))
	for index, text := range poll.Options {
		options = append(options, store.PollOptionRow{ID: int64(index + 1), Text: text})
	}
	return store.PollRow{
		ID:            3,
		Question:      poll.Question,
		Batch:         poll.Batch,
		Section:       poll.Section,
		Status:        store.PollOpen,
		CreatedByName: "Board CR",
		CreatedAt:     time.Now(),
		Options:       options,
	}, nil
}

func (f *fakePollStore) ClosePoll(ctx context.Context, pollID, createdBy int64, now time.Time) error {
	f.closedID, f.closedBy = pollID, createdBy
	return f.closeErr
}

func (f *fakePollStore) ListCreatorPolls(ctx context.Context, viewerUserID, createdBy int64) ([]store.PollRow, error) {
	f.askedViewer, f.askedCreator = viewerUserID, createdBy
	return f.polls, f.err
}

func (f *fakePollStore) ListStudentPolls(ctx context.Context, viewerUserID int64, batch, section string) ([]store.PollRow, error) {
	f.askedViewer, f.askedBatch, f.askedSection = viewerUserID, batch, section
	return f.polls, f.err
}

func (f *fakePollStore) VoteInPoll(ctx context.Context, pollID, optionID, userID int64, batch, section string) error {
	f.voted = append(f.voted, optionID)
	f.askedBatch, f.askedSection, f.askedViewer = batch, section, userID
	return f.voteErr
}

func newPollHandler(t *testing.T, polls PollStore, alerts BroadcastNotifier) *Poll {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load the timezone: %v", err)
	}
	return NewPoll(&config.Config{AppEnv: "test", Timezone: "Asia/Dhaka", Location: loc}, polls, alerts)
}

func crPollProfile(isCR bool) store.StudentProfile {
	return store.StudentProfile{UserID: noticeCRID, Batch: noticeBatch, Section: "A", IsCR: isCR}
}

func TestPollCRCreate(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		isCR        bool
		wantStatus  int
		wantSection string
		wantAlert   string
		wantError   string
	}{
		{
			name:        "for one section",
			body:        `{"question":"Which day suits the extra class?","options":["Sunday","Monday"],"scope":"section"}`,
			isCR:        true,
			wantStatus:  http.StatusCreated,
			wantSection: "A",
			wantAlert:   "A",
		},
		{
			name:        "for the whole batch",
			body:        `{"question":"Which day suits the extra class?","options":["Sunday","Monday"],"scope":"batch"}`,
			isCR:        true,
			wantStatus:  http.StatusCreated,
			wantSection: "ALL",
		},
		{
			name:       "without a question",
			body:       `{"question":"  ","options":["Sunday","Monday"],"scope":"batch"}`,
			isCR:       true,
			wantStatus: http.StatusBadRequest,
			wantError:  "Question is required.",
		},
		{
			name:       "with one option",
			body:       `{"question":"Which day?","options":["Sunday"],"scope":"batch"}`,
			isCR:       true,
			wantStatus: http.StatusBadRequest,
			wantError:  "Options must hold between 2 and 10 options.",
		},
		{
			name: "with eleven options",
			body: `{"question":"Which day?","options":["1","2","3","4","5","6","7","8","9","10","11"],` +
				`"scope":"batch"}`,
			isCR:       true,
			wantStatus: http.StatusBadRequest,
			wantError:  "Options must hold between 2 and 10 options.",
		},
		{
			name:       "with the same option twice",
			body:       `{"question":"Which day?","options":["Sunday","sunday"],"scope":"batch"}`,
			isCR:       true,
			wantStatus: http.StatusBadRequest,
			wantError:  "Options must not repeat the same option.",
		},
		{
			name:       "with an empty option",
			body:       `{"question":"Which day?","options":["Sunday","  "],"scope":"batch"}`,
			isCR:       true,
			wantStatus: http.StatusBadRequest,
			wantError:  "Options must not contain an empty option.",
		},
		{
			name:       "with an unknown scope",
			body:       `{"question":"Which day?","options":["Sunday","Monday"],"scope":"everyone"}`,
			isCR:       true,
			wantStatus: http.StatusBadRequest,
			wantError:  "Scope must be one of section, batch.",
		},
		{
			name:       "as a student who is not a class representative",
			body:       `{"question":"Which day?","options":["Sunday","Monday"],"scope":"batch"}`,
			wantStatus: http.StatusForbidden,
			wantError:  MessageNotACR,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			polls := &fakePollStore{profile: crPollProfile(tc.isCR)}
			alerts := &recordingAlerts{}
			req := jsonRequest(t, http.MethodPost, "/api/cr/polls", tc.body, noticeCRID, auth.RoleCR)
			rec := httptest.NewRecorder()
			newPollHandler(t, polls, alerts).CRCreate()(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusCreated {
				if got := errorOf(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
				if len(polls.created) != 0 {
					t.Errorf("the handler stored %d polls, want none", len(polls.created))
				}
				return
			}
			saved := polls.created[0]
			if saved.Batch != noticeBatch || saved.Section != tc.wantSection || saved.CreatedBy != noticeCRID {
				t.Errorf("stored poll = %+v, want %q/%q by %d", saved, noticeBatch, tc.wantSection, noticeCRID)
			}
			if len(saved.Options) != 2 {
				t.Errorf("stored %d options, want 2", len(saved.Options))
			}
			if alerts.calls != 1 || alerts.batch != noticeBatch || alerts.section != tc.wantAlert {
				t.Errorf("broadcast = %d to %q/%q, want one to %q/%q",
					alerts.calls, alerts.batch, alerts.section, noticeBatch, tc.wantAlert)
			}
		})
	}
}

func TestPollCRClose(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		closeErr   error
		wantStatus int
		wantError  string
	}{
		{name: "an open poll of this class representative", id: "3", wantStatus: http.StatusOK},
		{
			name:       "a poll of another class representative",
			id:         "4",
			closeErr:   store.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantError:  MessagePollUnknown,
		},
		{
			name:       "a poll that is already closed",
			id:         "3",
			closeErr:   store.ErrPollClosed,
			wantStatus: http.StatusConflict,
			wantError:  MessagePollClosed,
		},
		{name: "a broken id", id: "abc", wantStatus: http.StatusBadRequest, wantError: MessageInvalidID},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			polls := &fakePollStore{profile: crPollProfile(true), closeErr: tc.closeErr}
			req := jsonRequest(t, http.MethodPost, "/api/cr/polls/"+tc.id+"/close", "", noticeCRID, auth.RoleCR)
			req.SetPathValue("id", tc.id)
			rec := httptest.NewRecorder()
			newPollHandler(t, polls, &recordingAlerts{}).CRClose()(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantError != "" {
				if got := errorOf(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
			}
			if tc.wantStatus == http.StatusBadRequest {
				return
			}
			if polls.closedBy != noticeCRID {
				t.Errorf("closed on behalf of %d, want %d", polls.closedBy, noticeCRID)
			}
			if want, _ := strconv.ParseInt(tc.id, 10, 64); polls.closedID != want {
				t.Errorf("closed poll %d, want %d", polls.closedID, want)
			}
		})
	}
}

func TestPollLists(t *testing.T) {
	rows := []store.PollRow{
		{ID: 3, Question: "Which day?", Batch: noticeBatch, Section: "ALL", Status: store.PollOpen,
			CreatedByName: "Board CR", CreatedAt: time.Now(), TotalVotes: 1, MyOptionID: 2,
			Options: []store.PollOptionRow{{ID: 1, Text: "Sunday"}, {ID: 2, Text: "Monday", Votes: 1}}},
		{ID: 2, Question: "Lab slot?", Batch: noticeBatch, Section: "A", Status: store.PollClosed,
			CreatedByName: "Board CR", CreatedAt: time.Now(),
			Options: []store.PollOptionRow{{ID: 3, Text: "Morning"}, {ID: 4, Text: "Evening"}}},
	}
	tests := []struct {
		name        string
		path        string
		role        string
		pick        func(*Poll) http.HandlerFunc
		wantCreator int64
		wantBatch   string
		wantSection string
	}{
		{
			name:        "the polls a class representative made",
			path:        "/api/cr/polls",
			role:        auth.RoleCR,
			pick:        func(p *Poll) http.HandlerFunc { return p.CRList() },
			wantCreator: noticeCRID,
		},
		{
			name:        "the polls a student may answer",
			path:        "/api/students/polls",
			role:        auth.RoleStudent,
			pick:        func(p *Poll) http.HandlerFunc { return p.StudentList() },
			wantBatch:   noticeBatch,
			wantSection: "A",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			polls := &fakePollStore{profile: crPollProfile(true), polls: rows}
			req := jsonRequest(t, http.MethodGet, tc.path, "", noticeCRID, tc.role)
			rec := httptest.NewRecorder()
			tc.pick(newPollHandler(t, polls, &recordingAlerts{}))(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
			}
			if polls.askedViewer != noticeCRID || polls.askedCreator != tc.wantCreator {
				t.Errorf("asked as %d for the polls of %d, want %d and %d",
					polls.askedViewer, polls.askedCreator, noticeCRID, tc.wantCreator)
			}
			if polls.askedBatch != tc.wantBatch || polls.askedSection != tc.wantSection {
				t.Errorf("asked for %q/%q, want %q/%q", polls.askedBatch, polls.askedSection, tc.wantBatch, tc.wantSection)
			}
			items := noticeItems(t, rec)
			if len(items) != 2 {
				t.Fatalf("got %d items, want 2", len(items))
			}
			if items[0]["is_closed"] != false || items[0]["my_option_id"] != float64(2) || items[0]["total_votes"] != float64(1) {
				t.Errorf("first item = %v, want an open poll the viewer answered", items[0])
			}
			if items[1]["is_closed"] != true || items[1]["my_option_id"] != nil {
				t.Errorf("second item = %v, want a closed poll without a vote", items[1])
			}
			options, ok := items[0]["options"].([]any)
			if !ok || len(options) != 2 {
				t.Fatalf("the first poll has options %v, want two", items[0]["options"])
			}
			first, _ := options[1].(map[string]any)
			if first["text"] != "Monday" || first["votes"] != float64(1) {
				t.Errorf("second option = %v, want Monday with one vote", first)
			}
		})
	}
}

func TestPollStudentVote(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		voteErr    error
		wantStatus int
		wantError  string
	}{
		{name: "a first vote", body: `{"option_id":2}`, wantStatus: http.StatusCreated},
		{
			name:       "a second vote",
			body:       `{"option_id":2}`,
			voteErr:    store.ErrAlreadyVoted,
			wantStatus: http.StatusConflict,
			wantError:  MessageAlreadyVoted,
		},
		{
			name:       "a closed poll",
			body:       `{"option_id":2}`,
			voteErr:    store.ErrPollClosed,
			wantStatus: http.StatusConflict,
			wantError:  MessagePollClosed,
		},
		{
			name:       "an option of another poll",
			body:       `{"option_id":99}`,
			voteErr:    store.ErrOptionNotInPoll,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageOptionUnknown,
		},
		{
			name:       "a poll the student may not see",
			body:       `{"option_id":2}`,
			voteErr:    store.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantError:  MessagePollUnknown,
		},
		{
			name:       "without an option",
			body:       `{"option_id":0}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageOptionUnknown,
		},
		{name: "with a broken body", body: `{`, wantStatus: http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			polls := &fakePollStore{profile: crPollProfile(false), voteErr: tc.voteErr}
			req := jsonRequest(t, http.MethodPost, "/api/students/polls/3/vote", tc.body, noticeCRID, auth.RoleStudent)
			req.SetPathValue("id", "3")
			rec := httptest.NewRecorder()
			newPollHandler(t, polls, &recordingAlerts{}).StudentVote()(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantError != "" {
				if got := errorOf(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
			}
			if tc.wantStatus != http.StatusCreated {
				return
			}
			if len(polls.voted) != 1 || polls.voted[0] != 2 {
				t.Errorf("voted for %v, want option 2", polls.voted)
			}
			if polls.askedBatch != noticeBatch || polls.askedSection != "A" {
				t.Errorf("voted as %q/%q, want %q/A", polls.askedBatch, polls.askedSection, noticeBatch)
			}
		})
	}
}
