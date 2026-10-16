package auth

import "testing"

func TestEffectiveRole(t *testing.T) {
	tests := []struct {
		name   string
		dbRole string
		isCR   bool
		want   string
	}{
		{name: "plain student", dbRole: RoleStudent, isCR: false, want: RoleStudent},
		{name: "student who is a class representative", dbRole: RoleStudent, isCR: true, want: RoleCR},
		{name: "teacher", dbRole: RoleTeacher, isCR: false, want: RoleTeacher},
		{name: "teacher flagged as cr is still a teacher", dbRole: RoleTeacher, isCR: true, want: RoleTeacher},
		{name: "admin", dbRole: RoleAdmin, isCR: false, want: RoleAdmin},
		{name: "admin flagged as cr is still an admin", dbRole: RoleAdmin, isCR: true, want: RoleAdmin},
		{name: "unknown role is passed through", dbRole: "principal", isCR: false, want: "principal"},
		{name: "empty role is passed through", dbRole: "", isCR: true, want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := EffectiveRole(tc.dbRole, tc.isCR); got != tc.want {
				t.Errorf("EffectiveRole(%q, %t) = %q, want %q", tc.dbRole, tc.isCR, got, tc.want)
			}
		})
	}
}

func TestIsEffectiveRole(t *testing.T) {
	tests := []struct {
		name string
		role string
		want bool
	}{
		{name: "student", role: RoleStudent, want: true},
		{name: "cr", role: RoleCR, want: true},
		{name: "teacher", role: RoleTeacher, want: true},
		{name: "admin", role: RoleAdmin, want: true},
		{name: "empty", role: "", want: false},
		{name: "unknown", role: "principal", want: false},
		{name: "wrong case", role: "Admin", want: false},
		{name: "padded", role: " admin", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsEffectiveRole(tc.role); got != tc.want {
				t.Errorf("IsEffectiveRole(%q) = %t, want %t", tc.role, got, tc.want)
			}
		})
	}
}
