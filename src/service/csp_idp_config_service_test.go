package service

import (
	"testing"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupIdpServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.CspAccount{}, &model.CspIdpConfig{}))
	return db
}

func newIdpService(db *gorm.DB) *CspIdpConfigService {
	return NewCspIdpConfigService(db, &mockKeycloakService{})
}

func createCspAccount(t *testing.T, db *gorm.DB, name, cspType string) *model.CspAccount {
	t.Helper()
	acct := &model.CspAccount{Name: name, CspType: cspType, IsActive: true}
	require.NoError(t, db.Create(acct).Error)
	return acct
}

func createCspIdpConfig(t *testing.T, db *gorm.DB, name string, accountID uint, method model.AuthMethodType, isActive bool) *model.CspIdpConfig {
	t.Helper()
	cfg := &model.CspIdpConfig{
		Name:         name,
		CspAccountID: accountID,
		AuthMethod:   method,
		IsActive:     true, // 먼저 true로 생성 (GORM zero-value 스킵 방지)
		Config:       map[string]string{"key": "value"},
	}
	require.NoError(t, db.Create(cfg).Error)
	// isActive=false인 경우 명시적으로 업데이트
	if !isActive {
		require.NoError(t, db.Model(cfg).Update("is_active", false).Error)
	}
	return cfg
}

// --- GetIdpSummary 테스트 ---

// TestGetIdpSummary_Empty CSP 계정 없을 때 빈 배열 반환
func TestGetIdpSummary_Empty(t *testing.T) {
	db := setupIdpServiceTestDB(t)
	svc := newIdpService(db)

	summaries, err := svc.GetIdpSummary()
	require.NoError(t, err)
	assert.Empty(t, summaries)
}

// TestGetIdpSummary_MethodCounts method_counts 맵 정확히 생성됨
func TestGetIdpSummary_MethodCounts(t *testing.T) {
	db := setupIdpServiceTestDB(t)
	svc := newIdpService(db)

	acct := createCspAccount(t, db, "aws-prod", "aws")
	createCspIdpConfig(t, db, "oidc-1", acct.ID, model.AuthMethodOIDC, true)
	createCspIdpConfig(t, db, "saml-1", acct.ID, model.AuthMethodSAML, true)
	createCspIdpConfig(t, db, "saml-2", acct.ID, model.AuthMethodSAML, false)
	createCspIdpConfig(t, db, "secret-1", acct.ID, model.AuthMethodSecretKey, true)

	summaries, err := svc.GetIdpSummary()
	require.NoError(t, err)
	require.Len(t, summaries, 1)

	s := summaries[0]
	assert.Equal(t, acct.ID, s.CspAccountID)
	assert.Equal(t, "aws-prod", s.CspAccountName)
	assert.Equal(t, "aws", s.CspType)
	assert.Equal(t, 4, s.TotalCount)
	assert.Equal(t, 3, s.ActiveCount)
	assert.Equal(t, 1, s.MethodCounts["OIDC"])
	assert.Equal(t, 2, s.MethodCounts["SAML"])
	assert.Equal(t, 1, s.MethodCounts["SECRET_KEY"])
}

// TestGetIdpSummary_MultipleAccounts 계정별 독립 집계
func TestGetIdpSummary_MultipleAccounts(t *testing.T) {
	db := setupIdpServiceTestDB(t)
	svc := newIdpService(db)

	aws := createCspAccount(t, db, "aws-prod", "aws")
	gcp := createCspAccount(t, db, "gcp-dev", "gcp")

	createCspIdpConfig(t, db, "aws-oidc", aws.ID, model.AuthMethodOIDC, true)
	createCspIdpConfig(t, db, "gcp-saml", gcp.ID, model.AuthMethodSAML, true)

	summaries, err := svc.GetIdpSummary()
	require.NoError(t, err)
	require.Len(t, summaries, 2)

	assert.Equal(t, 1, summaries[0].TotalCount)
	assert.Equal(t, 1, summaries[1].TotalCount)
}

// TestGetIdpSummary_AccountWithNoConfigs IDP 없는 계정도 포함됨
func TestGetIdpSummary_AccountWithNoConfigs(t *testing.T) {
	db := setupIdpServiceTestDB(t)
	svc := newIdpService(db)

	createCspAccount(t, db, "empty-account", "azure")

	summaries, err := svc.GetIdpSummary()
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	assert.Equal(t, 0, summaries[0].TotalCount)
	assert.Equal(t, 0, summaries[0].MethodCounts["OIDC"])
}

// --- BulkHealthCheck 테스트 ---

// TestBulkHealthCheck_NoActiveConfigs 활성 설정 없을 때 빈 결과
func TestBulkHealthCheck_NoActiveConfigs(t *testing.T) {
	db := setupIdpServiceTestDB(t)
	svc := newIdpService(db)

	acct := createCspAccount(t, db, "aws-prod", "aws")
	createCspIdpConfig(t, db, "inactive-cfg", acct.ID, model.AuthMethodSAML, false)

	resp, err := svc.BulkHealthCheck(&model.BulkHealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, 0, resp.TotalCount)
	assert.Equal(t, 0, resp.ConnectedCount)
	assert.Equal(t, 0, resp.FailedCount)
	assert.Empty(t, resp.Results)
}

// TestBulkHealthCheck_SamlAlwaysFailed SAML은 미구현으로 항상 FAILED
func TestBulkHealthCheck_SamlAlwaysFailed(t *testing.T) {
	db := setupIdpServiceTestDB(t)
	svc := newIdpService(db)

	acct := createCspAccount(t, db, "aws-prod", "aws")
	createCspIdpConfig(t, db, "saml-cfg", acct.ID, model.AuthMethodSAML, true)

	resp, err := svc.BulkHealthCheck(&model.BulkHealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.TotalCount)
	assert.Equal(t, 0, resp.ConnectedCount)
	assert.Equal(t, 1, resp.FailedCount)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, "FAILED", resp.Results[0].Status)
	assert.NotEmpty(t, resp.Results[0].ErrorMsg)
	assert.NotEmpty(t, resp.Results[0].CheckedAt)
}

// TestBulkHealthCheck_MultipleConfigs 복수 설정 병렬 처리 결과 집계
func TestBulkHealthCheck_MultipleConfigs(t *testing.T) {
	db := setupIdpServiceTestDB(t)
	svc := newIdpService(db)

	acct := createCspAccount(t, db, "aws-prod", "aws")
	createCspIdpConfig(t, db, "saml-1", acct.ID, model.AuthMethodSAML, true)
	createCspIdpConfig(t, db, "saml-2", acct.ID, model.AuthMethodSAML, true)
	createCspIdpConfig(t, db, "saml-3", acct.ID, model.AuthMethodSAML, true)

	resp, err := svc.BulkHealthCheck(&model.BulkHealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, 3, resp.TotalCount)
	assert.Equal(t, 0, resp.ConnectedCount)
	assert.Equal(t, 3, resp.FailedCount)
	assert.Len(t, resp.Results, 3)
}

// TestBulkHealthCheck_FilterByCspAccountID csp_account_id 필터 동작 검증
func TestBulkHealthCheck_FilterByCspAccountID(t *testing.T) {
	db := setupIdpServiceTestDB(t)
	svc := newIdpService(db)

	aws := createCspAccount(t, db, "aws-prod", "aws")
	gcp := createCspAccount(t, db, "gcp-dev", "gcp")
	createCspIdpConfig(t, db, "aws-saml", aws.ID, model.AuthMethodSAML, true)
	createCspIdpConfig(t, db, "gcp-saml", gcp.ID, model.AuthMethodSAML, true)

	// aws 계정만 필터
	resp, err := svc.BulkHealthCheck(&model.BulkHealthCheckRequest{CspAccountID: &aws.ID})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.TotalCount)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, "aws-saml", resp.Results[0].ConfigName)
}

// TestBulkHealthCheck_ResultFields 결과 필드 완전성 검증
func TestBulkHealthCheck_ResultFields(t *testing.T) {
	db := setupIdpServiceTestDB(t)
	svc := newIdpService(db)

	acct := createCspAccount(t, db, "aws-prod", "aws")
	cfg := createCspIdpConfig(t, db, "saml-cfg", acct.ID, model.AuthMethodSAML, true)

	resp, err := svc.BulkHealthCheck(&model.BulkHealthCheckRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Results, 1)

	r := resp.Results[0]
	assert.Equal(t, cfg.ID, r.ConfigID)
	assert.Equal(t, "saml-cfg", r.ConfigName)
	assert.Equal(t, "aws", r.CspType)
	assert.Equal(t, string(model.AuthMethodSAML), r.AuthMethod)
	assert.Equal(t, "FAILED", r.Status)
	assert.NotEmpty(t, r.CheckedAt)
}

// TestBulkHealthCheck_NilRequest nil 요청 처리 (전체 대상)
func TestBulkHealthCheck_NilRequest(t *testing.T) {
	db := setupIdpServiceTestDB(t)
	svc := newIdpService(db)

	acct := createCspAccount(t, db, "aws-prod", "aws")
	createCspIdpConfig(t, db, "saml-cfg", acct.ID, model.AuthMethodSAML, true)

	resp, err := svc.BulkHealthCheck(nil)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.TotalCount)
}
