package upload

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
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
	defaultDir    = "uploads"
)

var (
	ErrBadForm  = errors.New("upload: the request is not a valid multipart form")
	ErrNotPDF   = errors.New("upload: only PDF files are accepted")
	ErrTooLarge = errors.New("upload: the file is larger than the allowed size")

	namePattern = regexp.MustCompile(`^[a-f0-9]{32}\.pdf$`)
)

type Saver struct {
	dir      string
	files    Store
	maxBytes int64
}

func NewSaver(dir string, maxMB int) *Saver {
	if dir == "" {
		dir = defaultDir
	}
	return NewSaverIn(NewDisk(dir), maxMB)
}

func NewSaverIn(files Store, maxMB int) *Saver {
	if maxMB <= 0 {
		maxMB = 1
	}
	saver := &Saver{files: files, maxBytes: int64(maxMB) << 20}
	if disk, ok := files.(*Disk); ok {
		saver.dir = disk.Dir(KindFile)
	}
	return saver
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
	return s.store(r.Context(), file)
}

func (s *Saver) store(ctx context.Context, file multipart.File) (string, error) {
	data, tooLarge, err := readLimited(file, s.maxBytes)
	if err != nil {
		return "", fmt.Errorf("upload: could not read the file: %w", err)
	}
	if tooLarge {
		return "", ErrTooLarge
	}
	name, err := randomName()
	if err != nil {
		return "", err
	}
	if err := s.files.Put(ctx, KindFile, name, ContentType, data); err != nil {
		return "", err
	}
	return URLPrefix + name, nil
}

func (s *Saver) Delete(filePath string) error {
	name := path.Base(strings.TrimSpace(filePath))
	if !namePattern.MatchString(name) {
		return nil
	}
	return s.files.Delete(context.Background(), KindFile, name)
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
