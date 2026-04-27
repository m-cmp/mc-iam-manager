package repository

import (
	"testing"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupGroupRoleTestDB(t *testing.T) *gorm.DB {
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

func seedOrganization(t *testing.T, db *gorm.DB, name, code string) *model.Organization {
	t.Helper()
	org := &model.Organization{Name: name, OrganizationCode: code}
	require.NoError(t, db.Create(org).Error)
	return org
}

func seedWorkspace(t *testing.T, db *gorm.DB, name string) *model.Workspace {
	t.Helper()
	ws := &model.Workspace{Name: name}
	require.NoError(t, db.Create(ws).Error)
	return ws
}

func seedRoleMaster(t *testing.T, db *gorm.DB, name string) *model.RoleMaster {
	t.Helper()
	role := &model.RoleMaster{Name: name}
	require.NoError(t, db.Create(role).Error)
	return role
}

// --- TC-M2-UG-025: CreateGroupWorkspaceRole ---

// TC-M2-UG-025-01: 정상 매핑 생성
func TestGroupRoleRepository_CreateGroupWorkspaceRole(t *testing.T) {
	db := setupGroupRoleTestDB(t)
	repo := NewGroupRoleRepository(db)

	org := seedOrganization(t, db, "TestGroup", "GRP-001")
	ws := seedWorkspace(t, db, "TestWorkspace")
	role := seedRoleMaster(t, db, "viewer")

	err := repo.CreateGroupWorkspaceRole(org.ID, ws.ID, role.ID)
	assert.NoError(t, err)

	var count int64
	db.Model(&model.GroupWorkspaceRole{}).
		Where("group_id = ? AND workspace_id = ? AND role_id = ?", org.ID, ws.ID, role.ID).
		Count(&count)
	assert.Equal(t, int64(1), count)
}

// TC-M2-UG-025-02: 중복 매핑 시도 → ErrGroupWorkspaceRoleDuplicate
func TestGroupRoleRepository_CreateGroupWorkspaceRole_Duplicate(t *testing.T) {
	db := setupGroupRoleTestDB(t)
	repo := NewGroupRoleRepository(db)

	org := seedOrganization(t, db, "TestGroup", "GRP-002")
	ws := seedWorkspace(t, db, "TestWorkspace2")
	role := seedRoleMaster(t, db, "editor")

	require.NoError(t, repo.CreateGroupWorkspaceRole(org.ID, ws.ID, role.ID))

	err := repo.CreateGroupWorkspaceRole(org.ID, ws.ID, role.ID)
	assert.ErrorIs(t, err, ErrGroupWorkspaceRoleDuplicate)
}

// --- TC-M2-UG-026: FindGroupWorkspaceRoles ---

// TC-M2-UG-026-02: 매핑 없을 때 빈 배열 반환
func TestGroupRoleRepository_FindGroupWorkspaceRoles_Empty(t *testing.T) {
	db := setupGroupRoleTestDB(t)
	repo := NewGroupRoleRepository(db)

	org := seedOrganization(t, db, "EmptyGroup", "GRP-003")

	results, err := repo.FindGroupWorkspaceRoles(org.ID)
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 0)
}

// TC-M2-UG-026-01: 매핑된 워크스페이스 목록 반환
func TestGroupRoleRepository_FindGroupWorkspaceRoles(t *testing.T) {
	db := setupGroupRoleTestDB(t)
	repo := NewGroupRoleRepository(db)

	org := seedOrganization(t, db, "MappedGroup", "GRP-004")
	ws1 := seedWorkspace(t, db, "Workspace-A")
	ws2 := seedWorkspace(t, db, "Workspace-B")
	role := seedRoleMaster(t, db, "member")

	require.NoError(t, repo.CreateGroupWorkspaceRole(org.ID, ws1.ID, role.ID))
	require.NoError(t, repo.CreateGroupWorkspaceRole(org.ID, ws2.ID, role.ID))

	results, err := repo.FindGroupWorkspaceRoles(org.ID)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

// --- TC-M2-UG-027: UpdateGroupWorkspaceRole ---

// TC-M2-UG-027-01: 정상 역할 변경
func TestGroupRoleRepository_UpdateGroupWorkspaceRole(t *testing.T) {
	db := setupGroupRoleTestDB(t)
	repo := NewGroupRoleRepository(db)

	org := seedOrganization(t, db, "UpdateGroup", "GRP-005")
	ws := seedWorkspace(t, db, "Workspace-C")
	role1 := seedRoleMaster(t, db, "viewer2")
	role2 := seedRoleMaster(t, db, "admin")

	require.NoError(t, repo.CreateGroupWorkspaceRole(org.ID, ws.ID, role1.ID))

	err := repo.UpdateGroupWorkspaceRole(org.ID, ws.ID, role2.ID)
	assert.NoError(t, err)

	var record model.GroupWorkspaceRole
	require.NoError(t, db.Where("group_id = ? AND workspace_id = ?", org.ID, ws.ID).First(&record).Error)
	assert.Equal(t, role2.ID, record.RoleID)
}

// TC-M2-UG-027-02: 존재하지 않는 매핑 업데이트 → ErrGroupWorkspaceRoleNotFound
func TestGroupRoleRepository_UpdateGroupWorkspaceRole_NotFound(t *testing.T) {
	db := setupGroupRoleTestDB(t)
	repo := NewGroupRoleRepository(db)

	err := repo.UpdateGroupWorkspaceRole(9999, 9999, 1)
	assert.ErrorIs(t, err, ErrGroupWorkspaceRoleNotFound)
}

// --- TC-M2-UG-028: DeleteGroupWorkspaceRole ---

// TC-M2-UG-028-01: 정상 매핑 삭제
func TestGroupRoleRepository_DeleteGroupWorkspaceRole(t *testing.T) {
	db := setupGroupRoleTestDB(t)
	repo := NewGroupRoleRepository(db)

	org := seedOrganization(t, db, "DeleteGroup", "GRP-006")
	ws := seedWorkspace(t, db, "Workspace-D")
	role := seedRoleMaster(t, db, "ops")

	require.NoError(t, repo.CreateGroupWorkspaceRole(org.ID, ws.ID, role.ID))

	err := repo.DeleteGroupWorkspaceRole(org.ID, ws.ID)
	assert.NoError(t, err)

	var count int64
	db.Model(&model.GroupWorkspaceRole{}).Where("group_id = ? AND workspace_id = ?", org.ID, ws.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

// TC-M2-UG-028-02: 존재하지 않는 매핑 삭제 → ErrGroupWorkspaceRoleNotFound
func TestGroupRoleRepository_DeleteGroupWorkspaceRole_NotFound(t *testing.T) {
	db := setupGroupRoleTestDB(t)
	repo := NewGroupRoleRepository(db)

	err := repo.DeleteGroupWorkspaceRole(9999, 9999)
	assert.ErrorIs(t, err, ErrGroupWorkspaceRoleNotFound)
}

// --- FindAvailableWorkspacesForGroup ---

func TestGroupRoleRepository_FindAvailableWorkspacesForGroup(t *testing.T) {
	db := setupGroupRoleTestDB(t)
	repo := NewGroupRoleRepository(db)

	org := seedOrganization(t, db, "AvailGroup", "GRP-007")
	ws1 := seedWorkspace(t, db, "Workspace-E")
	ws2 := seedWorkspace(t, db, "Workspace-F")
	ws3 := seedWorkspace(t, db, "Workspace-G")
	role := seedRoleMaster(t, db, "dev")

	// ws1만 매핑
	require.NoError(t, repo.CreateGroupWorkspaceRole(org.ID, ws1.ID, role.ID))

	available, err := repo.FindAvailableWorkspacesForGroup(org.ID)
	assert.NoError(t, err)
	// ws2, ws3은 미매핑 → 2개 반환
	assert.Len(t, available, 2)

	ids := []uint{available[0].ID, available[1].ID}
	assert.Contains(t, ids, ws2.ID)
	assert.Contains(t, ids, ws3.ID)
	assert.NotContains(t, ids, ws1.ID)
}
