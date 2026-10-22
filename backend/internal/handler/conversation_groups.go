package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strconv"
	"strings"
)

const (
	MessageGroupCreated     = "Group created."
	MessageGroupRenamed     = "Group renamed."
	MessageGroupLeft        = "You left the group."
	MessageMembersAdded     = "Members added."
	MessageMemberRemoved    = "Member removed."
	MessageMemberPromoted   = "Member is now a group admin."
	MessageMemberDemoted    = "Member is no longer a group admin."
	MessageNotAGroupAdmin   = "Only a group admin can do that."
	MessageNotAGroup        = "This conversation is not a group."
	MessageLastGroupAdmin   = "A group keeps at least one admin."
	MessageNotAGroupMember  = "This account is not in this group."
	MessageGroupNeedsPeople = "Add at least one other person to the group."
	MessageOwnBatchOnly     = "A class representative can only add their own batch."
	MessageTooManyMembers   = "A group holds at most 500 people."
	fieldGroupTitle         = "Group name"
	MaxGroupTitle           = 80
)

type batchTarget struct {
	Batch   string `json:"batch"`
	Section string `json:"section"`
}

type createGroupRequest struct {
	Title   string        `json:"title"`
	UserIDs []int64       `json:"user_ids"`
	Batches []batchTarget `json:"batches"`
}

type addMembersRequest struct {
	UserIDs []int64       `json:"user_ids"`
	Batches []batchTarget `json:"batches"`
}

type renameGroupRequest struct {
	Title string `json:"title"`
}

type memberItem struct {
	UserID      int64  `json:"user_id"`
	Name        string `json:"name"`
	Role        string `json:"role"`
	AccountRole string `json:"account_role"`
	Batch       string `json:"batch,omitempty"`
	Section     string `json:"section,omitempty"`
	Designation string `json:"designation,omitempty"`
	PhotoURL    string `json:"photo_url,omitempty"`
	AddedAt     string `json:"added_at"`
}

type membersResponse struct {
	Items []memberItem `json:"items"`
}

func canCreateGroups(role string) bool {
	return role == auth.RoleTeacher || role == auth.RoleCR
}

func (c *Conversations) StartGroup() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		viewer, ok := c.viewer(w, r)
		if !ok {
			return
		}
		if !canCreateGroups(viewer.Role) {
			httpx.WriteError(w, http.StatusForbidden, MessageGroupNotAllowed)
			return
		}
		var request createGroupRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		title, err := groupTitle(request.Title)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		members, ok := c.resolveMembers(w, r, viewer, request.UserIDs, request.Batches)
		if !ok {
			return
		}
		if len(members) == 0 {
			httpx.WriteError(w, http.StatusBadRequest, MessageGroupNeedsPeople)
			return
		}
		conversation, err := c.chats.CreateGroup(r.Context(), store.NewGroup{
			Title:     title,
			CreatedBy: viewer.UserID,
			Members:   members,
		})
		if err != nil {
			log.Printf("handler: could not create a group conversation: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		c.announceMembership(r.Context(), conversation.ID, viewer, "Added to "+title,
			viewer.Name+" started this group.")
		httpx.WriteJSON(w, http.StatusCreated, conversationResponse{
			ID:          conversation.ID,
			Kind:        conversation.Kind,
			Title:       conversation.Title,
			CanPost:     true,
			MemberRole:  store.MemberAdmin,
			MemberCount: len(members) + 1,
			Message:     MessageGroupCreated,
		})
	}
}

func (c *Conversations) Members() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		viewer, ok := c.viewer(w, r)
		if !ok {
			return
		}
		access, ok := c.access(w, r, viewer)
		if !ok {
			return
		}
		members, err := c.chats.ListConversationMembers(r.Context(), access.ID)
		if err != nil {
			log.Printf("handler: could not list the group members: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]memberItem, 0, len(members))
		for _, member := range members {
			items = append(items, memberItem{
				UserID:      member.UserID,
				Name:        member.Name,
				Role:        member.Role,
				AccountRole: member.AccountRole,
				Batch:       member.Batch,
				Section:     member.Section,
				Designation: member.Designation,
				PhotoURL:    member.PhotoPath,
				AddedAt:     timestamp(member.AddedAt),
			})
		}
		httpx.WriteJSON(w, http.StatusOK, membersResponse{Items: items})
	}
}

func (c *Conversations) AddMembers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		viewer, access, ok := c.groupAdmin(w, r)
		if !ok {
			return
		}
		var request addMembersRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		members, ok := c.resolveMembers(w, r, viewer, request.UserIDs, request.Batches)
		if !ok {
			return
		}
		if len(members) == 0 {
			httpx.WriteError(w, http.StatusBadRequest, MessageGroupNeedsPeople)
			return
		}
		if access.MemberCount+len(members) > store.MaxGroupMembers {
			httpx.WriteError(w, http.StatusBadRequest, MessageTooManyMembers)
			return
		}
		added, err := c.chats.AddConversationMembers(r.Context(), access.ID, viewer.UserID, members)
		if err != nil {
			log.Printf("handler: could not add group members: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if added > 0 {
			c.notifyMembers(r.Context(), members, viewer, "Added to "+access.Title,
				viewer.Name+" added you to this group.", access.ID)
		}
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageMembersAdded})
	}
}

func (c *Conversations) RemoveMember() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, access, ok := c.groupAdmin(w, r)
		if !ok {
			return
		}
		userID, ok := memberPathID(w, r)
		if !ok {
			return
		}
		err := c.chats.RemoveConversationMember(r.Context(), access.ID, userID)
		switch {
		case errors.Is(err, store.ErrNotAMember):
			httpx.WriteError(w, http.StatusNotFound, MessageNotAGroupMember)
		case errors.Is(err, store.ErrLastAdmin):
			httpx.WriteError(w, http.StatusConflict, MessageLastGroupAdmin)
		case err != nil:
			log.Printf("handler: could not remove a group member: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageMemberRemoved})
		}
	}
}

func (c *Conversations) SetMemberAdmin(admin bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, access, ok := c.groupAdmin(w, r)
		if !ok {
			return
		}
		userID, ok := memberPathID(w, r)
		if !ok {
			return
		}
		err := c.chats.SetConversationMemberAdmin(r.Context(), access.ID, userID, admin)
		switch {
		case errors.Is(err, store.ErrNotAMember):
			httpx.WriteError(w, http.StatusNotFound, MessageNotAGroupMember)
		case errors.Is(err, store.ErrLastAdmin):
			httpx.WriteError(w, http.StatusConflict, MessageLastGroupAdmin)
		case err != nil:
			log.Printf("handler: could not change a group role: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		case admin:
			httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageMemberPromoted})
		default:
			httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageMemberDemoted})
		}
	}
}

func (c *Conversations) Rename() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, access, ok := c.groupAdmin(w, r)
		if !ok {
			return
		}
		var request renameGroupRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		title, err := groupTitle(request.Title)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		err = c.chats.RenameConversation(r.Context(), access.ID, title)
		switch {
		case errors.Is(err, store.ErrNotAGroup):
			httpx.WriteError(w, http.StatusBadRequest, MessageNotAGroup)
		case err != nil:
			log.Printf("handler: could not rename a group: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusOK, conversationResponse{
				ID:      access.ID,
				Kind:    access.Kind,
				Title:   title,
				CanPost: true,
				Message: MessageGroupRenamed,
			})
		}
	}
}

func (c *Conversations) Leave() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		viewer, ok := c.viewer(w, r)
		if !ok {
			return
		}
		access, ok := c.access(w, r, viewer)
		if !ok {
			return
		}
		if access.Kind != store.ConversationGroup {
			httpx.WriteError(w, http.StatusBadRequest, MessageNotAGroup)
			return
		}
		err := c.chats.LeaveConversation(r.Context(), access.ID, viewer.UserID)
		switch {
		case errors.Is(err, store.ErrNotAMember):
			httpx.WriteError(w, http.StatusNotFound, MessageNotAGroupMember)
		case err != nil:
			log.Printf("handler: could not leave a group: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageGroupLeft})
		}
	}
}

func (c *Conversations) groupAdmin(w http.ResponseWriter, r *http.Request) (store.MessageViewer,
	store.ConversationAccess, bool) {
	viewer, ok := c.viewer(w, r)
	if !ok {
		return store.MessageViewer{}, store.ConversationAccess{}, false
	}
	access, ok := c.access(w, r, viewer)
	if !ok {
		return store.MessageViewer{}, store.ConversationAccess{}, false
	}
	if access.Kind != store.ConversationGroup {
		httpx.WriteError(w, http.StatusBadRequest, MessageNotAGroup)
		return store.MessageViewer{}, store.ConversationAccess{}, false
	}
	if !access.IsAdmin() {
		httpx.WriteError(w, http.StatusForbidden, MessageNotAGroupAdmin)
		return store.MessageViewer{}, store.ConversationAccess{}, false
	}
	return viewer, access, true
}

func (c *Conversations) resolveMembers(w http.ResponseWriter, r *http.Request, viewer store.MessageViewer,
	userIDs []int64, batches []batchTarget) ([]int64, bool) {
	seen := make(map[int64]bool)
	members := make([]int64, 0, len(userIDs))
	if len(userIDs) > 0 {
		active, err := c.chats.ActiveUserIDs(r.Context(), userIDs)
		if err != nil {
			log.Printf("handler: could not check the group members: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return nil, false
		}
		if len(active) != len(distinctIDs(userIDs)) {
			httpx.WriteError(w, http.StatusNotFound, MessageUnknownContact)
			return nil, false
		}
		for _, id := range active {
			if id == viewer.UserID || seen[id] {
				continue
			}
			seen[id] = true
			members = append(members, id)
		}
	}
	for _, target := range batches {
		batch, err := validate.Batch("batch", target.Batch)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return nil, false
		}
		section := ""
		if strings.TrimSpace(target.Section) != "" {
			section, err = validate.Section(fieldSection, target.Section)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return nil, false
			}
		}
		if viewer.Role == auth.RoleCR && batch != viewer.Batch {
			httpx.WriteError(w, http.StatusForbidden, MessageOwnBatchOnly)
			return nil, false
		}
		found, err := c.chats.BatchMembers(r.Context(), batch, section)
		if err != nil {
			log.Printf("handler: could not list a batch for a group: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return nil, false
		}
		for _, id := range found {
			if id == viewer.UserID || seen[id] {
				continue
			}
			seen[id] = true
			members = append(members, id)
		}
	}
	if len(members) >= store.MaxGroupMembers {
		httpx.WriteError(w, http.StatusBadRequest, MessageTooManyMembers)
		return nil, false
	}
	return members, true
}

func (c *Conversations) announceMembership(ctx context.Context, conversationID int64,
	viewer store.MessageViewer, title, body string) {
	recipients, err := c.chats.ConversationRecipients(ctx, conversationID, viewer.UserID)
	if err != nil {
		log.Printf("handler: could not find the group members to notify: %v", err)
		return
	}
	c.notifyMembers(ctx, recipients, viewer, title, body, conversationID)
}

func groupTitle(raw string) (string, error) {
	title, err := validate.ShortText(fieldGroupTitle, raw, MaxGroupTitle)
	if err != nil {
		return "", err
	}
	if title == "" {
		return "", errors.New(fieldGroupTitle + " is required.")
	}
	return title, nil
}

func memberPathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("user_id")), 10, 64)
	if err != nil || userID <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, MessageInvalidUserID)
		return 0, false
	}
	return userID, true
}

func distinctIDs(raw []int64) []int64 {
	seen := make(map[int64]bool, len(raw))
	ids := make([]int64, 0, len(raw))
	for _, id := range raw {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}
