package upload

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	_ "golang.org/x/image/webp"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"regexp"
	"strings"
)

const (
	PhotoFieldName   = "photo"
	PhotoDirName     = "photos"
	PhotoURLPrefix   = "/uploads/photos/"
	MaxPhotoBytes    = 2 << 20
	MinPhotoSide     = 200
	photoSniffBytes  = 512
	photoParseMemory = 1 << 20
)

var (
	ErrMissingPhoto  = errors.New("upload: a photo is required")
	ErrNotPhoto      = errors.New("upload: only JPG, PNG and WEBP photos are accepted")
	ErrPhotoTooLarge = errors.New("upload: the photo is larger than the allowed size")
	ErrPhotoTooSmall = errors.New("upload: the photo is smaller than the allowed size in pixels")

	photoNamePattern = regexp.MustCompile(`^[a-f0-9]{32}\.(jpg|png|webp)$`)
	photoTypes       = map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/webp": ".webp",
	}
)

type Photos struct {
	dir   string
	files Store
}

func NewPhotos(uploadDir string) *Photos {
	return NewPhotosIn(NewDisk(uploadDir))
}

func NewPhotosIn(files Store) *Photos {
	photos := &Photos{files: files}
	if disk, ok := files.(*Disk); ok {
		photos.dir = disk.Dir(KindPhoto)
	}
	return photos
}

func (p *Photos) Dir() string {
	return p.dir
}

func (p *Photos) Parse(r *http.Request) error {
	if r.MultipartForm != nil {
		return nil
	}
	r.Body = http.MaxBytesReader(nil, r.Body, MaxPhotoBytes+formOverhead)
	if err := r.ParseMultipartForm(photoParseMemory); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return ErrPhotoTooLarge
		}
		return ErrBadForm
	}
	return nil
}

func (p *Photos) Save(r *http.Request) (string, error) {
	if err := p.Parse(r); err != nil {
		return "", err
	}
	file, header, err := r.FormFile(PhotoFieldName)
	if errors.Is(err, http.ErrMissingFile) {
		return "", nil
	}
	if err != nil {
		return "", ErrBadForm
	}
	defer func() { _ = file.Close() }()
	if header.Size > MaxPhotoBytes {
		return "", ErrPhotoTooLarge
	}
	extension, err := photoExtension(file)
	if err != nil {
		return "", err
	}
	if err := checkPhotoSize(file); err != nil {
		return "", err
	}
	return p.store(r.Context(), file, extension)
}

func (p *Photos) store(ctx context.Context, file multipart.File, extension string) (string, error) {
	data, tooLarge, err := readLimited(file, MaxPhotoBytes)
	if err != nil {
		return "", fmt.Errorf("upload: could not read the photo: %w", err)
	}
	if tooLarge {
		return "", ErrPhotoTooLarge
	}
	name, err := randomPhotoName(extension)
	if err != nil {
		return "", err
	}
	if err := p.files.Put(ctx, KindPhoto, name, photoContentType(name), data); err != nil {
		return "", err
	}
	return PhotoURLPrefix + name, nil
}

func (p *Photos) Delete(photoPath string) error {
	name := path.Base(strings.TrimSpace(photoPath))
	if !photoNamePattern.MatchString(name) {
		return nil
	}
	return p.files.Delete(context.Background(), KindPhoto, name)
}

func (p *Photos) Serve() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue(PathValueName)
		if !photoNamePattern.MatchString(name) {
			http.NotFound(w, r)
			return
		}
		blob, err := p.files.Get(r.Context(), KindPhoto, name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", photoContentType(name))
		w.Header().Set("Content-Disposition", dispositionValue)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeContent(w, r, name, blob.ModTime, bytes.NewReader(blob.Data))
	}
}

func photoContentType(name string) string {
	for contentType, extension := range photoTypes {
		if strings.HasSuffix(name, extension) {
			return contentType
		}
	}
	return "application/octet-stream"
}

func photoExtension(file multipart.File) (string, error) {
	head := make([]byte, photoSniffBytes)
	read, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return "", ErrBadForm
	}
	extension, ok := photoTypes[strings.ToLower(contentTypeOf(head[:read]))]
	if !ok {
		return "", ErrNotPhoto
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", ErrBadForm
	}
	return extension, nil
}

func contentTypeOf(head []byte) string {
	detected := http.DetectContentType(head)
	if semicolon := strings.Index(detected, ";"); semicolon >= 0 {
		detected = detected[:semicolon]
	}
	return strings.TrimSpace(detected)
}

func checkPhotoSize(file multipart.File) error {
	settings, _, err := image.DecodeConfig(file)
	if err != nil {
		return ErrNotPhoto
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return ErrBadForm
	}
	if settings.Width < MinPhotoSide || settings.Height < MinPhotoSide {
		return ErrPhotoTooSmall
	}
	return nil
}

func randomPhotoName(extension string) (string, error) {
	name, err := randomName()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(name, Extension) + extension, nil
}
