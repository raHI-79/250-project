package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/release"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	MessageAPKBusy    = "The app download is not available right now. Please try again later."
	MessageAPKMissing = "No Android build is published for that version."
	apkFolder         = "apk"
	apkCacheControl   = "public, max-age=600"
)

type AssetSource interface {
	AssetFor(ctx context.Context, version string) (release.Details, error)
	Open(ctx context.Context, assetAPI string) (release.Stream, error)
}

type APKs struct {
	cfg      *config.Config
	source   AssetSource
	dir      string
	fallback func(w http.ResponseWriter)
}

func (a *APKs) WithFallback(page func(w http.ResponseWriter)) *APKs {
	a.fallback = page
	return a
}

func NewAPKs(cfg *config.Config, source AssetSource) *APKs {
	dir := "uploads"
	if cfg != nil && strings.TrimSpace(cfg.UploadDir) != "" {
		dir = cfg.UploadDir
	}
	return &APKs{cfg: cfg, source: source, dir: filepath.Join(dir, apkFolder)}
}

func (a *APKs) Path(version string) string {
	return filepath.Join(a.dir, release.AssetName(version))
}

func (a *APKs) cached(version string) (string, int64, bool) {
	path := a.Path(version)
	info, err := os.Stat(path)
	if err != nil || info.Size() <= 0 {
		return "", 0, false
	}
	return path, info.Size(), true
}

func (a *APKs) store(version string, body io.Reader) (string, int64, error) {
	if err := os.MkdirAll(a.dir, 0o750); err != nil {
		return "", 0, err
	}
	path := a.Path(version)
	temporary, err := os.CreateTemp(a.dir, "apk-*")
	if err != nil {
		return "", 0, err
	}
	written, err := io.Copy(temporary, body)
	closeErr := temporary.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(temporary.Name())
		return "", 0, err
	}
	if err := os.Rename(temporary.Name(), path); err != nil {
		_ = os.Remove(temporary.Name())
		return "", 0, err
	}
	return path, written, nil
}

func (a *APKs) Serve() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wanted := release.Version(strings.TrimSpace(r.PathValue("version")))
		details, err := a.source.AssetFor(r.Context(), wanted)
		switch {
		case errors.Is(err, release.ErrAssetGone), errors.Is(err, release.ErrNoAsset):
			if wanted == "" && a.fallback != nil {
				a.fallback(w)
				return
			}
			a.refuse(w, http.StatusNotFound, MessageAPKMissing)
			return
		case err != nil:
			log.Printf("handler: could not find the APK of %q: %v", wanted, err)
			a.refuse(w, http.StatusServiceUnavailable, MessageAPKBusy)
			return
		}
		version := details.Version
		if path, size, ok := a.cached(version); ok {
			a.stream(w, r, version, path, size)
			return
		}
		stream, err := a.source.Open(r.Context(), details.AssetAPI)
		switch {
		case errors.Is(err, release.ErrNoToken):
			log.Print("handler: the APK cannot be fetched because no GitHub token is set")
			a.refuse(w, http.StatusServiceUnavailable, MessageAPKBusy)
			return
		case errors.Is(err, release.ErrAssetGone):
			a.refuse(w, http.StatusNotFound, MessageAPKMissing)
			return
		case err != nil:
			log.Printf("handler: GitHub refused the APK of %s (status %d): %v",
				version, stream.Status, err)
			a.refuse(w, http.StatusServiceUnavailable, MessageAPKBusy)
			return
		}
		defer func() { _ = stream.Body.Close() }()
		path, size, err := a.store(version, stream.Body)
		if err != nil {
			log.Printf("handler: could not cache the APK of %s: %v", version, err)
			a.refuse(w, http.StatusServiceUnavailable, MessageAPKBusy)
			return
		}
		a.stream(w, r, version, path, size)
	}
}

func (a *APKs) stream(w http.ResponseWriter, r *http.Request, version, path string, size int64) {
	file, err := os.Open(path)
	if err != nil {
		log.Printf("handler: could not open the cached APK of %s: %v", version, err)
		a.refuse(w, http.StatusServiceUnavailable, MessageAPKBusy)
		return
	}
	defer func() { _ = file.Close() }()
	w.Header().Set("Content-Type", release.APKMime)
	w.Header().Set("Content-Disposition", `attachment; filename="`+release.AssetName(version)+`"`)
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Cache-Control", apkCacheControl)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	if _, err := io.Copy(w, file); err != nil {
		log.Printf("handler: could not send the APK of %s: %v", version, err)
	}
}

func (a *APKs) refuse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Cache-Control", noStore)
	httpx.WriteError(w, status, message)
}
