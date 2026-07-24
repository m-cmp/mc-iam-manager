package service

import (
	"testing"
	"time"

	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRolePermissionBackupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.RoleMaster{},
		&model.RoleSub{},
		&model.Menu{},
		&model.RoleMenuMapping{},
	))
	return db
}

func seedPlatformRole(t *testing.T, db *gorm.DB, name string) *model.RoleMaster {
	t.Helper()
	role := &model.RoleMaster{Name: name, Description: name, Predefined: true}
	require.NoError(t, db.Create(role).Error)
	require.NoError(t, db.Create(&model.RoleSub{
		RoleID:    role.ID,
		RoleType:  constants.RoleTypePlatform,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}).Error)
	return role
}

func seedMenu(t *testing.T, db *gorm.DB, id, display string) {
	t.Helper()
	require.NoError(t, db.Create(&model.Menu{
		ID:               id,
		DisplayName:      display,
		ResType:          "menu",
		Priority:         1,
		MenuNumber:       1,
		ViewType:         "local",
		FrameworkService: "mc-web-console-front",
		Path:             "",
	}).Error)
}

func TestBackupAndRestoreRolePermissions_Additive(t *testing.T) {
	db := setupRolePermissionBackupTestDB(t)
	svc := NewMenuService(db)

	admin := seedPlatformRole(t, db, "admin")
	seedPlatformRole(t, db, "operator")
	seedMenu(t, db, "operations", "Operations")
	seedMenu(t, db, "observability", "Monitorings")
	seedMenu(t, db, "costanalysis", "Cost Analysis")

	require.NoError(t, svc.CreateRoleMenuMappings([]*model.RoleMenuMapping{
		{RoleID: admin.ID, MenuID: "operations"},
		{RoleID: admin.ID, MenuID: "observability"},
	}))

	backup, err := svc.BackupRolePermissions([]string{"admin"}, []string{"menus"})
	require.NoError(t, err)
	require.Equal(t, "role-permission-backup", backup.Kind)
	require.Equal(t, "db", backup.Source)
	require.Len(t, backup.Permissions, 1)
	require.Equal(t, "admin", backup.Permissions[0].Role)
	require.Equal(t, []string{"observability", "operations"}, backup.Permissions[0].Menus)

	// wipe mapping then additive restore from backup that also adds costanalysis
	require.NoError(t, svc.DeleteRoleMenuMappingsByRoleID(admin.ID))
	backup.Permissions[0].Menus = append(backup.Permissions[0].Menus, "costanalysis")

	result, err := svc.RestoreRolePermissions(backup, "additive", []string{"menus"})
	require.NoError(t, err)
	require.Equal(t, 1, result.RolesProcessed)
	require.Equal(t, 3, result.MenusAdded)

	ids, err := svc.menuMappingRepo.GetMappedMenuIDs(admin.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"operations", "observability", "costanalysis"}, ids)
}

func TestRestoreRolePermissions_ReplaceRole(t *testing.T) {
	db := setupRolePermissionBackupTestDB(t)
	svc := NewMenuService(db)

	admin := seedPlatformRole(t, db, "admin")
	seedMenu(t, db, "operations", "Operations")
	seedMenu(t, db, "observability", "Monitorings")
	seedMenu(t, db, "costanalysis", "Cost Analysis")

	require.NoError(t, svc.CreateRoleMenuMappings([]*model.RoleMenuMapping{
		{RoleID: admin.ID, MenuID: "operations"},
		{RoleID: admin.ID, MenuID: "observability"},
		{RoleID: admin.ID, MenuID: "costanalysis"},
	}))

	backup := &model.RolePermissionBackup{
		Kind:     "role-permission-backup",
		Source:   "db",
		Sections: []string{"menus"},
		Permissions: []model.RolePermissionEntry{
			{Role: "admin", Menus: []string{"operations", "observability"}},
		},
	}

	result, err := svc.RestoreRolePermissions(backup, "replace-role", []string{"menus"})
	require.NoError(t, err)
	require.Equal(t, 3, result.MenusRemoved)
	require.Equal(t, 2, result.MenusAdded)

	ids, err := svc.menuMappingRepo.GetMappedMenuIDs(admin.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"operations", "observability"}, ids)
}

func TestParseRolePermissionBackupYAML(t *testing.T) {
	raw := []byte(`
kind: role-permission-backup
backupAt: "2026-07-15T12:00:00+09:00"
source: db
sections: [menus]
permissions:
  - role: viewer
    menus: [operations]
    operations: []
    csps: []
`)
	backup, err := ParseRolePermissionBackupYAML(raw)
	require.NoError(t, err)
	require.Equal(t, "viewer", backup.Permissions[0].Role)
	require.Equal(t, []string{"operations"}, backup.Permissions[0].Menus)
}
