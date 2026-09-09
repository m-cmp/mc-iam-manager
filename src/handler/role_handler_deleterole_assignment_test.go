package handler

import (
	"net/http"
	"testing"

	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/require"
)

// TC-IAM-BUG-035-01: 플랫폼 역할로 사용자에게 배정된 역할은 삭제 요청 시 409를 반환하고,
// role_master/role_subs가 그대로 남아 있어야 한다(부분삭제가 발생하면 안 된다).
func TestDeleteRole_AssignedToUserAsPlatformRole_Returns409AndKeepsData(t *testing.T) {
	db := setupDeleteRoleTestDB(t)
	role := model.RoleMaster{Name: "tc-iam-bug-035-platform-assigned"}
	require.NoError(t, db.Create(&role).Error)
	require.NoError(t, db.Create(&model.RoleSub{RoleID: role.ID, RoleType: constants.RoleTypePlatform}).Error)
	require.NoError(t, db.Create(&model.UserPlatformRole{UserID: 1, RoleID: role.ID}).Error)

	h := NewRoleHandler(db)
	h.keycloakService = &recordingKeycloakService{}

	c, rec := newDeleteRoleContext(t, role.ID, "platform")
	require.NoError(t, h.DeleteRole(c))
	require.Equal(t, http.StatusConflict, rec.Code)

	var masterCount, subCount int64
	require.NoError(t, db.Model(&model.RoleMaster{}).Where("id = ?", role.ID).Count(&masterCount).Error)
	require.NoError(t, db.Model(&model.RoleSub{}).Where("role_id = ?", role.ID).Count(&subCount).Error)
	require.EqualValues(t, 1, masterCount, "배정 중인 역할은 role_master가 삭제되면 안 된다")
	require.EqualValues(t, 1, subCount, "배정 중인 역할은 role_subs가 삭제되면 안 된다")
}

// TC-IAM-BUG-035-02: 워크스페이스 역할로 사용자에게 배정된 역할도 동일하게 409로 거부되어야 한다.
func TestDeleteRole_AssignedToUserAsWorkspaceRole_Returns409(t *testing.T) {
	db := setupDeleteRoleTestDB(t)
	role := model.RoleMaster{Name: "tc-iam-bug-035-workspace-assigned"}
	require.NoError(t, db.Create(&role).Error)
	require.NoError(t, db.Create(&model.RoleSub{RoleID: role.ID, RoleType: constants.RoleTypeWorkspace}).Error)
	require.NoError(t, db.Create(&model.UserWorkspaceRole{UserID: 1, WorkspaceID: 1, RoleID: role.ID}).Error)

	h := NewRoleHandler(db)
	h.keycloakService = &recordingKeycloakService{}

	c, rec := newDeleteRoleContext(t, role.ID, "workspace")
	require.NoError(t, h.DeleteRole(c))
	require.Equal(t, http.StatusConflict, rec.Code)

	var masterCount int64
	require.NoError(t, db.Model(&model.RoleMaster{}).Where("id = ?", role.ID).Count(&masterCount).Error)
	require.EqualValues(t, 1, masterCount, "배정 중인 역할은 role_master가 삭제되면 안 된다")
}

// TC-IAM-BUG-035-03: 배정되지 않은 역할은 삭제 시 role_master/role_subs/CSP 매핑/메뉴 매핑이
// 모두 하나의 트랜잭션으로 함께 삭제되어야 한다.
func TestDeleteRole_Unassigned_DeletesMasterSubsAndMappingsAtomically(t *testing.T) {
	db := setupDeleteRoleTestDB(t)
	role := model.RoleMaster{Name: "tc-iam-bug-035-unassigned"}
	require.NoError(t, db.Create(&role).Error)
	require.NoError(t, db.Create(&model.RoleSub{RoleID: role.ID, RoleType: constants.RoleTypePlatform}).Error)
	require.NoError(t, db.Create(&model.RoleSub{RoleID: role.ID, RoleType: constants.RoleTypeCSP}).Error)
	require.NoError(t, db.Create(&model.RoleMenuMapping{RoleID: role.ID, MenuID: "menu-1"}).Error)
	require.NoError(t, db.Create(&model.RoleMasterCspRoleMapping{RoleID: role.ID, AuthMethod: constants.AuthMethodOIDC, CspRoleID: 1}).Error)

	h := NewRoleHandler(db)
	h.keycloakService = &recordingKeycloakService{}

	c, rec := newDeleteRoleContext(t, role.ID, "platform")
	require.NoError(t, h.DeleteRole(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var masterCount, subCount, menuMappingCount, cspMappingCount int64
	require.NoError(t, db.Model(&model.RoleMaster{}).Where("id = ?", role.ID).Count(&masterCount).Error)
	require.NoError(t, db.Model(&model.RoleSub{}).Where("role_id = ?", role.ID).Count(&subCount).Error)
	require.NoError(t, db.Model(&model.RoleMenuMapping{}).Where("role_id = ?", role.ID).Count(&menuMappingCount).Error)
	require.NoError(t, db.Model(&model.RoleMasterCspRoleMapping{}).Where("role_id = ?", role.ID).Count(&cspMappingCount).Error)
	require.Zero(t, masterCount)
	require.Zero(t, subCount)
	require.Zero(t, menuMappingCount)
	require.Zero(t, cspMappingCount)
}
