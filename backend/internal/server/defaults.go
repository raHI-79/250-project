package server

import (
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/settings"
	"strconv"
	"strings"
)

func SettingDefaults(cfg *config.Config) map[string]string {
	values := map[string]string{
		settings.KeyMaintenance:        settings.Off,
		settings.KeyMaintenanceMessage: "",
		settings.KeyRegistration:       settings.On,
		settings.KeyRegistrationNote:   "",
		settings.KeyAnnouncement:       "",
		settings.KeyAnnouncementRoles:  "",
		settings.KeyAnnouncementUntil:  "",
		settings.KeyAnnouncementTone:   "info",
		settings.KeyAttendance:         settings.On,
		settings.KeyPolls:              settings.On,
		settings.KeyMessages:           settings.On,
		settings.KeyGroups:             settings.On,
		settings.KeyWebAttendance:      settings.On,
		settings.KeyOffDays:            settings.DefaultOffDays,
		settings.KeyMinAppVersion:      "",
		settings.KeyRetainViews:        "90",
		settings.KeyRetainEvents:       "365",
		settings.KeyRetainNotices:      "120",
		settings.KeyLateWeight:         "0.5",
	}
	if cfg == nil {
		return values
	}
	values[settings.KeyReminderTimes] = strings.Join(cfg.ReminderTimes, ",")
	values[settings.KeyMorningReminder] = cfg.MorningReminder
	values[settings.KeyDaysAhead] = strconv.Itoa(cfg.DaysAhead)
	return values
}
