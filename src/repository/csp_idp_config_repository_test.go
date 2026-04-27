package repository

import (
	"testing"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupIdpTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.CspAccount{}, &model.CspIdpConfig{}))
	return db
}

func seedCspAccount(t *testing.T, db *gorm.DB, name, cspType string) *model.CspAccount {
	t.Helper()
	acct := &model.CspAccount{Name: name, CspType: cspType, IsActive: true}
	require.NoError(t, db.Create(acct).Error)
	return acct
}

func seedCspIdpConfig(t *testing.T, db *gorm.DB, name string, accountID uint, authMethod model.AuthMethodType, isActive bool) *model.CspIdpConfig {
	t.Helper()
	cfg := &model.CspIdpConfig{
		Name:         name,
		CspAccountID: accountID,
		AuthMethod:   authMethod,
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

// TestGetSummary_NoAccounts CSP 계정 없을 때 빈 배열 반환
func TestGetSummary_NoAccounts(t *testing.T) {
	db := setupIdpTestDB(t)
	repo := NewCspIdpConfigRepository(db)

	rows, err := repo.GetSummary()
	require.NoError(t, err)
	assert.Empty(t, rows)
}

// TestGetSummary_AccountWithNoIdpConfigs IDP 설정이 없는 계정도 포함 (LEFT JOIN)
func TestGetSummary_AccountWithNoIdpConfigs(t *testing.T) {
	db := setupIdpTestDB(t)
	repo := NewCspIdpConfigRepository(db)

	seedCspAccount(t, db, "aws-prod", "aws")

	rows, err := repo.GetSummary()
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "aws-prod", rows[0].CspAccountName)
	assert.Equal(t, "aws", rows[0].CspType)
	assert.Equal(t, 0, rows[0].TotalCount)
	assert.Equal(t, 0, rows[0].ActiveCount)
	assert.Equal(t, 0, rows[0].OidcCount)
	assert.Equal(t, 0, rows[0].SamlCount)
	assert.Equal(t, 0, rows[0].SecretKeyCount)
}

// TestGetSummary_MethodCounts 인증 방식별 카운트 검증
func TestGetSummary_MethodCounts(t *testing.T) {
	db := setupIdpTestDB(t)
	repo := NewCspIdpConfigRepository(db)

	acct := seedCspAccount(t, db, "aws-prod", "aws")
	seedCspIdpConfig(t, db, "oidc-1", acct.ID, model.AuthMethodOIDC, true)
	seedCspIdpConfig(t, db, "saml-1", acct.ID, model.AuthMethodSAML, true)
	seedCspIdpConfig(t, db, "saml-2", acct.ID, model.AuthMethodSAML, false) // inactive
	seedCspIdpConfig(t, db, "secret-1", acct.ID, model.AuthMethodSecretKey, true)

	rows, err := repo.GetSummary()
	require.NoError(t, err)
	require.Len(t, rows, 1)

	row := rows[0]
	assert.Equal(t, 4, row.TotalCount)
	assert.Equal(t, 3, row.ActiveCount) // saml-2 제외
	assert.Equal(t, 1, row.OidcCount)
	assert.Equal(t, 2, row.SamlCount)
	assert.Equal(t, 1, row.SecretKeyCount)
}

// TestGetSummary_MultipleAccounts 복수 계정 각각 집계
func TestGetSummary_MultipleAccounts(t *testing.T) {
	db := setupIdpTestDB(t)
	repo := NewCspIdpConfigRepository(db)

	aws := seedCspAccount(t, db, "aws-prod", "aws")
	gcp := seedCspAccount(t, db, "gcp-dev", "gcp")

	seedCspIdpConfig(t, db, "aws-oidc", aws.ID, model.AuthMethodOIDC, true)
	seedCspIdpConfig(t, db, "gcp-saml", gcp.ID, model.AuthMethodSAML, true)
	seedCspIdpConfig(t, db, "gcp-secret", gcp.ID, model.AuthMethodSecretKey, false)

	rows, err := repo.GetSummary()
	require.NoError(t, err)
	require.Len(t, rows, 2)

	// ORDER BY c.id → aws가 먼저
	assert.Equal(t, "aws-prod", rows[0].CspAccountName)
	assert.Equal(t, 1, rows[0].TotalCount)
	assert.Equal(t, 1, rows[0].OidcCount)

	assert.Equal(t, "gcp-dev", rows[1].CspAccountName)
	assert.Equal(t, 2, rows[1].TotalCount)
	assert.Equal(t, 1, rows[1].ActiveCount) // gcp-secret inactive
	assert.Equal(t, 1, rows[1].SamlCount)
	assert.Equal(t, 1, rows[1].SecretKeyCount)
}
