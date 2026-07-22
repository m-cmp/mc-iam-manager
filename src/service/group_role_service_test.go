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

func createGRTestUser(t *testing.T, db *gorm.DB, username, kcID string) *model.User {
	t.Helper()
	u := &model.User{Username: username, KcId: kcID}
	require.NoError(t, db.Create(u).Error)
	return u
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

// ── ListGroupsByPlatformRole / ListGroupsByWorkspaceRole (역할→그룹 역방향 조회) ──

// TC-GR-LGPR-01: 특정 platform role이 부여된 그룹만 반환, 다른 역할을 가진 그룹은 제외
func TestGroupRoleListGroupsByPlatformRole_ReturnsAssignedGroupsOnly(t *testing.T) {
	svc, db := newTestGroupRoleService(t)
	orgA := createGRTestOrg(t, db, "test-group-lgpr01-a", "LGPR01A")
	orgB := createGRTestOrg(t, db, "test-group-lgpr01-b", "LGPR01B")
	roleX := createGRTestRole(t, db, "role-lgpr01-x")
	roleY := createGRTestRole(t, db, "role-lgpr01-y")

	svc.kcService = &mockKeycloakService{}
	require.NoError(t, svc.AssignGroupPlatformRole(context.Background(), orgA.ID, roleX.ID))
	require.NoError(t, svc.AssignGroupPlatformRole(context.Background(), orgB.ID, roleY.ID))

	results, err := svc.ListGroupsByPlatformRole(roleX.ID)

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, orgA.ID, results[0].GroupID)
}

// TC-GR-LGPR-02: 아무 그룹에도 할당되지 않은 역할 → 빈 슬라이스
func TestGroupRoleListGroupsByPlatformRole_Empty(t *testing.T) {
	svc, _ := newTestGroupRoleService(t)

	results, err := svc.ListGroupsByPlatformRole(99999)

	require.NoError(t, err)
	assert.Len(t, results, 0)
}

// TC-GR-LGWR-01: 특정 workspace role이 부여된 그룹만 반환
func TestGroupRoleListGroupsByWorkspaceRole_ReturnsAssignedGroupsOnly(t *testing.T) {
	svc, db := newTestGroupRoleService(t)
	orgA := createGRTestOrg(t, db, "test-group-lgwr01-a", "LGWR01A")
	orgB := createGRTestOrg(t, db, "test-group-lgwr01-b", "LGWR01B")
	ws := createGRTestWorkspace(t, db, "ws-lgwr01")
	roleX := createGRTestRole(t, db, "role-lgwr01-x")
	roleY := createGRTestRole(t, db, "role-lgwr01-y")

	require.NoError(t, svc.AssignGroupWorkspace(orgA.ID, ws.ID, roleX.ID))
	require.NoError(t, svc.AssignGroupWorkspace(orgB.ID, ws.ID, roleY.ID))

	results, err := svc.ListGroupsByWorkspaceRole(roleX.ID)

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, orgA.ID, results[0].GroupID)
}

// TC-GR-LGWR-02: 아무 그룹에도 할당되지 않은 역할 → 빈 슬라이스
func TestGroupRoleListGroupsByWorkspaceRole_Empty(t *testing.T) {
	svc, _ := newTestGroupRoleService(t)

	results, err := svc.ListGroupsByWorkspaceRole(99999)

	require.NoError(t, err)
	assert.Len(t, results, 0)
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

// ── TC-M2-UG-025: AssignGroupWorkspace (develop 브랜치 테스트) ────────────────

// setupGroupRoleServiceTestDB develop 브랜치 스타일 DB 셋업
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

// ── RemoveUsersFromGroup (bulk) ──────────────────────────────────────────────

// TC-GR-RUFG-01: 그룹이 존재하지 않는 경우 → ErrOrganizationNotFound
func TestGroupRoleRemoveUsersFromGroup_OrgNotFound(t *testing.T) {
	svc, _ := newTestGroupRoleService(t)

	err := svc.RemoveUsersFromGroup(context.Background(), 99999, []uint{1})

	require.Error(t, err)
	assert.ErrorIs(t, err, repository.ErrOrganizationNotFound)
}

// TC-GR-RUFG-02: 존재하지 않는 사용자 ID가 포함된 경우 → 오류 반환
func TestGroupRoleRemoveUsersFromGroup_UserNotFound(t *testing.T) {
	svc, db := newTestGroupRoleService(t)
	org := createGRTestOrg(t, db, "test-group-rufg02", "RUFG02")

	err := svc.RemoveUsersFromGroup(context.Background(), org.ID, []uint{99999})

	require.Error(t, err)
}

// TC-GR-RUFG-03: 여러 사용자를 그룹에서 일괄 제거 → 모두 제거됨
func TestGroupRoleRemoveUsersFromGroup_Success(t *testing.T) {
	svc, db := newTestGroupRoleService(t)
	svc.kcService = &mockKeycloakService{}
	org := createGRTestOrg(t, db, "test-group-rufg03", "RUFG03")
	userA := createGRTestUser(t, db, "user-rufg03-a", "kc-rufg03-a")
	userB := createGRTestUser(t, db, "user-rufg03-b", "kc-rufg03-b")

	require.NoError(t, db.Create(&model.UserOrganization{UserID: userA.ID, OrganizationID: org.ID}).Error)
	require.NoError(t, db.Create(&model.UserOrganization{UserID: userB.ID, OrganizationID: org.ID}).Error)

	err := svc.RemoveUsersFromGroup(context.Background(), org.ID, []uint{userA.ID, userB.ID})
	require.NoError(t, err)

	var count int64
	db.Model(&model.UserOrganization{}).Where("organization_id = ?", org.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}
