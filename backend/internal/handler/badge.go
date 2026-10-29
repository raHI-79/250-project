package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	MessageSectionSeen    = "Section marked as read."
	MessageUnknownSection = "That section has no badge."
	fieldBadgeSection     = "section"
)

type BadgeStore interface {
	BadgeViewer(ctx context.Context, userID int64) (store.BadgeViewer, error)
	SectionViews(ctx context.Context, userID int64) (map[string]time.Time, error)
	MarkSectionSeen(ctx context.Context, userID int64, section string, at time.Time) error
	CountNewInSection(ctx context.Context, viewer store.BadgeViewer, section string,
		since time.Time) (int, error)
}

type Badges struct {
	cfg   *config.Config
	views BadgeStore
}

func NewBadges(cfg *config.Config, views BadgeStore) *Badges {
	return &Badges{cfg: cfg, views: views}
}

type badgesResponse struct {
	Sections map[string]int `json:"sections"`
}

func badgeRole(role string) bool {
	return role == auth.RoleStudent || role == auth.RoleCR
}

func (b *Badges) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		sections := make(map[string]int, len(store.BadgeSections()))
		for _, section := range store.BadgeSections() {
			sections[section] = 0
		}
		if !badgeRole(current.Role) {
			httpx.WriteJSON(w, http.StatusOK, badgesResponse{Sections: sections})
			return
		}
		viewer, err := b.views.BadgeViewer(r.Context(), current.ID)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteJSON(w, http.StatusOK, badgesResponse{Sections: sections})
			return
		}
		if err != nil {
			log.Printf("handler: could not load a badge viewer: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		seen, err := b.views.SectionViews(r.Context(), current.ID)
		if err != nil {
			log.Printf("handler: could not read the section views: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		for _, section := range store.BadgeSections() {
			since, ok := seen[section]
			if !ok {
				continue
			}
			count, err := b.views.CountNewInSection(r.Context(), viewer, section, since)
			if err != nil {
				log.Printf("handler: could not count the new items of %s: %v", section, err)
				continue
			}
			sections[section] = count
		}
		httpx.WriteJSON(w, http.StatusOK, badgesResponse{Sections: sections})
	}
}

func (b *Badges) Seen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		section := strings.ToLower(strings.TrimSpace(r.PathValue(fieldBadgeSection)))
		if !store.IsBadgeSection(section) {
			httpx.WriteError(w, http.StatusBadRequest, MessageUnknownSection)
			return
		}
		err := b.views.MarkSectionSeen(r.Context(), current.ID, section, b.cfg.Now())
		if errors.Is(err, store.ErrUnknownSection) {
			httpx.WriteError(w, http.StatusBadRequest, MessageUnknownSection)
			return
		}
		if err != nil {
			log.Printf("handler: could not mark a section as seen: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageSectionSeen})
	}
}

func optionalStamp(at *time.Time) string {
	if at == nil {
		return ""
	}
	return at.Format(time.RFC3339)
}
