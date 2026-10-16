package store

import (
	"context"
	"errors"
	"fmt"
)

var ErrDuplicateCourse = errors.New("store: a course with this code already exists")

type Course struct {
	ID    int64
	Code  string
	Title string
}

const (
	insertCourseSQL = `INSERT INTO courses (code, title)
VALUES ($1, $2)
RETURNING id, code, title`
	listCoursesSQL = `SELECT id, code, title FROM courses ORDER BY code`
)

func (s *Store) CreateCourse(ctx context.Context, code, title string) (Course, error) {
	var course Course
	err := s.pool.QueryRow(ctx, insertCourseSQL, code, title).Scan(&course.ID, &course.Code, &course.Title)
	if isUniqueViolation(err) {
		return Course{}, ErrDuplicateCourse
	}
	if err != nil {
		return Course{}, fmt.Errorf("store: could not create the course %s: %w", code, err)
	}
	return course, nil
}

func (s *Store) ListCourses(ctx context.Context) ([]Course, error) {
	rows, err := s.pool.Query(ctx, listCoursesSQL)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the courses: %w", err)
	}
	defer rows.Close()
	courses := make([]Course, 0)
	for rows.Next() {
		var course Course
		if err := rows.Scan(&course.ID, &course.Code, &course.Title); err != nil {
			return nil, fmt.Errorf("store: could not scan a course: %w", err)
		}
		courses = append(courses, course)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the courses: %w", err)
	}
	return courses, nil
}
