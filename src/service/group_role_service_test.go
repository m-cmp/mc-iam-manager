package service

import (
	"testing"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupGroupRoleServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&model.Organization{},
		&model.Workspace{},
		&model.RoleMaster{},
		&model.GroupPlatformRole{},
		&model.GroupWorkspaceRole{},
	)
	require.NoError(t, err)
	return db
}

func seedOrg(t *testing.T, db *gorm.DB, name, code string) *model.Organization {
	t.Helper()
	org := &model.Organization{Name: name, OrganizationCode: code}
	require.NoError(t, db.Create(org).Error)
	return org
}

func seedWs(t *testing.T, db *gorm.DB, name string) *model.Workspace {
	t.Helper()
	ws := &model.Workspace{Name: name}
	require.NoError(t, db.Create(ws).Error)
	return ws
}

func seedRole(t *testing.T, db *gorm.DB, name string) *model.RoleMaster {
	t.Helper()
	role := &model.RoleMaster{Name: name}
	require.NoError(t, db.Create(role).Error)
	return role
}

// --- TC-M2-UG-025: AssignGroupWorkspace ---

// TC-M2-UG-025-01: 정상 매핑 생성
func TestGroupRoleService_AssignGroupWorkspace(t *testing.T) {
	db := setupGroupRoleServiceTestDB(t)
	svc := NewGroupRoleService(db)

	org := seedOrg(t, db, "Group-A", "SVC-001")
	ws := seedWs(t, db, "Workspace-A")
	role := seedRole(t, db, "svc-viewer")

	err := svc.AssignGroupWorkspace(org.ID, ws.ID, role.ID)
	assert.NoError(t, err)
}

// TC-M2-UG-025-02: 중복 매핑 시도 → ErrGroupWorkspaceRoleDuplicate
func TestGroupRoleService_AssignGroupWorkspace_Duplicate(t *testing.T) {
	db := setupGroupRoleServiceTestDB(t)
	svc := NewGroupRoleService(db)

	org := seedOrg(t, db, "Group-B", "SVC-002")
	ws := seedWs(t, db, "Workspace-B")
	role := seedRole(t, db, "svc-editor")

	require.NoError(t, svc.AssignGroupWorkspace(org.ID, ws.ID, role.ID))

	err := svc.AssignGroupWorkspace(org.ID, ws.ID, role.ID)
	assert.ErrorIs(t, err, repository.ErrGroupWorkspaceRoleDuplicate)
}

// TC-M2-UG-025-03: 존재하지 않는 워크스페이스 → ErrWorkspaceNotFound
func TestGroupRoleService_AssignGroupWorkspace_WorkspaceNotFound(t *testing.T) {
	db := setupGroupRoleServiceTestDB(t)
	svc := NewGroupRoleService(db)

	org := seedOrg(t, db, "Group-C", "SVC-003")
	role := seedRole(t, db, "svc-admin")

	err := svc.AssignGroupWorkspace(org.ID, 9999, role.ID)
	assert.ErrorIs(t, err, repository.ErrWorkspaceNotFound)
}

// 존재하지 않는 역할 → ErrRoleMasterNotFound
func TestGroupRoleService_AssignGroupWorkspace_RoleNotFound(t *testing.T) {
	db := setupGroupRoleServiceTestDB(t)
	svc := NewGroupRoleService(db)

	org := seedOrg(t, db, "Group-D", "SVC-004")
	ws := seedWs(t, db, "Workspace-C")

	err := svc.AssignGroupWorkspace(org.ID, ws.ID, 9999)
	assert.ErrorIs(t, err, repository.ErrRoleMasterNotFound)
}

// --- TC-M2-UG-026: GetGroupWorkspaces ---

// TC-M2-UG-026-02: 매핑 없을 때 빈 배열
func TestGroupRoleService_GetGroupWorkspaces_Empty(t *testing.T) {
	db := setupGroupRoleServiceTestDB(t)
	svc := NewGroupRoleService(db)

	org := seedOrg(t, db, "Group-E", "SVC-005")

	results, err := svc.GetGroupWorkspaces(org.ID)
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 0)
}

// TC-M2-UG-026-01: 매핑된 워크스페이스 2개 반환
func TestGroupRoleService_GetGroupWorkspaces(t *testing.T) {
	db := setupGroupRoleServiceTestDB(t)
	svc := NewGroupRoleService(db)

	org := seedOrg(t, db, "Group-F", "SVC-006")
	ws1 := seedWs(t, db, "Workspace-D")
	ws2 := seedWs(t, db, "Workspace-E")
	role := seedRole(t, db, "svc-member")

	require.NoError(t, svc.AssignGroupWorkspace(org.ID, ws1.ID, role.ID))
	require.NoError(t, svc.AssignGroupWorkspace(org.ID, ws2.ID, role.ID))

	results, err := svc.GetGroupWorkspaces(org.ID)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

// --- TC-M2-UG-027: UpdateGroupWorkspaceRole ---

// TC-M2-UG-027-01: 정상 역할 변경
func TestGroupRoleService_UpdateGroupWorkspaceRole(t *testing.T) {
	db := setupGroupRoleServiceTestDB(t)
	svc := NewGroupRoleService(db)

	org := seedOrg(t, db, "Group-G", "SVC-007")
	ws := seedWs(t, db, "Workspace-F")
	role1 := seedRole(t, db, "svc-viewer2")
	role2 := seedRole(t, db, "svc-ops")

	require.NoError(t, svc.AssignGroupWorkspace(org.ID, ws.ID, role1.ID))

	err := svc.UpdateGroupWorkspaceRole(org.ID, ws.ID, role2.ID)
	assert.NoError(t, err)
}

// TC-M2-UG-027-02: 존재하지 않는 매핑 업데이트 → ErrGroupWorkspaceRoleNotFound
func TestGroupRoleService_UpdateGroupWorkspaceRole_NotFound(t *testing.T) {
	db := setupGroupRoleServiceTestDB(t)
	svc := NewGroupRoleService(db)

	err := svc.UpdateGroupWorkspaceRole(9999, 9999, 1)
	assert.ErrorIs(t, err, repository.ErrGroupWorkspaceRoleNotFound)
}

// --- TC-M2-UG-028: RemoveGroupWorkspaceRole ---

// TC-M2-UG-028-01: 정상 매핑 삭제
func TestGroupRoleService_RemoveGroupWorkspaceRole(t *testing.T) {
	db := setupGroupRoleServiceTestDB(t)
	svc := NewGroupRoleService(db)

	org := seedOrg(t, db, "Group-H", "SVC-008")
	ws := seedWs(t, db, "Workspace-G")
	role := seedRole(t, db, "svc-dev")

	require.NoError(t, svc.AssignGroupWorkspace(org.ID, ws.ID, role.ID))

	err := svc.RemoveGroupWorkspaceRole(org.ID, ws.ID)
	assert.NoError(t, err)
}

// TC-M2-UG-028-02: 존재하지 않는 매핑 삭제 → ErrGroupWorkspaceRoleNotFound
func TestGroupRoleService_RemoveGroupWorkspaceRole_NotFound(t *testing.T) {
	db := setupGroupRoleServiceTestDB(t)
	svc := NewGroupRoleService(db)

	err := svc.RemoveGroupWorkspaceRole(9999, 9999)
	assert.ErrorIs(t, err, repository.ErrGroupWorkspaceRoleNotFound)
}

// --- GetAvailableGroupWorkspaces ---

func TestGroupRoleService_GetAvailableGroupWorkspaces(t *testing.T) {
	db := setupGroupRoleServiceTestDB(t)
	svc := NewGroupRoleService(db)

	org := seedOrg(t, db, "Group-I", "SVC-009")
	ws1 := seedWs(t, db, "Workspace-H")
	ws2 := seedWs(t, db, "Workspace-I")
	role := seedRole(t, db, "svc-lead")

	require.NoError(t, svc.AssignGroupWorkspace(org.ID, ws1.ID, role.ID))

	available, err := svc.GetAvailableGroupWorkspaces(org.ID)
	assert.NoError(t, err)
	assert.Len(t, available, 1)
	assert.Equal(t, ws2.ID, available[0].ID)
}
