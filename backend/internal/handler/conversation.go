package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/upload"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	MessageConversationMissing = "No conversation found with this ID."
	MessageCannotPost          = "Only members of this group can post in it."
	MessageNeedsContent        = "Write something, add a link or attach a file."
	MessageUnknownContact      = "No active account found with this ID."
	MessageSelfConversation    = "You cannot start a conversation with yourself."
	MessageGroupNotAllowed     = "Only teachers and class representatives can start a group."
	MessageContactBlocked      = "Students can only message teachers, class representatives and the department office."
	MessageMessageMissing      = "No message of yours found with this ID."
	MessageEditWindowOver      = "A message can only be edited or deleted within two minutes of sending it."
	MessageEdited              = "Message updated."
	MessageDeleted             = "Message deleted."
	MessageChatCleared         = "Chat deleted for you."
	MessageSent                = "Message sent."
	MessageFileNotAllowed      = "This file type is not allowed."
	fieldBody                  = "body"
	fieldLink                  = "link"
	fieldSection               = "section"
)

type ConversationStore interface {
	GetMessageViewer(ctx context.Context, userID int64) (store.MessageViewer, error)
	SearchContacts(ctx context.Context, viewerID int64, viewerRole, search, role string) ([]store.ContactRow, error)
	GetOrCreateDirect(ctx context.Context, viewerID int64, viewerRole string, otherID int64) (store.Conversation, error)
	CreateGroup(ctx context.Context, group store.NewGroup) (store.Conversation, error)
	ListConversations(ctx context.Context, viewer store.MessageViewer, limit int) ([]store.ConversationRow, error)
	FindConversationAccess(ctx context.Context, conversationID int64, viewer store.MessageViewer) (store.ConversationAccess, error)
	ListChatMessages(ctx context.Context, conversationID int64, before *time.Time, limit int, viewerID int64) ([]store.ChatMessageRow, error)
	CreateChatMessage(ctx context.Context, message store.NewChatMessage) (store.ChatMessageRow, error)
	FindOwnMessage(ctx context.Context, messageID, senderID int64) (store.OwnMessage, error)
	EditChatMessage(ctx context.Context, messageID, senderID int64, body string, now time.Time) (store.ChatMessageRow, error)
	DeleteChatMessage(ctx context.Context, messageID, senderID int64, now time.Time) (int64, error)
	ClearConversation(ctx context.Context, userID, conversationID int64, at time.Time) error
	MarkConversationRead(ctx context.Context, userID, conversationID int64, at time.Time) error
	CountUnreadMessages(ctx context.Context, viewer store.MessageViewer) (int, error)
	ConversationRecipients(ctx context.Context, conversationID, senderID int64) ([]int64, error)
	ListConversationMembers(ctx context.Context, conversationID int64) ([]store.MemberRow, error)
	AddConversationMembers(ctx context.Context, conversationID, actorID int64, userIDs []int64) (int, error)
	RemoveConversationMember(ctx context.Context, conversationID, userID int64) error
	SetConversationMemberAdmin(ctx context.Context, conversationID, userID int64, admin bool) error
	LeaveConversation(ctx context.Context, conversationID, userID int64) error
	RenameConversation(ctx context.Context, conversationID int64, title string) error
	ActiveUserIDs(ctx context.Context, userIDs []int64) ([]int64, error)
	BatchMembers(ctx context.Context, batch, section string) ([]int64, error)
}

type AttachmentStore interface {
	Parse(r *http.Request) error
	Save(r *http.Request) (upload.Attachment, error)
	Delete(storedPath string) error
}

type Conversations struct {
	cfg    *config.Config
	chats  ConversationStore
	files  AttachmentStore
	alerts BroadcastNotifier
}

func NewConversations(cfg *config.Config, chats ConversationStore, files AttachmentStore, alerts BroadcastNotifier) *Conversations {
	return &Conversations{cfg: cfg, chats: chats, files: files, alerts: alerts}
}

type contactItem struct {
	UserID      int64  `json:"user_id"`
	Name        string `json:"name"`
	Role        string `json:"role"`
	Batch       string `json:"batch,omitempty"`
	Section     string `json:"section,omitempty"`
	Designation string `json:"designation,omitempty"`
	PhotoURL    string `json:"photo_url,omitempty"`
}

type contactsResponse struct {
	Items []contactItem `json:"items"`
}

type conversationItem struct {
	ID            int64  `json:"id"`
	Kind          string `json:"kind"`
	Title         string `json:"title"`
	OtherUserID   int64  `json:"other_user_id,omitempty"`
	OtherRole     string `json:"other_role,omitempty"`
	AvatarURL     string `json:"photo_url"`
	CreatedBy     int64  `json:"created_by"`
	CanPost       bool   `json:"can_post"`
	MemberRole    string `json:"member_role,omitempty"`
	MemberCount   int    `json:"member_count,omitempty"`
	Preview       string `json:"preview"`
	LastMessageAt string `json:"last_message_at,omitempty"`
	UnreadCount   int    `json:"unread_count"`
}

type conversationsResponse struct {
	Items []conversationItem `json:"items"`
}

type conversationResponse struct {
	ID          int64  `json:"id"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	CanPost     bool   `json:"can_post"`
	MemberRole  string `json:"member_role,omitempty"`
	MemberCount int    `json:"member_count,omitempty"`
	Message     string `json:"message,omitempty"`
}

type chatMessageItem struct {
	ID         int64  `json:"id"`
	SenderID   int64  `json:"sender_id"`
	SenderName string `json:"sender_name"`
	SenderRole string `json:"sender_role"`
	PhotoURL   string `json:"photo_url,omitempty"`
	Body       string `json:"body"`
	LinkURL    string `json:"link_url,omitempty"`
	FileURL    string `json:"file_url,omitempty"`
	FileName   string `json:"file_name,omitempty"`
	FileMime   string `json:"file_mime,omitempty"`
	FileSize   int64  `json:"file_size,omitempty"`
	CreatedAt  string `json:"created_at"`
	EditedAt   string `json:"edited_at,omitempty"`
	IsDeleted  bool   `json:"is_deleted,omitempty"`
}

type chatMessagesResponse struct {
	Items []chatMessageItem `json:"items"`
}

type unreadResponse struct {
	Count int `json:"count"`
}

type directRequest struct {
	UserID int64 `json:"user_id"`
}

type editMessageRequest struct {
	Body string `json:"body"`
}

func (c *Conversations) viewer(w http.ResponseWriter, r *http.Request) (store.MessageViewer, bool) {
	current, ok := middleware.CurrentUser(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
		return store.MessageViewer{}, false
	}
	viewer, err := c.chats.GetMessageViewer(r.Context(), current.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageSessionExpired)
			return store.MessageViewer{}, false
		}
		log.Printf("handler: could not load a messaging viewer: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return store.MessageViewer{}, false
	}
	return viewer, true
}

func (c *Conversations) Contacts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		viewer, ok := c.viewer(w, r)
		if !ok {
			return
		}
		query := r.URL.Query()
		search, err := validate.Search("search", query.Get("search"))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		role := ""
		if raw := strings.TrimSpace(query.Get("role")); raw != "" {
			role, err = validate.MessageRole("role", raw)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
		}
		contacts, err := c.chats.SearchContacts(r.Context(), viewer.UserID, viewer.Role, search, role)
		if err != nil {
			log.Printf("handler: could not search the contacts: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]contactItem, 0, len(contacts))
		for _, contact := range contacts {
			items = append(items, contactItem{
				UserID:      contact.UserID,
				Name:        contact.Name,
				Role:        contact.Role,
				Batch:       contact.Batch,
				Section:     contact.Section,
				Designation: contact.Designation,
				PhotoURL:    contact.PhotoPath,
			})
		}
		httpx.WriteJSON(w, http.StatusOK, contactsResponse{Items: items})
	}
}

func (c *Conversations) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		viewer, ok := c.viewer(w, r)
		if !ok {
			return
		}
		rows, err := c.chats.ListConversations(r.Context(), viewer, store.MaxMessagePage)
		if err != nil {
			log.Printf("handler: could not list the conversations: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]conversationItem, 0, len(rows))
		for _, row := range rows {
			item := conversationItem{
				ID:          row.ID,
				Kind:        row.Kind,
				Title:       conversationTitle(row),
				CreatedBy:   row.CreatedBy,
				CanPost:     true,
				Preview:     preview(row),
				UnreadCount: row.UnreadCount,
			}
			if row.Kind == store.ConversationGroup {
				item.MemberRole = row.MemberRole
				item.MemberCount = row.MemberCount
			}
			if row.Kind == store.ConversationDirect {
				item.OtherUserID = row.OtherUserID
				item.OtherRole = row.OtherRole
				item.AvatarURL = row.OtherPhotoPath
			}
			if row.LastMessageAt != nil {
				item.LastMessageAt = timestamp(*row.LastMessageAt)
			}
			items = append(items, item)
		}
		httpx.WriteJSON(w, http.StatusOK, conversationsResponse{Items: items})
	}
}

func (c *Conversations) StartDirect() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		viewer, ok := c.viewer(w, r)
		if !ok {
			return
		}
		var request directRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		if request.UserID <= 0 {
			httpx.WriteError(w, http.StatusBadRequest, MessageInvalidUserID)
			return
		}
		conversation, err := c.chats.GetOrCreateDirect(r.Context(), viewer.UserID, viewer.Role, request.UserID)
		switch {
		case errors.Is(err, store.ErrContactBlocked):
			httpx.WriteError(w, http.StatusForbidden, MessageContactBlocked)
			return
		case errors.Is(err, store.ErrSameUser):
			httpx.WriteError(w, http.StatusBadRequest, MessageSelfConversation)
			return
		case errors.Is(err, store.ErrInactiveTarget):
			httpx.WriteError(w, http.StatusNotFound, MessageUnknownContact)
			return
		case err != nil:
			log.Printf("handler: could not open a direct conversation: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, conversationResponse{
			ID:      conversation.ID,
			Kind:    conversation.Kind,
			CanPost: true,
		})
	}
}

func (c *Conversations) Messages() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		viewer, ok := c.viewer(w, r)
		if !ok {
			return
		}
		access, ok := c.access(w, r, viewer)
		if !ok {
			return
		}
		query := r.URL.Query()
		var before *time.Time
		if raw := strings.TrimSpace(query.Get("before")); raw != "" {
			parsed, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, "before must be an RFC 3339 timestamp.")
				return
			}
			before = &parsed
		}
		limit := store.MessagePageSize
		if raw := strings.TrimSpace(query.Get("limit")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 1 || parsed > store.MaxMessagePage {
				httpx.WriteError(w, http.StatusBadRequest, "limit must be a number between 1 and 100.")
				return
			}
			limit = parsed
		}
		messages, err := c.chats.ListChatMessages(r.Context(), access.ID, before, limit, viewer.UserID)
		if err != nil {
			log.Printf("handler: could not list the messages: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if err := c.chats.MarkConversationRead(r.Context(), viewer.UserID, access.ID, c.cfg.Now()); err != nil {
			log.Printf("handler: could not mark the conversation read: %v", err)
		}
		items := make([]chatMessageItem, 0, len(messages))
		for _, message := range messages {
			items = append(items, newChatMessageItem(message))
		}
		httpx.WriteJSON(w, http.StatusOK, chatMessagesResponse{Items: items})
	}
}

func (c *Conversations) Send() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		viewer, ok := c.viewer(w, r)
		if !ok {
			return
		}
		if err := c.files.Parse(r); err != nil {
			status, message := attachmentError(err)
			httpx.WriteError(w, status, message)
			return
		}
		access, ok := c.access(w, r, viewer)
		if !ok {
			return
		}
		link, err := validate.OptionalLink("Link", r.FormValue(fieldLink))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		attachment, err := c.files.Save(r)
		if err != nil {
			status, message := attachmentError(err)
			httpx.WriteError(w, status, message)
			return
		}
		body, err := validate.MessageBody("Message", r.FormValue(fieldBody), attachment.Path != "" || link != "")
		if err != nil {
			c.removeAttachment(attachment.Path)
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		created, err := c.chats.CreateChatMessage(r.Context(), store.NewChatMessage{
			ConversationID: access.ID,
			SenderID:       viewer.UserID,
			Body:           body,
			LinkURL:        link,
			FilePath:       attachment.Path,
			FileName:       attachment.Name,
			FileMime:       attachment.Mime,
			FileSize:       attachment.Size,
		})
		if err != nil {
			c.removeAttachment(attachment.Path)
			log.Printf("handler: could not create a message: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		c.announce(r.Context(), access, viewer, created)
		item := newChatMessageItem(created)
		item.SenderName = viewer.Name
		item.SenderRole = viewer.Role
		httpx.WriteJSON(w, http.StatusCreated, item)
	}
}

func (c *Conversations) ClearChat() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		viewer, ok := c.viewer(w, r)
		if !ok {
			return
		}
		access, ok := c.access(w, r, viewer)
		if !ok {
			return
		}
		if err := c.chats.ClearConversation(r.Context(), viewer.UserID, access.ID, c.cfg.Now()); err != nil {
			log.Printf("handler: could not clear a conversation: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageChatCleared})
	}
}

func (c *Conversations) Edit() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		viewer, ok := c.viewer(w, r)
		if !ok {
			return
		}
		messageID, ok := pathID(w, r)
		if !ok {
			return
		}
		var request editMessageRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		own, ok := c.ownMessage(w, r, viewer, messageID)
		if !ok {
			return
		}
		body, err := validate.MessageBody("Message", request.Body, own.HasAttachment)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		updated, err := c.chats.EditChatMessage(r.Context(), messageID, viewer.UserID, body, c.cfg.Now())
		switch {
		case errors.Is(err, store.ErrEditWindowOver):
			httpx.WriteError(w, http.StatusConflict, MessageEditWindowOver)
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageMessageMissing)
		case err != nil:
			log.Printf("handler: could not edit a message: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusOK, newChatMessageItem(updated))
		}
	}
}

func (c *Conversations) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		viewer, ok := c.viewer(w, r)
		if !ok {
			return
		}
		messageID, ok := pathID(w, r)
		if !ok {
			return
		}
		own, ok := c.ownMessage(w, r, viewer, messageID)
		if !ok {
			return
		}
		_, err := c.chats.DeleteChatMessage(r.Context(), messageID, viewer.UserID, c.cfg.Now())
		switch {
		case errors.Is(err, store.ErrEditWindowOver):
			httpx.WriteError(w, http.StatusConflict, MessageEditWindowOver)
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageMessageMissing)
		case err != nil:
			log.Printf("handler: could not delete a message: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			c.removeAttachment(own.FilePath)
			httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageDeleted})
		}
	}
}

func (c *Conversations) ownMessage(w http.ResponseWriter, r *http.Request, viewer store.MessageViewer,
	messageID int64) (store.OwnMessage, bool) {
	own, err := c.chats.FindOwnMessage(r.Context(), messageID, viewer.UserID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, MessageMessageMissing)
		return store.OwnMessage{}, false
	}
	if err != nil {
		log.Printf("handler: could not load a message: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return store.OwnMessage{}, false
	}
	if own.Deleted || !c.cfg.Now().Before(own.CreatedAt.Add(store.EditWindow)) {
		httpx.WriteError(w, http.StatusConflict, MessageEditWindowOver)
		return store.OwnMessage{}, false
	}
	return own, true
}

func newChatMessageItem(message store.ChatMessageRow) chatMessageItem {
	item := chatMessageItem{
		ID:         message.ID,
		SenderID:   message.SenderID,
		SenderName: message.SenderName,
		SenderRole: message.SenderRole,
		PhotoURL:   message.SenderPhotoPath,
		Body:       message.Body,
		LinkURL:    message.LinkURL,
		FileURL:    upload.AttachmentURL(message.FilePath, message.FileName),
		FileName:   message.FileName,
		FileMime:   message.FileMime,
		FileSize:   message.FileSize,
		CreatedAt:  timestamp(message.CreatedAt),
		IsDeleted:  message.DeletedAt != nil,
	}
	if message.EditedAt != nil {
		item.EditedAt = timestamp(*message.EditedAt)
	}
	return item
}

func (c *Conversations) UnreadCount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		viewer, ok := c.viewer(w, r)
		if !ok {
			return
		}
		count, err := c.chats.CountUnreadMessages(r.Context(), viewer)
		if err != nil {
			log.Printf("handler: could not count the unread messages: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, unreadResponse{Count: count})
	}
}

func (c *Conversations) access(w http.ResponseWriter, r *http.Request, viewer store.MessageViewer) (store.ConversationAccess, bool) {
	conversationID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || conversationID <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, MessageInvalidID)
		return store.ConversationAccess{}, false
	}
	access, err := c.chats.FindConversationAccess(r.Context(), conversationID, viewer)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, MessageConversationMissing)
		return store.ConversationAccess{}, false
	}
	if err != nil {
		log.Printf("handler: could not load a conversation: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return store.ConversationAccess{}, false
	}
	return access, true
}

func (c *Conversations) announce(ctx context.Context, access store.ConversationAccess,
	viewer store.MessageViewer, message store.ChatMessageRow) {
	title := "New message from " + viewer.Name
	if access.Kind == store.ConversationGroup {
		title = viewer.Name + " in " + access.Title
	}
	recipients, err := c.chats.ConversationRecipients(ctx, access.ID, viewer.UserID)
	if err != nil {
		log.Printf("handler: could not find the message recipients: %v", err)
		return
	}
	c.notifyMembers(ctx, recipients, viewer, title, summary(message), access.ID)
}

func (c *Conversations) notifyMembers(ctx context.Context, recipients []int64,
	viewer store.MessageViewer, title, body string, conversationID int64) {
	link := notify.Ref(notify.LinkMessage, conversationID)
	for _, recipient := range recipients {
		if recipient == viewer.UserID {
			continue
		}
		if err := c.alerts.PushUser(ctx, recipient, notify.TypeMessage, title, body, link); err != nil {
			log.Printf("handler: could not notify a conversation member: %v", err)
		}
	}
}

func (c *Conversations) removeAttachment(storedPath string) {
	if storedPath == "" {
		return
	}
	if err := c.files.Delete(storedPath); err != nil {
		log.Printf("handler: could not delete an attachment: %v", err)
	}
}

func conversationTitle(row store.ConversationRow) string {
	if row.Kind == store.ConversationDirect {
		return row.OtherName
	}
	return row.Title
}

func preview(row store.ConversationRow) string {
	if row.LastBody != "" {
		return row.LastBody
	}
	if row.LastFileName != "" {
		return row.LastFileName
	}
	return ""
}

func summary(message store.ChatMessageRow) string {
	if message.Body != "" {
		return message.Body
	}
	if message.FileName != "" {
		return message.FileName
	}
	return message.LinkURL
}

func attachmentError(err error) (int, string) {
	switch {
	case errors.Is(err, upload.ErrTooLarge):
		return http.StatusRequestEntityTooLarge, MessageFileTooLarge
	case errors.Is(err, upload.ErrNotAllowedType):
		return http.StatusBadRequest, MessageFileNotAllowed
	default:
		return http.StatusBadRequest, httpx.MessageInvalidBody
	}
}
