package store

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"reflect"
	"strings"
	"testing"
)

func seedPoll(t *testing.T, ctx context.Context, store *Store, creator int64, maxChoices int) PollRow {
	t.Helper()
	poll, err := store.CreatePoll(ctx, NewPoll{
		Question:   "Which day suits the extra class?",
		Scope:      validate.PollScopeSection,
		Batch:      "CSE-23",
		Section:    "A",
		Options:    []string{"Sunday", "Monday", "Tuesday"},
		MaxChoices: maxChoices,
		CreatedBy:  creator,
	})
	if err != nil {
		t.Fatalf("CreatePoll() returned an unexpected error: %v", err)
	}
	return poll
}

func TestPollVotesFollowMaxChoices(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	crUser := insertUser(t, ctx, pool, "student", "2023331200", StatusActive, 0)
	insertStudent(t, ctx, pool, crUser, "2023331200", true)
	voter := insertUser(t, ctx, pool, "student", "2023331201", StatusActive, 0)
	insertStudent(t, ctx, pool, voter, "2023331201", false)
	poll := seedPoll(t, ctx, store, crUser, 2)
	other := seedPoll(t, ctx, store, crUser, 1)
	tests := []struct {
		name    string
		user    int64
		options []int64
		wantErr error
	}{
		{name: "no option at all", user: voter, options: nil, wantErr: ErrOptionNotInPoll},
		{
			name:    "an option of another poll",
			user:    voter,
			options: []int64{other.Options[0].ID},
			wantErr: ErrOptionNotInPoll,
		},
		{
			name:    "more options than allowed",
			user:    voter,
			options: []int64{poll.Options[0].ID, poll.Options[1].ID, poll.Options[2].ID},
			wantErr: ErrTooManyChoices,
		},
		{
			name:    "two allowed options",
			user:    voter,
			options: []int64{poll.Options[0].ID, poll.Options[1].ID},
		},
		{
			name:    "voting a second time",
			user:    voter,
			options: []int64{poll.Options[2].ID},
			wantErr: ErrAlreadyVoted,
		},
		{name: "another student votes once", user: crUser, options: []int64{poll.Options[0].ID}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := store.VoteInPoll(ctx, poll.ID, test.options, test.user, "CSE-23", "A")
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("VoteInPoll() error = %v, want %v", err, test.wantErr)
			}
		})
	}
	rows, err := store.ListStudentPolls(ctx, voter, "CSE-23", "A")
	if err != nil {
		t.Fatalf("ListStudentPolls() returned an unexpected error: %v", err)
	}
	var seen PollRow
	for _, row := range rows {
		if row.ID == poll.ID {
			seen = row
		}
	}
	if seen.TotalVoters != 2 {
		t.Errorf("total voters = %d, want 2", seen.TotalVoters)
	}
	if seen.MaxChoices != 2 {
		t.Errorf("max choices = %d, want 2", seen.MaxChoices)
	}
	if len(seen.MyOptionIDs) != 2 {
		t.Errorf("my options = %v, want the two the voter picked", seen.MyOptionIDs)
	}
	counts := make(map[string]int)
	for _, option := range seen.Options {
		counts[option.Text] = option.Votes
	}
	if counts["Sunday"] != 2 || counts["Monday"] != 1 || counts["Tuesday"] != 0 {
		t.Errorf("counts = %v, want Sunday 2, Monday 1 and Tuesday 0", counts)
	}
	creatorRows, err := store.ListCreatorPolls(ctx, crUser, crUser)
	if err != nil {
		t.Fatalf("ListCreatorPolls() returned an unexpected error: %v", err)
	}
	for _, row := range creatorRows {
		if row.ID != poll.ID {
			continue
		}
		if len(row.MyOptionIDs) != 1 {
			t.Errorf("the creator sees %v as their own vote, want only their own option", row.MyOptionIDs)
		}
	}
}

func TestPollRowsNeverCarryVoterIdentities(t *testing.T) {
	for _, value := range []any{PollRow{}, PollOptionRow{}} {
		shape := reflect.TypeOf(value)
		for index := 0; index < shape.NumField(); index++ {
			name := shape.Field(index).Name
			if name == "TotalVoters" || name == "MyOptionIDs" {
				continue
			}
			if strings.Contains(name, "Student") || strings.Contains(name, "Voter") ||
				strings.Contains(name, "User") {
				t.Errorf("%s.%s exposes who voted", shape.Name(), name)
			}
		}
	}
}

func TestPollAudienceFollowsTheScope(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	crUser := insertUser(t, ctx, pool, "student", "2023331300", StatusActive, 0)
	insertStudent(t, ctx, pool, crUser, "2023331300", true)
	sectionPoll := seedPoll(t, ctx, store, crUser, 1)
	batchPoll, err := store.CreatePoll(ctx, NewPoll{
		Question: "Which topics for the revision?", Scope: validate.PollScopeBatch,
		Batch: "CSE-23", Section: validate.SectionAll,
		Options: []string{"Indexing", "Transactions"}, CreatedBy: crUser,
	})
	if err != nil {
		t.Fatalf("CreatePoll() returned an unexpected error: %v", err)
	}
	everyBatch, err := store.CreatePoll(ctx, NewPoll{
		Question: "Which week for the seminar?", Scope: validate.PollScopeAll,
		Batch: validate.BatchAll, Section: validate.SectionAll,
		Options: []string{"First", "Second"}, CreatedBy: crUser,
	})
	if err != nil {
		t.Fatalf("CreatePoll() returned an unexpected error: %v", err)
	}
	readers := []struct {
		name    string
		regNo   string
		batch   string
		section string
		want    []int64
	}{
		{name: "the same section", regNo: "2023331301", batch: "CSE-23", section: "A",
			want: []int64{everyBatch.ID, batchPoll.ID, sectionPoll.ID}},
		{name: "the other section of the batch", regNo: "2023331302", batch: "CSE-23", section: "B",
			want: []int64{everyBatch.ID, batchPoll.ID}},
		{name: "another batch", regNo: "2022331303", batch: "CSE-22", section: "A",
			want: []int64{everyBatch.ID}},
	}
	for _, reader := range readers {
		t.Run(reader.name, func(t *testing.T) {
			userID := insertUser(t, ctx, pool, "student", reader.regNo, StatusActive, 0)
			query := `INSERT INTO students (user_id, full_name, reg_no, batch, section, email, phone, blood_group)
VALUES ($1, 'Poll Reader', $2, $3, $4, $2 || '@student.sust.edu', '01700000300', 'O+')`
			if _, err := pool.Exec(ctx, query, userID, reader.regNo, reader.batch, reader.section); err != nil {
				t.Fatalf("could not insert the reader: %v", err)
			}
			polls, err := store.ListStudentPolls(ctx, userID, reader.batch, reader.section)
			if err != nil {
				t.Fatalf("ListStudentPolls() returned an unexpected error: %v", err)
			}
			got := make([]int64, 0, len(polls))
			for _, poll := range polls {
				got = append(got, poll.ID)
			}
			if !reflect.DeepEqual(got, reader.want) {
				t.Errorf("polls = %v, want %v", got, reader.want)
			}
			if len(polls) > 0 && polls[0].Scope == "" {
				t.Error("the poll carries no scope")
			}
		})
	}
}

func TestVoteFollowsThePollAudience(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	crUser := insertUser(t, ctx, pool, "student", "2023331400", StatusActive, 0)
	insertStudent(t, ctx, pool, crUser, "2023331400", true)
	everyBatch, err := store.CreatePoll(ctx, NewPoll{
		Question: "Which week for the seminar?", Scope: validate.PollScopeAll,
		Batch: validate.BatchAll, Section: validate.SectionAll,
		Options: []string{"First", "Second"}, CreatedBy: crUser,
	})
	if err != nil {
		t.Fatalf("CreatePoll() returned an unexpected error: %v", err)
	}
	sectionPoll := seedPoll(t, ctx, store, crUser, 1)
	outsider := insertUser(t, ctx, pool, "student", "2022331401", StatusActive, 0)
	query := `INSERT INTO students (user_id, full_name, reg_no, batch, section, email, phone, blood_group)
VALUES ($1, 'Other Batch', '2022331401', 'CSE-22', 'B', 'other401@student.sust.edu', '01700000401', 'O+')`
	if _, err := pool.Exec(ctx, query, outsider); err != nil {
		t.Fatalf("could not insert the outsider: %v", err)
	}
	if err := store.VoteInPoll(ctx, everyBatch.ID, []int64{everyBatch.Options[0].ID},
		outsider, "CSE-22", "B"); err != nil {
		t.Errorf("a student of another batch could not vote in an all batches poll: %v", err)
	}
	err = store.VoteInPoll(ctx, sectionPoll.ID, []int64{sectionPoll.Options[0].ID},
		outsider, "CSE-22", "B")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("voting outside the audience: error = %v, want ErrNotFound", err)
	}
}
