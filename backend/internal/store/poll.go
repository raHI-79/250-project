package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

const (
	PollOpen   = "open"
	PollClosed = "closed"
)

var (
	ErrPollClosed      = errors.New("store: this poll is already closed")
	ErrPollOpen        = errors.New("store: this poll is already open")
	ErrPollHasVotes    = errors.New("store: this poll already has votes")
	ErrAlreadyVoted    = errors.New("store: this student has already voted in this poll")
	ErrOptionNotInPoll = errors.New("store: this option does not belong to this poll")
	ErrTooManyChoices  = errors.New("store: this poll allows fewer choices than that")
)

type NewPoll struct {
	Question   string
	Scope      string
	Batch      string
	Section    string
	Options    []string
	MaxChoices int
	CreatedBy  int64
}

type PollOptionRow struct {
	ID    int64
	Text  string
	Votes int
}

type PollRow struct {
	ID            int64
	Question      string
	Scope         string
	Batch         string
	Section       string
	Status        string
	MaxChoices    int
	CreatedByName string
	CreatedAt     time.Time
	EditedAt      *time.Time
	Options       []PollOptionRow
	TotalVoters   int
	MyOptionIDs   []int64
}

const (
	insertPollSQL = `INSERT INTO polls (question, scope, batch, section, max_choices, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, created_at`
	insertPollOptionSQL = `INSERT INTO poll_options (poll_id, option_text, position)
VALUES ($1, $2, $3)
RETURNING id`
	closePollSQL = `UPDATE polls
SET status = 'closed', closed_at = $3
WHERE id = $1 AND created_by = $2 AND status = 'open'
RETURNING id`
	reopenPollSQL = `UPDATE polls
SET status = 'open', closed_at = NULL
WHERE id = $1 AND created_by = $2 AND status = 'closed'
RETURNING id`
	deletePollSQL    = `DELETE FROM polls WHERE id = $1 AND created_by = $2 RETURNING id`
	pollVoteCountSQL = `SELECT COUNT(*) FROM poll_votes WHERE poll_id = $1`
	editPollSQL      = `UPDATE polls
SET question = $3, max_choices = $4, edited_at = now()
WHERE id = $1 AND created_by = $2
RETURNING id`
	deletePollOptionsSQL = `DELETE FROM poll_options WHERE poll_id = $1`
	pollStatusSQL        = `SELECT status FROM polls WHERE id = $1 AND created_by = $2`
	pollColumns          = `p.id, p.question, p.scope, p.batch, p.section, p.status, p.max_choices,
COALESCE(s.full_name, t.name, CASE WHEN p.created_by IS NULL THEN 'Deleted user' ELSE 'Administration' END),
p.created_at, p.edited_at,
(SELECT COUNT(DISTINCT v.student_id) FROM poll_votes v WHERE v.poll_id = p.id),
ARRAY(SELECT v.option_id FROM poll_votes v JOIN students vs ON vs.id = v.student_id
      WHERE v.poll_id = p.id AND vs.user_id = $1 ORDER BY v.option_id)`
	pollAudience = `(p.scope = 'all_batches'
    OR (p.batch = $2::text AND (p.section = 'ALL' OR p.section = $3::text)))`
	pollJoins = `FROM polls p
LEFT JOIN users u ON u.id = p.created_by
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id`
	listCreatorPollsSQL = `SELECT ` + pollColumns + `
` + pollJoins + `
WHERE p.created_by = $2
ORDER BY p.created_at DESC, p.id DESC`
	listStudentPollsSQL = `SELECT ` + pollColumns + `
` + pollJoins + `
WHERE ` + pollAudience + `
ORDER BY p.created_at DESC, p.id DESC`
	listPollOptionsSQL = `SELECT o.poll_id, o.id, o.option_text,
(SELECT COUNT(*) FROM poll_votes v WHERE v.option_id = o.id)
FROM poll_options o
WHERE o.poll_id = ANY($1)
ORDER BY o.poll_id, o.position, o.id`
	votablePollSQL = `SELECT p.status, p.max_choices,
(SELECT COUNT(*) FROM poll_options o WHERE o.poll_id = p.id AND o.id = ANY($2::bigint[])),
COALESCE((SELECT vs.id FROM students vs WHERE vs.user_id = $3), 0),
EXISTS (SELECT 1 FROM poll_votes v JOIN students vs ON vs.id = v.student_id
        WHERE v.poll_id = p.id AND vs.user_id = $3)
FROM polls p
WHERE p.id = $1 AND (p.scope = 'all_batches'
    OR (p.batch = $4::text AND (p.section = 'ALL' OR p.section = $5::text)))`
	insertPollVoteSQL = `INSERT INTO poll_votes (poll_id, option_id, student_id)
VALUES ($1, $2, $3)
RETURNING id`
)

func (s *Store) CreatePoll(ctx context.Context, poll NewPoll) (PollRow, error) {
	if poll.MaxChoices < 1 {
		poll.MaxChoices = 1
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PollRow{}, fmt.Errorf("store: could not start the poll transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	row := PollRow{
		Question:    poll.Question,
		Scope:       poll.Scope,
		Batch:       poll.Batch,
		Section:     poll.Section,
		Status:      PollOpen,
		MaxChoices:  poll.MaxChoices,
		Options:     make([]PollOptionRow, 0, len(poll.Options)),
		MyOptionIDs: make([]int64, 0),
	}
	err = tx.QueryRow(ctx, insertPollSQL,
		poll.Question, poll.Scope, poll.Batch, poll.Section, poll.MaxChoices, poll.CreatedBy).
		Scan(&row.ID, &row.CreatedAt)
	if err != nil {
		return PollRow{}, fmt.Errorf("store: could not create the poll: %w", err)
	}
	for position, text := range poll.Options {
		var optionID int64
		if err := tx.QueryRow(ctx, insertPollOptionSQL, row.ID, text, position).Scan(&optionID); err != nil {
			return PollRow{}, fmt.Errorf("store: could not create a poll option: %w", err)
		}
		row.Options = append(row.Options, PollOptionRow{ID: optionID, Text: text})
	}
	if err := tx.Commit(ctx); err != nil {
		return PollRow{}, fmt.Errorf("store: could not commit the poll: %w", err)
	}
	return row, nil
}

func (s *Store) ClosePoll(ctx context.Context, pollID, createdBy int64, now time.Time) error {
	var id int64
	err := s.pool.QueryRow(ctx, closePollSQL, pollID, createdBy, now).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		var status string
		statusErr := s.pool.QueryRow(ctx, pollStatusSQL, pollID, createdBy).Scan(&status)
		if errors.Is(statusErr, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if statusErr != nil {
			return fmt.Errorf("store: could not read the status of the poll %d: %w", pollID, statusErr)
		}
		return ErrPollClosed
	}
	if err != nil {
		return fmt.Errorf("store: could not close the poll %d: %w", pollID, err)
	}
	return nil
}

func (s *Store) ReopenPoll(ctx context.Context, pollID, createdBy int64) error {
	var id int64
	err := s.pool.QueryRow(ctx, reopenPollSQL, pollID, createdBy).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		var status string
		statusErr := s.pool.QueryRow(ctx, pollStatusSQL, pollID, createdBy).Scan(&status)
		if errors.Is(statusErr, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if statusErr != nil {
			return fmt.Errorf("store: could not read the status of the poll %d: %w", pollID, statusErr)
		}
		return ErrPollOpen
	}
	if err != nil {
		return fmt.Errorf("store: could not reopen the poll %d: %w", pollID, err)
	}
	return nil
}

func (s *Store) DeletePoll(ctx context.Context, pollID, createdBy int64) error {
	var id int64
	err := s.pool.QueryRow(ctx, deletePollSQL, pollID, createdBy).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not delete the poll %d: %w", pollID, err)
	}
	return nil
}

func (s *Store) EditPoll(ctx context.Context, pollID, createdBy int64, question string,
	maxChoices int, options []string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: could not start the poll edit: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var votes int
	if err := tx.QueryRow(ctx, pollVoteCountSQL, pollID).Scan(&votes); err != nil {
		return fmt.Errorf("store: could not count the votes of the poll %d: %w", pollID, err)
	}
	if votes > 0 {
		return ErrPollHasVotes
	}
	var id int64
	err = tx.QueryRow(ctx, editPollSQL, pollID, createdBy, question, maxChoices).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not edit the poll %d: %w", pollID, err)
	}
	if _, err := tx.Exec(ctx, deletePollOptionsSQL, pollID); err != nil {
		return fmt.Errorf("store: could not clear the options of the poll %d: %w", pollID, err)
	}
	for position, text := range options {
		var optionID int64
		if err := tx.QueryRow(ctx, insertPollOptionSQL, pollID, text, position).Scan(&optionID); err != nil {
			return fmt.Errorf("store: could not add a poll option: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store: could not commit the poll edit: %w", err)
	}
	return nil
}

func (s *Store) ListCreatorPolls(ctx context.Context, viewerUserID, createdBy int64) ([]PollRow, error) {
	return s.polls(ctx, listCreatorPollsSQL, viewerUserID, createdBy)
}

func (s *Store) ListStudentPolls(ctx context.Context, viewerUserID int64, batch, section string) ([]PollRow, error) {
	return s.polls(ctx, listStudentPollsSQL, viewerUserID, batch, section)
}

func (s *Store) VoteInPoll(ctx context.Context, pollID int64, optionIDs []int64, userID int64,
	batch, section string) error {
	if len(optionIDs) == 0 {
		return ErrOptionNotInPoll
	}
	var status string
	var maxChoices, belonging int
	var studentID int64
	var voted bool
	err := s.pool.QueryRow(ctx, votablePollSQL, pollID, optionIDs, userID, batch, section).
		Scan(&status, &maxChoices, &belonging, &studentID, &voted)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not load the poll %d: %w", pollID, err)
	}
	if studentID == 0 {
		return ErrNotFound
	}
	if status == PollClosed {
		return ErrPollClosed
	}
	if voted {
		return ErrAlreadyVoted
	}
	if belonging != len(optionIDs) {
		return ErrOptionNotInPoll
	}
	if len(optionIDs) > maxChoices {
		return ErrTooManyChoices
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: could not start the vote transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, optionID := range optionIDs {
		var voteID int64
		err := tx.QueryRow(ctx, insertPollVoteSQL, pollID, optionID, studentID).Scan(&voteID)
		if isUniqueViolation(err) {
			return ErrAlreadyVoted
		}
		if err != nil {
			return fmt.Errorf("store: could not save the vote in the poll %d: %w", pollID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store: could not commit the vote in the poll %d: %w", pollID, err)
	}
	return nil
}

func (s *Store) polls(ctx context.Context, query string, args ...any) ([]PollRow, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the polls: %w", err)
	}
	defer rows.Close()
	polls := make([]PollRow, 0)
	ids := make([]int64, 0)
	for rows.Next() {
		var poll PollRow
		err := rows.Scan(
			&poll.ID,
			&poll.Question,
			&poll.Scope,
			&poll.Batch,
			&poll.Section,
			&poll.Status,
			&poll.MaxChoices,
			&poll.CreatedByName,
			&poll.CreatedAt,
			&poll.EditedAt,
			&poll.TotalVoters,
			&poll.MyOptionIDs,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a poll: %w", err)
		}
		poll.Options = make([]PollOptionRow, 0)
		polls = append(polls, poll)
		ids = append(ids, poll.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the polls: %w", err)
	}
	if len(polls) == 0 {
		return polls, nil
	}
	options, err := s.pollOptions(ctx, ids)
	if err != nil {
		return nil, err
	}
	for index := range polls {
		polls[index].Options = append(polls[index].Options, options[polls[index].ID]...)
	}
	return polls, nil
}

func (s *Store) pollOptions(ctx context.Context, pollIDs []int64) (map[int64][]PollOptionRow, error) {
	rows, err := s.pool.Query(ctx, listPollOptionsSQL, pollIDs)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the poll options: %w", err)
	}
	defer rows.Close()
	options := make(map[int64][]PollOptionRow)
	for rows.Next() {
		var pollID int64
		var option PollOptionRow
		if err := rows.Scan(&pollID, &option.ID, &option.Text, &option.Votes); err != nil {
			return nil, fmt.Errorf("store: could not scan a poll option: %w", err)
		}
		options[pollID] = append(options[pollID], option)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the poll options: %w", err)
	}
	return options, nil
}
