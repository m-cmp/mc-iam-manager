package service

import (
	"context"
	"strings"
	"testing"
)

// IAM-BUG-055: 예약 realm 역할은 IAM API 경로(wrapper)로 생성·할당·회수·삭제할 수 없다.
// 차단은 Keycloak 설정(config.KC)을 읽기 전에 일어나야 한다.
func TestRealmRoleWrappers_RejectReservedNames(t *testing.T) {
	s := &keycloakService{}
	ctx := context.Background()
	calls := map[string]func(string) error{
		"CreateRealmRole":          func(n string) error { return s.CreateRealmRole(ctx, n) },
		"CreateRealmRoleAndWait":   func(n string) error { return s.CreateRealmRoleAndWait(ctx, n) },
		"AssignRealmRoleToUser":    func(n string) error { return s.AssignRealmRoleToUser(ctx, "kc-user", n) },
		"RemoveRealmRoleFromUser":  func(n string) error { return s.RemoveRealmRoleFromUser(ctx, "kc-user", n) },
		"AddRealmRoleToGroup":      func(n string) error { return s.AddRealmRoleToGroup(ctx, "grp", n) },
		"RemoveRealmRoleFromGroup": func(n string) error { return s.RemoveRealmRoleFromGroup(ctx, "grp", n) },
		"DeleteRealmRole":          func(n string) error { return s.DeleteRealmRole(ctx, n) },
	}
	for fn, call := range calls {
		for _, name := range []string{"platformAdmin", " PLATFORMADMIN ", "offline_access", "default-roles-mciam"} {
			err := call(name)
			if err == nil || !strings.Contains(err.Error(), "reserved realm role") {
				t.Errorf("%s(%q): want reserved realm role error, got %v", fn, name, err)
			}
		}
	}
}

func TestGuardReservedRealmRole_AllowsNormalNames(t *testing.T) {
	for _, name := range []string{"admin", "viewer", "운영자"} {
		if err := guardReservedRealmRole(name); err != nil {
			t.Errorf("guardReservedRealmRole(%q) = %v, want nil", name, err)
		}
	}
}
