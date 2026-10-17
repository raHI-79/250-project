package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

const BatchAll = "ALL"

type NewMaterial struct {
	Batch     string
	Title     string
	Link      string
	CreatedBy int64
}

type MaterialRow struct {
	ID            int64
	Batch         string
	Title         string
	Link          string
	CreatedByName string
	CreatedAt     time.Time
}

const (
	materialColumns = `m.id, m.batch, m.title, m.link, COALESCE(s.full_name, t.name, 'Administration'), m.created_at`
	materialJoins   = `FROM materials m
JOIN users u ON u.id = m.created_by
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id`
	insertMaterialSQL = `WITH inserted AS (
    INSERT INTO materials (batch, title, link, created_by)
    VALUES ($1, $2, $3, $4)
    RETURNING id, batch, title, link, created_by, created_at
)
SELECT m.id, m.batch, m.title, m.link, COALESCE(s.full_name, t.name, 'Administration'), m.created_at
FROM inserted m
JOIN users u ON u.id = m.created_by
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id`
	listMaterialsSQL = `SELECT ` + materialColumns + `
` + materialJoins + `
WHERE ($1::text = '' OR m.batch = $1::text)
ORDER BY m.created_at DESC, m.id DESC`
	listCreatorMaterialsSQL = `SELECT ` + materialColumns + `
` + materialJoins + `
WHERE m.created_by = $1
ORDER BY m.created_at DESC, m.id DESC`
	listStudentMaterialsSQL = `SELECT ` + materialColumns + `
` + materialJoins + `
WHERE m.batch = $1::text OR m.batch = 'ALL'
ORDER BY m.created_at DESC, m.id DESC`
	deleteMaterialSQL = `DELETE FROM materials WHERE id = $1 AND created_by = $2 RETURNING id`
)

func (s *Store) CreateMaterial(ctx context.Context, material NewMaterial) (MaterialRow, error) {
	var row MaterialRow
	err := s.pool.QueryRow(ctx, insertMaterialSQL,
		material.Batch,
		material.Title,
		material.Link,
		material.CreatedBy,
	).Scan(
		&row.ID,
		&row.Batch,
		&row.Title,
		&row.Link,
		&row.CreatedByName,
		&row.CreatedAt,
	)
	if err != nil {
		return MaterialRow{}, fmt.Errorf("store: could not create the material: %w", err)
	}
	return row, nil
}

func (s *Store) ListMaterials(ctx context.Context, batch string) ([]MaterialRow, error) {
	return s.materials(ctx, listMaterialsSQL, batch)
}

func (s *Store) ListCreatorMaterials(ctx context.Context, createdBy int64) ([]MaterialRow, error) {
	return s.materials(ctx, listCreatorMaterialsSQL, createdBy)
}

func (s *Store) ListStudentMaterials(ctx context.Context, batch string) ([]MaterialRow, error) {
	return s.materials(ctx, listStudentMaterialsSQL, batch)
}

func (s *Store) DeleteMaterial(ctx context.Context, materialID, createdBy int64) error {
	var id int64
	err := s.pool.QueryRow(ctx, deleteMaterialSQL, materialID, createdBy).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not delete the material %d: %w", materialID, err)
	}
	return nil
}

func (s *Store) materials(ctx context.Context, query string, args ...any) ([]MaterialRow, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the materials: %w", err)
	}
	defer rows.Close()
	materials := make([]MaterialRow, 0)
	for rows.Next() {
		var row MaterialRow
		err := rows.Scan(
			&row.ID,
			&row.Batch,
			&row.Title,
			&row.Link,
			&row.CreatedByName,
			&row.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a material: %w", err)
		}
		materials = append(materials, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the materials: %w", err)
	}
	return materials, nil
}
