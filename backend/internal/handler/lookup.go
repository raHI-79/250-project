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

const MessageBatchRequired = "batch is required."

type LookupStore interface {
	ListBatches(ctx context.Context) ([]string, error)
	ListSections(ctx context.Context, batch string) ([]string, error)
	ListCourses(ctx context.Context) ([]store.Course, error)
	ListTeacherOptions(ctx context.Context) ([]store.TeacherOption, error)
}

type Lookups struct {
	lookups LookupStore
}

func NewLookups(lookups LookupStore) *Lookups {
	return &Lookups{lookups: lookups}
}

type valuesResponse struct {
	Items []string `json:"items"`
}

type teacherOption struct {
	ID     int64  `json:"id"`
	UserID int64  `json:"user_id"`
	Name   string `json:"name"`
}

type teacherOptionsResponse struct {
	Items []teacherOption `json:"items"`
}

func (l *Lookups) Batches() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		batches, err := l.lookups.ListBatches(r.Context())
		if err != nil {
			log.Printf("handler: could not list the batches: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, valuesResponse{Items: batches})
	}
}

func (l *Lookups) Sections() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimSpace(r.URL.Query().Get("batch"))
		if raw == "" {
			httpx.WriteError(w, http.StatusBadRequest, MessageBatchRequired)
			return
		}
		batch, err := validate.Batch("batch", raw)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		sections, err := l.lookups.ListSections(r.Context(), batch)
		if err != nil {
			log.Printf("handler: could not list the sections: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, valuesResponse{Items: sections})
	}
}

func (l *Lookups) Courses() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courses, err := l.lookups.ListCourses(r.Context())
		if err != nil {
			log.Printf("handler: could not list the courses for the lookups: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, coursesResponse{Items: courseItems(courses)})
	}
}

func (l *Lookups) Teachers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teachers, err := l.lookups.ListTeacherOptions(r.Context())
		if err != nil {
			log.Printf("handler: could not list the teachers for the lookups: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]teacherOption, 0, len(teachers))
		for _, teacher := range teachers {
			items = append(items, teacherOption{ID: teacher.ID, UserID: teacher.UserID, Name: teacher.Name})
		}
		httpx.WriteJSON(w, http.StatusOK, teacherOptionsResponse{Items: items})
	}
}
