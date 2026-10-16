package notify

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/store"
	"strings"
	"time"
)

const (
	TypeNotice   = "notice"
	TypeRequest  = "request"
	TypeDecision = "decision"
	TypeClass    = "class"
	TypeMessage  = "message"
	DefaultLimit = 30
	MaxLimit     = 100
)

var (
	ErrInvalidRecipient = errors.New("notify: a notification needs a user id or a broadcast target")
	ErrInvalidRole      = errors.New("notify: the broadcast role is not a known role")
	ErrInvalidContent   = errors.New("notify: a notification needs a type and a title")
)

type Store interface {
	CreateDirectNotification(ctx context.Context, userID int64, kind, title, body string) error
	CreateBroadcastNotification(ctx context.Context, role, batch, section, kind, title, body string) error
	ListNotifications(ctx context.Context, userID int64, limit int) ([]store.Notification, error)
	CountNotificationsSince(ctx context.Context, userID int64, since time.Time) (int, error)
}

type Service struct {
	notifications Store
}

func New(notifications Store) *Service {
	return &Service{notifications: notifications}
}

func (s *Service) NotifyUser(ctx context.Context, userID int64, kind, title, body string) error {
	if userID <= 0 {
		return ErrInvalidRecipient
	}
	kind, title, body, err := content(kind, title, body)
	if err != nil {
		return err
	}
	return s.notifications.CreateDirectNotification(ctx, userID, kind, title, body)
}

func (s *Service) NotifyBroadcast(ctx context.Context, role, batch, section, kind, title, body string) error {
	role = strings.ToLower(strings.TrimSpace(role))
	batch = strings.ToUpper(strings.TrimSpace(batch))
	section = strings.ToUpper(strings.TrimSpace(section))
	if role == "" && batch == "" {
		return ErrInvalidRecipient
	}
	if role != "" && !auth.IsEffectiveRole(role) {
		return ErrInvalidRole
	}
	if section != "" && batch == "" {
		return ErrInvalidRecipient
	}
	kind, title, body, err := content(kind, title, body)
	if err != nil {
		return err
	}
	return s.notifications.CreateBroadcastNotification(ctx, role, batch, section, kind, title, body)
}

func (s *Service) Recent(ctx context.Context, userID int64, limit int) ([]store.Notification, error) {
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	return s.notifications.ListNotifications(ctx, userID, limit)
}

func (s *Service) UnreadCount(ctx context.Context, userID int64, since time.Time) (int, error) {
	return s.notifications.CountNotificationsSince(ctx, userID, since)
}

func content(kind, title, body string) (string, string, string, error) {
	kind = strings.TrimSpace(kind)
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if kind == "" || title == "" {
		return "", "", "", ErrInvalidContent
	}
	return kind, title, body, nil
}
