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
	MessageFileTooLarge   = "The attached file is larger than the allowed size."
	MessageFileNotPDF     = "Only PDF files can be attached."
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
	ListStudentAdminNotices(ctx context.Context, batch string) ([]store.NoticeRow, error)
	ListCRNotices(ctx context.Context, createdBy int64) ([]store.NoticeRow, error)
	ListBatchCRNotices(ctx context.Context, batch string, otherBatches bool) ([]store.NoticeRow, error)
	DeleteAdminNotice(ctx context.Context, noticeID int64) (string, error)
}

type FileStore interface {
	Parse(r *http.Request) error
	Save(r *http.Request) (string, error)
	Delete(filePath string) error
}

type Notice struct {
	notices NoticeStore
	files   FileStore
	alerts  BroadcastNotifier
}

func NewNotice(notices NoticeStore, files FileStore, alerts BroadcastNotifier) *Notice {
	return &Notice{notices: notices, files: files, alerts: alerts}
}

type noticeItem struct {
	ID            int64   `json:"id"`
	Source        string  `json:"source"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Batch         string  `json:"batch"`
	FileURL       *string `json:"file_url"`
	CreatedByName string  `json:"created_by_name"`
	CreatedAt     string  `json:"created_at"`
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
		filePath, err := n.notices.DeleteAdminNotice(r.Context(), noticeID)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, MessageNoticeUnknown)
			return
		}
		if err != nil {
			log.Printf("handler: could not delete a notice: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if filePath != "" {
			if err := n.files.Delete(filePath); err != nil {
				log.Printf("handler: could not delete the file of a notice: %v", err)
			}
		}
		httpx.WriteJSON(w, http.StatusNoContent, nil)
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

func (n *Notice) StudentCRNotices() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := n.studentProfile(w, r)
		if !ok {
			return
		}
		scope := validate.NoticeScopeMine
		if raw := strings.TrimSpace(r.URL.Query().Get("scope")); raw != "" {
			valid, err := validate.NoticeScope("scope", raw)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			scope = valid
		}
		notices, err := n.notices.ListBatchCRNotices(r.Context(), profile.Batch, scope == validate.NoticeScopeOthers)
		n.write(w, notices, err)
	}
}

func (n *Notice) StudentAdminNotices() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := n.studentProfile(w, r)
		if !ok {
			return
		}
		notices, err := n.notices.ListStudentAdminNotices(r.Context(), profile.Batch)
		n.write(w, notices, err)
	}
}

func (n *Notice) create(w http.ResponseWriter, r *http.Request, notice store.NewNotice) {
	filePath, err := n.files.Save(r)
	if err != nil {
		writeFileError(w, err)
		return
	}
	notice.FilePath = filePath
	created, err := n.notices.CreateNotice(r.Context(), notice)
	if err != nil {
		if filePath != "" {
			if removeErr := n.files.Delete(filePath); removeErr != nil {
				log.Printf("handler: could not delete the file of a rejected notice: %v", removeErr)
			}
		}
		log.Printf("handler: could not create a notice: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return
	}
	body := created.Description
	if body == "" {
		body = created.Title
	}
	err = n.alerts.NotifyBroadcast(r.Context(), auth.RoleStudent, created.Batch, created.Section, notify.TypeNotice, created.Title, body)
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
	item := noticeItem{
		ID:            notice.ID,
		Source:        notice.Source,
		Title:         notice.Title,
		Description:   notice.Description,
		Batch:         notice.Batch,
		CreatedByName: notice.CreatedByName,
		CreatedAt:     notice.CreatedAt.Format(time.RFC3339),
	}
	if notice.FilePath != "" {
		link := notice.FilePath
		item.FileURL = &link
	}
	return item
}

func writeFileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, upload.ErrTooLarge):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, MessageFileTooLarge)
	case errors.Is(err, upload.ErrNotPDF):
		httpx.WriteError(w, http.StatusBadRequest, MessageFileNotPDF)
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
