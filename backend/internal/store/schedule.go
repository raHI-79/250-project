package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

const (
	ItemAwaiting    = "awaiting_teacher"
	ItemNoResponse  = "no_response"
	ItemConfirmed   = "confirmed"
	ItemCancelled   = "cancelled"
	ItemTeacherAsk  = "teacher_reschedule_request"
	ItemCRAsk       = "cr_reschedule_request"
	ItemDeclined    = "reschedule_declined"
	ItemRescheduled = "rescheduled"
	ItemScheduled   = "scheduled"

	OriginFixed       = "fixed"
	OriginCR          = "cr"
	OriginTeacherExam = "teacher_exam"

	ModeRoom   = "room"
	ModeOnline = "online"

	KindExam = "exam"

	ConflictRoom  = "room"
	ConflictBatch = "batch"
)

var (
	ErrItemNotFound     = errors.New("store: no schedule item with this id for this account")
	ErrItemState        = errors.New("store: this schedule item is not in a state that allows this change")
	ErrNoProposal       = errors.New("store: this schedule item carries no proposed slot")
	ErrItemStarted      = errors.New("store: this class has already started")
	ErrNothingToPublish = errors.New("store: no schedule item was selected for publishing")
)

type SlotConflict struct {
	Reason     string
	Batch      string
	Section    string
	CourseCode string
	Room       string
	StartTime  string
	EndTime    string
}

func (e *SlotConflict) Error() string {
	if e.Reason == ConflictRoom {
		return fmt.Sprintf("store: room %s is taken by %s section %s from %s to %s",
			e.Room, e.Batch, e.Section, e.StartTime, e.EndTime)
	}
	return fmt.Sprintf("store: %s section %s already has %s from %s to %s",
		e.Batch, e.Section, e.CourseCode, e.StartTime, e.EndTime)
}

type ItemSnapshot struct {
	Exists      bool
	Status      string
	ItemDate    string
	StartTime   string
	EndTime     string
	Mode        string
	Room        string
	MeetingLink string
}

type ScheduleItem struct {
	ID             int64
	Batch          string
	Section        string
	CourseID       int64
	CourseCode     string
	CourseTitle    string
	TeacherID      int64
	TeacherUserID  int64
	TeacherName    string
	Kind           string
	Origin         string
	RoutineID      int64
	ItemDate       string
	StartTime      string
	EndTime        string
	Mode           string
	Room           string
	MeetingLink    string
	Status         string
	TeacherMessage string
	CancelReason   string
	Syllabus       string
	Instructions   string
	Note           string
	ProposedDate   string
	ProposedStart  string
	ProposedEnd    string
	ProposedMode   string
	ProposedRoom   string
	ProposedLink   string
	InFinal        bool
	PriorStatus    string
	CreatedBy      int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Snapshot       ItemSnapshot
}

func (i ScheduleItem) CourseName() string {
	if i.CourseTitle != "" {
		return i.CourseTitle
	}
	return i.CourseCode
}

func (i ScheduleItem) HasTimes() bool {
	return i.StartTime != "" && i.EndTime != ""
}

func (i ScheduleItem) Complete() bool {
	if !i.HasTimes() {
		return false
	}
	if i.Mode == ModeOnline {
		return i.MeetingLink != ""
	}
	return i.Room != ""
}

func (i ScheduleItem) Published() bool {
	return i.Snapshot.Exists
}

func (i ScheduleItem) ChangedSincePublish() bool {
	if !i.Snapshot.Exists {
		return false
	}
	return i.Snapshot.Status != i.Status ||
		i.Snapshot.ItemDate != i.ItemDate ||
		i.Snapshot.StartTime != i.StartTime ||
		i.Snapshot.EndTime != i.EndTime ||
		i.Snapshot.Mode != i.Mode ||
		i.Snapshot.Room != i.Room ||
		i.Snapshot.MeetingLink != i.MeetingLink
}

type NewScheduleItem struct {
	Batch        string
	Section      string
	CourseID     int64
	TeacherID    int64
	Kind         string
	Origin       string
	ItemDate     string
	StartTime    string
	EndTime      string
	Mode         string
	Room         string
	MeetingLink  string
	Status       string
	Syllabus     string
	Instructions string
	Note         string
	CreatedBy    int64
}

type ItemPatch struct {
	ItemDate     *string
	StartTime    *string
	EndTime      *string
	Mode         *string
	Room         *string
	MeetingLink  *string
	Note         *string
	Syllabus     *string
	Instructions *string
}

type Proposal struct {
	ItemDate    string
	StartTime   string
	EndTime     string
	Mode        string
	Room        string
	MeetingLink string
}

type scanner interface {
	Scan(dest ...any) error
}

const itemColumns = `i.id, i.batch, i.section, i.course_id, c.code, c.title,
COALESCE(i.teacher_id, 0), COALESCE(t.user_id, 0), COALESCE(t.name, 'Deleted user'),
i.kind, i.origin, COALESCE(i.routine_id, 0), to_char(i.item_date, 'YYYY-MM-DD'),
COALESCE(to_char(i.start_time, 'HH24:MI'), ''), COALESCE(to_char(i.end_time, 'HH24:MI'), ''),
i.mode, i.room, i.meeting_link, i.status, i.teacher_message, i.cancel_reason,
i.syllabus, i.instructions, i.note,
COALESCE(to_char(i.proposed_date, 'YYYY-MM-DD'), ''),
COALESCE(to_char(i.proposed_start, 'HH24:MI'), ''), COALESCE(to_char(i.proposed_end, 'HH24:MI'), ''),
COALESCE(i.proposed_mode, ''), i.proposed_room, i.proposed_link,
i.in_final, i.prior_status, COALESCE(i.created_by, 0), i.created_at, i.updated_at,
(pi.item_id IS NOT NULL), COALESCE(pi.status, ''),
COALESCE(to_char(pi.item_date, 'YYYY-MM-DD'), ''),
COALESCE(to_char(pi.start_time, 'HH24:MI'), ''), COALESCE(to_char(pi.end_time, 'HH24:MI'), ''),
COALESCE(pi.mode, ''), COALESCE(pi.room, ''), COALESCE(pi.meeting_link, '')`

const itemJoins = `FROM schedule_items i
JOIN courses c ON c.id = i.course_id
LEFT JOIN teachers t ON t.id = i.teacher_id
LEFT JOIN published_items pi ON pi.item_id = i.id`

const selectItem = `SELECT ` + itemColumns + `
` + itemJoins

const (
	generateFixedItemsSQL = `INSERT INTO schedule_items
(batch, section, course_id, teacher_id, kind, origin, routine_id, item_date,
start_time, end_time, mode, room, status)
SELECT r.batch, r.section, r.course_id, r.teacher_id, r.type, 'fixed', r.id, day::date,
r.start_time, r.end_time, 'room', r.room, 'awaiting_teacher'
FROM routines r
JOIN batch_semesters bs ON bs.batch = r.batch AND bs.running_semester = r.semester
CROSS JOIN generate_series($1::date, $2::date, interval '1 day') AS day
WHERE EXTRACT(DOW FROM day)::int = r.day_of_week
ON CONFLICT DO NOTHING`
	markNoResponseSQL = `WITH updated AS (
    UPDATE schedule_items
    SET status = 'no_response', updated_at = now()
    WHERE item_date = $1::date AND status = 'awaiting_teacher'
    RETURNING id
)
` + selectItem + `
WHERE i.id IN (SELECT id FROM updated)
ORDER BY i.batch, i.section, i.start_time, i.id`
	promoteExamsSQL = `WITH updated AS (
    UPDATE schedule_items
    SET in_final = TRUE, updated_at = now()
    WHERE item_date = $1::date AND kind = 'exam' AND NOT in_final AND status <> 'cancelled'
    RETURNING id
)
` + selectItem + `
WHERE i.id IN (SELECT id FROM updated)
ORDER BY i.batch, i.section, i.start_time, i.id`
	findItemSQL = selectItem + `
WHERE i.id = $1`
	lockBatchDateSQL    = `SELECT pg_advisory_xact_lock(hashtext('batch:' || $1::text || ':' || $2::text))`
	lockRoomDateSQL     = `SELECT pg_advisory_xact_lock(hashtext('room:' || $1::text || ':' || $2::text))`
	roomIsManagedSQL    = `SELECT EXISTS (SELECT 1 FROM routines WHERE room = $1)`
	findRoomConflictSQL = `SELECT i.batch, i.section, c.code,
to_char(i.start_time, 'HH24:MI'), to_char(i.end_time, 'HH24:MI'), i.room
FROM schedule_items i
JOIN courses c ON c.id = i.course_id
WHERE i.id <> $1 AND i.mode = 'room' AND i.room = $2 AND i.item_date = $3::date
  AND i.status <> 'cancelled' AND i.start_time IS NOT NULL
  AND $4::time < i.end_time AND i.start_time < $5::time
ORDER BY i.start_time
LIMIT 1`
	findBatchConflictSQL = `SELECT i.batch, i.section, c.code,
to_char(i.start_time, 'HH24:MI'), to_char(i.end_time, 'HH24:MI'), i.room
FROM schedule_items i
JOIN courses c ON c.id = i.course_id
WHERE i.id <> $1 AND i.batch = $2 AND i.item_date = $3::date
  AND i.status <> 'cancelled' AND i.start_time IS NOT NULL
  AND (i.section = $4::text OR i.section = 'ALL' OR $4::text = 'ALL')
  AND $5::time < i.end_time AND i.start_time < $6::time
ORDER BY i.start_time
LIMIT 1`
	insertItemSQL = `INSERT INTO schedule_items
(batch, section, course_id, teacher_id, kind, origin, item_date, start_time, end_time,
mode, room, meeting_link, status, syllabus, instructions, note, created_by)
VALUES ($1, $2, $3, NULLIF($4::bigint, 0), $5, $6, $7::date,
NULLIF($8::text, '')::time, NULLIF($9::text, '')::time, $10, $11, $12, $13, $14, $15, $16,
NULLIF($17::bigint, 0))
RETURNING id`
	courseTeacherSQL = `SELECT r.teacher_id
FROM routines r
WHERE r.batch = $1 AND r.course_id = $2 AND r.teacher_id IS NOT NULL
  AND (r.section = $3::text OR r.section = 'ALL' OR $3::text = 'ALL')
ORDER BY r.id
LIMIT 1`
)

type slotRequest struct {
	ExcludeID int64
	Batch     string
	Section   string
	ItemDate  string
	StartTime string
	EndTime   string
	Mode      string
	Room      string
}

func scanItem(row scanner) (ScheduleItem, error) {
	var item ScheduleItem
	err := row.Scan(
		&item.ID,
		&item.Batch,
		&item.Section,
		&item.CourseID,
		&item.CourseCode,
		&item.CourseTitle,
		&item.TeacherID,
		&item.TeacherUserID,
		&item.TeacherName,
		&item.Kind,
		&item.Origin,
		&item.RoutineID,
		&item.ItemDate,
		&item.StartTime,
		&item.EndTime,
		&item.Mode,
		&item.Room,
		&item.MeetingLink,
		&item.Status,
		&item.TeacherMessage,
		&item.CancelReason,
		&item.Syllabus,
		&item.Instructions,
		&item.Note,
		&item.ProposedDate,
		&item.ProposedStart,
		&item.ProposedEnd,
		&item.ProposedMode,
		&item.ProposedRoom,
		&item.ProposedLink,
		&item.InFinal,
		&item.PriorStatus,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.Snapshot.Exists,
		&item.Snapshot.Status,
		&item.Snapshot.ItemDate,
		&item.Snapshot.StartTime,
		&item.Snapshot.EndTime,
		&item.Snapshot.Mode,
		&item.Snapshot.Room,
		&item.Snapshot.MeetingLink,
	)
	return item, err
}

func (s *Store) items(ctx context.Context, query string, args ...any) ([]ScheduleItem, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the schedule items: %w", err)
	}
	defer rows.Close()
	items := make([]ScheduleItem, 0)
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a schedule item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the schedule items: %w", err)
	}
	return items, nil
}

func (s *Store) GetScheduleItem(ctx context.Context, itemID int64) (ScheduleItem, error) {
	item, err := scanItem(s.pool.QueryRow(ctx, findItemSQL, itemID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ScheduleItem{}, ErrItemNotFound
	}
	if err != nil {
		return ScheduleItem{}, fmt.Errorf("store: could not load the schedule item %d: %w", itemID, err)
	}
	return item, nil
}

func (s *Store) GenerateFixedItems(ctx context.Context, from, to string) (int, error) {
	tag, err := s.pool.Exec(ctx, generateFixedItemsSQL, from, to)
	if err != nil {
		return 0, fmt.Errorf("store: could not generate the fixed routine items: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

func (s *Store) MarkNoResponse(ctx context.Context, date string) ([]ScheduleItem, error) {
	return s.items(ctx, markNoResponseSQL, date)
}

func (s *Store) PromoteExamsToFinal(ctx context.Context, date string) ([]ScheduleItem, error) {
	return s.items(ctx, promoteExamsSQL, date)
}

func checkSlot(ctx context.Context, tx pgx.Tx, want slotRequest) error {
	if want.StartTime == "" || want.EndTime == "" {
		return nil
	}
	if _, err := tx.Exec(ctx, lockBatchDateSQL, want.Batch, want.ItemDate); err != nil {
		return fmt.Errorf("store: could not lock the batch and date: %w", err)
	}
	conflict, err := findConflict(ctx, tx, findBatchConflictSQL, ConflictBatch,
		want.ExcludeID, want.Batch, want.ItemDate, want.Section, want.StartTime, want.EndTime)
	if err != nil || conflict != nil {
		return conflictOr(conflict, err)
	}
	if want.Mode != ModeRoom || want.Room == "" {
		return nil
	}
	var managed bool
	if err := tx.QueryRow(ctx, roomIsManagedSQL, want.Room).Scan(&managed); err != nil {
		return fmt.Errorf("store: could not check whether room %s is managed: %w", want.Room, err)
	}
	if !managed {
		return nil
	}
	if _, err := tx.Exec(ctx, lockRoomDateSQL, want.Room, want.ItemDate); err != nil {
		return fmt.Errorf("store: could not lock the room and date: %w", err)
	}
	conflict, err = findConflict(ctx, tx, findRoomConflictSQL, ConflictRoom,
		want.ExcludeID, want.Room, want.ItemDate, want.StartTime, want.EndTime)
	return conflictOr(conflict, err)
}

func findConflict(ctx context.Context, tx pgx.Tx, query, reason string, args ...any) (*SlotConflict, error) {
	conflict := SlotConflict{Reason: reason}
	err := tx.QueryRow(ctx, query, args...).Scan(
		&conflict.Batch,
		&conflict.Section,
		&conflict.CourseCode,
		&conflict.StartTime,
		&conflict.EndTime,
		&conflict.Room,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: could not check the %s bookings: %w", reason, err)
	}
	return &conflict, nil
}

func conflictOr(conflict *SlotConflict, err error) error {
	if err != nil {
		return err
	}
	if conflict != nil {
		return conflict
	}
	return nil
}

func (s *Store) insertItem(ctx context.Context, item NewScheduleItem) (ScheduleItem, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ScheduleItem{}, fmt.Errorf("store: could not start the schedule item transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	err = checkSlot(ctx, tx, slotRequest{
		Batch:     item.Batch,
		Section:   item.Section,
		ItemDate:  item.ItemDate,
		StartTime: item.StartTime,
		EndTime:   item.EndTime,
		Mode:      item.Mode,
		Room:      item.Room,
	})
	if err != nil {
		return ScheduleItem{}, err
	}
	var itemID int64
	err = tx.QueryRow(ctx, insertItemSQL,
		item.Batch,
		item.Section,
		item.CourseID,
		item.TeacherID,
		item.Kind,
		item.Origin,
		item.ItemDate,
		item.StartTime,
		item.EndTime,
		item.Mode,
		item.Room,
		item.MeetingLink,
		item.Status,
		item.Syllabus,
		item.Instructions,
		item.Note,
		item.CreatedBy,
	).Scan(&itemID)
	if err != nil {
		if isForeignKeyViolation(err) {
			return ScheduleItem{}, ErrCourseNotFound
		}
		return ScheduleItem{}, fmt.Errorf("store: could not create the schedule item: %w", err)
	}
	created, err := scanItem(tx.QueryRow(ctx, findItemSQL, itemID))
	if err != nil {
		return ScheduleItem{}, fmt.Errorf("store: could not read back the schedule item: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ScheduleItem{}, fmt.Errorf("store: could not commit the schedule item: %w", err)
	}
	return created, nil
}

func (s *Store) TeacherOfCourse(ctx context.Context, batch, section string, courseID int64) (int64, error) {
	var teacherID int64
	err := s.pool.QueryRow(ctx, courseTeacherSQL, batch, courseID, section).Scan(&teacherID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("store: could not find the teacher of the course %d: %w", courseID, err)
	}
	return teacherID, nil
}
