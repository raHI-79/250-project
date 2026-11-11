package upload

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func fileRequest(t *testing.T, field, name string, data []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(field, name)
	if err != nil {
		t.Fatalf("could not create the file part: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("could not write the file part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("could not close the multipart writer: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/upload", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func TestAPhotoIsRefusedUnlessItsContentIsAPicture(t *testing.T) {
	photos := NewPhotos(t.TempDir())
	tests := []struct {
		name     string
		fileName string
		data     []byte
	}{
		{name: "a script dressed as a png", fileName: "student.png", data: []byte("#!/bin/sh\necho hello\n")},
		{name: "an svg", fileName: "student.svg", data: []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`)},
		{name: "an empty file", fileName: "student.png", data: nil},
		{name: "a pdf dressed as a jpeg", fileName: "student.jpg", data: []byte("%PDF-1.4\n%%EOF\n")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := photos.Save(fileRequest(t, PhotoFieldName, test.fileName, test.data)); err == nil {
				t.Error("the uploader kept a photo it should have refused")
			}
		})
	}
}

func TestAnAttachmentIsRefusedUnlessItsTypeIsOnTheList(t *testing.T) {
	attachments := NewAttachments(t.TempDir(), 5)
	tests := []struct {
		name     string
		fileName string
		data     []byte
	}{
		{name: "a program", fileName: "notes.exe", data: []byte("MZ\x90\x00\x03")},
		{name: "a shell script", fileName: "notes.sh", data: []byte("#!/bin/sh\n")},
		{name: "an html page", fileName: "notes.html", data: []byte("<html></html>")},
		{name: "no extension at all", fileName: "notes", data: []byte("plain words")},
		{name: "a program dressed as a pdf", fileName: "notes.pdf", data: []byte("MZ\x90\x00\x03 not a pdf")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := attachments.Save(fileRequest(t, FieldName, test.fileName, test.data))
			if err == nil {
				t.Error("the uploader kept an attachment it should have refused")
			}
			if !errors.Is(err, ErrNotAllowedType) && !errors.Is(err, ErrBadForm) {
				t.Errorf("the error = %v, want it to name the type", err)
			}
		})
	}
}

func TestEveryUploaderStopsAtItsSizeLimit(t *testing.T) {
	attachments := NewAttachments(t.TempDir(), 1)
	if attachments.MaxBytes() != 1<<20 {
		t.Fatalf("the attachment limit = %d bytes, want one megabyte", attachments.MaxBytes())
	}
	big := make([]byte, 2<<20)
	copy(big, []byte("%PDF-1.4\n"))
	if _, err := attachments.Save(fileRequest(t, FieldName, "notes.pdf", big)); err == nil {
		t.Error("an attachment twice the limit was kept")
	}
	if MaxPhotoBytes <= 0 {
		t.Fatalf("the photo limit = %d bytes, which is not a limit", MaxPhotoBytes)
	}
	photos := NewPhotos(t.TempDir())
	oversized := make([]byte, MaxPhotoBytes+(1<<16))
	copy(oversized, []byte("\x89PNG\r\n\x1a\n"))
	if _, err := photos.Save(fileRequest(t, PhotoFieldName, "student.png", oversized)); err == nil {
		t.Error("a photo over the limit was kept")
	}
}
