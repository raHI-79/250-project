package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/release"
	"github.com/raHI-79/250-project/backend/internal/settings"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	MessageSettingsSaved = "Settings saved."
	MessageBlockAdded    = "The address is blocked."
	MessageBlockGone     = "The address is open again."
	MessageBlockBadRange = "Give an address like 203.0.113.4 or a range like 203.0.113.0/24."
	MessageUnlocked      = "The account can sign in again."
	MessageUpdateSent    = "The update notification is on its way."
	MessageNoRelease     = "No Android release was found."
	maxBlockHours        = 24 * 365
)

type ControlStore interface {
	ListIPBlocks(ctx context.Context) ([]store.IPBlock, error)
	BlockIP(ctx context.Context, cidr, reason string, by int64, lifetime time.Duration) (int64, error)
	UnblockIP(ctx context.Context, blockID int64) (string, error)
	LockedAccounts(ctx context.Context) ([]store.LockedAccount, error)
	ClearFailures(ctx context.Context, userID int64) error
	FailuresByAddress(ctx context.Context, since time.Time) ([]store.FailureCount, error)
	FailuresByAccount(ctx context.Context, since time.Time) ([]store.FailureCount, error)
	ListEvents(ctx context.Context, filter store.AuditFilter) ([]store.AuditEntry, error)
}

type ReleaseAnnouncer interface {
	Announce(ctx context.Context, version string) (int, error)
}

type DataStore interface {
	DataCounts(ctx context.Context) (store.DataCounts, error)
	BackupStatus(ctx context.Context) (store.BackupState, error)
}

type Control struct {
	cfg     *config.Config
	data    DataStore
	values  *settings.Cache
	guards  ControlStore
	blocks  *Blocks
	apps    ReleaseLookup
	notices ReleaseAnnouncer
	events  *Recorder
}

func NewControl(cfg *config.Config, values *settings.Cache, guards ControlStore, blocks *Blocks,
	apps ReleaseLookup, notices ReleaseAnnouncer, events *Recorder) *Control {
	return &Control{cfg: cfg, values: values, guards: guards, blocks: blocks, apps: apps,
		notices: notices, events: events}
}

func (c *Control) WithData(data DataStore) *Control {
	c.data = data
	return c
}

type settingsResponse struct {
	Values   map[string]string `json:"values"`
	Editable []string          `json:"editable"`
	Features []string          `json:"features"`
}

type settingsRequest struct {
	Values map[string]string `json:"values"`
}

type blockItem struct {
	ID        string `json:"id"`
	CIDR      string `json:"cidr"`
	Reason    string `json:"reason,omitempty"`
	CreatedBy string `json:"created_by,omitempty"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type lockedItem struct {
	ID          int64  `json:"id"`
	Identifier  string `json:"identifier"`
	Role        string `json:"role"`
	Failures    int    `json:"failures"`
	LockedUntil string `json:"locked_until,omitempty"`
}

type countItem struct {
	Label string `json:"label"`
	Total int    `json:"total"`
}

type securityResponse struct {
	Addresses []countItem    `json:"addresses"`
	Accounts  []countItem    `json:"accounts"`
	Locked    []lockedItem   `json:"locked"`
	Blocks    []blockItem    `json:"blocks"`
	Events    []activityItem `json:"events"`
}

type blockRequest struct {
	CIDR   string `json:"cidr"`
	Reason string `json:"reason"`
	Hours  int    `json:"hours"`
}

type releaseResponse struct {
	Version    string `json:"version,omitempty"`
	Notes      string `json:"notes,omitempty"`
	Downloads  int    `json:"downloads"`
	ShareLink  string `json:"share_link"`
	MinVersion string `json:"min_version,omitempty"`
}

func (c *Control) Settings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, settingsResponse{
			Values:   c.values.All(r.Context()),
			Editable: settings.Editable(),
			Features: settings.Features(),
		})
	}
}

func (c *Control) SaveSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request settingsRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		if len(request.Values) == 0 {
			httpx.WriteError(w, http.StatusBadRequest, "Nothing to save.")
			return
		}
		for key := range request.Values {
			if !settings.IsEditable(key) {
				httpx.WriteError(w, http.StatusBadRequest, "Unknown setting "+key+".")
				return
			}
		}
		current, _ := middleware.CurrentUser(r.Context())
		if err := c.values.Save(r.Context(), request.Values, current.ID); err != nil {
			log.Printf("handler: could not save the settings: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		names := make([]string, 0, len(request.Values))
		for key := range request.Values {
			names = append(names, key)
		}
		c.events.Record(r, store.ActionSettings, "settings", "", "Saved "+strings.Join(names, ", "))
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageSettingsSaved})
	}
}

func (c *Control) Security() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		since := c.cfg.Now().Add(-7 * 24 * time.Hour)
		addresses, err := c.guards.FailuresByAddress(r.Context(), since)
		if err != nil {
			c.fail(w, err)
			return
		}
		accounts, err := c.guards.FailuresByAccount(r.Context(), since)
		if err != nil {
			c.fail(w, err)
			return
		}
		locked, err := c.guards.LockedAccounts(r.Context())
		if err != nil {
			c.fail(w, err)
			return
		}
		blocks, err := c.guards.ListIPBlocks(r.Context())
		if err != nil {
			c.fail(w, err)
			return
		}
		events, err := c.guards.ListEvents(r.Context(), store.AuditFilter{
			Action: store.ActionSignInFailed,
			Limit:  20,
		})
		if err != nil {
			c.fail(w, err)
			return
		}
		body := securityResponse{
			Addresses: counts(addresses),
			Accounts:  counts(accounts),
			Locked:    make([]lockedItem, 0, len(locked)),
			Blocks:    make([]blockItem, 0, len(blocks)),
			Events:    activityItems(events),
		}
		for _, account := range locked {
			item := lockedItem{
				ID:         account.ID,
				Identifier: account.Identifier,
				Role:       account.Role,
				Failures:   account.Failures,
			}
			if account.LockedUntil != nil {
				item.LockedUntil = account.LockedUntil.Format(time.RFC3339)
			}
			body.Locked = append(body.Locked, item)
		}
		for _, block := range blocks {
			item := blockItem{
				ID:        strconv.FormatInt(block.ID, 10),
				CIDR:      block.CIDR,
				Reason:    block.Reason,
				CreatedBy: block.CreatedBy,
				CreatedAt: block.CreatedAt.Format(time.RFC3339),
			}
			if block.ExpiresAt != nil {
				item.ExpiresAt = block.ExpiresAt.Format(time.RFC3339)
			}
			body.Blocks = append(body.Blocks, item)
		}
		httpx.WriteJSON(w, http.StatusOK, body)
	}
}

func (c *Control) Block() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request blockRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		cidr := strings.TrimSpace(request.CIDR)
		if !validRange(cidr) {
			httpx.WriteError(w, http.StatusBadRequest, MessageBlockBadRange)
			return
		}
		var lifetime time.Duration
		if request.Hours > 0 {
			if request.Hours > maxBlockHours {
				httpx.WriteError(w, http.StatusBadRequest, "That is too long.")
				return
			}
			lifetime = time.Duration(request.Hours) * time.Hour
		}
		current, _ := middleware.CurrentUser(r.Context())
		if _, err := c.guards.BlockIP(r.Context(), cidr, strings.TrimSpace(request.Reason),
			current.ID, lifetime); err != nil {
			log.Printf("handler: could not block an address: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		c.blocks.Forget()
		c.events.Record(r, store.ActionBlockIP, "address", cidr, "Blocked "+cidr)
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageBlockAdded})
	}
}

func (c *Control) Unblock() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		blockID, ok := pathID(w, r)
		if !ok {
			return
		}
		cidr, err := c.guards.UnblockIP(r.Context(), blockID)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, MessageBlockGone)
			return
		}
		if err != nil {
			log.Printf("handler: could not unblock an address: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		c.blocks.Forget()
		c.events.Record(r, store.ActionUnblockIP, "address", cidr, "Unblocked "+cidr)
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageBlockGone})
	}
}

func (c *Control) Unlock() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := pathID(w, r)
		if !ok {
			return
		}
		if err := c.guards.ClearFailures(r.Context(), userID); err != nil {
			log.Printf("handler: could not unlock an account: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		c.events.Record(r, store.ActionUnlock, "account", idText(userID),
			"Unlocked account "+idText(userID))
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageUnlocked})
	}
}

func (c *Control) Release() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body := releaseResponse{
			ShareLink:  strings.TrimRight(c.cfg.FrontendOrigin, "/") + DownloadAndroidPath,
			MinVersion: c.values.Get(r.Context(), settings.KeyMinAppVersion),
		}
		if c.apps != nil {
			details, err := c.apps.Latest(r.Context())
			if err == nil {
				body.Version = details.Version
				body.Notes = details.Notes
				body.Downloads = details.Downloads
			}
		}
		httpx.WriteJSON(w, http.StatusOK, body)
	}
}

func (c *Control) AnnounceRelease() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c.apps == nil || c.notices == nil {
			httpx.WriteError(w, http.StatusConflict, MessageNoRelease)
			return
		}
		details, err := c.apps.Latest(r.Context())
		if err != nil || strings.TrimSpace(details.Version) == "" {
			httpx.WriteError(w, http.StatusConflict, MessageNoRelease)
			return
		}
		reached, err := c.notices.Announce(r.Context(), details.Version)
		if err != nil {
			log.Printf("handler: could not announce a release: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		c.events.Record(r, store.ActionAnnounce, "release", details.Version,
			"Announced "+details.Version+" to "+strconv.Itoa(reached)+" devices")
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageUpdateSent})
	}
}

func (c *Control) fail(w http.ResponseWriter, err error) {
	log.Printf("handler: could not read the security monitor: %v", err)
	httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
}

func counts(rows []store.FailureCount) []countItem {
	items := make([]countItem, 0, len(rows))
	for _, row := range rows {
		label := row.Label
		if len(label) > 12 {
			label = label[:12]
		}
		items = append(items, countItem{Label: label, Total: row.Total})
	}
	return items
}

func validRange(value string) bool {
	if value == "" {
		return false
	}
	if _, _, err := net.ParseCIDR(value); err == nil {
		return true
	}
	return net.ParseIP(value) != nil
}

func announcementFor(ctx context.Context, values *settings.Cache, role string) (string, string) {
	text := strings.TrimSpace(values.Get(ctx, settings.KeyAnnouncement))
	if text == "" {
		return "", ""
	}
	until := strings.TrimSpace(values.Get(ctx, settings.KeyAnnouncementUntil))
	if until != "" {
		ends, err := time.Parse("2006-01-02", until)
		if err == nil && time.Now().After(ends.AddDate(0, 0, 1)) {
			return "", ""
		}
	}
	roles := values.List(ctx, settings.KeyAnnouncementRoles)
	if len(roles) > 0 && role != "" {
		seen := false
		for _, entry := range roles {
			if entry == role || entry == "all" {
				seen = true
				break
			}
		}
		if !seen {
			return "", ""
		}
	}
	tone := strings.TrimSpace(values.Get(ctx, settings.KeyAnnouncementTone))
	if tone == "" {
		tone = "info"
	}
	return text, tone
}

var _ = auth.RoleAdmin

var _ = release.Details{}

type dataCountsResponse struct {
	Views         int64          `json:"views"`
	Events        int64          `json:"events"`
	Notifications int64          `json:"notifications"`
	Sessions      int64          `json:"sessions"`
	BackupAt      string         `json:"backup_at,omitempty"`
	BackupNote    string         `json:"backup_note,omitempty"`
	Backups       []activityItem `json:"backups"`
	Retention     map[string]int `json:"retention"`
}

func (c *Control) Data() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		counts, err := c.data.DataCounts(r.Context())
		if err != nil {
			c.fail(w, err)
			return
		}
		state, err := c.data.BackupStatus(r.Context())
		if err != nil {
			c.fail(w, err)
			return
		}
		events, err := c.guards.ListEvents(r.Context(), store.AuditFilter{
			Action: store.ActionBackup,
			Limit:  10,
		})
		if err != nil {
			c.fail(w, err)
			return
		}
		body := dataCountsResponse{
			Views:         counts.Views,
			Events:        counts.Events,
			Notifications: counts.Notifications,
			Sessions:      counts.Sessions,
			Backups:       activityItems(events),
			Retention: map[string]int{
				"analytics":     c.values.Number(r.Context(), settings.KeyRetainViews, 90),
				"audit":         c.values.Number(r.Context(), settings.KeyRetainEvents, 365),
				"notifications": c.values.Number(r.Context(), settings.KeyRetainNotices, 120),
			},
		}
		if state.LastSuccessAt != nil {
			body.BackupAt = state.LastSuccessAt.Format(time.RFC3339)
			body.BackupNote = state.Note
		}
		httpx.WriteJSON(w, http.StatusOK, body)
	}
}
