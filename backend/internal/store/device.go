package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

var (
	ErrDeviceTaken      = errors.New("store: this phone is already registered to another account")
	ErrRequestPending   = errors.New("store: this account already has a pending phone request")
	ErrRequestDecided   = errors.New("store: this phone request is already decided")
	ErrNoteRequired     = errors.New("store: this approval needs a note from the teacher")
	ErrNotTheirStudent  = errors.New("store: this student is not taught by this teacher")
	ErrSameDeviceBound  = errors.New("store: this phone is already the registered phone")
	ErrAttendanceClosed = errors.New("store: the attendance session is closed")
	ErrAlreadyMarked    = errors.New("store: the student already has a record in this session")
	ErrDeviceUsed       = errors.New("store: this phone already marked a student in this session")
)

type DeviceBinding struct {
	UserID     int64
	DeviceID   string
	BoundAt    time.Time
	LastSeenAt time.Time
}

type DeviceChangeRequest struct {
	ID           int64
	UserID       int64
	StudentName  string
	RegNo        string
	Batch        string
	Section      string
	NewDeviceID  string
	Reason       string
	Status       string
	Semester     string
	DecidedBy    int64
	DeciderName  string
	DecisionNote string
	DecidedAt    time.Time
	CreatedAt    time.Time
}

const (
	bindDeviceSQL = `INSERT INTO device_bindings (user_id, device_id)
VALUES ($1, $2)
ON CONFLICT (user_id) DO NOTHING
RETURNING user_id, device_id, bound_at, last_seen_at`
	readDeviceSQL = `SELECT user_id, device_id, bound_at, last_seen_at
FROM device_bindings WHERE user_id = $1`
	touchDeviceSQL = `UPDATE device_bindings SET last_seen_at = now()
WHERE user_id = $1 AND device_id = $2`
	deviceOwnerSQL = `SELECT user_id FROM device_bindings WHERE device_id = $1 AND user_id <> $2`
)

func (s *Store) BindDevice(ctx context.Context, userID int64, deviceID string) (DeviceBinding, bool, error) {
	var binding DeviceBinding
	err := s.pool.QueryRow(ctx, bindDeviceSQL, userID, deviceID).Scan(
		&binding.UserID, &binding.DeviceID, &binding.BoundAt, &binding.LastSeenAt)
	if err == nil {
		return binding, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return DeviceBinding{}, false, fmt.Errorf("store: could not register the phone of %d: %w", userID, err)
	}
	binding, err = s.DeviceBinding(ctx, userID)
	return binding, false, err
}

func (s *Store) DeviceBinding(ctx context.Context, userID int64) (DeviceBinding, error) {
	var binding DeviceBinding
	err := s.pool.QueryRow(ctx, readDeviceSQL, userID).Scan(
		&binding.UserID, &binding.DeviceID, &binding.BoundAt, &binding.LastSeenAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return DeviceBinding{}, ErrNotFound
	}
	if err != nil {
		return DeviceBinding{}, fmt.Errorf("store: could not read the phone of %d: %w", userID, err)
	}
	return binding, nil
}

func (s *Store) TouchDevice(ctx context.Context, userID int64, deviceID string) error {
	if _, err := s.pool.Exec(ctx, touchDeviceSQL, userID, deviceID); err != nil {
		return fmt.Errorf("store: could not touch the phone of %d: %w", userID, err)
	}
	return nil
}

const (
	createRequestSQL = `INSERT INTO device_change_requests (user_id, new_device_id, reason, semester)
VALUES ($1, $2, $3, $4)
RETURNING id, created_at`
	countApprovedSQL = `SELECT COUNT(*) FROM device_change_requests
WHERE user_id = $1 AND status = 'approved' AND semester = $2`
	readRequestSQL = `SELECT r.id, r.user_id, COALESCE(s.full_name, ''), COALESCE(s.reg_no, ''),
COALESCE(s.batch, ''), COALESCE(s.section, ''), r.new_device_id, r.reason, r.status, r.semester,
COALESCE(r.decided_by, 0), COALESCE(t.name, d.identifier, ''), r.decision_note, r.decided_at, r.created_at
FROM device_change_requests r
LEFT JOIN students s ON s.user_id = r.user_id
LEFT JOIN users d ON d.id = r.decided_by
LEFT JOIN teachers t ON t.user_id = r.decided_by
WHERE r.id = $1`
	lockRequestSQL = `SELECT user_id, new_device_id, status, semester FROM device_change_requests
WHERE id = $1 FOR UPDATE`
	decideRequestSQL = `UPDATE device_change_requests
SET status = $2, decided_by = $3, decision_note = $4, decided_at = now()
WHERE id = $1 AND status = 'pending'
RETURNING id`
	rebindDeviceSQL = `INSERT INTO device_bindings (user_id, device_id)
VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE
SET device_id = EXCLUDED.device_id, bound_at = now(), last_seen_at = now()`
	teachesStudentSQL = `SELECT EXISTS (
    SELECT 1 FROM routines r
    JOIN batch_semesters bs ON bs.batch = r.batch AND bs.running_semester = r.semester
    JOIN teachers t ON t.id = r.teacher_id
    JOIN students s ON s.batch = r.batch AND (r.section = 'ALL' OR s.section = r.section)
    WHERE t.user_id = $1 AND s.user_id = $2
)`
	batchTeachersSQL = `SELECT DISTINCT t.user_id
FROM routines r
JOIN batch_semesters bs ON bs.batch = r.batch AND bs.running_semester = r.semester
JOIN teachers t ON t.id = r.teacher_id
JOIN users u ON u.id = t.user_id
WHERE r.batch = $1 AND (r.section = 'ALL' OR $2 = 'ALL' OR r.section = $2) AND u.status = 'active'`
	pendingForTeacherSQL = `SELECT r.id, r.user_id, s.full_name, s.reg_no, s.batch, s.section,
r.new_device_id, r.reason, r.status, r.semester, 0, '', r.decision_note, r.decided_at, r.created_at
FROM device_change_requests r
JOIN students s ON s.user_id = r.user_id
JOIN users su ON su.id = r.user_id
WHERE r.status = 'pending' AND su.status = 'active' AND EXISTS (
    SELECT 1 FROM routines ro
    JOIN batch_semesters bs ON bs.batch = ro.batch AND bs.running_semester = ro.semester
    JOIN teachers t ON t.id = ro.teacher_id
    WHERE t.user_id = $1 AND ro.batch = s.batch AND (ro.section = 'ALL' OR ro.section = s.section)
)
ORDER BY r.created_at DESC`
	requestHistorySQL = `SELECT r.id, r.user_id, COALESCE(s.full_name, ''), COALESCE(s.reg_no, ''),
COALESCE(s.batch, ''), COALESCE(s.section, ''), r.new_device_id, r.reason, r.status, r.semester,
COALESCE(r.decided_by, 0), COALESCE(t.name, d.identifier, ''), r.decision_note, r.decided_at, r.created_at
FROM device_change_requests r
LEFT JOIN students s ON s.user_id = r.user_id
LEFT JOIN users d ON d.id = r.decided_by
LEFT JOIN teachers t ON t.user_id = r.decided_by
ORDER BY r.created_at DESC
LIMIT $1`
	myRequestSQL = `SELECT r.id, r.user_id, '', '', '', '', r.new_device_id, r.reason, r.status, r.semester,
COALESCE(r.decided_by, 0), COALESCE(t.name, ''), r.decision_note, r.decided_at, r.created_at
FROM device_change_requests r
LEFT JOIN teachers t ON t.user_id = r.decided_by
WHERE r.user_id = $1
ORDER BY r.created_at DESC
LIMIT 1`
)

func (s *Store) CreateDeviceChangeRequest(ctx context.Context, userID int64,
	deviceID, reason, semester string) (DeviceChangeRequest, error) {
	var other int64
	err := s.pool.QueryRow(ctx, deviceOwnerSQL, deviceID, userID).Scan(&other)
	if err == nil {
		return DeviceChangeRequest{}, ErrDeviceTaken
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return DeviceChangeRequest{}, fmt.Errorf("store: could not check the owner of a phone: %w", err)
	}
	bound, err := s.DeviceBinding(ctx, userID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return DeviceChangeRequest{}, err
	}
	if err == nil && bound.DeviceID == deviceID {
		return DeviceChangeRequest{}, ErrSameDeviceBound
	}
	request := DeviceChangeRequest{
		UserID:      userID,
		NewDeviceID: deviceID,
		Reason:      reason,
		Status:      RequestPending,
		Semester:    semester,
	}
	err = s.pool.QueryRow(ctx, createRequestSQL, userID, deviceID, reason, semester).
		Scan(&request.ID, &request.CreatedAt)
	if isUniqueViolation(err) {
		return DeviceChangeRequest{}, ErrRequestPending
	}
	if err != nil {
		return DeviceChangeRequest{}, fmt.Errorf("store: could not save the phone request of %d: %w", userID, err)
	}
	return request, nil
}

func (s *Store) ApprovedDeviceChanges(ctx context.Context, userID int64, semester string) (int, error) {
	var count int
	if err := s.pool.QueryRow(ctx, countApprovedSQL, userID, semester).Scan(&count); err != nil {
		return 0, fmt.Errorf("store: could not count the phone changes of %d: %w", userID, err)
	}
	return count, nil
}

func (s *Store) TeachesStudent(ctx context.Context, teacherUserID, studentUserID int64) (bool, error) {
	var teaches bool
	err := s.pool.QueryRow(ctx, teachesStudentSQL, teacherUserID, studentUserID).Scan(&teaches)
	if err != nil {
		return false, fmt.Errorf("store: could not check whether %d teaches %d: %w",
			teacherUserID, studentUserID, err)
	}
	return teaches, nil
}

func (s *Store) ListBatchTeachers(ctx context.Context, batch, section string) ([]int64, error) {
	rows, err := s.pool.Query(ctx, batchTeachersSQL, batch, section)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the teachers of %s: %w", batch, err)
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("store: could not scan a batch teacher: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the batch teachers: %w", err)
	}
	return ids, nil
}

func (s *Store) DeviceRequest(ctx context.Context, requestID int64) (DeviceChangeRequest, error) {
	request, err := scanDeviceRequest(s.pool.QueryRow(ctx, readRequestSQL, requestID))
	if errors.Is(err, pgx.ErrNoRows) {
		return DeviceChangeRequest{}, ErrNotFound
	}
	if err != nil {
		return DeviceChangeRequest{}, fmt.Errorf("store: could not read the phone request %d: %w", requestID, err)
	}
	return request, nil
}

func (s *Store) LatestDeviceRequest(ctx context.Context, userID int64) (DeviceChangeRequest, error) {
	request, err := scanDeviceRequest(s.pool.QueryRow(ctx, myRequestSQL, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return DeviceChangeRequest{}, ErrNotFound
	}
	if err != nil {
		return DeviceChangeRequest{}, fmt.Errorf("store: could not read the phone request of %d: %w", userID, err)
	}
	return request, nil
}

func (s *Store) ListTeacherDeviceRequests(ctx context.Context, teacherUserID int64) ([]DeviceChangeRequest, error) {
	return s.deviceRequests(ctx, pendingForTeacherSQL, teacherUserID)
}

func (s *Store) ListDeviceRequestHistory(ctx context.Context, limit int) ([]DeviceChangeRequest, error) {
	return s.deviceRequests(ctx, requestHistorySQL, limit)
}

func (s *Store) deviceRequests(ctx context.Context, query string, args ...any) ([]DeviceChangeRequest, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the phone requests: %w", err)
	}
	defer rows.Close()
	requests := make([]DeviceChangeRequest, 0)
	for rows.Next() {
		request, err := scanDeviceRequest(rows)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a phone request: %w", err)
		}
		requests = append(requests, request)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the phone requests: %w", err)
	}
	return requests, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDeviceRequest(row rowScanner) (DeviceChangeRequest, error) {
	var request DeviceChangeRequest
	var decidedAt *time.Time
	err := row.Scan(
		&request.ID,
		&request.UserID,
		&request.StudentName,
		&request.RegNo,
		&request.Batch,
		&request.Section,
		&request.NewDeviceID,
		&request.Reason,
		&request.Status,
		&request.Semester,
		&request.DecidedBy,
		&request.DeciderName,
		&request.DecisionNote,
		&decidedAt,
		&request.CreatedAt,
	)
	if err != nil {
		return DeviceChangeRequest{}, err
	}
	if decidedAt != nil {
		request.DecidedAt = *decidedAt
	}
	return request, nil
}

func (s *Store) DecideDeviceChange(ctx context.Context, requestID, deciderUserID int64,
	approve bool, note string, changeLimit int) (DeviceChangeRequest, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return DeviceChangeRequest{}, fmt.Errorf("store: could not start the phone decision: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var studentUserID int64
	var deviceID, status, semester string
	err = tx.QueryRow(ctx, lockRequestSQL, requestID).Scan(&studentUserID, &deviceID, &status, &semester)
	if errors.Is(err, pgx.ErrNoRows) {
		return DeviceChangeRequest{}, ErrNotFound
	}
	if err != nil {
		return DeviceChangeRequest{}, fmt.Errorf("store: could not lock the phone request %d: %w", requestID, err)
	}
	if status != RequestPending {
		return DeviceChangeRequest{}, ErrRequestDecided
	}
	teaches, err := s.TeachesStudent(ctx, deciderUserID, studentUserID)
	if err != nil {
		return DeviceChangeRequest{}, err
	}
	if !teaches {
		return DeviceChangeRequest{}, ErrNotTheirStudent
	}
	decision := RequestRejected
	if approve {
		decision = RequestApproved
		var approved int
		if err := tx.QueryRow(ctx, countApprovedSQL, studentUserID, semester).Scan(&approved); err != nil {
			return DeviceChangeRequest{}, fmt.Errorf("store: could not count the phone changes of %d: %w",
				studentUserID, err)
		}
		if approved >= changeLimit && note == "" {
			return DeviceChangeRequest{}, ErrNoteRequired
		}
	}
	var decided int64
	err = tx.QueryRow(ctx, decideRequestSQL, requestID, decision, deciderUserID, note).Scan(&decided)
	if errors.Is(err, pgx.ErrNoRows) {
		return DeviceChangeRequest{}, ErrRequestDecided
	}
	if err != nil {
		return DeviceChangeRequest{}, fmt.Errorf("store: could not decide the phone request %d: %w", requestID, err)
	}
	if approve {
		if _, err := tx.Exec(ctx, rebindDeviceSQL, studentUserID, deviceID); err != nil {
			return DeviceChangeRequest{}, fmt.Errorf("store: could not rebind the phone of %d: %w",
				studentUserID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return DeviceChangeRequest{}, fmt.Errorf("store: could not finish the phone decision: %w", err)
	}
	return s.DeviceRequest(ctx, requestID)
}
