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
	"time"
)

const (
	authorizationHeader   = "Authorization"
	bearerPrefix          = "Bearer "
	MessageUnauthorized   = "Unauthorized"
	MessageSessionExpired = "Session expired. Please log in again."
	MessageAccessDenied   = "403 Access Denied / Unauthorized Access"
	MessageServerError    = "Internal server error"
	MessageReadOnly       = "The control room opens the office pages read-only."
)

type UserStore interface {
	GetAuthUser(ctx context.Context, userID int64) (store.AuthUser, error)
}

type TokenParser interface {
	ParseToken(token string) (auth.Claims, error)
}

type SessionStore interface {
	LoadSession(ctx context.Context, sessionID int64) (store.Session, error)
	TouchSession(ctx context.Context, sessionID int64, ipHash, country string) error
}

type Visitor interface {
	Address(r *http.Request) (string, string)
}

type User struct {
	ID         int64
	Identifier string
	Role       string
	PhotoPath  string
	ReadOnly   bool
	SessionID  int64
	SudoUntil  *time.Time
}

func (u User) HasSudo(now time.Time) bool {
	return u.SudoUntil != nil && u.SudoUntil.After(now)
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

type Guard struct {
	tokens   TokenParser
	users    UserStore
	sessions SessionStore
	visitors Visitor
	now      func() time.Time
}

func NewGuard(tokens TokenParser, users UserStore) *Guard {
	return &Guard{tokens: tokens, users: users, now: time.Now}
}

func (g *Guard) WithSessions(sessions SessionStore, visitors Visitor) *Guard {
	g.sessions = sessions
	g.visitors = visitors
	return g
}

func VerifyToken(tokens TokenParser, users UserStore) func(http.Handler) http.Handler {
	return NewGuard(tokens, users).Middleware
}

func (g *Guard) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r.Header.Get(authorizationHeader))
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, MessageUnauthorized)
			return
		}
		claims, err := g.tokens.ParseToken(token)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, MessageUnauthorized)
			return
		}
		user, err := g.users.GetAuthUser(r.Context(), claims.UserID)
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
		effectiveRole, allowed := auth.SessionRole(user.Role, user.IsCR, user.OfficeAccess, claims.Role)
		if !allowed {
			httpx.WriteError(w, http.StatusUnauthorized, MessageSessionExpired)
			return
		}
		if FromAndroidApp(r) && OfficeRole(effectiveRole) {
			httpx.WriteError(w, http.StatusForbidden, MessageAppForbidden)
			return
		}
		current := User{
			ID:         user.ID,
			Identifier: user.Identifier,
			Role:       effectiveRole,
			PhotoPath:  user.PhotoPath,
			ReadOnly:   effectiveRole == auth.RoleSuperAdmin,
			SessionID:  claims.SessionID,
		}
		if !g.sessionLives(w, r, &current, claims.SessionID, user.ID) {
			return
		}
		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), current)))
	})
}

func (g *Guard) sessionLives(w http.ResponseWriter, r *http.Request, current *User,
	sessionID, userID int64) bool {
	if g.sessions == nil || sessionID <= 0 {
		return true
	}
	now := g.now()
	session, err := g.sessions.LoadSession(r.Context(), sessionID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusUnauthorized, MessageSessionExpired)
		return false
	}
	if err != nil {
		log.Printf("middleware: could not load the session of a request: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, MessageServerError)
		return false
	}
	if session.UserID != userID || !session.Live(now) {
		httpx.WriteError(w, http.StatusUnauthorized, MessageSessionExpired)
		return false
	}
	current.SudoUntil = session.SudoUntil
	if now.Sub(session.LastSeenAt) < store.SessionTouchAfter {
		return true
	}
	ipHash, country := "", ""
	if g.visitors != nil {
		ipHash, country = g.visitors.Address(r)
	}
	if err := g.sessions.TouchSession(r.Context(), session.ID, ipHash, country); err != nil {
		log.Printf("middleware: could not touch a session: %v", err)
	}
	return true
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

func ReadOnly(roles ...string) func(http.Handler) http.Handler {
	watched := make(map[string]bool, len(roles))
	for _, role := range roles {
		watched[role] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := CurrentUser(r.Context())
			if !ok {
				httpx.WriteError(w, http.StatusUnauthorized, MessageUnauthorized)
				return
			}
			if watched[user.Role] && r.Method != http.MethodGet && r.Method != http.MethodHead {
				httpx.WriteError(w, http.StatusForbidden, MessageReadOnly)
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

const (
	ClientHeader        = "X-Client"
	ClientAndroid       = "android"
	MessageAppForbidden = "The Android app is for students and teachers. Office accounts use the website."
)

func OfficeRole(role string) bool {
	return auth.IsOfficeRole(role)
}

func FromAndroidApp(r *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(r.Header.Get(ClientHeader)), ClientAndroid)
}
