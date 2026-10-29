package server

import (
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/googleauth"
	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/mailer"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/release"
	"github.com/raHI-79/250-project/backend/internal/turnstile"
	"github.com/raHI-79/250-project/backend/internal/upload"
	"github.com/raHI-79/250-project/backend/internal/web"
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
	Numbers   handler.PublicStore
	Devices   handler.PushStore
	PushTests handler.PushTester
	People    handler.AdminStore
	Catalog   handler.CatalogStore
	Lookups   handler.LookupStore
	Periods   handler.PeriodStore
	Alerts    handler.NotificationService
	Classes   handler.CRStore
	Notifier  handler.BroadcastNotifier
	Items     handler.ScheduleStore
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
	Badges    handler.BadgeStore
	Passes    handler.HumanPassIssuer
	Humans    turnstile.Checker
	Releases  handler.ReleaseLookup
	Marks     handler.AttendanceStore
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
		mail = mailer.New(deps.Config.MailProvider, mailer.Options{})
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
	humans := deps.Humans
	if humans == nil && deps.Config.HumanCheckEnabled() {
		humans = turnstile.New(deps.Config.TurnstileSecretKey, "")
	}
	humanOnly := handler.RequireHuman(deps.Config, deps.Passes)
	throttled := RouteGroup{mux: mux, middleware: []func(http.Handler) http.Handler{limiter.Middleware}}
	guarded := RouteGroup{
		mux:        mux,
		middleware: []func(http.Handler) http.Handler{limiter.Middleware, humanOnly},
	}
	authenticated := AuthenticatedGroup(mux, deps)
	mux.HandleFunc("GET /api/health", handler.Health(deps.Config, deps.Database))
	releases := deps.Releases
	if releases == nil {
		releases = release.New(deps.Config.ReleasesRepo, deps.Config.ReleasesAPI)
	}
	apps := handler.NewAppRelease(releases, deps.Config.FrontendOrigin)
	throttled.Handle("GET "+handler.DownloadAndroidPath, apps.Download())
	throttled.Handle("GET "+handler.AndroidLatestPath, apps.Latest())
	mux.HandleFunc("GET /api/auth/config", authHandler.Config())
	throttled.Handle("POST /api/human/verify", handler.NewHuman(deps.Config, deps.Passes, humans).Verify())
	throttled.Handle("GET /api/public/stats", handler.NewPublic(deps.Numbers, nil).Stats())
	RouteGroup{mux: mux, middleware: []func(http.Handler) http.Handler{humanOnly}}.
		Handle("POST /api/auth/register", authHandler.Register())
	throttled.Handle("GET /api/auth/drop-course-options", authHandler.DropCourseOptions())
	throttled.Handle("GET /api/auth/sections", authHandler.Sections())
	guarded.Handle("POST /api/auth/google/verify-email", authHandler.GoogleVerifyEmail())
	guarded.Handle("POST /api/auth/verify-email", authHandler.VerifyEmail())
	guarded.Handle("POST /api/auth/resend-verification", authHandler.ResendVerification())
	mux.HandleFunc("GET /api/auth/invite", authHandler.Invite())
	throttled.Handle("GET /api/auth/user-id-available", authHandler.UserIDAvailable())
	guarded.Handle("POST /api/auth/activate", authHandler.Activate())
	guarded.Handle("POST /api/auth/login", authHandler.Login())
	guarded.Handle("POST /api/auth/admin/login", authHandler.AdminLogin())
	guarded.Handle("POST /api/auth/forgot-password", authHandler.ForgotPassword())
	guarded.Handle("POST /api/auth/verify-reset-code", authHandler.VerifyResetCode())
	guarded.Handle("POST /api/auth/reset-password", authHandler.ResetPassword())
	authenticated.Handle("POST /api/auth/logout", authHandler.Logout())
	authenticated.Handle("GET /api/me", handler.Me())
	badges := handler.NewBadges(deps.Config, deps.Badges)
	authenticated.Handle("GET /api/me/badges", badges.List())
	authenticated.Handle("POST /api/me/badges/{section}/seen", badges.Seen())
	registerProfileRoutes(authenticated, handler.NewProfile(deps.Config, deps.Accounts, photos))
	admin := handler.NewAdmin(deps.Config, deps.People, deps.Notifier, mail, photos)
	periods := handler.NewPeriods(deps.Periods)
	registerAdminRoutes(AdminGroup(mux, deps), admin, handler.NewCatalog(deps.Catalog))
	registerPeriodRoutes(AdminGroup(mux, deps), periods)
	registerLookupRoutes(authenticated, handler.NewLookups(deps.Lookups))
	authenticated.Handle("GET /api/lookups/periods", periods.List())
	registerNotificationRoutes(NotificationGroup(mux, deps), handler.NewNotifications(deps.Alerts))
	registerPushRoutes(mux, deps, handler.NewPush(deps.Config, deps.Devices, deps.PushTests))
	registerCRRoutes(CRGroup(mux, deps), handler.NewCR(deps.Config, deps.Classes, deps.Alarms))
	authenticated.Handle("PATCH /api/schedule/items/{id}/link",
		handler.NewSchedule(deps.Config, deps.Items, deps.Alarms).Link())
	registerTeacherRoutes(TeacherGroup(mux, deps), handler.NewTeacher(deps.Config, deps.Teachers, deps.Alarms))
	marks := handler.NewAttendance(deps.Config, deps.Marks, deps.Alarms)
	registerAttendanceTeacherRoutes(TeacherGroup(mux, deps), marks)
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
	if deps.Config.ServesStatic() {
		mux.Handle("/", web.Static(deps.Config.StaticDir, http.HandlerFunc(notFound)))
	} else {
		mux.HandleFunc("/", notFound)
	}
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
	group.Handle("GET /api/teacher/fixed-routine", teacher.FixedRoutine())
	group.Handle("GET /api/teacher/upcoming/next-day", teacher.NextDay())
	group.Handle("GET /api/teacher/upcoming/final", teacher.FinalRoutine())
	group.Handle("POST /api/teacher/items/{id}/confirm", teacher.Confirm())
	group.Handle("POST /api/teacher/items/{id}/cancel", teacher.Cancel())
	group.Handle("POST /api/teacher/items/{id}/request-reschedule", teacher.RequestReschedule())
	group.Handle("POST /api/teacher/items/{id}/reschedule-request/accept", teacher.AcceptReschedule())
	group.Handle("POST /api/teacher/items/{id}/reschedule-request/decline", teacher.DeclineReschedule())
	group.Handle("GET /api/teacher/exams", teacher.Exams())
	group.Handle("POST /api/teacher/exams", teacher.CreateExam())
	group.Handle("PATCH /api/teacher/exams/{id}", teacher.UpdateExam())
	group.Handle("POST /api/teacher/exams/{id}/cancel", teacher.CancelExam())
	group.Handle("DELETE /api/teacher/exams/{id}", teacher.DeleteExam())
}

func registerAttendanceTeacherRoutes(group RouteGroup, marks *handler.Attendance) {
	group.Handle("GET /api/teacher/attendance/options", marks.Options())
	group.Handle("GET /api/teacher/attendance/sessions", marks.History())
	group.Handle("POST /api/teacher/attendance/sessions", marks.Start())
	group.Handle("GET /api/teacher/attendance/sessions/{id}", marks.Session())
	group.Handle("POST /api/teacher/attendance/sessions/{id}/location", marks.Location())
	group.Handle("POST /api/teacher/attendance/sessions/{id}/extend", marks.Extend())
	group.Handle("POST /api/teacher/attendance/sessions/{id}/close", marks.Close())
	group.Handle("PATCH /api/teacher/attendance/sessions/{id}/students/{student_id}", marks.EditMark())
	group.Handle("GET /api/teacher/attendance/classes", marks.Classes())
	group.Handle("GET /api/teacher/attendance/export", marks.Export())
}

func registerStudentRoutes(group RouteGroup, student *handler.Student) {
	group.Handle("GET /api/students/today", student.Today())
	group.Handle("GET /api/students/upcoming-routine", student.UpcomingRoutine())
	group.Handle("GET /api/students/exams", student.Exams())
	group.Handle("GET /api/students/routine", student.Routine())
	group.Handle("GET /api/students/courses", student.Courses())
	group.Handle("GET /api/students/drop-course-options", student.DropCourseOptions())
}

func registerCRRoutes(group RouteGroup, cr *handler.CR) {
	group.Handle("GET /api/cr/routine", cr.Routine())
	group.Handle("GET /api/cr/courses", cr.Courses())
	group.Handle("GET /api/cr/room-availability", cr.RoomAvailability())
	group.Handle("GET /api/cr/upcoming", cr.Upcoming())
	group.Handle("POST /api/cr/items", cr.CreateItem())
	group.Handle("PATCH /api/cr/items/{id}", cr.UpdateItem())
	group.Handle("DELETE /api/cr/items/{id}", cr.DeleteItem())
	group.Handle("POST /api/cr/items/{id}/request-reschedule", cr.RequestReschedule())
	group.Handle("POST /api/cr/items/{id}/cancel", cr.CancelItem())
	group.Handle("POST /api/cr/items/{id}/final", cr.AddToFinal())
	group.Handle("DELETE /api/cr/items/{id}/final", cr.RemoveFromFinal())
	group.Handle("GET /api/cr/final", cr.Final())
	group.Handle("GET /api/cr/final/preview", cr.FinalPreview())
	group.Handle("POST /api/cr/final/publish", cr.Publish())
}

func registerNoticeRoutes(mux *http.ServeMux, deps Dependencies, notices *handler.Notice) {
	admin := AdminGroup(mux, deps)
	admin.Handle("POST /api/admin/notices", notices.AdminCreate())
	admin.Handle("GET /api/admin/notices", notices.AdminList())
	admin.Handle("PATCH /api/admin/notices/{id}", notices.AdminEdit())
	admin.Handle("DELETE /api/admin/notices/{id}", notices.AdminDelete())
	cr := CRGroup(mux, deps)
	teacher := TeacherGroup(mux, deps)
	teacher.Handle("POST /api/teacher/notices", notices.TeacherCreate())
	teacher.Handle("GET /api/teacher/notices", notices.TeacherList())
	teacher.Handle("PATCH /api/teacher/notices/{id}", notices.TeacherEdit())
	teacher.Handle("DELETE /api/teacher/notices/{id}", notices.TeacherDelete())
	cr.Handle("POST /api/cr/notices", notices.CRCreate())
	cr.Handle("GET /api/cr/notices", notices.CRList())
	cr.Handle("PATCH /api/cr/notices/{id}", notices.CREdit())
	cr.Handle("DELETE /api/cr/notices/{id}", notices.CRDelete())
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
	group.Handle("PATCH /api/messages/conversations/{id}", messages.Rename())
	group.Handle("GET /api/messages/conversations/{id}/members", messages.Members())
	group.Handle("POST /api/messages/conversations/{id}/members", messages.AddMembers())
	group.Handle("DELETE /api/messages/conversations/{id}/members/{user_id}", messages.RemoveMember())
	group.Handle("POST /api/messages/conversations/{id}/members/{user_id}/admin", messages.SetMemberAdmin(true))
	group.Handle("DELETE /api/messages/conversations/{id}/members/{user_id}/admin", messages.SetMemberAdmin(false))
	group.Handle("POST /api/messages/conversations/{id}/leave", messages.Leave())
	group.Handle("DELETE /api/messages/conversations/{id}", messages.ClearChat())
	group.Handle("PATCH /api/messages/{id}", messages.Edit())
	group.Handle("DELETE /api/messages/{id}", messages.Delete())
}

func registerPollRoutes(mux *http.ServeMux, deps Dependencies, polls *handler.Poll) {
	cr := CRGroup(mux, deps)
	cr.Handle("POST /api/cr/polls", polls.CRCreate())
	cr.Handle("POST /api/cr/polls/{id}/close", polls.CRClose())
	cr.Handle("POST /api/cr/polls/{id}/reopen", polls.CRReopen())
	cr.Handle("PATCH /api/cr/polls/{id}", polls.CREdit())
	cr.Handle("DELETE /api/cr/polls/{id}", polls.CRDelete())
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

func registerPeriodRoutes(group RouteGroup, periods *handler.Periods) {
	group.Handle("GET /api/admin/periods", periods.List())
	group.Handle("POST /api/admin/periods", periods.Create())
	group.Handle("PATCH /api/admin/periods/{id}", periods.Update())
	group.Handle("DELETE /api/admin/periods/{id}", periods.Delete())
	group.Handle("POST /api/admin/periods/reorder", periods.Reorder())
}

func registerLookupRoutes(group RouteGroup, lookups *handler.Lookups) {
	group.Handle("GET /api/lookups/batches", lookups.Batches())
	group.Handle("GET /api/lookups/sections", lookups.Sections())
	group.Handle("GET /api/lookups/courses", lookups.Courses())
	group.Handle("GET /api/lookups/teachers", lookups.Teachers())
}

func registerPushRoutes(mux *http.ServeMux, deps Dependencies, pushes *handler.Push) {
	mux.HandleFunc("GET /api/push/public-key", pushes.PublicKey())
	group := NotificationGroup(mux, deps)
	group.Handle("POST /api/push/subscribe", pushes.Subscribe())
	group.Handle("DELETE /api/push/subscribe", pushes.Unsubscribe())
	group.Handle("POST /api/push/test", pushes.Test())
	group.Handle("POST /api/push/fcm", pushes.SaveToken())
	group.Handle("DELETE /api/push/fcm", pushes.DeleteToken())
	reports := middleware.NewRateLimiter(middleware.ClientLogRateLimit, middleware.DefaultRateWindow)
	limited := RouteGroup{
		mux: mux,
		middleware: []func(http.Handler) http.Handler{
			reports.Middleware,
			middleware.VerifyToken(deps.Tokens, deps.Users),
			middleware.RequireRole(auth.RoleStudent, auth.RoleCR, auth.RoleTeacher),
		},
	}
	limited.Handle("POST /api/push/client-log", pushes.ClientLog())
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
	group.Handle("POST /api/admin/users/{id}/restore", admin.Restore())
	group.Handle("DELETE /api/admin/users/{id}/permanent", admin.PermanentDelete())
	group.Handle("GET /api/admin/audit", admin.Audit())
	group.Handle("GET /api/admin/stats", admin.Stats())
	group.Handle("POST /api/admin/courses", catalog.CreateCourse())
	group.Handle("GET /api/admin/courses", catalog.Courses())
	group.Handle("DELETE /api/admin/courses/{id}", catalog.DeleteCourse())
	group.Handle("POST /api/admin/routines", catalog.CreateRoutine())
	group.Handle("GET /api/admin/routines", catalog.Routines())
	group.Handle("PATCH /api/admin/routines/{id}", catalog.MoveRoutine())
	group.Handle("DELETE /api/admin/routines/{id}", catalog.DeleteRoutine())
	group.Handle("GET /api/admin/batches/running-semester", catalog.RunningSemesters())
	group.Handle("PUT /api/admin/batches/{batch}/running-semester", catalog.SetRunningSemester())
}

func notFound(w http.ResponseWriter, r *http.Request) {
	httpx.WriteError(w, http.StatusNotFound, "Route not found")
}
