package server

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/googleauth"
	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/mailer"
	"github.com/raHI-79/250-project/backend/internal/metrics"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/release"
	"github.com/raHI-79/250-project/backend/internal/settings"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/turnstile"
	"github.com/raHI-79/250-project/backend/internal/upload"
	"github.com/raHI-79/250-project/backend/internal/web"
	"net/http"
	"strconv"
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
	Presence  handler.StudentAttendanceStore
	Phones    handler.PhoneStore
	Office    handler.OfficeStore
	Invites   handler.OfficeInvites
	Events    handler.EventSink
	Trail     handler.AuditStore
	Views     handler.AnalyticsStore
	Figures   handler.DashboardStore
	Guards    handler.SecurityStore
	Watch     handler.DeviceWatch
	Live      middleware.SessionStore
	Locks     handler.Lockout
	Granted   middleware.PermissionStore
	Needs     *RouteNeeds
	Folks     handler.PeopleStore
	Values    *settings.Cache
	Keys      settings.Store
	Guardians handler.ControlStore
	Asks      handler.ApprovalStore
	Walls     handler.BlockStore
	Announcer handler.ReleaseAnnouncer
	Figures2  handler.DataStore
	Calendar  handler.HolidayStore
	Counter   *metrics.Counter
}

type RouteGroup struct {
	mux        *http.ServeMux
	middleware []func(http.Handler) http.Handler
	needs      *RouteNeeds
}

func (g RouteGroup) With(extra ...func(http.Handler) http.Handler) RouteGroup {
	chain := make([]func(http.Handler) http.Handler, 0, len(g.middleware)+len(extra))
	chain = append(chain, g.middleware...)
	chain = append(chain, extra...)
	return RouteGroup{mux: g.mux, middleware: chain, needs: g.needs}
}

func (g RouteGroup) Handle(pattern string, handlerFunc http.HandlerFunc) {
	if g.needs != nil {
		g.needs.Seen(pattern)
	}
	var wrapped http.Handler = handlerFunc
	for i := len(g.middleware) - 1; i >= 0; i-- {
		wrapped = g.middleware[i](wrapped)
	}
	g.mux.Handle(pattern, wrapped)
}

func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	if deps.Needs == nil {
		deps.Needs = OfficeNeeds()
	}
	if deps.Values == nil {
		deps.Values = settings.New(deps.Keys, SettingDefaults(deps.Config))
	}
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
	events := handler.NewRecorder(deps.Config, deps.Events)
	gates := handler.NewGates(deps.Values)
	blocks := handler.NewBlocks(deps.Walls, events)
	asks := handler.NewApprovals(deps.Asks, events)
	authHandler := handler.NewAuth(deps.Config, deps.Auth, deps.Tokens, mail, proofs, google, photos).
		WithOffice(deps.Invites, events).
		WithWatch(deps.Watch).
		WithLockout(deps.Locks).
		WithSettings(deps.Values)
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
		releases = release.New(deps.Config.ReleasesRepo, deps.Config.ReleasesAPI).
			WithToken(deps.Config.ReleasesToken)
	}
	apps := handler.NewAppRelease(releases, deps.Config.FrontendOrigin).WithSettings(deps.Values)
	throttled.Handle("GET "+handler.AndroidLatestPath, apps.Latest())
	if assets, ok := releases.(handler.AssetSource); ok {
		apks := handler.NewAPKs(deps.Config, assets).WithFallback(apps.Fallback)
		throttled.Handle("GET "+handler.DownloadAndroidPath, apks.Serve())
		throttled.Handle("GET "+handler.DownloadAndroidPath+"/{version}", apks.Serve())
	} else {
		throttled.Handle("GET "+handler.DownloadAndroidPath, apps.Download())
	}
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
	guarded.Handle("POST /api/auth/control-room/login", authHandler.ControlLogin())
	guarded.Handle("POST /api/auth/two-step", authHandler.TwoStepLogin())
	mux.HandleFunc("GET /api/auth/admin-invite", authHandler.AdminInvite())
	guarded.Handle("POST /api/auth/admin-invite/accept", authHandler.AcceptAdminInvite())
	guarded.Handle("POST /api/auth/forgot-password", authHandler.ForgotPassword())
	guarded.Handle("POST /api/auth/verify-reset-code", authHandler.VerifyResetCode())
	guarded.Handle("POST /api/auth/reset-password", authHandler.ResetPassword())
	authenticated.Handle("POST /api/auth/logout", authHandler.Logout())
	authenticated.Handle("GET /api/me", handler.Me())
	badges := handler.NewBadges(deps.Config, deps.Badges)
	authenticated.Handle("GET /api/me/badges", badges.List())
	authenticated.Handle("POST /api/me/badges/{section}/seen", badges.Seen())
	registerProfileRoutes(authenticated, handler.NewProfile(deps.Config, deps.Accounts, photos))
	admin := handler.NewAdmin(deps.Config, deps.People, deps.Notifier, mail, photos).
		WithEvents(events).
		WithApprovals(asks)
	periods := handler.NewPeriods(deps.Periods)
	registerAdminRoutes(AdminGroup(mux, deps), admin, handler.NewCatalog(deps.Catalog))
	registerPeriodRoutes(AdminGroup(mux, deps), periods)
	calendar := handler.NewHolidays(deps.Config, deps.Calendar, deps.Values, deps.Alarms)
	registerHolidayRoutes(AdminGroup(mux, deps), authenticated, calendar)
	registerSuperAdminRoutes(SuperAdminGroup(mux, deps),
		handler.NewSuperAdmin(deps.Config, deps.Office, mail, events))
	security := handler.NewSecurity(deps.Config, deps.Guards, mail, events)
	keepers := OfficeAccountGroup(mux, deps)
	keepers.Handle("GET /api/office/security", security.State())
	keepers.Handle("GET /api/office/permissions", security.Granted())
	keepers.Handle("POST /api/office/security/confirm", security.Confirm())
	keepers.Handle("POST /api/office/security/password", security.ChangePassword())
	keepers.Handle("PATCH /api/office/security/contact", security.ChangeContact())
	keepers.Handle("PUT /api/office/security/two-step", security.TwoStep())
	keepers.Handle("GET /api/office/security/sessions", security.Sessions())
	keepers.Handle("DELETE /api/office/security/sessions/{id}", security.CloseSession())
	keepers.Handle("DELETE /api/office/security/sessions", security.CloseOtherSessions())
	activity := handler.NewActivity(deps.Config, deps.Trail)
	officeTrail := AdminGroup(mux, deps)
	officeTrail.Handle("GET /api/admin/activity", activity.List())
	officeTrail.Handle("GET /api/admin/activity/people", activity.People())
	officeTrail.Handle("GET /api/admin/activity/export", activity.Export())
	registerPeopleRoutes(SuperAdminGroup(mux, deps), handler.NewPeople(deps.Config, deps.Folks, events))
	chiefOnly := SuperAdminGroup(mux, deps)
	chiefOnly.Handle("GET /api/super-admin/activity/{id}", activity.Monitor())
	figures := handler.NewDashboard(deps.Config, deps.Figures)
	chiefOnly.Handle("GET /api/super-admin/dashboard", figures.Numbers())
	chiefOnly.Handle("POST /api/super-admin/alerts/{key}/dismiss", figures.Dismiss())
	beacons := middleware.NewRateLimiter(middleware.ClientLogRateLimit, middleware.DefaultRateWindow)
	views := handler.NewAnalytics(deps.Config, deps.Views).WithEvents(events)
	RouteGroup{mux: mux, middleware: []func(http.Handler) http.Handler{beacons.Middleware}}.
		Handle("POST /api/analytics/view", views.View())
	crashes := middleware.NewRateLimiter(middleware.ClientLogRateLimit, middleware.DefaultRateWindow)
	RouteGroup{mux: mux, middleware: []func(http.Handler) http.Handler{crashes.Middleware}}.
		Handle("POST /api/client-log", views.Crash())
	throttled.Handle("POST /api/system/backup-status", views.BackupStatus())
	registerLookupRoutes(authenticated, handler.NewLookups(deps.Lookups))
	authenticated.Handle("GET /api/lookups/periods", periods.List())
	registerNotificationRoutes(NotificationGroup(mux, deps), handler.NewNotifications(deps.Alerts))
	registerPushRoutes(mux, deps, handler.NewPush(deps.Config, deps.Devices, deps.PushTests))
	registerCRRoutes(CRGroup(mux, deps),
		handler.NewCR(deps.Config, deps.Classes, deps.Alarms).WithSettings(deps.Values))
	authenticated.Handle("PATCH /api/schedule/items/{id}/link",
		handler.NewSchedule(deps.Config, deps.Items, deps.Alarms).Link())
	registerTeacherRoutes(TeacherGroup(mux, deps),
		handler.NewTeacher(deps.Config, deps.Teachers, deps.Alarms).WithSettings(deps.Values))
	marks := handler.NewAttendance(deps.Config, deps.Marks, deps.Alarms)
	attendanceOn := gates.Feature(settings.KeyAttendance)
	registerAttendanceTeacherRoutes(TeacherGroup(mux, deps).With(attendanceOn), marks)
	registerAttendanceStudentRoutes(StudentGroup(mux, deps).With(attendanceOn),
		StudentGroup(mux, deps).With(attendanceOn, gates.WebAttendance),
		handler.NewStudentAttendance(deps.Config, deps.Presence, deps.Alarms))
	phones := handler.NewPhones(deps.Config, deps.Phones, deps.Alarms).WithApprovals(asks)
	registerPhoneRoutes(mux, deps, phones)
	asks.Handle(handler.ActionDeleteUser, func(ctx context.Context, request store.Approval) error {
		userID, err := strconv.ParseInt(request.TargetID, 10, 64)
		if err != nil {
			return err
		}
		return admin.RunPermanentDelete(ctx, userID)
	})
	asks.Handle(handler.ActionDeleteSheet, func(ctx context.Context, request store.Approval) error {
		return phones.RunDeleteSheet(ctx, request.Payload)
	})
	keepers.Handle("GET /api/office/approvals", asks.List())
	keepers.Handle("POST /api/office/approvals/{id}/approve", asks.Decide(true))
	keepers.Handle("POST /api/office/approvals/{id}/decline", asks.Decide(false))
	control := handler.NewControl(deps.Config, deps.Values, deps.Guardians, blocks, releases,
		deps.Announcer, events).WithData(deps.Figures2)
	chiefOnly.Handle("GET /api/super-admin/settings", control.Settings())
	chiefOnly.Handle("PUT /api/super-admin/settings", control.SaveSettings())
	chiefOnly.Handle("GET /api/super-admin/security", control.Security())
	officeWeight := AdminGroup(mux, deps)
	officeWeight.Handle("GET /api/admin/attendance/late-weight", control.LateWeight())
	officeWeight.Handle("PUT /api/admin/attendance/late-weight", control.SaveLateWeight())
	chiefOnly.Handle("POST /api/super-admin/security/blocks", control.Block())
	chiefOnly.Handle("DELETE /api/super-admin/security/blocks/{id}", control.Unblock())
	chiefOnly.Handle("POST /api/super-admin/security/unlock/{id}", control.Unlock())
	chiefOnly.Handle("GET /api/super-admin/data", control.Data())
	chiefOnly.Handle("GET /api/super-admin/release", control.Release())
	chiefOnly.Handle("POST /api/super-admin/release/announce", control.AnnounceRelease())
	chiefOnly.Handle("GET /api/super-admin/calendar", calendar.Calendar())
	chiefOnly.Handle("PUT /api/super-admin/calendar/holidays", calendar.Save())
	chiefOnly.Handle("DELETE /api/super-admin/calendar/holidays/{date}", calendar.Delete())
	chiefOnly.Handle("PUT /api/super-admin/calendar/off-days", calendar.SaveOffDays())
	registerStudentRoutes(StudentGroup(mux, deps),
		handler.NewStudent(deps.Config, deps.Students).WithSettings(deps.Values))
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
	registerMessageRoutes(mux, deps, messages, gates.Feature(settings.KeyMessages),
		gates.Feature(settings.KeyGroups))
	registerPollRoutes(mux, deps, polls, gates.Feature(settings.KeyPolls))
	mux.HandleFunc("GET "+upload.URLPrefix+"{"+upload.PathValueName+"}", files.Serve())
	mux.HandleFunc("GET "+upload.AttachmentURLPrefix+"{"+upload.PathValueName+"}", shared.Serve())
	mux.HandleFunc("GET "+upload.PhotoURLPrefix+"{"+upload.PathValueName+"}", photos.Serve())
	if deps.Config.ServesStatic() {
		mux.Handle("/", web.Static(deps.Config.StaticDir, http.HandlerFunc(notFound)))
	} else {
		mux.HandleFunc("/", notFound)
	}
	var root http.Handler = mux
	if sink, ok := deps.Events.(metrics.Sink); ok {
		root = metrics.Responses(sink, deps.Counter)(root)
	} else if deps.Counter != nil {
		root = metrics.Responses(nil, deps.Counter)(root)
	}
	root = gates.Maintenance(root)
	if deps.Walls != nil {
		root = blocks.Middleware(root)
	}
	return middleware.Recoverer(middleware.Logger(middleware.CORS(deps.Config.FrontendOrigin)(root)))
}

func OfficeAccountGroup(mux *http.ServeMux, deps Dependencies) RouteGroup {
	return RouteGroup{
		mux: mux,
		middleware: []func(http.Handler) http.Handler{
			Tokens(deps),
			handler.Trail(handler.NewRecorder(deps.Config, deps.Events)),
			middleware.RequireRole(auth.RoleAdmin, auth.RoleSuperAdmin),
		},
	}
}

func Tokens(deps Dependencies) func(http.Handler) http.Handler {
	guard := middleware.NewGuard(deps.Tokens, deps.Users)
	if deps.Live != nil {
		guard = guard.WithSessions(deps.Live, handler.NewVisitors(deps.Config))
	}
	return guard.Middleware
}

func AuthenticatedGroup(mux *http.ServeMux, deps Dependencies) RouteGroup {
	return RouteGroup{
		mux: mux,
		middleware: []func(http.Handler) http.Handler{
			Tokens(deps),
			handler.Trail(handler.NewRecorder(deps.Config, deps.Events)),
		},
	}
}

func AdminGroup(mux *http.ServeMux, deps Dependencies) RouteGroup {
	return RouteGroup{
		mux:   mux,
		needs: deps.Needs,
		middleware: []func(http.Handler) http.Handler{
			Tokens(deps),
			handler.Trail(handler.NewRecorder(deps.Config, deps.Events)),
			middleware.RequireRole(auth.RoleAdmin, auth.RoleSuperAdmin),
			middleware.ReadOnly(auth.RoleSuperAdmin),
			middleware.RequirePermission(deps.Needs, deps.Granted),
		},
	}
}

func SuperAdminGroup(mux *http.ServeMux, deps Dependencies) RouteGroup {
	return RouteGroup{
		mux: mux,
		middleware: []func(http.Handler) http.Handler{
			Tokens(deps),
			handler.Trail(handler.NewRecorder(deps.Config, deps.Events)),
			middleware.RequireRole(auth.RoleSuperAdmin),
			handler.RequireSudo(deps.Config),
		},
	}
}

func registerPeopleRoutes(group RouteGroup, folks *handler.People) {
	group.Handle("GET /api/super-admin/users", folks.List())
	group.Handle("GET /api/super-admin/users/export", folks.Export())
	group.Handle("GET /api/super-admin/users/{id}", folks.Detail())
	group.Handle("POST /api/super-admin/users/{id}/suspend", folks.Suspend())
	group.Handle("POST /api/super-admin/users/{id}/restore", folks.Restore())
	group.Handle("POST /api/super-admin/users/{id}/sign-out", folks.SignOut())
	group.Handle("POST /api/super-admin/users/{id}/reset-phone", folks.ResetPhone())
	group.Handle("PUT /api/super-admin/users/{id}/role", folks.SetRole())
	group.Handle("POST /api/super-admin/users/bulk", folks.Bulk())
}

func registerSuperAdminRoutes(group RouteGroup, chief *handler.SuperAdmin) {
	group.Handle("GET /api/super-admin/admins", chief.Admins())
	group.Handle("POST /api/super-admin/admins/invite", chief.Invite())
	group.Handle("POST /api/super-admin/admins/{id}/resend-invite", chief.ResendInvite())
	group.Handle("DELETE /api/super-admin/admins/{id}/invite", chief.CancelInvite())
	group.Handle("POST /api/super-admin/admins/{id}/suspend", chief.Suspend())
	group.Handle("POST /api/super-admin/admins/{id}/restore", chief.Restore())
	group.Handle("DELETE /api/super-admin/admins/{id}", chief.Remove())
	group.Handle("GET /api/super-admin/permissions", chief.PermissionChoices())
	group.Handle("PUT /api/super-admin/admins/{id}/permissions", chief.SetPermissions())
	group.Handle("GET /api/super-admin/teachers", chief.Teachers())
	group.Handle("POST /api/super-admin/teachers/{id}/office-access", chief.SetOfficeAccess(true))
	group.Handle("DELETE /api/super-admin/teachers/{id}/office-access", chief.SetOfficeAccess(false))
}

func NotificationGroup(mux *http.ServeMux, deps Dependencies) RouteGroup {
	return RouteGroup{
		mux: mux,
		middleware: []func(http.Handler) http.Handler{
			Tokens(deps),
			handler.Trail(handler.NewRecorder(deps.Config, deps.Events)),
			middleware.RequireRole(auth.RoleStudent, auth.RoleCR, auth.RoleTeacher),
		},
	}
}

func ContactGroup(mux *http.ServeMux, deps Dependencies) RouteGroup {
	return RouteGroup{
		mux: mux,
		middleware: []func(http.Handler) http.Handler{
			Tokens(deps),
			handler.Trail(handler.NewRecorder(deps.Config, deps.Events)),
			middleware.RequireRole(auth.RoleStudent, auth.RoleCR, auth.RoleTeacher),
		},
	}
}

func CRGroup(mux *http.ServeMux, deps Dependencies) RouteGroup {
	return RouteGroup{
		mux: mux,
		middleware: []func(http.Handler) http.Handler{
			Tokens(deps),
			handler.Trail(handler.NewRecorder(deps.Config, deps.Events)),
			middleware.RequireRole(auth.RoleCR),
		},
	}
}

func TeacherGroup(mux *http.ServeMux, deps Dependencies) RouteGroup {
	return RouteGroup{
		mux: mux,
		middleware: []func(http.Handler) http.Handler{
			Tokens(deps),
			handler.Trail(handler.NewRecorder(deps.Config, deps.Events)),
			middleware.RequireRole(auth.RoleTeacher),
		},
	}
}

func StudentGroup(mux *http.ServeMux, deps Dependencies) RouteGroup {
	return RouteGroup{
		mux: mux,
		middleware: []func(http.Handler) http.Handler{
			Tokens(deps),
			handler.Trail(handler.NewRecorder(deps.Config, deps.Events)),
			middleware.RequireRole(auth.RoleStudent, auth.RoleCR),
		},
	}
}

func registerTeacherRoutes(group RouteGroup, teacher *handler.Teacher) {
	group.Handle("GET /api/teacher/fixed-routine", teacher.FixedRoutine())
	group.Handle("GET /api/teacher/upcoming", teacher.Upcoming())
	group.Handle("GET /api/teacher/upcoming/next-day", teacher.NextDay())
	group.Handle("GET /api/teacher/upcoming/final", teacher.FinalRoutine())
	group.Handle("PATCH /api/teacher/items/{id}", teacher.UpdateItem())
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
	group.Handle("POST /api/teacher/attendance/sessions/{id}/code-only", marks.CodeOnly())
	group.Handle("POST /api/teacher/attendance/sessions/{id}/close", marks.Close())
	group.Handle("POST /api/teacher/attendance/sessions/{id}/mark-all", marks.MarkAll())
	group.Handle("DELETE /api/teacher/attendance/sessions/{id}", marks.DeleteSession())
	group.Handle("DELETE /api/teacher/attendance/sheet", marks.DeleteSheet())
	group.Handle("PATCH /api/teacher/attendance/sessions/{id}/students/{student_id}", marks.EditMark())
	group.Handle("PATCH /api/teacher/attendance/sessions/{id}/students/{student_id}/note", marks.SaveNote())
	group.Handle("GET /api/teacher/attendance/classes", marks.Classes())
	group.Handle("GET /api/teacher/attendance/export", marks.Export())
}

func registerPhoneRoutes(mux *http.ServeMux, deps Dependencies, phones *handler.Phones) {
	teacher := TeacherGroup(mux, deps)
	teacher.Handle("GET /api/teacher/phone-requests", phones.TeacherList())
	teacher.Handle("POST /api/teacher/phone-requests/{id}/approve", phones.Decide(true))
	teacher.Handle("POST /api/teacher/phone-requests/{id}/reject", phones.Decide(false))
	admin := AdminGroup(mux, deps)
	admin.Handle("GET /api/admin/phone-changes", phones.AdminHistory())
	admin.Handle("GET /api/admin/attendance", phones.AdminOverview())
	admin.Handle("GET /api/admin/attendance/export", phones.AdminExport())
	admin.Handle("GET /api/admin/attendance/sessions", phones.AdminSessions())
	admin.Handle("GET /api/admin/attendance/sessions/{id}", phones.AdminSession())
	admin.Handle("DELETE /api/admin/attendance/sessions/{id}", phones.AdminDeleteSession())
	admin.Handle("DELETE /api/admin/attendance/sheet", phones.AdminDeleteSheet())
}

func registerAttendanceStudentRoutes(group, marking RouteGroup,
	presence *handler.StudentAttendance) {
	group.Handle("GET /api/students/attendance/open", presence.Open())
	marking.Handle("POST /api/students/attendance/mark", presence.Mark())
	group.Handle("GET /api/students/attendance/mine", presence.Mine())
	group.Handle("POST /api/students/attendance/phone", presence.RegisterPhone())
	group.Handle("POST /api/students/attendance/phone-request", presence.RequestPhone())
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
	group.Handle("GET /api/cr/course-teachers", cr.CourseTeachers())
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
	group.Handle("POST /api/cr/final/remove", cr.RemoveManyFromFinal())
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

func registerMessageRoutes(mux *http.ServeMux, deps Dependencies, messages *handler.Conversations,
	messagesOn, groupsOn func(http.Handler) http.Handler) {
	group := AuthenticatedGroup(mux, deps).With(messagesOn)
	group.Handle("GET /api/messages/contacts", messages.Contacts())
	group.Handle("GET /api/messages/conversations", messages.List())
	group.Handle("POST /api/messages/direct", messages.StartDirect())
	group.With(groupsOn).Handle("POST /api/messages/groups", messages.StartGroup())
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

func registerPollRoutes(mux *http.ServeMux, deps Dependencies, polls *handler.Poll,
	pollsOn func(http.Handler) http.Handler) {
	cr := CRGroup(mux, deps).With(pollsOn)
	cr.Handle("POST /api/cr/polls", polls.CRCreate())
	cr.Handle("POST /api/cr/polls/{id}/close", polls.CRClose())
	cr.Handle("POST /api/cr/polls/{id}/reopen", polls.CRReopen())
	cr.Handle("PATCH /api/cr/polls/{id}", polls.CREdit())
	cr.Handle("DELETE /api/cr/polls/{id}", polls.CRDelete())
	cr.Handle("GET /api/cr/polls", polls.CRList())
	student := StudentGroup(mux, deps).With(pollsOn)
	student.Handle("GET /api/students/polls", polls.StudentList())
	student.Handle("POST /api/students/polls/{id}/vote", polls.StudentVote())
}

func registerBoardRoutes(group RouteGroup, notices *handler.Notice, contacts *handler.Directory) {
	group.Handle("GET /api/students/notices", notices.StudentNotices())
	group.Handle("GET /api/students/notice/{id}", notices.StudentNotice())
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
			Tokens(deps),
			handler.Trail(handler.NewRecorder(deps.Config, deps.Events)),
			middleware.RequireRole(auth.RoleStudent, auth.RoleCR, auth.RoleTeacher),
		},
	}
	limited.Handle("POST /api/push/client-log", pushes.ClientLog())
}

func registerNotificationRoutes(group RouteGroup, notifications *handler.Notifications) {
	group.Handle("GET /api/notifications", notifications.List())
	group.Handle("GET /api/notifications/unread-count", notifications.UnreadCount())
	group.Handle("POST /api/notifications/read", notifications.Read())
	group.Handle("POST /api/notifications/read-all", notifications.ReadAll())
}

func registerAdminRoutes(group RouteGroup, admin *handler.Admin, catalog *handler.Catalog) {
	group.Handle("GET /api/admin/registrations", admin.Registrations())
	group.Handle("POST /api/admin/registrations/{id}/approve", admin.ApproveRegistration())
	group.Handle("POST /api/admin/registrations/{id}/reject", admin.RejectRegistration())
	group.Handle("GET /api/admin/rejection-reasons", admin.RejectionReasons())
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

func registerHolidayRoutes(office, everyone RouteGroup, calendar *handler.Holidays) {
	office.Handle("GET /api/admin/calendar", calendar.Calendar())
	office.Handle("PUT /api/admin/calendar/holidays", calendar.Save())
	office.Handle("DELETE /api/admin/calendar/holidays/{date}", calendar.Delete())
	office.Handle("PUT /api/admin/calendar/off-days", calendar.SaveOffDays())
	everyone.Handle("GET /api/calendar", calendar.Calendar())
}

func notFound(w http.ResponseWriter, r *http.Request) {
	httpx.WriteError(w, http.StatusNotFound, "Route not found")
}
