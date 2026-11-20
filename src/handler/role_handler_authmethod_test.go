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

type authMethodTestValidator struct{ v *validator.Validate }

func (tv *authMethodTestValidator) Validate(i interface{}) error { return tv.v.Struct(i) }

func setupAuthMethodTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// NewRoleHandler가 구성하는 MenuRepository 등 다른 리포지토리가 자체적으로
	// AutoMigrate를 호출하며 새 커넥션을 열 수 있어 shared cache로 같은 DB를 공유한다.
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&model.RoleMaster{},
		&model.RoleSub{},
		&model.CspRole{},
		&model.RoleMasterCspRoleMapping{},
		&model.CspAccount{},
		&model.CspIdpConfig{},
	))
	return db
}

// seedSamlIdpConfig 매핑 검증(소속 Config 필수)을 통과하도록 CSP 계정과 SAML IdP Config 를 만든다.
func seedSamlIdpConfig(t *testing.T, db *gorm.DB, name, cspType string) *model.CspIdpConfig {
	t.Helper()
	account := &model.CspAccount{Name: name + "-account", CspType: cspType}
	require.NoError(t, db.Create(account).Error)
	cfg := &model.CspIdpConfig{Name: name, CspAccountID: account.ID, AuthMethod: model.AuthMethodSAML, IsActive: true, Config: map[string]string{}}
	require.NoError(t, db.Create(cfg).Error)
	return cfg
}

// samlCspRoleJSON 매핑 검증을 통과하는 SAML cspRoles 항목
func samlCspRoleJSON(name string, cfgID uint) string {
	return `{"cspRoleName":"` + name + `","cspType":"gcp","authMethod":"SAML","cspIdpConfigId":` + strconv.FormatUint(uint64(cfgID), 10) +
		`,"idpIdentifier":"//iam.googleapis.com/test","iamIdentifier":"` + name + `@test.iam.gserviceaccount.com"}`
}

func authMethodTestEcho() *echo.Echo {
	e := echo.New()
	e.Validator = &authMethodTestValidator{v: validator.New()}
	return e
}

// TC-IAM-BUG-020-05: CreateRole이 cspRoles에 지정된 authMethod(SAML)를
// RoleMasterCspRoleMapping에 그대로 반영해야 한다 — 수정 전에는 이 값이
// createdCspRoles 재구성 과정에서 누락되어 항상 OIDC로 기록됐다.
func TestCreateRole_CspRoleWithSAMLAuthMethod_MappingPreservesAuthMethod(t *testing.T) {
	db := setupAuthMethodTestDB(t)
	h := NewRoleHandler(db)

	cfg := seedSamlIdpConfig(t, db, "tc-020-create-saml", "gcp")
	e := authMethodTestEcho()
	body := `{"name":"tc-iam-bug-020-create","roleTypes":["csp"],"cspRoles":[` + samlCspRoleJSON("tc-role-saml", cfg.ID) + `]}`
	req := httptest.NewRequest(http.MethodPost, "/api/roles", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	require.NoError(t, h.CreateRole(c))
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var role model.RoleMaster
	require.NoError(t, db.Where("name = ?", "tc-iam-bug-020-create").First(&role).Error)

	var mapping model.RoleMasterCspRoleMapping
	require.NoError(t, db.Where("role_id = ?", role.ID).First(&mapping).Error)
	assert.Equal(t, constants.AuthMethodSAML, mapping.AuthMethod, "SAML로 생성한 CSP 역할의 매핑은 SAML로 기록되어야 한다")
}

// TC-IAM-BUG-020-06: UpdateRole이 cspRoles에 지정된 authMethod(SAML)를
// 그대로 반영해야 한다 — 수정 전에는 항상 constants.AuthMethodOIDC로 고정되어,
// 저장할 때마다 SAML 매핑이 OIDC로 덮어써졌다.
func TestUpdateRole_CspRoleWithSAMLAuthMethod_MappingPreservesAuthMethod(t *testing.T) {
	db := setupAuthMethodTestDB(t)
	h := NewRoleHandler(db)

	role := &model.RoleMaster{Name: "tc-iam-bug-020-update"}
	require.NoError(t, db.Create(role).Error)

	cfg := seedSamlIdpConfig(t, db, "tc-020-update-saml", "gcp")
	e := authMethodTestEcho()
	body := `{"name":"tc-iam-bug-020-update","roleTypes":["csp"],"cspRoles":[` + samlCspRoleJSON("tc-role-saml-update", cfg.ID) + `]}`
	req := httptest.NewRequest(http.MethodPut, "/api/roles/id/"+strconv.FormatUint(uint64(role.ID), 10), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("roleId")
	c.SetParamValues(strconv.FormatUint(uint64(role.ID), 10))

	require.NoError(t, h.UpdateRole(c))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var mapping model.RoleMasterCspRoleMapping
	require.NoError(t, db.Where("role_id = ?", role.ID).First(&mapping).Error)
	assert.Equal(t, constants.AuthMethodSAML, mapping.AuthMethod, "SAML로 지정한 CSP 역할 매핑은 SAML로 저장되어야 한다(수정 전 버그는 OIDC로 덮어씀)")
}
