package handler

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	MessageInvalidNotificationLimit = "limit must be a number between 1 and 100."
	MessageInvalidSince             = "since must be a timestamp like 2026-10-16T09:30:00+06:00."
)

type NotificationService interface {
	Recent(ctx context.Context, userID int64, limit int) ([]store.Notification, error)
	UnreadCount(ctx context.Context, userID int64, since time.Time) (int, error)
}

type Notifications struct {
	service NotificationService
}

func NewNotifications(service NotificationService) *Notifications {
	return &Notifications{service: service}
}

type notificationItem struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

type notificationsResponse struct {
	Items []notificationItem `json:"items"`
}

type unreadCountResponse struct {
	Count int    `json:"count"`
	Since string `json:"since"`
}

func (n *Notifications) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		limit := notify.DefaultLimit
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil || value < 1 || value > notify.MaxLimit {
				httpx.WriteError(w, http.StatusBadRequest, MessageInvalidNotificationLimit)
				return
			}
			limit = value
		}
		notifications, err := n.service.Recent(r.Context(), current.ID, limit)
		if err != nil {
			log.Printf("handler: could not list the notifications: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]notificationItem, 0, len(notifications))
		for _, notification := range notifications {
			items = append(items, notificationItem{
				ID:        notification.ID,
				Type:      notification.Type,
				Title:     notification.Title,
				Body:      notification.Body,
				CreatedAt: timestamp(notification.CreatedAt),
			})
		}
		httpx.WriteJSON(w, http.StatusOK, notificationsResponse{Items: items})
	}
}

func (n *Notifications) UnreadCount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		since := time.Unix(0, 0).UTC()
		if raw := strings.TrimSpace(r.URL.Query().Get("since")); raw != "" {
			value, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, MessageInvalidSince)
				return
			}
			since = value
		}
		count, err := n.service.UnreadCount(r.Context(), current.ID, since)
		if err != nil {
			log.Printf("handler: could not count the notifications: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, unreadCountResponse{Count: count, Since: timestamp(since)})
	}
}
