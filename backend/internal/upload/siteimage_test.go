package upload

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func picture(width, height int, transparent bool) image.Image {
	canvas := image.NewNRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			alpha := uint8(255)
			if transparent && x < width/2 {
				alpha = 0
			}
			canvas.Set(x, y, color.NRGBA{R: 10, G: 111, B: 175, A: alpha})
		}
	}
	return canvas
}

func TestSiteImagesAreCheckedAndEncodedAgain(t *testing.T) {
	var photo bytes.Buffer
	if err := jpeg.Encode(&photo, picture(40, 30, false), nil); err != nil {
		t.Fatalf("could not make a JPG: %v", err)
	}
	trailing := append(append([]byte{}, photo.Bytes()...), []byte("<script>alert(1)</script>")...)
	encoded, extension, err := ReencodeImage(trailing)
	if err != nil {
		t.Fatalf("a JPG was refused: %v", err)
	}
	if extension != ".jpg" || bytes.Contains(encoded, []byte("<script>")) {
		t.Fatalf("the image was stored as it came, with what was appended to it (%s)", extension)
	}
	if _, _, err := image.Decode(bytes.NewReader(encoded)); err != nil {
		t.Fatalf("the encoded image cannot be read back: %v", err)
	}
	var logo bytes.Buffer
	if err := png.Encode(&logo, picture(40, 30, true)); err != nil {
		t.Fatalf("could not make a PNG: %v", err)
	}
	if _, extension, err := ReencodeImage(logo.Bytes()); err != nil || extension != ".png" {
		t.Fatalf("a transparent PNG became %q, %v, want .png", extension, err)
	}
	if _, _, err := ReencodeImage([]byte("<svg onload=alert(1)></svg>")); !errors.Is(err, ErrNotImage) {
		t.Errorf("an SVG returned %v, want ErrNotImage", err)
	}
	if _, _, err := ReencodeImage(nil); !errors.Is(err, ErrMissingImage) {
		t.Errorf("no image returned %v, want ErrMissingImage", err)
	}
	if _, _, err := ReencodeImage(make([]byte, MaxSiteImageBytes+1)); !errors.Is(err, ErrImageTooLarge) {
		t.Errorf("a large file returned %v, want ErrImageTooLarge", err)
	}
	var wide bytes.Buffer
	if err := png.Encode(&wide, image.NewGray(image.Rect(0, 0, MaxSiteImageSide+1, 1))); err != nil {
		t.Fatalf("could not make a wide image: %v", err)
	}
	if _, _, err := ReencodeImage(wide.Bytes()); !errors.Is(err, ErrImageTooWide) {
		t.Errorf("a very wide image returned %v, want ErrImageTooWide", err)
	}
}
