package service

import (
	"testing"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ── DB 셋업 헬퍼 ──────────────────────────────────────────────────────────────

func setupRoleServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&model.User{},
		&model.RoleMaster{},
		&model.Workspace{},
		&model.Organization{},
		&model.UserOrganization{},
		&model.UserPlatformRole{},
		&model.UserWorkspaceRole{},
		&model.GroupPlatformRole{},
		&model.GroupWorkspaceRole{},
	)
	require.NoError(t, err)
	return db
}

// ── 시드 헬퍼 ─────────────────────────────────────────────────────────────────

func seedRoleUser(t *testing.T, db *gorm.DB, kcID, username string) *model.User {
	t.Helper()
	u := &model.User{KcId: kcID, Username: username}
	require.NoError(t, db.Create(u).Error)
	return u
}

func seedRoleMaster(t *testing.T, db *gorm.DB, name string) *model.RoleMaster {
	t.Helper()
	r := &model.RoleMaster{Name: name}
	require.NoError(t, db.Create(r).Error)
	return r
}

func seedWorkspace(t *testing.T, db *gorm.DB, name string) *model.Workspace {
	t.Helper()
	ws := &model.Workspace{Name: name}
	require.NoError(t, db.Create(ws).Error)
	return ws
}

func seedOrganization(t *testing.T, db *gorm.DB, name, code string) *model.Organization {
	t.Helper()
	org := &model.Organization{Name: name, OrganizationCode: code}
	require.NoError(t, db.Create(org).Error)
	return org
}

func assignUserPlatformRole(t *testing.T, db *gorm.DB, userID, roleID uint) {
	t.Helper()
	upr := &model.UserPlatformRole{UserID: userID, RoleID: roleID}
	require.NoError(t, db.Create(upr).Error)
}

func assignUserWorkspaceRole(t *testing.T, db *gorm.DB, userID, workspaceID, roleID uint) {
	t.Helper()
	uwr := &model.UserWorkspaceRole{UserID: userID, WorkspaceID: workspaceID, RoleID: roleID}
	require.NoError(t, db.Create(uwr).Error)
}

func assignUserOrg(t *testing.T, db *gorm.DB, userID, orgID uint) {
	t.Helper()
	uo := &model.UserOrganization{UserID: userID, OrganizationID: orgID}
	require.NoError(t, db.Create(uo).Error)
}

func assignGroupPlatformRole(t *testing.T, db *gorm.DB, groupID, roleID uint) {
	t.Helper()
	gpr := &model.GroupPlatformRole{GroupID: groupID, RoleID: roleID}
	require.NoError(t, db.Create(gpr).Error)
}

func assignGroupWorkspaceRole(t *testing.T, db *gorm.DB, groupID, workspaceID, roleID uint) {
	t.Helper()
	gwr := &model.GroupWorkspaceRole{GroupID: groupID, WorkspaceID: workspaceID, RoleID: roleID}
	require.NoError(t, db.Create(gwr).Error)
}

// ── GetEffectivePlatformRoles 단위 테스트 ────────────────────────────────────

// TC-ROLE-EPR-01: 직접 할당 플랫폼 역할만 있는 경우
func TestGetEffectivePlatformRoles_DirectOnly(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)

	user := seedRoleUser(t, db, "kc-user-01", "alice")
	role := seedRoleMaster(t, db, "platform-admin")
	assignUserPlatformRole(t, db, user.ID, role.ID)

	roles, err := svc.GetEffectivePlatformRoles(user.ID)

	require.NoError(t, err)
	assert.Len(t, roles, 1)
	assert.Equal(t, role.ID, roles[0].ID)
	assert.Equal(t, "platform-admin", roles[0].Name)
}

// TC-ROLE-EPR-02: 그룹 상속 플랫폼 역할만 있는 경우
func TestGetEffectivePlatformRoles_GroupInheritedOnly(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)

	user := seedRoleUser(t, db, "kc-user-02", "bob")
	org := seedOrganization(t, db, "DevTeam", "DEV-001")
	role := seedRoleMaster(t, db, "platform-viewer")

	// 사용자 → 그룹 소속, 그룹 → 플랫폼 역할
	assignUserOrg(t, db, user.ID, org.ID)
	assignGroupPlatformRole(t, db, org.ID, role.ID)

	roles, err := svc.GetEffectivePlatformRoles(user.ID)

	require.NoError(t, err)
	assert.Len(t, roles, 1)
	assert.Equal(t, role.ID, roles[0].ID)
	assert.Equal(t, "platform-viewer", roles[0].Name)
}

// TC-ROLE-EPR-03: 직접 할당 + 그룹 상속 모두 있는 경우 — 중복 제거
func TestGetEffectivePlatformRoles_DirectAndGroupDeduplicated(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)

	user := seedRoleUser(t, db, "kc-user-03", "carol")
	org := seedOrganization(t, db, "OpsTeam", "OPS-001")
	roleA := seedRoleMaster(t, db, "platform-ops")
	roleB := seedRoleMaster(t, db, "platform-dev")

	// 직접: roleA
	assignUserPlatformRole(t, db, user.ID, roleA.ID)
	// 그룹 상속: roleA (중복) + roleB
	assignUserOrg(t, db, user.ID, org.ID)
	assignGroupPlatformRole(t, db, org.ID, roleA.ID)
	assignGroupPlatformRole(t, db, org.ID, roleB.ID)

	roles, err := svc.GetEffectivePlatformRoles(user.ID)

	require.NoError(t, err)
	// roleA는 중복 제거되어 1번만 등장
	assert.Len(t, roles, 2)
	ids := make(map[uint]bool)
	for _, r := range roles {
		ids[r.ID] = true
	}
	assert.True(t, ids[roleA.ID], "roleA should be present")
	assert.True(t, ids[roleB.ID], "roleB should be present")
}

// TC-ROLE-EPR-04: 역할 없는 사용자 → 빈 슬라이스
func TestGetEffectivePlatformRoles_NoRoles(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)

	user := seedRoleUser(t, db, "kc-user-04", "dave")

	roles, err := svc.GetEffectivePlatformRoles(user.ID)

	require.NoError(t, err)
	assert.Len(t, roles, 0)
}

// TC-ROLE-EPR-05: 존재하지 않는 사용자 ID → 빈 슬라이스 (오류 없음)
func TestGetEffectivePlatformRoles_NonExistentUser(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)

	roles, err := svc.GetEffectivePlatformRoles(9999)

	require.NoError(t, err)
	assert.Len(t, roles, 0)
}

// TC-ROLE-EPR-06: 다수 그룹 소속 — 각 그룹의 역할 통합
func TestGetEffectivePlatformRoles_MultipleGroups(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)

	user := seedRoleUser(t, db, "kc-user-06", "eve")
	org1 := seedOrganization(t, db, "Group-Alpha", "ALPHA-001")
	org2 := seedOrganization(t, db, "Group-Beta", "BETA-001")
	role1 := seedRoleMaster(t, db, "platform-alpha-role")
	role2 := seedRoleMaster(t, db, "platform-beta-role")

	assignUserOrg(t, db, user.ID, org1.ID)
	assignUserOrg(t, db, user.ID, org2.ID)
	assignGroupPlatformRole(t, db, org1.ID, role1.ID)
	assignGroupPlatformRole(t, db, org2.ID, role2.ID)

	roles, err := svc.GetEffectivePlatformRoles(user.ID)

	require.NoError(t, err)
	assert.Len(t, roles, 2)
}

// ── GetEffectiveWorkspaceRoles 단위 테스트 ───────────────────────────────────

// TC-ROLE-EWR-01: 직접 할당 워크스페이스 역할만 있는 경우
func TestGetEffectiveWorkspaceRoles_DirectOnly(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)

	user := seedRoleUser(t, db, "kc-ws-01", "frank")
	ws := seedWorkspace(t, db, "workspace-alpha")
	role := seedRoleMaster(t, db, "ws-admin")
	assignUserWorkspaceRole(t, db, user.ID, ws.ID, role.ID)

	results, err := svc.GetEffectiveWorkspaceRoles(user.ID)

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, ws.ID, results[0].WorkspaceID)
	assert.Equal(t, ws.Name, results[0].WorkspaceName)
	assert.Equal(t, role.ID, results[0].RoleID)
	assert.Equal(t, role.Name, results[0].RoleName)
}

// TC-ROLE-EWR-02: 그룹 상속 워크스페이스 역할만 있는 경우
func TestGetEffectiveWorkspaceRoles_GroupInheritedOnly(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)

	user := seedRoleUser(t, db, "kc-ws-02", "grace")
	org := seedOrganization(t, db, "WsTeam-A", "WST-001")
	ws := seedWorkspace(t, db, "workspace-beta")
	role := seedRoleMaster(t, db, "ws-viewer")

	assignUserOrg(t, db, user.ID, org.ID)
	assignGroupWorkspaceRole(t, db, org.ID, ws.ID, role.ID)

	results, err := svc.GetEffectiveWorkspaceRoles(user.ID)

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, ws.ID, results[0].WorkspaceID)
	assert.Equal(t, ws.Name, results[0].WorkspaceName)
	assert.Equal(t, role.ID, results[0].RoleID)
	assert.Equal(t, role.Name, results[0].RoleName)
}

// TC-ROLE-EWR-03: 직접 할당 + 그룹 상속 통합 (동일 workspace, 동일 role) → 중복 제거
func TestGetEffectiveWorkspaceRoles_DirectAndGroupDeduplicated(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)

	user := seedRoleUser(t, db, "kc-ws-03", "hank")
	org := seedOrganization(t, db, "WsTeam-B", "WST-002")
	ws := seedWorkspace(t, db, "workspace-gamma")
	role := seedRoleMaster(t, db, "ws-editor")

	// 직접 할당 + 그룹 상속 → 같은 (workspace_id, role_id)
	assignUserWorkspaceRole(t, db, user.ID, ws.ID, role.ID)
	assignUserOrg(t, db, user.ID, org.ID)
	assignGroupWorkspaceRole(t, db, org.ID, ws.ID, role.ID)

	results, err := svc.GetEffectiveWorkspaceRoles(user.ID)

	require.NoError(t, err)
	// DISTINCT로 중복 제거되어 1개만 반환
	assert.Len(t, results, 1)
}

// TC-ROLE-EWR-04: 직접 + 그룹 상속으로 서로 다른 워크스페이스 역할 통합
func TestGetEffectiveWorkspaceRoles_DirectAndGroupDifferentWorkspaces(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)

	user := seedRoleUser(t, db, "kc-ws-04", "irene")
	org := seedOrganization(t, db, "WsTeam-C", "WST-003")
	ws1 := seedWorkspace(t, db, "workspace-delta")
	ws2 := seedWorkspace(t, db, "workspace-epsilon")
	roleA := seedRoleMaster(t, db, "ws-role-a")
	roleB := seedRoleMaster(t, db, "ws-role-b")

	// 직접: ws1 + roleA
	assignUserWorkspaceRole(t, db, user.ID, ws1.ID, roleA.ID)
	// 그룹 상속: ws2 + roleB
	assignUserOrg(t, db, user.ID, org.ID)
	assignGroupWorkspaceRole(t, db, org.ID, ws2.ID, roleB.ID)

	results, err := svc.GetEffectiveWorkspaceRoles(user.ID)

	require.NoError(t, err)
	assert.Len(t, results, 2)

	wsIDs := make(map[uint]bool)
	roleIDs := make(map[uint]bool)
	for _, r := range results {
		wsIDs[r.WorkspaceID] = true
		roleIDs[r.RoleID] = true
	}
	assert.True(t, wsIDs[ws1.ID])
	assert.True(t, wsIDs[ws2.ID])
	assert.True(t, roleIDs[roleA.ID])
	assert.True(t, roleIDs[roleB.ID])
}

// TC-ROLE-EWR-05: 역할 없는 사용자 → 빈 슬라이스
func TestGetEffectiveWorkspaceRoles_NoRoles(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)

	user := seedRoleUser(t, db, "kc-ws-05", "jack")

	results, err := svc.GetEffectiveWorkspaceRoles(user.ID)

	require.NoError(t, err)
	assert.Len(t, results, 0)
}

// TC-ROLE-EWR-06: 존재하지 않는 사용자 ID → 빈 슬라이스 (오류 없음)
func TestGetEffectiveWorkspaceRoles_NonExistentUser(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)

	results, err := svc.GetEffectiveWorkspaceRoles(9999)

	require.NoError(t, err)
	assert.Len(t, results, 0)
}

// TC-ROLE-EWR-07: 다수 그룹에서 같은 워크스페이스에 다른 역할 — 모두 포함
func TestGetEffectiveWorkspaceRoles_MultipleGroupsSameWorkspace(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)

	user := seedRoleUser(t, db, "kc-ws-07", "kate")
	org1 := seedOrganization(t, db, "WsGroup-X", "WSX-001")
	org2 := seedOrganization(t, db, "WsGroup-Y", "WSY-001")
	ws := seedWorkspace(t, db, "workspace-shared")
	roleX := seedRoleMaster(t, db, "ws-x-role")
	roleY := seedRoleMaster(t, db, "ws-y-role")

	assignUserOrg(t, db, user.ID, org1.ID)
	assignUserOrg(t, db, user.ID, org2.ID)
	assignGroupWorkspaceRole(t, db, org1.ID, ws.ID, roleX.ID)
	assignGroupWorkspaceRole(t, db, org2.ID, ws.ID, roleY.ID)

	results, err := svc.GetEffectiveWorkspaceRoles(user.ID)

	require.NoError(t, err)
	// 같은 워크스페이스지만 다른 역할 → 2개
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, ws.ID, r.WorkspaceID)
		assert.Equal(t, ws.Name, r.WorkspaceName)
	}
}
