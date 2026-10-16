package handler

import (
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"net/http"
)

type meResponse struct {
	ID         int64  `json:"id"`
	Identifier string `json:"identifier"`
	Role       string `json:"role"`
}

func Me() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, meResponse{
			ID:         user.ID,
			Identifier: user.Identifier,
			Role:       user.Role,
		})
	}
}
