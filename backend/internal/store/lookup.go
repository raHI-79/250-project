package store

import (
	"context"
	"fmt"
)

type TeacherOption struct {
	ID     int64
	UserID int64
	Name   string
}

const (
	listBatchesSQL  = `SELECT name FROM batches WHERE status = 'active' ORDER BY name`
	listSectionsSQL = `SELECT section FROM batches b, unnest(b.sections) AS section
WHERE b.name = $1 AND b.status = 'active'
ORDER BY section`
	listTeacherOptionsSQL = `SELECT t.id, t.user_id, t.name
FROM teachers t
JOIN users u ON u.id = t.user_id
WHERE u.status = 'active'
ORDER BY t.name, t.id`
)

func (s *Store) ListBatches(ctx context.Context) ([]string, error) {
	return s.textColumn(ctx, listBatchesSQL, "batches")
}

func (s *Store) ListSections(ctx context.Context, batch string) ([]string, error) {
	return s.textColumn(ctx, listSectionsSQL, "sections", batch)
}

func (s *Store) textColumn(ctx context.Context, query, label string, args ...any) ([]string, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the %s: %w", label, err)
	}
	defer rows.Close()
	values := make([]string, 0)
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("store: could not scan one of the %s: %w", label, err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the %s: %w", label, err)
	}
	return values, nil
}

func (s *Store) ListTeacherOptions(ctx context.Context) ([]TeacherOption, error) {
	rows, err := s.pool.Query(ctx, listTeacherOptionsSQL)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the teacher options: %w", err)
	}
	defer rows.Close()
	teachers := make([]TeacherOption, 0)
	for rows.Next() {
		var teacher TeacherOption
		if err := rows.Scan(&teacher.ID, &teacher.UserID, &teacher.Name); err != nil {
			return nil, fmt.Errorf("store: could not scan a teacher option: %w", err)
		}
		teachers = append(teachers, teacher)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the teacher options: %w", err)
	}
	return teachers, nil
}
