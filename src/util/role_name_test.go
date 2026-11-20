package util

import "testing"

func TestIsReservedRoleName(t *testing.T) {
	cases := []struct {
		name     string
		reserved bool
	}{
		{"platformAdmin", true},
		{"PlatformAdmin", true},
		{"PLATFORMADMIN", true},
		{" platformAdmin ", true},
		{"\tplatformAdmin\n", true},
		{"ｐｌａｔｆｏｒｍＡｄｍｉｎ", true}, // 전각 (NFKC)
		{"offline_access", true},
		{"uma_authorization", true},
		{"default-roles-mciam", true},
		{"Default-Roles-tenant1", true},
		{"admin", false},
		{"viewer", false},
		{"platform-admin", false},
		{"platformAdmins", false},
		{"운영자", false},
		{"", false},
	}
	for _, tc := range cases {
		if got, _ := IsReservedRoleName(tc.name); got != tc.reserved {
			t.Errorf("IsReservedRoleName(%q) = %v, want %v", tc.name, got, tc.reserved)
		}
	}
}

func TestIsAdminRoleName(t *testing.T) {
	for _, n := range []string{"admin", "Admin", " ADMIN ", "ａｄｍｉｎ"} {
		if !IsAdminRoleName(n) {
			t.Errorf("IsAdminRoleName(%q) = false, want true", n)
		}
	}
	for _, n := range []string{"admins", "billadmin", "platformAdmin", ""} {
		if IsAdminRoleName(n) {
			t.Errorf("IsAdminRoleName(%q) = true, want false", n)
		}
	}
}
