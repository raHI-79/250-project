package upload

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func solid(width, height int) image.Image {
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			canvas.Set(x, y, color.RGBA{R: uint8(x % 253), G: uint8(y % 239), B: 70, A: 255})
		}
	}
	return canvas
}

func pngBytes(t *testing.T, width, height int) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, solid(width, height)); err != nil {
		t.Fatalf("could not encode a png: %v", err)
	}
	return buffer.Bytes()
}

type bitWriter struct {
	bytes []byte
	value uint32
	used  uint
}

func (w *bitWriter) put(value uint32, count uint) {
	for i := uint(0); i < count; i++ {
		w.value |= ((value >> i) & 1) << w.used
		w.used++
		if w.used == 8 {
			w.bytes = append(w.bytes, byte(w.value))
			w.value, w.used = 0, 0
		}
	}
}

func (w *bitWriter) flush() []byte {
	if w.used > 0 {
		w.bytes = append(w.bytes, byte(w.value))
		w.value, w.used = 0, 0
	}
	return w.bytes
}

func (w *bitWriter) singleSymbolTree(symbol uint32, wide bool) {
	w.put(1, 1)
	w.put(0, 1)
	if wide {
		w.put(1, 1)
		w.put(symbol, 8)
		return
	}
	w.put(0, 1)
	w.put(symbol&1, 1)
}

func webpBytes(t *testing.T, width, height int) []byte {
	t.Helper()
	bits := &bitWriter{}
	bits.put(0x2f, 8)
	bits.put(uint32(width-1), 14)
	bits.put(uint32(height-1), 14)
	bits.put(0, 1)
	bits.put(0, 3)
	bits.put(0, 1)
	bits.put(0, 1)
	bits.put(0, 1)
	bits.singleSymbolTree(90, true)
	bits.singleSymbolTree(160, true)
	bits.singleSymbolTree(200, true)
	bits.singleSymbolTree(255, true)
	bits.singleSymbolTree(0, false)
	payload := bits.flush()
	if len(payload)%2 == 1 {
		payload = append(payload, 0)
	}
	var file bytes.Buffer
	file.WriteString("RIFF")
	if err := binary.Write(&file, binary.LittleEndian, uint32(4+8+len(payload))); err != nil {
		t.Fatalf("could not write the webp size: %v", err)
	}
	file.WriteString("WEBPVP8L")
	if err := binary.Write(&file, binary.LittleEndian, uint32(len(payload))); err != nil {
		t.Fatalf("could not write the webp chunk size: %v", err)
	}
	file.Write(payload)
	return file.Bytes()
}

func jpegBytes(t *testing.T, width, height int) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, solid(width, height), &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("could not encode a jpeg: %v", err)
	}
	return buffer.Bytes()
}

func photoRequest(t *testing.T, name string, data []byte, extraFields map[string]string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for field, value := range extraFields {
		if err := writer.WriteField(field, value); err != nil {
			t.Fatalf("could not write the field %q: %v", field, err)
		}
	}
	if name != "" {
		part, err := writer.CreateFormFile(PhotoFieldName, name)
		if err != nil {
			t.Fatalf("could not create the photo part: %v", err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatalf("could not write the photo part: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("could not close the multipart writer: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func TestPhotosSave(t *testing.T) {
	tests := []struct {
		name          string
		fileName      string
		data          func(t *testing.T) []byte
		skipFile      bool
		wantExtension string
		wantErr       error
	}{
		{
			name:          "png",
			fileName:      "student.png",
			data:          func(t *testing.T) []byte { return pngBytes(t, 240, 240) },
			wantExtension: ".png",
		},
		{
			name:          "jpeg",
			fileName:      "student.jpg",
			data:          func(t *testing.T) []byte { return jpegBytes(t, 400, 300) },
			wantExtension: ".jpg",
		},
		{
			name:          "the content decides, not the name",
			fileName:      "student.gif",
			data:          func(t *testing.T) []byte { return pngBytes(t, 220, 220) },
			wantExtension: ".png",
		},
		{
			name:          "exactly the smallest allowed size",
			fileName:      "student.png",
			data:          func(t *testing.T) []byte { return pngBytes(t, MinPhotoSide, MinPhotoSide) },
			wantExtension: ".png",
		},
		{
			name:          "webp",
			fileName:      "student.webp",
			data:          func(t *testing.T) []byte { return webpBytes(t, 220, 220) },
			wantExtension: ".webp",
		},
		{
			name:     "webp below the smallest size",
			fileName: "student.webp",
			data:     func(t *testing.T) []byte { return webpBytes(t, 199, 220) },
			wantErr:  ErrPhotoTooSmall,
		},
		{
			name:     "no photo at all",
			skipFile: true,
		},
		{
			name:     "a pdf renamed to png",
			fileName: "student.png",
			data:     func(t *testing.T) []byte { return []byte("%PDF-1.7 not a photo at all") },
			wantErr:  ErrNotPhoto,
		},
		{
			name:     "plain text",
			fileName: "student.png",
			data:     func(t *testing.T) []byte { return []byte(strings.Repeat("hello", 200)) },
			wantErr:  ErrNotPhoto,
		},
		{
			name:     "too narrow",
			fileName: "student.png",
			data:     func(t *testing.T) []byte { return pngBytes(t, MinPhotoSide-1, 400) },
			wantErr:  ErrPhotoTooSmall,
		},
		{
			name:     "too short",
			fileName: "student.png",
			data:     func(t *testing.T) []byte { return pngBytes(t, 400, MinPhotoSide-1) },
			wantErr:  ErrPhotoTooSmall,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			photos := NewPhotos(t.TempDir())
			var data []byte
			name := tc.fileName
			if tc.skipFile {
				name = ""
			} else {
				data = tc.data(t)
			}
			path, err := photos.Save(photoRequest(t, name, data, nil))
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("Save() error = %v, want %v", err, tc.wantErr)
				}
				if path != "" {
					t.Errorf("Save() returned %q alongside the error, want an empty path", path)
				}
				if stored := storedPhotos(t, photos); stored != 0 {
					t.Errorf("%d photos stayed on disk, want none", stored)
				}
				return
			}
			if err != nil {
				t.Fatalf("Save() returned an unexpected error: %v", err)
			}
			if tc.skipFile {
				if path != "" {
					t.Errorf("Save() returned %q for a request without a photo, want an empty path", path)
				}
				return
			}
			if !strings.HasPrefix(path, PhotoURLPrefix) {
				t.Errorf("Save() returned %q, want a path under %q", path, PhotoURLPrefix)
			}
			if !strings.HasSuffix(path, tc.wantExtension) {
				t.Errorf("Save() returned %q, want the %q extension", path, tc.wantExtension)
			}
			if !photoNamePattern.MatchString(filepath.Base(path)) {
				t.Errorf("the stored name %q does not match the strict pattern", filepath.Base(path))
			}
			if strings.Contains(path, tc.fileName) {
				t.Errorf("Save() kept the submitted file name in %q", path)
			}
			if stored := storedPhotos(t, photos); stored != 1 {
				t.Errorf("%d photos are on disk, want 1", stored)
			}
		})
	}
}

func TestPhotosRejectLargeFiles(t *testing.T) {
	photos := NewPhotos(t.TempDir())
	oversized := make([]byte, MaxPhotoBytes+formOverhead+1)
	copy(oversized, pngBytes(t, 220, 220))
	if _, err := photos.Save(photoRequest(t, "student.png", oversized, nil)); !errors.Is(err, ErrPhotoTooLarge) {
		t.Fatalf("Save() error = %v, want ErrPhotoTooLarge", err)
	}
	if stored := storedPhotos(t, photos); stored != 0 {
		t.Errorf("%d photos stayed on disk, want none", stored)
	}
}

func TestPhotosKeepTheOtherFields(t *testing.T) {
	photos := NewPhotos(t.TempDir())
	request := photoRequest(t, "student.png", pngBytes(t, 220, 220), map[string]string{"reg_no": "2023331079"})
	if _, err := photos.Save(request); err != nil {
		t.Fatalf("Save() returned an unexpected error: %v", err)
	}
	if got := request.FormValue("reg_no"); got != "2023331079" {
		t.Errorf("reg_no = %q, want 2023331079", got)
	}
}

func TestPhotosDelete(t *testing.T) {
	photos := NewPhotos(t.TempDir())
	path, err := photos.Save(photoRequest(t, "student.png", pngBytes(t, 220, 220), nil))
	if err != nil {
		t.Fatalf("Save() returned an unexpected error: %v", err)
	}
	tests := []struct {
		name       string
		path       string
		wantStored int
	}{
		{name: "another name", path: PhotoURLPrefix + "0123456789abcdef0123456789abcdef.png", wantStored: 1},
		{name: "a path outside the pattern", path: "../../etc/passwd", wantStored: 1},
		{name: "an empty path", path: "", wantStored: 1},
		{name: "the saved photo", path: path, wantStored: 0},
		{name: "the saved photo again", path: path, wantStored: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := photos.Delete(tc.path); err != nil {
				t.Fatalf("Delete() returned an unexpected error: %v", err)
			}
			if stored := storedPhotos(t, photos); stored != tc.wantStored {
				t.Errorf("%d photos are on disk, want %d", stored, tc.wantStored)
			}
		})
	}
}

func TestPhotosServe(t *testing.T) {
	photos := NewPhotos(t.TempDir())
	path, err := photos.Save(photoRequest(t, "student.png", pngBytes(t, 220, 220), nil))
	if err != nil {
		t.Fatalf("Save() returned an unexpected error: %v", err)
	}
	name := filepath.Base(path)
	tests := []struct {
		name        string
		pathValue   string
		wantStatus  int
		wantType    string
		wantNoSniff bool
	}{
		{
			name:        "the stored photo",
			pathValue:   name,
			wantStatus:  http.StatusOK,
			wantType:    "image/png",
			wantNoSniff: true,
		},
		{name: "another name", pathValue: "0123456789abcdef0123456789abcdef.png", wantStatus: http.StatusNotFound},
		{name: "a traversal attempt", pathValue: "../" + name, wantStatus: http.StatusNotFound},
		{name: "a wrong extension", pathValue: strings.TrimSuffix(name, ".png") + ".pdf", wantStatus: http.StatusNotFound},
		{name: "a short name", pathValue: "abc.png", wantStatus: http.StatusNotFound},
		{name: "an empty name", pathValue: "", wantStatus: http.StatusNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, PhotoURLPrefix+tc.pathValue, nil)
			request.SetPathValue(PathValueName, tc.pathValue)
			rec := httptest.NewRecorder()
			photos.Serve()(rec, request)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantStatus != http.StatusOK {
				return
			}
			if got := rec.Header().Get("Content-Type"); got != tc.wantType {
				t.Errorf("Content-Type = %q, want %q", got, tc.wantType)
			}
			if tc.wantNoSniff && rec.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Error("the photo is served without the nosniff header")
			}
		})
	}
}

func storedPhotos(t *testing.T, photos *Photos) int {
	t.Helper()
	entries, err := os.ReadDir(photos.Dir())
	if errors.Is(err, os.ErrNotExist) {
		return 0
	}
	if err != nil {
		t.Fatalf("could not read the photo folder: %v", err)
	}
	return len(entries)
}
