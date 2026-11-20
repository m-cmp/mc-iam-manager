package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 여정 1: Master Role 먼저 저장, 대상 CSP Role 항목별 결과·경고, PUT 변경분 반영 (IAM-BUG-043)

type roleSaveResponseBody struct {
	ID             uint            `json:"id"`
	Name           string          `json:"name"`
	CspRoleResults []cspRoleResult `json:"cspRoleResults"`
	Warnings       []string        `json:"warnings"`
}

// seedAwsManagementConfig 관리용 Role(role_arn) 이 있는 AWS OIDC Config. 이 Role 은 매핑 대상이 될 수 없다
func seedAwsManagementConfig(t *testing.T, db *gorm.DB, name, mgmtRole string) *model.CspIdpConfig {
	t.Helper()
	account := &model.CspAccount{Name: name + "-account", CspType: "aws"}
	require.NoError(t, db.Create(account).Error)
	cfg := &model.CspIdpConfig{Name: name, CspAccountID: account.ID, AuthMethod: model.AuthMethodOIDC, IsActive: true,
		Config: map[string]string{"role_arn": "arn:aws:iam::111111111111:role/" + mgmtRole}}
	require.NoError(t, db.Create(cfg).Error)
	return cfg
}

func callRoleSave(t *testing.T, h *RoleHandler, method string, roleID uint, body string) (*httptest.ResponseRecorder, roleSaveResponseBody) {
	t.Helper()
	e := authMethodTestEcho()
	req := httptest.NewRequest(method, "/api/roles", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if method == http.MethodPut {
		c.SetParamNames("roleId")
		c.SetParamValues(fmt.Sprint(roleID))
		require.NoError(t, h.UpdateRole(c))
	} else {
		require.NoError(t, h.CreateRole(c))
	}
	var out roleSaveResponseBody
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec, out
}

func resultsByName(rs []cspRoleResult) map[string]cspRoleResult {
	m := map[string]cspRoleResult{}
	for _, r := range rs {
		m[r.CspRoleName] = r
	}
	return m
}

func countMappings(t *testing.T, db *gorm.DB, roleID uint) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&model.RoleMasterCspRoleMapping{}).Where("role_id = ?", roleID).Count(&n).Error)
	return n
}

func TestCreateRole_EmptyCspRoleName_400_NoMasterRole(t *testing.T) {
	db := setupAuthMethodTestDB(t)
	h := NewRoleHandler(db)
	rec, _ := callRoleSave(t, h, http.MethodPost, 0, `{"name":"t043-empty","roleTypes":["csp"],"cspRoles":[{"cspRoleName":"","cspType":"aws"}]}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "cspRoleName is required")
	var n int64
	require.NoError(t, db.Model(&model.RoleMaster{}).Where("name = ?", "t043-empty").Count(&n).Error)
	assert.Zero(t, n, "요청 값 오류면 Master Role 도 만들지 않는다 (자동 명명 없음)")
}

func TestCreateRole_PartialFailure_MasterSavedAndResultsReported(t *testing.T) {
	db := setupAuthMethodTestDB(t)
	h := NewRoleHandler(db)
	seedAwsManagementConfig(t, db, "t043-create-mgmt", "mciam-t043-create-mgmt")
	gcp := seedSamlIdpConfig(t, db, "t043-create-gcp", "gcp")

	body := `{"name":"t043-partial","roleTypes":["csp"],"cspRoles":[` + samlCspRoleJSON("t043-ok", gcp.ID) +
		`,{"cspRoleName":"t043-create-mgmt","cspType":"aws"}]}`
	rec, out := callRoleSave(t, h, http.MethodPost, 0, body)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Equal(t, "t043-partial", out.Name, "역할 필드는 그대로 응답한다")
	require.NotZero(t, out.ID)

	rs := resultsByName(out.CspRoleResults)
	require.Len(t, rs, 2)
	assert.Equal(t, "registered_and_mapped", rs["mciam-t043-ok"].Result)
	assert.Equal(t, "failed", rs["mciam-t043-create-mgmt"].Result)
	assert.Equal(t, "protected", rs["mciam-t043-create-mgmt"].ErrorCode)
	assert.Zero(t, rs["mciam-t043-create-mgmt"].CspRoleID, "protected 는 레코드가 없어 cspRoleId 를 비운다")
	require.Len(t, out.Warnings, 1)
	assert.Contains(t, out.Warnings[0], "mciam-t043-create-mgmt")
	assert.Equal(t, int64(1), countMappings(t, db, out.ID), "실패한 항목만 매핑에서 빠진다")
}

func TestUpdateRole_AppliesOnlyChanges(t *testing.T) {
	db := setupAuthMethodTestDB(t)
	h := NewRoleHandler(db)
	seedAwsManagementConfig(t, db, "t043-update-mgmt", "mciam-t043-update-mgmt")
	gcp := seedSamlIdpConfig(t, db, "t043-update-gcp", "gcp")

	_, created := callRoleSave(t, h, http.MethodPost, 0,
		`{"name":"t043-update","roleTypes":["csp"],"cspRoles":[`+samlCspRoleJSON("t043-keep", gcp.ID)+`]}`)
	require.NotZero(t, created.ID)
	require.Equal(t, int64(1), countMappings(t, db, created.ID))

	// 기존 매핑 유지 + 실패 항목 추가 → 기존은 unchanged, 추가 항목만 failed
	rec, out := callRoleSave(t, h, http.MethodPut, created.ID,
		`{"name":"t043-update","roleTypes":["csp"],"cspRoles":[`+samlCspRoleJSON("t043-keep", gcp.ID)+`,{"cspRoleName":"t043-update-mgmt","cspType":"aws"}]}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rs := resultsByName(out.CspRoleResults)
	assert.Equal(t, "unchanged", rs["mciam-t043-keep"].Result)
	assert.Equal(t, "failed", rs["mciam-t043-update-mgmt"].Result)
	assert.Len(t, out.Warnings, 1)
	assert.Equal(t, int64(1), countMappings(t, db, created.ID), "기존 매핑은 지우지 않는다")

	// 다른 항목으로 교체 → 빠진 항목은 해제, 새 항목은 매핑
	rec, out = callRoleSave(t, h, http.MethodPut, created.ID,
		`{"name":"t043-update","roleTypes":["csp"],"cspRoles":[`+samlCspRoleJSON("t043-new", gcp.ID)+`]}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rs = resultsByName(out.CspRoleResults)
	assert.Equal(t, "unmapped", rs["mciam-t043-keep"].Result)
	assert.Equal(t, "registered_and_mapped", rs["mciam-t043-new"].Result)
	assert.Empty(t, out.Warnings)
	assert.Equal(t, int64(1), countMappings(t, db, created.ID))

	// cspRoles 를 보내지 않으면 매핑을 건드리지 않는다
	rec, out = callRoleSave(t, h, http.MethodPut, created.ID, `{"name":"t043-update","roleTypes":["csp"]}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Empty(t, out.CspRoleResults)
	assert.Equal(t, int64(1), countMappings(t, db, created.ID))
}

func TestCheckCspRoles_RequiresBody(t *testing.T) {
	db := setupAuthMethodTestDB(t)
	h := NewRoleHandler(db)
	e := authMethodTestEcho()
	req := httptest.NewRequest(http.MethodPost, "/api/roles/csp/check", strings.NewReader(`{"cspRoles":[]}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	require.NoError(t, h.CheckCspRoles(e.NewContext(req, rec)))
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	seedAwsManagementConfig(t, db, "t043-check-mgmt", "mciam-t043-check-mgmt")
	req = httptest.NewRequest(http.MethodPost, "/api/roles/csp/check", strings.NewReader(`{"cspRoles":[{"cspRoleName":"t043-check-mgmt","cspType":"aws"},{"cspRoleName":"","cspType":"aws"}]}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	require.NoError(t, h.CheckCspRoles(e.NewContext(req, rec)))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"status":"protected"`)
	assert.Contains(t, rec.Body.String(), `"status":"invalid"`)
}

func TestFailedCspRoleResult_CspRoleIDOnlyForCspError(t *testing.T) {
	base := cspRoleResult{CspRoleName: "mciam-t045-x", CspType: "aws", CspRoleID: 42}

	r := failedCspRoleResult(base, errors.New("failed to create AWS role mciam-t045-x: ValidationError"))
	assert.Equal(t, "csp_error", r.ErrorCode)
	assert.Equal(t, uint(42), r.CspRoleID, "csp_error 는 남은 failed 레코드 id 를 그대로 둔다")

	r = failedCspRoleResult(base, &service.CspRoleRequestError{Msg: "cspRoleName is required"})
	assert.Equal(t, "invalid_request", r.ErrorCode)
	assert.Zero(t, r.CspRoleID)

	r = failedCspRoleResult(base, &service.CspRoleProtectedError{Msg: "management role"})
	assert.Equal(t, "protected", r.ErrorCode)
	assert.Zero(t, r.CspRoleID)
}
