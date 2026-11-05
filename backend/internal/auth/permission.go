package auth

const (
	PermRegistrations = "registrations"
	PermStudents      = "students"
	PermTeachers      = "teachers"
	PermCatalog       = "catalog"
	PermPeriods       = "periods"
	PermRoutines      = "routines"
	PermNotices       = "notices"
	PermAttendance    = "attendance"
	PermExports       = "exports"
)

var permissionOrder = []string{
	PermRegistrations,
	PermStudents,
	PermTeachers,
	PermCatalog,
	PermPeriods,
	PermRoutines,
	PermNotices,
	PermAttendance,
	PermExports,
}

var permissionLabels = map[string]string{
	PermRegistrations: "Approve registrations",
	PermStudents:      "Manage students and CRs",
	PermTeachers:      "Invite and manage teachers",
	PermCatalog:       "Courses and semesters",
	PermPeriods:       "Class periods and rooms",
	PermRoutines:      "Routines",
	PermNotices:       "Notices",
	PermAttendance:    "Attendance and phone changes",
	PermExports:       "Exports",
}

func Permissions() []string {
	names := make([]string, len(permissionOrder))
	copy(names, permissionOrder)
	return names
}

func PermissionLabel(name string) string {
	if label, ok := permissionLabels[name]; ok {
		return label
	}
	return name
}

func IsPermission(name string) bool {
	_, ok := permissionLabels[name]
	return ok
}

func DefaultPermissions() []string {
	names := make([]string, 0, len(permissionOrder)-1)
	for _, name := range permissionOrder {
		if name == PermExports {
			continue
		}
		names = append(names, name)
	}
	return names
}

func CleanPermissions(names []string) []string {
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		if IsPermission(name) {
			seen[name] = true
		}
	}
	kept := make([]string, 0, len(seen))
	for _, name := range permissionOrder {
		if seen[name] {
			kept = append(kept, name)
		}
	}
	return kept
}
