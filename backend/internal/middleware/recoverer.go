package middleware

import (
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"log"
	"net/http"
	"runtime/debug"
)

func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recording := &recorder{ResponseWriter: w}
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			log.Printf("middleware: panic while handling %s %s: %v\n%s", r.Method, r.URL.Path, recovered, debug.Stack())
			if recording.status == 0 {
				httpx.WriteError(recording, http.StatusInternalServerError, MessageServerError)
			}
		}()
		next.ServeHTTP(recording, r)
	})
}
