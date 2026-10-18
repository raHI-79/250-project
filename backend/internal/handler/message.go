package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	MessageSent            = "Message sent."
	MessageUnknownTeacher  = "No active teacher found with this id."
	MessageUnknownCR       = "No active class representative found with this id."
	MessageUnknownStudent  = "No active student found with this id."
	MessageUnknownReply    = "You can only reply to a message a class representative sent you."
	MessageOwnBatchMessage = "Use own_batch to write to your own batch."
	fieldContent           = "content"
	fieldFile              = "file"
	fieldTargetUser        = "target_user_id"
	fieldTeacherUser       = "teacher_user_id"
	fieldSection           = "section"
	fieldReplyTo           = "reply_to_id"
)

type MessageStore interface {
	GetMessageSender(ctx context.Context, userID int64) (store.Sender, error)
	FindMessageTarget(ctx context.Context, userID int64, kind string) (store.MessageTarget, error)
	FindReplyTarget(ctx context.Context, messageID, teacherUserID int64) (store.MessageTarget, error)
	CreateMessage(ctx context.Context, message store.NewMessage) (store.MessageRow, error)
	ListCRInbox(ctx context.Context, crUserID int64) ([]store.MessageRow, error)
	ListTeacherInbox(ctx context.Context, teacherUserID int64) ([]store.MessageRow, error)
	ListSentMessages(ctx context.Context, senderID int64) ([]store.MessageRow, error)
	ListStudentMessages(ctx context.Context, userID int64, batch, section string) ([]store.MessageRow, error)
	ListCROptions(ctx context.Context, batch string) ([]store.CROption, error)
	SearchStudents(ctx context.Context, search string) ([]store.StudentOption, error)
}

type Message struct {
	messages MessageStore
	files    FileStore
	alerts   BroadcastNotifier
}

func NewMessage(messages MessageStore, files FileStore, alerts BroadcastNotifier) *Message {
	return &Message{messages: messages, files: files, alerts: alerts}
}

type messageItem struct {
	ID            int64   `json:"id"`
	SenderName    string  `json:"sender_name"`
	SenderRole    string  `json:"sender_role"`
	SenderBatch   string  `json:"sender_batch,omitempty"`
	SenderSection string  `json:"sender_section,omitempty"`
	TargetType    string  `json:"target_type"`
	TargetBatch   string  `json:"target_batch,omitempty"`
	TargetSection string  `json:"target_section,omitempty"`
	Content       string  `json:"content"`
	FileURL       *string `json:"file_url"`
	ReplyToID     *int64  `json:"reply_to_id"`
	CreatedAt     string  `json:"created_at"`
}

type messagesResponse struct {
	Items []messageItem `json:"items"`
}

type sentMessageResponse struct {
	messageItem
	Message string `json:"message"`
}

type crOptionItem struct {
	UserID   int64  `json:"user_id"`
	FullName string `json:"full_name"`
	Batch    string `json:"batch"`
	Section  string `json:"section"`
}

type crOptionsResponse struct {
	Items []crOptionItem `json:"items"`
}

type studentOptionItem struct {
	UserID   int64  `json:"user_id"`
	FullName string `json:"full_name"`
	RegNo    string `json:"reg_no"`
	Batch    string `json:"batch"`
	Section  string `json:"section"`
	PhotoURL string `json:"photo_url,omitempty"`
}

type studentOptionsResponse struct {
	Items []studentOptionItem `json:"items"`
}

func (m *Message) CRSend() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sender, ok := m.sender(w, r, auth.RoleCR)
		if !ok {
			return
		}
		content, ok := m.content(w, r)
		if !ok {
			return
		}
		target, err := validate.CRMessageTarget("Target", r.FormValue(fieldTarget))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		message := store.NewMessage{
			SenderID:   sender.UserID,
			SenderRole: sender.Role,
			SenderName: sender.Name,
			Content:    content,
		}
		var direct int64
		switch target {
		case validate.TargetOwnBatch:
			message.TargetType = store.TargetBatch
			message.TargetBatch = sender.Batch
		case validate.TargetBatch:
			batch, err := validate.Batch("Batch", r.FormValue(fieldBatch))
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			if batch == sender.Batch {
				httpx.WriteError(w, http.StatusBadRequest, MessageOwnBatchMessage)
				return
			}
			message.TargetType = store.TargetBatch
			message.TargetBatch = batch
		default:
			teacher, ok := m.lookup(w, r, fieldTeacherUser, store.KindTeacher, MessageUnknownTeacher)
			if !ok {
				return
			}
			message.TargetType = store.TargetUser
			message.TargetUserID = teacher.UserID
			direct = teacher.UserID
		}
		m.send(w, r, message, direct)
	}
}

func (m *Message) CRInbox() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sender, ok := m.sender(w, r, auth.RoleCR)
		if !ok {
			return
		}
		messages, err := m.messages.ListCRInbox(r.Context(), sender.UserID)
		m.write(w, messages, err)
	}
}

func (m *Message) CRSent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sender, ok := m.sender(w, r, auth.RoleCR)
		if !ok {
			return
		}
		messages, err := m.messages.ListSentMessages(r.Context(), sender.UserID)
		m.write(w, messages, err)
	}
}

func (m *Message) TeacherSend() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sender, ok := m.sender(w, r, auth.RoleTeacher)
		if !ok {
			return
		}
		content, ok := m.content(w, r)
		if !ok {
			return
		}
		message := store.NewMessage{
			SenderID:   sender.UserID,
			SenderRole: sender.Role,
			SenderName: sender.Name,
			Content:    content,
		}
		if raw := strings.TrimSpace(r.FormValue(fieldReplyTo)); raw != "" {
			replyTo, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || replyTo <= 0 {
				httpx.WriteError(w, http.StatusBadRequest, MessageInvalidID)
				return
			}
			target, err := m.messages.FindReplyTarget(r.Context(), replyTo, sender.UserID)
			if errors.Is(err, store.ErrNotFound) {
				httpx.WriteError(w, http.StatusBadRequest, MessageUnknownReply)
				return
			}
			if err != nil {
				log.Printf("handler: could not load the message a teacher replies to: %v", err)
				httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
				return
			}
			message.TargetType = store.TargetUser
			message.TargetUserID = target.UserID
			message.ReplyToID = replyTo
			m.send(w, r, message, target.UserID)
			return
		}
		target, err := validate.TeacherMessageTarget("Target", r.FormValue(fieldTarget))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		var direct int64
		switch target {
		case validate.TargetBatch:
			batch, err := validate.Batch("Batch", r.FormValue(fieldBatch))
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			message.TargetType = store.TargetBatch
			message.TargetBatch = batch
			if raw := strings.TrimSpace(r.FormValue(fieldSection)); raw != "" {
				section, err := validate.Section("Section", raw)
				if err != nil {
					httpx.WriteError(w, http.StatusBadRequest, err.Error())
					return
				}
				message.TargetType = store.TargetSection
				message.TargetSection = section
			}
		case validate.TargetCR:
			cr, ok := m.lookup(w, r, fieldTargetUser, store.KindCR, MessageUnknownCR)
			if !ok {
				return
			}
			message.TargetType = store.TargetUser
			message.TargetUserID = cr.UserID
			direct = cr.UserID
		default:
			student, ok := m.lookup(w, r, fieldTargetUser, store.KindStudent, MessageUnknownStudent)
			if !ok {
				return
			}
			message.TargetType = store.TargetUser
			message.TargetUserID = student.UserID
			direct = student.UserID
		}
		m.send(w, r, message, direct)
	}
}

func (m *Message) TeacherInbox() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sender, ok := m.sender(w, r, auth.RoleTeacher)
		if !ok {
			return
		}
		messages, err := m.messages.ListTeacherInbox(r.Context(), sender.UserID)
		m.write(w, messages, err)
	}
}

func (m *Message) TeacherSent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sender, ok := m.sender(w, r, auth.RoleTeacher)
		if !ok {
			return
		}
		messages, err := m.messages.ListSentMessages(r.Context(), sender.UserID)
		m.write(w, messages, err)
	}
}

func (m *Message) CRLookup() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		batch := ""
		if raw := strings.TrimSpace(r.URL.Query().Get("batch")); raw != "" {
			valid, err := validate.Batch("batch", raw)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			batch = valid
		}
		options, err := m.messages.ListCROptions(r.Context(), batch)
		if err != nil {
			log.Printf("handler: could not list the class representatives: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]crOptionItem, 0, len(options))
		for _, option := range options {
			items = append(items, crOptionItem{
				UserID:   option.UserID,
				FullName: option.FullName,
				Batch:    option.Batch,
				Section:  option.Section,
			})
		}
		httpx.WriteJSON(w, http.StatusOK, crOptionsResponse{Items: items})
	}
}

func (m *Message) StudentLookup() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		search, err := validate.Search("search", r.URL.Query().Get("search"))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		options, err := m.messages.SearchStudents(r.Context(), search)
		if err != nil {
			log.Printf("handler: could not search the students: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]studentOptionItem, 0, len(options))
		for _, option := range options {
			items = append(items, studentOptionItem{
				UserID:   option.UserID,
				FullName: option.FullName,
				RegNo:    option.RegNo,
				Batch:    option.Batch,
				Section:  option.Section,
				PhotoURL: option.PhotoPath,
			})
		}
		httpx.WriteJSON(w, http.StatusOK, studentOptionsResponse{Items: items})
	}
}

func (m *Message) StudentInbox() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		sender, ok := m.load(w, r, current.ID)
		if !ok {
			return
		}
		if sender.Batch == "" {
			httpx.WriteError(w, http.StatusForbidden, MessageNotAStudent)
			return
		}
		messages, err := m.messages.ListStudentMessages(r.Context(), sender.UserID, sender.Batch, sender.Section)
		m.write(w, messages, err)
	}
}

func (m *Message) send(w http.ResponseWriter, r *http.Request, message store.NewMessage, direct int64) {
	filePath, err := m.files.Save(r)
	if err != nil {
		writeFileError(w, err)
		return
	}
	if message.Content == "" && filePath == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Content is required when no file is attached.")
		return
	}
	message.FilePath = filePath
	created, err := m.messages.CreateMessage(r.Context(), message)
	if err != nil {
		if filePath != "" {
			if removeErr := m.files.Delete(filePath); removeErr != nil {
				log.Printf("handler: could not delete the file of a rejected message: %v", removeErr)
			}
		}
		log.Printf("handler: could not create a message: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return
	}
	m.notify(r.Context(), created, direct)
	httpx.WriteJSON(w, http.StatusCreated, sentMessageResponse{messageItem: newMessageItem(created), Message: MessageSent})
}

func (m *Message) notify(ctx context.Context, message store.MessageRow, direct int64) {
	title := "New message from " + message.SenderName
	body := message.Content
	if body == "" {
		body = title
	}
	var err error
	if direct > 0 {
		err = m.alerts.NotifyUser(ctx, direct, notify.TypeMessage, title, body)
	} else {
		err = m.alerts.NotifyBroadcast(ctx, auth.RoleStudent, message.TargetBatch, message.TargetSection,
			notify.TypeMessage, title, body)
	}
	if err != nil {
		log.Printf("handler: could not notify the recipients of a message: %v", err)
	}
}

func (m *Message) content(w http.ResponseWriter, r *http.Request) (string, bool) {
	if err := m.files.Parse(r); err != nil {
		writeFileError(w, err)
		return "", false
	}
	hasFile := r.MultipartForm != nil && len(r.MultipartForm.File[fieldFile]) > 0
	content, err := validate.MessageContent("Content", r.FormValue(fieldContent), hasFile)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return "", false
	}
	return content, true
}

func (m *Message) sender(w http.ResponseWriter, r *http.Request, want string) (store.Sender, bool) {
	current, ok := middleware.CurrentUser(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
		return store.Sender{}, false
	}
	sender, ok := m.load(w, r, current.ID)
	if !ok {
		return store.Sender{}, false
	}
	if sender.Role != want {
		if want == auth.RoleTeacher {
			httpx.WriteError(w, http.StatusForbidden, MessageNotATeacher)
			return store.Sender{}, false
		}
		httpx.WriteError(w, http.StatusForbidden, MessageNotACR)
		return store.Sender{}, false
	}
	return sender, true
}

func (m *Message) load(w http.ResponseWriter, r *http.Request, userID int64) (store.Sender, bool) {
	sender, err := m.messages.GetMessageSender(r.Context(), userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusForbidden, middleware.MessageAccessDenied)
			return store.Sender{}, false
		}
		log.Printf("handler: could not load the sender of a message: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return store.Sender{}, false
	}
	return sender, true
}

func (m *Message) lookup(w http.ResponseWriter, r *http.Request, field, kind, unknown string) (store.MessageTarget, bool) {
	userID, err := strconv.ParseInt(strings.TrimSpace(r.FormValue(field)), 10, 64)
	if err != nil || userID <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, MessageInvalidID)
		return store.MessageTarget{}, false
	}
	target, err := m.messages.FindMessageTarget(r.Context(), userID, kind)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusBadRequest, unknown)
		return store.MessageTarget{}, false
	}
	if err != nil {
		log.Printf("handler: could not load the target of a message: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return store.MessageTarget{}, false
	}
	return target, true
}

func (m *Message) write(w http.ResponseWriter, messages []store.MessageRow, err error) {
	if err != nil {
		log.Printf("handler: could not list the messages: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return
	}
	items := make([]messageItem, 0, len(messages))
	for _, message := range messages {
		items = append(items, newMessageItem(message))
	}
	httpx.WriteJSON(w, http.StatusOK, messagesResponse{Items: items})
}

func newMessageItem(message store.MessageRow) messageItem {
	item := messageItem{
		ID:            message.ID,
		SenderName:    message.SenderName,
		SenderRole:    message.SenderRole,
		SenderBatch:   message.SenderBatch,
		SenderSection: message.SenderSection,
		TargetType:    message.TargetType,
		TargetBatch:   message.TargetBatch,
		TargetSection: message.TargetSection,
		Content:       message.Content,
		CreatedAt:     message.CreatedAt.Format(time.RFC3339),
	}
	if message.FilePath != "" {
		link := message.FilePath
		item.FileURL = &link
	}
	if message.ReplyToID != 0 {
		replyTo := message.ReplyToID
		item.ReplyToID = &replyTo
	}
	return item
}
