package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/raHI-79/250-project/backend/internal/daydate"
)

var ErrNotAHoliday = errors.New("store: no holiday or working day is set for that date")

// Holiday is one entry of the calendar the office keeps. A row with WorkingDay set
// turns an off day or a holiday back into a teaching day that follows FollowsWeekday;
// every other row stops the automatic generation of the fixed routine for its date.
type Holiday struct {
	Date       string
	Name       string
	WorkingDay bool
	Follows    int
	CreatedBy  int64
}

func (h Holiday) FollowsName() string {
	if !h.WorkingDay {
		return ""
	}
	return daydate.DayName(h.Follows)
}

// DayKind tells the views what to show at the top of a day.
type DayKind struct {
	Date       string
	Holiday    bool
	WorkingDay bool
	Name       string
	Follows    int
}

func (d DayKind) FollowsName() string {
	if !d.WorkingDay {
		return ""
	}
	return daydate.DayName(d.Follows)
}

const (
	listHolidaysSQL = `SELECT to_char(holiday_date, 'YYYY-MM-DD'), name, working_day,
COALESCE(follows_weekday, -1), COALESCE(created_by, 0)
FROM holidays
WHERE holiday_date >= $1::date
ORDER BY holiday_date ASC`
	listHolidaysBetweenSQL = `SELECT to_char(holiday_date, 'YYYY-MM-DD'), name, working_day,
COALESCE(follows_weekday, -1), COALESCE(created_by, 0)
FROM holidays
WHERE holiday_date >= $1::date AND holiday_date <= $2::date
ORDER BY holiday_date ASC`
	findHolidaySQL = `SELECT to_char(holiday_date, 'YYYY-MM-DD'), name, working_day,
COALESCE(follows_weekday, -1), COALESCE(created_by, 0)
FROM holidays
WHERE holiday_date = $1::date`
	saveHolidaySQL = `INSERT INTO holidays (holiday_date, name, working_day, follows_weekday, created_by)
VALUES ($1::date, $2, $3, NULLIF($4::int, -1), NULLIF($5::bigint, 0))
ON CONFLICT (holiday_date) DO UPDATE
SET name = EXCLUDED.name,
    working_day = EXCLUDED.working_day,
    follows_weekday = EXCLUDED.follows_weekday,
    updated_at = now()`
	deleteHolidaySQL = `DELETE FROM holidays WHERE holiday_date = $1::date`
	// An item counts as not yet decided while it still waits for its teacher and
	// while no class representative has taken it into a final routine.
	undecidedOnDateSQL = selectItem + `
WHERE i.item_date = $1::date AND i.origin = 'fixed'
  AND i.status IN ('awaiting_teacher', 'no_response')
  AND NOT i.in_final AND pi.item_id IS NULL
ORDER BY i.batch, i.section, i.start_time NULLS LAST, i.id`
	deleteUndecidedOnDateSQL = `DELETE FROM schedule_items
WHERE id = ANY($1::bigint[])`
)

func scanHoliday(row scanner) (Holiday, error) {
	var holiday Holiday
	err := row.Scan(&holiday.Date, &holiday.Name, &holiday.WorkingDay, &holiday.Follows,
		&holiday.CreatedBy)
	return holiday, err
}

func (s *Store) holidays(ctx context.Context, query string, args ...any) ([]Holiday, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the holidays: %w", err)
	}
	defer rows.Close()
	kept := make([]Holiday, 0)
	for rows.Next() {
		holiday, err := scanHoliday(rows)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a holiday: %w", err)
		}
		kept = append(kept, holiday)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not list the holidays: %w", err)
	}
	return kept, nil
}

func (s *Store) ListHolidays(ctx context.Context, from string) ([]Holiday, error) {
	return s.holidays(ctx, listHolidaysSQL, from)
}

func (s *Store) ListHolidaysBetween(ctx context.Context, from, to string) ([]Holiday, error) {
	return s.holidays(ctx, listHolidaysBetweenSQL, from, to)
}

func (s *Store) FindHoliday(ctx context.Context, date string) (Holiday, error) {
	holiday, err := scanHoliday(s.pool.QueryRow(ctx, findHolidaySQL, date))
	if errors.Is(err, pgx.ErrNoRows) {
		return Holiday{}, ErrNotAHoliday
	}
	if err != nil {
		return Holiday{}, fmt.Errorf("store: could not read the holiday of %s: %w", date, err)
	}
	return holiday, nil
}

// SaveHoliday writes the entry and, for a holiday that stops the generation, takes the
// automatically generated items of that date away again. It returns the items it
// removed so that the caller can tell their teacher and class representatives.
func (s *Store) SaveHoliday(ctx context.Context, holiday Holiday) ([]ScheduleItem, error) {
	if !daydate.Valid(holiday.Date) {
		return nil, fmt.Errorf("store: %q is not a date like 2026-11-06", holiday.Date)
	}
	follows := holiday.Follows
	if !holiday.WorkingDay {
		follows = -1
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("store: could not start the holiday transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, saveHolidaySQL, holiday.Date, holiday.Name, holiday.WorkingDay,
		follows, holiday.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("store: could not save the holiday of %s: %w", holiday.Date, err)
	}
	removed := make([]ScheduleItem, 0)
	if !holiday.WorkingDay {
		removed, err = itemsOn(ctx, tx, undecidedOnDateSQL, holiday.Date)
		if err != nil {
			return nil, err
		}
		if len(removed) > 0 {
			ids := make([]int64, 0, len(removed))
			for _, item := range removed {
				ids = append(ids, item.ID)
			}
			if _, err := tx.Exec(ctx, deleteUndecidedOnDateSQL, ids); err != nil {
				return nil, fmt.Errorf("store: could not clear the generated items of %s: %w",
					holiday.Date, err)
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("store: could not commit the holiday of %s: %w", holiday.Date, err)
	}
	return removed, nil
}

func (s *Store) DeleteHoliday(ctx context.Context, date string) error {
	tag, err := s.pool.Exec(ctx, deleteHolidaySQL, date)
	if err != nil {
		return fmt.Errorf("store: could not delete the holiday of %s: %w", date, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotAHoliday
	}
	return nil
}

// DayKinds answers, for every date in the range, whether it is an off day, a holiday or
// a working day that follows another weekday. offDays holds the weekday numbers that are
// weekly off days.
func (s *Store) DayKinds(ctx context.Context, from, to string, offDays []int) (map[string]DayKind, error) {
	entries, err := s.ListHolidaysBetween(ctx, from, to)
	if err != nil {
		return nil, err
	}
	byDate := make(map[string]Holiday, len(entries))
	for _, entry := range entries {
		byDate[entry.Date] = entry
	}
	kinds := make(map[string]DayKind)
	for _, date := range daydate.Range(from, to) {
		kind, ok := DayKindOf(date, byDate[date], offDays, len(byDate[date].Date) > 0)
		if ok {
			kinds[date] = kind
		}
	}
	return kinds, nil
}

// DayKindOf decides what a single date is. It is the one place that combines the weekly
// off days with the holiday calendar, so that every view agrees.
func DayKindOf(date string, entry Holiday, offDays []int, known bool) (DayKind, bool) {
	if known && entry.WorkingDay {
		return DayKind{Date: date, WorkingDay: true, Name: entry.Name, Follows: entry.Follows}, true
	}
	if known {
		return DayKind{Date: date, Holiday: true, Name: entry.Name}, true
	}
	if IsOffDay(date, offDays) {
		return DayKind{Date: date, Holiday: true, Name: daydate.WeekdayName(date)}, true
	}
	return DayKind{}, false
}

func IsOffDay(date string, offDays []int) bool {
	weekday := daydate.Weekday(date)
	for _, day := range offDays {
		if day == weekday {
			return true
		}
	}
	return false
}
