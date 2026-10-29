package auth

const (
	RoleStudent    = "student"
	RoleCR         = "cr"
	RoleTeacher    = "teacher"
	RoleAdmin      = "admin"
	RoleSuperAdmin = "super_admin"
)

func EffectiveRole(dbRole string, isCR bool) string {
	if dbRole == RoleStudent && isCR {
		return RoleCR
	}
	return dbRole
}

func IsEffectiveRole(role string) bool {
	switch role {
	case RoleStudent, RoleCR, RoleTeacher, RoleAdmin:
		return true
	default:
		return false
	}
}
