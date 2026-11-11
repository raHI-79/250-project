package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

var (
	ErrApprovalGone = errors.New("store: the request is no longer waiting")
	ErrOwnApproval  = errors.New("store: a request cannot be approved by the person who asked")
)

const (
	ApprovalApproved = "approved"
	ApprovalDeclined = "declined"
	ApprovalExpired  = "expired"
	MaxFailedSignIns = 5
	LockWindow       = 15 * time.Minute
)

type IPBlock struct {
	ID        int64
	CIDR      string
	Reason    string
	CreatedBy string
	CreatedAt time.Time
	ExpiresAt *time.Time
}

type Approval struct {
	ID          int64
	Action      string
	TargetType  string
	TargetID    string
	Summary     string
	Payload     string
	RequestedBy int64
	Requester   string
	Status      string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	DecidedAt   *time.Time
	DecidedBy   string
}

type LockedAccount struct {
	ID          int64
	Identifier  string
	Role        string
	Failures    int
	LockedUntil *time.Time
}

type FailureCount struct {
	Label string
	Total int
}

const (
	allSettingsSQL = `SELECT key, value FROM settings`
	saveSettingSQL = `INSERT INTO settings (key, value, updated_by, updated_at)
VALUES ($1, $2, NULLIF($3, 0), now())
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value, updated_by = EXCLUDED.updated_by, updated_at = now()`
	listBlocksSQL = `SELECT b.id, b.cidr, b.reason, COALESCE(u.identifier, ''), b.created_at, b.expires_at
FROM ip_blocks b
LEFT JOIN users u ON u.id = b.created_by
WHERE b.expires_at IS NULL OR b.expires_at > now()
ORDER BY b.created_at DESC`
	activeBlocksSQL = `SELECT cidr FROM ip_blocks WHERE expires_at IS NULL OR expires_at > now()`
	insertBlockSQL  = `INSERT INTO ip_blocks (cidr, reason, created_by, expires_at)
VALUES ($1, $2, NULLIF($3, 0), now() + make_interval(secs => NULLIF($4::double precision, 0)))
ON CONFLICT (cidr) DO UPDATE
SET reason = EXCLUDED.reason, created_by = EXCLUDED.created_by,
    expires_at = EXCLUDED.expires_at, created_at = now()
RETURNING id`
	deleteBlockSQL    = `DELETE FROM ip_blocks WHERE id = $1 RETURNING cidr`
	insertApprovalSQL = `INSERT INTO approval_requests
(action, target_type, target_id, summary, payload, requested_by, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, now() + make_interval(secs => $7))
RETURNING id`
	approvalColumns = `a.id, a.action, a.target_type, a.target_id, a.summary, a.payload,
a.requested_by, COALESCE(asker.identifier, s.full_name, t.name, ''), a.status, a.created_at,
a.expires_at, a.decided_at, COALESCE(decider.identifier, '')`
	approvalFrom = `FROM approval_requests a
LEFT JOIN users asker ON asker.id = a.requested_by
LEFT JOIN students s ON s.user_id = asker.id
LEFT JOIN teachers t ON t.user_id = asker.id
LEFT JOIN users decider ON decider.id = a.decided_by`
	listApprovalsSQL = `SELECT ` + approvalColumns + `
` + approvalFrom + `
ORDER BY a.created_at DESC
LIMIT 50`
	loadApprovalSQL = `SELECT ` + approvalColumns + `
` + approvalFrom + `
WHERE a.id = $1`
	decideApprovalSQL = `UPDATE approval_requests
SET status = $2, decided_by = $3, decided_at = now()
WHERE id = $1 AND status = 'pending' AND expires_at > now()
RETURNING id`
	expireApprovalsSQL = `UPDATE approval_requests
SET status = 'expired', decided_at = now()
WHERE status = 'pending' AND expires_at <= now()`
	otherOfficeAdminsSQL = `SELECT COUNT(*) FROM users
WHERE id <> $1 AND status = 'active' AND (role = 'admin' OR office_access)`
	registerFailureSQL = `UPDATE users
SET failed_sign_ins = CASE WHEN locked_until IS NOT NULL AND locked_until < now()
        THEN 1 ELSE failed_sign_ins + 1 END,
    locked_until = CASE WHEN (CASE WHEN locked_until IS NOT NULL AND locked_until < now()
        THEN 1 ELSE failed_sign_ins + 1 END) >= $2 THEN now() + $3::interval ELSE locked_until END,
    updated_at = now()
WHERE id = $1
RETURNING failed_sign_ins, locked_until`
	clearFailuresSQL = `UPDATE users
SET failed_sign_ins = 0, locked_until = NULL, updated_at = now()
WHERE id = $1 AND (failed_sign_ins > 0 OR locked_until IS NOT NULL)`
	lockedAccountsSQL = `SELECT id, COALESCE(identifier, ''), role, failed_sign_ins, locked_until
FROM users
WHERE locked_until IS NOT NULL AND locked_until > now()
ORDER BY locked_until DESC
LIMIT 50`
	failuresByIPSQL = `SELECT ip_hash, COUNT(*)
FROM audit_events
WHERE action = 'sign_in_failed' AND ip_hash <> '' AND created_at >= $1
GROUP BY ip_hash
ORDER BY COUNT(*) DESC
LIMIT 10`
	failuresByAccountSQL = `SELECT COALESCE(NULLIF(actor_label, ''), 'unknown'), COUNT(*)
FROM audit_events
WHERE action = 'sign_in_failed' AND created_at >= $1
GROUP BY 1
ORDER BY COUNT(*) DESC
LIMIT 10`
)

func (s *Store) AllSettings(ctx context.Context) (map[string]string, error) {
	rows, err := s.pool.Query(ctx, allSettingsSQL)
	if err != nil {
		return nil, fmt.Errorf("store: could not read the settings: %w", err)
	}
	defer rows.Close()
	values := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("store: could not scan a setting: %w", err)
		}
		values[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the settings: %w", err)
	}
	return values, nil
}

func (s *Store) SaveSettings(ctx context.Context, values map[string]string, by int64) error {
	batch := &pgx.Batch{}
	for key, value := range values {
		batch.Queue(saveSettingSQL, key, value, by)
	}
	if batch.Len() == 0 {
		return nil
	}
	if err := s.pool.SendBatch(ctx, batch).Close(); err != nil {
		return fmt.Errorf("store: could not save the settings: %w", err)
	}
	return nil
}

func (s *Store) ListIPBlocks(ctx context.Context) ([]IPBlock, error) {
	rows, err := s.pool.Query(ctx, listBlocksSQL)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the blocked addresses: %w", err)
	}
	defer rows.Close()
	blocks := make([]IPBlock, 0)
	for rows.Next() {
		var block IPBlock
		if err := rows.Scan(&block.ID, &block.CIDR, &block.Reason, &block.CreatedBy,
			&block.CreatedAt, &block.ExpiresAt); err != nil {
			return nil, fmt.Errorf("store: could not scan a blocked address: %w", err)
		}
		blocks = append(blocks, block)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the blocked addresses: %w", err)
	}
	return blocks, nil
}

func (s *Store) ActiveIPBlocks(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, activeBlocksSQL)
	if err != nil {
		return nil, fmt.Errorf("store: could not read the blocked addresses: %w", err)
	}
	defer rows.Close()
	ranges := make([]string, 0)
	for rows.Next() {
		var cidr string
		if err := rows.Scan(&cidr); err != nil {
			return nil, fmt.Errorf("store: could not scan a blocked address: %w", err)
		}
		ranges = append(ranges, cidr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the blocked addresses: %w", err)
	}
	return ranges, nil
}

func (s *Store) BlockIP(ctx context.Context, cidr, reason string, by int64,
	lifetime time.Duration) (int64, error) {
	var id int64
	if err := s.pool.QueryRow(ctx, insertBlockSQL, cidr, reason, by, lifetime.Seconds()).Scan(&id); err != nil {
		return 0, fmt.Errorf("store: could not block %q: %w", cidr, err)
	}
	return id, nil
}

func (s *Store) UnblockIP(ctx context.Context, blockID int64) (string, error) {
	var cidr string
	err := s.pool.QueryRow(ctx, deleteBlockSQL, blockID).Scan(&cidr)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store: could not unblock the address %d: %w", blockID, err)
	}
	return cidr, nil
}

func scanApproval(row pgx.Row) (Approval, error) {
	var approval Approval
	err := row.Scan(
		&approval.ID,
		&approval.Action,
		&approval.TargetType,
		&approval.TargetID,
		&approval.Summary,
		&approval.Payload,
		&approval.RequestedBy,
		&approval.Requester,
		&approval.Status,
		&approval.CreatedAt,
		&approval.ExpiresAt,
		&approval.DecidedAt,
		&approval.DecidedBy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Approval{}, ErrNotFound
	}
	if err != nil {
		return Approval{}, fmt.Errorf("store: could not read the request: %w", err)
	}
	return approval, nil
}

func (s *Store) CreateApproval(ctx context.Context, approval Approval, window time.Duration) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, insertApprovalSQL, approval.Action, approval.TargetType,
		approval.TargetID, approval.Summary, approval.Payload, approval.RequestedBy,
		window.Seconds()).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("store: could not record the %s request: %w", approval.Action, err)
	}
	return id, nil
}

func (s *Store) ListApprovals(ctx context.Context) ([]Approval, error) {
	if _, err := s.pool.Exec(ctx, expireApprovalsSQL); err != nil {
		return nil, fmt.Errorf("store: could not expire the old requests: %w", err)
	}
	rows, err := s.pool.Query(ctx, listApprovalsSQL)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the requests: %w", err)
	}
	defer rows.Close()
	approvals := make([]Approval, 0)
	for rows.Next() {
		approval, err := scanApproval(rows)
		if err != nil {
			return nil, err
		}
		approvals = append(approvals, approval)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the requests: %w", err)
	}
	return approvals, nil
}

func (s *Store) LoadApproval(ctx context.Context, approvalID int64) (Approval, error) {
	return scanApproval(s.pool.QueryRow(ctx, loadApprovalSQL, approvalID))
}

func (s *Store) DecideApproval(ctx context.Context, approvalID, deciderID int64,
	status string) (Approval, error) {
	approval, err := s.LoadApproval(ctx, approvalID)
	if err != nil {
		return Approval{}, err
	}
	if approval.RequestedBy == deciderID {
		return Approval{}, ErrOwnApproval
	}
	var id int64
	err = s.pool.QueryRow(ctx, decideApprovalSQL, approvalID, status, deciderID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Approval{}, ErrApprovalGone
	}
	if err != nil {
		return Approval{}, fmt.Errorf("store: could not decide the request %d: %w", approvalID, err)
	}
	return approval, nil
}

func (s *Store) OtherOfficeAdmins(ctx context.Context, userID int64) (int, error) {
	var total int
	if err := s.pool.QueryRow(ctx, otherOfficeAdminsSQL, userID).Scan(&total); err != nil {
		return 0, fmt.Errorf("store: could not count the other office admins: %w", err)
	}
	return total, nil
}

func (s *Store) RegisterFailure(ctx context.Context, userID int64) (bool, error) {
	var failures int
	var lockedUntil *time.Time
	err := s.pool.QueryRow(ctx, registerFailureSQL, userID, MaxFailedSignIns,
		LockWindow.String()).Scan(&failures, &lockedUntil)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrNotFound
	}
	if err != nil {
		return false, fmt.Errorf("store: could not record a failed sign-in: %w", err)
	}
	return lockedUntil != nil && lockedUntil.After(time.Now()), nil
}

func (s *Store) ClearFailures(ctx context.Context, userID int64) error {
	if _, err := s.pool.Exec(ctx, clearFailuresSQL, userID); err != nil {
		return fmt.Errorf("store: could not clear the failed sign-ins of user %d: %w", userID, err)
	}
	return nil
}

func (s *Store) LockedAccounts(ctx context.Context) ([]LockedAccount, error) {
	rows, err := s.pool.Query(ctx, lockedAccountsSQL)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the locked accounts: %w", err)
	}
	defer rows.Close()
	locked := make([]LockedAccount, 0)
	for rows.Next() {
		var account LockedAccount
		if err := rows.Scan(&account.ID, &account.Identifier, &account.Role, &account.Failures,
			&account.LockedUntil); err != nil {
			return nil, fmt.Errorf("store: could not scan a locked account: %w", err)
		}
		locked = append(locked, account)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the locked accounts: %w", err)
	}
	return locked, nil
}

func (s *Store) FailuresByAddress(ctx context.Context, since time.Time) ([]FailureCount, error) {
	return s.failures(ctx, failuresByIPSQL, since)
}

func (s *Store) FailuresByAccount(ctx context.Context, since time.Time) ([]FailureCount, error) {
	return s.failures(ctx, failuresByAccountSQL, since)
}

func (s *Store) failures(ctx context.Context, query string, since time.Time) ([]FailureCount, error) {
	rows, err := s.pool.Query(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("store: could not count the failed sign-ins: %w", err)
	}
	defer rows.Close()
	counts := make([]FailureCount, 0)
	for rows.Next() {
		var count FailureCount
		if err := rows.Scan(&count.Label, &count.Total); err != nil {
			return nil, fmt.Errorf("store: could not scan a failed sign-in count: %w", err)
		}
		counts = append(counts, count)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the failed sign-in counts: %w", err)
	}
	return counts, nil
}
