package service

// Keycloak realm-role 조회의 부분일치(substring) 함정에 대한 회귀 테스트.
//
// Keycloak 의 realm-role `search` 파라미터는 부분일치이고 결과는 이름 오름차순이다.
// 따라서 "viewer" 로 검색하면 ["billviewer", "viewer"] 가 돌아오고, 기존 코드처럼
// roles[0] 을 쓰면 배정 요청한 역할이 아니라 **billviewer** 가 부여된다.
// 원격 dev 서버에서 실증됨: 그룹에 viewer 를 배정했더니 소속 사용자 토큰에
// realm_access.roles = ["billviewer"] 가 실렸다.
//
// pickExactRealmRole 은 exact 이름 일치만 선택해 이 문제를 막는다.

import (
	"testing"

	"github.com/Nerzal/gocloak/v13"
)

func rolesNamed(names ...string) []*gocloak.Role {
	out := make([]*gocloak.Role, 0, len(names))
	for _, n := range names {
		name := n
		out = append(out, &gocloak.Role{Name: &name})
	}
	return out
}

func TestPickExactRealmRole_SubstringMatchDoesNotWin(t *testing.T) {
	// Keycloak 이 이름 오름차순으로 반환하므로 billviewer 가 먼저 온다
	roles := rolesNamed("billviewer", "viewer")

	got := pickExactRealmRole(roles, "viewer")

	if got == nil || got.Name == nil {
		t.Fatalf("viewer 를 찾지 못했다")
	}
	if *got.Name != "viewer" {
		t.Fatalf("got %q, want %q — 부분일치 결과의 첫 항목을 골랐다", *got.Name, "viewer")
	}
}

func TestPickExactRealmRole_LongerNameStillSelectable(t *testing.T) {
	roles := rolesNamed("billviewer", "viewer")

	got := pickExactRealmRole(roles, "billviewer")

	if got == nil || got.Name == nil || *got.Name != "billviewer" {
		t.Fatalf("billviewer 선택 실패: %v", got)
	}
}

func TestPickExactRealmRole_AdminFamily(t *testing.T) {
	// admin / billadmin / platformAdmin 은 알파벳 순서상 우연히 admin 이 먼저라
	// 기존 코드에서도 정상 동작했으나, exact 매칭에서도 동일해야 한다
	roles := rolesNamed("admin", "billadmin", "platformAdmin")

	for _, want := range []string{"admin", "billadmin", "platformAdmin"} {
		got := pickExactRealmRole(roles, want)
		if got == nil || got.Name == nil || *got.Name != want {
			t.Fatalf("%q 선택 실패: %v", want, got)
		}
	}
}

func TestPickExactRealmRole_NoExactMatchReturnsNil(t *testing.T) {
	// 부분일치로 다른 역할을 잡아오지 않고 nil 을 반환해야 한다.
	// 호출부는 이를 not-found 로 처리한다 (IAM-BUG-026 의 500 메시지 경로).
	roles := rolesNamed("billviewer", "wfoperator")

	if got := pickExactRealmRole(roles, "viewer"); got != nil {
		t.Fatalf("exact 일치가 없는데 %v 를 반환했다", *got.Name)
	}
}

func TestPickExactRealmRole_EmptyAndNilSafe(t *testing.T) {
	if got := pickExactRealmRole(nil, "viewer"); got != nil {
		t.Fatalf("nil 슬라이스에서 %v 반환", got)
	}
	if got := pickExactRealmRole([]*gocloak.Role{}, "viewer"); got != nil {
		t.Fatalf("빈 슬라이스에서 %v 반환", got)
	}
	// Name 이 nil 인 항목이 섞여도 패닉 없이 건너뛴다
	mixed := append([]*gocloak.Role{nil, {}}, rolesNamed("viewer")...)
	got := pickExactRealmRole(mixed, "viewer")
	if got == nil || got.Name == nil || *got.Name != "viewer" {
		t.Fatalf("nil 항목 혼재 시 선택 실패: %v", got)
	}
}
