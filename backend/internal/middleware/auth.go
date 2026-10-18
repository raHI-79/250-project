package middleware

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"net/http"
	"strings"
)

const (
	authorizationHeader   = "Authorization"
	bearerPrefix          = "Bearer "
	MessageUnauthorized   = "Unauthorized"
	MessageSessionExpired = "Session expired. Please log in again."
	MessageAccessDenied   = "403 Access Denied / Unauthorized Access"
	MessageServerError    = "Internal server error"
)

type UserStore interface {
	GetAuthUser(ctx context.Context, userID int64) (store.AuthUser, error)
}

type TokenParser interface {
	ParseToken(token string) (auth.Claims, error)
}

type User struct {
	ID         int64
	Identifier string
	Role       string
	PhotoPath  string
}

type contextKey struct{}

var currentUserKey contextKey

func CurrentUser(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(currentUserKey).(User)
	return user, ok
}

func WithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, currentUserKey, user)
}

func VerifyToken(tokens TokenParser, users UserStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r.Header.Get(authorizationHeader))
			if !ok {
				httpx.WriteError(w, http.StatusUnauthorized, MessageUnauthorized)
				return
			}
			claims, err := tokens.ParseToken(token)
			if err != nil {
				httpx.WriteError(w, http.StatusUnauthorized, MessageUnauthorized)
				return
			}
			user, err := users.GetAuthUser(r.Context(), claims.UserID)
			if errors.Is(err, store.ErrNotFound) {
				httpx.WriteError(w, http.StatusUnauthorized, MessageSessionExpired)
				return
			}
			if err != nil {
				log.Printf("middleware: could not load the user of a request: %v", err)
				httpx.WriteError(w, http.StatusInternalServerError, MessageServerError)
				return
			}
			if !user.IsActive() || user.TokenVersion != claims.TokenVersion {
				httpx.WriteError(w, http.StatusUnauthorized, MessageSessionExpired)
				return
			}
			effectiveRole := auth.EffectiveRole(user.Role, user.IsCR)
			if effectiveRole != claims.Role {
				httpx.WriteError(w, http.StatusUnauthorized, MessageSessionExpired)
				return
			}
			current := User{
				ID:         user.ID,
				Identifier: user.Identifier,
				Role:       effectiveRole,
				PhotoPath:  user.PhotoPath,
			}
			next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), current)))
		})
	}
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, role := range roles {
		allowed[role] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := CurrentUser(r.Context())
			if !ok {
				httpx.WriteError(w, http.StatusUnauthorized, MessageUnauthorized)
				return
			}
			if !allowed[user.Role] {
				httpx.WriteError(w, http.StatusForbidden, MessageAccessDenied)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func bearerToken(header string) (string, bool) {
	if len(header) <= len(bearerPrefix) {
		return "", false
	}
	if !strings.EqualFold(header[:len(bearerPrefix)], bearerPrefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(bearerPrefix):])
	if token == "" {
		return "", false
	}
	return token, true
}
