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

type examItem struct {
	ID           int64  `json:"id"`
	ItemDate     string `json:"item_date"`
	StartTime    string `json:"start_time,omitempty"`
	EndTime      string `json:"end_time,omitempty"`
	CourseCode   string `json:"course_code"`
	CourseTitle  string `json:"course_title"`
	Batch        string `json:"batch"`
	Section      string `json:"section"`
	Mode         string `json:"mode,omitempty"`
	Room         string `json:"room,omitempty"`
	MeetingLink  string `json:"meeting_link,omitempty"`
	Syllabus     string `json:"syllabus,omitempty"`
	Instructions string `json:"instructions,omitempty"`
	Origin       string `json:"origin"`
	Published    bool   `json:"published"`
}

type examsResponse struct {
	Items []examItem `json:"items"`
}

type routineWeekResponse struct {
	Batch           string        `json:"batch"`
	Semester        string        `json:"semester"`
	SemesterLabel   string        `json:"semester_label"`
	RunningSemester string        `json:"running_semester"`
	Items           []routineItem `json:"items"`
}

type courseTeacherItem struct {
	UserID      int64  `json:"user_id"`
	Name        string `json:"name"`
	Designation string `json:"designation,omitempty"`
}

type runningCourseItem struct {
	CourseID       int64               `json:"course_id"`
	Code           string              `json:"code"`
	Title          string              `json:"title"`
	Credit         float64             `json:"credit"`
	Teachers       []courseTeacherItem `json:"teachers"`
	Sections       string              `json:"sections"`
	ClassesPerWeek int                 `json:"classes_per_week"`
	LabsPerWeek    int                 `json:"labs_per_week"`
}

type runningCoursesResponse struct {
	Batch         string              `json:"batch"`
	Semester      string              `json:"semester"`
	SemesterLabel string              `json:"semester_label"`
	TotalCredits  float64             `json:"total_credits"`
	Items         []runningCourseItem `json:"items"`
}

func (s *Student) Exams() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := s.profile(w, r)
		if !ok {
			return
		}
		codes, ok := s.dropCourseCodes(w, r, profile.StudentID)
		if !ok {
			return
		}
		exams, err := s.students.ListStudentExams(r.Context(), profile.Batch, profile.Section,
			s.cfg.Now(), codes)
		if err != nil {
			log.Printf("handler: could not list the exams of a student: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]examItem, 0, len(exams))
		for _, exam := range exams {
			items = append(items, newExamItem(exam))
		}
		httpx.WriteJSON(w, http.StatusOK, examsResponse{Items: items})
	}
}

func newExamItem(exam store.StudentExam) examItem {
	return examItem{
		ID:           exam.ID,
		ItemDate:     exam.ItemDate,
		StartTime:    exam.StartTime,
		EndTime:      exam.EndTime,
		CourseCode:   exam.CourseCode,
		CourseTitle:  exam.CourseTitle,
		Batch:        exam.Batch,
		Section:      exam.Section,
		Mode:         exam.Mode,
		Room:         exam.Room,
		MeetingLink:  exam.MeetingLink,
		Syllabus:     exam.Syllabus,
		Instructions: exam.Instructions,
		Origin:       exam.Origin,
		Published:    exam.Published,
	}
}

func (s *Student) DropCourseOptions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := s.profile(w, r)
		if !ok {
			return
		}
		search, err := validate.Search("search", r.URL.Query().Get("search"))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		courses, err := s.students.SearchOtherBatchDropCourses(r.Context(), profile.Batch, search)
		if err != nil {
			log.Printf("handler: could not list the drop course options of a student: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, coursesResponse{Items: courseItems(courses)})
	}
}

func (s *Student) Routine() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := s.profile(w, r)
		if !ok {
			return
		}
		filter, err := routineBrowserFilter(r.URL.Query(), profile)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		running, err := s.students.RunningSemester(r.Context(), filter.Batch)
		if err != nil {
			log.Printf("handler: could not read the running semester: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if filter.Semester == "" {
			filter.Semester = running
		}
		entries, err := s.students.ListFilteredRoutines(r.Context(), filter)
		if err != nil {
			log.Printf("handler: could not list the routine for a student: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]routineItem, 0, len(entries))
		for _, entry := range entries {
			items = append(items, newRoutineItem(entry))
		}
		httpx.WriteJSON(w, http.StatusOK, routineWeekResponse{
			Batch:           filter.Batch,
			Semester:        filter.Semester,
			SemesterLabel:   validate.SemesterLabel(filter.Semester),
			RunningSemester: running,
			Items:           items,
		})
	}
}

func (s *Student) Courses() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := s.profile(w, r)
		if !ok {
			return
		}
		batch := profile.Batch
		if raw := strings.TrimSpace(r.URL.Query().Get("batch")); raw != "" {
			value, err := validate.Batch("batch", raw)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			batch = value
		}
		running, err := s.students.RunningSemester(r.Context(), batch)
		if err != nil {
			log.Printf("handler: could not read the running semester: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		slots, err := s.students.ListRunningCourseSlots(r.Context(), batch)
		if err != nil {
			log.Printf("handler: could not list the running courses: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := groupRunningCourses(slots)
		total := 0.0
		for _, item := range items {
			total += item.Credit
		}
		httpx.WriteJSON(w, http.StatusOK, runningCoursesResponse{
			Batch:         batch,
			Semester:      running,
			SemesterLabel: validate.SemesterLabel(running),
			TotalCredits:  total,
			Items:         items,
		})
	}
}

func groupRunningCourses(slots []store.RunningCourseSlot) []runningCourseItem {
	order := make([]int64, 0)
	byCourse := make(map[int64]*runningCourseItem)
	sections := make(map[int64]map[string]bool)
	teachers := make(map[int64]map[int64]bool)
	for _, slot := range slots {
		item, seen := byCourse[slot.CourseID]
		if !seen {
			item = &runningCourseItem{
				CourseID: slot.CourseID,
				Code:     slot.CourseCode,
				Title:    slot.CourseTitle,
				Credit:   slot.Credit,
				Teachers: make([]courseTeacherItem, 0, 1),
			}
			byCourse[slot.CourseID] = item
			sections[slot.CourseID] = make(map[string]bool)
			teachers[slot.CourseID] = make(map[int64]bool)
			order = append(order, slot.CourseID)
		}
		if slot.Type == validate.RoutineLab {
			item.LabsPerWeek++
		} else {
			item.ClassesPerWeek++
		}
		for _, section := range sectionsOf(slot.Section) {
			sections[slot.CourseID][section] = true
		}
		if !teachers[slot.CourseID][slot.TeacherUserID] {
			teachers[slot.CourseID][slot.TeacherUserID] = true
			item.Teachers = append(item.Teachers, courseTeacherItem{
				UserID:      slot.TeacherUserID,
				Name:        slot.TeacherName,
				Designation: slot.TeacherDesignation,
			})
		}
	}
	items := make([]runningCourseItem, 0, len(order))
	for _, courseID := range order {
		item := byCourse[courseID]
		item.Sections = sectionLabelOf(sections[courseID])
		items = append(items, *item)
	}
	return items
}

func sectionsOf(section string) []string {
	if section == validate.SectionAll {
		return validate.Sections()
	}
	return []string{section}
}

func sectionLabelOf(seen map[string]bool) string {
	chosen := make([]string, 0, len(seen))
	for _, section := range validate.Sections() {
		if seen[section] {
			chosen = append(chosen, section)
		}
	}
	if len(chosen) == len(validate.Sections()) {
		return validate.LabelBothSections
	}
	if len(chosen) == 0 {
		return ""
	}
	return "Section " + strings.Join(chosen, " and ")
}

func routineBrowserFilter(query url.Values, profile store.StudentProfile) (store.RoutineFilter, error) {
	filter := store.RoutineFilter{
		Batch:     profile.Batch,
		DayOfWeek: store.NoDayFilter,
	}
	if raw := strings.TrimSpace(query.Get("semester")); raw != "" {
		semester, err := validate.Semester("semester", raw)
		if err != nil {
			return store.RoutineFilter{}, err
		}
		filter.Semester = semester
	}
	if raw := strings.TrimSpace(query.Get("batch")); raw != "" {
		batch, err := validate.Batch("batch", raw)
		if err != nil {
			return store.RoutineFilter{}, err
		}
		filter.Batch = batch
	}
	if raw := strings.TrimSpace(query.Get("section")); raw != "" {
		section, err := validate.RoutineSection("section", raw)
		if err != nil {
			return store.RoutineFilter{}, err
		}
		filter.Section = section
	}
	if raw := strings.TrimSpace(query.Get("day")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return store.RoutineFilter{}, errors.New("day must be a number from 0 (Sunday) to 6 (Saturday).")
		}
		day, err := validate.DayOfWeek("day", value)
		if err != nil {
			return store.RoutineFilter{}, err
		}
		filter.DayOfWeek = day
	}
	if raw := strings.TrimSpace(query.Get("course")); raw != "" {
		code, err := validate.CourseCode("course", raw)
		if err != nil {
			return store.RoutineFilter{}, err
		}
		filter.CourseCode = validate.SqueezeCourseCode(code)
	}
	return filter, nil
}
