package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupDeleteRoleLastSubTestDB는 role_master가 하나 이상의 role_sub을 가질 수 있는
// 상황을 검증하기 위한 전용 테스트 DB다. 이름을 t.Name()으로 스코프해 다른 테스트
// 파일의 shared-cache 인메모리 DB와 충돌하지 않게 한다.
func setupDeleteRoleLastSubTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.RoleMaster{},
		&model.RoleSub{},
		&model.RoleMasterCspRoleMapping{},
		&model.RoleMenuMapping{},
	))
	return db
}

func newDeleteSubRequestContext(t *testing.T, method string, path string, roleID uint) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)
	c.SetParamNames("roleId")
	c.SetParamValues(fmt.Sprint(roleID))
	return c, rec
}

// TC-IAM-BUG-035-04: platform sub가 이 역할의 유일한 sub일 때 DeletePlatformRole로
// 지우면 role_master도 함께(고아로 남지 않고) 삭제되어야 한다.
func TestDeletePlatformRole_LastSub_AlsoDeletesRoleMaster(t *testing.T) {
	db := setupDeleteRoleLastSubTestDB(t)
	role := model.RoleMaster{Name: "tc-iam-bug-035-platform-only"}
	require.NoError(t, db.Create(&role).Error)
	require.NoError(t, db.Create(&model.RoleSub{RoleID: role.ID, RoleType: constants.RoleTypePlatform}).Error)

	h := NewRoleHandler(db)
	c, rec := newDeleteSubRequestContext(t, http.MethodDelete, "/api/roles/platform-roles/id/x", role.ID)
	require.NoError(t, h.DeletePlatformRole(c))
	require.Equal(t, http.StatusNoContent, rec.Code)

	var masterCount int64
	require.NoError(t, db.Model(&model.RoleMaster{}).Where("id = ?", role.ID).Count(&masterCount).Error)
	require.Zero(t, masterCount, "마지막 남은 sub을 지웠으므로 role_master도 함께 삭제되어야 한다")
}

// TC-IAM-BUG-035-05: platform sub 외에 workspace sub이 남아 있으면 DeletePlatformRole은
// platform sub만 지우고 role_master와 다른 sub은 그대로 유지해야 한다.
func TestDeletePlatformRole_OtherSubsRemain_KeepsRoleMaster(t *testing.T) {
	db := setupDeleteRoleLastSubTestDB(t)
	role := model.RoleMaster{Name: "tc-iam-bug-035-platform-and-workspace"}
	require.NoError(t, db.Create(&role).Error)
	require.NoError(t, db.Create(&model.RoleSub{RoleID: role.ID, RoleType: constants.RoleTypePlatform}).Error)
	require.NoError(t, db.Create(&model.RoleSub{RoleID: role.ID, RoleType: constants.RoleTypeWorkspace}).Error)

	h := NewRoleHandler(db)
	c, rec := newDeleteSubRequestContext(t, http.MethodDelete, "/api/roles/platform-roles/id/x", role.ID)
	require.NoError(t, h.DeletePlatformRole(c))
	require.Equal(t, http.StatusNoContent, rec.Code)

	var masterCount, platformSubCount, workspaceSubCount int64
	require.NoError(t, db.Model(&model.RoleMaster{}).Where("id = ?", role.ID).Count(&masterCount).Error)
	require.NoError(t, db.Model(&model.RoleSub{}).Where("role_id = ? AND role_type = ?", role.ID, constants.RoleTypePlatform).Count(&platformSubCount).Error)
	require.NoError(t, db.Model(&model.RoleSub{}).Where("role_id = ? AND role_type = ?", role.ID, constants.RoleTypeWorkspace).Count(&workspaceSubCount).Error)
	require.EqualValues(t, 1, masterCount, "다른 sub이 남아있으므로 role_master는 삭제되면 안 된다")
	require.Zero(t, platformSubCount, "요청한 platform sub은 삭제되어야 한다")
	require.EqualValues(t, 1, workspaceSubCount, "요청하지 않은 workspace sub은 그대로 유지되어야 한다")
}

// TC-IAM-BUG-035-06: workspace sub가 이 역할의 유일한 sub일 때 DeleteWorkspaceRole로
// 지우면 role_master도 함께 삭제되어야 한다.
func TestDeleteWorkspaceRole_LastSub_AlsoDeletesRoleMaster(t *testing.T) {
	db := setupDeleteRoleLastSubTestDB(t)
	role := model.RoleMaster{Name: "tc-iam-bug-035-workspace-only"}
	require.NoError(t, db.Create(&role).Error)
	require.NoError(t, db.Create(&model.RoleSub{RoleID: role.ID, RoleType: constants.RoleTypeWorkspace}).Error)

	h := NewRoleHandler(db)
	c, rec := newDeleteSubRequestContext(t, http.MethodDelete, "/api/roles/workspace-roles/id/x", role.ID)
	require.NoError(t, h.DeleteWorkspaceRole(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var masterCount int64
	require.NoError(t, db.Model(&model.RoleMaster{}).Where("id = ?", role.ID).Count(&masterCount).Error)
	require.Zero(t, masterCount, "마지막 남은 sub을 지웠으므로 role_master도 함께 삭제되어야 한다")
}
