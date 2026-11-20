package handler

import (
	"fmt"

	"github.com/m-cmp/mc-iam-manager/util"
)

// checkReservedRoleName Master Role 이름이 예약어(platformAdmin, Keycloak 기본 역할)면 오류 (IAM-BUG-055)
func checkReservedRoleName(name string) error {
	if reserved, reason := util.IsReservedRoleName(name); reserved {
		return fmt.Errorf("reserved role name: %q (%s)", name, reason)
	}
	return nil
}

// checkRoleRename Master Role 이름 변경을 검사한다 (IAM-BUG-055).
//   - 새 이름이 예약어면 오류
//   - admin Master Role 은 이름을 바꿀 수 없다. 쓰기 권한(Write)이 realm 역할 이름 "admin" 으로 판정되기 때문
//
// 새 이름이 비어 있으면 이름을 바꾸지 않는 요청으로 본다.
func checkRoleRename(currentName, newName string) error {
	if newName == "" {
		return nil
	}
	if err := checkReservedRoleName(newName); err != nil {
		return err
	}
	if util.IsAdminRoleName(currentName) && util.NormalizeRoleName(newName) != util.NormalizeRoleName(currentName) {
		return fmt.Errorf("the %q role cannot be renamed: write permission is granted by this role name", currentName)
	}
	return nil
}
