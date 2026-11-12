package upload

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	_ "golang.org/x/image/webp"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
)

const (
	SiteImageField    = "image"
	MaxSiteImageBytes = 4 << 20
	MaxSiteImageSide  = 6000
	siteImageQuality  = 86
)

var (
	ErrMissingImage  = errors.New("upload: an image is required")
	ErrNotImage      = errors.New("upload: only JPG, PNG and WEBP images are accepted")
	ErrImageTooLarge = errors.New("upload: the image is larger than 4 MB")
	ErrImageTooWide  = errors.New("upload: the image is wider or taller than 6000 pixels")
)

func opaque(picture image.Image) bool {
	type opaquer interface{ Opaque() bool }
	if checker, ok := picture.(opaquer); ok {
		return checker.Opaque()
	}
	return true
}

func ReencodeImage(data []byte) ([]byte, string, error) {
	if len(data) == 0 {
		return nil, "", ErrMissingImage
	}
	if len(data) > MaxSiteImageBytes {
		return nil, "", ErrImageTooLarge
	}
	settings, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", ErrNotImage
	}
	if settings.Width <= 0 || settings.Height <= 0 ||
		settings.Width > MaxSiteImageSide || settings.Height > MaxSiteImageSide {
		return nil, "", ErrImageTooWide
	}
	picture, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", ErrNotImage
	}
	var out bytes.Buffer
	if opaque(picture) {
		if err := jpeg.Encode(&out, picture, &jpeg.Options{Quality: siteImageQuality}); err != nil {
			return nil, "", fmt.Errorf("upload: could not write the image: %w", err)
		}
		return out.Bytes(), ".jpg", nil
	}
	if err := png.Encode(&out, picture); err != nil {
		return nil, "", fmt.Errorf("upload: could not write the image: %w", err)
	}
	return out.Bytes(), ".png", nil
}

func (p *Photos) SaveSiteImage(r *http.Request) (string, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, MaxSiteImageBytes+(1<<20))
	if err := r.ParseMultipartForm(photoParseMemory); err != nil {
		return "", ErrImageTooLarge
	}
	file, header, err := r.FormFile(SiteImageField)
	if errors.Is(err, http.ErrMissingFile) {
		return "", ErrMissingImage
	}
	if err != nil {
		return "", ErrBadForm
	}
	defer func() { _ = file.Close() }()
	if header.Size > MaxSiteImageBytes {
		return "", ErrImageTooLarge
	}
	data, tooLarge, err := readLimited(file, MaxSiteImageBytes)
	if err != nil {
		return "", fmt.Errorf("upload: could not read the image: %w", err)
	}
	if tooLarge {
		return "", ErrImageTooLarge
	}
	return p.PutSiteImage(r.Context(), data)
}

func (p *Photos) PutSiteImage(ctx context.Context, data []byte) (string, error) {
	encoded, extension, err := ReencodeImage(data)
	if err != nil {
		return "", err
	}
	name, err := randomPhotoName(extension)
	if err != nil {
		return "", err
	}
	if err := p.files.Put(ctx, KindPhoto, name, photoContentType(name), encoded); err != nil {
		return "", err
	}
	return PhotoURLPrefix + name, nil
}
