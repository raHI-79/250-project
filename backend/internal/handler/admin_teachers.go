package handler

import (
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strings"
)

type createTeacherRequest struct {
	Name            string `json:"name"`
	TeacherID       string `json:"teacher_id"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

type teacherItem struct {
	ID        int64  `json:"id"`
	TeacherID string `json:"teacher_id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type teachersResponse struct {
	Items []teacherItem `json:"items"`
}

func (a *Admin) CreateTeacher() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request createTeacherRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		teacher, err := request.toNewTeacher()
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		hash, err := auth.HashPassword(request.Password)
		if err != nil {
			log.Printf("handler: could not hash a teacher password: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		teacher.PasswordHash = hash
		userID, err := a.people.CreateTeacher(r.Context(), teacher)
		if err != nil {
			if errors.Is(err, store.ErrDuplicateAccount) {
				httpx.WriteError(w, http.StatusConflict, MessageDuplicateTeacher)
				return
			}
			log.Printf("handler: could not create a teacher: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, createdResponse{ID: userID, Message: MessageTeacherCreated})
	}
}

func (request createTeacherRequest) toNewTeacher() (store.NewTeacher, error) {
	var teacher store.NewTeacher
	fields := []struct {
		target *string
		value  string
		label  string
		check  func(string, string) (string, error)
	}{
		{target: &teacher.Name, value: request.Name, label: "Name", check: validate.Required},
		{target: &teacher.TeacherID, value: request.TeacherID, label: "Teacher ID", check: validate.TeacherID},
		{target: &teacher.Email, value: request.Email, label: "Email", check: validate.Email},
		{target: &teacher.Phone, value: request.Phone, label: "Phone", check: validate.Phone},
	}
	for _, field := range fields {
		value, err := field.check(field.label, field.value)
		if err != nil {
			return store.NewTeacher{}, err
		}
		*field.target = value
	}
	if err := validate.Password("Password", request.Password, request.ConfirmPassword); err != nil {
		return store.NewTeacher{}, err
	}
	return teacher, nil
}

func (a *Admin) Teachers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		status, err := statusFilter(query.Get("status"))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		filter := store.TeacherFilter{Search: strings.TrimSpace(query.Get("search")), Status: status}
		teachers, err := a.people.ListTeachers(r.Context(), filter)
		if err != nil {
			log.Printf("handler: could not list the teachers: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]teacherItem, 0, len(teachers))
		for _, teacher := range teachers {
			items = append(items, teacherItem{
				ID:        teacher.ID,
				TeacherID: teacher.TeacherID,
				Name:      teacher.Name,
				Email:     teacher.Email,
				Phone:     teacher.Phone,
				Status:    teacher.Status,
				CreatedAt: timestamp(teacher.CreatedAt),
			})
		}
		httpx.WriteJSON(w, http.StatusOK, teachersResponse{Items: items})
	}
}

func (a *Admin) DeactivateTeacher() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := pathID(w, r)
		if !ok {
			return
		}
		err := a.people.DeactivateTeacher(r.Context(), userID)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageTeacherNotFound)
		case errors.Is(err, store.ErrAlreadyRemoved):
			httpx.WriteError(w, http.StatusConflict, MessageAlreadyRemoved)
		case err != nil:
			log.Printf("handler: could not deactivate a teacher: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageTeacherDeactivated})
		}
	}
}
