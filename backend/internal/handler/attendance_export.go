package handler

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/attendance"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	workbookType      = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	MessageNoWorkbook = "There is no attendance to export for this range."
)

func exportRows(rows []store.ExportRow) []attendance.ExportRow {
	out := make([]attendance.ExportRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, attendance.ExportRow{
			StudentID: row.StudentID,
			FullName:  row.FullName,
			RegNo:     row.RegNo,
			SessionID: row.SessionID,
			StartedAt: row.StartedAt,
			Status:    row.Status,
			Note:      row.Note,
		})
	}
	return out
}

func writeWorkbook(w http.ResponseWriter, classes []attendance.ExportClass,
	withSummary bool, from, to time.Time) {
	buffer, err := attendance.BuildWorkbook(classes, withSummary)
	if err != nil {
		log.Printf("handler: could not build an attendance workbook: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return
	}
	w.Header().Set("Content-Type", workbookType)
	w.Header().Set("Content-Disposition",
		`attachment; filename="`+attendance.FileName(classes, from, to.AddDate(0, 0, -1))+`"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(buffer.Bytes()); err != nil {
		log.Printf("handler: could not send an attendance workbook: %v", err)
	}
}

func (a *Attendance) teacherClass(ctx context.Context, teacherID int64, class store.ClassSummary,
	from, to time.Time) (attendance.ExportClass, error) {
	rows, err := a.store.TeacherExportRows(ctx, teacherID, class.CourseID,
		class.Batch, class.Section, from, to)
	if err != nil {
		return attendance.ExportClass{}, err
	}
	return attendance.ExportClass{
		CourseCode:  class.CourseCode,
		CourseTitle: class.CourseTitle,
		Batch:       class.Batch,
		Section:     class.Section,
		Sessions:    class.Sessions,
		Rows:        exportRows(rows),
	}, nil
}

func (a *Attendance) Export() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := a.teacher(w, r)
		if !ok {
			return
		}
		from, to := a.window(r)
		classes, err := a.store.ListTeacherClasses(r.Context(), teacher.ID, from, to)
		if err != nil {
			log.Printf("handler: could not list the classes of an attendance export: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		courseID := sessionIDQuery(r, "course_id")
		batch := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("batch")))
		section := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("section")))
		wanted := make([]store.ClassSummary, 0, len(classes))
		for _, class := range classes {
			if courseID > 0 && class.CourseID != courseID {
				continue
			}
			if batch != "" && class.Batch != batch {
				continue
			}
			if section != "" && class.Section != section {
				continue
			}
			wanted = append(wanted, class)
		}
		if len(wanted) == 0 {
			httpx.WriteError(w, http.StatusNotFound, MessageNoWorkbook)
			return
		}
		sheets := make([]attendance.ExportClass, 0, len(wanted))
		for _, class := range wanted {
			sheet, err := a.teacherClass(r.Context(), teacher.ID, class, from, to)
			if err != nil {
				log.Printf("handler: could not read the rows of an attendance export: %v", err)
				httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
				return
			}
			sheets = append(sheets, sheet)
		}
		writeWorkbook(w, sheets, len(sheets) > 1, from, to)
	}
}
