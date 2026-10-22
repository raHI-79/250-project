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
	"strconv"
	"strings"
)

const (
	MessageCourseCreated    = "Course added."
	MessageRoutineMoved     = "Routine entry moved."
	MessageRunningSemester  = "Running semester updated."
	MessageCourseDeleted    = "Course deleted."
	MessageDuplicateCourse  = "A course with this code already exists."
	MessageCourseUnknown    = "No course found with this ID."
	MessageTeacherNotActive = "This teacher account is not active."
)

type CatalogStore interface {
	CreateCourse(ctx context.Context, code, title string, credit float64) (store.Course, error)
	ListCourses(ctx context.Context) ([]store.Course, error)
	SearchCourses(ctx context.Context, search string) ([]store.Course, error)
	CreateRoutine(ctx context.Context, routine store.NewRoutine) (store.RoutineRow, error)
	ListRoutines(ctx context.Context, batch, section, semester string) ([]store.RoutineRow, error)
	UpdateRoutineSemester(ctx context.Context, routineID int64, semester string) error
	ListRunningSemesters(ctx context.Context) (map[string]string, error)
	SetRunningSemester(ctx context.Context, batch, semester string) error
	DeleteRoutine(ctx context.Context, routineID int64) error
	DeleteCourse(ctx context.Context, courseID int64) (store.CourseUsage, error)
	ListPeriods(ctx context.Context) ([]store.Period, error)
}

type Catalog struct {
	catalog CatalogStore
}

func NewCatalog(catalog CatalogStore) *Catalog {
	return &Catalog{catalog: catalog}
}

type createCourseRequest struct {
	Code   string  `json:"code"`
	Title  string  `json:"title"`
	Credit float64 `json:"credit"`
}

type courseItem struct {
	ID     int64   `json:"id"`
	Code   string  `json:"code"`
	Title  string  `json:"title"`
	Credit float64 `json:"credit"`
}

type coursesResponse struct {
	Items []courseItem `json:"items"`
}

type courseResponse struct {
	courseItem
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
		credit, err := validate.Credit("Credit", request.Credit)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		course, err := c.catalog.CreateCourse(r.Context(), code, title, credit)
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
			courseItem: newCourseItem(course),
			Message:    MessageCourseCreated,
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
		items = append(items, newCourseItem(course))
	}
	return items
}

func newCourseItem(course store.Course) courseItem {
	return courseItem{ID: course.ID, Code: course.Code, Title: course.Title, Credit: course.Credit}
}

func (c *Catalog) DeleteCourse() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseID, ok := pathID(w, r)
		if !ok {
			return
		}
		usage, err := c.catalog.DeleteCourse(r.Context(), courseID)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageCourseUnknown)
		case errors.Is(err, store.ErrCourseInUse):
			httpx.WriteError(w, http.StatusConflict, courseInUseMessage(usage))
		case err != nil:
			log.Printf("handler: could not delete a course: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageCourseDeleted})
		}
	}
}

func courseInUseMessage(usage store.CourseUsage) string {
	parts := make([]string, 0, 3)
	if usage.Routines > 0 {
		parts = append(parts, plural(usage.Routines, "routine entry", "routine entries"))
	}
	if usage.Items > 0 {
		parts = append(parts, plural(usage.Items, "class or exam", "classes and exams"))
	}
	if usage.DropCourses > 0 {
		parts = append(parts, plural(usage.DropCourses, "student", "students")+" with it as a drop course")
	}
	return "This course is still in use by " + join(parts) + ". Remove those first."
}

func plural(count int, one, many string) string {
	if count == 1 {
		return strconv.Itoa(count) + " " + one
	}
	return strconv.Itoa(count) + " " + many
}

func join(parts []string) string {
	switch len(parts) {
	case 0:
		return "the department"
	case 1:
		return parts[0]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
	}
}
