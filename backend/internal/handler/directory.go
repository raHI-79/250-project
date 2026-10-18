package handler

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strings"
)

type DirectoryStore interface {
	ListDirectory(ctx context.Context, filter store.DirectoryFilter) ([]store.DirectoryRow, error)
	ListTeacherInfo(ctx context.Context, search string) ([]store.TeacherInfoRow, error)
}

type Directory struct {
	contacts DirectoryStore
}

func NewDirectory(contacts DirectoryStore) *Directory {
	return &Directory{contacts: contacts}
}

type directoryItem struct {
	UserID     int64  `json:"user_id"`
	PhotoURL   string `json:"photo_url,omitempty"`
	FullName   string `json:"full_name"`
	RegNo      string `json:"reg_no"`
	Batch      string `json:"batch"`
	Section    string `json:"section"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	BloodGroup string `json:"blood_group"`
	IsCR       bool   `json:"is_cr"`
}

type directoryResponse struct {
	Items []directoryItem `json:"items"`
}

type teacherInfoItem struct {
	UserID      int64  `json:"user_id"`
	Name        string `json:"name"`
	Designation string `json:"designation,omitempty"`
	OfficeRoom  string `json:"office_room,omitempty"`
	Phone       string `json:"phone"`
	Email       string `json:"email"`
	PhotoURL    string `json:"photo_url,omitempty"`
}

type teacherInfoResponse struct {
	Items []teacherInfoItem `json:"items"`
}

func (d *Directory) Students() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		var filter store.DirectoryFilter
		if raw := strings.TrimSpace(query.Get("batch")); raw != "" {
			batch, err := validate.Batch("batch", raw)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			filter.Batch = batch
		}
		if raw := strings.TrimSpace(query.Get("blood_group")); raw != "" {
			group, err := validate.BloodGroup("blood_group", raw)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			filter.BloodGroup = group
		}
		search, err := validate.Search("search", query.Get("search"))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		filter.Search = search
		people, err := d.contacts.ListDirectory(r.Context(), filter)
		if err != nil {
			log.Printf("handler: could not list the student directory: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]directoryItem, 0, len(people))
		for _, person := range people {
			items = append(items, directoryItem{
				UserID:     person.UserID,
				PhotoURL:   person.PhotoPath,
				FullName:   person.FullName,
				RegNo:      person.RegNo,
				Batch:      person.Batch,
				Section:    person.Section,
				Phone:      person.Phone,
				Email:      person.Email,
				BloodGroup: person.BloodGroup,
				IsCR:       person.IsCR,
			})
		}
		httpx.WriteJSON(w, http.StatusOK, directoryResponse{Items: items})
	}
}

func (d *Directory) Teachers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		search, err := validate.Search("search", r.URL.Query().Get("search"))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		teachers, err := d.contacts.ListTeacherInfo(r.Context(), search)
		if err != nil {
			log.Printf("handler: could not list the teachers: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]teacherInfoItem, 0, len(teachers))
		for _, teacher := range teachers {
			items = append(items, teacherInfoItem{
				UserID:      teacher.UserID,
				Name:        teacher.Name,
				Designation: teacher.Designation,
				OfficeRoom:  teacher.OfficeRoom,
				Phone:       teacher.Phone,
				Email:       teacher.Email,
				PhotoURL:    teacher.PhotoPath,
			})
		}
		httpx.WriteJSON(w, http.StatusOK, teacherInfoResponse{Items: items})
	}
}
