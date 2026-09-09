package repository

import (
	"testing"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupMenuDeleteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.Menu{},
		&model.RoleMenuMapping{},
	))
	// model.MciamPermission의 created_at/updated_at 컬럼은 postgres 전용 default:now()를
	// 쓰고 있어 sqlite AutoMigrate가 실패한다. 테스트에서는 sqlite 호환 스키마로 직접 생성.
	require.NoError(t, db.Exec(`
CREATE TABLE mcmp_mciam_permissions (
  id varchar(255) PRIMARY KEY,
  framework_id varchar(100) NOT NULL,
  resource_type_id varchar(100) NOT NULL,
  action varchar(100) NOT NULL,
  name varchar(100) NOT NULL,
  description varchar(1000),
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
)`).Error)
	return db
}

func seedMenuWithChildAndMappings(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Create(&model.Menu{ID: "parentmenu", DisplayName: "Parent", ResType: "menu"}).Error)
	require.NoError(t, db.Create(&model.Menu{ID: "childmenu", ParentID: "parentmenu", DisplayName: "Child", ResType: "menu"}).Error)

	require.NoError(t, db.Create(&model.RoleMenuMapping{RoleID: 1, MenuID: "parentmenu"}).Error)
	require.NoError(t, db.Create(&model.RoleMenuMapping{RoleID: 1, MenuID: "childmenu"}).Error)
	require.NoError(t, db.Create(&model.RoleMenuMapping{RoleID: 2, MenuID: "childmenu"}).Error)

	require.NoError(t, db.Create(&model.MciamPermission{
		ID: "menu:menu:view:parentmenu", FrameworkID: "menu", ResourceTypeID: "menu",
		Action: "view:parentmenu", Name: "View Menu Parent",
	}).Error)
	require.NoError(t, db.Create(&model.MciamPermission{
		ID: "menu:menu:view:childmenu", FrameworkID: "menu", ResourceTypeID: "menu",
		Action: "view:childmenu", Name: "View Menu Child",
	}).Error)
}

// TC-IAM-TECH-037-A-01: 부모 메뉴 삭제 시 하위 메뉴뿐 아니라 관련 role-menu 매핑과
// view 권한(mciam_permissions)도 같은 트랜잭션에서 함께 정리되어야 한다
// (수정 전에는 mcmp_menus만 지우고 매핑/권한이 orphan으로 남았음).
func TestDeleteMenuWithChildren_CleansUpMappingsAndPermissions(t *testing.T) {
	db := setupMenuDeleteTestDB(t)
	seedMenuWithChildAndMappings(t, db)
	r := &MenuRepository{db: db}

	err := r.DeleteMenuWithChildren("parentmenu")
	require.NoError(t, err)

	var menuCount int64
	require.NoError(t, db.Model(&model.Menu{}).Where("id IN ?", []string{"parentmenu", "childmenu"}).Count(&menuCount).Error)
	require.Zero(t, menuCount, "부모+자식 메뉴가 모두 삭제되어야 한다")

	var mappingCount int64
	require.NoError(t, db.Model(&model.RoleMenuMapping{}).Where("menu_id IN ?", []string{"parentmenu", "childmenu"}).Count(&mappingCount).Error)
	require.Zero(t, mappingCount, "삭제된 메뉴에 딸린 role-menu 매핑도 모두 삭제되어야 한다")

	var permCount int64
	require.NoError(t, db.Model(&model.MciamPermission{}).
		Where("id IN ?", []string{"menu:menu:view:parentmenu", "menu:menu:view:childmenu"}).
		Count(&permCount).Error)
	require.Zero(t, permCount, "삭제된 메뉴의 view 권한도 모두 삭제되어야 한다")
}

// TC-IAM-TECH-037-A-02: 존재하지 않는 메뉴 id 삭제 시도는 ErrMenuNotFound를 반환하고
// 아무 것도 지우지 않아야 한다.
func TestDeleteMenuWithChildren_NotFound_LeavesDataUntouched(t *testing.T) {
	db := setupMenuDeleteTestDB(t)
	seedMenuWithChildAndMappings(t, db)
	r := &MenuRepository{db: db}

	err := r.DeleteMenuWithChildren("does-not-exist")
	require.ErrorIs(t, err, ErrMenuNotFound)

	var menuCount int64
	require.NoError(t, db.Model(&model.Menu{}).Count(&menuCount).Error)
	require.Equal(t, int64(2), menuCount, "존재하지 않는 id 삭제 시도는 기존 메뉴에 영향을 주지 않아야 한다")
}
