package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
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
	MessageCannotPost          = "Only the person who created this group can post in it."
	MessageNeedsContent        = "Write something, add a link or attach a file."
	MessageUnknownContact      = "No active account found with this ID."
	MessageSelfConversation    = "You cannot start a conversation with yourself."
	MessageGroupNotAllowed     = "Your role cannot start this kind of group."
	MessageSent                = "Message sent."
	MessageFileNotAllowed      = "This file type is not allowed."
	fieldBody                  = "body"
	fieldLink                  = "link"
	fieldScope                 = "scope"
	fieldSection               = "section"
)

type ConversationStore interface {
	GetMessageViewer(ctx context.Context, userID int64) (store.MessageViewer, error)
	SearchContacts(ctx context.Context, viewerID int64, search, role string) ([]store.ContactRow, error)
	GetOrCreateDirect(ctx context.Context, viewerID, otherID int64) (store.Conversation, error)
	GetOrCreateGroup(ctx context.Context, group store.NewGroup) (store.Conversation, error)
	ListConversations(ctx context.Context, viewer store.MessageViewer, limit int) ([]store.ConversationRow, error)
	FindConversationAccess(ctx context.Context, conversationID int64, viewer store.MessageViewer) (store.ConversationAccess, error)
	ListChatMessages(ctx context.Context, conversationID int64, before *time.Time, limit int) ([]store.ChatMessageRow, error)
	CreateChatMessage(ctx context.Context, message store.NewChatMessage) (store.ChatMessageRow, error)
	MarkConversationRead(ctx context.Context, userID, conversationID int64, at time.Time) error
	CountUnreadMessages(ctx context.Context, viewer store.MessageViewer) (int, error)
	DirectRecipient(ctx context.Context, conversationID, senderID int64) (int64, error)
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
	Scope         string `json:"scope,omitempty"`
	Batch         string `json:"batch,omitempty"`
	Section       string `json:"section,omitempty"`
	OtherUserID   int64  `json:"other_user_id,omitempty"`
	AvatarURL     string `json:"photo_url"`
	CreatedBy     int64  `json:"created_by"`
	CanPost       bool   `json:"can_post"`
	Preview       string `json:"preview"`
	LastMessageAt string `json:"last_message_at,omitempty"`
	UnreadCount   int    `json:"unread_count"`
}

type conversationsResponse struct {
	Items []conversationItem `json:"items"`
}

type conversationResponse struct {
	ID      int64  `json:"id"`
	Kind    string `json:"kind"`
	Title   string `json:"title"`
	Scope   string `json:"scope,omitempty"`
	Batch   string `json:"batch,omitempty"`
	Section string `json:"section,omitempty"`
	CanPost bool   `json:"can_post"`
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

type groupRequest struct {
	Scope   string `json:"scope"`
	Batch   string `json:"batch"`
	Section string `json:"section"`
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
		contacts, err := c.chats.SearchContacts(r.Context(), viewer.UserID, search, role)
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
				Scope:       row.Scope,
				Batch:       row.Batch,
				Section:     row.Section,
				CreatedBy:   row.CreatedBy,
				CanPost:     row.Kind == store.ConversationDirect || row.CreatedBy == viewer.UserID,
				Preview:     preview(row),
				UnreadCount: row.UnreadCount,
			}
			if row.Kind == store.ConversationDirect {
				item.OtherUserID = row.OtherUserID
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
		conversation, err := c.chats.GetOrCreateDirect(r.Context(), viewer.UserID, request.UserID)
		switch {
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
			Title:   "",
			CanPost: true,
		})
	}
}

func (c *Conversations) StartGroup() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		viewer, ok := c.viewer(w, r)
		if !ok {
			return
		}
		var request groupRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		group, err := groupFor(viewer, request)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if !canStartGroup(viewer.Role, group.Scope) {
			httpx.WriteError(w, http.StatusForbidden, MessageGroupNotAllowed)
			return
		}
		conversation, err := c.chats.GetOrCreateGroup(r.Context(), group)
		if err != nil {
			log.Printf("handler: could not open a group conversation: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, conversationResponse{
			ID:      conversation.ID,
			Kind:    conversation.Kind,
			Title:   groupTitle(conversation.Scope, conversation.Batch, conversation.Section),
			Scope:   conversation.Scope,
			Batch:   conversation.Batch,
			Section: conversation.Section,
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
		messages, err := c.chats.ListChatMessages(r.Context(), access.ID, before, limit)
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
			items = append(items, chatMessageItem{
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
			})
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
		if !access.CanPost {
			httpx.WriteError(w, http.StatusForbidden, MessageCannotPost)
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
		httpx.WriteJSON(w, http.StatusCreated, chatMessageItem{
			ID:         created.ID,
			SenderID:   created.SenderID,
			SenderName: viewer.Name,
			SenderRole: viewer.Role,
			Body:       created.Body,
			LinkURL:    created.LinkURL,
			FileURL:    upload.AttachmentURL(created.FilePath, created.FileName),
			FileName:   created.FileName,
			FileMime:   created.FileMime,
			FileSize:   created.FileSize,
			CreatedAt:  timestamp(created.CreatedAt),
		})
	}
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

func (c *Conversations) announce(ctx context.Context, access store.ConversationAccess, viewer store.MessageViewer, message store.ChatMessageRow) {
	title := "New message from " + viewer.Name
	body := summary(message)
	if access.Kind == store.ConversationDirect {
		recipient, err := c.chats.DirectRecipient(ctx, access.ID, viewer.UserID)
		if err != nil {
			log.Printf("handler: could not find the message recipient: %v", err)
			return
		}
		link := notify.Ref(notify.LinkMessage, access.ID)
		if err := c.alerts.NotifyUser(ctx, recipient, notify.TypeMessage, title, body, link); err != nil {
			log.Printf("handler: could not notify a message recipient: %v", err)
		}
		return
	}
	role, batch, section := groupAudience(access.Scope, access.Batch, access.Section)
	link := notify.Ref(notify.LinkMessage, access.ID)
	if err := c.alerts.NotifyBroadcast(ctx, role, batch, section, notify.TypeMessage, title, body, link); err != nil {
		log.Printf("handler: could not notify a message group: %v", err)
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

func groupFor(viewer store.MessageViewer, request groupRequest) (store.NewGroup, error) {
	scope, err := validate.ConversationScope(fieldScope, request.Scope)
	if err != nil {
		return store.NewGroup{}, err
	}
	group := store.NewGroup{Scope: scope, CreatedBy: viewer.UserID}
	switch scope {
	case store.ScopeBatch, store.ScopeSection:
		batch := strings.TrimSpace(request.Batch)
		if batch == "" && viewer.Batch != "" {
			batch = viewer.Batch
		}
		checked, err := validate.Batch("batch", batch)
		if err != nil {
			return store.NewGroup{}, err
		}
		group.Batch = checked
		if scope == store.ScopeSection {
			section := strings.TrimSpace(request.Section)
			if section == "" && viewer.Section != "" {
				section = viewer.Section
			}
			checkedSection, err := validate.Section("section", section)
			if err != nil {
				return store.NewGroup{}, err
			}
			group.Section = checkedSection
		}
	}
	return group, nil
}

func canStartGroup(role, scope string) bool {
	switch role {
	case auth.RoleTeacher:
		return scope == store.ScopeBatch || scope == store.ScopeSection
	case auth.RoleCR:
		return scope == store.ScopeBatch || scope == store.ScopeSection
	case auth.RoleAdmin:
		return true
	case auth.RoleStudent:
		return scope == store.ScopeBatch
	default:
		return false
	}
}

func groupAudience(scope, batch, section string) (string, string, string) {
	switch scope {
	case store.ScopeBatch:
		return auth.RoleStudent, batch, ""
	case store.ScopeSection:
		return auth.RoleStudent, batch, section
	case store.ScopeAllStudents:
		return auth.RoleStudent, "", ""
	case store.ScopeAllTeachers:
		return auth.RoleTeacher, "", ""
	default:
		return "", batch, section
	}
}

func conversationTitle(row store.ConversationRow) string {
	if row.Kind == store.ConversationDirect {
		return row.OtherName
	}
	return groupTitle(row.Scope, row.Batch, row.Section)
}

func groupTitle(scope, batch, section string) string {
	switch scope {
	case store.ScopeBatch:
		return batch
	case store.ScopeSection:
		return batch + " · Section " + section
	case store.ScopeAllStudents:
		return "All students"
	case store.ScopeAllTeachers:
		return "All teachers"
	default:
		return batch
	}
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
