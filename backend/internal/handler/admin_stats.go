package handler

import (
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"log"
	"net/http"
)

type statsResponse struct {
	TotalStudents    int `json:"total_students"`
	TotalCRs         int `json:"total_crs"`
	TotalTeachers    int `json:"total_teachers"`
	PendingApprovals int `json:"pending_approvals"`
}

func (a *Admin) Stats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := a.people.Stats(r.Context())
		if err != nil {
			log.Printf("handler: could not load the overview stats: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, statsResponse{
			TotalStudents:    stats.TotalStudents,
			TotalCRs:         stats.TotalCRs,
			TotalTeachers:    stats.TotalTeachers,
			PendingApprovals: stats.PendingApprovals,
		})
	}
}
