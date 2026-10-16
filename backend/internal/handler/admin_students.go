package handler

import (
	"errors"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type studentItem struct {
	ID             int64  `json:"id"`
	FullName       string `json:"full_name"`
	RegNo          string `json:"reg_no"`
	Batch          string `json:"batch"`
	Section        string `json:"section"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	BloodGroup     string `json:"blood_group"`
	HasDropCourse  bool   `json:"has_drop_course"`
	DropCourseCode string `json:"drop_course_code"`
	IsCR           bool   `json:"is_cr"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
}

type studentsResponse struct {
	Items []studentItem `json:"items"`
	Total int           `json:"total"`
}

func (a *Admin) Students() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter, err := studentFilter(r.URL.Query())
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		students, total, err := a.people.ListStudents(r.Context(), filter)
		if err != nil {
			log.Printf("handler: could not list the students: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, studentsResponse{Items: studentItems(students), Total: total})
	}
}

func studentItems(students []store.StudentRow) []studentItem {
	items := make([]studentItem, 0, len(students))
	for _, student := range students {
		items = append(items, studentItem{
			ID:             student.ID,
			FullName:       student.FullName,
			RegNo:          student.RegNo,
			Batch:          student.Batch,
			Section:        student.Section,
			Email:          student.Email,
			Phone:          student.Phone,
			BloodGroup:     student.BloodGroup,
			HasDropCourse:  student.HasDropCourse,
			DropCourseCode: student.DropCourseCode,
			IsCR:           student.IsCR,
			Status:         student.Status,
			CreatedAt:      timestamp(student.CreatedAt),
		})
	}
	return items
}

func studentFilter(query url.Values) (store.StudentFilter, error) {
	filter := store.StudentFilter{
		Search: strings.TrimSpace(query.Get("search")),
		Limit:  store.DefaultStudentPage,
	}
	if batch := strings.TrimSpace(query.Get("batch")); batch != "" {
		value, err := validate.Batch("batch", batch)
		if err != nil {
			return store.StudentFilter{}, err
		}
		filter.Batch = value
	}
	if section := strings.TrimSpace(query.Get("section")); section != "" {
		value, err := validate.Section("section", section)
		if err != nil {
			return store.StudentFilter{}, err
		}
		filter.Section = value
	}
	status, err := statusFilter(query.Get("status"))
	if err != nil {
		return store.StudentFilter{}, err
	}
	filter.Status = status
	if raw := strings.TrimSpace(query.Get("limit")); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > store.MaxStudentPage {
			return store.StudentFilter{}, errors.New(MessageInvalidLimit)
		}
		filter.Limit = limit
	}
	if raw := strings.TrimSpace(query.Get("offset")); raw != "" {
		offset, err := strconv.Atoi(raw)
		if err != nil || offset < 0 {
			return store.StudentFilter{}, errors.New(MessageInvalidOffset)
		}
		filter.Offset = offset
	}
	return filter, nil
}

func statusFilter(raw string) (string, error) {
	status := strings.ToLower(strings.TrimSpace(raw))
	switch status {
	case "", store.StatusFilterAll, store.StatusActive, store.StatusPending, store.StatusRejected, store.StatusRemoved:
		return status, nil
	default:
		return "", errors.New(MessageInvalidStatus)
	}
}

func (a *Admin) RemoveStudent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := pathID(w, r)
		if !ok {
			return
		}
		err := a.people.RemoveStudent(r.Context(), userID)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageStudentNotFound)
		case errors.Is(err, store.ErrAlreadyRemoved):
			httpx.WriteError(w, http.StatusConflict, MessageAlreadyRemoved)
		case err != nil:
			log.Printf("handler: could not remove a student: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageStudentRemoved})
		}
	}
}
