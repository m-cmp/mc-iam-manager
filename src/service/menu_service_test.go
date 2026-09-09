package service

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupMenuServiceTestDB는 sqlite 인메모리 shared-cache DB에 메뉴/역할 관련 테이블을
// AutoMigrate한다. role_handler_createplatformrole_test.go의 패턴을 재사용하되,
// DSN에 테스트 이름을 넣어 shared-cache DB가 테스트 간에 공유되지 않도록 격리한다.
func setupMenuServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&model.RoleMaster{},
		&model.RoleSub{},
		&model.Menu{},
		&model.RoleMenuMapping{},
	))
	return db
}

// UpsertMenus는 "SET CONSTRAINTS ALL DEFERRED" 등 PostgreSQL 전용 구문을 사용해
// sqlite에서 동작하지 않는다(이 티켓 범위 밖). 체이닝 동작만 검증하기 위해
// menu.yaml에는 upsert 대상에서 제외되는 home 메뉴만 둬서 UpsertMenus 호출 자체를 건너뛴다.
const testMenuYAML = `
menus:
  - id: home
    parentid: ""
    displayname: Home
    restype: menu
    isaction: false
    priority: 0
    menunumber: 1
`

const testPermissionYAML = `
permissions:
  - role: admin
    menus:
      - home
`

// seedAdminPlatformRole은 permission.yaml의 admin 역할이 조회되도록 role_masters/role_subs를 시딩한다.
func seedAdminPlatformRole(t *testing.T, db *gorm.DB) {
	t.Helper()
	role := model.RoleMaster{Name: "admin"}
	require.NoError(t, db.Create(&role).Error)
	require.NoError(t, db.Create(&model.RoleSub{
		RoleID:   role.ID,
		RoleType: constants.RoleTypePlatform,
	}).Error)
}

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

// TC-IAM-TECH-034-01: permission.yaml이 정상이면 메뉴 등록에 이어 권한 시딩까지
// 체이닝되어 성공하고, 경고 문자열은 빈 값이어야 한다.
func TestLoadAndRegisterMenusFromYAML_ChainsPermissionSeed_Success(t *testing.T) {
	db := setupMenuServiceTestDB(t)
	seedAdminPlatformRole(t, db)
	// home은 upsert 대상이 아니라 UpdateMenu만 되므로 매핑 대상이 되려면 선존재해야 한다
	// (DB에 없는 메뉴 id는 매핑을 만들지 않고 missing으로 보고된다)
	require.NoError(t, db.Create(&model.Menu{ID: "home", DisplayName: "Home", ResType: "menu"}).Error)

	permPath := writeTempFile(t, "permission.yaml", testPermissionYAML)
	t.Setenv("MC_WEB_CONSOLE_MENU_PERMISSIONS", permPath)

	menuPath := writeTempFile(t, "menu.yaml", testMenuYAML)

	s := NewMenuService(db)
	result, err := s.LoadAndRegisterMenusFromYAML(menuPath, false)
	require.NoError(t, err)
	require.False(t, result.Skipped)
	require.Empty(t, result.PermissionsWarning, "permission seed succeeded, so no warning should be returned")

	var mapping model.RoleMenuMapping
	require.NoError(t, db.Where("menu_id = ?", "home").First(&mapping).Error)
}

// TC-IAM-TECH-034-02: permission.yaml을 찾지 못해 권한 시딩이 실패해도 메뉴 등록
// 자체는 실패하지 않아야 하며(에러가 nil), 실패 사실은 경고 문자열로 반환되어야 한다.
func TestLoadAndRegisterMenusFromYAML_PermissionSeedMissing_ReturnsWarningNotError(t *testing.T) {
	db := setupMenuServiceTestDB(t)

	missingPermPath := filepath.Join(t.TempDir(), "does-not-exist-permission.yaml")
	t.Setenv("MC_WEB_CONSOLE_MENU_PERMISSIONS", missingPermPath)

	menuPath := writeTempFile(t, "menu.yaml", testMenuYAML)

	s := NewMenuService(db)
	result, err := s.LoadAndRegisterMenusFromYAML(menuPath, false)
	require.NoError(t, err, "permission seed failure must not fail menu registration")
	require.NotEmpty(t, result.PermissionsWarning, "permission seed failure should surface as a warning")

	var mappingCount int64
	require.NoError(t, db.Model(&model.RoleMenuMapping{}).Count(&mappingCount).Error)
	require.Zero(t, mappingCount, "no mapping should be created when permission seed fails")
}

// TC-IAM-TECH-037-D-01: DB에는 있지만 이번 yaml에는 없는 메뉴 id는
// orphanMenusDetected로 감지되어야 하며, 실제로 삭제되지는 않아야 한다.
func TestLoadAndRegisterMenusFromYAML_DetectsOrphanMenus_WithoutDeleting(t *testing.T) {
	db := setupMenuServiceTestDB(t)

	// yaml 재동기화 이전부터 DB에 존재하던, 이번 yaml에는 빠진 메뉴
	require.NoError(t, db.Create(&model.Menu{ID: "removed-from-yaml", DisplayName: "Removed", ResType: "menu"}).Error)

	missingPermPath := filepath.Join(t.TempDir(), "does-not-exist-permission.yaml")
	t.Setenv("MC_WEB_CONSOLE_MENU_PERMISSIONS", missingPermPath)

	menuPath := writeTempFile(t, "menu.yaml", testMenuYAML) // home만 포함, removed-from-yaml 없음

	// 메뉴가 이미 있으므로 가드를 넘기려면 force가 필요하다 (백업 파일은 임시 cwd에 쓴다)
	chdirTemp(t)
	s := NewMenuService(db)
	result, err := s.LoadAndRegisterMenusFromYAML(menuPath, true)
	require.NoError(t, err)
	require.Contains(t, result.OrphanMenuIDs, "removed-from-yaml", "yaml에서 빠진 기존 메뉴 id가 감지되어야 한다")

	// 감지만 하고 삭제는 하지 않아야 한다 — 재조회로 DB에 그대로 남아있는지 확인
	var stillExists model.Menu
	require.NoError(t, db.Where("id = ?", "removed-from-yaml").First(&stillExists).Error,
		"orphan으로 감지된 메뉴가 자동으로 삭제되면 안 된다")
}

// TC-IAM-TECH-037-D-02: yaml에 포함된 메뉴는 DB에도 있는 상태라면 orphan으로 감지되지 않아야 한다.
func TestLoadAndRegisterMenusFromYAML_NoOrphans_WhenYamlCoversAllExistingMenus(t *testing.T) {
	db := setupMenuServiceTestDB(t)
	require.NoError(t, db.Create(&model.Menu{ID: "home", DisplayName: "Home", ResType: "menu"}).Error)

	missingPermPath := filepath.Join(t.TempDir(), "does-not-exist-permission.yaml")
	t.Setenv("MC_WEB_CONSOLE_MENU_PERMISSIONS", missingPermPath)

	menuPath := writeTempFile(t, "menu.yaml", testMenuYAML)

	s := NewMenuService(db)
	result, err := s.LoadAndRegisterMenusFromYAML(menuPath, false)
	require.NoError(t, err)
	require.False(t, result.Skipped, "home만 있는 DB는 '시딩됨'으로 보지 않아야 한다")
	require.Empty(t, result.OrphanMenuIDs, "yaml이 기존 DB 메뉴를 모두 포함하면 orphan이 없어야 한다")
}

// chdirTemp는 force 재시딩 시 SaveRolePermissionBackupFile이 cwd 기준 asset/menu/backups/에
// 쓰는 백업 파일이 저장소를 오염시키지 않도록 임시 디렉토리로 이동한다.
func chdirTemp(t *testing.T) string {
	t.Helper()
	orig, err := os.Getwd()
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(orig) })
	return dir
}

// 최초 1회 가드: DB에 (home 외) 메뉴가 이미 있으면 force=false 호출은 아무것도 바꾸지 않고
// Skipped를 돌려줘야 한다 — 체이닝되는 권한 시딩도 실행되지 않는다.
func TestLoadAndRegisterMenusFromYAML_SkipsWhenMenusExist_WithoutForce(t *testing.T) {
	db := setupMenuServiceTestDB(t)
	seedAdminPlatformRole(t, db)
	require.NoError(t, db.Create(&model.Menu{ID: "users", DisplayName: "Edited In DB", ResType: "menu"}).Error)

	permPath := writeTempFile(t, "permission.yaml", testPermissionYAML)
	t.Setenv("MC_WEB_CONSOLE_MENU_PERMISSIONS", permPath)
	menuPath := writeTempFile(t, "menu.yaml", testMenuYAML)

	s := NewMenuService(db)
	result, err := s.LoadAndRegisterMenusFromYAML(menuPath, false)
	require.NoError(t, err)
	require.True(t, result.Skipped)
	require.Equal(t, 1, result.ExistingMenuCount)
	require.Zero(t, result.RegisteredCount)

	var mappingCount int64
	require.NoError(t, db.Model(&model.RoleMenuMapping{}).Count(&mappingCount).Error)
	require.Zero(t, mappingCount, "skip 시 권한 시딩도 실행되면 안 된다")

	var menu model.Menu
	require.NoError(t, db.Where("id = ?", "users").First(&menu).Error)
	require.Equal(t, "Edited In DB", menu.DisplayName, "skip 시 DB 편집이 유지되어야 한다")
}

// force=true면 가드를 넘어 시딩이 진행되고, 그 전에 역할 권한 백업 파일이 저장되어야 한다.
func TestLoadAndRegisterMenusFromYAML_ForceOverridesGuard_WithBackup(t *testing.T) {
	db := setupMenuServiceTestDB(t)
	seedAdminPlatformRole(t, db)
	require.NoError(t, db.Create(&model.Menu{ID: "users", DisplayName: "Users", ResType: "menu"}).Error)

	permPath := writeTempFile(t, "permission.yaml", testPermissionYAML)
	t.Setenv("MC_WEB_CONSOLE_MENU_PERMISSIONS", permPath)
	menuPath := writeTempFile(t, "menu.yaml", testMenuYAML)

	tmp := chdirTemp(t)
	s := NewMenuService(db)
	result, err := s.LoadAndRegisterMenusFromYAML(menuPath, true)
	require.NoError(t, err)
	require.False(t, result.Skipped)
	require.Equal(t, 1, result.ExistingMenuCount)
	require.Empty(t, result.BackupWarning)
	require.NotEmpty(t, result.BackupPath)
	require.FileExists(t, filepath.Join(tmp, result.BackupPath))
	require.Contains(t, result.OrphanMenuIDs, "users")
}

// 빈 DB에서는 force 없이도 시딩되고 백업은 만들지 않는다 (최초 설치 경로).
func TestLoadAndRegisterMenusFromYAML_FreshDB_SeedsWithoutBackup(t *testing.T) {
	db := setupMenuServiceTestDB(t)
	seedAdminPlatformRole(t, db)
	permPath := writeTempFile(t, "permission.yaml", testPermissionYAML)
	t.Setenv("MC_WEB_CONSOLE_MENU_PERMISSIONS", permPath)
	menuPath := writeTempFile(t, "menu.yaml", testMenuYAML)

	s := NewMenuService(db)
	result, err := s.LoadAndRegisterMenusFromYAML(menuPath, false)
	require.NoError(t, err)
	require.False(t, result.Skipped)
	require.Zero(t, result.ExistingMenuCount)
	require.Equal(t, 1, result.RegisteredCount)
	require.Empty(t, result.BackupPath)
}

// initial-menus2(본문 시딩)에도 동일한 가드가 적용된다. 잘못된 yaml은 가드보다 먼저 거부된다.
func TestRegisterMenusFromContent_GuardAndParseOrder(t *testing.T) {
	db := setupMenuServiceTestDB(t)
	require.NoError(t, db.Create(&model.Menu{ID: "users", DisplayName: "Users", ResType: "menu"}).Error)

	s := NewMenuService(db)
	result, err := s.RegisterMenusFromContent([]byte(testMenuYAML), false)
	require.NoError(t, err)
	require.True(t, result.Skipped)

	_, err = s.RegisterMenusFromContent([]byte("menus: [not: [valid"), false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "error unmarshalling")
}

// permission.yaml이 DB에 없는 메뉴 id를 참조하면 해당 매핑은 만들지 않고 목록으로 보고한다.
func TestInitializeMenuPermissionsFromYAML_ReportsMissingMenuIDs(t *testing.T) {
	db := setupMenuServiceTestDB(t)
	seedAdminPlatformRole(t, db)
	require.NoError(t, db.Create(&model.Menu{ID: "home", DisplayName: "Home", ResType: "menu"}).Error)

	permPath := writeTempFile(t, "permission.yaml", `
permissions:
  - role: admin
    menus:
      - home
      - ghost-menu
      - another-ghost
`)

	s := NewMenuService(db)
	missing, err := s.InitializeMenuPermissionsFromYAML(permPath)
	require.NoError(t, err)
	require.Equal(t, []string{"another-ghost", "ghost-menu"}, missing)

	var mappings []model.RoleMenuMapping
	require.NoError(t, db.Find(&mappings).Error)
	require.Len(t, mappings, 1)
	require.Equal(t, "home", mappings[0].MenuID)
}
