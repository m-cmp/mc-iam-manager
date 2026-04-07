package service

// group_role_service_test.go
//
// GroupRoleService 단위 테스트 (SQLite in-memory DB)
// Keycloak 연동이 필요한 메서드는 KC 호출 이전 검증 실패 케이스만 커버한다.
// KC 연동이 없는 순수 DB 메서드는 전체 케이스를 커버한다.

import (
	"context"
	"testing"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ── DB 헬퍼 ───────────────────────────────────────────────────────────────────

func setupGroupRoleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&model.Organization{},
		&model.UserOrganization{},
		&model.User{},
		&model.RoleMaster{},
		&model.RoleSub{},
		&model.Workspace{},
		&model.GroupPlatformRole{},
		&model.GroupWorkspaceRole{},
	))
	return db
}

func newTestGroupRoleService(t *testing.T) (*GroupRoleService, *gorm.DB) {
	t.Helper()
	db := setupGroupRoleTestDB(t)
	svc := &GroupRoleService{
		db:            db,
		groupRoleRepo: repository.NewGroupRoleRepository(db),
		orgRepo:       repository.NewOrganizationRepository(db),
		kcService:     nil, // KC 불필요한 메서드만 테스트
	}
	return svc, db
}

func createGRTestOrg(t *testing.T, db *gorm.DB, name, code string) *model.Organization {
	t.Helper()
	org := &model.Organization{Name: name, OrganizationCode: code}
	require.NoError(t, db.Create(org).Error)
	return org
}

func createGRTestRole(t *testing.T, db *gorm.DB, name string) *model.RoleMaster {
	t.Helper()
	role := &model.RoleMaster{Name: name}
	require.NoError(t, db.Create(role).Error)
	return role
}

func createGRTestWorkspace(t *testing.T, db *gorm.DB, name string) *model.Workspace {
	t.Helper()
	ws := &model.Workspace{Name: name}
	require.NoError(t, db.Create(ws).Error)
	return ws
}

// ── AssignGroupPlatformRole — KC 호출 전 검증 실패 케이스 ─────────────────────

// TC-GR-APR-01: 그룹(조직)이 존재하지 않는 경우 → 오류 반환
func TestGroupRoleAssignPlatformRole_OrgNotFound(t *testing.T) {
	svc, _ := newTestGroupRoleService(t)

	err := svc.AssignGroupPlatformRole(context.Background(), 99999, 1)

	require.Error(t, err)
}

// TC-GR-APR-02: 역할(RoleMaster)이 존재하지 않는 경우 → 오류 반환
func TestGroupRoleAssignPlatformRole_RoleNotFound(t *testing.T) {
	svc, db := newTestGroupRoleService(t)
	org := createGRTestOrg(t, db, "test-group-apr02", "GR02")
	svc.kcService = nil // KC nil이면 KC 이전에서 실패

	err := svc.AssignGroupPlatformRole(context.Background(), org.ID, 99999)

	require.Error(t, err)
	assert.Equal(t, repository.ErrRoleMasterNotFound, err)
}

// ── AssignGroupWorkspace — 순수 DB 메서드 ────────────────────────────────────

// TC-GR-AGW-01: 워크스페이스가 존재하지 않는 경우 → ErrWorkspaceNotFound
func TestGroupRoleAssignGroupWorkspace_WorkspaceNotFound(t *testing.T) {
	svc, db := newTestGroupRoleService(t)
	org := createGRTestOrg(t, db, "test-group-agw01", "GW01")
	role := createGRTestRole(t, db, "ws-role-agw01")

	err := svc.AssignGroupWorkspace(org.ID, 99999, role.ID)

	require.Error(t, err)
	assert.Equal(t, repository.ErrWorkspaceNotFound, err)
}

// TC-GR-AGW-02: 역할이 존재하지 않는 경우 → ErrRoleMasterNotFound
func TestGroupRoleAssignGroupWorkspace_RoleNotFound(t *testing.T) {
	svc, db := newTestGroupRoleService(t)
	org := createGRTestOrg(t, db, "test-group-agw02", "GW02")
	ws := createGRTestWorkspace(t, db, "ws-agw02")

	err := svc.AssignGroupWorkspace(org.ID, ws.ID, 99999)

	require.Error(t, err)
	assert.Equal(t, repository.ErrRoleMasterNotFound, err)
}

// TC-GR-AGW-03: 정상 할당 → 오류 없음
func TestGroupRoleAssignGroupWorkspace_Success(t *testing.T) {
	svc, db := newTestGroupRoleService(t)
	org := createGRTestOrg(t, db, "test-group-agw03", "GW03")
	ws := createGRTestWorkspace(t, db, "ws-agw03")
	role := createGRTestRole(t, db, "ws-role-agw03")

	err := svc.AssignGroupWorkspace(org.ID, ws.ID, role.ID)

	require.NoError(t, err)
}

// ── GetGroupWorkspaces ────────────────────────────────────────────────────────

// TC-GR-GGW-01: 매핑이 없는 경우 → 빈 슬라이스 반환
func TestGroupRoleGetGroupWorkspaces_Empty(t *testing.T) {
	svc, db := newTestGroupRoleService(t)
	org := createGRTestOrg(t, db, "test-group-ggw01", "GGW01")

	result, err := svc.GetGroupWorkspaces(org.ID)

	require.NoError(t, err)
	assert.Len(t, result, 0)
}

// TC-GR-GGW-02: 매핑이 있는 경우 → 결과 반환
func TestGroupRoleGetGroupWorkspaces_WithData(t *testing.T) {
	svc, db := newTestGroupRoleService(t)
	org := createGRTestOrg(t, db, "test-group-ggw02", "GGW02")
	ws := createGRTestWorkspace(t, db, "ws-ggw02")
	role := createGRTestRole(t, db, "ws-role-ggw02")

	require.NoError(t, svc.AssignGroupWorkspace(org.ID, ws.ID, role.ID))

	result, err := svc.GetGroupWorkspaces(org.ID)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, ws.ID, result[0].WorkspaceID)
}

// ── UpdateGroupWorkspaceRole + RemoveGroupWorkspaceRole ──────────────────────

// TC-GR-UGWR-01: 업데이트 후 조회 → 변경된 역할 확인
func TestGroupRoleUpdateGroupWorkspaceRole_Success(t *testing.T) {
	svc, db := newTestGroupRoleService(t)
	org := createGRTestOrg(t, db, "test-group-ugwr01", "UGWR01")
	ws := createGRTestWorkspace(t, db, "ws-ugwr01")
	role1 := createGRTestRole(t, db, "role-ugwr01-a")
	role2 := createGRTestRole(t, db, "role-ugwr01-b")

	require.NoError(t, svc.AssignGroupWorkspace(org.ID, ws.ID, role1.ID))
	require.NoError(t, svc.UpdateGroupWorkspaceRole(org.ID, ws.ID, role2.ID))

	result, err := svc.GetGroupWorkspaces(org.ID)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, role2.ID, result[0].RoleID)
}

// TC-GR-RGWR-01: 매핑 제거 후 목록이 비어 있는지 확인
func TestGroupRoleRemoveGroupWorkspaceRole_Success(t *testing.T) {
	svc, db := newTestGroupRoleService(t)
	org := createGRTestOrg(t, db, "test-group-rgwr01", "RGWR01")
	ws := createGRTestWorkspace(t, db, "ws-rgwr01")
	role := createGRTestRole(t, db, "ws-role-rgwr01")

	require.NoError(t, svc.AssignGroupWorkspace(org.ID, ws.ID, role.ID))
	require.NoError(t, svc.RemoveGroupWorkspaceRole(org.ID, ws.ID))

	result, err := svc.GetGroupWorkspaces(org.ID)
	require.NoError(t, err)
	assert.Len(t, result, 0)
}

// ── GetAvailableWorkspaces ────────────────────────────────────────────────────

// TC-GR-GAW-01: 할당되지 않은 워크스페이스 조회
func TestGroupRoleGetAvailableWorkspaces_ReturnsUnassigned(t *testing.T) {
	svc, db := newTestGroupRoleService(t)
	org := createGRTestOrg(t, db, "test-group-gaw01", "GAW01")
	ws1 := createGRTestWorkspace(t, db, "ws-gaw01-assigned")
	ws2 := createGRTestWorkspace(t, db, "ws-gaw01-free")
	role := createGRTestRole(t, db, "ws-role-gaw01")

	require.NoError(t, svc.AssignGroupWorkspace(org.ID, ws1.ID, role.ID))

	available, err := svc.GetAvailableWorkspaces(org.ID)

	require.NoError(t, err)
	ids := make([]uint, len(available))
	for i, w := range available {
		ids[i] = w.ID
	}
	assert.Contains(t, ids, ws2.ID)
	assert.NotContains(t, ids, ws1.ID)
}

// ── GetUserAccessSummary ──────────────────────────────────────────────────────

// TC-GR-GUAS-01: 역할/그룹 없는 사용자 → 빈 요약 반환
func TestGroupRoleGetUserAccessSummary_NoRoles(t *testing.T) {
	svc, _ := newTestGroupRoleService(t)

	result, err := svc.GetUserAccessSummary(99999)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, uint(99999), result.UserID)
	assert.Empty(t, result.DirectRoles)
	assert.Empty(t, result.Groups)
}
