package handler

import (
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"net/http"
	"time"
)

type healthResponse struct {
	Status      string `json:"status"`
	Environment string `json:"environment"`
	Time        string `json:"time"`
}

func Health(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, healthResponse{
			Status:      "ok",
			Environment: cfg.AppEnv,
			Time:        cfg.Now().Format(time.RFC3339),
		})
	}
}
