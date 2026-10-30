package notify

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/push"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
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
	TypeAttend   = "attendance"
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
	LinkAttendMark     = "attendance_mark"
	LinkAttendMine     = "attendance_mine"
	LinkAttendTeacher  = "attendance_teacher"
	LinkPhoneRequests  = "phone_requests"
)

var (
	ErrInvalidRecipient = errors.New("notify: a notification needs a user id or a broadcast target")
	ErrInvalidRole      = errors.New("notify: the broadcast role is not a known role")
	ErrInvalidContent   = errors.New("notify: a notification needs a type and a title")
	ErrInvalidLink      = errors.New("notify: a notification needs a known link kind")
)

const (
	TabNextDay  = "next-day"
	TabFinal    = "final"
	TabUpcoming = "upcoming"
)

var linkPages = map[string]string{
	LinkTeacherNextDay: "/teacher/upcoming-routine",
	LinkTeacherFinal:   "/teacher/upcoming-routine",
	LinkCRUpcoming:     "/cr/create-class",
	LinkCRFinal:        "/cr/create-class",
	LinkRoutine:        "/student/upcoming-routine",
	LinkExam:           "/student/exams",
	LinkNotice:         "/student/notices",
	LinkMessage:        "/student/messages",
	LinkPoll:           "/student/polls",
	LinkCRRole:         "/profile",
	LinkAttendMark:     "/student/attendance",
	LinkAttendMine:     "/student/my-attendance",
	LinkAttendTeacher:  "/teacher/attendance",
	LinkPhoneRequests:  "/teacher/phone-requests",
}

type Store interface {
	CreateDirectNotification(ctx context.Context, userID int64, kind, title, body string, link store.NotificationLink) error
	CreateBroadcastNotification(ctx context.Context, role, batch, section, kind, title, body string, link store.NotificationLink) error
	ListNotifications(ctx context.Context, userID int64, limit int) ([]store.Notification, error)
	CountNotificationsSince(ctx context.Context, userID int64, since time.Time) (int, error)
	ListNotificationRecipients(ctx context.Context, role, batch, section string) ([]int64, error)
	ListNotificationAudience(ctx context.Context, role, batch, section string) ([]store.NotificationRecipient, error)
	NotificationRole(ctx context.Context, userID int64) (string, error)
}

type Pusher interface {
	Queue(userIDs []int64, payload push.Payload)
}

type Devices interface {
	Deliver(ctx context.Context, userIDs []int64, title, body, url, tag string) int
	Ready() bool
}

type Service struct {
	notifications Store
	pushes        Pusher
	devices       Devices
	origin        string
}

func New(notifications Store) *Service {
	return &Service{notifications: notifications}
}

func (s *Service) WithPush(pushes Pusher, origin string) *Service {
	s.pushes = pushes
	s.origin = origin
	return s
}

func (s *Service) WithDevices(devices Devices) *Service {
	s.devices = devices
	return s
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

func pageFor(kind, role string) (string, bool) {
	switch kind {
	case LinkExam:
		if role == auth.RoleTeacher {
			return "/teacher/create-exam", true
		}
	case LinkNotice:
		if role == auth.RoleTeacher {
			return "/teacher/notices", true
		}
	case LinkMessage:
		if role == auth.RoleTeacher {
			return "/teacher/messages", true
		}
	case LinkPoll:
		if role == auth.RoleCR {
			return "/cr/polls", true
		}
	}
	page, ok := linkPages[kind]
	return page, ok
}

func linkQuery(link store.NotificationLink) url.Values {
	query := url.Values{}
	switch link.Kind {
	case LinkTeacherNextDay:
		query.Set("tab", TabNextDay)
	case LinkTeacherFinal:
		query.Set("tab", TabFinal)
	case LinkCRUpcoming:
		query.Set("tab", TabUpcoming)
	case LinkCRFinal:
		query.Set("tab", TabFinal)
	}
	switch link.Kind {
	case LinkTeacherNextDay, LinkTeacherFinal, LinkCRUpcoming, LinkCRFinal:
		if link.Ref > 0 {
			query.Set("item", strconv.FormatInt(link.Ref, 10))
		}
	case LinkMessage:
		if link.Ref > 0 {
			query.Set("c", strconv.FormatInt(link.Ref, 10))
		}
	case LinkExam, LinkNotice, LinkPoll, LinkAttendTeacher, LinkPhoneRequests:
		if link.Ref > 0 {
			query.Set("highlight", strconv.FormatInt(link.Ref, 10))
		}
	case LinkAttendMark:
		if link.Ref > 0 {
			query.Set("session", strconv.FormatInt(link.Ref, 10))
		}
	case LinkAttendMine:
		if link.Ref > 0 {
			query.Set("course", strconv.FormatInt(link.Ref, 10))
		}
	case LinkRoutine:
		if link.Date != "" {
			query.Set("date", link.Date)
		}
	}
	return query
}

func PageURL(origin, role string, link store.NotificationLink) string {
	base := strings.TrimSuffix(origin, "/")
	page, ok := pageFor(link.Kind, role)
	if !ok {
		return base
	}
	address := base + page
	if query := linkQuery(link); len(query) > 0 {
		address += "?" + query.Encode()
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
	if err := s.notifications.CreateDirectNotification(ctx, userID, kind, title, body, link); err != nil {
		return err
	}
	role, err := s.notifications.NotificationRole(ctx, userID)
	if err != nil {
		log.Printf("notify: could not read the role of a push recipient: %v", err)
		role = ""
	}
	s.queue(ctx, []int64{userID}, PageURL(s.origin, role, link), kind, title, body)
	return nil
}

func (s *Service) PushUser(ctx context.Context, userID int64, kind, title, body string,
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
	role, err := s.notifications.NotificationRole(ctx, userID)
	if err != nil {
		log.Printf("notify: could not read the role of a push recipient: %v", err)
		role = ""
	}
	s.queue(ctx, []int64{userID}, PageURL(s.origin, role, link), kind, title, body)
	return nil
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
	err = s.notifications.CreateBroadcastNotification(ctx, role, batch, section, kind, title, body, link)
	if err != nil {
		return err
	}
	audience, err := s.notifications.ListNotificationAudience(ctx, role, batch, section)
	if err != nil {
		log.Printf("notify: could not list the push recipients: %v", err)
		return nil
	}
	for page, userIDs := range s.pagesFor(audience, link) {
		s.queue(ctx, userIDs, page, kind, title, body)
	}
	return nil
}

func (s *Service) pagesFor(audience []store.NotificationRecipient,
	link store.NotificationLink) map[string][]int64 {
	pages := make(map[string][]int64)
	for role, userIDs := range GroupByRole(audience) {
		page := PageURL(s.origin, role, link)
		pages[page] = append(pages[page], userIDs...)
	}
	return pages
}

func GroupByRole(audience []store.NotificationRecipient) map[string][]int64 {
	grouped := make(map[string][]int64)
	for _, recipient := range audience {
		if recipient.UserID <= 0 {
			continue
		}
		grouped[recipient.Role] = append(grouped[recipient.Role], recipient.UserID)
	}
	return grouped
}

func (s *Service) queue(ctx context.Context, userIDs []int64, page, kind, title, body string) {
	if len(userIDs) == 0 || ctx.Err() != nil {
		return
	}
	if s.pushes != nil {
		s.pushes.Queue(userIDs, push.Payload{Title: title, Body: body, URL: page, Tag: kind})
	}
	if s.devices != nil && s.devices.Ready() {
		go s.devices.Deliver(context.WithoutCancel(ctx), userIDs, title, body, page, kind)
	}
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
