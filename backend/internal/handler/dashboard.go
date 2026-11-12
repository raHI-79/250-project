package handler

import (
	"context"
	"errors"
	"fmt"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	RangeToday    = "today"
	RangeWeek     = "7d"
	RangeMonth    = "30d"
	RangeSemester = "semester"
	RangeCustom   = "custom"

	AlertServerErrors = "server_errors"
	AlertEmailFailure = "email_failures"
	AlertPushFailure  = "push_failures"
	AlertBackupStale  = "backup_stale"
	AlertSignInSpike  = "sign_in_spike"
	AlertSheetDeletes = "sheet_deletions"

	MessageAlertDismissed = "The alert stays quiet for a day."
	MessageAlertUnknown   = "No alert goes by that name."

	SectionSystemErrors  = "server-errors"
	SectionSystemEmail   = "email"
	SectionSystemPush    = "notifications"
	SectionSystemBackup  = "backup"
	SectionSystemSignIns = "sign-ins"
	SectionSystemSheets  = "attendance-sheets"

	serverErrorSpike   = 10
	serverErrorPerCent = 100
	alertDetailLimit   = 5
	failureShare       = 0.2
	backupStaleAfter   = 36 * time.Hour
	signInSpikeLimit   = 10
	spikeWindow        = time.Hour
	semesterDays       = 180
	minFailureVolume   = 5
	minPushVolume      = 20
	minRequestVolume   = 200
	failingPathLimit   = 8
	sheetDeleteWindow  = 24 * time.Hour
)

var (
	errInvalidDate = errors.New(MessageBadDate)
	errInvalidID   = errors.New(MessageInvalidID)
)

type DashboardStore interface {
	OverviewTotals(ctx context.Context, window store.Window, dayStart time.Time) (store.OverviewTotals, error)
	VisitorSeries(ctx context.Context, window store.Window) ([]store.DayPoint, error)
	SignInsByRole(ctx context.Context, window store.Window) ([]store.Slice, error)
	ClientSplit(ctx context.Context, window store.Window) ([]store.Slice, error)
	AppVersions(ctx context.Context, window store.Window) ([]store.Slice, error)
	Devices(ctx context.Context, window store.Window) ([]store.Slice, error)
	TopPages(ctx context.Context, window store.Window) ([]store.PagePoint, error)
	BatchAttendance(ctx context.Context, window store.Window) ([]store.BatchAttendance, error)
	CourseClasses(ctx context.Context, window store.Window) ([]store.CourseClasses, error)
	UndecidedTeachers(ctx context.Context, day time.Time) ([]store.TeacherPending, error)
	AcademicTotals(ctx context.Context, window store.Window, tomorrow time.Time) (store.AcademicTotals, error)
	SystemTotals(ctx context.Context, window store.Window) (store.SystemTotals, error)
	LatestServerErrors(ctx context.Context) ([]store.ErrorPath, error)
	FailedSignInSpike(ctx context.Context, since time.Time) (int, error)
	SheetDeletions(ctx context.Context, since time.Time) (int, error)
	RequestVolume(ctx context.Context, window store.Window) (store.RequestVolume, error)
	PushVolume(ctx context.Context, window store.Window) (store.PushVolume, error)
	PushBreakdown(ctx context.Context, window store.Window) ([]store.PushBreakdown, error)
	FailingPaths(ctx context.Context, window store.Window, limit int) ([]store.FailingPath, error)
	ClientCrashes(ctx context.Context, window store.Window, limit int) ([]store.FailingPath, error)
	QuietAlerts(ctx context.Context) (map[string]time.Time, error)
	DismissAlert(ctx context.Context, key string, by int64, quiet time.Duration) error
	BackupStatus(ctx context.Context) (store.BackupState, error)
}

type Dashboard struct {
	cfg     *config.Config
	numbers DashboardStore
	started time.Time
}

func NewDashboard(cfg *config.Config, numbers DashboardStore) *Dashboard {
	return &Dashboard{cfg: cfg, numbers: numbers, started: cfg.Now()}
}

type kpiCard struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Value    int    `json:"value"`
	Previous int    `json:"previous"`
	Hint     string `json:"hint"`
	Spark    []int  `json:"spark"`
}

type seriesPoint struct {
	Day      string `json:"day"`
	Visitors int    `json:"visitors"`
	SignIns  int    `json:"sign_ins"`
}

type sliceItem struct {
	Label string `json:"label"`
	Total int    `json:"total"`
}

type pageItem struct {
	Path     string `json:"path"`
	Views    int    `json:"views"`
	Visitors int    `json:"visitors"`
}

type overviewPanel struct {
	Cards    []kpiCard     `json:"cards"`
	Series   []seriesPoint `json:"series"`
	Roles    []sliceItem   `json:"roles"`
	Clients  []sliceItem   `json:"clients"`
	Versions []sliceItem   `json:"versions"`
	Devices  []sliceItem   `json:"devices"`
	Pages    []pageItem    `json:"pages"`
}

type attendanceItem struct {
	Batch   string  `json:"batch"`
	Section string  `json:"section"`
	Average float64 `json:"average"`
	Classes int     `json:"classes"`
}

type courseClassItem struct {
	Code        string `json:"code"`
	Title       string `json:"title"`
	Held        int    `json:"held"`
	Cancelled   int    `json:"cancelled"`
	Rescheduled int    `json:"rescheduled"`
}

type academicPanel struct {
	Attendance   []attendanceItem  `json:"attendance"`
	Courses      []courseClassItem `json:"courses"`
	Undecided    []sliceItem       `json:"undecided"`
	BelowHalf    int               `json:"below_half"`
	PhonePending int               `json:"phone_pending"`
	Notices      int               `json:"notices"`
	Polls        int               `json:"polls"`
	Messages     int               `json:"messages"`
	ClassesHeld  int               `json:"classes_held"`
	Cancelled    int               `json:"cancelled"`
	Rescheduled  int               `json:"rescheduled"`
}

type errorItem struct {
	Path string `json:"path"`
	At   string `json:"at"`
}

type failingPathItem struct {
	Path    string `json:"path"`
	Total   int    `json:"total"`
	Message string `json:"message,omitempty"`
	Last    string `json:"last"`
}

type pushReasonItem struct {
	Channel string `json:"channel"`
	Reason  string `json:"reason"`
	Label   string `json:"label"`
	Total   int    `json:"total"`
}

type systemPanel struct {
	Version       string            `json:"version"`
	Commit        string            `json:"commit"`
	UptimeSeconds int               `json:"uptime_seconds"`
	DatabaseBytes int64             `json:"database_bytes"`
	FileBytes     int64             `json:"file_bytes"`
	Requests      int               `json:"requests"`
	Client4xx     int               `json:"client_4xx"`
	Server5xx     int               `json:"server_5xx"`
	ServerErrors  []errorItem       `json:"server_errors"`
	FailingPaths  []failingPathItem `json:"failing_paths"`
	ClientCrashes []failingPathItem `json:"client_crashes"`
	EmailsSent    int               `json:"emails_sent"`
	EmailsFailed  int               `json:"emails_failed"`
	PushSent      int               `json:"push_sent"`
	PushFailed    int               `json:"push_failed"`
	PushCleaned   int               `json:"push_cleaned"`
	PushShare     float64           `json:"push_failure_share"`
	PushReasons   []pushReasonItem  `json:"push_reasons"`
	FCMSent       int               `json:"fcm_sent"`
	FCMFailed     int               `json:"fcm_failed"`
	PushTokens    int               `json:"push_tokens"`
	FCMTokens     int               `json:"fcm_tokens"`
	BackupAt      string            `json:"backup_at,omitempty"`
	BackupNote    string            `json:"backup_note,omitempty"`
}

type alertDetail struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type alertItem struct {
	Key     string        `json:"key"`
	Message string        `json:"message"`
	Tone    string        `json:"tone"`
	Section string        `json:"section,omitempty"`
	Details []alertDetail `json:"details,omitempty"`
}

type dashboardResponse struct {
	Range    string        `json:"range"`
	From     string        `json:"from"`
	To       string        `json:"to"`
	Updated  string        `json:"updated_at"`
	Overview overviewPanel `json:"overview"`
	Academic academicPanel `json:"academic"`
	System   systemPanel   `json:"system"`
	Alerts   []alertItem   `json:"alerts"`
}

func (d *Dashboard) windowFrom(r *http.Request) (string, store.Window, error) {
	query := r.URL.Query()
	name := strings.TrimSpace(query.Get("range"))
	now := d.cfg.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, d.cfg.Location)
	tomorrow := today.AddDate(0, 0, 1)
	switch name {
	case "", RangeWeek:
		return RangeWeek, store.Window{From: today.AddDate(0, 0, -6), To: tomorrow}, nil
	case RangeToday:
		return RangeToday, store.Window{From: today, To: tomorrow}, nil
	case RangeMonth:
		return RangeMonth, store.Window{From: today.AddDate(0, 0, -29), To: tomorrow}, nil
	case RangeSemester:
		return RangeSemester, store.Window{From: today.AddDate(0, 0, -semesterDays), To: tomorrow}, nil
	case RangeCustom:
		from, err := d.day(query.Get("from"))
		if err != nil {
			return "", store.Window{}, err
		}
		to, err := d.day(query.Get("to"))
		if err != nil {
			return "", store.Window{}, err
		}
		if from == nil || to == nil {
			return "", store.Window{}, errInvalidDate
		}
		end := to.AddDate(0, 0, 1)
		if !end.After(*from) {
			return "", store.Window{}, errInvalidDate
		}
		return RangeCustom, store.Window{From: *from, To: end}, nil
	default:
		return "", store.Window{}, errInvalidDate
	}
}

func (d *Dashboard) day(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	day, err := time.ParseInLocation("2006-01-02", trimmed, d.cfg.Location)
	if err != nil {
		return nil, errInvalidDate
	}
	return &day, nil
}

func (d *Dashboard) Numbers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name, window, err := d.windowFrom(r)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		now := d.cfg.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, d.cfg.Location)
		tomorrow := today.AddDate(0, 0, 1)
		span := window.To.Sub(window.From)
		earlier := store.Window{From: window.From.Add(-span), To: window.From}
		body := dashboardResponse{
			Range:   name,
			From:    window.From.Format(time.RFC3339),
			To:      window.To.Format(time.RFC3339),
			Updated: now.Format(time.RFC3339),
		}
		totals, err := d.numbers.OverviewTotals(r.Context(), window, today)
		if err != nil {
			d.fail(w, err)
			return
		}
		before, err := d.numbers.OverviewTotals(r.Context(), earlier, today)
		if err != nil {
			d.fail(w, err)
			return
		}
		series, err := d.numbers.VisitorSeries(r.Context(), window)
		if err != nil {
			d.fail(w, err)
			return
		}
		visitorSpark := make([]int, 0, len(series))
		signInSpark := make([]int, 0, len(series))
		points := make([]seriesPoint, 0, len(series))
		for _, point := range series {
			visitorSpark = append(visitorSpark, point.Visitors)
			signInSpark = append(signInSpark, point.SignIns)
			points = append(points, seriesPoint{
				Day:      point.Day.Format("2006-01-02"),
				Visitors: point.Visitors,
				SignIns:  point.SignIns,
			})
		}
		body.Overview.Series = points
		body.Overview.Cards = []kpiCard{
			{
				Key: "visitors", Label: "Unique visitors", Value: totals.Visitors,
				Previous: before.Visitors, Spark: visitorSpark,
				Hint: "People who opened a page at least once in this period.",
			},
			{
				Key: "views", Label: "Page views", Value: totals.Views, Previous: before.Views,
				Spark: visitorSpark, Hint: "Every page opened, including repeat visits.",
			},
			{
				Key: "active", Label: "Active users today", Value: totals.ActiveToday,
				Previous: 0, Hint: "Accounts that did something recorded today.",
			},
			{
				Key: "sign_ins", Label: "Sign-ins", Value: totals.SignIns, Previous: before.SignIns,
				Spark: signInSpark, Hint: "Successful sign-ins of every role in this period.",
			},
			{
				Key: "registrations", Label: "New registrations", Value: totals.Registrations,
				Previous: before.Registrations, Hint: "Student accounts created in this period.",
			},
			{
				Key: "pending", Label: "Pending approvals", Value: totals.Pending, Previous: 0,
				Hint: "Registrations waiting for the office right now.",
			},
		}
		if body.Overview.Roles, err = d.slices(r.Context(), d.numbers.SignInsByRole, window); err != nil {
			d.fail(w, err)
			return
		}
		if body.Overview.Clients, err = d.slices(r.Context(), d.numbers.ClientSplit, window); err != nil {
			d.fail(w, err)
			return
		}
		if body.Overview.Versions, err = d.slices(r.Context(), d.numbers.AppVersions, window); err != nil {
			d.fail(w, err)
			return
		}
		if body.Overview.Devices, err = d.slices(r.Context(), d.numbers.Devices, window); err != nil {
			d.fail(w, err)
			return
		}
		pages, err := d.numbers.TopPages(r.Context(), window)
		if err != nil {
			d.fail(w, err)
			return
		}
		body.Overview.Pages = make([]pageItem, 0, len(pages))
		for _, page := range pages {
			body.Overview.Pages = append(body.Overview.Pages, pageItem{
				Path: page.Path, Views: page.Views, Visitors: page.Visitors,
			})
		}
		averages, err := d.numbers.BatchAttendance(r.Context(), window)
		if err != nil {
			d.fail(w, err)
			return
		}
		body.Academic.Attendance = make([]attendanceItem, 0, len(averages))
		for _, average := range averages {
			body.Academic.Attendance = append(body.Academic.Attendance, attendanceItem{
				Batch: average.Batch, Section: average.Section,
				Average: average.Average, Classes: average.Classes,
			})
		}
		courses, err := d.numbers.CourseClasses(r.Context(), window)
		if err != nil {
			d.fail(w, err)
			return
		}
		body.Academic.Courses = make([]courseClassItem, 0, len(courses))
		for _, course := range courses {
			body.Academic.Courses = append(body.Academic.Courses, courseClassItem{
				Code: course.Code, Title: course.Title, Held: course.Held,
				Cancelled: course.Cancelled, Rescheduled: course.Rescheduled,
			})
		}
		undecided, err := d.numbers.UndecidedTeachers(r.Context(), tomorrow)
		if err != nil {
			d.fail(w, err)
			return
		}
		body.Academic.Undecided = make([]sliceItem, 0, len(undecided))
		for _, teacher := range undecided {
			body.Academic.Undecided = append(body.Academic.Undecided, sliceItem{
				Label: teacher.Name, Total: teacher.Total,
			})
		}
		academic, err := d.numbers.AcademicTotals(r.Context(), window, tomorrow)
		if err != nil {
			d.fail(w, err)
			return
		}
		body.Academic.BelowHalf = academic.BelowHalf
		body.Academic.PhonePending = academic.PhonePending
		body.Academic.Notices = academic.Notices
		body.Academic.Polls = academic.Polls
		body.Academic.Messages = academic.Messages
		body.Academic.ClassesHeld = academic.ClassesHeld
		body.Academic.Cancelled = academic.Cancelled
		body.Academic.Rescheduled = academic.Rescheduled
		health, err := d.numbers.SystemTotals(r.Context(), window)
		if err != nil {
			d.fail(w, err)
			return
		}
		failures, err := d.numbers.LatestServerErrors(r.Context())
		if err != nil {
			d.fail(w, err)
			return
		}
		backup, err := d.numbers.BackupStatus(r.Context())
		if err != nil {
			d.fail(w, err)
			return
		}
		body.System = systemPanel{
			Version:       d.cfg.Version,
			Commit:        d.cfg.Commit,
			UptimeSeconds: int(now.Sub(d.started).Seconds()),
			DatabaseBytes: health.DatabaseBytes,
			FileBytes:     health.FileBytes,
			Client4xx:     health.Client4xx,
			Server5xx:     health.Server5xx,
			EmailsSent:    health.EmailsSent,
			EmailsFailed:  health.EmailsFailed,
			PushSent:      health.PushSent,
			PushFailed:    health.PushFailed,
			FCMSent:       health.FCMSent,
			FCMFailed:     health.FCMFailed,
			PushTokens:    health.PushTokens,
			FCMTokens:     health.FCMTokens,
		}
		body.System.ServerErrors = make([]errorItem, 0, len(failures))
		for _, failure := range failures {
			body.System.ServerErrors = append(body.System.ServerErrors, errorItem{
				Path: failure.Path, At: failure.At.Format(time.RFC3339),
			})
		}
		if backup.LastSuccessAt != nil {
			body.System.BackupAt = backup.LastSuccessAt.Format(time.RFC3339)
			body.System.BackupNote = backup.Note
		}
		spike, err := d.numbers.FailedSignInSpike(r.Context(), now.Add(-spikeWindow))
		if err != nil {
			d.fail(w, err)
			return
		}
		deletions, err := d.numbers.SheetDeletions(r.Context(), now.Add(-sheetDeleteWindow))
		if err != nil {
			d.fail(w, err)
			return
		}
		volume, err := d.numbers.RequestVolume(r.Context(), window)
		if err != nil {
			d.fail(w, err)
			return
		}
		pushes, err := d.numbers.PushVolume(r.Context(), window)
		if err != nil {
			d.fail(w, err)
			return
		}
		reasons, err := d.numbers.PushBreakdown(r.Context(), window)
		if err != nil {
			d.fail(w, err)
			return
		}
		paths, err := d.numbers.FailingPaths(r.Context(), window, failingPathLimit)
		if err != nil {
			d.fail(w, err)
			return
		}
		crashes, err := d.numbers.ClientCrashes(r.Context(), window, failingPathLimit)
		if err != nil {
			d.fail(w, err)
			return
		}
		quiet, err := d.numbers.QuietAlerts(r.Context())
		if err != nil {
			d.fail(w, err)
			return
		}
		body.System.Requests = volume.Requests
		body.System.PushCleaned = pushes.Cleaned
		body.System.PushShare = pushes.Share()
		body.System.PushReasons = make([]pushReasonItem, 0, len(reasons))
		for _, entry := range reasons {
			body.System.PushReasons = append(body.System.PushReasons, pushReasonItem{
				Channel: entry.Channel,
				Reason:  entry.Reason,
				Label:   pushReasonLabel(entry.Channel, entry.Reason),
				Total:   entry.Total,
			})
		}
		body.System.FailingPaths = make([]failingPathItem, 0, len(paths))
		for _, entry := range paths {
			body.System.FailingPaths = append(body.System.FailingPaths, failingPathItem{
				Path:    entry.Path,
				Total:   entry.Total,
				Message: entry.Message,
				Last:    entry.Last.Format(time.RFC3339),
			})
		}
		body.System.ClientCrashes = make([]failingPathItem, 0, len(crashes))
		for _, entry := range crashes {
			body.System.ClientCrashes = append(body.System.ClientCrashes, failingPathItem{
				Path:    entry.Path,
				Total:   entry.Total,
				Message: entry.Message,
				Last:    entry.Last.Format(time.RFC3339),
			})
		}
		body.Alerts = alertsFor(alertFacts{
			Health:    health,
			Requests:  realFailures(volume, health.Server5xx),
			Push:      pushes,
			Backup:    backup,
			Spike:     spike,
			Deletions: deletions,
			Paths:     paths,
			Quiet:     quiet,
			Now:       now,
		})
		httpx.WriteJSON(w, http.StatusOK, body)
	}
}

func (d *Dashboard) slices(ctx context.Context,
	read func(context.Context, store.Window) ([]store.Slice, error),
	window store.Window) ([]sliceItem, error) {
	rows, err := read(ctx, window)
	if err != nil {
		return nil, err
	}
	items := make([]sliceItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, sliceItem{Label: row.Label, Total: row.Total})
	}
	return items, nil
}

func (d *Dashboard) fail(w http.ResponseWriter, err error) {
	log.Printf("handler: could not build the dashboard: %v", err)
	httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
}

func failing(sent, failed int) bool {
	total := sent + failed
	if total < minFailureVolume {
		return false
	}
	return float64(failed)/float64(total) > failureShare
}

func pushAlerting(volume store.PushVolume) bool {
	if volume.Attempts() < minPushVolume {
		return false
	}
	return volume.Share() > failureShare
}

func topPaths(paths []store.FailingPath) []alertDetail {
	details := make([]alertDetail, 0, alertDetailLimit)
	for _, entry := range paths {
		if len(details) == alertDetailLimit {
			break
		}
		details = append(details, alertDetail{Label: entry.Path, Count: entry.Total})
	}
	return details
}

func realFailures(volume store.RequestVolume, recorded int) store.RequestVolume {
	if recorded < volume.Failures {
		volume.Failures = recorded
	}
	return volume
}

func serverErrorAlerting(volume store.RequestVolume) bool {
	if volume.Failures < serverErrorSpike {
		return false
	}
	return volume.Failures*serverErrorPerCent > volume.Requests
}

var pushReasonLabels = map[string]string{
	store.PushReasonExpired:     "Expired or unregistered",
	store.PushReasonInvalidKey:  "Invalid key or refused",
	store.PushReasonRateLimited: "Rate limited",
	store.PushReasonNetwork:     "Could not be reached",
	store.PushReasonPayload:     "Payload refused",
	store.PushReasonOther:       "Something else",
}

var pushChannelLabels = map[string]string{
	store.ChannelWebPush: "Web push",
	store.ChannelFCM:     "App notifications",
}

func pushReasonLabel(channel, reason string) string {
	where := pushChannelLabels[channel]
	if where == "" {
		where = channel
	}
	why := pushReasonLabels[reason]
	if why == "" {
		why = reason
	}
	return where + " · " + why
}

type alertFacts struct {
	Health    store.SystemTotals
	Requests  store.RequestVolume
	Push      store.PushVolume
	Backup    store.BackupState
	Spike     int
	Deletions int
	Paths     []store.FailingPath
	Quiet     map[string]time.Time
	Now       time.Time
}

func alertsFor(facts alertFacts) []alertItem {
	alerts := make([]alertItem, 0)
	add := func(alert alertItem) {
		if until, ok := facts.Quiet[alert.Key]; ok && until.After(facts.Now) {
			return
		}
		alerts = append(alerts, alert)
	}
	if serverErrorAlerting(facts.Requests) {
		add(alertItem{
			Key:     AlertServerErrors,
			Tone:    "error",
			Section: SectionSystemErrors,
			Message: fmt.Sprintf(
				"%d of the %d requests in this period ended in a server error.",
				facts.Requests.Failures, facts.Requests.Requests),
			Details: topPaths(facts.Paths),
		})
	}
	if failing(facts.Health.EmailsSent, facts.Health.EmailsFailed) {
		add(alertItem{
			Key:     AlertEmailFailure,
			Tone:    "error",
			Section: SectionSystemEmail,
			Message: "More than a fifth of the emails in this period failed.",
		})
	}
	if pushAlerting(facts.Push) {
		add(alertItem{
			Key:     AlertPushFailure,
			Tone:    "warning",
			Section: SectionSystemPush,
			Message: fmt.Sprintf(
				"%d of the %d notifications in this period failed, beyond the expired ones the server cleaned up.",
				facts.Push.Failed-facts.Push.Cleaned, facts.Push.Attempts()-facts.Push.Cleaned),
		})
	}
	if facts.Backup.LastSuccessAt == nil ||
		facts.Now.Sub(*facts.Backup.LastSuccessAt) > backupStaleAfter {
		add(alertItem{
			Key:     AlertBackupStale,
			Tone:    "warning",
			Section: SectionSystemBackup,
			Message: "The last database backup is older than 36 hours.",
		})
	}
	if facts.Spike >= signInSpikeLimit {
		add(alertItem{
			Key:     AlertSignInSpike,
			Tone:    "warning",
			Section: SectionSystemSignIns,
			Message: "One address failed to sign in many times in the last hour.",
		})
	}
	if facts.Deletions > 0 {
		add(alertItem{
			Key:     AlertSheetDeletes,
			Tone:    "info",
			Section: SectionSystemSheets,
			Message: "An attendance sheet was deleted in the last day.",
		})
	}
	return alerts
}

func (d *Dashboard) Dismiss() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		key := strings.TrimSpace(r.PathValue("key"))
		if !KnownAlert(key) {
			httpx.WriteError(w, http.StatusNotFound, MessageAlertUnknown)
			return
		}
		err := d.numbers.DismissAlert(r.Context(), key, current.ID, store.AlertQuietWindow)
		if err != nil {
			log.Printf("handler: could not dismiss the alert %q: %v", key, err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageAlertDismissed})
	}
}

func KnownAlert(key string) bool {
	switch key {
	case AlertServerErrors, AlertEmailFailure, AlertPushFailure, AlertBackupStale,
		AlertSignInSpike, AlertSheetDeletes:
		return true
	default:
		return false
	}
}
