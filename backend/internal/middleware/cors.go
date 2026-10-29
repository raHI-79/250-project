package middleware

import "net/http"

const (
	allowedMethods  = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	allowedHeaders  = "Authorization, Content-Type, X-Human-Pass, X-Client"
	preflightMaxAge = "600"
	CapacitorHTTPS  = "https://localhost"
	CapacitorScheme = "capacitor://localhost"
)

func NativeOrigin(origin string) bool {
	return origin == CapacitorHTTPS || origin == CapacitorScheme
}

func CORS(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Vary", "Origin")
			origin := r.Header.Get("Origin")
			if origin != "" && (origin == allowedOrigin || NativeOrigin(origin)) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
				w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
				w.Header().Set("Access-Control-Max-Age", preflightMaxAge)
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
