package store

import (
	"context"
	"fmt"
)

const DirectoryLimit = 200

type DirectoryFilter struct {
	Batch      string
	BloodGroup string
	Search     string
}

type DirectoryRow struct {
	UserID     int64
	PhotoPath  string
	FullName   string
	RegNo      string
	Batch      string
	Section    string
	Phone      string
	Email      string
	BloodGroup string
	IsCR       bool
}

type TeacherInfoRow struct {
	UserID      int64
	Name        string
	Designation string
	OfficeRoom  string
	Phone       string
	Email       string
	PhotoPath   string
}

const (
	listDirectorySQL = `SELECT u.id, COALESCE(u.photo_path, ''), s.full_name, s.reg_no, s.batch, s.section,
s.phone, s.email, s.blood_group, s.is_cr
FROM students s
JOIN users u ON u.id = s.user_id
WHERE u.status = 'active'
  AND ($1::text = '' OR s.batch = $1::text)
  AND ($2::text = '' OR s.blood_group = $2::text)
  AND ($3::text = '' OR s.full_name ILIKE '%' || $3::text || '%' OR s.reg_no ILIKE '%' || $3::text || '%')
ORDER BY s.batch, s.section, s.reg_no
LIMIT $4`
	listTeacherInfoSQL = `SELECT u.id, t.name, t.designation, t.office_room, t.phone, t.email,
COALESCE(u.photo_path, '')
FROM teachers t
JOIN users u ON u.id = t.user_id
WHERE u.status = 'active'
  AND ($1::text = '' OR t.name ILIKE '%' || $1::text || '%')
ORDER BY t.name, t.id`
)

func (s *Store) ListDirectory(ctx context.Context, filter DirectoryFilter) ([]DirectoryRow, error) {
	rows, err := s.pool.Query(ctx, listDirectorySQL, filter.Batch, filter.BloodGroup, filter.Search, DirectoryLimit)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the student directory: %w", err)
	}
	defer rows.Close()
	people := make([]DirectoryRow, 0)
	for rows.Next() {
		var row DirectoryRow
		err := rows.Scan(
			&row.UserID,
			&row.PhotoPath,
			&row.FullName,
			&row.RegNo,
			&row.Batch,
			&row.Section,
			&row.Phone,
			&row.Email,
			&row.BloodGroup,
			&row.IsCR,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a directory entry: %w", err)
		}
		people = append(people, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the student directory: %w", err)
	}
	return people, nil
}

func (s *Store) ListTeacherInfo(ctx context.Context, search string) ([]TeacherInfoRow, error) {
	rows, err := s.pool.Query(ctx, listTeacherInfoSQL, search)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the teachers: %w", err)
	}
	defer rows.Close()
	teachers := make([]TeacherInfoRow, 0)
	for rows.Next() {
		var row TeacherInfoRow
		err := rows.Scan(&row.UserID, &row.Name, &row.Designation, &row.OfficeRoom,
			&row.Phone, &row.Email, &row.PhotoPath)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a teacher: %w", err)
		}
		teachers = append(teachers, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the teachers: %w", err)
	}
	return teachers, nil
}
