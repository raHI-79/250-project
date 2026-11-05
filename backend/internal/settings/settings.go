package settings

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	KeyMaintenance        = "maintenance_mode"
	KeyMaintenanceMessage = "maintenance_message"
	KeyRegistration       = "registration_open"
	KeyRegistrationNote   = "registration_message"
	KeyAnnouncement       = "announcement_text"
	KeyAnnouncementRoles  = "announcement_roles"
	KeyAnnouncementUntil  = "announcement_until"
	KeyAnnouncementTone   = "announcement_severity"
	KeyAttendance         = "feature_attendance"
	KeyPolls              = "feature_polls"
	KeyMessages           = "feature_messages"
	KeyGroups             = "feature_group_chats"
	KeyWebAttendance      = "feature_web_attendance"
	KeyReminderTimes      = "reminder_times"
	KeyMorningReminder    = "morning_reminder"
	KeyDaysAhead          = "schedule_days_ahead"
	KeyMinAppVersion      = "min_app_version"

	On          = "on"
	Off         = "off"
	CacheWindow = 30 * time.Second
)

var editable = []string{
	KeyMaintenance,
	KeyMaintenanceMessage,
	KeyRegistration,
	KeyRegistrationNote,
	KeyAnnouncement,
	KeyAnnouncementRoles,
	KeyAnnouncementUntil,
	KeyAnnouncementTone,
	KeyAttendance,
	KeyPolls,
	KeyMessages,
	KeyGroups,
	KeyWebAttendance,
	KeyReminderTimes,
	KeyMorningReminder,
	KeyDaysAhead,
	KeyMinAppVersion,
}

var features = []string{KeyAttendance, KeyPolls, KeyMessages, KeyGroups, KeyWebAttendance}

func Editable() []string {
	names := make([]string, len(editable))
	copy(names, editable)
	return names
}

func Features() []string {
	names := make([]string, len(features))
	copy(names, features)
	return names
}

func IsEditable(key string) bool {
	for _, name := range editable {
		if name == key {
			return true
		}
	}
	return false
}

type Store interface {
	AllSettings(ctx context.Context) (map[string]string, error)
	SaveSettings(ctx context.Context, values map[string]string, by int64) error
}

type Cache struct {
	store    Store
	defaults map[string]string
	ttl      time.Duration
	now      func() time.Time
	mu       sync.RWMutex
	values   map[string]string
	readAt   time.Time
}

func New(store Store, defaults map[string]string) *Cache {
	kept := make(map[string]string, len(defaults))
	for key, value := range defaults {
		kept[key] = value
	}
	return &Cache{store: store, defaults: kept, ttl: CacheWindow, now: time.Now}
}

func (c *Cache) load(ctx context.Context) map[string]string {
	c.mu.RLock()
	fresh := c.values != nil && c.now().Sub(c.readAt) < c.ttl
	values := c.values
	c.mu.RUnlock()
	if fresh {
		return values
	}
	if c.store == nil {
		return c.defaults
	}
	stored, err := c.store.AllSettings(ctx)
	if err != nil {
		if values != nil {
			return values
		}
		return c.defaults
	}
	merged := make(map[string]string, len(c.defaults)+len(stored))
	for key, value := range c.defaults {
		merged[key] = value
	}
	for key, value := range stored {
		if strings.TrimSpace(value) != "" {
			merged[key] = value
		}
	}
	c.mu.Lock()
	c.values = merged
	c.readAt = c.now()
	c.mu.Unlock()
	return merged
}

func (c *Cache) Forget() {
	c.mu.Lock()
	c.values = nil
	c.mu.Unlock()
}

func (c *Cache) All(ctx context.Context) map[string]string {
	loaded := c.load(ctx)
	copied := make(map[string]string, len(loaded))
	for key, value := range loaded {
		copied[key] = value
	}
	return copied
}

func (c *Cache) Get(ctx context.Context, key string) string {
	return c.load(ctx)[key]
}

func (c *Cache) On(ctx context.Context, key string) bool {
	value := strings.ToLower(strings.TrimSpace(c.Get(ctx, key)))
	return value == On || value == "true" || value == "1" || value == "yes"
}

func (c *Cache) Number(ctx context.Context, key string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(c.Get(ctx, key)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func (c *Cache) List(ctx context.Context, key string) []string {
	raw := strings.Split(c.Get(ctx, key), ",")
	kept := make([]string, 0, len(raw))
	for _, entry := range raw {
		trimmed := strings.TrimSpace(entry)
		if trimmed != "" {
			kept = append(kept, trimmed)
		}
	}
	return kept
}

func (c *Cache) Save(ctx context.Context, values map[string]string, by int64) error {
	kept := make(map[string]string, len(values))
	for key, value := range values {
		if IsEditable(key) {
			kept[key] = strings.TrimSpace(value)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	if err := c.store.SaveSettings(ctx, kept, by); err != nil {
		return err
	}
	c.Forget()
	return nil
}

func (c *Cache) Maintenance(ctx context.Context) (bool, string) {
	if !c.On(ctx, KeyMaintenance) {
		return false, ""
	}
	message := strings.TrimSpace(c.Get(ctx, KeyMaintenanceMessage))
	if message == "" {
		message = "The portal is closed for maintenance. Please try again later."
	}
	return true, message
}

func (c *Cache) Registration(ctx context.Context) (bool, string) {
	open := c.On(ctx, KeyRegistration)
	message := strings.TrimSpace(c.Get(ctx, KeyRegistrationNote))
	if !open && message == "" {
		message = "Registration is closed right now. Please contact the department office."
	}
	return open, message
}
