package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/upload"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	MessageNoticePosted   = "Notice posted."
	MessageNoticeUnknown  = "No notice found with this ID."
	MessageNoticeEdited   = "Notice updated."
	MessageFileTooLarge   = "The attached file is larger than the allowed size."
	MessageOwnBatchTarget = "Use own to post a notice to your own batch."
	fieldTitle            = "title"
	fieldDescription      = "description"
	fieldBatch            = "batch"
	fieldTarget           = "target"
)

type NoticeStore interface {
	GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error)
	CreateNotice(ctx context.Context, notice store.NewNotice) (store.NoticeRow, error)
	ListAdminNotices(ctx context.Context) ([]store.NoticeRow, error)
	ListCRNotices(ctx context.Context, createdBy int64) ([]store.NoticeRow, error)
	ListTeacherNotices(ctx context.Context, createdBy int64) ([]store.NoticeRow, error)
	ListStudentNotices(ctx context.Context, batch, section, source, scope string) ([]store.NoticeRow, error)
	DeleteAdminNotice(ctx context.Context, noticeID int64) (string, error)
	DeleteAnyNotice(ctx context.Context, noticeID int64) (string, error)
	DeleteOwnNotice(ctx context.Context, noticeID int64, source string, createdBy int64) (string, error)
	EditNotice(ctx context.Context, noticeID int64, title, description, linkURL, source string,
		createdBy int64) error
}

type Notice struct {
	notices NoticeStore
	files   AttachmentStore
	alerts  BroadcastNotifier
}

func NewNotice(notices NoticeStore, files AttachmentStore, alerts BroadcastNotifier) *Notice {
	return &Notice{notices: notices, files: files, alerts: alerts}
}

type noticeItem struct {
	ID            int64  `json:"id"`
	Source        string `json:"source"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Batch         string `json:"batch"`
	Section       string `json:"section,omitempty"`
	LinkURL       string `json:"link_url,omitempty"`
	FileURL       string `json:"file_url,omitempty"`
	FileName      string `json:"file_name,omitempty"`
	FileMime      string `json:"file_mime,omitempty"`
	FileSize      int64  `json:"file_size,omitempty"`
	CreatedByName string `json:"created_by_name"`
	PostedByPhoto string `json:"posted_by_photo_url,omitempty"`
	CreatedAt     string `json:"created_at"`
	EditedAt      string `json:"edited_at,omitempty"`
}

type noticesResponse struct {
	Items []noticeItem `json:"items"`
}

type noticeResponse struct {
	noticeItem
	Message string `json:"message"`
}

func (n *Notice) AdminCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		title, description, ok := n.content(w, r)
		if !ok {
			return
		}
		batch := ""
		if raw := strings.TrimSpace(r.FormValue(fieldBatch)); raw != "" {
			valid, err := validate.Batch("Batch", raw)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			batch = valid
		}
		n.create(w, r, store.NewNotice{
			Source:      store.SourceAdmin,
			Title:       title,
			Description: description,
			Batch:       batch,
			CreatedBy:   current.ID,
		})
	}
}

func (n *Notice) AdminList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notices, err := n.notices.ListAdminNotices(r.Context())
		n.write(w, notices, err)
	}
}

func (n *Notice) AdminDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		noticeID, ok := pathID(w, r)
		if !ok {
			return
		}
		filePath, err := n.notices.DeleteAnyNotice(r.Context(), noticeID)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, MessageNoticeUnknown)
			return
		}
		if err != nil {
			log.Printf("handler: could not delete a notice: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		n.removeFile(filePath)
		httpx.WriteJSON(w, http.StatusNoContent, nil)
	}
}

type editNoticeRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	LinkURL     string `json:"link_url"`
}

func (n *Notice) edit(source string, anyOwner bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		noticeID, ok := pathID(w, r)
		if !ok {
			return
		}
		var request editNoticeRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		title, err := validate.Required("Title", request.Title)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		description, err := validate.Description("Description", request.Description)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		link, err := validate.OptionalLink("Link", request.LinkURL)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		owner := current.ID
		if anyOwner {
			owner = 0
		}
		err = n.notices.EditNotice(r.Context(), noticeID, title, description, link, source, owner)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, MessageNoticeUnknown)
			return
		}
		if err != nil {
			log.Printf("handler: could not edit a notice: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageNoticeEdited})
	}
}

func (n *Notice) AdminEdit() http.HandlerFunc {
	return n.edit("", true)
}

func (n *Notice) TeacherEdit() http.HandlerFunc {
	return n.edit(store.SourceTeacher, false)
}

func (n *Notice) CREdit() http.HandlerFunc {
	return n.edit(store.SourceCR, false)
}

func (n *Notice) deleteOwn(source string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		noticeID, ok := pathID(w, r)
		if !ok {
			return
		}
		filePath, err := n.notices.DeleteOwnNotice(r.Context(), noticeID, source, current.ID)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, MessageNoticeUnknown)
			return
		}
		if err != nil {
			log.Printf("handler: could not delete a notice: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		n.removeFile(filePath)
		httpx.WriteJSON(w, http.StatusNoContent, nil)
	}
}

func (n *Notice) CRDelete() http.HandlerFunc {
	return n.deleteOwn(store.SourceCR)
}

func (n *Notice) removeFile(filePath string) {
	if filePath == "" {
		return
	}
	if err := n.files.Delete(filePath); err != nil {
		log.Printf("handler: could not delete the file of a notice: %v", err)
	}
}

func (n *Notice) CRCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := n.crProfile(w, r)
		if !ok {
			return
		}
		title, description, ok := n.content(w, r)
		if !ok {
			return
		}
		target, err := validate.Required("Target", r.FormValue(fieldTarget))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		batch := profile.Batch
		if !strings.EqualFold(target, validate.NoticeTargetOwn) {
			batch, err = validate.Batch("Target", target)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			if batch == profile.Batch {
				httpx.WriteError(w, http.StatusBadRequest, MessageOwnBatchTarget)
				return
			}
		}
		n.create(w, r, store.NewNotice{
			Source:      store.SourceCR,
			Title:       title,
			Description: description,
			Batch:       batch,
			CreatedBy:   profile.UserID,
		})
	}
}

func (n *Notice) CRList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := n.crProfile(w, r)
		if !ok {
			return
		}
		notices, err := n.notices.ListCRNotices(r.Context(), profile.UserID)
		n.write(w, notices, err)
	}
}

func (n *Notice) TeacherCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		title, description, ok := n.content(w, r)
		if !ok {
			return
		}
		batch, err := validate.Batch("Batch", r.FormValue(fieldBatch))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		section := ""
		if raw := strings.TrimSpace(r.FormValue(fieldSection)); raw != "" {
			section, err = validate.Section("Section", raw)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
		}
		n.create(w, r, store.NewNotice{
			Source:      store.SourceTeacher,
			Title:       title,
			Description: description,
			Batch:       batch,
			Section:     section,
			CreatedBy:   current.ID,
		})
	}
}

func (n *Notice) TeacherList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		notices, err := n.notices.ListTeacherNotices(r.Context(), current.ID)
		n.write(w, notices, err)
	}
}

func (n *Notice) TeacherDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		noticeID, ok := pathID(w, r)
		if !ok {
			return
		}
		filePath, err := n.notices.DeleteOwnNotice(r.Context(), noticeID, store.SourceTeacher, current.ID)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, MessageNoticeUnknown)
			return
		}
		if err != nil {
			log.Printf("handler: could not delete a notice: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		n.removeFile(filePath)
		httpx.WriteJSON(w, http.StatusNoContent, nil)
	}
}

func (n *Notice) StudentNotices() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := n.studentProfile(w, r)
		if !ok {
			return
		}
		query := r.URL.Query()
		source := store.NoticeAll
		if raw := strings.TrimSpace(query.Get("source")); raw != "" {
			valid, err := validate.NoticeSource("source", raw)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			source = valid
		}
		scope := validate.NoticeScopeMine
		if raw := strings.TrimSpace(query.Get("scope")); raw != "" {
			valid, err := validate.NoticeScope("scope", raw)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			scope = valid
		}
		notices, err := n.notices.ListStudentNotices(r.Context(), profile.Batch, profile.Section, source, scope)
		n.write(w, notices, err)
	}
}

func (n *Notice) create(w http.ResponseWriter, r *http.Request, notice store.NewNotice) {
	link, err := validate.OptionalLink("Link", r.FormValue(fieldLink))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	attachment, err := n.files.Save(r)
	if err != nil {
		writeFileError(w, err)
		return
	}
	notice.LinkURL = link
	notice.FilePath = attachment.Path
	notice.FileName = attachment.Name
	notice.FileMime = attachment.Mime
	notice.FileSize = attachment.Size
	created, err := n.notices.CreateNotice(r.Context(), notice)
	if err != nil {
		n.removeFile(attachment.Path)
		log.Printf("handler: could not create a notice: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return
	}
	body := created.Description
	if body == "" {
		body = created.Title
	}
	err = n.alerts.NotifyBroadcast(r.Context(), auth.RoleStudent, created.Batch, created.Section,
		notify.TypeNotice, created.Title, body, notify.Ref(notify.LinkNotice, created.ID))
	if err != nil {
		log.Printf("handler: could not notify the students of a notice: %v", err)
	}
	httpx.WriteJSON(w, http.StatusCreated, noticeResponse{noticeItem: newNoticeItem(created), Message: MessageNoticePosted})
}

func (n *Notice) content(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	if err := n.files.Parse(r); err != nil {
		writeFileError(w, err)
		return "", "", false
	}
	title, err := validate.Required("Title", r.FormValue(fieldTitle))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return "", "", false
	}
	description, err := validate.Description("Description", r.FormValue(fieldDescription))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return "", "", false
	}
	return title, description, true
}

func (n *Notice) crProfile(w http.ResponseWriter, r *http.Request) (store.StudentProfile, bool) {
	return crProfile(w, r, n.notices.GetStudentProfile)
}

func (n *Notice) studentProfile(w http.ResponseWriter, r *http.Request) (store.StudentProfile, bool) {
	return studentProfile(w, r, n.notices.GetStudentProfile)
}

func (n *Notice) write(w http.ResponseWriter, notices []store.NoticeRow, err error) {
	if err != nil {
		log.Printf("handler: could not list the notices: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return
	}
	items := make([]noticeItem, 0, len(notices))
	for _, notice := range notices {
		items = append(items, newNoticeItem(notice))
	}
	httpx.WriteJSON(w, http.StatusOK, noticesResponse{Items: items})
}

func newNoticeItem(notice store.NoticeRow) noticeItem {
	return noticeItem{
		ID:            notice.ID,
		Source:        notice.Source,
		Title:         notice.Title,
		Description:   notice.Description,
		Batch:         notice.Batch,
		Section:       notice.Section,
		LinkURL:       notice.LinkURL,
		FileURL:       upload.AttachmentURL(notice.FilePath, notice.FileName),
		FileName:      notice.FileName,
		FileMime:      notice.FileMime,
		FileSize:      notice.FileSize,
		CreatedByName: notice.CreatedByName,
		PostedByPhoto: notice.CreatedByPhoto,
		CreatedAt:     notice.CreatedAt.Format(time.RFC3339),
		EditedAt:      optionalStamp(notice.EditedAt),
	}
}

func writeFileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, upload.ErrTooLarge):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, MessageFileTooLarge)
	case errors.Is(err, upload.ErrNotAllowedType):
		httpx.WriteError(w, http.StatusBadRequest, MessageFileNotAllowed)
	case errors.Is(err, upload.ErrBadForm):
		httpx.WriteError(w, http.StatusBadRequest, httpx.MessageInvalidBody)
	default:
		log.Printf("handler: could not save an attached file: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
	}
}

type profileLoader func(ctx context.Context, userID int64) (store.StudentProfile, error)

func studentProfile(w http.ResponseWriter, r *http.Request, load profileLoader) (store.StudentProfile, bool) {
	current, ok := middleware.CurrentUser(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
		return store.StudentProfile{}, false
	}
	profile, err := load(r.Context(), current.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusForbidden, MessageNotAStudent)
			return store.StudentProfile{}, false
		}
		log.Printf("handler: could not load the profile of a student: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return store.StudentProfile{}, false
	}
	if profile.UserID == 0 {
		profile.UserID = current.ID
	}
	return profile, true
}

func crProfile(w http.ResponseWriter, r *http.Request, load profileLoader) (store.StudentProfile, bool) {
	profile, ok := studentProfile(w, r, load)
	if !ok {
		return store.StudentProfile{}, false
	}
	if !profile.IsCR {
		httpx.WriteError(w, http.StatusForbidden, MessageNotACR)
		return store.StudentProfile{}, false
	}
	return profile, true
}
