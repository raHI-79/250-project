package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
)

const (
	MessageCourseCreated    = "Course added."
	MessageDuplicateCourse  = "A course with this code already exists."
	MessageCourseUnknown    = "No course found with this ID."
	MessageTeacherNotActive = "This teacher account is not active."
)

type CatalogStore interface {
	CreateCourse(ctx context.Context, code, title string) (store.Course, error)
	ListCourses(ctx context.Context) ([]store.Course, error)
	SearchCourses(ctx context.Context, search string) ([]store.Course, error)
	CreateRoutine(ctx context.Context, routine store.NewRoutine) (store.RoutineRow, error)
	ListRoutines(ctx context.Context, batch, section string) ([]store.RoutineRow, error)
	DeleteRoutine(ctx context.Context, routineID int64) error
}

type Catalog struct {
	catalog CatalogStore
}

func NewCatalog(catalog CatalogStore) *Catalog {
	return &Catalog{catalog: catalog}
}

type createCourseRequest struct {
	Code  string `json:"code"`
	Title string `json:"title"`
}

type courseItem struct {
	ID    int64  `json:"id"`
	Code  string `json:"code"`
	Title string `json:"title"`
}

type coursesResponse struct {
	Items []courseItem `json:"items"`
}

type courseResponse struct {
	ID      int64  `json:"id"`
	Code    string `json:"code"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

func (c *Catalog) CreateCourse() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request createCourseRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		code, err := validate.CourseCode("Course code", request.Code)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		title, err := validate.Required("Title", request.Title)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		course, err := c.catalog.CreateCourse(r.Context(), code, title)
		if err != nil {
			if errors.Is(err, store.ErrDuplicateCourse) {
				httpx.WriteError(w, http.StatusConflict, MessageDuplicateCourse)
				return
			}
			log.Printf("handler: could not create a course: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, courseResponse{
			ID:      course.ID,
			Code:    course.Code,
			Title:   course.Title,
			Message: MessageCourseCreated,
		})
	}
}

func (c *Catalog) Courses() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		search, err := validate.Search("search", r.URL.Query().Get("search"))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		courses, err := c.catalog.SearchCourses(r.Context(), search)
		if err != nil {
			log.Printf("handler: could not list the courses: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, coursesResponse{Items: courseItems(courses)})
	}
}

func courseItems(courses []store.Course) []courseItem {
	items := make([]courseItem, 0, len(courses))
	for _, course := range courses {
		items = append(items, courseItem{ID: course.ID, Code: course.Code, Title: course.Title})
	}
	return items
}
