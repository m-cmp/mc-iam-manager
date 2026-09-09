package handler

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCspRoleMasterTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// NewRoleHandler가 구성하는 MenuRepository 등 다른 리포지토리가 자체적으로
	// AutoMigrate를 호출하며 새 커넥션을 열 수 있어 shared cache로 같은 DB를 공유한다.
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	// CspRolePermission(mcmp_csp_role_permissions)은 의도적으로 마이그레이트하지 않는다 —
	// 실 환경(main.go AutoMigrate)에도 이 테이블이 없다(IAM-BUG-021). 여기서 만들어버리면
	// 관련 회귀를 테스트가 잡아내지 못한다.
	require.NoError(t, db.AutoMigrate(
		&model.RoleMaster{},
		&model.RoleSub{},
		&model.CspRole{},
		&model.RoleMasterCspRoleMapping{},
		&model.CspPolicy{},
		&model.CspRolePolicyMapping{},
	))
	return db
}

func newTestValidatorEcho() *echo.Echo {
	e := echo.New()
	e.Validator = &testValidator{v: validator.New()}
	return e
}

func uintToStr(v uint) string {
	return strconv.FormatUint(uint64(v), 10)
}

// TC-IAM-TECH-022-02: DeleteCspRole(레코드 삭제)는 CspRole 레코드와 그 레코드를 참조하는
// RoleMasterCspRoleMapping을 정리하고, RoleMaster/RoleSub(csp)는 건드리지 않아야 한다.
func TestDeleteCspRole_DeletesRecordAndMapping_KeepsRoleMaster(t *testing.T) {
	db := setupCspRoleMasterTestDB(t)
	h := NewRoleHandler(db)

	role := &model.RoleMaster{Name: "tc-iam-tech-022-role"}
	require.NoError(t, db.Create(role).Error)
	require.NoError(t, db.Create(&model.RoleSub{RoleID: role.ID, RoleType: constants.RoleTypeCSP}).Error)

	cspRole := &model.CspRole{Name: "mciam-test-role", CspType: "gcp"}
	require.NoError(t, db.Create(cspRole).Error)
	require.NoError(t, db.Create(&model.RoleMasterCspRoleMapping{
		RoleID: role.ID, AuthMethod: constants.AuthMethodOIDC, CspRoleID: cspRole.ID,
	}).Error)

	e := newTestValidatorEcho()
	req := httptest.NewRequest(http.MethodDelete, "/api/roles/csp/id/"+uintToStr(cspRole.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("roleId")
	c.SetParamValues(uintToStr(cspRole.ID))

	require.NoError(t, h.DeleteCspRole(c))
	require.Equal(t, http.StatusNoContent, rec.Code)

	var cspRoleCount, mappingCount int64
	require.NoError(t, db.Model(&model.CspRole{}).Where("id = ?", cspRole.ID).Count(&cspRoleCount).Error)
	require.NoError(t, db.Model(&model.RoleMasterCspRoleMapping{}).Where("csp_role_id = ?", cspRole.ID).Count(&mappingCount).Error)
	assert.Zero(t, cspRoleCount, "CspRole 레코드가 삭제되어야 한다")
	assert.Zero(t, mappingCount, "참조 매핑이 정리되어야 한다")

	var stillThereRole model.RoleMaster
	require.NoError(t, db.Where("id = ?", role.ID).First(&stillThereRole).Error, "RoleMaster는 그대로 유지되어야 한다")

	var subCount int64
	require.NoError(t, db.Model(&model.RoleSub{}).Where("role_id = ? AND role_type = ?", role.ID, constants.RoleTypeCSP).Count(&subCount).Error)
	assert.Equal(t, int64(1), subCount, "RoleSub(csp)도 그대로 유지되어야 한다")
}

// TC-IAM-TECH-022-03: 존재하지 않는 CSP Role ID는 404를 반환해야 한다.
func TestDeleteCspRole_NotFound_Returns404(t *testing.T) {
	db := setupCspRoleMasterTestDB(t)
	h := NewRoleHandler(db)

	e := newTestValidatorEcho()
	req := httptest.NewRequest(http.MethodDelete, "/api/roles/csp/id/9999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("roleId")
	c.SetParamValues("9999")

	require.NoError(t, h.DeleteCspRole(c))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// TC-IAM-TECH-022-04: DeleteCspRoleMaster는 Predefined 역할을 삭제할 수 없어야 한다
// (DeletePlatformRole/DeleteWorkspaceRole과 동일한 가드).
func TestDeleteCspRoleMaster_PredefinedRole_Forbidden(t *testing.T) {
	db := setupCspRoleMasterTestDB(t)
	h := NewRoleHandler(db)

	role := &model.RoleMaster{Name: "tc-iam-tech-022-predefined", Predefined: true}
	require.NoError(t, db.Create(role).Error)
	require.NoError(t, db.Create(&model.RoleSub{RoleID: role.ID, RoleType: constants.RoleTypeCSP}).Error)

	e := newTestValidatorEcho()
	req := httptest.NewRequest(http.MethodDelete, "/api/roles/csp-roles/master/id/"+uintToStr(role.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("roleId")
	c.SetParamValues(uintToStr(role.ID))

	require.NoError(t, h.DeleteCspRoleMaster(c))
	assert.Equal(t, http.StatusForbidden, rec.Code)

	var subCount int64
	require.NoError(t, db.Model(&model.RoleSub{}).Where("role_id = ?", role.ID).Count(&subCount).Error)
	assert.Equal(t, int64(1), subCount, "Predefined 역할은 RoleSub도 그대로 유지되어야 한다")
}

// TC-IAM-TECH-022-05: DeleteCspRoleMaster는 존재하지 않는 역할에 404를 반환해야 한다.
func TestDeleteCspRoleMaster_NotFound_Returns404(t *testing.T) {
	db := setupCspRoleMasterTestDB(t)
	h := NewRoleHandler(db)

	e := newTestValidatorEcho()
	req := httptest.NewRequest(http.MethodDelete, "/api/roles/csp-roles/master/id/9999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("roleId")
	c.SetParamValues("9999")

	require.NoError(t, h.DeleteCspRoleMaster(c))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// TC-IAM-BUG-035-07: csp sub가 이 역할의 유일한 sub일 때 DeleteCspRoleMaster로 지우면
// role_master도 함께(고아로 남지 않고) 삭제되어야 한다.
func TestDeleteCspRoleMaster_LastSub_AlsoDeletesRoleMaster(t *testing.T) {
	db := setupCspRoleMasterTestDB(t)
	h := NewRoleHandler(db)

	role := &model.RoleMaster{Name: "tc-iam-bug-035-csp-only"}
	require.NoError(t, db.Create(role).Error)
	require.NoError(t, db.Create(&model.RoleSub{RoleID: role.ID, RoleType: constants.RoleTypeCSP}).Error)

	e := newTestValidatorEcho()
	req := httptest.NewRequest(http.MethodDelete, "/api/roles/csp-roles/master/id/"+uintToStr(role.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("roleId")
	c.SetParamValues(uintToStr(role.ID))

	require.NoError(t, h.DeleteCspRoleMaster(c))
	require.Equal(t, http.StatusNoContent, rec.Code)

	var masterCount int64
	require.NoError(t, db.Model(&model.RoleMaster{}).Where("id = ?", role.ID).Count(&masterCount).Error)
	assert.Zero(t, masterCount, "마지막 남은 sub을 지웠으므로 role_master도 함께 삭제되어야 한다")
}

// TC-IAM-BUG-035-08: csp sub 외에 platform sub이 남아 있으면 DeleteCspRoleMaster는 csp
// sub만 지우고 role_master와 다른 sub은 그대로 유지해야 한다.
func TestDeleteCspRoleMaster_OtherSubsRemain_KeepsRoleMaster(t *testing.T) {
	db := setupCspRoleMasterTestDB(t)
	h := NewRoleHandler(db)

	role := &model.RoleMaster{Name: "tc-iam-bug-035-csp-and-platform"}
	require.NoError(t, db.Create(role).Error)
	require.NoError(t, db.Create(&model.RoleSub{RoleID: role.ID, RoleType: constants.RoleTypeCSP}).Error)
	require.NoError(t, db.Create(&model.RoleSub{RoleID: role.ID, RoleType: constants.RoleTypePlatform}).Error)

	e := newTestValidatorEcho()
	req := httptest.NewRequest(http.MethodDelete, "/api/roles/csp-roles/master/id/"+uintToStr(role.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("roleId")
	c.SetParamValues(uintToStr(role.ID))

	require.NoError(t, h.DeleteCspRoleMaster(c))
	require.Equal(t, http.StatusNoContent, rec.Code)

	var masterCount, cspSubCount, platformSubCount int64
	require.NoError(t, db.Model(&model.RoleMaster{}).Where("id = ?", role.ID).Count(&masterCount).Error)
	require.NoError(t, db.Model(&model.RoleSub{}).Where("role_id = ? AND role_type = ?", role.ID, constants.RoleTypeCSP).Count(&cspSubCount).Error)
	require.NoError(t, db.Model(&model.RoleSub{}).Where("role_id = ? AND role_type = ?", role.ID, constants.RoleTypePlatform).Count(&platformSubCount).Error)
	assert.EqualValues(t, 1, masterCount, "다른 sub이 남아있으므로 role_master는 삭제되면 안 된다")
	assert.Zero(t, cspSubCount, "요청한 csp sub은 삭제되어야 한다")
	assert.EqualValues(t, 1, platformSubCount, "요청하지 않은 platform sub은 그대로 유지되어야 한다")
}

// TC-IAM-BUG-035-09: 실제로 RoleMasterCspRoleMapping이 존재하면 DeleteCspRoleMaster는
// 여전히 403으로 거부해야 한다(가드 로직을 고치면서 이 경우까지 뚫려버리면 안 된다).
func TestDeleteCspRoleMaster_HasActualCspMapping_Forbidden(t *testing.T) {
	db := setupCspRoleMasterTestDB(t)
	h := NewRoleHandler(db)

	role := &model.RoleMaster{Name: "tc-iam-bug-035-csp-with-mapping"}
	require.NoError(t, db.Create(role).Error)
	require.NoError(t, db.Create(&model.RoleSub{RoleID: role.ID, RoleType: constants.RoleTypeCSP}).Error)

	cspRole := &model.CspRole{Name: "mciam-test-role-2", CspType: "gcp"}
	require.NoError(t, db.Create(cspRole).Error)
	require.NoError(t, db.Create(&model.RoleMasterCspRoleMapping{
		RoleID: role.ID, AuthMethod: constants.AuthMethodOIDC, CspRoleID: cspRole.ID,
	}).Error)

	e := newTestValidatorEcho()
	req := httptest.NewRequest(http.MethodDelete, "/api/roles/csp-roles/master/id/"+uintToStr(role.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("roleId")
	c.SetParamValues(uintToStr(role.ID))

	require.NoError(t, h.DeleteCspRoleMaster(c))
	assert.Equal(t, http.StatusForbidden, rec.Code)

	var masterCount, subCount int64
	require.NoError(t, db.Model(&model.RoleMaster{}).Where("id = ?", role.ID).Count(&masterCount).Error)
	require.NoError(t, db.Model(&model.RoleSub{}).Where("role_id = ?", role.ID).Count(&subCount).Error)
	assert.EqualValues(t, 1, masterCount, "매핑이 남아있으므로 role_master는 삭제되면 안 된다")
	assert.EqualValues(t, 1, subCount, "매핑이 남아있으므로 role_sub도 삭제되면 안 된다")
}

// TC-IAM-TECH-022-06: CreateCspRoleMaster는 CreatePlatformRole/CreateWorkspaceRole과
// 대칭적으로 RoleSub(csp)를 요청 개수만큼 정확히 생성해야 한다(IAM-BUG-019 회귀 패턴 재확인).
func TestCreateCspRoleMaster_CreatesRoleSubsWithCspType(t *testing.T) {
	db := setupCspRoleMasterTestDB(t)
	h := NewRoleHandler(db)

	e := newTestValidatorEcho()
	body := `{"name":"tc-iam-tech-022-create","roleTypes":["csp"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/roles/csp-roles/master", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	require.NoError(t, h.CreateCspRoleMaster(c))
	require.Equal(t, http.StatusCreated, rec.Code)

	var role model.RoleMaster
	require.NoError(t, db.Where("name = ?", "tc-iam-tech-022-create").First(&role).Error)

	var subs []model.RoleSub
	require.NoError(t, db.Where("role_id = ?", role.ID).Find(&subs).Error)
	require.Len(t, subs, 1)
	assert.Equal(t, constants.RoleTypeCSP, subs[0].RoleType)
}
