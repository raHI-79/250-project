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
	Catalog  handler.CatalogStore
	Lookups  handler.LookupStore
	Alerts   handler.NotificationService
	Classes  handler.CRStore
	Notifier handler.Notifier
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
	registerAdminRoutes(AdminGroup(mux, deps), handler.NewAdmin(deps.People), handler.NewCatalog(deps.Catalog))
	registerLookupRoutes(authenticated, handler.NewLookups(deps.Lookups))
	registerNotificationRoutes(NotificationGroup(mux, deps), handler.NewNotifications(deps.Alerts))
	registerCRRoutes(CRGroup(mux, deps), handler.NewCR(deps.Config, deps.Classes, deps.Notifier))
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

func NotificationGroup(mux *http.ServeMux, deps Dependencies) RouteGroup {
	return RouteGroup{
		mux: mux,
		middleware: []func(http.Handler) http.Handler{
			middleware.VerifyToken(deps.Tokens, deps.Users),
			middleware.RequireRole(auth.RoleStudent, auth.RoleCR, auth.RoleTeacher),
		},
	}
}

func CRGroup(mux *http.ServeMux, deps Dependencies) RouteGroup {
	return RouteGroup{
		mux: mux,
		middleware: []func(http.Handler) http.Handler{
			middleware.VerifyToken(deps.Tokens, deps.Users),
			middleware.RequireRole(auth.RoleCR),
		},
	}
}

func registerCRRoutes(group RouteGroup, cr *handler.CR) {
	group.Handle("GET /api/cr/routine", cr.Routine())
	group.Handle("POST /api/cr/class-requests", cr.CreateClassRequest())
	group.Handle("GET /api/cr/requests", cr.Requests())
}

func registerLookupRoutes(group RouteGroup, lookups *handler.Lookups) {
	group.Handle("GET /api/lookups/batches", lookups.Batches())
	group.Handle("GET /api/lookups/sections", lookups.Sections())
	group.Handle("GET /api/lookups/courses", lookups.Courses())
	group.Handle("GET /api/lookups/teachers", lookups.Teachers())
}

func registerNotificationRoutes(group RouteGroup, notifications *handler.Notifications) {
	group.Handle("GET /api/notifications", notifications.List())
	group.Handle("GET /api/notifications/unread-count", notifications.UnreadCount())
}

func registerAdminRoutes(group RouteGroup, admin *handler.Admin, catalog *handler.Catalog) {
	group.Handle("GET /api/admin/registrations", admin.Registrations())
	group.Handle("POST /api/admin/registrations/{id}/approve", admin.ApproveRegistration())
	group.Handle("POST /api/admin/registrations/{id}/reject", admin.RejectRegistration())
	group.Handle("GET /api/admin/students", admin.Students())
	group.Handle("POST /api/admin/students/{id}/remove", admin.RemoveStudent())
	group.Handle("POST /api/admin/crs", admin.CreateCR())
	group.Handle("GET /api/admin/crs", admin.CRs())
	group.Handle("POST /api/admin/crs/{id}/demote", admin.DemoteCR())
	group.Handle("POST /api/admin/crs/{id}/deactivate", admin.DeactivateCR())
	group.Handle("POST /api/admin/teachers", admin.CreateTeacher())
	group.Handle("GET /api/admin/teachers", admin.Teachers())
	group.Handle("POST /api/admin/teachers/{id}/deactivate", admin.DeactivateTeacher())
	group.Handle("GET /api/admin/stats", admin.Stats())
	group.Handle("POST /api/admin/courses", catalog.CreateCourse())
	group.Handle("GET /api/admin/courses", catalog.Courses())
	group.Handle("POST /api/admin/routines", catalog.CreateRoutine())
	group.Handle("GET /api/admin/routines", catalog.Routines())
	group.Handle("DELETE /api/admin/routines/{id}", catalog.DeleteRoutine())
}

func notFound(w http.ResponseWriter, r *http.Request) {
	httpx.WriteError(w, http.StatusNotFound, "Route not found")
}
