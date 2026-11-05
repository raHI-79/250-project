package middleware

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"log"
	"net/http"
)

const MessageNoPermission = "You do not have permission for this"

type PermissionStore interface {
	Permissions(ctx context.Context, userID int64) ([]string, error)
}

type Needs interface {
	Permission(pattern string) (string, bool)
}

func RequirePermission(needs Needs, granted PermissionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := CurrentUser(r.Context())
			if !ok {
				httpx.WriteError(w, http.StatusUnauthorized, MessageUnauthorized)
				return
			}
			if user.Role == auth.RoleSuperAdmin || needs == nil || granted == nil {
				next.ServeHTTP(w, r)
				return
			}
			permission, known := needs.Permission(r.Pattern)
			if !known {
				httpx.WriteError(w, http.StatusForbidden, MessageNoPermission)
				return
			}
			if permission == "" {
				next.ServeHTTP(w, r)
				return
			}
			names, err := granted.Permissions(r.Context(), user.ID)
			if err != nil {
				log.Printf("middleware: could not read the permissions of a request: %v", err)
				httpx.WriteError(w, http.StatusInternalServerError, MessageServerError)
				return
			}
			for _, name := range names {
				if name == permission {
					next.ServeHTTP(w, r)
					return
				}
			}
			httpx.WriteError(w, http.StatusForbidden, MessageNoPermission)
		})
	}
}
