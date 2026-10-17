package upload

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testMaxMB = 1

func multipartRequest(t *testing.T, fields map[string]string, filename, content string) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatalf("could not write the field %q: %v", name, err)
		}
	}
	if filename != "" {
		part, err := writer.CreateFormFile(FieldName, filename)
		if err != nil {
			t.Fatalf("could not create the file part: %v", err)
		}
		if _, err := part.Write([]byte(content)); err != nil {
			t.Fatalf("could not write the file part: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("could not close the multipart writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/notices", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestSaverSave(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  string
		fields   map[string]string
		wantErr  error
		wantFile bool
	}{
		{
			name:     "a pdf is accepted",
			filename: "notice.pdf",
			content:  signature + "1.7 body",
			fields:   map[string]string{"title": "Exam routine"},
			wantFile: true,
		},
		{
			name:     "an upper case extension is accepted",
			filename: "NOTICE.PDF",
			content:  signature + "1.4",
			wantFile: true,
		},
		{
			name:    "no file at all is allowed",
			fields:  map[string]string{"title": "Exam routine"},
			wantErr: nil,
		},
		{
			name:     "another kind of file inside a pdf name is rejected",
			filename: "notice.pdf",
			content:  "PK\x03\x04 this is a zip",
			wantErr:  ErrNotPDF,
		},
		{
			name:     "a pdf body under another extension is rejected",
			filename: "notice.txt",
			content:  signature + "1.7",
			wantErr:  ErrNotPDF,
		},
		{
			name:     "an empty file is rejected",
			filename: "notice.pdf",
			wantErr:  ErrNotPDF,
		},
		{
			name:     "a file over the limit is rejected",
			filename: "notice.pdf",
			content:  signature + strings.Repeat("x", testMaxMB<<20),
			wantErr:  ErrTooLarge,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			saver := NewSaver(t.TempDir(), testMaxMB)
			path, err := saver.Save(multipartRequest(t, tc.fields, tc.filename, tc.content))
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Save() error = %v, want %v", err, tc.wantErr)
			}
			if !tc.wantFile {
				if path != "" {
					t.Fatalf("Save() = %q, want an empty path", path)
				}
				return
			}
			name := strings.TrimPrefix(path, URLPrefix)
			if !strings.HasPrefix(path, URLPrefix) || !namePattern.MatchString(name) {
				t.Fatalf("Save() = %q, want %s<32 hex characters>.pdf", path, URLPrefix)
			}
			saved, err := os.ReadFile(filepath.Join(saver.Dir(), name))
			if err != nil {
				t.Fatalf("the saved file could not be read: %v", err)
			}
			if string(saved) != tc.content {
				t.Errorf("the saved file holds %d bytes, want %d", len(saved), len(tc.content))
			}
		})
	}
}

func TestSaverSaveRejectsABrokenForm(t *testing.T) {
	saver := NewSaver(t.TempDir(), testMaxMB)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/notices", strings.NewReader("not a form"))
	req.Header.Set("Content-Type", "application/json")
	if _, err := saver.Save(req); !errors.Is(err, ErrBadForm) {
		t.Fatalf("Save() error = %v, want %v", err, ErrBadForm)
	}
}

func TestSaverSaveKeepsTheFieldsReadable(t *testing.T) {
	saver := NewSaver(t.TempDir(), testMaxMB)
	req := multipartRequest(t, map[string]string{"title": "Exam routine", "batch": "CSE-23"}, "notice.pdf", signature+"1.7")
	if _, err := saver.Save(req); err != nil {
		t.Fatalf("Save() returned an unexpected error: %v", err)
	}
	if got := req.FormValue("title"); got != "Exam routine" {
		t.Errorf("title = %q, want %q", got, "Exam routine")
	}
	if got := req.FormValue("batch"); got != "CSE-23" {
		t.Errorf("batch = %q, want %q", got, "CSE-23")
	}
}

func TestSaverDelete(t *testing.T) {
	saver := NewSaver(t.TempDir(), testMaxMB)
	path, err := saver.Save(multipartRequest(t, nil, "notice.pdf", signature+"1.7"))
	if err != nil {
		t.Fatalf("Save() returned an unexpected error: %v", err)
	}
	target := filepath.Join(saver.Dir(), strings.TrimPrefix(path, URLPrefix))
	tests := []struct {
		name  string
		value string
	}{
		{name: "the saved path", value: path},
		{name: "the same path again", value: path},
		{name: "an empty path", value: ""},
		{name: "a traversal attempt", value: "/uploads/../../secret.pdf"},
		{name: "a name that is not a random pdf", value: "/uploads/notice.pdf"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := saver.Delete(tc.value); err != nil {
				t.Fatalf("Delete(%q) returned an unexpected error: %v", tc.value, err)
			}
		})
	}
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the file still exists after Delete(), error = %v", err)
	}
}
