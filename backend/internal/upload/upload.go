package upload

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	FieldName     = "file"
	Extension     = ".pdf"
	ContentType   = "application/pdf"
	URLPrefix     = "/uploads/"
	signature     = "%PDF-"
	nameBytes     = 16
	formOverhead  = 1 << 16
	parseMemory   = 1 << 20
	directoryMode = 0o755
	fileMode      = 0o644
)

var (
	ErrBadForm  = errors.New("upload: the request is not a valid multipart form")
	ErrNotPDF   = errors.New("upload: only PDF files are accepted")
	ErrTooLarge = errors.New("upload: the file is larger than the allowed size")

	namePattern = regexp.MustCompile(`^[a-f0-9]{32}\.pdf$`)
)

type Saver struct {
	dir      string
	maxBytes int64
}

func NewSaver(dir string, maxMB int) *Saver {
	if dir == "" {
		dir = "uploads"
	}
	if maxMB <= 0 {
		maxMB = 1
	}
	return &Saver{dir: dir, maxBytes: int64(maxMB) << 20}
}

func (s *Saver) Dir() string {
	return s.dir
}

func (s *Saver) MaxBytes() int64 {
	return s.maxBytes
}

func (s *Saver) Parse(r *http.Request) error {
	if r.MultipartForm != nil {
		return nil
	}
	r.Body = http.MaxBytesReader(nil, r.Body, s.maxBytes+formOverhead)
	if err := r.ParseMultipartForm(parseMemory); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return ErrTooLarge
		}
		return ErrBadForm
	}
	return nil
}

func (s *Saver) Save(r *http.Request) (string, error) {
	if err := s.Parse(r); err != nil {
		return "", err
	}
	file, header, err := r.FormFile(FieldName)
	if errors.Is(err, http.ErrMissingFile) {
		return "", nil
	}
	if err != nil {
		return "", ErrBadForm
	}
	defer func() { _ = file.Close() }()
	if header.Size > s.maxBytes {
		return "", ErrTooLarge
	}
	if !strings.EqualFold(path.Ext(header.Filename), Extension) {
		return "", ErrNotPDF
	}
	if err := checkSignature(file); err != nil {
		return "", err
	}
	return s.store(file)
}

func (s *Saver) store(file multipart.File) (string, error) {
	if err := os.MkdirAll(s.dir, directoryMode); err != nil {
		return "", fmt.Errorf("upload: could not create the upload folder: %w", err)
	}
	name, err := randomName()
	if err != nil {
		return "", err
	}
	target := filepath.Join(s.dir, name)
	destination, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fileMode)
	if err != nil {
		return "", fmt.Errorf("upload: could not create the file: %w", err)
	}
	written, err := io.Copy(destination, io.LimitReader(file, s.maxBytes+1))
	if closeErr := destination.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(target)
		return "", fmt.Errorf("upload: could not write the file: %w", err)
	}
	if written > s.maxBytes {
		_ = os.Remove(target)
		return "", ErrTooLarge
	}
	return URLPrefix + name, nil
}

func (s *Saver) Delete(filePath string) error {
	name := path.Base(strings.TrimSpace(filePath))
	if !namePattern.MatchString(name) {
		return nil
	}
	if err := os.Remove(filepath.Join(s.dir, name)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("upload: could not delete the file %s: %w", name, err)
	}
	return nil
}

func checkSignature(file multipart.File) error {
	head := make([]byte, len(signature))
	read, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return ErrBadForm
	}
	if read < len(signature) || string(head) != signature {
		return ErrNotPDF
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return ErrBadForm
	}
	return nil
}

func randomName() (string, error) {
	raw := make([]byte, nameBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("upload: could not generate a file name: %w", err)
	}
	return hex.EncodeToString(raw) + Extension, nil
}
