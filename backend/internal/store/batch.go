package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

const (
	BatchActive   = "active"
	BatchInactive = "inactive"
)

var (
	ErrDuplicateBatch = errors.New("store: a batch with this name already exists")
	ErrBatchUnchanged = errors.New("store: the batch is already in that state")
)

type Batch struct {
	ID              int64
	Name            string
	RunningSemester string
	Sections        []string
	Status          string
	Students        int
	CreatedBy       string
	UpdatedBy       string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type BatchRemoval struct {
	Students   int
	Routines   int
	Items      int
	Attendance int
}

const (
	batchColumns = `b.id, b.name, b.running_semester, b.sections, b.status,
(SELECT COUNT(*) FROM students s JOIN users u ON u.id = s.user_id
 WHERE s.batch = b.name AND u.status = 'active'),
COALESCE(cu.identifier, ''), COALESCE(uu.identifier, ''), b.created_at, b.updated_at
FROM batches b
LEFT JOIN users cu ON cu.id = b.created_by
LEFT JOIN users uu ON uu.id = b.updated_by`
	listBatchRowsSQL = `SELECT ` + batchColumns + ` ORDER BY b.name`
	findBatchSQL     = `SELECT ` + batchColumns + ` WHERE b.name = $1`
	insertBatchSQL   = `INSERT INTO batches (name, running_semester, created_by, updated_by)
VALUES ($1, $2, NULLIF($3::bigint, 0), NULLIF($3::bigint, 0))`
	ensureBatchSQL = `INSERT INTO batches (name, running_semester)
VALUES ($1, $2) ON CONFLICT (name) DO NOTHING`
	setBatchSemesterSQL = `UPDATE batches
SET semester_since = CASE WHEN running_semester = $2 THEN semester_since ELSE now() END,
    running_semester = $2, updated_by = NULLIF($3::bigint, 0), updated_at = now()
WHERE name = $1 RETURNING id`
	setBatchStatusSQL = `UPDATE batches
SET status = $2, updated_by = NULLIF($3::bigint, 0), updated_at = now()
WHERE name = $1 AND status <> $2 RETURNING id`
	batchOpenSQL       = `SELECT EXISTS (SELECT 1 FROM batches WHERE name = $1 AND status = 'active')`
	lockBatchSQL       = `SELECT id FROM batches WHERE name = $1 FOR UPDATE`
	batchStudentsSQL   = `SELECT user_id FROM students WHERE batch = $1`
	signOutStudentsSQL = `UPDATE users SET token_version = token_version + 1, updated_at = now()
WHERE id = ANY($1::bigint[])`
	revokeStudentSessionsSQL = `UPDATE user_sessions SET revoked_at = now()
WHERE revoked_at IS NULL AND user_id = ANY($1::bigint[])`
	dropStudentFCMSQL    = `DELETE FROM fcm_tokens WHERE user_id = ANY($1::bigint[])`
	dropStudentPushSQL   = `DELETE FROM push_subscriptions WHERE user_id = ANY($1::bigint[])`
	removeBatchPeopleSQL = `UPDATE users
SET status = 'removed', token_version = token_version + 1, updated_at = now()
WHERE id = ANY($1::bigint[]) AND role = 'student' AND status <> 'removed'`
	deleteBatchAttendanceSQL  = `DELETE FROM attendance_sessions WHERE batch = $1`
	deleteBatchPublishedSQL   = `DELETE FROM published_items WHERE batch = $1`
	deleteBatchPublicationSQL = `DELETE FROM publications WHERE batch = $1`
	deleteBatchItemsSQL       = `DELETE FROM schedule_items WHERE batch = $1`
	deleteBatchRoutinesSQL    = `DELETE FROM routines WHERE batch = $1`
	deleteBatchDriveLinkSQL   = `DELETE FROM batch_drive_links WHERE batch = $1`
	deleteBatchSQL            = `DELETE FROM batches WHERE name = $1`
	batchWatchersSQL          = `SELECT u.id FROM users u
WHERE u.id <> $1 AND u.status = 'active' AND (u.role = 'admin' OR u.office_access)
  AND EXISTS (SELECT 1 FROM admin_permissions p WHERE p.user_id = u.id AND p.permission = 'batches')
ORDER BY u.id`
)

func scanBatch(row rowScanner) (Batch, error) {
	var batch Batch
	err := row.Scan(&batch.ID, &batch.Name, &batch.RunningSemester, &batch.Sections, &batch.Status,
		&batch.Students, &batch.CreatedBy, &batch.UpdatedBy, &batch.CreatedAt, &batch.UpdatedAt)
	return batch, err
}

func (s *Store) ListBatchRows(ctx context.Context) ([]Batch, error) {
	rows, err := s.pool.Query(ctx, listBatchRowsSQL)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the batches: %w", err)
	}
	defer rows.Close()
	batches := make([]Batch, 0)
	for rows.Next() {
		batch, err := scanBatch(rows)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a batch: %w", err)
		}
		batches = append(batches, batch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the batches: %w", err)
	}
	return batches, nil
}

func (s *Store) FindBatch(ctx context.Context, name string) (Batch, error) {
	batch, err := scanBatch(s.pool.QueryRow(ctx, findBatchSQL, name))
	if errors.Is(err, pgx.ErrNoRows) {
		return Batch{}, ErrNotFound
	}
	if err != nil {
		return Batch{}, fmt.Errorf("store: could not load the batch %s: %w", name, err)
	}
	return batch, nil
}

func (s *Store) CreateBatch(ctx context.Context, name, semester string, by int64) (Batch, error) {
	if _, err := s.pool.Exec(ctx, insertBatchSQL, name, semester, by); err != nil {
		if isUniqueViolation(err) {
			return Batch{}, ErrDuplicateBatch
		}
		return Batch{}, fmt.Errorf("store: could not create the batch %s: %w", name, err)
	}
	return s.FindBatch(ctx, name)
}

func (s *Store) SetBatchSemester(ctx context.Context, name, semester string, by int64) error {
	var id int64
	err := s.pool.QueryRow(ctx, setBatchSemesterSQL, name, semester, by).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not set the running semester of %s: %w", name, err)
	}
	return nil
}

func (s *Store) BatchOpen(ctx context.Context, name string) (bool, error) {
	var open bool
	if err := s.pool.QueryRow(ctx, batchOpenSQL, name).Scan(&open); err != nil {
		return false, fmt.Errorf("store: could not check the batch %s: %w", name, err)
	}
	return open, nil
}

func (s *Store) SetBatchStatus(ctx context.Context, name, status string, by int64) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("store: could not start the batch transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id int64
	err = tx.QueryRow(ctx, lockBatchSQL, name).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("store: could not lock the batch %s: %w", name, err)
	}
	err = tx.QueryRow(ctx, setBatchStatusSQL, name, status, by).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrBatchUnchanged
	}
	if err != nil {
		return 0, fmt.Errorf("store: could not change the status of %s: %w", name, err)
	}
	students, err := batchStudents(ctx, tx, name)
	if err != nil {
		return 0, err
	}
	if status == BatchInactive && len(students) > 0 {
		if err := signOutStudents(ctx, tx, students, signOutStudentsSQL); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("store: could not commit the status of %s: %w", name, err)
	}
	return len(students), nil
}

func batchStudents(ctx context.Context, tx pgx.Tx, name string) ([]int64, error) {
	rows, err := tx.Query(ctx, batchStudentsSQL, name)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the students of %s: %w", name, err)
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("store: could not scan a student of %s: %w", name, err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the students of %s: %w", name, err)
	}
	return ids, nil
}

func signOutStudents(ctx context.Context, tx pgx.Tx, students []int64, first string) error {
	statements := []string{first, revokeStudentSessionsSQL, dropStudentFCMSQL, dropStudentPushSQL}
	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement, students); err != nil {
			return fmt.Errorf("store: could not sign the students of a batch out: %w", err)
		}
	}
	return nil
}

func (s *Store) DeleteBatch(ctx context.Context, name string) (BatchRemoval, error) {
	var removal BatchRemoval
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return removal, fmt.Errorf("store: could not start the batch deletion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id int64
	err = tx.QueryRow(ctx, lockBatchSQL, name).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return removal, ErrNotFound
	}
	if err != nil {
		return removal, fmt.Errorf("store: could not lock the batch %s: %w", name, err)
	}
	students, err := batchStudents(ctx, tx, name)
	if err != nil {
		return removal, err
	}
	removal.Students = len(students)
	if len(students) > 0 {
		if err := signOutStudents(ctx, tx, students, removeBatchPeopleSQL); err != nil {
			return removal, err
		}
	}
	steps := []struct {
		query string
		count *int
	}{
		{query: deleteBatchAttendanceSQL, count: &removal.Attendance},
		{query: deleteBatchPublishedSQL},
		{query: deleteBatchPublicationSQL},
		{query: deleteBatchItemsSQL, count: &removal.Items},
		{query: deleteBatchRoutinesSQL, count: &removal.Routines},
		{query: deleteBatchDriveLinkSQL},
		{query: deleteBatchSQL},
	}
	for _, step := range steps {
		tag, err := tx.Exec(ctx, step.query, name)
		if err != nil {
			return removal, fmt.Errorf("store: could not delete the batch %s: %w", name, err)
		}
		if step.count != nil {
			*step.count = int(tag.RowsAffected())
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return removal, fmt.Errorf("store: could not commit the deletion of %s: %w", name, err)
	}
	return removal, nil
}

func (s *Store) BatchWatchers(ctx context.Context, exceptUserID int64) ([]int64, error) {
	rows, err := s.pool.Query(ctx, batchWatchersSQL, exceptUserID)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the admins who manage batches: %w", err)
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("store: could not scan an admin who manages batches: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the admins who manage batches: %w", err)
	}
	return ids, nil
}
