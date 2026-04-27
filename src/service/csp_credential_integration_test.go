package service

// csp_credential_integration_test.go
//
// CSP별 임시자격증명 발급 실제 연동 통합 테스트.
// 실행 방법:
//   INTEGRATION_TEST=1 go test github.com/m-cmp/mc-iam-manager/service -run "TestIntegrationCred" -v -count=1
//
// 필수 공통 환경변수 (.env 로드 후):
//   DB:  MC_IAM_MANAGER_DATABASE_* (DB 접속 정보)
//   KC:  MC_IAM_MANAGER_KEYCLOAK_HOST, MC_IAM_MANAGER_KEYCLOAK_REALM
//        MC_IAM_MANAGER_KEYCLOAK_OIDC_CLIENT_NAME, MC_IAM_MANAGER_KEYCLOAK_OIDC_CLIENT_SECRET
//
// ARN 정보는 DB의 mcmp_role_csp_roles 테이블에서 자동 조회.
// (csp_type + auth_method에 맞는 CspRole이 DB에 없으면 해당 테스트를 자동 skip)

import (
	"context"
	"os"
	"testing"

	"github.com/m-cmp/mc-iam-manager/config"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// initGormDB 통합 테스트용 실제 PostgreSQL 연결 (실패 시 skip)
func initGormDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbConfig := config.NewDatabaseConfig()
	db, err := gorm.Open(postgres.Open(dbConfig.GetDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("DB 연결 실패 — DB 미연결로 테스트를 건너뜁니다: %v", err)
	}
	return db
}

// findCspRole DB에서 csp_type + auth_method에 맞는 첫 번째 활성 CspRole 반환 (없으면 skip)
func findCspRole(t *testing.T, db *gorm.DB, cspType string, authMethod model.AuthMethodType) *model.CspRole {
	t.Helper()
	var roles []model.CspRole
	err := db.Preload("CspIdpConfig").
		Joins("JOIN mcmp_csp_idp_configs ON mcmp_csp_idp_configs.id = mcmp_role_csp_roles.csp_idp_config_id").
		Where("mcmp_role_csp_roles.csp_type = ? AND mcmp_csp_idp_configs.auth_method = ? AND mcmp_role_csp_roles.deleted_at IS NULL",
			cspType, string(authMethod)).
		Find(&roles).Error
	if err != nil || len(roles) == 0 {
		t.Skipf("DB에 %s/%s CspRole 없음 — 건너뜁니다", cspType, authMethod)
	}
	r := roles[0]
	if r.IdpIdentifier == "" || r.IamIdentifier == "" {
		t.Skipf("CspRole(id=%d) IdpIdentifier 또는 IamIdentifier 미설정 — 건너뜁니다", r.ID)
	}
	return &r
}

// initKCForCred Keycloak 초기화 (TEST_PLATFORMADMIN_ID/PASSWORD 환경 주입 포함)
func initKCForCred(t *testing.T) {
	t.Helper()
	if id := os.Getenv("TEST_PLATFORMADMIN_ID"); id != "" && id != "notyet" {
		t.Setenv("MC_IAM_MANAGER_PLATFORMADMIN_ID", id)
	}
	if pw := os.Getenv("TEST_PLATFORMADMIN_PASSWORD"); pw != "" && pw != "notyet" {
		t.Setenv("MC_IAM_MANAGER_PLATFORMADMIN_PASSWORD", pw)
	}
	if config.KC != nil {
		return
	}
	if err := config.InitKeycloak(); err != nil {
		t.Skipf("Keycloak 초기화 실패: %v", err)
	}
}

// ── AWS OIDC ──────────────────────────────────────────────────────────────────

// TestIntegrationCred_AWS_OIDC DB에서 AWS OIDC CspRole 조회 → AssumeRoleWithWebIdentity
func TestIntegrationCred_AWS_OIDC(t *testing.T) {
	skipIfNotIntegration(t)
	initKCForCred(t)
	db := initGormDB(t)
	cspRole := findCspRole(t, db, "aws", model.AuthMethodOIDC)

	providerArn := cspRole.IdpIdentifier
	roleArn := cspRole.IamIdentifier
	ctx := context.Background()

	kcSvc := NewKeycloakService()
	jwt, err := kcSvc.GetImpersonationTokenByServiceAccount(ctx)
	require.NoError(t, err, "Keycloak OIDC 토큰 발급 실패")
	require.NotEmpty(t, jwt.AccessToken)
	t.Logf("KC OIDC token len=%d", len(jwt.AccessToken))

	awsSvc := NewAwsCredentialService()
	cred, err := awsSvc.AssumeRoleWithWebIdentity(ctx, roleArn, "it-test", jwt.AccessToken, providerArn, "ap-northeast-2")
	require.NoError(t, err, "AWS AssumeRoleWithWebIdentity 실패")

	assert.NotEmpty(t, cred.AccessKeyId)
	assert.NotEmpty(t, cred.SecretAccessKey)
	assert.NotEmpty(t, cred.SessionToken)
	assert.Equal(t, "aws", cred.CspType)
	t.Logf("AWS OIDC cred: AccessKeyId=%s Expiration=%s (CspRole id=%d, providerArn=%s)",
		cred.AccessKeyId, cred.Expiration, cspRole.ID, providerArn)
}

// ── AWS SAML ──────────────────────────────────────────────────────────────────

// TestIntegrationCred_AWS_SAML DB에서 AWS SAML CspRole 조회 → AssumeRoleWithSAML
func TestIntegrationCred_AWS_SAML(t *testing.T) {
	skipIfNotIntegration(t)
	initKCForCred(t)
	db := initGormDB(t)
	cspRole := findCspRole(t, db, "aws", model.AuthMethodSAML)

	providerArn := cspRole.IdpIdentifier
	roleArn := cspRole.IamIdentifier
	samlClientID := envOrDefault("TEST_KC_SAML_CLIENT_ID", "urn:amazon:webservices")
	if v, ok := cspRole.ExtendedConfig["saml_client_id"].(string); ok && v != "" {
		samlClientID = v
	}
	ctx := context.Background()

	kcSvc := NewKeycloakService()
	assertion, err := kcSvc.GetSamlAssertionByServiceAccount(ctx, samlClientID)
	require.NoError(t, err, "Keycloak SAML assertion 발급 실패")
	require.NotEmpty(t, assertion)
	t.Logf("KC SAML assertion len=%d", len(assertion))

	awsSvc := NewAwsCredentialService()
	cred, err := awsSvc.AssumeRoleWithSAML(ctx, roleArn, providerArn, assertion, "ap-northeast-2")
	require.NoError(t, err, "AWS AssumeRoleWithSAML 실패")

	assert.NotEmpty(t, cred.AccessKeyId)
	assert.NotEmpty(t, cred.SecretAccessKey)
	assert.NotEmpty(t, cred.SessionToken)
	assert.Equal(t, "aws", cred.CspType)
	t.Logf("AWS SAML cred: AccessKeyId=%s Expiration=%s (CspRole id=%d)", cred.AccessKeyId, cred.Expiration, cspRole.ID)
}

// ── GCP SAML ─────────────────────────────────────────────────────────────────

// TestIntegrationCred_GCP_SAML DB에서 GCP SAML CspRole 조회 → ExchangeTokenAndImpersonate
func TestIntegrationCred_GCP_SAML(t *testing.T) {
	skipIfNotIntegration(t)
	initKCForCred(t)
	db := initGormDB(t)
	cspRole := findCspRole(t, db, "gcp", model.AuthMethodSAML)

	wifProvider := cspRole.IdpIdentifier
	saEmail := cspRole.IamIdentifier
	samlClientID := wifProvider
	if v, ok := cspRole.ExtendedConfig["saml_client_id"].(string); ok && v != "" {
		samlClientID = v
	}
	ctx := context.Background()

	kcSvc := NewKeycloakService()
	assertion, err := kcSvc.GetSamlAssertionByServiceAccount(ctx, samlClientID)
	require.NoError(t, err, "Keycloak SAML assertion 발급 실패")
	require.NotEmpty(t, assertion)
	t.Logf("KC SAML assertion len=%d", len(assertion))

	gcpSvc := NewGcpCredentialService()
	cred, err := gcpSvc.ExchangeTokenAndImpersonate(ctx, wifProvider, saEmail, assertion, "saml2")
	require.NoError(t, err, "GCP WIF SAML 토큰 교환 실패")

	assert.NotEmpty(t, cred.AccessToken)
	assert.Equal(t, "gcp", cred.CspType)
	t.Logf("GCP SAML cred: TokenType=%s Expiration=%s (CspRole id=%d)", cred.TokenType, cred.Expiration, cspRole.ID)
}

// ── Alibaba SAML ──────────────────────────────────────────────────────────────

// TestIntegrationCred_Alibaba_SAML DB에서 Alibaba SAML CspRole 조회 → AssumeRoleWithSAML
func TestIntegrationCred_Alibaba_SAML(t *testing.T) {
	skipIfNotIntegration(t)
	initKCForCred(t)
	db := initGormDB(t)
	cspRole := findCspRole(t, db, "alibaba", model.AuthMethodSAML)

	providerArn := cspRole.IdpIdentifier
	roleArn := cspRole.IamIdentifier
	samlClientID := envOrDefault("TEST_KC_ALIBABA_SAML_CLIENT_ID", "urn:alibaba:cloudcomputing")
	if v, ok := cspRole.ExtendedConfig["saml_client_id"].(string); ok && v != "" {
		samlClientID = v
	}
	ctx := context.Background()

	kcSvc := NewKeycloakService()
	assertion, err := kcSvc.GetSamlAssertionByServiceAccount(ctx, samlClientID)
	require.NoError(t, err, "Keycloak SAML assertion 발급 실패")
	require.NotEmpty(t, assertion)
	t.Logf("KC SAML assertion len=%d", len(assertion))

	alibabaSvc := NewAlibabaCredentialService()
	cred, err := alibabaSvc.AssumeRoleWithSAML(ctx, providerArn, roleArn, assertion, "ap-northeast-2")
	require.NoError(t, err, "Alibaba AssumeRoleWithSAML 실패")

	assert.NotEmpty(t, cred.AccessKeyId)
	assert.NotEmpty(t, cred.AccessKeySecret)
	assert.NotEmpty(t, cred.SecurityToken)
	assert.Equal(t, "alibaba", cred.CspType)
	t.Logf("Alibaba SAML cred: AccessKeyId=%s Expiration=%s (CspRole id=%d, providerArn=%s)",
		cred.AccessKeyId, cred.Expiration, cspRole.ID, providerArn)
}

// ── Alibaba OIDC ──────────────────────────────────────────────────────────────

// TestIntegrationCred_Alibaba_OIDC DB에서 Alibaba OIDC CspRole 조회 → AssumeRoleWithOIDC
func TestIntegrationCred_Alibaba_OIDC(t *testing.T) {
	skipIfNotIntegration(t)
	initKCForCred(t)
	db := initGormDB(t)
	cspRole := findCspRole(t, db, "alibaba", model.AuthMethodOIDC)

	providerArn := cspRole.IdpIdentifier
	roleArn := cspRole.IamIdentifier
	ctx := context.Background()

	kcSvc := NewKeycloakService()
	jwt, err := kcSvc.GetImpersonationTokenByServiceAccount(ctx)
	require.NoError(t, err, "Keycloak OIDC 토큰 발급 실패")
	require.NotEmpty(t, jwt.AccessToken)
	t.Logf("KC OIDC access_token len=%d, id_token len=%d", len(jwt.AccessToken), len(jwt.IDToken))

	// Alibaba STS는 ID 토큰(aud=단일값) 요구
	oidcToken := jwt.IDToken
	if oidcToken == "" {
		oidcToken = jwt.AccessToken
	}

	audience := ""
	if cspRole.CspIdpConfig != nil {
		audience = cspRole.CspIdpConfig.Config["audience"]
	}
	t.Logf("Alibaba OIDC audience=%s", audience)

	alibabaSvc := NewAlibabaCredentialService()
	cred, err := alibabaSvc.AssumeRoleWithOIDC(ctx, providerArn, roleArn, oidcToken, "ap-northeast-2", audience)
	require.NoError(t, err, "Alibaba AssumeRoleWithOIDC 실패")

	assert.NotEmpty(t, cred.AccessKeyId)
	assert.NotEmpty(t, cred.AccessKeySecret)
	assert.NotEmpty(t, cred.SecurityToken)
	assert.Equal(t, "alibaba", cred.CspType)
	t.Logf("Alibaba OIDC cred: AccessKeyId=%s Expiration=%s (CspRole id=%d, providerArn=%s)",
		cred.AccessKeyId, cred.Expiration, cspRole.ID, providerArn)
}

// ── Azure OIDC ────────────────────────────────────────────────────────────────

// TestIntegrationCred_Azure_OIDC DB에서 Azure OIDC CspRole 조회 → GetTokenByFederatedCredential
func TestIntegrationCred_Azure_OIDC(t *testing.T) {
	skipIfNotIntegration(t)
	initKCForCred(t)
	db := initGormDB(t)
	cspRole := findCspRole(t, db, "azure", model.AuthMethodOIDC)

	if cspRole.CspIdpConfig == nil {
		t.Skip("Azure CspRole에 CspIdpConfig 없음 — 건너뜁니다")
	}
	tenantID := cspRole.CspIdpConfig.Config["tenant_id"]
	clientID := cspRole.CspIdpConfig.Config["client_id"]
	if tenantID == "" || clientID == "" {
		t.Skipf("CspIdpConfig(id=%d) tenant_id 또는 client_id 미설정 — 건너뜁니다", cspRole.CspIdpConfig.ID)
	}
	ctx := context.Background()

	kcSvc := NewKeycloakService()
	jwt, err := kcSvc.GetImpersonationTokenByServiceAccount(ctx)
	require.NoError(t, err, "Keycloak OIDC 토큰 발급 실패")
	require.NotEmpty(t, jwt.AccessToken)
	t.Logf("KC OIDC token len=%d", len(jwt.AccessToken))

	azureSvc := NewAzureCredentialService()
	cred, err := azureSvc.GetTokenByFederatedCredential(ctx, tenantID, clientID, jwt.AccessToken)
	require.NoError(t, err, "Azure GetTokenByFederatedCredential 실패")

	assert.NotEmpty(t, cred.AccessToken)
	assert.Equal(t, "azure", cred.CspType)
	t.Logf("Azure OIDC cred: TokenType=%s Expiration=%s (CspRole id=%d)", cred.TokenType, cred.Expiration, cspRole.ID)
}

// ── Tencent SAML ──────────────────────────────────────────────────────────────

// TestIntegrationCred_Tencent_SAML DB에서 Tencent SAML CspRole 조회 → AssumeRoleWithSAML
func TestIntegrationCred_Tencent_SAML(t *testing.T) {
	skipIfNotIntegration(t)
	initKCForCred(t)
	db := initGormDB(t)
	cspRole := findCspRole(t, db, "tencent", model.AuthMethodSAML)

	if cspRole.CspIdpConfig == nil {
		t.Skip("Tencent CspRole에 CspIdpConfig 없음 — 건너뜁니다")
	}
	secretID := cspRole.CspIdpConfig.Config["secret_id"]
	secretKey := cspRole.CspIdpConfig.Config["secret_key"]
	if secretID == "" || secretKey == "" {
		t.Skipf("CspIdpConfig(id=%d) secret_id 또는 secret_key 미설정 — 건너뜁니다", cspRole.CspIdpConfig.ID)
	}
	roleArn := cspRole.IamIdentifier
	principalArn := cspRole.IdpIdentifier
	samlClientID := principalArn
	if v, ok := cspRole.ExtendedConfig["saml_client_id"].(string); ok && v != "" {
		samlClientID = v
	}
	ctx := context.Background()

	kcSvc := NewKeycloakService()
	assertion, err := kcSvc.GetSamlAssertionByServiceAccount(ctx, samlClientID)
	require.NoError(t, err, "Keycloak SAML assertion 발급 실패")
	require.NotEmpty(t, assertion)
	t.Logf("KC SAML assertion len=%d", len(assertion))

	tencentSvc := NewTencentCredentialService()
	cred, err := tencentSvc.AssumeRoleWithSAML(ctx, secretID, secretKey, roleArn, principalArn, assertion, "ap-guangzhou")
	require.NoError(t, err, "Tencent AssumeRoleWithSAML 실패")

	assert.NotEmpty(t, cred.AccessKeyId)
	assert.NotEmpty(t, cred.SecretAccessKey)
	assert.NotEmpty(t, cred.SessionToken)
	assert.Equal(t, "tencent", cred.CspType)
	t.Logf("Tencent SAML cred: AccessKeyId=%s Expiration=%s (CspRole id=%d)", cred.AccessKeyId, cred.Expiration, cspRole.ID)
}

// ── IBM OIDC ──────────────────────────────────────────────────────────────────

// TestIntegrationCred_IBM_OIDC DB에서 IBM OIDC CspRole 조회 → GetTokenByTrustedProfile
func TestIntegrationCred_IBM_OIDC(t *testing.T) {
	skipIfNotIntegration(t)
	initKCForCred(t)
	db := initGormDB(t)
	cspRole := findCspRole(t, db, "ibm", model.AuthMethodOIDC)

	if cspRole.CspIdpConfig == nil {
		t.Skip("IBM CspRole에 CspIdpConfig 없음 — 건너뜁니다")
	}
	profileID := cspRole.CspIdpConfig.Config["profile_id"]
	if profileID == "" {
		t.Skipf("CspIdpConfig(id=%d) profile_id 미설정 — 건너뜁니다", cspRole.CspIdpConfig.ID)
	}
	ctx := context.Background()

	kcSvc := NewKeycloakService()
	jwt, err := kcSvc.GetImpersonationTokenByServiceAccount(ctx)
	require.NoError(t, err, "Keycloak OIDC 토큰 발급 실패")
	require.NotEmpty(t, jwt.AccessToken)
	t.Logf("KC OIDC token len=%d", len(jwt.AccessToken))

	ibmSvc := NewIbmCredentialService()
	cred, err := ibmSvc.GetTokenByTrustedProfile(ctx, profileID, jwt.AccessToken)
	require.NoError(t, err, "IBM GetTokenByTrustedProfile 실패")

	assert.NotEmpty(t, cred.AccessToken)
	assert.Equal(t, "ibm", cred.CspType)
	t.Logf("IBM OIDC cred: TokenType=%s Expiration=%s (CspRole id=%d)", cred.TokenType, cred.Expiration, cspRole.ID)
}
