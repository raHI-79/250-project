package handler

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/sitecontent"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/upload"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	MessageSiteDraftSaved = "The draft is saved. Publish it to make it live."
	MessageSitePublished  = "The website is published."
	MessageSiteRestored   = "The earlier version is live again."
	MessageSiteNoDraft    = "There is no draft to publish. Save your changes first."
	MessageSiteNoVersion  = "That version does not exist."
	siteCacheWindow       = 30 * time.Second
	siteVersionLimit      = 50
)

type SiteStore interface {
	SaveSiteDraft(ctx context.Context, content []byte, userID int64) error
	SiteDraft(ctx context.Context) (store.SiteVersion, error)
	SitePublished(ctx context.Context) (store.SiteVersion, error)
	SiteVersion(ctx context.Context, id int64) (store.SiteVersion, error)
	PublishSite(ctx context.Context, content []byte, userID int64, note string) (int64, error)
	ListSiteVersions(ctx context.Context, limit int) ([]store.SiteVersion, error)
}

type SiteImages interface {
	SaveSiteImage(r *http.Request) (string, error)
}

type siteCredit struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Path  string `json:"path"`
}

type siteBody struct {
	Content   sitecontent.Content `json:"content"`
	Version   int64               `json:"version"`
	SavedAt   string              `json:"saved_at,omitempty"`
	SavedBy   string              `json:"saved_by,omitempty"`
	Fallback  bool                `json:"fallback"`
	Draft     bool                `json:"draft"`
	Developer siteCredit          `json:"developer"`
}

type siteVersionItem struct {
	ID      int64  `json:"id"`
	SavedAt string `json:"saved_at"`
	SavedBy string `json:"saved_by"`
	Note    string `json:"note,omitempty"`
	Live    bool   `json:"live"`
}

type siteVersionsBody struct {
	Items []siteVersionItem `json:"items"`
}

type siteImageBody struct {
	URL string `json:"url"`
}

type Site struct {
	cfg      *config.Config
	store    SiteStore
	images   SiteImages
	events   *Recorder
	mu       sync.Mutex
	cached   *siteBody
	cachedAt time.Time
}

func NewSite(cfg *config.Config, content SiteStore, images SiteImages, events *Recorder) *Site {
	return &Site{cfg: cfg, store: content, images: images, events: events}
}

func developerCredit() siteCredit {
	return siteCredit{
		Name:  sitecontent.CreditName,
		Label: sitecontent.CreditLabel,
		Path:  sitecontent.CreditPath,
	}
}

func fallbackSite() siteBody {
	return siteBody{Content: sitecontent.Default(), Fallback: true, Developer: developerCredit()}
}

func siteFrom(version store.SiteVersion) (siteBody, bool) {
	var content sitecontent.Content
	if err := json.Unmarshal(version.Content, &content); err != nil {
		return siteBody{}, false
	}
	clean, err := sitecontent.Validate(content)
	if err != nil {
		return siteBody{}, false
	}
	return siteBody{
		Content:   clean,
		Version:   version.ID,
		SavedAt:   version.SavedAt.Format(time.RFC3339),
		SavedBy:   version.SavedName,
		Draft:     version.Kind == store.SiteDraft,
		Developer: developerCredit(),
	}, true
}

func (s *Site) forget() {
	s.mu.Lock()
	s.cached = nil
	s.mu.Unlock()
}

func (s *Site) published(ctx context.Context) siteBody {
	s.mu.Lock()
	if s.cached != nil && time.Since(s.cachedAt) < siteCacheWindow {
		body := *s.cached
		s.mu.Unlock()
		return body
	}
	s.mu.Unlock()
	body := fallbackSite()
	version, err := s.store.SitePublished(ctx)
	switch {
	case err == nil:
		if saved, ok := siteFrom(version); ok {
			body = saved
		} else {
			log.Printf("handler: the published website content %d is not valid, serving the built-in content", version.ID)
		}
	case !errors.Is(err, store.ErrNotFound):
		log.Printf("handler: could not read the website content, serving the built-in content: %v", err)
		return body
	}
	s.mu.Lock()
	s.cached, s.cachedAt = &body, time.Now()
	s.mu.Unlock()
	return body
}

func (s *Site) Public() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=30")
		httpx.WriteJSON(w, http.StatusOK, s.published(r.Context()))
	}
}

func (s *Site) Draft() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		version, err := s.store.SiteDraft(r.Context())
		if err == nil {
			if body, ok := siteFrom(version); ok {
				httpx.WriteJSON(w, http.StatusOK, body)
				return
			}
		} else if !errors.Is(err, store.ErrNotFound) {
			log.Printf("handler: could not read the website draft: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		s.forget()
		httpx.WriteJSON(w, http.StatusOK, s.published(r.Context()))
	}
}

func (s *Site) SaveDraft() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, sitecontent.MaxContentSize+1))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "The website content is too large to save.")
			return
		}
		content, err := sitecontent.Parse(raw)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		clean, err := json.Marshal(content)
		if err != nil {
			log.Printf("handler: could not encode the website draft: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if err := s.store.SaveSiteDraft(r.Context(), clean, current.ID); err != nil {
			log.Printf("handler: could not save the website draft: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageSiteDraftSaved})
	}
}

func (s *Site) Publish() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		draft, err := s.store.SiteDraft(r.Context())
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusConflict, MessageSiteNoDraft)
			return
		}
		if err != nil {
			log.Printf("handler: could not read the website draft: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		content, err := sitecontent.Parse(draft.Content)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		clean, _ := json.Marshal(content)
		id, err := s.store.PublishSite(r.Context(), clean, current.ID, "")
		if err != nil {
			log.Printf("handler: could not publish the website: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		s.forget()
		s.events.Record(r, store.ActionSitePublish, "website", strconv.FormatInt(id, 10),
			"Published the website, version "+strconv.FormatInt(id, 10))
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageSitePublished})
	}
}

func (s *Site) Versions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		versions, err := s.store.ListSiteVersions(r.Context(), siteVersionLimit)
		if err != nil {
			log.Printf("handler: could not list the website versions: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		body := siteVersionsBody{Items: make([]siteVersionItem, 0, len(versions))}
		for index, version := range versions {
			body.Items = append(body.Items, siteVersionItem{
				ID:      version.ID,
				SavedAt: version.SavedAt.Format(time.RFC3339),
				SavedBy: version.SavedName,
				Note:    version.Note,
				Live:    index == 0,
			})
		}
		httpx.WriteJSON(w, http.StatusOK, body)
	}
}

func (s *Site) Restore() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			httpx.WriteError(w, http.StatusNotFound, MessageSiteNoVersion)
			return
		}
		version, err := s.store.SiteVersion(r.Context(), id)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, MessageSiteNoVersion)
			return
		}
		if err != nil {
			log.Printf("handler: could not read a website version: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		content, err := sitecontent.Parse(version.Content)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		clean, _ := json.Marshal(content)
		note := "Restored version " + strconv.FormatInt(id, 10)
		fresh, err := s.store.PublishSite(r.Context(), clean, current.ID, note)
		if err != nil {
			log.Printf("handler: could not restore a website version: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if err := s.store.SaveSiteDraft(r.Context(), clean, current.ID); err != nil {
			log.Printf("handler: could not reset the website draft: %v", err)
		}
		s.forget()
		s.events.Record(r, store.ActionSiteRestore, "website", strconv.FormatInt(fresh, 10), note)
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageSiteRestored})
	}
}

func siteImageMessage(err error) (int, string) {
	switch {
	case errors.Is(err, upload.ErrMissingImage), errors.Is(err, upload.ErrBadForm):
		return http.StatusBadRequest, "Choose an image to upload."
	case errors.Is(err, upload.ErrNotImage):
		return http.StatusBadRequest, "Only JPG, PNG and WEBP images are accepted."
	case errors.Is(err, upload.ErrImageTooLarge):
		return http.StatusRequestEntityTooLarge, "The image is larger than 4 MB."
	case errors.Is(err, upload.ErrImageTooWide):
		return http.StatusBadRequest, "The image is wider or taller than 6000 pixels."
	}
	return http.StatusInternalServerError, middleware.MessageServerError
}

func (s *Site) UploadImage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.images == nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, "Images cannot be stored right now.")
			return
		}
		address, err := s.images.SaveSiteImage(r)
		if err != nil {
			status, message := siteImageMessage(err)
			if status == http.StatusInternalServerError {
				log.Printf("handler: could not store a website image: %v", err)
			}
			httpx.WriteError(w, status, message)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, siteImageBody{URL: address})
	}
}
