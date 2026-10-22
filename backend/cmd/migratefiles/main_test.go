package main

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raHI-79/250-project/backend/internal/db"
	"github.com/raHI-79/250-project/backend/internal/dbtest"
	"github.com/raHI-79/250-project/backend/internal/filestore"
	"github.com/raHI-79/250-project/backend/internal/upload"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	if err := dbtest.LoadEnv(); err != nil {
		log.Fatalf("migratefiles: could not load backend/.env: %v", err)
	}
	os.Exit(m.Run())
}

func uploadDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		filepath.Join("0123456789abcdef0123456789abcdef.pdf"):           "%PDF-1.7 notice",
		filepath.Join("photos", "fedcba9876543210fedcba9876543210.jpg"): "photo bytes",
		filepath.Join("photos", "not-a-stored-name.jpg"):                "skip me",
		filepath.Join("files", "00112233445566778899aabbccddeeff.docx"): "attachment bytes",
		filepath.Join("files", "readme.txt"):                            "skip me too",
	}
	for name, body := range files {
		target := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("could not create %s: %v", target, err)
		}
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			t.Fatalf("could not write %s: %v", target, err)
		}
	}
	return dir
}

func migrated(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	ctx, pool := dbtest.Pool(t)
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("db.Migrate() returned an unexpected error: %v", err)
	}
	return ctx, pool
}

func TestCopyAllMovesEveryStoredFileOnce(t *testing.T) {
	ctx, pool := migrated(t)
	dir := uploadDir(t)
	totals, err := copyAll(ctx, pool, dir)
	if err != nil {
		t.Fatalf("copyAll() returned an unexpected error: %v", err)
	}
	if totals.Copied != 3 || totals.Skipped != 0 {
		t.Fatalf("totals = %+v, want three copied and none skipped", totals)
	}
	files := filestore.New(pool)
	wanted := []struct {
		kind string
		name string
		body string
	}{
		{kind: upload.KindFile, name: "0123456789abcdef0123456789abcdef.pdf", body: "%PDF-1.7 notice"},
		{kind: upload.KindPhoto, name: "fedcba9876543210fedcba9876543210.jpg", body: "photo bytes"},
		{kind: upload.KindAttachment, name: "00112233445566778899aabbccddeeff.docx", body: "attachment bytes"},
	}
	for _, want := range wanted {
		blob, err := files.Get(ctx, want.kind, want.name)
		if err != nil {
			t.Fatalf("%s was not copied: %v", want.name, err)
		}
		if string(blob.Data) != want.body {
			t.Errorf("%s holds %q, want %q", want.name, blob.Data, want.body)
		}
	}
	total, err := files.Count(ctx)
	if err != nil {
		t.Fatalf("Count() returned an unexpected error: %v", err)
	}
	if total != 3 {
		t.Errorf("stored files = %d, want only the three with stored names", total)
	}
}

func TestCopyAllRunsTwiceWithoutDuplicating(t *testing.T) {
	ctx, pool := migrated(t)
	dir := uploadDir(t)
	if _, err := copyAll(ctx, pool, dir); err != nil {
		t.Fatalf("the first run returned an unexpected error: %v", err)
	}
	again, err := copyAll(ctx, pool, dir)
	if err != nil {
		t.Fatalf("the second run returned an unexpected error: %v", err)
	}
	if again.Copied != 0 || again.Skipped != 3 {
		t.Errorf("second run = %+v, want nothing copied and three skipped", again)
	}
	total, err := filestore.New(pool).Count(ctx)
	if err != nil {
		t.Fatalf("Count() returned an unexpected error: %v", err)
	}
	if total != 3 {
		t.Errorf("stored files = %d after two runs, want 3", total)
	}
}

func TestCopyAllAcceptsAMissingFolder(t *testing.T) {
	ctx, pool := migrated(t)
	totals, err := copyAll(ctx, pool, filepath.Join(t.TempDir(), "nothing-here"))
	if err != nil {
		t.Fatalf("copyAll() returned an unexpected error: %v", err)
	}
	if totals.Copied != 0 || totals.Skipped != 0 {
		t.Errorf("totals = %+v, want nothing done", totals)
	}
}
