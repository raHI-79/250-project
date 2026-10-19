package server

import (
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/googleauth"
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
	Config    *config.Config
	RateLimit int
	Proofs    handler.ProofIssuer
	Google    googleauth.Validator
	Photos    *upload.Photos
	Accounts  handler.ProfileStore
	Database  handler.DatabasePinger
	Tokens    TokenService
	Users     middleware.UserStore
	Auth      handler.AuthStore
	People    handler.AdminStore
	Catalog   handler.CatalogStore
	Lookups   handler.LookupStore
	Alerts    handler.NotificationService
	Classes   handler.CRStore
	Notifier  handler.Notifier
	Teachers  handler.TeacherStore
	Students  handler.StudentStore
	Alarms    handler.BroadcastNotifier
	Notices   handler.NoticeStore
	Drives    handler.DriveLinkStore
	Contacts  handler.DirectoryStore
	Files     *upload.Saver
	Mailer    mailer.Sender
	Chats     handler.ConversationStore
	Shared    *upload.Attachments
	Votes     handler.PollStore
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
	photos := deps.Photos
	if photos == nil {
		photos = upload.NewPhotos(deps.Config.UploadDir)
	}
	proofs := deps.Proofs
	google := deps.Google
	if google == nil && deps.Config.GoogleEnabled() {
		google = googleauth.New()
	}
	authHandler := handler.NewAuth(deps.Config, deps.Auth, deps.Tokens, mail, proofs, google, photos)
	rateLimit := deps.RateLimit
	if rateLimit <= 0 {
		rateLimit = middleware.DefaultRateLimit
	}
	limiter := middleware.NewRateLimiter(rateLimit, middleware.DefaultRateWindow)
	throttled := RouteGroup{mux: mux, middleware: []func(http.Handler) http.Handler{limiter.Middleware}}
	authenticated := AuthenticatedGroup(mux, deps)
	mux.HandleFunc("GET /api/health", handler.Health(deps.Config, deps.Database))
	mux.HandleFunc("GET /api/auth/config", authHandler.Config())
	mux.HandleFunc("POST /api/auth/register", authHandler.Register())
	throttled.Handle("GET /api/auth/drop-course-options", authHandler.DropCourseOptions())
	throttled.Handle("POST /api/auth/google/verify-email", authHandler.GoogleVerifyEmail())
	throttled.Handle("POST /api/auth/verify-email", authHandler.VerifyEmail())
	throttled.Handle("POST /api/auth/resend-verification", authHandler.ResendVerification())
	mux.HandleFunc("GET /api/auth/invite", authHandler.Invite())
	throttled.Handle("POST /api/auth/activate", authHandler.Activate())
	throttled.Handle("POST /api/auth/login", authHandler.Login())
	throttled.Handle("POST /api/auth/admin/login", authHandler.AdminLogin())
	throttled.Handle("POST /api/auth/forgot-password", authHandler.ForgotPassword())
	throttled.Handle("POST /api/auth/verify-reset-code", authHandler.VerifyResetCode())
	mux.HandleFunc("POST /api/auth/reset-password", authHandler.ResetPassword())
	authenticated.Handle("POST /api/auth/logout", authHandler.Logout())
	authenticated.Handle("GET /api/me", handler.Me())
	registerProfileRoutes(authenticated, handler.NewProfile(deps.Config, deps.Accounts, photos))
	admin := handler.NewAdmin(deps.Config, deps.People, deps.Notifier, mail, photos)
	registerAdminRoutes(AdminGroup(mux, deps), admin, handler.NewCatalog(deps.Catalog))
	registerLookupRoutes(authenticated, handler.NewLookups(deps.Lookups))
	registerNotificationRoutes(NotificationGroup(mux, deps), handler.NewNotifications(deps.Alerts))
	registerCRRoutes(CRGroup(mux, deps), handler.NewCR(deps.Config, deps.Classes, deps.Notifier))
	registerTeacherRoutes(TeacherGroup(mux, deps), handler.NewTeacher(deps.Config, deps.Teachers, deps.Alarms))
	registerStudentRoutes(StudentGroup(mux, deps), handler.NewStudent(deps.Config, deps.Students))
	files := deps.Files
	if files == nil {
		files = upload.NewSaver(deps.Config.UploadDir, deps.Config.MaxUploadMB)
	}
	shared := deps.Shared
	if shared == nil {
		shared = upload.NewAttachments(deps.Config.UploadDir, deps.Config.MaxUploadMB)
	}
	notices := handler.NewNotice(deps.Notices, shared, deps.Alarms)
	drives := handler.NewDriveLink(deps.Drives)
	contacts := handler.NewDirectory(deps.Contacts)
	registerNoticeRoutes(mux, deps, notices)
	registerDriveLinkRoutes(CRGroup(mux, deps), StudentGroup(mux, deps), drives)
	registerBoardRoutes(StudentGroup(mux, deps), notices, contacts)
	ContactGroup(mux, deps).Handle("GET /api/teachers-info", contacts.Teachers())
	messages := handler.NewConversations(deps.Config, deps.Chats, shared, deps.Alarms)
	polls := handler.NewPoll(deps.Config, deps.Votes, deps.Alarms)
	registerMessageRoutes(mux, deps, messages)
	registerPollRoutes(mux, deps, polls)
	mux.HandleFunc("GET "+upload.URLPrefix+"{"+upload.PathValueName+"}", files.Serve())
	mux.HandleFunc("GET "+upload.AttachmentURLPrefix+"{"+upload.PathValueName+"}", shared.Serve())
	mux.HandleFunc("GET "+upload.PhotoURLPrefix+"{"+upload.PathValueName+"}", photos.Serve())
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
	group.Handle("GET /api/students/today", student.Today())
	group.Handle("GET /api/students/upcoming-routine", student.UpcomingRoutine())
	group.Handle("GET /api/students/exams", student.Exams())
	group.Handle("GET /api/students/routine", student.Routine())
}

func registerCRRoutes(group RouteGroup, cr *handler.CR) {
	group.Handle("GET /api/cr/routine", cr.Routine())
	group.Handle("GET /api/cr/courses", cr.Courses())
	group.Handle("GET /api/cr/day", cr.Day())
	group.Handle("POST /api/cr/class-requests", cr.CreateClassRequest())
	group.Handle("GET /api/cr/room-availability", cr.RoomAvailability())
	group.Handle("GET /api/cr/requests", cr.Requests())
}

func registerNoticeRoutes(mux *http.ServeMux, deps Dependencies, notices *handler.Notice) {
	admin := AdminGroup(mux, deps)
	admin.Handle("POST /api/admin/notices", notices.AdminCreate())
	admin.Handle("GET /api/admin/notices", notices.AdminList())
	admin.Handle("DELETE /api/admin/notices/{id}", notices.AdminDelete())
	cr := CRGroup(mux, deps)
	teacher := TeacherGroup(mux, deps)
	teacher.Handle("POST /api/teacher/notices", notices.TeacherCreate())
	teacher.Handle("GET /api/teacher/notices", notices.TeacherList())
	teacher.Handle("DELETE /api/teacher/notices/{id}", notices.TeacherDelete())
	cr.Handle("POST /api/cr/notices", notices.CRCreate())
	cr.Handle("GET /api/cr/notices", notices.CRList())
}

func registerDriveLinkRoutes(cr, students RouteGroup, drives *handler.DriveLink) {
	cr.Handle("GET /api/cr/drive-link", drives.CRGet())
	cr.Handle("PUT /api/cr/drive-link", drives.CRSave())
	cr.Handle("DELETE /api/cr/drive-link", drives.CRDelete())
	students.Handle("GET /api/students/drive-links", drives.StudentList())
}

func registerMessageRoutes(mux *http.ServeMux, deps Dependencies, messages *handler.Conversations) {
	group := AuthenticatedGroup(mux, deps)
	group.Handle("GET /api/messages/contacts", messages.Contacts())
	group.Handle("GET /api/messages/conversations", messages.List())
	group.Handle("POST /api/messages/direct", messages.StartDirect())
	group.Handle("POST /api/messages/groups", messages.StartGroup())
	group.Handle("GET /api/messages/conversations/{id}/messages", messages.Messages())
	group.Handle("POST /api/messages/conversations/{id}/messages", messages.Send())
	group.Handle("GET /api/messages/unread-count", messages.UnreadCount())
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

func registerBoardRoutes(group RouteGroup, notices *handler.Notice, contacts *handler.Directory) {
	group.Handle("GET /api/students/notices", notices.StudentNotices())
	group.Handle("GET /api/students/directory", contacts.Students())
}

func registerProfileRoutes(group RouteGroup, profile *handler.Profile) {
	group.Handle("GET /api/profile", profile.Get())
	group.Handle("PATCH /api/profile", profile.Patch())
	group.Handle("POST /api/profile/password", profile.ChangePassword())
	group.Handle("GET /api/people/{user_id}", profile.Person())
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
	group.Handle("POST /api/admin/crs/promote", admin.PromoteCR())
	group.Handle("GET /api/admin/crs", admin.CRs())
	group.Handle("POST /api/admin/crs/{id}/demote", admin.DemoteCR())
	group.Handle("POST /api/admin/crs/{id}/deactivate", admin.DeactivateCR())
	group.Handle("POST /api/admin/teachers", admin.CreateTeacher())
	group.Handle("GET /api/admin/teachers", admin.Teachers())
	group.Handle("PATCH /api/admin/teachers/{id}", admin.UpdateTeacher())
	group.Handle("POST /api/admin/teachers/{id}/resend-invite", admin.ResendInvite())
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
