package server

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/middleware"
)

// routePattern matches the way every route is registered: a method, a space and a path.
var routePattern = regexp.MustCompile(`^(GET|POST|PUT|PATCH|DELETE) (/\S*)$`)

// registeredRoutes reads every route the router registers straight out of the source, so
// that the tables below cannot fall behind a route somebody adds. http.ServeMux keeps no
// list of its own patterns, and a table nobody checks for holes is a table that grows
// holes.
func registeredRoutes(t *testing.T) []string {
	t.Helper()
	found := make(map[string]bool)
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("could not list the server sources: %v", err)
	}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("could not read %s: %v", name, err)
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), name, source, 0)
		if err != nil {
			t.Fatalf("could not parse %s: %v", name, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(literal.Value)
			if err != nil {
				return true
			}
			if routePattern.MatchString(value) {
				found[value] = true
			}
			return true
		})
	}
	routes := make([]string, 0, len(found))
	for route := range found {
		routes = append(routes, route)
	}
	sort.Strings(routes)
	return routes
}

// publicRoutes are the addresses that answer without a token on purpose: the sign-in and
// registration flows, the public numbers of the landing page, the beacon the website
// posts its page views to, the health check and the app download.
func publicRoutes() map[string]string {
	return map[string]string{
		"GET /api/health":                    "the platform health probe",
		"GET /api/public/stats":              "the numbers on the landing page",
		"GET /api/auth/config":               "which sign-in options the portal offers",
		"GET /api/auth/sections":             "the sections a batch has, for the registration form",
		"GET /api/auth/user-id-available":    "whether a chosen user id is free, on the registration form",
		"GET /api/auth/drop-course-options":  "the courses a new student may carry over",
		"POST /api/auth/login":               "signing in",
		"POST /api/auth/admin/login":         "signing in to the office portal",
		"POST /api/auth/control-room/login":  "signing in to the control room",
		"POST /api/auth/two-step":            "the second step of a sign-in",
		"POST /api/auth/register":            "registering",
		"POST /api/auth/verify-email":        "the code a new student was mailed",
		"POST /api/auth/resend-verification": "another verification code",
		"POST /api/auth/google/verify-email": "verifying a student address with Google",
		"POST /api/auth/forgot-password":     "asking for a reset code",
		"POST /api/auth/verify-reset-code":   "the reset code",
		"POST /api/auth/reset-password":      "setting the new password",
		"GET /api/auth/invite":               "reading a teacher invitation",
		"POST /api/auth/activate":            "activating an invited teacher",
		"GET /api/auth/admin-invite":         "reading an office invitation",
		"POST /api/auth/admin-invite/accept": "accepting an office invitation",
		"POST /api/human/verify":             "the human check",
		"GET /api/push/public-key":           "the key a browser needs to subscribe",
		"POST /api/analytics/view":           "the page view beacon",
		"POST /api/system/backup-status":     "the backup workflow reporting in",
		"GET /api/app/android-latest":        "the version the app checks against",
		"GET /download/android":              "the app download",
	}
}

// androidOnlyRoutes are the addresses that only answer the Android app, because they
// bind or move a phone and a browser has no phone to bind.
func androidOnlyRoutes() []string {
	return []string{
		"POST /api/students/attendance/phone",
		"POST /api/students/attendance/phone-request",
	}
}

// tableRows is every row of the role table, with its query string dropped: a query is
// not part of the pattern the router registered.
func tableRows() [][2]string {
	rows := make([][2]string, 0, len(protectedRoutes()))
	for _, route := range protectedRoutes() {
		path := route.path
		if index := strings.Index(path, "?"); index >= 0 {
			path = path[:index]
		}
		rows = append(rows, [2]string{route.method, path})
	}
	return rows
}

// covers answers whether a registered pattern and a table row are the same route. A
// wildcard in the pattern stands for whatever the table row put in its place, so
// "/api/cr/items/{id}/final" is matched by "/api/cr/items/1/final" and
// "/api/me/badges/{section}/seen" by "/api/me/badges/routine/seen".
func covers(pattern, path string) bool {
	left := strings.Split(pattern, "/")
	right := strings.Split(path, "/")
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if strings.HasPrefix(left[index], "{") && strings.HasSuffix(left[index], "}") {
			if right[index] == "" {
				return false
			}
			continue
		}
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func inTable(method, pattern string) bool {
	for _, row := range tableRows() {
		if row[0] == method && covers(pattern, row[1]) {
			return true
		}
	}
	return false
}

// TestEveryApiRouteIsAccountedFor is the completeness check behind the tables above. A
// new route has to be either public on purpose, carried by the role table, or covered by
// one of the portal suites named here, so that nothing reaches production without
// somebody deciding who may call it.
func TestEveryApiRouteIsAccountedFor(t *testing.T) {
	// The office portal and control room routes carry their own permission table, which
	// TestEveryOfficeRouteDeclaresAPermission proves complete, and their own access
	// suites. They are listed by prefix rather than one by one.
	coveredPrefixes := []string{
		"GET /api/admin/", "POST /api/admin/", "PUT /api/admin/", "PATCH /api/admin/",
		"DELETE /api/admin/",
		"GET /api/office/", "POST /api/office/", "PUT /api/office/", "PATCH /api/office/",
		"DELETE /api/office/",
		"GET /api/super-admin/", "POST /api/super-admin/", "PUT /api/super-admin/",
		"PATCH /api/super-admin/", "DELETE /api/super-admin/",
	}
	public := publicRoutes()
	android := make(map[string]bool, len(androidOnlyRoutes()))
	for _, route := range androidOnlyRoutes() {
		android[route] = true
	}
	missing := make([]string, 0)
	for _, route := range registeredRoutes(t) {
		if !strings.HasPrefix(route, "GET /api/") && !strings.HasPrefix(route, "POST /api/") &&
			!strings.HasPrefix(route, "PUT /api/") && !strings.HasPrefix(route, "PATCH /api/") &&
			!strings.HasPrefix(route, "DELETE /api/") &&
			!strings.HasPrefix(route, "GET /download/") {
			continue
		}
		if _, ok := public[route]; ok {
			continue
		}
		if android[route] {
			continue
		}
		covered := false
		for _, prefix := range coveredPrefixes {
			if strings.HasPrefix(route, prefix) {
				covered = true
				break
			}
		}
		if covered {
			continue
		}
		parts := strings.SplitN(route, " ", 2)
		if inTable(parts[0], parts[1]) {
			continue
		}
		missing = append(missing, route)
	}
	if len(missing) > 0 {
		t.Errorf("these routes name nobody who may call them, add them to protectedRoutes, "+
			"publicRoutes or androidOnlyRoutes: %v", missing)
	}
}

// TestThePublicRoutesStayPublic keeps the list above honest in the other direction: a
// route somebody put on the public list has to actually answer without a token, or the
// list is lying about the surface of the portal.
func TestThePublicRoutesStayPublic(t *testing.T) {
	_, router := newAccessRouter(t)
	for route := range publicRoutes() {
		parts := strings.SplitN(route, " ", 2)
		if parts[0] != http.MethodGet {
			continue
		}
		t.Run(route, func(t *testing.T) {
			rec := serve(t, router, parts[0], parts[1], "", "")
			if rec.Code == http.StatusUnauthorized {
				t.Errorf("%s answered 401 without a token although it is on the public list", route)
			}
		})
	}
}

// TestThePhoneRoutesOnlyAnswerTheApp covers the X-Client rule from the other side: a
// phone is bound from the app, so the website is turned away even with a good token.
func TestThePhoneRoutesOnlyAnswerTheApp(t *testing.T) {
	deps, router := newAccessRouter(t)
	token := issue(t, deps, accessStudentID, auth.RoleStudent)
	for _, route := range androidOnlyRoutes() {
		parts := strings.SplitN(route, " ", 2)
		t.Run(route+" from the website", func(t *testing.T) {
			rec := serve(t, router, parts[0], parts[1], `{"device_id":"device-1"}`, token)
			if rec.Code != http.StatusForbidden {
				t.Errorf("status = %d, want 403 from the website (body = %q)",
					rec.Code, rec.Body.String())
			}
		})
	}
}

// TestAnOfficeTokenIsRefusedFromTheApp states the other half of the X-Client rule in one
// place: the office portal and the control room never answer the app, whatever the token.
func TestAnOfficeTokenIsRefusedFromTheApp(t *testing.T) {
	deps, router := newAccessRouter(t)
	// The access router knows one office account, so the rule is stated with the role
	// that account carries. The super admin follows the same path through the guard.
	for _, role := range []string{auth.RoleAdmin} {
		token := issue(t, deps, accessAdminID, role)
		rec := serveAsApp(t, router, http.MethodGet, "/api/me", token)
		if rec.Code != http.StatusForbidden {
			t.Errorf("a %s token from the app = %d, want 403 (body = %q)",
				role, rec.Code, rec.Body.String())
		}
		if got := errorField(t, rec); got != middleware.MessageAppForbidden {
			t.Errorf("error = %q, want %q", got, middleware.MessageAppForbidden)
		}
	}
}
