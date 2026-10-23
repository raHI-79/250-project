package web

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	IndexFile     = "index.html"
	AssetsPrefix  = "/assets/"
	APIPrefix     = "/api/"
	UploadsPrefix = "/uploads/"
	WorkerPath    = "/sw.js"
	ManifestPath  = "/manifest.webmanifest"
	WorkerType    = "application/javascript; charset=utf-8"
	ManifestType  = "application/manifest+json; charset=utf-8"
	WorkerScope   = "/"
	assetCache    = "public, max-age=31536000, immutable"
	pageCache     = "no-cache"
)

func Static(dir string, fallback http.Handler) http.Handler {
	root := http.Dir(dir)
	index := filepath.Join(dir, IndexFile)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, APIPrefix) || strings.HasPrefix(r.URL.Path, UploadsPrefix) {
			fallback.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			fallback.ServeHTTP(w, r)
			return
		}
		secure(w)
		name := path.Clean("/" + r.URL.Path)
		if file, info, ok := openFile(root, name); ok {
			defer func() { _ = file.Close() }()
			switch {
			case name == WorkerPath:
				w.Header().Set("Content-Type", WorkerType)
				w.Header().Set("Service-Worker-Allowed", WorkerScope)
				w.Header().Set("Cache-Control", pageCache)
			case name == ManifestPath:
				w.Header().Set("Content-Type", ManifestType)
				w.Header().Set("Cache-Control", pageCache)
			case strings.HasPrefix(name, AssetsPrefix):
				w.Header().Set("Cache-Control", assetCache)
			default:
				w.Header().Set("Cache-Control", pageCache)
			}
			http.ServeContent(w, r, info.Name(), info.ModTime(), file)
			return
		}
		w.Header().Set("Cache-Control", pageCache)
		http.ServeFile(w, r, index)
	})
}

func openFile(root http.Dir, name string) (http.File, os.FileInfo, bool) {
	if name == "/" || name == "/"+IndexFile {
		return nil, nil, false
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, nil, false
	}
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		_ = file.Close()
		return nil, nil, false
	}
	return file, info, true
}

func secure(w http.ResponseWriter) {
	header := w.Header()
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("X-Frame-Options", "DENY")
	header.Set("Referrer-Policy", "no-referrer")
}
