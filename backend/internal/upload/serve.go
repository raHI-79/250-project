package upload

import (
	"bytes"
	"net/http"
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
		blob, err := s.files.Get(r.Context(), KindFile, name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", ContentType)
		w.Header().Set("Content-Disposition", dispositionValue)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeContent(w, r, name, blob.ModTime, bytes.NewReader(blob.Data))
	}
}
