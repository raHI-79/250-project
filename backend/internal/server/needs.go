package server

import (
	"github.com/raHI-79/250-project/backend/internal/auth"
	"sort"
	"sync"
)

type RouteNeeds struct {
	mu    sync.RWMutex
	table map[string]string
	seen  map[string]bool
}

func NewRouteNeeds() *RouteNeeds {
	return &RouteNeeds{table: make(map[string]string), seen: make(map[string]bool)}
}

func (n *RouteNeeds) Need(permission string, patterns ...string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, pattern := range patterns {
		n.table[pattern] = permission
	}
}

func (n *RouteNeeds) Seen(pattern string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.seen[pattern] = true
}

func (n *RouteNeeds) Permission(pattern string) (string, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	permission, ok := n.table[pattern]
	return permission, ok
}

func (n *RouteNeeds) Table() map[string]string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	copied := make(map[string]string, len(n.table))
	for pattern, permission := range n.table {
		copied[pattern] = permission
	}
	return copied
}

func (n *RouteNeeds) Undeclared() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	missing := make([]string, 0)
	for pattern := range n.seen {
		if _, ok := n.table[pattern]; !ok {
			missing = append(missing, pattern)
		}
	}
	sort.Strings(missing)
	return missing
}

func (n *RouteNeeds) Unused() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	extra := make([]string, 0)
	for pattern := range n.table {
		if !n.seen[pattern] {
			extra = append(extra, pattern)
		}
	}
	sort.Strings(extra)
	return extra
}

func OfficeNeeds() *RouteNeeds {
	needs := NewRouteNeeds()
	needs.Need(auth.PermRegistrations,
		"GET /api/admin/registrations",
		"POST /api/admin/registrations/{id}/approve",
		"POST /api/admin/registrations/{id}/reject",
		"GET /api/admin/rejection-reasons",
	)
	needs.Need(auth.PermStudents,
		"GET /api/admin/students",
		"POST /api/admin/students/{id}/remove",
		"POST /api/admin/crs/promote",
		"GET /api/admin/crs",
		"POST /api/admin/crs/{id}/demote",
		"POST /api/admin/crs/{id}/deactivate",
		"POST /api/admin/users/{id}/restore",
		"DELETE /api/admin/users/{id}/permanent",
	)
	needs.Need(auth.PermTeachers,
		"POST /api/admin/teachers",
		"GET /api/admin/teachers",
		"PATCH /api/admin/teachers/{id}",
		"POST /api/admin/teachers/{id}/resend-invite",
		"POST /api/admin/teachers/{id}/deactivate",
	)
	needs.Need(auth.PermCatalog,
		"POST /api/admin/courses",
		"GET /api/admin/courses",
		"DELETE /api/admin/courses/{id}",
		"GET /api/admin/batches/running-semester",
		"PUT /api/admin/batches/{batch}/running-semester",
	)
	needs.Need(auth.PermPeriods,
		"GET /api/admin/periods",
		"POST /api/admin/periods",
		"PATCH /api/admin/periods/{id}",
		"DELETE /api/admin/periods/{id}",
		"POST /api/admin/periods/reorder",
	)
	needs.Need(auth.PermRoutines,
		"GET /api/admin/calendar",
		"PUT /api/admin/calendar/holidays",
		"DELETE /api/admin/calendar/holidays/{date}",
		"PUT /api/admin/calendar/off-days",
		"POST /api/admin/routines",
		"GET /api/admin/routines",
		"PATCH /api/admin/routines/{id}",
		"DELETE /api/admin/routines/{id}",
	)
	needs.Need(auth.PermNotices,
		"POST /api/admin/notices",
		"GET /api/admin/notices",
		"PATCH /api/admin/notices/{id}",
		"DELETE /api/admin/notices/{id}",
	)
	needs.Need(auth.PermAttendance,
		"GET /api/admin/phone-changes",
		"GET /api/admin/attendance",
		"GET /api/admin/attendance/sessions",
		"GET /api/admin/attendance/sessions/{id}",
		"DELETE /api/admin/attendance/sessions/{id}",
		"DELETE /api/admin/attendance/sheet",
	)
	needs.Need(auth.PermExports,
		"GET /api/admin/attendance/export",
		"GET /api/admin/activity/export",
	)
	needs.Need("",
		"GET /api/admin/stats",
		"GET /api/admin/audit",
		"GET /api/admin/activity",
		"GET /api/admin/activity/people",
	)
	return needs
}
