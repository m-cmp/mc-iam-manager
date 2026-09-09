package repository

// user_repository_test.go
//
// FindEffectiveUserRoleInWorkspace 단위 테스트 (SQLite in-memory DB) — P4-1(6-4) 회귀.
// 발급 근거를 직접 배정만 보던 FindUserRoleInWorkspace에서 직접∪그룹으로 넓히면서,
// "직접 우선, 없으면 그룹" 선택 규칙과 다중 그룹 충돌 시 결정적 선택을 검증한다.

import (
	"testing"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupUserRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&model.Organization{},
		&model.UserOrganization{},
		&model.User{},
		&model.RoleMaster{},
		&model.Workspace{},
		&model.UserWorkspaceRole{},
		&model.GroupWorkspaceRole{},
	))
	return db
}

func seedUserRepoOrg(t *testing.T, db *gorm.DB, name, code string) *model.Organization {
	t.Helper()
	org := &model.Organization{Name: name, OrganizationCode: code}
	require.NoError(t, db.Create(org).Error)
	return org
}

func seedUserRepoUser(t *testing.T, db *gorm.DB, username string) *model.User {
	t.Helper()
	u := &model.User{Username: username}
	require.NoError(t, db.Create(u).Error)
	return u
}

func seedUserRepoWorkspace(t *testing.T, db *gorm.DB, name string) *model.Workspace {
	t.Helper()
	ws := &model.Workspace{Name: name}
	require.NoError(t, db.Create(ws).Error)
	return ws
}

func seedUserRepoRole(t *testing.T, db *gorm.DB, name string) *model.RoleMaster {
	t.Helper()
	role := &model.RoleMaster{Name: name}
	require.NoError(t, db.Create(role).Error)
	return role
}

// TC-FEURW-01: 직접·그룹 배정 둘 다 없으면 (nil, nil)
func TestFindEffectiveUserRoleInWorkspace_NoRole(t *testing.T) {
	db := setupUserRepoTestDB(t)
	repo := NewUserRepository(db)
	user := seedUserRepoUser(t, db, "u-feurw01")
	ws := seedUserRepoWorkspace(t, db, "ws-feurw01")

	role, err := repo.FindEffectiveUserRoleInWorkspace(user.ID, ws.ID)

	require.NoError(t, err)
	assert.Nil(t, role)
}

// TC-FEURW-02: 직접 배정만 있으면 그것을 반환
func TestFindEffectiveUserRoleInWorkspace_DirectOnly(t *testing.T) {
	db := setupUserRepoTestDB(t)
	repo := NewUserRepository(db)
	user := seedUserRepoUser(t, db, "u-feurw02")
	ws := seedUserRepoWorkspace(t, db, "ws-feurw02")
	role := seedUserRepoRole(t, db, "role-feurw02-direct")
	require.NoError(t, db.Create(&model.UserWorkspaceRole{UserID: user.ID, WorkspaceID: ws.ID, RoleID: role.ID}).Error)

	got, err := repo.FindEffectiveUserRoleInWorkspace(user.ID, ws.ID)

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, role.ID, got.RoleID)
}

// TC-FEURW-03: 직접 배정 없이 그룹 배정만 있으면 그룹 상속 역할을 반환 (P1-3/OPEN-016 자동 부여 결정)
func TestFindEffectiveUserRoleInWorkspace_GroupOnly(t *testing.T) {
	db := setupUserRepoTestDB(t)
	repo := NewUserRepository(db)
	user := seedUserRepoUser(t, db, "u-feurw03")
	ws := seedUserRepoWorkspace(t, db, "ws-feurw03")
	org := seedUserRepoOrg(t, db, "group-feurw03", "FEURW03")
	role := seedUserRepoRole(t, db, "role-feurw03-group")
	require.NoError(t, db.Create(&model.UserOrganization{UserID: user.ID, OrganizationID: org.ID}).Error)
	require.NoError(t, db.Create(&model.GroupWorkspaceRole{GroupID: org.ID, WorkspaceID: ws.ID, RoleID: role.ID}).Error)

	got, err := repo.FindEffectiveUserRoleInWorkspace(user.ID, ws.ID)

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, role.ID, got.RoleID)
}

// TC-FEURW-04: 직접·그룹 배정이 둘 다 있고 서로 다르면 직접이 우선 (Union+dedup 직접 우선 원칙)
func TestFindEffectiveUserRoleInWorkspace_DirectWinsOverGroup(t *testing.T) {
	db := setupUserRepoTestDB(t)
	repo := NewUserRepository(db)
	user := seedUserRepoUser(t, db, "u-feurw04")
	ws := seedUserRepoWorkspace(t, db, "ws-feurw04")
	org := seedUserRepoOrg(t, db, "group-feurw04", "FEURW04")
	directRole := seedUserRepoRole(t, db, "role-feurw04-direct")
	groupRole := seedUserRepoRole(t, db, "role-feurw04-group")
	require.NoError(t, db.Create(&model.UserWorkspaceRole{UserID: user.ID, WorkspaceID: ws.ID, RoleID: directRole.ID}).Error)
	require.NoError(t, db.Create(&model.UserOrganization{UserID: user.ID, OrganizationID: org.ID}).Error)
	require.NoError(t, db.Create(&model.GroupWorkspaceRole{GroupID: org.ID, WorkspaceID: ws.ID, RoleID: groupRole.ID}).Error)

	got, err := repo.FindEffectiveUserRoleInWorkspace(user.ID, ws.ID)

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, directRole.ID, got.RoleID, "직접 배정이 그룹 상속보다 우선해야 한다")
}

// TC-FEURW-05: 직접 배정 없이 서로 다른 두 그룹이 같은 워크스페이스에 다른 역할을 부여하면
// role_id 오름차순으로 결정적 선택 (다중 매핑 선택 규칙 확정 전까지의 잠정 규칙, P4-3)
func TestFindEffectiveUserRoleInWorkspace_MultipleGroupsDeterministicPick(t *testing.T) {
	db := setupUserRepoTestDB(t)
	repo := NewUserRepository(db)
	user := seedUserRepoUser(t, db, "u-feurw05")
	ws := seedUserRepoWorkspace(t, db, "ws-feurw05")
	orgA := seedUserRepoOrg(t, db, "group-feurw05-a", "FEURW05A")
	orgB := seedUserRepoOrg(t, db, "group-feurw05-b", "FEURW05B")
	roleLow := seedUserRepoRole(t, db, "role-feurw05-low")
	roleHigh := seedUserRepoRole(t, db, "role-feurw05-high")
	require.Less(t, roleLow.ID, roleHigh.ID)
	require.NoError(t, db.Create(&model.UserOrganization{UserID: user.ID, OrganizationID: orgA.ID}).Error)
	require.NoError(t, db.Create(&model.UserOrganization{UserID: user.ID, OrganizationID: orgB.ID}).Error)
	require.NoError(t, db.Create(&model.GroupWorkspaceRole{GroupID: orgA.ID, WorkspaceID: ws.ID, RoleID: roleHigh.ID}).Error)
	require.NoError(t, db.Create(&model.GroupWorkspaceRole{GroupID: orgB.ID, WorkspaceID: ws.ID, RoleID: roleLow.ID}).Error)

	got, err := repo.FindEffectiveUserRoleInWorkspace(user.ID, ws.ID)

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, roleLow.ID, got.RoleID, "다중 그룹 충돌 시 role_id가 가장 작은 쪽을 결정적으로 선택해야 한다")
}

// TC-FEURW-06: 다른 워크스페이스의 그룹 역할은 섞이지 않는다
func TestFindEffectiveUserRoleInWorkspace_ScopedToWorkspace(t *testing.T) {
	db := setupUserRepoTestDB(t)
	repo := NewUserRepository(db)
	user := seedUserRepoUser(t, db, "u-feurw06")
	wsA := seedUserRepoWorkspace(t, db, "ws-feurw06-a")
	wsB := seedUserRepoWorkspace(t, db, "ws-feurw06-b")
	org := seedUserRepoOrg(t, db, "group-feurw06", "FEURW06")
	role := seedUserRepoRole(t, db, "role-feurw06")
	require.NoError(t, db.Create(&model.UserOrganization{UserID: user.ID, OrganizationID: org.ID}).Error)
	require.NoError(t, db.Create(&model.GroupWorkspaceRole{GroupID: org.ID, WorkspaceID: wsA.ID, RoleID: role.ID}).Error)

	got, err := repo.FindEffectiveUserRoleInWorkspace(user.ID, wsB.ID)

	require.NoError(t, err)
	assert.Nil(t, got, "다른 워크스페이스에 배정된 그룹 역할이 새어 들어오면 안 된다")
}
