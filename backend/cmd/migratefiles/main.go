package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/db"
	"github.com/raHI-79/250-project/backend/internal/filestore"
	"github.com/raHI-79/250-project/backend/internal/upload"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const timeout = 10 * time.Minute

type report struct {
	Copied  int
	Skipped int
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("%v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, db.ConnectWindow)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		return err
	}
	totals, err := copyAll(ctx, pool, cfg.UploadDir)
	if err != nil {
		return err
	}
	log.Printf("migratefiles: copied %d files, left %d already stored", totals.Copied, totals.Skipped)
	return nil
}

func copyAll(ctx context.Context, pool *pgxpool.Pool, uploadDir string) (report, error) {
	if uploadDir == "" {
		uploadDir = "uploads"
	}
	files := filestore.New(pool)
	disk := upload.NewDisk(uploadDir)
	var totals report
	for _, kind := range []string{upload.KindFile, upload.KindPhoto, upload.KindAttachment} {
		counted, err := copyKind(ctx, files, disk.Dir(kind), kind)
		if err != nil {
			return report{}, err
		}
		totals.Copied += counted.Copied
		totals.Skipped += counted.Skipped
	}
	return totals, nil
}

func copyKind(ctx context.Context, files *filestore.Database, dir, kind string) (report, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return report{}, nil
	}
	if err != nil {
		return report{}, fmt.Errorf("migratefiles: could not read %s: %w", dir, err)
	}
	var totals report
	for _, entry := range entries {
		if entry.IsDir() || !upload.StoredName(kind, entry.Name()) {
			continue
		}
		if _, err := files.Get(ctx, kind, entry.Name()); err == nil {
			totals.Skipped++
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return report{}, fmt.Errorf("migratefiles: could not read %s: %w", entry.Name(), err)
		}
		if err := files.Put(ctx, kind, entry.Name(), mimeOf(entry.Name()), data); err != nil {
			return report{}, err
		}
		totals.Copied++
	}
	return totals, nil
}

func mimeOf(name string) string {
	extension := strings.ToLower(filepath.Ext(name))
	if found := mime.TypeByExtension(extension); found != "" {
		return found
	}
	return "application/octet-stream"
}
