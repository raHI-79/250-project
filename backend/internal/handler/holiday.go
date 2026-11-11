package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/daydate"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/settings"
	"github.com/raHI-79/250-project/backend/internal/store"
)

const (
	MessageHolidaySaved   = "Calendar entry saved."
	MessageHolidayDeleted = "Calendar entry removed."
	MessageHolidayUnknown = "No holiday or working day is set for that date."
	MessageHolidayName    = "A holiday needs a name."
	MessageHolidayFollows = "Choose the weekday a working day follows."
	MessageOffDays        = "Weekly off days must be weekday numbers from 0 (Sunday) to 6 (Saturday)."
	MessageDateFormat     = "The date must look like 2026-11-06."
	TitleHolidayCleared   = "A holiday cleared the generated classes"
)

type HolidayStore interface {
	ListHolidays(ctx context.Context, from string) ([]store.Holiday, error)
	SaveHoliday(ctx context.Context, holiday store.Holiday) ([]store.ScheduleItem, error)
	DeleteHoliday(ctx context.Context, date string) error
	ListActiveCRs(ctx context.Context, batch, section string) ([]int64, error)
}

type Holidays struct {
	cfg      *config.Config
	calendar HolidayStore
	values   *settings.Cache
	alerts   Notifier
}

func NewHolidays(cfg *config.Config, calendar HolidayStore, values *settings.Cache,
	alerts Notifier) *Holidays {
	return &Holidays{cfg: cfg, calendar: calendar, values: values, alerts: alerts}
}

type holidayRequest struct {
	Date       string `json:"date"`
	Name       string `json:"name"`
	WorkingDay bool   `json:"working_day"`
	Follows    *int   `json:"follows_weekday"`
}

type offDaysRequest struct {
	OffDays []int `json:"off_days"`
}

type holidayBody struct {
	Date        string `json:"date"`
	Label       string `json:"label"`
	Name        string `json:"name"`
	Weekday     string `json:"weekday"`
	WorkingDay  bool   `json:"working_day"`
	Follows     int    `json:"follows_weekday,omitempty"`
	FollowsName string `json:"follows_weekday_name,omitempty"`
}

type calendarResponse struct {
	OffDays     []int         `json:"off_days"`
	OffDayNames []string      `json:"off_day_names"`
	Holidays    []holidayBody `json:"holidays"`
}

type holidayResponse struct {
	Holiday holidayBody `json:"holiday"`
	Removed int         `json:"removed_items"`
	Message string      `json:"message"`
}

func newHolidayBody(holiday store.Holiday) holidayBody {
	body := holidayBody{
		Date:       holiday.Date,
		Label:      daydate.Readable(holiday.Date),
		Name:       holiday.Name,
		Weekday:    daydate.WeekdayName(holiday.Date),
		WorkingDay: holiday.WorkingDay,
	}
	if holiday.WorkingDay {
		body.Follows = holiday.Follows
		body.FollowsName = holiday.FollowsName()
	}
	return body
}

func (h *Holidays) offDays(ctx context.Context) []int {
	if h.values == nil {
		return settings.OffDayNumbers()
	}
	return h.values.Numbers(ctx, settings.KeyOffDays, settings.OffDayNumbers())
}

func (h *Holidays) OffDays(ctx context.Context) []int {
	return h.offDays(ctx)
}

func (h *Holidays) Calendar() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from := h.cfg.Today()
		if raw := strings.TrimSpace(r.URL.Query().Get("from")); raw != "" {
			parsed, err := h.cfg.ParseDate(raw)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, MessageDateFormat)
				return
			}
			from = h.cfg.DateOf(parsed)
		}
		entries, err := h.calendar.ListHolidays(r.Context(), from)
		if err != nil {
			log.Printf("handler: could not list the holidays: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		bodies := make([]holidayBody, 0, len(entries))
		for _, entry := range entries {
			bodies = append(bodies, newHolidayBody(entry))
		}
		offDays := h.offDays(r.Context())
		names := make([]string, 0, len(offDays))
		for _, day := range offDays {
			names = append(names, daydate.DayName(day))
		}
		httpx.WriteJSON(w, http.StatusOK, calendarResponse{
			OffDays:     offDays,
			OffDayNames: names,
			Holidays:    bodies,
		})
	}
}

func (h *Holidays) Save() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		var request holidayRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		parsed, err := h.cfg.ParseDate(request.Date)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, MessageDateFormat)
			return
		}
		holiday := store.Holiday{
			Date:       h.cfg.DateOf(parsed),
			Name:       strings.TrimSpace(request.Name),
			WorkingDay: request.WorkingDay,
			CreatedBy:  current.ID,
		}
		if holiday.WorkingDay {
			if request.Follows == nil || *request.Follows < 0 || *request.Follows >= daydate.Week {
				httpx.WriteError(w, http.StatusBadRequest, MessageHolidayFollows)
				return
			}
			holiday.Follows = *request.Follows
			if holiday.Name == "" {
				holiday.Name = "Working day"
			}
		} else if holiday.Name == "" {
			httpx.WriteError(w, http.StatusBadRequest, MessageHolidayName)
			return
		}
		removed, err := h.calendar.SaveHoliday(r.Context(), holiday)
		if err != nil {
			log.Printf("handler: could not save the holiday of %s: %v", holiday.Date, err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		h.tellAboutRemovals(r, holiday, removed)
		httpx.WriteJSON(w, http.StatusOK, holidayResponse{
			Holiday: newHolidayBody(holiday),
			Removed: len(removed),
			Message: MessageHolidaySaved,
		})
	}
}

func (h *Holidays) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimSpace(r.PathValue("date"))
		parsed, err := h.cfg.ParseDate(raw)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, MessageDateFormat)
			return
		}
		err = h.calendar.DeleteHoliday(r.Context(), h.cfg.DateOf(parsed))
		if errors.Is(err, store.ErrNotAHoliday) {
			httpx.WriteError(w, http.StatusNotFound, MessageHolidayUnknown)
			return
		}
		if err != nil {
			log.Printf("handler: could not delete the holiday of %s: %v", raw, err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageHolidayDeleted})
	}
}

func (h *Holidays) SaveOffDays() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		var request offDaysRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		seen := make(map[int]bool, len(request.OffDays))
		days := make([]string, 0, len(request.OffDays))
		for _, day := range request.OffDays {
			if day < 0 || day >= daydate.Week {
				httpx.WriteError(w, http.StatusBadRequest, MessageOffDays)
				return
			}
			if seen[day] {
				continue
			}
			seen[day] = true
			days = append(days, strconv.Itoa(day))
		}
		values := map[string]string{settings.KeyOffDays: strings.Join(days, ",")}
		if err := h.values.Save(r.Context(), values, current.ID); err != nil {
			log.Printf("handler: could not save the weekly off days: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageHolidaySaved})
	}
}

func (h *Holidays) tellAboutRemovals(r *http.Request, holiday store.Holiday,
	removed []store.ScheduleItem) {
	if h.alerts == nil || len(removed) == 0 {
		return
	}
	reason := holiday.Name
	if reason == "" {
		reason = "a holiday"
	}
	for _, item := range removed {
		body := strings.Join([]string{
			item.CourseName(),
			"on",
			daydate.Readable(item.ItemDate),
			"was dropped because the day is now " + reason + ".",
		}, " ")
		link := notify.OnDate(notify.LinkTeacherNextDay, item.ItemDate)
		if item.TeacherUserID > 0 {
			err := h.alerts.NotifyUser(r.Context(), item.TeacherUserID, notify.TypeRoutine,
				TitleHolidayCleared, body, link)
			if err != nil {
				log.Printf("handler: could not tell the teacher %d about a holiday: %v",
					item.TeacherUserID, err)
			}
		}
		crs, err := h.calendar.ListActiveCRs(r.Context(), item.Batch, item.Section)
		if err != nil {
			log.Printf("handler: could not list the class representatives of %s: %v", item.Batch, err)
			continue
		}
		crLink := notify.OnDate(notify.LinkCRUpcoming, item.ItemDate)
		for _, userID := range crs {
			err := h.alerts.NotifyUser(r.Context(), userID, notify.TypeRoutine,
				TitleHolidayCleared, body, crLink)
			if err != nil {
				log.Printf("handler: could not tell the class representative %d about a holiday: %v",
					userID, err)
			}
		}
	}
}
