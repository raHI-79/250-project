package handler

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"net/http"
	"time"
)

const (
	databaseOK          = "ok"
	databaseUnavailable = "unavailable"
	databasePingTimeout = 2 * time.Second
)

type DatabasePinger interface {
	Ping(ctx context.Context) error
}

type healthResponse struct {
	Status      string `json:"status"`
	Environment string `json:"environment"`
	Version     string `json:"version"`
	Commit      string `json:"commit"`
	Database    string `json:"database"`
	Time        string `json:"time"`
}

func Health(cfg *config.Config, database DatabasePinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, healthResponse{
			Status:      "ok",
			Environment: cfg.AppEnv,
			Version:     cfg.Version,
			Commit:      cfg.Commit,
			Database:    databaseStatus(r.Context(), database),
			Time:        cfg.Now().Format(time.RFC3339),
		})
	}
}

func databaseStatus(ctx context.Context, database DatabasePinger) string {
	if database == nil {
		return databaseUnavailable
	}
	pingContext, cancel := context.WithTimeout(ctx, databasePingTimeout)
	defer cancel()
	if err := database.Ping(pingContext); err != nil {
		return databaseUnavailable
	}
	return databaseOK
}
