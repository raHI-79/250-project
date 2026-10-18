package upload

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func attachmentRequest(t *testing.T, name string, data []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if name != "" {
		part, err := writer.CreateFormFile(AttachmentField, name)
		if err != nil {
			t.Fatalf("could not create the file part: %v", err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatalf("could not write the file part: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("could not close the multipart writer: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/messages/conversations/1/messages",
		bytes.NewReader(body.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func zipBytes() []byte {
	return append([]byte("PK\x03\x04"), bytes.Repeat([]byte{0}, 40)...)
}

func docBytes() []byte {
	return append([]byte("\xd0\xcf\x11\xe0\xa1\xb1\x1a\xe1"), bytes.Repeat([]byte{0}, 40)...)
}

func TestAttachmentsSave(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		data     []byte
		skip     bool
		wantExt  string
		wantMime string
		wantName string
		wantErr  error
	}{
		{
			name:     "pdf",
			fileName: "notice.pdf",
			data:     []byte("%PDF-1.7 a real notice"),
			wantExt:  ".pdf",
			wantMime: "application/pdf",
			wantName: "notice.pdf",
		},
		{
			name:     "png",
			fileName: "photo.png",
			data:     pngBytes(t, 20, 20),
			wantExt:  ".png",
			wantMime: "image/png",
			wantName: "photo.png",
		},
		{
			name:     "jpeg",
			fileName: "photo.jpeg",
			data:     jpegBytes(t, 20, 20),
			wantExt:  ".jpeg",
			wantMime: "image/jpeg",
			wantName: "photo.jpeg",
		},
		{
			name:     "webp",
			fileName: "photo.webp",
			data:     webpBytes(t, 20, 20),
			wantExt:  ".webp",
			wantMime: "image/webp",
			wantName: "photo.webp",
		},
		{
			name:     "docx is a zip by content",
			fileName: "report.docx",
			data:     zipBytes(),
			wantExt:  ".docx",
			wantMime: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			wantName: "report.docx",
		},
		{
			name:     "legacy doc",
			fileName: "report.doc",
			data:     docBytes(),
			wantExt:  ".doc",
			wantMime: "application/msword",
			wantName: "report.doc",
		},
		{
			name:     "xlsx",
			fileName: "marks.xlsx",
			data:     zipBytes(),
			wantExt:  ".xlsx",
			wantMime: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			wantName: "marks.xlsx",
		},
		{
			name:     "pptx",
			fileName: "slides.pptx",
			data:     zipBytes(),
			wantExt:  ".pptx",
			wantMime: "application/vnd.openxmlformats-officedocument.presentationml.presentation",
			wantName: "slides.pptx",
		},
		{
			name:     "csv",
			fileName: "marks.csv",
			data:     []byte("reg_no,marks\n2023331079,80\n"),
			wantExt:  ".csv",
			wantMime: "text/csv",
			wantName: "marks.csv",
		},
		{
			name:     "txt",
			fileName: "notes.txt",
			data:     []byte("remember the lab sheet"),
			wantExt:  ".txt",
			wantMime: "text/plain",
			wantName: "notes.txt",
		},
		{
			name:     "zip",
			fileName: "bundle.zip",
			data:     zipBytes(),
			wantExt:  ".zip",
			wantMime: "application/zip",
			wantName: "bundle.zip",
		},
		{
			name:     "an unsafe original name falls back",
			fileName: "../../etc/pass word\"evil.pdf",
			data:     []byte("%PDF-1.7 odd name"),
			wantExt:  ".pdf",
			wantMime: "application/pdf",
			wantName: "attachment.pdf",
		},
		{name: "no file at all", skip: true},
		{
			name:     "an executable",
			fileName: "virus.exe",
			data:     []byte("MZ binary"),
			wantErr:  ErrNotAllowedType,
		},
		{
			name:     "a script",
			fileName: "hack.js",
			data:     []byte("alert(1)"),
			wantErr:  ErrNotAllowedType,
		},
		{
			name:     "an svg pretending to be a png",
			fileName: "logo.png",
			data:     []byte("<svg xmlns=\"http://www.w3.org/2000/svg\"></svg>"),
			wantErr:  ErrNotAllowedType,
		},
		{
			name:     "html renamed to txt",
			fileName: "page.txt",
			data:     []byte("<!DOCTYPE html><html><body>hi</body></html>"),
			wantErr:  ErrNotAllowedType,
		},
		{
			name:     "a pdf renamed to png",
			fileName: "notice.png",
			data:     []byte("%PDF-1.7 not a photo"),
			wantErr:  ErrNotAllowedType,
		},
		{
			name:     "no extension",
			fileName: "notice",
			data:     []byte("%PDF-1.7 nameless"),
			wantErr:  ErrNotAllowedType,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			files := NewAttachments(t.TempDir(), DefaultAttachmentMB)
			name := tc.fileName
			if tc.skip {
				name = ""
			}
			saved, err := files.Save(attachmentRequest(t, name, tc.data))
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("Save() error = %v, want %v", err, tc.wantErr)
				}
				if storedFiles(t, files) != 0 {
					t.Error("a refused file stayed on disk")
				}
				return
			}
			if err != nil {
				t.Fatalf("Save() returned an unexpected error: %v", err)
			}
			if tc.skip {
				if saved.Path != "" {
					t.Errorf("Save() returned %q without a file, want an empty attachment", saved.Path)
				}
				return
			}
			if !strings.HasPrefix(saved.Path, AttachmentURLPrefix) {
				t.Errorf("path = %q, want a path under %q", saved.Path, AttachmentURLPrefix)
			}
			if !strings.HasSuffix(saved.Path, tc.wantExt) {
				t.Errorf("path = %q, want the %q extension", saved.Path, tc.wantExt)
			}
			if !attachmentNamePattern.MatchString(filepath.Base(saved.Path)) {
				t.Errorf("stored name %q does not match the strict pattern", filepath.Base(saved.Path))
			}
			if saved.Mime != tc.wantMime {
				t.Errorf("mime = %q, want %q", saved.Mime, tc.wantMime)
			}
			if saved.Name != tc.wantName {
				t.Errorf("name = %q, want %q", saved.Name, tc.wantName)
			}
			if saved.Size != int64(len(tc.data)) {
				t.Errorf("size = %d, want %d", saved.Size, len(tc.data))
			}
			if storedFiles(t, files) != 1 {
				t.Errorf("%d files are on disk, want 1", storedFiles(t, files))
			}
		})
	}
}

func TestAttachmentsRejectLargeFiles(t *testing.T) {
	files := NewAttachments(t.TempDir(), 1)
	oversized := make([]byte, (1<<20)+formOverhead+1)
	copy(oversized, []byte("%PDF-1.7 "))
	if _, err := files.Save(attachmentRequest(t, "big.pdf", oversized)); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("Save() error = %v, want ErrTooLarge", err)
	}
	if storedFiles(t, files) != 0 {
		t.Error("the oversized file stayed on disk")
	}
}

func TestAttachmentsDelete(t *testing.T) {
	files := NewAttachments(t.TempDir(), DefaultAttachmentMB)
	saved, err := files.Save(attachmentRequest(t, "notice.pdf", []byte("%PDF-1.7 keep")))
	if err != nil {
		t.Fatalf("Save() returned an unexpected error: %v", err)
	}
	tests := []struct {
		name       string
		path       string
		wantStored int
	}{
		{name: "another name", path: AttachmentURLPrefix + "0123456789abcdef0123456789abcdef.pdf", wantStored: 1},
		{name: "a traversal attempt", path: "../../etc/passwd", wantStored: 1},
		{name: "an empty path", path: "", wantStored: 1},
		{name: "the saved file", path: saved.Path, wantStored: 0},
		{name: "the saved file again", path: saved.Path, wantStored: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := files.Delete(tc.path); err != nil {
				t.Fatalf("Delete() returned an unexpected error: %v", err)
			}
			if stored := storedFiles(t, files); stored != tc.wantStored {
				t.Errorf("%d files are on disk, want %d", stored, tc.wantStored)
			}
		})
	}
}

func TestAttachmentsServe(t *testing.T) {
	files := NewAttachments(t.TempDir(), DefaultAttachmentMB)
	inline, err := files.Save(attachmentRequest(t, "notice.pdf", []byte("%PDF-1.7 inline")))
	if err != nil {
		t.Fatalf("Save() returned an unexpected error: %v", err)
	}
	download, err := files.Save(attachmentRequest(t, "marks.csv", []byte("reg_no,marks\n")))
	if err != nil {
		t.Fatalf("Save() returned an unexpected error: %v", err)
	}
	tests := []struct {
		name            string
		pathValue       string
		query           string
		wantStatus      int
		wantType        string
		wantDisposition string
	}{
		{
			name:            "a pdf is shown inline",
			pathValue:       filepath.Base(inline.Path),
			wantStatus:      http.StatusOK,
			wantType:        "application/pdf",
			wantDisposition: "inline",
		},
		{
			name:            "a csv is downloaded",
			pathValue:       filepath.Base(download.Path),
			wantStatus:      http.StatusOK,
			wantType:        "text/csv",
			wantDisposition: "attachment",
		},
		{
			name:            "a csv keeps the original name",
			pathValue:       filepath.Base(download.Path),
			query:           "?name=marks.csv",
			wantStatus:      http.StatusOK,
			wantType:        "text/csv",
			wantDisposition: `attachment; filename="marks.csv"`,
		},
		{
			name:            "a hostile download name is ignored",
			pathValue:       filepath.Base(download.Path),
			query:           "?name=" + url.QueryEscape(`a"; rm -rf /.csv`),
			wantStatus:      http.StatusOK,
			wantType:        "text/csv",
			wantDisposition: "attachment",
		},
		{
			name:            "a name with another extension is ignored",
			pathValue:       filepath.Base(download.Path),
			query:           "?name=marks.exe",
			wantStatus:      http.StatusOK,
			wantType:        "text/csv",
			wantDisposition: "attachment",
		},
		{name: "another name", pathValue: "0123456789abcdef0123456789abcdef.pdf", wantStatus: http.StatusNotFound},
		{name: "a traversal attempt", pathValue: "../" + filepath.Base(inline.Path), wantStatus: http.StatusNotFound},
		{name: "a refused extension", pathValue: "0123456789abcdef0123456789abcdef.exe", wantStatus: http.StatusNotFound},
		{name: "an empty name", pathValue: "", wantStatus: http.StatusNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, AttachmentURLPrefix+tc.pathValue+tc.query, nil)
			request.SetPathValue(PathValueName, tc.pathValue)
			rec := httptest.NewRecorder()
			files.Serve()(rec, request)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantStatus != http.StatusOK {
				return
			}
			if got := rec.Header().Get("Content-Type"); got != tc.wantType {
				t.Errorf("Content-Type = %q, want %q", got, tc.wantType)
			}
			if got := rec.Header().Get("Content-Disposition"); got != tc.wantDisposition {
				t.Errorf("Content-Disposition = %q, want %q", got, tc.wantDisposition)
			}
			if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Error("the file is served without the nosniff header")
			}
		})
	}
}

func TestAttachmentURL(t *testing.T) {
	tests := []struct {
		name string
		path string
		file string
		want string
	}{
		{name: "empty", path: "", file: "notice.pdf", want: ""},
		{
			name: "an inline pdf keeps a clean path",
			path: AttachmentURLPrefix + "0123456789abcdef0123456789abcdef.pdf",
			file: "notice.pdf",
			want: AttachmentURLPrefix + "0123456789abcdef0123456789abcdef.pdf",
		},
		{
			name: "an image keeps a clean path",
			path: AttachmentURLPrefix + "0123456789abcdef0123456789abcdef.png",
			file: "photo.png",
			want: AttachmentURLPrefix + "0123456789abcdef0123456789abcdef.png",
		},
		{
			name: "a document carries its name",
			path: AttachmentURLPrefix + "0123456789abcdef0123456789abcdef.docx",
			file: "report.docx",
			want: AttachmentURLPrefix + "0123456789abcdef0123456789abcdef.docx?name=report.docx",
		},
		{
			name: "a hostile name is dropped",
			path: AttachmentURLPrefix + "0123456789abcdef0123456789abcdef.docx",
			file: `evil"name.docx`,
			want: AttachmentURLPrefix + "0123456789abcdef0123456789abcdef.docx",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := AttachmentURL(tc.path, tc.file); got != tc.want {
				t.Errorf("AttachmentURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAttachmentInline(t *testing.T) {
	tests := []struct {
		mime string
		want bool
	}{
		{mime: "application/pdf", want: true},
		{mime: "image/png", want: true},
		{mime: "image/webp", want: true},
		{mime: "text/csv", want: false},
		{mime: "application/zip", want: false},
		{mime: "application/octet-stream", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.mime, func(t *testing.T) {
			if got := AttachmentInline(tc.mime); got != tc.want {
				t.Errorf("AttachmentInline(%q) = %t, want %t", tc.mime, got, tc.want)
			}
		})
	}
}

func storedFiles(t *testing.T, files *Attachments) int {
	t.Helper()
	entries, err := os.ReadDir(files.Dir())
	if errors.Is(err, os.ErrNotExist) {
		return 0
	}
	if err != nil {
		t.Fatalf("could not read the attachment folder: %v", err)
	}
	return len(entries)
}
