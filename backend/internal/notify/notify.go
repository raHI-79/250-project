package notify

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	TypeNotice   = "notice"
	TypeRequest  = "request"
	TypeDecision = "decision"
	TypeClass    = "class"
	TypeMessage  = "message"
	TypeRoutine  = "routine"
	TypeExam     = "exam"
	DefaultLimit = 30
	MaxLimit     = 100

	LinkTeacherNextDay = "teacher_next_day"
	LinkTeacherFinal   = "teacher_final"
	LinkCRUpcoming     = "cr_upcoming"
	LinkCRFinal        = "cr_final"
	LinkRoutine        = "routine"
	LinkExam           = "exam"
	LinkNotice         = "notice"
	LinkMessage        = "message"
	LinkPoll           = "poll"
	LinkCRRole         = "cr_role"
)

var (
	ErrInvalidRecipient = errors.New("notify: a notification needs a user id or a broadcast target")
	ErrInvalidRole      = errors.New("notify: the broadcast role is not a known role")
	ErrInvalidContent   = errors.New("notify: a notification needs a type and a title")
	ErrInvalidLink      = errors.New("notify: a notification needs a known link kind")
)

var linkPages = map[string]string{
	LinkTeacherNextDay: "/teacher/next-day",
	LinkTeacherFinal:   "/teacher/final-routine",
	LinkCRUpcoming:     "/cr/upcoming",
	LinkCRFinal:        "/cr/final-routine",
	LinkRoutine:        "/student/upcoming-routine",
	LinkExam:           "/student/exams",
	LinkNotice:         "/student/notices",
	LinkMessage:        "/student/messages",
	LinkPoll:           "/student/polls",
	LinkCRRole:         "/profile",
}

var linkAnchors = map[string]string{
	LinkCRUpcoming: "item",
	LinkExam:       "exam",
	LinkNotice:     "notice",
	LinkPoll:       "poll",
}

type Store interface {
	CreateDirectNotification(ctx context.Context, userID int64, kind, title, body string, link store.NotificationLink) error
	CreateBroadcastNotification(ctx context.Context, role, batch, section, kind, title, body string, link store.NotificationLink) error
	ListNotifications(ctx context.Context, userID int64, limit int) ([]store.Notification, error)
	CountNotificationsSince(ctx context.Context, userID int64, since time.Time) (int, error)
}

type Service struct {
	notifications Store
}

func New(notifications Store) *Service {
	return &Service{notifications: notifications}
}

func Page(kind string) store.NotificationLink {
	return store.NotificationLink{Kind: kind}
}

func Ref(kind string, ref int64) store.NotificationLink {
	return store.NotificationLink{Kind: kind, Ref: ref}
}

func OnDate(kind, date string) store.NotificationLink {
	return store.NotificationLink{Kind: kind, Date: date}
}

func KnownLink(kind string) bool {
	_, ok := linkPages[kind]
	return ok
}

func PageURL(origin string, link store.NotificationLink) string {
	page, ok := linkPages[link.Kind]
	if !ok {
		return strings.TrimSuffix(origin, "/")
	}
	address := strings.TrimSuffix(origin, "/") + page
	if link.Date != "" {
		address += "?date=" + url.QueryEscape(link.Date)
	}
	if link.Kind == LinkMessage && link.Ref > 0 {
		return address + "?conversation=" + strconv.FormatInt(link.Ref, 10)
	}
	if anchor, has := linkAnchors[link.Kind]; has && link.Ref > 0 {
		return address + "#" + anchor + "-" + strconv.FormatInt(link.Ref, 10)
	}
	return address
}

func (s *Service) NotifyUser(ctx context.Context, userID int64, kind, title, body string,
	link store.NotificationLink) error {
	if userID <= 0 {
		return ErrInvalidRecipient
	}
	kind, title, body, err := content(kind, title, body)
	if err != nil {
		return err
	}
	if !KnownLink(link.Kind) {
		return ErrInvalidLink
	}
	return s.notifications.CreateDirectNotification(ctx, userID, kind, title, body, link)
}

func (s *Service) NotifyBroadcast(ctx context.Context, role, batch, section, kind, title, body string,
	link store.NotificationLink) error {
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
	if !KnownLink(link.Kind) {
		return ErrInvalidLink
	}
	return s.notifications.CreateBroadcastNotification(ctx, role, batch, section, kind, title, body, link)
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
