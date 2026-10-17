package server

import (
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/mailer"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/upload"
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
	Teachers handler.TeacherStore
	Students handler.StudentStore
	Alarms   handler.BroadcastNotifier
	Notices  handler.NoticeStore
	Library  handler.MaterialStore
	Contacts handler.DirectoryStore
	Files    *upload.Saver
	Mailer   mailer.Sender
	Mail     handler.MessageStore
	Votes    handler.PollStore
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
	mail := deps.Mailer
	if mail == nil {
		mail = mailer.NewSMTP(mailer.Options{})
	}
	authHandler := handler.NewAuth(deps.Config, deps.Auth, deps.Tokens, mail)
	limiter := middleware.NewRateLimiter(middleware.DefaultRateLimit, middleware.DefaultRateWindow)
	throttled := RouteGroup{mux: mux, middleware: []func(http.Handler) http.Handler{limiter.Middleware}}
	authenticated := AuthenticatedGroup(mux, deps)
	mux.HandleFunc("GET /api/health", handler.Health(deps.Config, deps.Database))
	mux.HandleFunc("POST /api/auth/register", authHandler.Register())
	throttled.Handle("POST /api/auth/login", authHandler.Login())
	throttled.Handle("POST /api/auth/admin/login", authHandler.AdminLogin())
	throttled.Handle("POST /api/auth/forgot-password", authHandler.ForgotPassword())
	throttled.Handle("POST /api/auth/verify-reset-code", authHandler.VerifyResetCode())
	mux.HandleFunc("POST /api/auth/reset-password", authHandler.ResetPassword())
	authenticated.Handle("POST /api/auth/logout", authHandler.Logout())
	authenticated.Handle("GET /api/me", handler.Me())
	registerAdminRoutes(AdminGroup(mux, deps), handler.NewAdmin(deps.People), handler.NewCatalog(deps.Catalog))
	registerLookupRoutes(authenticated, handler.NewLookups(deps.Lookups))
	registerNotificationRoutes(NotificationGroup(mux, deps), handler.NewNotifications(deps.Alerts))
	registerCRRoutes(CRGroup(mux, deps), handler.NewCR(deps.Config, deps.Classes, deps.Notifier))
	registerTeacherRoutes(TeacherGroup(mux, deps), handler.NewTeacher(deps.Config, deps.Teachers, deps.Alarms))
	registerStudentRoutes(StudentGroup(mux, deps), handler.NewStudent(deps.Config, deps.Students))
	files := deps.Files
	if files == nil {
		files = upload.NewSaver(deps.Config.UploadDir, deps.Config.MaxUploadMB)
	}
	notices := handler.NewNotice(deps.Notices, files, deps.Alarms)
	materials := handler.NewMaterial(deps.Library)
	contacts := handler.NewDirectory(deps.Contacts)
	registerNoticeRoutes(mux, deps, notices, materials)
	registerBoardRoutes(StudentGroup(mux, deps), notices, materials, contacts)
	ContactGroup(mux, deps).Handle("GET /api/teachers-info", contacts.Teachers())
	messages := handler.NewMessage(deps.Mail, files, deps.Alarms)
	polls := handler.NewPoll(deps.Config, deps.Votes, deps.Alarms)
	registerMessageRoutes(mux, deps, messages)
	registerPollRoutes(mux, deps, polls)
	mux.HandleFunc("GET "+upload.URLPrefix+"{"+upload.PathValueName+"}", files.Serve())
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

func ContactGroup(mux *http.ServeMux, deps Dependencies) RouteGroup {
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

func TeacherGroup(mux *http.ServeMux, deps Dependencies) RouteGroup {
	return RouteGroup{
		mux: mux,
		middleware: []func(http.Handler) http.Handler{
			middleware.VerifyToken(deps.Tokens, deps.Users),
			middleware.RequireRole(auth.RoleTeacher),
		},
	}
}

func StudentGroup(mux *http.ServeMux, deps Dependencies) RouteGroup {
	return RouteGroup{
		mux: mux,
		middleware: []func(http.Handler) http.Handler{
			middleware.VerifyToken(deps.Tokens, deps.Users),
			middleware.RequireRole(auth.RoleStudent, auth.RoleCR),
		},
	}
}

func registerTeacherRoutes(group RouteGroup, teacher *handler.Teacher) {
	group.Handle("GET /api/teacher/requests", teacher.Requests())
	group.Handle("POST /api/teacher/requests/{id}/approve", teacher.Approve())
	group.Handle("POST /api/teacher/requests/{id}/reject", teacher.Reject())
	group.Handle("GET /api/teacher/schedule", teacher.Schedule())
	group.Handle("GET /api/teacher/classes/upcoming", teacher.UpcomingClasses())
	group.Handle("POST /api/teacher/classes/{id}/cancel", teacher.CancelClass())
	group.Handle("POST /api/teacher/routines/{id}/cancel", teacher.CancelRoutine())
}

func registerStudentRoutes(group RouteGroup, student *handler.Student) {
	group.Handle("GET /api/students/upcoming-routine", student.UpcomingRoutine())
	group.Handle("GET /api/students/drop-course-routine", student.DropCourseRoutine())
	group.Handle("GET /api/students/exams", student.Exams())
	group.Handle("GET /api/students/routine", student.Routine())
}

func registerCRRoutes(group RouteGroup, cr *handler.CR) {
	group.Handle("GET /api/cr/routine", cr.Routine())
	group.Handle("POST /api/cr/class-requests", cr.CreateClassRequest())
	group.Handle("GET /api/cr/room-availability", cr.RoomAvailability())
	group.Handle("GET /api/cr/requests", cr.Requests())
}

func registerNoticeRoutes(mux *http.ServeMux, deps Dependencies, notices *handler.Notice, materials *handler.Material) {
	admin := AdminGroup(mux, deps)
	admin.Handle("POST /api/admin/notices", notices.AdminCreate())
	admin.Handle("GET /api/admin/notices", notices.AdminList())
	admin.Handle("DELETE /api/admin/notices/{id}", notices.AdminDelete())
	admin.Handle("POST /api/admin/materials", materials.AdminCreate())
	admin.Handle("GET /api/admin/materials", materials.AdminList())
	cr := CRGroup(mux, deps)
	cr.Handle("POST /api/cr/notices", notices.CRCreate())
	cr.Handle("GET /api/cr/notices", notices.CRList())
	cr.Handle("POST /api/cr/materials", materials.CRCreate())
	cr.Handle("GET /api/cr/materials", materials.CRList())
	cr.Handle("DELETE /api/cr/materials/{id}", materials.CRDelete())
}

func registerMessageRoutes(mux *http.ServeMux, deps Dependencies, messages *handler.Message) {
	cr := CRGroup(mux, deps)
	cr.Handle("POST /api/cr/messages", messages.CRSend())
	cr.Handle("GET /api/cr/messages/inbox", messages.CRInbox())
	cr.Handle("GET /api/cr/messages/sent", messages.CRSent())
	teacher := TeacherGroup(mux, deps)
	teacher.Handle("POST /api/teacher/messages", messages.TeacherSend())
	teacher.Handle("GET /api/teacher/messages/inbox", messages.TeacherInbox())
	teacher.Handle("GET /api/teacher/messages/sent", messages.TeacherSent())
	teacher.Handle("GET /api/teacher/lookups/crs", messages.CRLookup())
	teacher.Handle("GET /api/teacher/lookups/students", messages.StudentLookup())
	StudentGroup(mux, deps).Handle("GET /api/students/messages", messages.StudentInbox())
}

func registerPollRoutes(mux *http.ServeMux, deps Dependencies, polls *handler.Poll) {
	cr := CRGroup(mux, deps)
	cr.Handle("POST /api/cr/polls", polls.CRCreate())
	cr.Handle("POST /api/cr/polls/{id}/close", polls.CRClose())
	cr.Handle("GET /api/cr/polls", polls.CRList())
	student := StudentGroup(mux, deps)
	student.Handle("GET /api/students/polls", polls.StudentList())
	student.Handle("POST /api/students/polls/{id}/vote", polls.StudentVote())
}

func registerBoardRoutes(group RouteGroup, notices *handler.Notice, materials *handler.Material, contacts *handler.Directory) {
	group.Handle("GET /api/students/notices/cr", notices.StudentCRNotices())
	group.Handle("GET /api/students/notices/admin", notices.StudentAdminNotices())
	group.Handle("GET /api/students/materials", materials.StudentList())
	group.Handle("GET /api/students/directory", contacts.Students())
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
