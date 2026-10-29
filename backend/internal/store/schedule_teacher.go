package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

const (
	lockItemSQL = selectItem + `
WHERE i.id = $1
FOR UPDATE OF i`
	listTeacherRoutineSQL = `SELECT r.id, r.batch, r.semester, r.section, r.course_id, c.code, c.title,
COALESCE(r.teacher_id, 0), COALESCE(t.name, 'Deleted user'),
r.day_of_week, to_char(r.start_time, 'HH24:MI'), to_char(r.end_time, 'HH24:MI'), r.room, r.type
FROM routines r
JOIN courses c ON c.id = r.course_id
JOIN batch_semesters bs ON bs.batch = r.batch AND bs.running_semester = r.semester
LEFT JOIN teachers t ON t.id = r.teacher_id
WHERE r.teacher_id = $1
ORDER BY r.day_of_week, r.start_time, r.batch, r.section`
	listTeacherDayItemsSQL = selectItem + `
WHERE i.teacher_id = $1 AND i.item_date = $2::date
ORDER BY i.start_time NULLS LAST, i.batch, i.section, i.id`
	listTeacherFinalSQL = selectItem + `
WHERE i.teacher_id = $1 AND i.item_date >= $2::date
  AND (i.status IN ('confirmed', 'rescheduled', 'reschedule_declined') OR i.in_final OR pi.item_id IS NOT NULL)
ORDER BY i.item_date, i.start_time NULLS LAST, i.id`
	listTeacherExamsSQL = selectItem + `
WHERE i.teacher_id = $1 AND i.origin = 'teacher_exam' AND i.item_date >= $2::date
ORDER BY i.item_date, i.start_time NULLS LAST, i.id`
	confirmItemSQL = `UPDATE schedule_items
SET status = $2, mode = $3, room = $4, meeting_link = $5,
    teacher_message = '', prior_status = '',
    proposed_date = NULL, proposed_start = NULL, proposed_end = NULL,
    proposed_mode = NULL, proposed_room = '', proposed_link = '',
    updated_at = now()
WHERE id = $1`
	cancelItemSQL = `UPDATE schedule_items
SET status = 'cancelled', cancel_reason = $2, prior_status = '',
    proposed_date = NULL, proposed_start = NULL, proposed_end = NULL,
    proposed_mode = NULL, proposed_room = '', proposed_link = '',
    updated_at = now()
WHERE id = $1`
	teacherAskRescheduleSQL = `UPDATE schedule_items
SET status = 'teacher_reschedule_request', teacher_message = $2, prior_status = $3, updated_at = now()
WHERE id = $1`
	applyProposalSQL = `UPDATE schedule_items
SET status = 'rescheduled',
    item_date = $2::date,
    start_time = $3::time,
    end_time = $4::time,
    mode = $5,
    room = $6,
    meeting_link = $7,
    prior_status = '',
    proposed_date = NULL, proposed_start = NULL, proposed_end = NULL,
    proposed_mode = NULL, proposed_room = '', proposed_link = '',
    updated_at = now()
WHERE id = $1`
	declineProposalSQL = `UPDATE schedule_items
SET status = $2, prior_status = '',
    proposed_date = NULL, proposed_start = NULL, proposed_end = NULL,
    proposed_mode = NULL, proposed_room = '', proposed_link = '',
    updated_at = now()
WHERE id = $1`
	updateTeacherExamSQL = `UPDATE schedule_items
SET item_date = COALESCE($2::date, item_date),
    syllabus = COALESCE($3, syllabus),
    instructions = COALESCE($4, instructions),
    updated_at = now()
WHERE id = $1`
	deleteItemSQL = `DELETE FROM schedule_items WHERE id = $1`
)

func (s *Store) withItem(ctx context.Context, itemID int64,
	change func(context.Context, pgx.Tx, ScheduleItem) error) (ScheduleItem, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ScheduleItem{}, fmt.Errorf("store: could not start the schedule item transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	item, err := scanItem(tx.QueryRow(ctx, lockItemSQL, itemID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ScheduleItem{}, ErrItemNotFound
	}
	if err != nil {
		return ScheduleItem{}, fmt.Errorf("store: could not lock the schedule item %d: %w", itemID, err)
	}
	if err := change(ctx, tx, item); err != nil {
		return ScheduleItem{}, err
	}
	updated, err := scanItem(tx.QueryRow(ctx, findItemSQL, itemID))
	if errors.Is(err, pgx.ErrNoRows) {
		updated = item
	} else if err != nil {
		return ScheduleItem{}, fmt.Errorf("store: could not read back the schedule item %d: %w", itemID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ScheduleItem{}, fmt.Errorf("store: could not commit the schedule item change: %w", err)
	}
	return updated, nil
}

func (s *Store) ListTeacherRoutine(ctx context.Context, teacherID int64) ([]RoutineRow, error) {
	rows, err := s.pool.Query(ctx, listTeacherRoutineSQL, teacherID)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the routine of the teacher %d: %w", teacherID, err)
	}
	defer rows.Close()
	entries := make([]RoutineRow, 0)
	for rows.Next() {
		var entry RoutineRow
		err := rows.Scan(
			&entry.ID,
			&entry.Batch,
			&entry.Semester,
			&entry.Section,
			&entry.CourseID,
			&entry.CourseCode,
			&entry.CourseTitle,
			&entry.TeacherID,
			&entry.TeacherName,
			&entry.DayOfWeek,
			&entry.StartTime,
			&entry.EndTime,
			&entry.Room,
			&entry.Type,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a routine entry: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the routine entries: %w", err)
	}
	return entries, nil
}

func (s *Store) ListTeacherDayItems(ctx context.Context, teacherID int64, date string) ([]ScheduleItem, error) {
	return s.items(ctx, listTeacherDayItemsSQL, teacherID, date)
}

func (s *Store) ListTeacherFinalItems(ctx context.Context, teacherID int64, from string) ([]ScheduleItem, error) {
	return s.items(ctx, listTeacherFinalSQL, teacherID, from)
}

func (s *Store) ListTeacherExams(ctx context.Context, teacherID int64, from string) ([]ScheduleItem, error) {
	return s.items(ctx, listTeacherExamsSQL, teacherID, from)
}

func ownedByTeacher(item ScheduleItem, teacherID int64) error {
	if item.TeacherID != teacherID {
		return ErrItemNotFound
	}
	return nil
}

func started(item ScheduleItem, now time.Time) bool {
	if !item.HasTimes() {
		return false
	}
	return item.ItemDate+" "+item.StartTime < now.Format("2006-01-02 15:04")
}

func (s *Store) ConfirmItem(ctx context.Context, itemID, teacherID int64, mode, room, link string,
	now time.Time) (ScheduleItem, error) {
	return s.withItem(ctx, itemID, func(ctx context.Context, tx pgx.Tx, item ScheduleItem) error {
		if err := ownedByTeacher(item, teacherID); err != nil {
			return err
		}
		switch item.Status {
		case ItemAwaiting, ItemNoResponse, ItemTeacherAsk, ItemConfirmed, ItemDeclined, ItemRescheduled:
		default:
			return ErrItemState
		}
		if started(item, now) {
			return ErrItemStarted
		}
		status := ItemConfirmed
		if item.Status == ItemRescheduled {
			status = ItemRescheduled
		}
		err := checkSlot(ctx, tx, slotRequest{
			ExcludeID: item.ID,
			Batch:     item.Batch,
			Section:   item.Section,
			ItemDate:  item.ItemDate,
			StartTime: item.StartTime,
			EndTime:   item.EndTime,
			Mode:      mode,
			Room:      room,
		})
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, confirmItemSQL, item.ID, status, mode, room, link); err != nil {
			return fmt.Errorf("store: could not confirm the schedule item %d: %w", item.ID, err)
		}
		return nil
	})
}

func (s *Store) CancelItemAsTeacher(ctx context.Context, itemID, teacherID int64, reason string,
	now time.Time) (ScheduleItem, error) {
	return s.withItem(ctx, itemID, func(ctx context.Context, tx pgx.Tx, item ScheduleItem) error {
		if err := ownedByTeacher(item, teacherID); err != nil {
			return err
		}
		if item.Status == ItemCancelled {
			return ErrItemState
		}
		if started(item, now) {
			return ErrItemStarted
		}
		return cancelAndSync(ctx, tx, item, reason)
	})
}

func (s *Store) TeacherRequestReschedule(ctx context.Context, itemID, teacherID int64, message string,
	now time.Time) (ScheduleItem, error) {
	return s.withItem(ctx, itemID, func(ctx context.Context, tx pgx.Tx, item ScheduleItem) error {
		if err := ownedByTeacher(item, teacherID); err != nil {
			return err
		}
		switch item.Status {
		case ItemAwaiting, ItemNoResponse, ItemConfirmed, ItemDeclined:
		default:
			return ErrItemState
		}
		if started(item, now) {
			return ErrItemStarted
		}
		if _, err := tx.Exec(ctx, teacherAskRescheduleSQL, item.ID, message, item.Status); err != nil {
			return fmt.Errorf("store: could not ask for a reschedule of the item %d: %w", item.ID, err)
		}
		return nil
	})
}

func (s *Store) AcceptProposal(ctx context.Context, itemID, teacherID int64) (ScheduleItem, error) {
	return s.withItem(ctx, itemID, func(ctx context.Context, tx pgx.Tx, item ScheduleItem) error {
		if err := ownedByTeacher(item, teacherID); err != nil {
			return err
		}
		if item.Status != ItemCRAsk {
			return ErrItemState
		}
		if item.ProposedDate == "" || item.ProposedStart == "" || item.ProposedEnd == "" {
			return ErrNoProposal
		}
		mode := item.ProposedMode
		if mode == "" {
			mode = item.Mode
		}
		err := checkSlot(ctx, tx, slotRequest{
			ExcludeID: item.ID,
			Batch:     item.Batch,
			Section:   item.Section,
			ItemDate:  item.ProposedDate,
			StartTime: item.ProposedStart,
			EndTime:   item.ProposedEnd,
			Mode:      mode,
			Room:      item.ProposedRoom,
		})
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, applyProposalSQL, item.ID, item.ProposedDate, item.ProposedStart,
			item.ProposedEnd, mode, item.ProposedRoom, item.ProposedLink)
		if err != nil {
			return fmt.Errorf("store: could not apply the proposal of the item %d: %w", item.ID, err)
		}
		return nil
	})
}

func (s *Store) DeclineProposal(ctx context.Context, itemID, teacherID int64) (ScheduleItem, error) {
	return s.withItem(ctx, itemID, func(ctx context.Context, tx pgx.Tx, item ScheduleItem) error {
		if err := ownedByTeacher(item, teacherID); err != nil {
			return err
		}
		if item.Status != ItemCRAsk {
			return ErrItemState
		}
		status := ItemDeclined
		if item.PriorStatus == ItemConfirmed || item.PriorStatus == ItemRescheduled {
			status = item.PriorStatus
		}
		if _, err := tx.Exec(ctx, declineProposalSQL, item.ID, status); err != nil {
			return fmt.Errorf("store: could not decline the proposal of the item %d: %w", item.ID, err)
		}
		return nil
	})
}

func (s *Store) CreateTeacherExam(ctx context.Context, exam NewScheduleItem) (ScheduleItem, error) {
	exam.Kind = KindExam
	exam.Origin = OriginTeacherExam
	exam.Status = ItemScheduled
	exam.Mode = ModeRoom
	return s.insertItem(ctx, exam)
}

func (s *Store) UpdateTeacherExam(ctx context.Context, itemID, teacherID int64, patch ItemPatch) (ScheduleItem, error) {
	return s.withItem(ctx, itemID, func(ctx context.Context, tx pgx.Tx, item ScheduleItem) error {
		if err := teacherExam(item, teacherID); err != nil {
			return err
		}
		date := item.ItemDate
		if patch.ItemDate != nil {
			date = *patch.ItemDate
		}
		err := checkSlot(ctx, tx, slotRequest{
			ExcludeID: item.ID,
			Batch:     item.Batch,
			Section:   item.Section,
			ItemDate:  date,
			StartTime: item.StartTime,
			EndTime:   item.EndTime,
			Mode:      item.Mode,
			Room:      item.Room,
		})
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, updateTeacherExamSQL, item.ID, patch.ItemDate, patch.Syllabus, patch.Instructions)
		if err != nil {
			return fmt.Errorf("store: could not update the exam %d: %w", item.ID, err)
		}
		return nil
	})
}

func (s *Store) CancelTeacherExam(ctx context.Context, itemID, teacherID int64, reason string) (ScheduleItem, error) {
	return s.withItem(ctx, itemID, func(ctx context.Context, tx pgx.Tx, item ScheduleItem) error {
		if err := teacherExam(item, teacherID); err != nil {
			return err
		}
		if item.Status == ItemCancelled {
			return ErrItemState
		}
		return cancelAndSync(ctx, tx, item, reason)
	})
}

func (s *Store) DeleteTeacherExam(ctx context.Context, itemID, teacherID int64) (ScheduleItem, error) {
	return s.withItem(ctx, itemID, func(ctx context.Context, tx pgx.Tx, item ScheduleItem) error {
		if err := teacherExam(item, teacherID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, deleteItemSQL, item.ID); err != nil {
			return fmt.Errorf("store: could not delete the exam %d: %w", item.ID, err)
		}
		return nil
	})
}

func teacherExam(item ScheduleItem, teacherID int64) error {
	if item.TeacherID != teacherID || item.Origin != OriginTeacherExam {
		return ErrItemNotFound
	}
	return nil
}
