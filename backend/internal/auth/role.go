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
	case RoleStudent, RoleCR, RoleTeacher, RoleAdmin, RoleSuperAdmin:
		return true
	default:
		return false
	}
}

func IsOfficeRole(role string) bool {
	return role == RoleAdmin || role == RoleSuperAdmin
}

func SessionRole(dbRole string, isCR, officeAccess bool, claimed string) (string, bool) {
	if claimed == RoleAdmin && dbRole == RoleTeacher && officeAccess {
		return RoleAdmin, true
	}
	effective := EffectiveRole(dbRole, isCR)
	return effective, effective == claimed
}
