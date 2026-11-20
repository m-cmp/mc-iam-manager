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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// IAM-BUG-055: Master Role 이름 예약어, admin 이름 변경 금지, AssignPlatformRole 의 roleName 임의 부여

var reservedNameSamples = []string{"platformAdmin", "PlatformAdmin ", "ｐｌａｔｆｏｒｍＡｄｍｉｎ", "default-roles-mciam"}

func reservedNameCall(t *testing.T, h *RoleHandler, method, path string, params map[string]string, body string, fn func(*RoleHandler, echo.Context) error) *httptest.ResponseRecorder {
	t.Helper()
	e := authMethodTestEcho()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	for k, v := range params {
		c.SetParamNames(k)
		c.SetParamValues(v)
	}
	require.NoError(t, fn(h, c))
	return rec
}

func seedMasterRole(t *testing.T, db *gorm.DB, name string, types ...constants.IAMRoleType) *model.RoleMaster {
	t.Helper()
	role := &model.RoleMaster{Name: name}
	require.NoError(t, db.Create(role).Error)
	for _, rt := range types {
		require.NoError(t, db.Create(&model.RoleSub{RoleID: role.ID, RoleType: rt}).Error)
	}
	return role
}

func TestCreateMasterRole_ReservedName_400(t *testing.T) {
	db := setupAuthMethodTestDB(t)
	h := NewRoleHandler(db)
	creates := map[string]func(*RoleHandler, echo.Context) error{
		"CreateRole":          (*RoleHandler).CreateRole,
		"CreatePlatformRole":  (*RoleHandler).CreatePlatformRole,
		"CreateWorkspaceRole": (*RoleHandler).CreateWorkspaceRole,
		"CreateCspRoleMaster": (*RoleHandler).CreateCspRoleMaster,
	}
	for fnName, fn := range creates {
		for _, name := range reservedNameSamples {
			body := fmt.Sprintf(`{"name":%q,"roleTypes":["platform"]}`, name)
			rec := reservedNameCall(t, h, http.MethodPost, "/api/roles", nil, body, fn)
			assert.Equal(t, http.StatusBadRequest, rec.Code, "%s(%q): %s", fnName, name, rec.Body.String())
			assert.Contains(t, rec.Body.String(), "reserved role name", "%s(%q)", fnName, name)
		}
	}
	var count int64
	require.NoError(t, db.Model(&model.RoleMaster{}).Where("name IN ?", reservedNameSamples).Count(&count).Error)
	assert.Zero(t, count, "예약어 Master Role 이 저장되면 안 된다")
}

func TestUpdateMasterRole_RenameGuards(t *testing.T) {
	db := setupAuthMethodTestDB(t)
	h := NewRoleHandler(db)
	updates := []struct {
		name string
		fn   func(*RoleHandler, echo.Context) error
		sub  constants.IAMRoleType
	}{
		{"UpdateRole", (*RoleHandler).UpdateRole, constants.RoleTypePlatform},
		{"UpdatePlatformRole", (*RoleHandler).UpdatePlatformRole, constants.RoleTypePlatform},
		{"UpdateWorkspaceRole", (*RoleHandler).UpdateWorkspaceRole, constants.RoleTypeWorkspace},
		{"UpdateCspRoleMaster", (*RoleHandler).UpdateCspRoleMaster, constants.RoleTypeCSP},
	}
	for i, u := range updates {
		// 예약어로 이름 변경 → 400
		target := seedMasterRole(t, db, fmt.Sprintf("t055-rename-%d", i), u.sub)
		for _, name := range reservedNameSamples {
			body := fmt.Sprintf(`{"name":%q,"roleTypes":[%q]}`, name, u.sub)
			rec := reservedNameCall(t, h, http.MethodPut, "/api/roles/id/x", map[string]string{"roleId": fmt.Sprint(target.ID)}, body, u.fn)
			assert.Equal(t, http.StatusBadRequest, rec.Code, "%s → %q: %s", u.name, name, rec.Body.String())
		}
		var reloaded model.RoleMaster
		require.NoError(t, db.First(&reloaded, target.ID).Error)
		assert.Equal(t, target.Name, reloaded.Name, "%s: 예약어로 이름이 바뀌면 안 된다", u.name)

		// admin Master Role 이름 변경 → 400
		admin := seedMasterRole(t, db, map[int]string{0: "admin", 1: "Admin", 2: " ADMIN", 3: "ａｄｍｉｎ"}[i], u.sub)
		body := fmt.Sprintf(`{"name":"ops-%d","roleTypes":[%q]}`, i, u.sub)
		rec := reservedNameCall(t, h, http.MethodPut, "/api/roles/id/x", map[string]string{"roleId": fmt.Sprint(admin.ID)}, body, u.fn)
		assert.Equal(t, http.StatusBadRequest, rec.Code, "%s admin rename: %s", u.name, rec.Body.String())
		assert.Contains(t, rec.Body.String(), "cannot be renamed", u.name)
	}
}

func TestUpdateRole_AdminSameNameAndOtherRenameAllowed(t *testing.T) {
	db := setupAuthMethodTestDB(t)
	h := NewRoleHandler(db)

	// admin 이름을 그대로 두는 수정은 이름 변경 검사에 걸리지 않는다
	admin := seedMasterRole(t, db, "t055-admin-keep", constants.RoleTypePlatform)
	require.NoError(t, checkRoleRename("admin", "admin"))
	require.NoError(t, checkRoleRename("admin", " Admin "), "정규화가 같으면 이름 변경이 아니다")
	_ = admin

	// admin 이 아닌 Master Role 의 일반 이름 변경은 허용
	other := seedMasterRole(t, db, "t055-other", constants.RoleTypePlatform)
	body := `{"name":"t055-other-renamed","roleTypes":["platform"]}`
	rec := reservedNameCall(t, h, http.MethodPut, "/api/roles/id/x", map[string]string{"roleId": fmt.Sprint(other.ID)}, body, (*RoleHandler).UpdateRole)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var reloaded model.RoleMaster
	require.NoError(t, db.First(&reloaded, other.ID).Error)
	assert.Equal(t, "t055-other-renamed", reloaded.Name)
}

func TestAssignPlatformRole_RoleNameMustMatchDB(t *testing.T) {
	db := setupAuthMethodTestDB(t)
	h := NewRoleHandler(db)
	viewer := seedMasterRole(t, db, "t055-viewer", constants.RoleTypePlatform)

	// 요청 roleName 을 조작해 roleId 와 다른 realm 역할(platformAdmin)을 부여하려는 요청 → 400
	body := fmt.Sprintf(`{"userId":"1","roleId":"%d","roleName":"platformAdmin","roleType":"platform"}`, viewer.ID)
	rec := reservedNameCall(t, h, http.MethodPost, "/api/roles/assign/platform-role", nil, body, (*RoleHandler).AssignPlatformRole)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "does not match")

	// DB 에 예약어 이름이 남아 있는 레거시 데이터여도 Keycloak 에 부여하지 않는다 → 400
	legacy := seedMasterRole(t, db, "platformAdmin", constants.RoleTypePlatform)
	body = fmt.Sprintf(`{"userId":"1","roleId":"%d","roleType":"platform"}`, legacy.ID)
	rec = reservedNameCall(t, h, http.MethodPost, "/api/roles/assign/platform-role", nil, body, (*RoleHandler).AssignPlatformRole)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "reserved role name")
	require.NoError(t, db.Delete(legacy).Error)
}
