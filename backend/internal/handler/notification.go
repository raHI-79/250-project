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
	MessageReadNothing              = "Say which notification was opened."
	MessageInvalidBefore            = "before must be a timestamp like 2026-10-16T09:30:00+06:00."
	MaxReadPageLength               = 300
)

type NotificationService interface {
	Recent(ctx context.Context, userID int64, limit int) ([]store.Notification, error)
	UnreadCount(ctx context.Context, userID int64, since time.Time) (int, error)
	Unread(ctx context.Context, userID int64) (int, error)
	Read(ctx context.Context, userID int64, ids []int64) error
	ReadPage(ctx context.Context, userID int64, page string) (int, error)
	ReadAll(ctx context.Context, userID int64, before time.Time) error
}

type Notifications struct {
	service NotificationService
}

func NewNotifications(service NotificationService) *Notifications {
	return &Notifications{service: service}
}

type notificationLink struct {
	Kind string `json:"kind"`
	Ref  int64  `json:"ref,omitempty"`
	Date string `json:"date,omitempty"`
}

type notificationItem struct {
	ID        int64             `json:"id"`
	Type      string            `json:"type"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Link      *notificationLink `json:"link,omitempty"`
	CreatedAt string            `json:"created_at"`
	Read      bool              `json:"read"`
}

type notificationsResponse struct {
	Items  []notificationItem `json:"items"`
	Unread int                `json:"unread"`
}

type readRequest struct {
	ID   int64  `json:"id"`
	Page string `json:"page"`
}

type readAllRequest struct {
	Before string `json:"before"`
}

type notificationUnreadResponse struct {
	Unread int `json:"unread"`
	Marked int `json:"marked"`
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
			item := notificationItem{
				ID:        notification.ID,
				Type:      notification.Type,
				Title:     notification.Title,
				Body:      notification.Body,
				CreatedAt: timestamp(notification.CreatedAt),
				Read:      notification.Read,
			}
			if notification.Link.Kind != "" {
				item.Link = &notificationLink{
					Kind: notification.Link.Kind,
					Ref:  notification.Link.Ref,
					Date: notification.Link.Date,
				}
			}
			items = append(items, item)
		}
		unread, err := n.service.Unread(r.Context(), current.ID)
		if err != nil {
			log.Printf("handler: could not count the unread notifications: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, notificationsResponse{Items: items, Unread: unread})
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

func (n *Notifications) answerUnread(w http.ResponseWriter, r *http.Request, userID int64, marked int) {
	unread, err := n.service.Unread(r.Context(), userID)
	if err != nil {
		log.Printf("handler: could not count the unread notifications: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, notificationUnreadResponse{Unread: unread, Marked: marked})
}

func (n *Notifications) Read() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		var request readRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		page := strings.TrimSpace(request.Page)
		if request.ID <= 0 && (page == "" || len(page) > MaxReadPageLength) {
			httpx.WriteError(w, http.StatusBadRequest, MessageReadNothing)
			return
		}
		marked := 0
		if request.ID > 0 {
			if err := n.service.Read(r.Context(), current.ID, []int64{request.ID}); err != nil {
				log.Printf("handler: could not mark a notification read: %v", err)
				httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
				return
			}
			marked = 1
		} else {
			count, err := n.service.ReadPage(r.Context(), current.ID, page)
			if err != nil {
				log.Printf("handler: could not mark the notifications of a page read: %v", err)
				httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
				return
			}
			marked = count
		}
		n.answerUnread(w, r, current.ID, marked)
	}
}

func (n *Notifications) ReadAll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		var request readAllRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		before := time.Now()
		if raw := strings.TrimSpace(request.Before); raw != "" {
			value, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, MessageInvalidBefore)
				return
			}
			if value.Before(before) {
				before = value
			}
		}
		if err := n.service.ReadAll(r.Context(), current.ID, before); err != nil {
			log.Printf("handler: could not mark every notification read: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		n.answerUnread(w, r, current.ID, 0)
	}
}
