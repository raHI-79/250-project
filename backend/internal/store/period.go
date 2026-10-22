package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/raHI-79/250-project/backend/internal/validate"
)

var (
	ErrPeriodOverlap = errors.New("store: this period overlaps another one")
	ErrPeriodInUse   = errors.New("store: this period still holds classes")
)

type Period struct {
	ID       int64
	Position int
	validate.Period
}

type NewPeriod struct {
	Label   string
	Start   string
	End     string
	IsBreak bool
}

type PeriodPatch struct {
	Label   *string
	Start   *string
	End     *string
	IsBreak *bool
}

const (
	periodColumns   = `id, label, to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI'), is_break, position`
	listPeriodsSQL  = `SELECT ` + periodColumns + ` FROM class_periods ORDER BY position, start_time`
	getPeriodSQL    = `SELECT ` + periodColumns + ` FROM class_periods WHERE id = $1`
	insertPeriodSQL = `INSERT INTO class_periods (label, start_time, end_time, is_break, position)
VALUES ($1, $2::time, $3::time, $4, COALESCE((SELECT MAX(position) FROM class_periods), 0) + 1)
RETURNING ` + periodColumns
	updatePeriodSQL = `UPDATE class_periods
SET label = COALESCE($2, label),
    start_time = COALESCE($3::time, start_time),
    end_time = COALESCE($4::time, end_time),
    is_break = COALESCE($5, is_break)
WHERE id = $1
RETURNING ` + periodColumns
	deletePeriodSQL  = `DELETE FROM class_periods WHERE id = $1 RETURNING id`
	parkPositionsSQL = `UPDATE class_periods SET position = -position WHERE position > 0`
	setPositionSQL   = `UPDATE class_periods SET position = $2 WHERE id = $1 RETURNING id`
	countPeriodsSQL  = `SELECT COUNT(*) FROM class_periods`
)

func (s *Store) ListPeriods(ctx context.Context) ([]Period, error) {
	rows, err := s.pool.Query(ctx, listPeriodsSQL)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the class periods: %w", err)
	}
	defer rows.Close()
	periods := make([]Period, 0)
	for rows.Next() {
		period, err := scanPeriod(rows)
		if err != nil {
			return nil, err
		}
		periods = append(periods, period)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the class periods: %w", err)
	}
	return periods, nil
}

func (s *Store) CreatePeriod(ctx context.Context, period NewPeriod) (Period, error) {
	existing, err := s.ListPeriods(ctx)
	if err != nil {
		return Period{}, err
	}
	candidate := validate.Period{Label: period.Label, Start: period.Start, End: period.End, IsBreak: period.IsBreak}
	if err := validate.PeriodFits(PeriodRules(existing), candidate); err != nil {
		return Period{}, fmt.Errorf("%w: %s", ErrPeriodOverlap, err.Error())
	}
	created, err := scanPeriod(s.pool.QueryRow(ctx, insertPeriodSQL,
		period.Label, period.Start, period.End, period.IsBreak))
	if err != nil {
		return Period{}, err
	}
	return created, nil
}

func (s *Store) UpdatePeriod(ctx context.Context, periodID int64, patch PeriodPatch) (Period, error) {
	existing, err := s.ListPeriods(ctx)
	if err != nil {
		return Period{}, err
	}
	current, others := splitPeriods(existing, periodID)
	if current.ID == 0 {
		return Period{}, ErrNotFound
	}
	candidate := current.Period
	if patch.Label != nil {
		candidate.Label = *patch.Label
	}
	if patch.Start != nil {
		candidate.Start = *patch.Start
	}
	if patch.End != nil {
		candidate.End = *patch.End
	}
	if patch.IsBreak != nil {
		candidate.IsBreak = *patch.IsBreak
	}
	if err := validate.PeriodFits(PeriodRules(others), candidate); err != nil {
		return Period{}, fmt.Errorf("%w: %s", ErrPeriodOverlap, err.Error())
	}
	updated, err := scanPeriod(s.pool.QueryRow(ctx, updatePeriodSQL,
		periodID, patch.Label, patch.Start, patch.End, patch.IsBreak))
	if errors.Is(err, pgx.ErrNoRows) {
		return Period{}, ErrNotFound
	}
	if err != nil {
		return Period{}, err
	}
	return updated, nil
}

func (s *Store) DeletePeriod(ctx context.Context, periodID int64) error {
	var count int
	if err := s.pool.QueryRow(ctx, countPeriodsSQL).Scan(&count); err != nil {
		return fmt.Errorf("store: could not count the class periods: %w", err)
	}
	if count <= 1 {
		return ErrPeriodInUse
	}
	var id int64
	err := s.pool.QueryRow(ctx, deletePeriodSQL, periodID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not delete the class period %d: %w", periodID, err)
	}
	return nil
}

func (s *Store) ReorderPeriods(ctx context.Context, ids []int64) ([]Period, error) {
	existing, err := s.ListPeriods(ctx)
	if err != nil {
		return nil, err
	}
	if len(ids) != len(existing) {
		return nil, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("store: could not start the period transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, parkPositionsSQL); err != nil {
		return nil, fmt.Errorf("store: could not park the period order: %w", err)
	}
	for index, id := range ids {
		var updated int64
		err := tx.QueryRow(ctx, setPositionSQL, id, index+1).Scan(&updated)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("store: could not move the class period %d: %w", id, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("store: could not commit the period order: %w", err)
	}
	return s.ListPeriods(ctx)
}

func PeriodRules(periods []Period) []validate.Period {
	rules := make([]validate.Period, 0, len(periods))
	for _, period := range periods {
		rules = append(rules, period.Period)
	}
	return rules
}

func splitPeriods(periods []Period, periodID int64) (Period, []Period) {
	var current Period
	others := make([]Period, 0, len(periods))
	for _, period := range periods {
		if period.ID == periodID {
			current = period
			continue
		}
		others = append(others, period)
	}
	return current, others
}

func scanPeriod(row pgx.Row) (Period, error) {
	var period Period
	err := row.Scan(&period.ID, &period.Label, &period.Start, &period.End, &period.IsBreak, &period.Position)
	if errors.Is(err, pgx.ErrNoRows) {
		return Period{}, ErrNotFound
	}
	if err != nil {
		return Period{}, fmt.Errorf("store: could not scan a class period: %w", err)
	}
	return period, nil
}
