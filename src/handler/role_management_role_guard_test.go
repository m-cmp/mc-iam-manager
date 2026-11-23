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
)

func guardCall(t *testing.T, method, path, body string, params map[string]string, fn func(echo.Context) error) *httptest.ResponseRecorder {
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
	require.NoError(t, fn(c))
	return rec
}

// 관리용 Role 보호: 구 매핑 API·CspRole 등록·수정·삭제 409, 매핑 해제는 허용
func TestManagementRoleGuard_Handlers(t *testing.T) {
	db := setupAuthMethodTestDB(t)
	h := NewRoleHandler(db)
	mgmt := "mciam-t050-mgmt"
	cfg := seedAwsManagementConfig(t, db, "t050-guard", mgmt)

	// 기존 설치의 부트스트랩 9-3·9-4 산출물 재현: 관리용 Role CspRole 레코드와 그 매핑
	legacy := &model.CspRole{Name: mgmt, CspType: "aws", Status: "created", IamIdentifier: cfg.Config["role_arn"], IdpIdentifier: "arn:aws:iam::111111111111:oidc-provider/x", CspIdpConfigID: &cfg.ID}
	require.NoError(t, db.Create(legacy).Error)
	master := &model.RoleMaster{Name: "t050-viewer"}
	require.NoError(t, db.Create(master).Error)
	require.NoError(t, db.Create(&model.RoleSub{RoleID: master.ID, RoleType: constants.RoleTypeCSP}).Error)
	require.NoError(t, db.Create(&model.RoleMasterCspRoleMapping{RoleID: master.ID, CspRoleID: legacy.ID, AuthMethod: constants.AuthMethodOIDC}).Error)

	// 구 매핑 API: 관리용 Role 로 새 매핑 → 409
	other := &model.RoleMaster{Name: "t050-operator"}
	require.NoError(t, db.Create(other).Error)
	rec := guardCall(t, http.MethodPost, "/api/roles/csp-roles",
		fmt.Sprintf(`{"roleId":"%d","cspRoleId":"%d","authMethod":"OIDC","cspType":"aws"}`, other.ID, legacy.ID), nil, h.AddCspRoleMappings)
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	assert.Equal(t, int64(0), countMappings(t, db, other.ID))

	// 등록 → 409
	rec = guardCall(t, http.MethodPost, "/api/roles/csp", fmt.Sprintf(`{"cspRoleName":%q,"cspType":"aws"}`, mgmt), nil, h.CreateCspRole)
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	rec = guardCall(t, http.MethodPost, "/api/roles/csp/batch", fmt.Sprintf(`{"cspRoles":[{"cspRoleName":%q,"cspType":"aws"}]}`, mgmt), nil, h.CreateCspRoles)
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())

	// 수정·삭제 → 409, 레코드·매핑 그대로
	id := map[string]string{"roleId": fmt.Sprint(legacy.ID)}
	rec = guardCall(t, http.MethodPut, "/api/roles/csp/id/x", `{"description":"x"}`, id, h.UpdateCspRole)
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	rec = guardCall(t, http.MethodDelete, "/api/roles/csp/id/x", ``, id, h.DeleteCspRole)
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	assert.Equal(t, int64(1), countMappings(t, db, master.ID), "409 이면 매핑도 지우지 않는다")

	// 매핑 해제(DELETE /roles/csp-roles)는 허용 — 기존 설치 정리 경로
	rec = guardCall(t, http.MethodDelete, "/api/roles/csp-roles",
		fmt.Sprintf(`{"roleId":"%d","cspRoleId":"%d","authMethod":"OIDC"}`, master.ID, legacy.ID), nil, h.RemoveCspRoleMappings)
	assert.Less(t, rec.Code, 300, rec.Body.String())
	assert.Zero(t, countMappings(t, db, master.ID))
}
