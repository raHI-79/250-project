package db

import (
	"context"
	"embed"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"io/fs"
	"log"
	"path"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const (
	migrationsDir            = "migrations"
	migrationsExtension      = ".sql"
	createMigrationsTableSQL = `CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`
)

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	return migrate(ctx, pool, migrationFiles)
}

func migrate(ctx context.Context, pool *pgxpool.Pool, files fs.FS) error {
	if _, err := pool.Exec(ctx, createMigrationsTableSQL); err != nil {
		return fmt.Errorf("db: could not create the schema_migrations table: %w", err)
	}
	versions, err := migrationVersions(files)
	if err != nil {
		return err
	}
	applied, err := appliedVersions(ctx, pool)
	if err != nil {
		return err
	}
	pending := 0
	for _, version := range versions {
		if applied[version] {
			continue
		}
		statements, err := fs.ReadFile(files, path.Join(migrationsDir, version))
		if err != nil {
			return fmt.Errorf("db: could not read migration %s: %w", version, err)
		}
		if err := applyMigration(ctx, pool, version, string(statements)); err != nil {
			return err
		}
		log.Printf("db: applied migration %s", version)
		pending++
	}
	if pending == 0 {
		log.Println("db: schema is up to date")
	}
	return nil
}

func applyMigration(ctx context.Context, pool *pgxpool.Pool, version, statements string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("db: could not start a transaction for migration %s: %w", version, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, statements); err != nil {
		return fmt.Errorf("db: migration %s failed: %w", version, err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
		return fmt.Errorf("db: could not record migration %s: %w", version, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("db: could not commit migration %s: %w", version, err)
	}
	return nil
}

func migrationVersions(files fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(files, migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("db: could not list the migrations directory: %w", err)
	}
	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), migrationsExtension) {
			continue
		}
		versions = append(versions, entry.Name())
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("db: no migrations found in %s", migrationsDir)
	}
	sort.Strings(versions)
	return versions, nil
}

func appliedVersions(ctx context.Context, pool *pgxpool.Pool) (map[string]bool, error) {
	rows, err := pool.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("db: could not read applied migrations: %w", err)
	}
	defer rows.Close()
	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("db: could not scan an applied migration: %w", err)
		}
		applied[version] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: could not read applied migrations: %w", err)
	}
	return applied, nil
}
