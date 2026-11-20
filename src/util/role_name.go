package util

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

// 역할 이름 예약어 (IAM-BUG-055)
//
// Master Role 이름은 플랫폼 할당 시 같은 이름의 Keycloak realm 역할로 부여되고, IAM 은 권한을
// 토큰 realm 역할 이름 문자열로 판정한다. 그래서 Master Role 이 예약된 realm 역할 이름을 가지면
// 할당만으로 그 권한(platformAdmin 등)이 부여되거나, 삭제 시 realm 역할 자체가 지워진다.

// reservedRoleNames 정규화(NormalizeRoleName)한 값으로 비교하는 예약어
var reservedRoleNames = map[string]string{
	"platformadmin":     "platform administrator realm role",
	"offline_access":    "keycloak default realm role",
	"uma_authorization": "keycloak default realm role",
}

// reservedRoleNamePrefixes 정규화한 값의 접두사로 비교하는 예약어 (default-roles-<realm>)
var reservedRoleNamePrefixes = []string{"default-roles-"}

// adminRoleName 쓰기 권한 판정에 쓰는 Master Role 이름 (이름 변경 금지)
const adminRoleName = "admin"

// NormalizeRoleName 역할 이름 비교용 정규화: 앞뒤 공백 제거 → Unicode NFKC(전각 등 호환 문자 통일) → 소문자
func NormalizeRoleName(name string) string {
	return strings.ToLower(norm.NFKC.String(strings.TrimSpace(name)))
}

// IsReservedRoleName 역할 이름이 예약어인지 확인한다. 예약어면 사유를 함께 돌려준다.
func IsReservedRoleName(name string) (bool, string) {
	n := NormalizeRoleName(name)
	if reason, ok := reservedRoleNames[n]; ok {
		return true, reason
	}
	for _, prefix := range reservedRoleNamePrefixes {
		if strings.HasPrefix(n, prefix) {
			return true, "keycloak default realm role"
		}
	}
	return false, ""
}

// IsAdminRoleName 쓰기 권한 판정에 쓰이는 admin Master Role 인지 확인한다.
func IsAdminRoleName(name string) bool {
	return NormalizeRoleName(name) == adminRoleName
}
