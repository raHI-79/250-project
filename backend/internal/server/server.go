package server

import (
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"net/http"
)

func NewRouter(cfg *config.Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", handler.Health(cfg))
	mux.HandleFunc("/", notFound)
	return mux
}

func notFound(w http.ResponseWriter, r *http.Request) {
	httpx.WriteError(w, http.StatusNotFound, "Route not found")
}
