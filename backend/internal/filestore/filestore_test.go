package filestore

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raHI-79/250-project/backend/internal/db"
	"github.com/raHI-79/250-project/backend/internal/dbtest"
	"github.com/raHI-79/250-project/backend/internal/upload"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const (
	pdfName   = "0123456789abcdef0123456789abcdef.pdf"
	photoName = "fedcba9876543210fedcba9876543210.jpg"
	fileName  = "00112233445566778899aabbccddeeff.docx"
)

func TestMain(m *testing.M) {
	if err := dbtest.LoadEnv(); err != nil {
		log.Fatalf("filestore: could not load backend/.env: %v", err)
	}
	os.Exit(m.Run())
}

func newDatabase(t *testing.T) (context.Context, *pgxpool.Pool, *Database) {
	t.Helper()
	ctx, pool := dbtest.Pool(t)
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("db.Migrate() returned an unexpected error: %v", err)
	}
	return ctx, pool, New(pool)
}

func TestDatabaseStoreRoundTrip(t *testing.T) {
	ctx, _, files := newDatabase(t)
	data := []byte("%PDF-1.7 a short document")
	if err := files.Put(ctx, upload.KindFile, pdfName, upload.ContentType, data); err != nil {
		t.Fatalf("Put() returned an unexpected error: %v", err)
	}
	blob, err := files.Get(ctx, upload.KindFile, pdfName)
	if err != nil {
		t.Fatalf("Get() returned an unexpected error: %v", err)
	}
	if string(blob.Data) != string(data) {
		t.Errorf("data = %q, want %q", blob.Data, data)
	}
	if blob.ModTime.IsZero() {
		t.Error("the stored file carries no time")
	}
	total, err := files.Count(ctx)
	if err != nil {
		t.Fatalf("Count() returned an unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("stored files = %d, want 1", total)
	}
	if err := files.Put(ctx, upload.KindFile, pdfName, upload.ContentType, data); err != nil {
		t.Fatalf("storing the same name twice returned an error: %v", err)
	}
	if again, _ := files.Count(ctx); again != 1 {
		t.Errorf("stored files = %d after a repeat, want 1", again)
	}
}

func TestDatabaseStoreKeepsTheKindsApart(t *testing.T) {
	ctx, _, files := newDatabase(t)
	if err := files.Put(ctx, upload.KindPhoto, photoName, "image/jpeg", []byte("photo")); err != nil {
		t.Fatalf("Put() returned an unexpected error: %v", err)
	}
	if _, err := files.Get(ctx, upload.KindAttachment, photoName); !errors.Is(err, upload.ErrFileNotFound) {
		t.Errorf("reading a photo as an attachment: error = %v, want ErrFileNotFound", err)
	}
	if _, err := files.Get(ctx, upload.KindPhoto, pdfName); !errors.Is(err, upload.ErrFileNotFound) {
		t.Errorf("reading an unknown name: error = %v, want ErrFileNotFound", err)
	}
}

func TestDatabaseStoreDeletes(t *testing.T) {
	ctx, _, files := newDatabase(t)
	if err := files.Put(ctx, upload.KindAttachment, fileName, "application/msword", []byte("doc")); err != nil {
		t.Fatalf("Put() returned an unexpected error: %v", err)
	}
	if err := files.Delete(ctx, upload.KindAttachment, fileName); err != nil {
		t.Fatalf("Delete() returned an unexpected error: %v", err)
	}
	if _, err := files.Get(ctx, upload.KindAttachment, fileName); !errors.Is(err, upload.ErrFileNotFound) {
		t.Errorf("error = %v, want ErrFileNotFound once it is deleted", err)
	}
	if err := files.Delete(ctx, upload.KindAttachment, fileName); err != nil {
		t.Errorf("deleting twice returned an error: %v", err)
	}
}

func TestUploadsServeFromTheDatabase(t *testing.T) {
	ctx, _, files := newDatabase(t)
	photos := upload.NewPhotosIn(files)
	attachments := upload.NewAttachmentsIn(files, 10)
	saver := upload.NewSaverIn(files, 10)
	if err := files.Put(ctx, upload.KindPhoto, photoName, "image/jpeg", []byte("photo-bytes")); err != nil {
		t.Fatalf("Put() returned an unexpected error: %v", err)
	}
	if err := files.Put(ctx, upload.KindFile, pdfName, upload.ContentType, []byte("%PDF-1.7")); err != nil {
		t.Fatalf("Put() returned an unexpected error: %v", err)
	}
	if err := files.Put(ctx, upload.KindAttachment, fileName, "application/msword", []byte("doc")); err != nil {
		t.Fatalf("Put() returned an unexpected error: %v", err)
	}
	tests := []struct {
		name     string
		handler  http.HandlerFunc
		stored   string
		wantBody string
		wantType string
	}{
		{name: "a photo", handler: photos.Serve(), stored: photoName,
			wantBody: "photo-bytes", wantType: "image/jpeg"},
		{name: "a notice file", handler: saver.Serve(), stored: pdfName,
			wantBody: "%PDF-1.7", wantType: upload.ContentType},
		{name: "a message attachment", handler: attachments.Serve(), stored: fileName,
			wantBody: "doc",
			wantType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/uploads/"+test.stored, nil)
			request.SetPathValue(upload.PathValueName, test.stored)
			rec := httptest.NewRecorder()
			test.handler(rec, request)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			if rec.Body.String() != test.wantBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), test.wantBody)
			}
			if got := rec.Header().Get("Content-Type"); got != test.wantType {
				t.Errorf("content type = %q, want %q", got, test.wantType)
			}
			if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("nosniff header = %q, want nosniff", got)
			}
			if rec.Header().Get("Content-Disposition") == "" {
				t.Error("the answer carries no content disposition")
			}
		})
	}
}

func TestUploadsAnswerNotFoundForUnknownNames(t *testing.T) {
	_, _, files := newDatabase(t)
	photos := upload.NewPhotosIn(files)
	saver := upload.NewSaverIn(files, 10)
	attachments := upload.NewAttachmentsIn(files, 10)
	tests := []struct {
		name    string
		handler http.HandlerFunc
		stored  string
	}{
		{name: "a photo that was never stored", handler: photos.Serve(), stored: photoName},
		{name: "a file that was never stored", handler: saver.Serve(), stored: pdfName},
		{name: "an attachment that was never stored", handler: attachments.Serve(), stored: fileName},
		{name: "a name that breaks the rules", handler: photos.Serve(), stored: "../secret.jpg"},
		{name: "an empty name", handler: saver.Serve(), stored: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/uploads/x", nil)
			request.SetPathValue(upload.PathValueName, test.stored)
			rec := httptest.NewRecorder()
			test.handler(rec, request)
			if rec.Code != http.StatusNotFound {
				t.Errorf("status = %d, want 404", rec.Code)
			}
		})
	}
}

func TestUploadsDeleteThroughTheDatabase(t *testing.T) {
	ctx, _, files := newDatabase(t)
	photos := upload.NewPhotosIn(files)
	if err := files.Put(ctx, upload.KindPhoto, photoName, "image/jpeg", []byte("photo")); err != nil {
		t.Fatalf("Put() returned an unexpected error: %v", err)
	}
	if err := photos.Delete(upload.PhotoURLPrefix + photoName); err != nil {
		t.Fatalf("Delete() returned an unexpected error: %v", err)
	}
	if _, err := files.Get(ctx, upload.KindPhoto, photoName); !errors.Is(err, upload.ErrFileNotFound) {
		t.Errorf("error = %v, want the photo gone", err)
	}
}
