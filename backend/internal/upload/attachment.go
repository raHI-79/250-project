package upload

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	AttachmentField     = "file"
	AttachmentDirName   = "files"
	AttachmentURLPrefix = "/uploads/files/"
	DefaultAttachmentMB = 10
	downloadNameQuery   = "name"
	sniffBytes          = 512
)

var (
	ErrNotAllowedType = errors.New("upload: this file type is not allowed")

	attachmentTypes = map[string]string{
		"pdf":  "application/pdf",
		"jpg":  "image/jpeg",
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"webp": "image/webp",
		"gif":  "image/gif",
		"doc":  "application/msword",
		"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"xls":  "application/vnd.ms-excel",
		"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"ppt":  "application/vnd.ms-powerpoint",
		"pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"csv":  "text/csv",
		"txt":  "text/plain",
		"zip":  "application/zip",
	}

	sniffedTypes = map[string]string{
		"pdf":  "application/pdf",
		"jpg":  "image/jpeg",
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"webp": "image/webp",
		"gif":  "image/gif",
	}

	inlineTypes = map[string]bool{
		"pdf":  true,
		"jpg":  true,
		"jpeg": true,
		"png":  true,
		"webp": true,
		"gif":  true,
	}

	attachmentNamePattern = regexp.MustCompile(
		`^[a-f0-9]{32}\.(pdf|jpe?g|png|webp|gif|docx?|xlsx?|pptx?|csv|txt|zip)$`)

	downloadNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._()+-]{0,119}$`)
)

type Attachment struct {
	Path string
	Name string
	Mime string
	Size int64
}

type Attachments struct {
	dir      string
	maxBytes int64
}

func NewAttachments(uploadDir string, maxMB int) *Attachments {
	if uploadDir == "" {
		uploadDir = "uploads"
	}
	if maxMB <= 0 {
		maxMB = DefaultAttachmentMB
	}
	return &Attachments{
		dir:      filepath.Join(uploadDir, AttachmentDirName),
		maxBytes: int64(maxMB) << 20,
	}
}

func (a *Attachments) Dir() string {
	return a.dir
}

func (a *Attachments) MaxBytes() int64 {
	return a.maxBytes
}

func (a *Attachments) Parse(r *http.Request) error {
	if r.MultipartForm != nil {
		return nil
	}
	r.Body = http.MaxBytesReader(nil, r.Body, a.maxBytes+formOverhead)
	if err := r.ParseMultipartForm(parseMemory); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return ErrTooLarge
		}
		return ErrBadForm
	}
	return nil
}

func (a *Attachments) Save(r *http.Request) (Attachment, error) {
	if err := a.Parse(r); err != nil {
		return Attachment{}, err
	}
	file, header, err := r.FormFile(AttachmentField)
	if errors.Is(err, http.ErrMissingFile) {
		return Attachment{}, nil
	}
	if err != nil {
		return Attachment{}, ErrBadForm
	}
	defer func() { _ = file.Close() }()
	if header.Size > a.maxBytes {
		return Attachment{}, ErrTooLarge
	}
	extension := strings.ToLower(strings.TrimPrefix(path.Ext(header.Filename), "."))
	mime, allowed := attachmentTypes[extension]
	if !allowed {
		return Attachment{}, ErrNotAllowedType
	}
	if err := checkAttachmentContent(file, extension); err != nil {
		return Attachment{}, err
	}
	stored, size, err := a.store(file, extension)
	if err != nil {
		return Attachment{}, err
	}
	return Attachment{Path: stored, Name: safeOriginalName(header.Filename, extension), Mime: mime, Size: size}, nil
}

func (a *Attachments) store(file multipart.File, extension string) (string, int64, error) {
	if err := os.MkdirAll(a.dir, directoryMode); err != nil {
		return "", 0, fmt.Errorf("upload: could not create the attachment folder: %w", err)
	}
	name, err := randomName()
	if err != nil {
		return "", 0, err
	}
	name = strings.TrimSuffix(name, Extension) + "." + extension
	target := filepath.Join(a.dir, name)
	destination, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fileMode)
	if err != nil {
		return "", 0, fmt.Errorf("upload: could not create the attachment: %w", err)
	}
	written, err := io.Copy(destination, io.LimitReader(file, a.maxBytes+1))
	if closeErr := destination.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(target)
		return "", 0, fmt.Errorf("upload: could not write the attachment: %w", err)
	}
	if written > a.maxBytes {
		_ = os.Remove(target)
		return "", 0, ErrTooLarge
	}
	return AttachmentURLPrefix + name, written, nil
}

func (a *Attachments) Delete(storedPath string) error {
	name := path.Base(strings.TrimSpace(storedPath))
	if !attachmentNamePattern.MatchString(name) {
		return nil
	}
	if err := os.Remove(filepath.Join(a.dir, name)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("upload: could not delete the attachment %s: %w", name, err)
	}
	return nil
}

func (a *Attachments) Serve() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue(PathValueName)
		if !attachmentNamePattern.MatchString(name) {
			http.NotFound(w, r)
			return
		}
		file, err := os.Open(filepath.Join(a.dir, name))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer func() { _ = file.Close() }()
		info, err := file.Stat()
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		extension := strings.ToLower(strings.TrimPrefix(path.Ext(name), "."))
		w.Header().Set("Content-Type", attachmentTypes[extension])
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Disposition", disposition(extension, r.URL.Query().Get(downloadNameQuery)))
		http.ServeContent(w, r, name, info.ModTime(), file)
	}
}

func AttachmentURL(storedPath, originalName string) string {
	trimmed := strings.TrimSpace(storedPath)
	if trimmed == "" {
		return ""
	}
	extension := strings.ToLower(strings.TrimPrefix(path.Ext(trimmed), "."))
	if inlineTypes[extension] || !isDownloadName(originalName, extension) {
		return trimmed
	}
	return trimmed + "?" + downloadNameQuery + "=" + url.QueryEscape(originalName)
}

func AttachmentInline(mime string) bool {
	for extension, allowed := range attachmentTypes {
		if allowed == mime {
			return inlineTypes[extension]
		}
	}
	return false
}

func disposition(extension, requested string) string {
	if inlineTypes[extension] {
		return dispositionValue
	}
	if isDownloadName(requested, extension) {
		return `attachment; filename="` + requested + `"`
	}
	return "attachment"
}

func isDownloadName(name, extension string) bool {
	if !downloadNamePattern.MatchString(name) {
		return false
	}
	return strings.EqualFold(strings.TrimPrefix(path.Ext(name), "."), extension)
}

func safeOriginalName(filename, extension string) string {
	name := path.Base(strings.ReplaceAll(strings.TrimSpace(filename), `\`, "/"))
	if isDownloadName(name, extension) {
		return name
	}
	return "attachment." + extension
}

func checkAttachmentContent(file multipart.File, extension string) error {
	head := make([]byte, sniffBytes)
	read, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return ErrBadForm
	}
	detected := contentTypeOf(head[:read])
	if detected == "text/html" {
		return ErrNotAllowedType
	}
	if expected, strict := sniffedTypes[extension]; strict && detected != expected {
		return ErrNotAllowedType
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return ErrBadForm
	}
	return nil
}
