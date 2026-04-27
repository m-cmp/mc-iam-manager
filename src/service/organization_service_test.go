package service

// organization_service_test.go
// 조직 계층 구조(Organization Hierarchy) OrganizationService 메서드 단위 테스트
//
// 테스트 범위:
//   - buildOrganizationTree:          평면 목록 → 트리 변환 (순수 Go 로직)
//   - CreateOrganization:             부모 미존재, 이름 중복, 코드 중복, 정상 생성
//   - GetOrganizationByID:            존재하지 않는 ID → not found
//   - GetOrganizationByCode:          존재하지 않는 코드 → not found
//   - DeleteOrganization:             하위 조직/사용자 존재 시 차단, 정상 삭제
//   - CheckOrganizationDeletable:     하위 조직 존재, 사용자 존재, deletable=true 케이스
//   - DeleteOrganizationCascade:      존재하지 않는 ID → not found
//   - AssignUserToOrganizations:      조직 미존재 → 오류, 정상 할당
//   - RemoveUserFromOrganization:     매핑 없음 → ErrUserOrganizationNotFound
//   - GetUserOrganizations:           정상 조회 (빈 목록)
//   - ReplaceUserGroups:              그룹 미존재 → 오류
//   - GetOrganizationSubtree:         존재하지 않는 조직 → not found
//   - MoveOrganization:               존재하지 않는 조직, 자기 자신으로 이동 → 순환 참조
//   - UpdateOrganization:             존재하지 않는 조직 → not found
//
// NOTE: FindTreeFlat, FindSubtreeFlat, GetSubtreeDepth, GetAncestorDepth,
//       UpdateDescendantCodes 는 PostgreSQL CTE / SUBSTRING FROM 구문을 사용하므로
//       SQLite in-memory DB 에서는 실행할 수 없습니다.
//       해당 경로를 거치는 MoveOrganization 의 정상 경로와
//       GetOrganizationTree / GetOrganizationSubtree 의 정상 경로는
//       통합 테스트(PostgreSQL)로 별도 검증합니다.

import (
	"errors"
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

func setupOrgServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&model.Organization{},
		&model.User{},
		&model.UserOrganization{},
	))
	return db
}

func newTestOrgService(t *testing.T) (*OrganizationService, *gorm.DB) {
	t.Helper()
	db := setupOrgServiceTestDB(t)
	svc := NewOrganizationService(db)
	return svc, db
}

// createOrg 는 테스트용 조직을 DB에 직접 삽입하고 ID를 반환합니다.
func createOrg(t *testing.T, db *gorm.DB, code, name string, parentID *uint) *model.Organization {
	t.Helper()
	org := &model.Organization{
		OrganizationCode: code,
		Name:             name,
		ParentID:         parentID,
	}
	require.NoError(t, db.Create(org).Error)
	return org
}

// createOrgUser 는 테스트용 사용자를 DB에 직접 삽입합니다.
func createOrgUser(t *testing.T, db *gorm.DB, username, kcID string) *model.User {
	t.Helper()
	u := &model.User{Username: username, KcId: kcID}
	require.NoError(t, db.Create(u).Error)
	return u
}

// ── buildOrganizationTree 테스트 ──────────────────────────────────────────────

// TC-BT-01: 빈 입력 → 빈 슬라이스 반환
func TestBuildOrganizationTree_Empty(t *testing.T) {
	result := buildOrganizationTree(nil)
	assert.Empty(t, result)
}

// TC-BT-02: 루트 노드 1개 → children 없는 트리
func TestBuildOrganizationTree_SingleRoot(t *testing.T) {
	flat := []model.OrganizationTree{
		{ID: 1, ParentID: nil, OrganizationCode: "01", Name: "Root", Level: 1},
	}
	roots := buildOrganizationTree(flat)

	require.Len(t, roots, 1)
	assert.Equal(t, uint(1), roots[0].ID)
	assert.Empty(t, roots[0].Children)
}

// TC-BT-03: 루트 → 자식 → 손자 계층 구조
func TestBuildOrganizationTree_ThreeLevels(t *testing.T) {
	rootID := uint(1)
	childID := uint(2)
	flat := []model.OrganizationTree{
		{ID: rootID, ParentID: nil, OrganizationCode: "01", Name: "Root", Level: 1},
		{ID: childID, ParentID: &rootID, OrganizationCode: "0101", Name: "Child", Level: 2},
		{ID: 3, ParentID: &childID, OrganizationCode: "010101", Name: "GrandChild", Level: 3},
	}
	roots := buildOrganizationTree(flat)

	require.Len(t, roots, 1)
	require.Len(t, roots[0].Children, 1)
	assert.Equal(t, "Child", roots[0].Children[0].Name)
	require.Len(t, roots[0].Children[0].Children, 1)
	assert.Equal(t, "GrandChild", roots[0].Children[0].Children[0].Name)
}

// TC-BT-04: 루트 2개 + 각각 자식 1개
func TestBuildOrganizationTree_MultipleRoots(t *testing.T) {
	root1ID := uint(1)
	root2ID := uint(2)
	flat := []model.OrganizationTree{
		{ID: root1ID, ParentID: nil, OrganizationCode: "01", Name: "Root1", Level: 1},
		{ID: root2ID, ParentID: nil, OrganizationCode: "02", Name: "Root2", Level: 1},
		{ID: 3, ParentID: &root1ID, OrganizationCode: "0101", Name: "Child1", Level: 2},
		{ID: 4, ParentID: &root2ID, OrganizationCode: "0201", Name: "Child2", Level: 2},
	}
	roots := buildOrganizationTree(flat)

	require.Len(t, roots, 2)
	assert.Len(t, roots[0].Children, 1)
	assert.Len(t, roots[1].Children, 1)
}

// ── CreateOrganization 테스트 ─────────────────────────────────────────────────

// TC-CO-01: 부모 조직이 존재하지 않으면 오류
func TestCreateOrganization_ParentNotFound(t *testing.T) {
	svc, _ := newTestOrgService(t)
	parentID := uint(99999)

	_, err := svc.CreateOrganization(&model.CreateOrganizationRequest{
		Name:     "Child",
		ParentID: &parentID,
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, repository.ErrOrganizationNotFound))
}

// TC-CO-02: 동일 부모 하 이름 중복 → ErrOrganizationNameDuplicate
func TestCreateOrganization_DuplicateName(t *testing.T) {
	svc, db := newTestOrgService(t)
	createOrg(t, db, "01", "HR", nil)

	_, err := svc.CreateOrganization(&model.CreateOrganizationRequest{
		Name:             "HR",
		OrganizationCode: "02",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, repository.ErrOrganizationNameDuplicate)
}

// TC-CO-03: 코드 직접 입력 시 중복 → ErrOrganizationCodeDuplicate
func TestCreateOrganization_DuplicateCode(t *testing.T) {
	svc, db := newTestOrgService(t)
	createOrg(t, db, "01", "Existing", nil)

	_, err := svc.CreateOrganization(&model.CreateOrganizationRequest{
		Name:             "New",
		OrganizationCode: "01",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, repository.ErrOrganizationCodeDuplicate)
}

// TC-CO-04: 코드 직접 입력으로 최상위 조직 생성 성공
func TestCreateOrganization_Success_ManualCode(t *testing.T) {
	svc, db := newTestOrgService(t)

	org, err := svc.CreateOrganization(&model.CreateOrganizationRequest{
		Name:             "HR",
		OrganizationCode: "01",
		Description:      "Human Resources",
	})

	require.NoError(t, err)
	require.NotNil(t, org)
	assert.Equal(t, "01", org.OrganizationCode)
	assert.Equal(t, "HR", org.Name)

	// DB에서 확인
	var saved model.Organization
	require.NoError(t, db.First(&saved, org.ID).Error)
	assert.Equal(t, "HR", saved.Name)
}

// TC-CO-05: 코드 자동 생성으로 최상위 조직 생성 (DB에 아무것도 없으면 "01")
func TestCreateOrganization_Success_AutoCode(t *testing.T) {
	svc, _ := newTestOrgService(t)

	org, err := svc.CreateOrganization(&model.CreateOrganizationRequest{
		Name: "Engineering",
	})

	require.NoError(t, err)
	assert.Equal(t, "01", org.OrganizationCode)
}

// TC-CO-06: 하위 조직 코드 자동 생성 (부모 코드 "01" → 자식 "0101")
func TestCreateOrganization_Success_ChildAutoCode(t *testing.T) {
	svc, db := newTestOrgService(t)
	parent := createOrg(t, db, "01", "Engineering", nil)

	child, err := svc.CreateOrganization(&model.CreateOrganizationRequest{
		Name:     "Backend",
		ParentID: &parent.ID,
	})

	require.NoError(t, err)
	assert.Equal(t, "0101", child.OrganizationCode)
	assert.Equal(t, &parent.ID, child.ParentID)
}

// ── GetOrganizationByID 테스트 ────────────────────────────────────────────────

// TC-GI-01: 존재하지 않는 ID → ErrOrganizationNotFound
func TestGetOrganizationByID_NotFound(t *testing.T) {
	svc, _ := newTestOrgService(t)

	_, err := svc.GetOrganizationByID(99999)

	require.Error(t, err)
	assert.ErrorIs(t, err, repository.ErrOrganizationNotFound)
}

// TC-GI-02: 존재하는 조직 조회 성공
func TestGetOrganizationByID_Success(t *testing.T) {
	svc, db := newTestOrgService(t)
	org := createOrg(t, db, "01", "Marketing", nil)

	got, err := svc.GetOrganizationByID(org.ID)

	require.NoError(t, err)
	assert.Equal(t, "Marketing", got.Name)
}

// ── GetOrganizationByCode 테스트 ──────────────────────────────────────────────

// TC-GC-01: 존재하지 않는 코드 → ErrOrganizationNotFound
func TestGetOrganizationByCode_NotFound(t *testing.T) {
	svc, _ := newTestOrgService(t)

	_, err := svc.GetOrganizationByCode("NOPE")

	require.Error(t, err)
	assert.ErrorIs(t, err, repository.ErrOrganizationNotFound)
}

// TC-GC-02: 존재하는 코드 조회 성공
func TestGetOrganizationByCode_Success(t *testing.T) {
	svc, db := newTestOrgService(t)
	createOrg(t, db, "01", "Sales", nil)

	got, err := svc.GetOrganizationByCode("01")

	require.NoError(t, err)
	assert.Equal(t, "Sales", got.Name)
}

// ── DeleteOrganization 테스트 ─────────────────────────────────────────────────

// TC-DO-01: 존재하지 않는 조직 삭제 → ErrOrganizationNotFound
func TestDeleteOrganization_NotFound(t *testing.T) {
	svc, _ := newTestOrgService(t)

	err := svc.DeleteOrganization(99999)

	require.Error(t, err)
	assert.ErrorIs(t, err, repository.ErrOrganizationNotFound)
}

// TC-DO-02: 하위 조직 존재 시 삭제 차단 → ErrOrganizationHasChildren
func TestDeleteOrganization_HasChildren(t *testing.T) {
	svc, db := newTestOrgService(t)
	parent := createOrg(t, db, "01", "Parent", nil)
	createOrg(t, db, "0101", "Child", &parent.ID)

	err := svc.DeleteOrganization(parent.ID)

	require.Error(t, err)
	assert.ErrorIs(t, err, repository.ErrOrganizationHasChildren)
}

// TC-DO-03: 소속 사용자 존재 시 삭제 차단 → ErrOrganizationHasUsers
func TestDeleteOrganization_HasUsers(t *testing.T) {
	svc, db := newTestOrgService(t)
	org := createOrg(t, db, "01", "HR", nil)
	user := createOrgUser(t, db, "alice", "kc-alice-01")

	// 사용자-조직 매핑 직접 삽입
	require.NoError(t, db.Create(&model.UserOrganization{
		UserID:         user.ID,
		OrganizationID: org.ID,
	}).Error)

	err := svc.DeleteOrganization(org.ID)

	require.Error(t, err)
	assert.ErrorIs(t, err, repository.ErrOrganizationHasUsers)
}

// TC-DO-04: 하위 조직/사용자 없는 조직 정상 삭제
func TestDeleteOrganization_Success(t *testing.T) {
	svc, db := newTestOrgService(t)
	org := createOrg(t, db, "01", "Temp", nil)

	err := svc.DeleteOrganization(org.ID)

	require.NoError(t, err)

	// DB에서 조회 시 not found 확인
	var count int64
	db.Model(&model.Organization{}).Where("id = ?", org.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

// ── CheckOrganizationDeletable 테스트 ─────────────────────────────────────────

// TC-CD-01: 존재하지 않는 조직 → ErrOrganizationNotFound
func TestCheckOrganizationDeletable_NotFound(t *testing.T) {
	svc, _ := newTestOrgService(t)

	_, err := svc.CheckOrganizationDeletable(99999)

	require.Error(t, err)
	assert.ErrorIs(t, err, repository.ErrOrganizationNotFound)
}

// TC-CD-02: 하위 조직 존재 → Deletable=false
func TestCheckOrganizationDeletable_HasChildren(t *testing.T) {
	svc, db := newTestOrgService(t)
	parent := createOrg(t, db, "01", "Parent", nil)
	createOrg(t, db, "0101", "Child", &parent.ID)

	resp, err := svc.CheckOrganizationDeletable(parent.ID)

	require.NoError(t, err)
	assert.False(t, resp.Deletable)
	assert.NotEmpty(t, resp.Reason)
}

// TC-CD-03: 소속 사용자 존재 → Deletable=false
func TestCheckOrganizationDeletable_HasUsers(t *testing.T) {
	svc, db := newTestOrgService(t)
	org := createOrg(t, db, "01", "HR", nil)
	user := createOrgUser(t, db, "bob", "kc-bob-01")
	require.NoError(t, db.Create(&model.UserOrganization{
		UserID:         user.ID,
		OrganizationID: org.ID,
	}).Error)

	resp, err := svc.CheckOrganizationDeletable(org.ID)

	require.NoError(t, err)
	assert.False(t, resp.Deletable)
	assert.NotEmpty(t, resp.Reason)
}

// TC-CD-04: 하위 조직/사용자 없음 → Deletable=true
func TestCheckOrganizationDeletable_Deletable(t *testing.T) {
	svc, db := newTestOrgService(t)
	org := createOrg(t, db, "01", "Leaf", nil)

	resp, err := svc.CheckOrganizationDeletable(org.ID)

	require.NoError(t, err)
	assert.True(t, resp.Deletable)
	assert.Empty(t, resp.Reason)
}

// ── DeleteOrganizationCascade 테스트 ─────────────────────────────────────────

// TC-DC-01: 존재하지 않는 조직 cascade 삭제 → ErrOrganizationNotFound
func TestDeleteOrganizationCascade_NotFound(t *testing.T) {
	svc, _ := newTestOrgService(t)

	err := svc.DeleteOrganizationCascade(99999)

	require.Error(t, err)
	assert.ErrorIs(t, err, repository.ErrOrganizationNotFound)
}

// ── AssignUserToOrganizations 테스트 ─────────────────────────────────────────

// TC-AU-01: 존재하지 않는 조직 → 오류
func TestAssignUserToOrganizations_OrgNotFound(t *testing.T) {
	svc, _ := newTestOrgService(t)

	err := svc.AssignUserToOrganizations(1, []uint{99999})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "organization not found")
}

// TC-AU-02: 정상 할당 후 매핑 확인
func TestAssignUserToOrganizations_Success(t *testing.T) {
	svc, db := newTestOrgService(t)
	org := createOrg(t, db, "01", "Dev", nil)
	user := createOrgUser(t, db, "charlie", "kc-charlie-01")

	err := svc.AssignUserToOrganizations(user.ID, []uint{org.ID})

	require.NoError(t, err)

	var count int64
	db.Model(&model.UserOrganization{}).
		Where("user_id = ? AND organization_id = ?", user.ID, org.ID).
		Count(&count)
	assert.Equal(t, int64(1), count)
}

// TC-AU-03: 이미 할당된 경우 재할당 → 중복 없이 멱등 처리
func TestAssignUserToOrganizations_Idempotent(t *testing.T) {
	svc, db := newTestOrgService(t)
	org := createOrg(t, db, "01", "Dev", nil)
	user := createOrgUser(t, db, "diana", "kc-diana-01")

	require.NoError(t, svc.AssignUserToOrganizations(user.ID, []uint{org.ID}))
	require.NoError(t, svc.AssignUserToOrganizations(user.ID, []uint{org.ID}))

	var count int64
	db.Model(&model.UserOrganization{}).
		Where("user_id = ? AND organization_id = ?", user.ID, org.ID).
		Count(&count)
	assert.Equal(t, int64(1), count)
}

// ── RemoveUserFromOrganization 테스트 ─────────────────────────────────────────

// TC-RU-01: 매핑이 없으면 ErrUserOrganizationNotFound
func TestRemoveUserFromOrganization_NotFound(t *testing.T) {
	svc, _ := newTestOrgService(t)

	err := svc.RemoveUserFromOrganization(1, 1)

	require.Error(t, err)
	assert.ErrorIs(t, err, repository.ErrUserOrganizationNotFound)
}

// TC-RU-02: 정상 제거
func TestRemoveUserFromOrganization_Success(t *testing.T) {
	svc, db := newTestOrgService(t)
	org := createOrg(t, db, "01", "Finance", nil)
	user := createOrgUser(t, db, "eve", "kc-eve-01")
	require.NoError(t, db.Create(&model.UserOrganization{
		UserID: user.ID, OrganizationID: org.ID,
	}).Error)

	err := svc.RemoveUserFromOrganization(user.ID, org.ID)

	require.NoError(t, err)

	var count int64
	db.Model(&model.UserOrganization{}).
		Where("user_id = ? AND organization_id = ?", user.ID, org.ID).
		Count(&count)
	assert.Equal(t, int64(0), count)
}

// ── GetUserOrganizations 테스트 ───────────────────────────────────────────────

// TC-GU-01: 소속 조직 없는 사용자 → 빈 슬라이스
func TestGetUserOrganizations_Empty(t *testing.T) {
	svc, _ := newTestOrgService(t)

	orgs, err := svc.GetUserOrganizations(99999)

	require.NoError(t, err)
	assert.Empty(t, orgs)
}

// TC-GU-02: 소속 조직 목록 정상 반환
func TestGetUserOrganizations_WithOrgs(t *testing.T) {
	svc, db := newTestOrgService(t)
	org1 := createOrg(t, db, "01", "Alpha", nil)
	org2 := createOrg(t, db, "02", "Beta", nil)
	user := createOrgUser(t, db, "frank", "kc-frank-01")

	require.NoError(t, db.Create(&model.UserOrganization{UserID: user.ID, OrganizationID: org1.ID}).Error)
	require.NoError(t, db.Create(&model.UserOrganization{UserID: user.ID, OrganizationID: org2.ID}).Error)

	orgs, err := svc.GetUserOrganizations(user.ID)

	require.NoError(t, err)
	assert.Len(t, orgs, 2)
}

// ── ReplaceUserGroups 테스트 ──────────────────────────────────────────────────

// TC-RG-01: 신규 그룹이 존재하지 않으면 오류
func TestReplaceUserGroups_GroupNotFound(t *testing.T) {
	svc, _ := newTestOrgService(t)

	err := svc.ReplaceUserGroups(1, []uint{99999})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "group not found")
}

// TC-RG-02: 기존 그룹 전부 제거 후 신규 할당 (빈 배열)
func TestReplaceUserGroups_RemoveAll(t *testing.T) {
	svc, db := newTestOrgService(t)
	org := createOrg(t, db, "01", "Old", nil)
	user := createOrgUser(t, db, "grace", "kc-grace-01")
	require.NoError(t, db.Create(&model.UserOrganization{UserID: user.ID, OrganizationID: org.ID}).Error)

	err := svc.ReplaceUserGroups(user.ID, []uint{})

	require.NoError(t, err)

	var count int64
	db.Model(&model.UserOrganization{}).Where("user_id = ?", user.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

// TC-RG-03: 기존 그룹 교체 성공
func TestReplaceUserGroups_Replace(t *testing.T) {
	svc, db := newTestOrgService(t)
	org1 := createOrg(t, db, "01", "OldGroup", nil)
	org2 := createOrg(t, db, "02", "NewGroup", nil)
	user := createOrgUser(t, db, "henry", "kc-henry-01")
	require.NoError(t, db.Create(&model.UserOrganization{UserID: user.ID, OrganizationID: org1.ID}).Error)

	err := svc.ReplaceUserGroups(user.ID, []uint{org2.ID})

	require.NoError(t, err)

	orgs, err := svc.GetUserOrganizations(user.ID)
	require.NoError(t, err)
	require.Len(t, orgs, 1)
	assert.Equal(t, "NewGroup", orgs[0].Name)
}

// ── GetOrganizationSubtree 테스트 ─────────────────────────────────────────────

// TC-GS-01: 존재하지 않는 조직 → ErrOrganizationNotFound
func TestGetOrganizationSubtree_OrgNotFound(t *testing.T) {
	svc, _ := newTestOrgService(t)

	_, err := svc.GetOrganizationSubtree(99999)

	require.Error(t, err)
	assert.ErrorIs(t, err, repository.ErrOrganizationNotFound)
}

// ── MoveOrganization 테스트 ───────────────────────────────────────────────────

// TC-MO-01: 존재하지 않는 조직 이동 → ErrOrganizationNotFound
func TestMoveOrganization_OrgNotFound(t *testing.T) {
	svc, _ := newTestOrgService(t)

	err := svc.MoveOrganization(99999, &model.MoveOrganizationRequest{})

	require.Error(t, err)
	assert.ErrorIs(t, err, repository.ErrOrganizationNotFound)
}

// TC-MO-02: 자기 자신을 새 부모로 이동 → ErrCircularReference
func TestMoveOrganization_SelfAsParent(t *testing.T) {
	svc, db := newTestOrgService(t)
	org := createOrg(t, db, "01", "Self", nil)

	err := svc.MoveOrganization(org.ID, &model.MoveOrganizationRequest{NewParentID: &org.ID})

	require.Error(t, err)
	assert.ErrorIs(t, err, repository.ErrCircularReference)
}

// TC-MO-03: 자식 조직을 손자의 하위로 이동 → ErrCircularReference
func TestMoveOrganization_CircularReference(t *testing.T) {
	svc, db := newTestOrgService(t)
	parent := createOrg(t, db, "01", "Parent", nil)
	child := createOrg(t, db, "0101", "Child", &parent.ID)

	// 부모(01)를 자식(0101)의 하위로 이동 시도
	err := svc.MoveOrganization(parent.ID, &model.MoveOrganizationRequest{NewParentID: &child.ID})

	require.Error(t, err)
	assert.ErrorIs(t, err, repository.ErrCircularReference)
}

// ── UpdateOrganization 테스트 ─────────────────────────────────────────────────

// TC-UO-01: 존재하지 않는 조직 수정 → ErrOrganizationNotFound
func TestUpdateOrganization_NotFound(t *testing.T) {
	svc, _ := newTestOrgService(t)

	err := svc.UpdateOrganization(99999, &model.UpdateOrganizationRequest{Name: "New"})

	require.Error(t, err)
	assert.ErrorIs(t, err, repository.ErrOrganizationNotFound)
}

// TC-UO-02: 변경 사항 없음 → 오류 없이 종료
func TestUpdateOrganization_NoChanges(t *testing.T) {
	svc, db := newTestOrgService(t)
	org := createOrg(t, db, "01", "Stable", nil)

	// 동일 이름/코드로 업데이트 (변경 없음)
	err := svc.UpdateOrganization(org.ID, &model.UpdateOrganizationRequest{Name: "Stable"})

	require.NoError(t, err)
}

// TC-UO-03: 이름 중복 → ErrOrganizationNameDuplicate
func TestUpdateOrganization_DuplicateName(t *testing.T) {
	svc, db := newTestOrgService(t)
	createOrg(t, db, "01", "Existing", nil)
	target := createOrg(t, db, "02", "Target", nil)

	err := svc.UpdateOrganization(target.ID, &model.UpdateOrganizationRequest{Name: "Existing"})

	require.Error(t, err)
	assert.ErrorIs(t, err, repository.ErrOrganizationNameDuplicate)
}

// TC-UO-04: 코드 중복 → ErrOrganizationCodeDuplicate
func TestUpdateOrganization_DuplicateCode(t *testing.T) {
	svc, db := newTestOrgService(t)
	createOrg(t, db, "01", "Alpha", nil)
	target := createOrg(t, db, "02", "Beta", nil)

	err := svc.UpdateOrganization(target.ID, &model.UpdateOrganizationRequest{
		OrganizationCode: "01",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, repository.ErrOrganizationCodeDuplicate)
}

// TC-UO-05: 이름 정상 수정
func TestUpdateOrganization_NameSuccess(t *testing.T) {
	svc, db := newTestOrgService(t)
	org := createOrg(t, db, "01", "OldName", nil)

	err := svc.UpdateOrganization(org.ID, &model.UpdateOrganizationRequest{Name: "NewName"})

	require.NoError(t, err)

	var updated model.Organization
	require.NoError(t, db.First(&updated, org.ID).Error)
	assert.Equal(t, "NewName", updated.Name)
}

// ── GetOrganizationUsers 테스트 ───────────────────────────────────────────────

// TC-GOU-01: 소속 사용자 없는 조직 → 빈 슬라이스
func TestGetOrganizationUsers_Empty(t *testing.T) {
	svc, db := newTestOrgService(t)
	org := createOrg(t, db, "01", "Empty", nil)

	users, err := svc.GetOrganizationUsers(org.ID)

	require.NoError(t, err)
	assert.Empty(t, users)
}

// TC-GOU-02: 소속 사용자 목록 정상 반환
func TestGetOrganizationUsers_WithUsers(t *testing.T) {
	svc, db := newTestOrgService(t)
	org := createOrg(t, db, "01", "IT", nil)
	u1 := createOrgUser(t, db, "ivan", "kc-ivan-01")
	u2 := createOrgUser(t, db, "judy", "kc-judy-01")
	require.NoError(t, db.Create(&model.UserOrganization{UserID: u1.ID, OrganizationID: org.ID}).Error)
	require.NoError(t, db.Create(&model.UserOrganization{UserID: u2.ID, OrganizationID: org.ID}).Error)

	users, err := svc.GetOrganizationUsers(org.ID)

	require.NoError(t, err)
	assert.Len(t, users, 2)
}
