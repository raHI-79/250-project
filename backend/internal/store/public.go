package store

import (
	"context"
	"fmt"
)

type PublicStats struct {
	Students int
	Teachers int
	Courses  int
	Batches  int
}

const publicStatsSQL = `SELECT
(SELECT COUNT(*) FROM users u JOIN students s ON s.user_id = u.id WHERE u.status = 'active'),
(SELECT COUNT(*) FROM users u JOIN teachers t ON t.user_id = u.id WHERE u.status = 'active'),
(SELECT COUNT(*) FROM courses),
(SELECT COUNT(*) FROM (
    SELECT s.batch AS batch FROM students s JOIN users u ON u.id = s.user_id WHERE u.status = 'active'
    UNION
    SELECT r.batch AS batch FROM routines r
) sources WHERE batch <> '')`

func (s *Store) PublicStats(ctx context.Context) (PublicStats, error) {
	var stats PublicStats
	err := s.pool.QueryRow(ctx, publicStatsSQL).Scan(
		&stats.Students,
		&stats.Teachers,
		&stats.Courses,
		&stats.Batches,
	)
	if err != nil {
		return PublicStats{}, fmt.Errorf("store: could not load the public stats: %w", err)
	}
	return stats, nil
}
