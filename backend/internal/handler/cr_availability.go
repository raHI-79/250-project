package handler

import (
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strings"
)

const (
	LabelYours       = "yours"
	LabelUnavailable = "unavailable"
)

type bookingItem struct {
	Room       string `json:"room"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	Batch      string `json:"batch"`
	Section    string `json:"section"`
	CourseCode string `json:"course_code"`
	Source     string `json:"source"`
	Status     string `json:"status"`
	Label      string `json:"label"`
}

type availabilityResponse struct {
	Date  string        `json:"date"`
	Room  string        `json:"room,omitempty"`
	Items []bookingItem `json:"items"`
}

func (c *CR) RoomAvailability() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := c.profile(w, r)
		if !ok {
			return
		}
		query := r.URL.Query()
		raw := strings.TrimSpace(query.Get("date"))
		if raw == "" {
			httpx.WriteError(w, http.StatusBadRequest, MessageDateRequired)
			return
		}
		date, err := validate.Date("date", raw, c.cfg.Location)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		room := strings.TrimSpace(query.Get("room"))
		day := validate.DateString(date)
		bookings, err := c.classes.ListRoomBookings(r.Context(), day, room)
		if err != nil {
			log.Printf("handler: could not list the room bookings: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]bookingItem, 0, len(bookings))
		for _, booking := range bookings {
			items = append(items, bookingItem{
				Room:       booking.Room,
				StartTime:  booking.StartTime,
				EndTime:    booking.EndTime,
				Batch:      booking.Batch,
				Section:    booking.Section,
				CourseCode: booking.CourseCode,
				Source:     booking.Source,
				Status:     booking.Status,
				Label:      bookingLabel(booking, profile.Batch),
			})
		}
		httpx.WriteJSON(w, http.StatusOK, availabilityResponse{Date: day, Room: room, Items: items})
	}
}

func bookingLabel(booking store.Booking, batch string) string {
	if booking.Source == store.BookingSourceFixed && booking.Batch == batch {
		return LabelYours
	}
	return LabelUnavailable
}
