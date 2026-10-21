package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
)

type Teacher struct {
	ID     int64
	UserID int64
	Name   string
}

const (
	findTeacherByUserSQL = `SELECT id, user_id, name FROM teachers WHERE user_id = $1`
	listActiveCRsSQL     = `SELECT u.id
FROM users u
JOIN students s ON s.user_id = u.id
WHERE u.status = 'active' AND s.is_cr AND s.batch = $1
  AND ($2::text = '' OR $2::text = 'ALL' OR s.section = $2::text)
ORDER BY u.id`
)

func (s *Store) GetTeacherByUser(ctx context.Context, userID int64) (Teacher, error) {
	var teacher Teacher
	err := s.pool.QueryRow(ctx, findTeacherByUserSQL, userID).Scan(&teacher.ID, &teacher.UserID, &teacher.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return Teacher{}, ErrNotFound
	}
	if err != nil {
		return Teacher{}, fmt.Errorf("store: could not load the teacher of the user %d: %w", userID, err)
	}
	return teacher, nil
}

func (s *Store) ListActiveCRs(ctx context.Context, batch, section string) ([]int64, error) {
	rows, err := s.pool.Query(ctx, listActiveCRsSQL, batch, section)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the class representatives of %s: %w", batch, err)
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("store: could not scan a class representative: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the class representatives: %w", err)
	}
	return ids, nil
}
