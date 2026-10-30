package handler

import (
	"context"
	"errors"
	"html/template"
	"log"
	"net/http"

	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/release"
)

const (
	DownloadAndroidPath  = "/download/android"
	AndroidLatestPath    = "/api/app/android-latest"
	downloadCacheControl = "public, max-age=600"
	noStore              = "no-store"

	MessageUpdateBusy = "Update service is busy, try again later."
)

type ReleaseLookup interface {
	Latest(ctx context.Context) (release.Details, error)
	ReleasesPage() string
}

type androidLatestResponse struct {
	Version     string `json:"version"`
	VersionCode int    `json:"version_code"`
	APKURL      string `json:"apk_url"`
	Notes       string `json:"notes"`
	Required    bool   `json:"required"`
}

var fallbackPage = template.Must(template.New("fallback").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>The SUST CSE app is not ready yet</title>
<style>
:root { color-scheme: light dark; }
body { margin: 0; display: grid; min-height: 100vh; place-items: center;
  background: #f7f8fa; color: #101828; font: 16px/1.6 "Inter", system-ui, sans-serif; }
main { max-width: 26rem; padding: 2rem 1.5rem; text-align: center; }
h1 { margin: 0 0 0.75rem; font-size: 1.375rem; letter-spacing: -0.01em; }
p { margin: 0 0 1rem; color: #475467; }
a { display: inline-block; margin-top: 0.5rem; border-radius: 0.75rem; background: #1d4ed8;
  padding: 0.75rem 1.25rem; color: #fff; font-weight: 600; text-decoration: none; }
@media (prefers-color-scheme: dark) {
  body { background: #0e1014; color: #f2f4f7; }
  p { color: #98a2b3; }
  a { background: #93b4fb; color: #101828; }
}
</style>
</head>
<body>
<main>
<h1>The app is not ready to download</h1>
<p>No build is published yet. Sign in on this website in the meantime, and try
this link again in a little while.</p>
<a href="{{ .Home }}">Open the website</a>
</main>
</body>
</html>
`))

type AppRelease struct {
	releases ReleaseLookup
	home     string
}

func NewAppRelease(releases ReleaseLookup, home string) *AppRelease {
	return &AppRelease{releases: releases, home: home}
}

func (a *AppRelease) Download() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		details, err := a.releases.Latest(r.Context())
		if err != nil && !errors.Is(err, release.ErrNoAsset) {
			if !errors.Is(err, release.ErrUnavailable) {
				log.Printf("handler: could not find the newest APK: %v", err)
			}
			w.Header().Set("Cache-Control", noStore)
			http.Redirect(w, r, a.releases.ReleasesPage(), http.StatusFound)
			return
		}
		if details.APKURL == "" {
			a.writeFallback(w)
			return
		}
		w.Header().Set("Cache-Control", downloadCacheControl)
		http.Redirect(w, r, details.APKURL, http.StatusFound)
	}
}

func (a *AppRelease) Latest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		details, err := a.releases.Latest(r.Context())
		if err != nil && !errors.Is(err, release.ErrNoAsset) {
			if !errors.Is(err, release.ErrUnavailable) {
				log.Printf("handler: could not read the newest release: %v", err)
			}
			w.Header().Set("Cache-Control", noStore)
			httpx.WriteError(w, http.StatusServiceUnavailable, MessageUpdateBusy)
			return
		}
		w.Header().Set("Cache-Control", downloadCacheControl)
		httpx.WriteJSON(w, http.StatusOK, androidLatestResponse{
			Version:     details.Version,
			VersionCode: details.VersionCode,
			APKURL:      details.APKURL,
			Notes:       details.Notes,
			Required:    details.Required,
		})
	}
}

func (a *AppRelease) writeFallback(w http.ResponseWriter) {
	home := a.home
	if home == "" {
		home = "/"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", noStore)
	w.WriteHeader(http.StatusServiceUnavailable)
	if err := fallbackPage.Execute(w, struct{ Home string }{Home: home}); err != nil {
		log.Printf("handler: could not write the download fallback page: %v", err)
	}
}
