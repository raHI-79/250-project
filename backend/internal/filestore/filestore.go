package filestore

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raHI-79/250-project/backend/internal/upload"
	"time"
)

const (
	putSQL = `INSERT INTO stored_files (name, kind, mime, size, data)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (name) DO NOTHING`
	getSQL    = `SELECT data, created_at FROM stored_files WHERE name = $1 AND kind = $2`
	deleteSQL = `DELETE FROM stored_files WHERE name = $1 AND kind = $2`
	countSQL  = `SELECT count(*) FROM stored_files`
)

type Database struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Database {
	return &Database{pool: pool}
}

func (d *Database) Put(ctx context.Context, kind, name, mime string, data []byte) error {
	_, err := d.pool.Exec(ctx, putSQL, name, kind, mime, len(data), data)
	if err != nil {
		return fmt.Errorf("filestore: could not store the file %s: %w", name, err)
	}
	return nil
}

func (d *Database) Get(ctx context.Context, kind, name string) (upload.Blob, error) {
	var data []byte
	var createdAt time.Time
	err := d.pool.QueryRow(ctx, getSQL, name, kind).Scan(&data, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return upload.Blob{}, upload.ErrFileNotFound
	}
	if err != nil {
		return upload.Blob{}, fmt.Errorf("filestore: could not read the file %s: %w", name, err)
	}
	return upload.Blob{Data: data, ModTime: createdAt}, nil
}

func (d *Database) Delete(ctx context.Context, kind, name string) error {
	if _, err := d.pool.Exec(ctx, deleteSQL, name, kind); err != nil {
		return fmt.Errorf("filestore: could not delete the file %s: %w", name, err)
	}
	return nil
}

func (d *Database) Count(ctx context.Context) (int, error) {
	var total int
	if err := d.pool.QueryRow(ctx, countSQL).Scan(&total); err != nil {
		return 0, fmt.Errorf("filestore: could not count the stored files: %w", err)
	}
	return total, nil
}
