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
			if strings.HasPrefix(name, AssetsPrefix) {
				w.Header().Set("Cache-Control", assetCache)
			} else {
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
