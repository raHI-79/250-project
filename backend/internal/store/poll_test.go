package store

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func seedPoll(t *testing.T, ctx context.Context, store *Store, creator int64, maxChoices int) PollRow {
	t.Helper()
	poll, err := store.CreatePoll(ctx, NewPoll{
		Question:   "Which day suits the extra class?",
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
