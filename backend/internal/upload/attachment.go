package upload

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
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
	files    Store
	maxBytes int64
}

func NewAttachments(uploadDir string, maxMB int) *Attachments {
	return NewAttachmentsIn(NewDisk(uploadDir), maxMB)
}

func NewAttachmentsIn(files Store, maxMB int) *Attachments {
	if maxMB <= 0 {
		maxMB = DefaultAttachmentMB
	}
	attachments := &Attachments{files: files, maxBytes: int64(maxMB) << 20}
	if disk, ok := files.(*Disk); ok {
		attachments.dir = disk.Dir(KindAttachment)
	}
	return attachments
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
	stored, size, err := a.store(r.Context(), file, extension)
	if err != nil {
		return Attachment{}, err
	}
	return Attachment{Path: stored, Name: safeOriginalName(header.Filename, extension), Mime: mime, Size: size}, nil
}

func (a *Attachments) store(ctx context.Context, file multipart.File,
	extension string) (string, int64, error) {
	data, tooLarge, err := readLimited(file, a.maxBytes)
	if err != nil {
		return "", 0, fmt.Errorf("upload: could not read the attachment: %w", err)
	}
	if tooLarge {
		return "", 0, ErrTooLarge
	}
	name, err := randomName()
	if err != nil {
		return "", 0, err
	}
	name = strings.TrimSuffix(name, Extension) + "." + extension
	if err := a.files.Put(ctx, KindAttachment, name, attachmentTypes[extension], data); err != nil {
		return "", 0, err
	}
	return AttachmentURLPrefix + name, int64(len(data)), nil
}

func (a *Attachments) Delete(storedPath string) error {
	name := path.Base(strings.TrimSpace(storedPath))
	if !attachmentNamePattern.MatchString(name) {
		return nil
	}
	return a.files.Delete(context.Background(), KindAttachment, name)
}

func (a *Attachments) Serve() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue(PathValueName)
		if !attachmentNamePattern.MatchString(name) {
			http.NotFound(w, r)
			return
		}
		blob, err := a.files.Get(r.Context(), KindAttachment, name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		extension := strings.ToLower(strings.TrimPrefix(path.Ext(name), "."))
		w.Header().Set("Content-Type", attachmentTypes[extension])
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Disposition", disposition(extension, r.URL.Query().Get(downloadNameQuery)))
		http.ServeContent(w, r, name, blob.ModTime, bytes.NewReader(blob.Data))
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
