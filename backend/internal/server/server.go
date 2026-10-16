package server

import (
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"net/http"
)

type TokenService interface {
	middleware.TokenParser
	handler.TokenIssuer
}

type Dependencies struct {
	Config   *config.Config
	Database handler.DatabasePinger
	Tokens   TokenService
	Users    middleware.UserStore
	Auth     handler.AuthStore
	People   handler.AdminStore
}

type RouteGroup struct {
	mux        *http.ServeMux
	middleware []func(http.Handler) http.Handler
}

func (g RouteGroup) Handle(pattern string, handlerFunc http.HandlerFunc) {
	var wrapped http.Handler = handlerFunc
	for i := len(g.middleware) - 1; i >= 0; i-- {
		wrapped = g.middleware[i](wrapped)
	}
	g.mux.Handle(pattern, wrapped)
}

func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	authHandler := handler.NewAuth(deps.Config, deps.Auth, deps.Tokens)
	limiter := middleware.NewRateLimiter(middleware.DefaultRateLimit, middleware.DefaultRateWindow)
	throttled := RouteGroup{mux: mux, middleware: []func(http.Handler) http.Handler{limiter.Middleware}}
	authenticated := AuthenticatedGroup(mux, deps)
	mux.HandleFunc("GET /api/health", handler.Health(deps.Config, deps.Database))
	mux.HandleFunc("POST /api/auth/register", authHandler.Register())
	throttled.Handle("POST /api/auth/login", authHandler.Login())
	throttled.Handle("POST /api/auth/admin/login", authHandler.AdminLogin())
	throttled.Handle("POST /api/auth/forgot-password", authHandler.ForgotPassword())
	mux.HandleFunc("POST /api/auth/reset-password", authHandler.ResetPassword())
	authenticated.Handle("POST /api/auth/logout", authHandler.Logout())
	authenticated.Handle("GET /api/me", handler.Me())
	registerAdminRoutes(AdminGroup(mux, deps), handler.NewAdmin(deps.People))
	mux.HandleFunc("/", notFound)
	return middleware.Recoverer(middleware.Logger(middleware.CORS(deps.Config.FrontendOrigin)(mux)))
}

func AuthenticatedGroup(mux *http.ServeMux, deps Dependencies) RouteGroup {
	return RouteGroup{
		mux:        mux,
		middleware: []func(http.Handler) http.Handler{middleware.VerifyToken(deps.Tokens, deps.Users)},
	}
}

func AdminGroup(mux *http.ServeMux, deps Dependencies) RouteGroup {
	return RouteGroup{
		mux: mux,
		middleware: []func(http.Handler) http.Handler{
			middleware.VerifyToken(deps.Tokens, deps.Users),
			middleware.RequireRole(auth.RoleAdmin),
		},
	}
}

func registerAdminRoutes(group RouteGroup, admin *handler.Admin) {
	group.Handle("GET /api/admin/registrations", admin.Registrations())
	group.Handle("POST /api/admin/registrations/{id}/approve", admin.ApproveRegistration())
	group.Handle("POST /api/admin/registrations/{id}/reject", admin.RejectRegistration())
	group.Handle("GET /api/admin/students", admin.Students())
	group.Handle("POST /api/admin/students/{id}/remove", admin.RemoveStudent())
}

func notFound(w http.ResponseWriter, r *http.Request) {
	httpx.WriteError(w, http.StatusNotFound, "Route not found")
}
