package upload

import (
	"net/http"
	"os"
	"path/filepath"
)

const (
	PathValueName    = "name"
	dispositionValue = "inline"
)

func (s *Saver) Serve() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue(PathValueName)
		if !namePattern.MatchString(name) {
			http.NotFound(w, r)
			return
		}
		file, err := os.Open(filepath.Join(s.dir, name))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer func() { _ = file.Close() }()
		info, err := file.Stat()
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", ContentType)
		w.Header().Set("Content-Disposition", dispositionValue)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeContent(w, r, name, info.ModTime(), file)
	}
}
