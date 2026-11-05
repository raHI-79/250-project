package handler

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	MessageViewsStored  = "Thanks."
	MessageBackupStored = "Backup time recorded."
	MessageBackupDenied = "Unknown backup secret."
	MessageViewsTooMany = "Too many page views in one beacon."
	BackupSecretHeader  = "X-Backup-Secret"
	maxViewPath         = 200
	maxViewID           = 64
	maxViewVersion      = 24
	maxViewDevice       = 24
)

type AnalyticsStore interface {
	RecordViews(ctx context.Context, views []store.PageView) error
	SetBackupStatus(ctx context.Context, at time.Time, note string) error
}

type Analytics struct {
	cfg   *config.Config
	views AnalyticsStore
}

func NewAnalytics(cfg *config.Config, views AnalyticsStore) *Analytics {
	return &Analytics{cfg: cfg, views: views}
}

type viewItem struct {
	Path       string `json:"path"`
	SessionID  string `json:"session_id"`
	Device     string `json:"device"`
	AppVersion string `json:"app_version"`
}

type viewsRequest struct {
	VisitorID string     `json:"visitor_id"`
	SessionID string     `json:"session_id"`
	Client    string     `json:"client"`
	Device    string     `json:"device"`
	Version   string     `json:"app_version"`
	Items     []viewItem `json:"items"`
}

type backupRequest struct {
	Note string `json:"note"`
}

func clipped(value string, limit int) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) > limit {
		return trimmed[:limit]
	}
	return trimmed
}

func (a *Analytics) View() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request viewsRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		visitor := clipped(request.VisitorID, maxViewID)
		if visitor == "" {
			httpx.WriteError(w, http.StatusBadRequest, "A visitor id is required.")
			return
		}
		if len(request.Items) > store.MaxViewBatch {
			httpx.WriteError(w, http.StatusBadRequest, MessageViewsTooMany)
			return
		}
		client := RequestClient(r)
		if strings.EqualFold(strings.TrimSpace(request.Client), store.ClientAndroid) {
			client = store.ClientAndroid
		}
		hashed := HashAddress(a.cfg.AnalyticsSalt, RequestAddress(r))
		views := make([]store.PageView, 0, len(request.Items))
		for _, item := range request.Items {
			path := clipped(item.Path, maxViewPath)
			if path == "" {
				continue
			}
			session := clipped(item.SessionID, maxViewID)
			if session == "" {
				session = clipped(request.SessionID, maxViewID)
			}
			device := clipped(item.Device, maxViewDevice)
			if device == "" {
				device = clipped(request.Device, maxViewDevice)
			}
			version := clipped(item.AppVersion, maxViewVersion)
			if version == "" {
				version = clipped(request.Version, maxViewVersion)
			}
			views = append(views, store.PageView{
				VisitorID:  visitor,
				SessionID:  session,
				Path:       path,
				Client:     client,
				Device:     device,
				AppVersion: version,
				IPHash:     hashed,
			})
		}
		if len(views) == 0 {
			httpx.WriteJSON(w, http.StatusAccepted, messageResponse{Message: MessageViewsStored})
			return
		}
		if err := a.views.RecordViews(r.Context(), views); err != nil {
			log.Printf("handler: could not record page views: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusAccepted, messageResponse{Message: MessageViewsStored})
	}
}

func (a *Analytics) BackupStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secret := strings.TrimSpace(a.cfg.BackupSecret)
		sent := strings.TrimSpace(r.Header.Get(BackupSecretHeader))
		if secret == "" || sent == "" || sent != secret {
			httpx.WriteError(w, http.StatusForbidden, MessageBackupDenied)
			return
		}
		var request backupRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		if err := a.views.SetBackupStatus(r.Context(), a.cfg.Now(), clipped(request.Note, maxViewPath)); err != nil {
			log.Printf("handler: could not record the backup status: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageBackupStored})
	}
}
