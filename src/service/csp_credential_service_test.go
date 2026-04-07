package service

import (
	"context"
	"testing"

	"github.com/Nerzal/gocloak/v13"
	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── 헬퍼 ─────────────────────────────────────────────────────────────────────

func stdUserRole() *model.UserWorkspaceRole {
	return &model.UserWorkspaceRole{RoleID: 1}
}

func oidcKC() *mockKeycloakForCred {
	tok := &gocloak.JWT{AccessToken: "kc_access_token"}
	return &mockKeycloakForCred{oidcToken: tok}
}

func samlKC() *mockKeycloakForCred {
	return &mockKeycloakForCred{samlAssertion: "base64_saml_assertion"}
}

func failOidcKC() *mockKeycloakForCred {
	return &mockKeycloakForCred{oidcErr: errKeycloakFail}
}

func failSamlKC() *mockKeycloakForCred {
	return &mockKeycloakForCred{samlErr: errKeycloakFail}
}

func req(cspType, authMethod string) *model.CspCredentialRequest {
	return &model.CspCredentialRequest{
		WorkspaceID: "1",
		CspType:     cspType,
		Region:      "ap-northeast-2",
		AuthMethod:  authMethod,
	}
}

const (
	idpArn  = "arn:aws:iam::123456789012:saml-provider/keycloak"
	roleArn = "arn:aws:iam::123456789012:role/mciam-test"
)

// ── AWS OIDC ─────────────────────────────────────────────────────────────────

// TC-CRED-01: AWS OIDC — 정상 발급
func TestGetTemporaryCredentials_AWS_OIDC_Success(t *testing.T) {
	aws := &mockAwsCredService{oidcResult: awsOidcCred}
	svc := newCredServiceWithMocks(credServiceDeps{
		aws:      aws,
		gcp:      &mockGcpCredService{},
		alibaba:  &mockAlibabaCredService{},
		kc:       oidcKC(),
		userRepo: &mockUserRepoForCred{role: stdUserRole()},
		mapRepo:  &mockCspMappingRepo{mapping: buildMapping(constants.AuthMethodOIDC, idpArn, roleArn, model.AuthMethodOIDC, nil)},
	})

	cred, err := svc.GetTemporaryCredentials(context.Background(), 1, "kc_user_id", req("aws", "OIDC"))
	require.NoError(t, err)
	assert.Equal(t, "ASIA_OIDC", cred.AccessKeyId)
	assert.Equal(t, "aws", cred.CspType)
}

// TC-CRED-02: AWS OIDC — authMethod 미지정 시 OIDC 기본값 동작
func TestGetTemporaryCredentials_AWS_OIDC_DefaultFallback(t *testing.T) {
	aws := &mockAwsCredService{oidcResult: awsOidcCred}
	svc := newCredServiceWithMocks(credServiceDeps{
		aws:      aws,
		gcp:      &mockGcpCredService{},
		alibaba:  &mockAlibabaCredService{},
		kc:       oidcKC(),
		userRepo: &mockUserRepoForCred{role: stdUserRole()},
		// CspIdpConfig.AuthMethod = "" → 기본값 OIDC 적용
		mapRepo: &mockCspMappingRepo{mapping: buildMapping(constants.AuthMethodOIDC, idpArn, roleArn, model.AuthMethodType(""), nil)},
	})

	cred, err := svc.GetTemporaryCredentials(context.Background(), 1, "kc_user_id", req("aws", ""))
	require.NoError(t, err)
	assert.Equal(t, "ASIA_OIDC", cred.AccessKeyId)
}

// TC-CRED-03: AWS OIDC — Keycloak 토큰 획득 실패
func TestGetTemporaryCredentials_AWS_OIDC_KeycloakFail(t *testing.T) {
	svc := newCredServiceWithMocks(credServiceDeps{
		aws:      &mockAwsCredService{},
		gcp:      &mockGcpCredService{},
		alibaba:  &mockAlibabaCredService{},
		kc:       failOidcKC(),
		userRepo: &mockUserRepoForCred{role: stdUserRole()},
		mapRepo:  &mockCspMappingRepo{mapping: buildMapping(constants.AuthMethodOIDC, idpArn, roleArn, model.AuthMethodOIDC, nil)},
	})

	_, err := svc.GetTemporaryCredentials(context.Background(), 1, "kc_user_id", req("aws", "OIDC"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get impersonation token")
}

// TC-CRED-04: AWS OIDC — STS 호출 실패
func TestGetTemporaryCredentials_AWS_OIDC_STSFail(t *testing.T) {
	svc := newCredServiceWithMocks(credServiceDeps{
		aws:      &mockAwsCredService{oidcErr: errStsFail},
		gcp:      &mockGcpCredService{},
		alibaba:  &mockAlibabaCredService{},
		kc:       oidcKC(),
		userRepo: &mockUserRepoForCred{role: stdUserRole()},
		mapRepo:  &mockCspMappingRepo{mapping: buildMapping(constants.AuthMethodOIDC, idpArn, roleArn, model.AuthMethodOIDC, nil)},
	})

	_, err := svc.GetTemporaryCredentials(context.Background(), 1, "kc_user_id", req("aws", "OIDC"))
	require.Error(t, err)
}

// ── AWS SAML ─────────────────────────────────────────────────────────────────

// TC-CRED-05: AWS SAML — 정상 발급
func TestGetTemporaryCredentials_AWS_SAML_Success(t *testing.T) {
	aws := &mockAwsCredService{samlResult: awsSamlCred}
	svc := newCredServiceWithMocks(credServiceDeps{
		aws:      aws,
		gcp:      &mockGcpCredService{},
		alibaba:  &mockAlibabaCredService{},
		kc:       samlKC(),
		userRepo: &mockUserRepoForCred{role: stdUserRole()},
		mapRepo:  &mockCspMappingRepo{mapping: buildMapping(constants.AuthMethodSAML, idpArn, roleArn, model.AuthMethodSAML, nil)},
	})

	cred, err := svc.GetTemporaryCredentials(context.Background(), 1, "kc_user_id", req("aws", "SAML"))
	require.NoError(t, err)
	assert.Equal(t, "ASIA_SAML", cred.AccessKeyId)
	assert.Equal(t, "aws", cred.CspType)
}

// TC-CRED-06: AWS SAML — saml_client_id ExtendedConfig 재정의
func TestGetTemporaryCredentials_AWS_SAML_CustomAudience(t *testing.T) {
	aws := &mockAwsCredService{samlResult: awsSamlCred}
	kc := &mockKeycloakForCred{samlAssertion: "saml_assertion_custom"}

	mapping := buildMapping(constants.AuthMethodSAML, idpArn, roleArn, model.AuthMethodSAML, nil)
	mapping.CspRoles[0].ExtendedConfig = map[string]interface{}{
		"saml_client_id": "custom-saml-client",
	}

	svc := newCredServiceWithMocks(credServiceDeps{
		aws:      aws,
		gcp:      &mockGcpCredService{},
		alibaba:  &mockAlibabaCredService{},
		kc:       kc,
		userRepo: &mockUserRepoForCred{role: stdUserRole()},
		mapRepo:  &mockCspMappingRepo{mapping: mapping},
	})

	cred, err := svc.GetTemporaryCredentials(context.Background(), 1, "kc_user_id", req("aws", "SAML"))
	require.NoError(t, err)
	assert.Equal(t, "ASIA_SAML", cred.AccessKeyId)
}

// TC-CRED-07: AWS SAML — Keycloak assertion 획득 실패
func TestGetTemporaryCredentials_AWS_SAML_KeycloakFail(t *testing.T) {
	svc := newCredServiceWithMocks(credServiceDeps{
		aws:      &mockAwsCredService{},
		gcp:      &mockGcpCredService{},
		alibaba:  &mockAlibabaCredService{},
		kc:       failSamlKC(),
		userRepo: &mockUserRepoForCred{role: stdUserRole()},
		mapRepo:  &mockCspMappingRepo{mapping: buildMapping(constants.AuthMethodSAML, idpArn, roleArn, model.AuthMethodSAML, nil)},
	})

	_, err := svc.GetTemporaryCredentials(context.Background(), 1, "kc_user_id", req("aws", "SAML"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get SAML assertion for AWS")
}

// ── AWS SECRET_KEY ────────────────────────────────────────────────────────────

// TC-CRED-08: AWS SECRET_KEY — 정적 키 반환
func TestGetTemporaryCredentials_AWS_SecretKey_Success(t *testing.T) {
	config := map[string]string{
		"access_key_id":     "AKIAIOSFODNN7EXAMPLE",
		"secret_access_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}
	svc := newCredServiceWithMocks(credServiceDeps{
		aws:      &mockAwsCredService{},
		gcp:      &mockGcpCredService{},
		alibaba:  &mockAlibabaCredService{},
		kc:       &mockKeycloakForCred{},
		userRepo: &mockUserRepoForCred{role: stdUserRole()},
		mapRepo:  &mockCspMappingRepo{mapping: buildMapping(constants.AuthMethodSecretKey, idpArn, roleArn, model.AuthMethodSecretKey, config)},
	})

	cred, err := svc.GetTemporaryCredentials(context.Background(), 1, "kc_user_id", req("aws", "SECRET_KEY"))
	require.NoError(t, err)
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", cred.AccessKeyId)
	assert.Equal(t, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", cred.SecretAccessKey)
	assert.Empty(t, cred.SessionToken) // 세션 토큰 없음
}

// TC-CRED-09: AWS SECRET_KEY — Config 누락 시 에러
func TestGetTemporaryCredentials_AWS_SecretKey_MissingConfig(t *testing.T) {
	svc := newCredServiceWithMocks(credServiceDeps{
		aws:      &mockAwsCredService{},
		gcp:      &mockGcpCredService{},
		alibaba:  &mockAlibabaCredService{},
		kc:       &mockKeycloakForCred{},
		userRepo: &mockUserRepoForCred{role: stdUserRole()},
		mapRepo:  &mockCspMappingRepo{mapping: buildMapping(constants.AuthMethodSecretKey, idpArn, roleArn, model.AuthMethodSecretKey, map[string]string{})},
	})

	_, err := svc.GetTemporaryCredentials(context.Background(), 1, "kc_user_id", req("aws", "SECRET_KEY"))
	require.Error(t, err)
}

// ── GCP OIDC ─────────────────────────────────────────────────────────────────

// TC-CRED-10: GCP OIDC — 정상 발급
func TestGetTemporaryCredentials_GCP_OIDC_Success(t *testing.T) {
	svc := newCredServiceWithMocks(credServiceDeps{
		aws:      &mockAwsCredService{},
		gcp:      &mockGcpCredService{result: gcpOidcCred},
		alibaba:  &mockAlibabaCredService{},
		kc:       oidcKC(),
		userRepo: &mockUserRepoForCred{role: stdUserRole()},
		mapRepo:  &mockCspMappingRepo{mapping: buildMapping(constants.AuthMethodOIDC, "wif-provider", "sa@project.iam.gserviceaccount.com", model.AuthMethodOIDC, nil)},
	})

	cred, err := svc.GetTemporaryCredentials(context.Background(), 1, "kc_user_id", req("gcp", "OIDC"))
	require.NoError(t, err)
	assert.Equal(t, "gcp_access_token", cred.AccessToken)
	assert.Equal(t, "gcp", cred.CspType)
}

// TC-CRED-11: GCP SECRET_KEY — 정적 키 반환
func TestGetTemporaryCredentials_GCP_SecretKey_Success(t *testing.T) {
	config := map[string]string{
		"access_key_id":     "gcp_key_id",
		"secret_access_key": "gcp_key_secret",
	}
	svc := newCredServiceWithMocks(credServiceDeps{
		aws:      &mockAwsCredService{},
		gcp:      &mockGcpCredService{},
		alibaba:  &mockAlibabaCredService{},
		kc:       &mockKeycloakForCred{},
		userRepo: &mockUserRepoForCred{role: stdUserRole()},
		mapRepo:  &mockCspMappingRepo{mapping: buildMapping(constants.AuthMethodSecretKey, idpArn, roleArn, model.AuthMethodSecretKey, config)},
	})

	cred, err := svc.GetTemporaryCredentials(context.Background(), 1, "kc_user_id", req("gcp", "SECRET_KEY"))
	require.NoError(t, err)
	assert.Equal(t, "gcp_key_id", cred.AccessKeyId)
}

// TC-CRED-12: GCP SAML — 미구현 → ErrUnsupportedAuthMethod
// TC-CRED-11 (updated): GCP SAML — 정상 발급 (Phase 1 구현)
func TestGetTemporaryCredentials_GCP_SAML_Success(t *testing.T) {
	svc := newCredServiceWithMocks(credServiceDeps{
		aws:      &mockAwsCredService{},
		gcp:      &mockGcpCredService{result: gcpSamlCred},
		alibaba:  &mockAlibabaCredService{},
		kc:       samlKC(),
		userRepo: &mockUserRepoForCred{role: stdUserRole()},
		mapRepo:  &mockCspMappingRepo{mapping: buildMapping(constants.AuthMethodSAML, idpArn, roleArn, model.AuthMethodSAML, nil)},
	})

	cred, err := svc.GetTemporaryCredentials(context.Background(), 1, "kc_user_id", req("gcp", "SAML"))
	require.NoError(t, err)
	assert.Equal(t, "gcp_saml_access_token", cred.AccessToken)
	assert.Equal(t, "gcp", cred.CspType)
}

// TC-CRED-11b: GCP SAML — Keycloak SAML assertion 실패 → 에러 전파
func TestGetTemporaryCredentials_GCP_SAML_KeycloakFail(t *testing.T) {
	svc := newCredServiceWithMocks(credServiceDeps{
		aws:      &mockAwsCredService{},
		gcp:      &mockGcpCredService{},
		alibaba:  &mockAlibabaCredService{},
		kc:       failSamlKC(),
		userRepo: &mockUserRepoForCred{role: stdUserRole()},
		mapRepo:  &mockCspMappingRepo{mapping: buildMapping(constants.AuthMethodSAML, idpArn, roleArn, model.AuthMethodSAML, nil)},
	})

	_, err := svc.GetTemporaryCredentials(context.Background(), 1, "kc_user_id", req("gcp", "SAML"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get SAML assertion for GCP")
}

// ── Alibaba SAML ──────────────────────────────────────────────────────────────

// TC-CRED-13: Alibaba SAML — 정상 발급
func TestGetTemporaryCredentials_Alibaba_SAML_Success(t *testing.T) {
	svc := newCredServiceWithMocks(credServiceDeps{
		aws:      &mockAwsCredService{},
		gcp:      &mockGcpCredService{},
		alibaba:  &mockAlibabaCredService{result: alibabaSamlCred},
		kc:       samlKC(),
		userRepo: &mockUserRepoForCred{role: stdUserRole()},
		mapRepo:  &mockCspMappingRepo{mapping: buildMapping(constants.AuthMethodSAML, idpArn, roleArn, model.AuthMethodSAML, nil)},
	})

	cred, err := svc.GetTemporaryCredentials(context.Background(), 1, "kc_user_id", req("alibaba", "SAML"))
	require.NoError(t, err)
	assert.Equal(t, "STS_ALIBABA", cred.AccessKeyId)
	assert.Equal(t, "alibaba", cred.CspType)
}

// TC-CRED-14: Alibaba OIDC — 미구현 → ErrUnsupportedAuthMethod
func TestGetTemporaryCredentials_Alibaba_OIDC_Unsupported(t *testing.T) {
	svc := newCredServiceWithMocks(credServiceDeps{
		aws:      &mockAwsCredService{},
		gcp:      &mockGcpCredService{},
		alibaba:  &mockAlibabaCredService{},
		kc:       &mockKeycloakForCred{},
		userRepo: &mockUserRepoForCred{role: stdUserRole()},
		mapRepo:  &mockCspMappingRepo{mapping: buildMapping(constants.AuthMethodOIDC, idpArn, roleArn, model.AuthMethodOIDC, nil)},
	})

	_, err := svc.GetTemporaryCredentials(context.Background(), 1, "kc_user_id", req("alibaba", "OIDC"))
	require.Error(t, err)
	assert.Equal(t, ErrUnsupportedAuthMethod, err)
}

// ── Azure / 기타 ──────────────────────────────────────────────────────────────

// TC-CRED-15: Azure SECRET_KEY — 정상 반환
func TestGetTemporaryCredentials_Azure_SecretKey_Success(t *testing.T) {
	config := map[string]string{
		"access_key_id":     "azure_client_id",
		"secret_access_key": "azure_client_secret",
	}
	svc := newCredServiceWithMocks(credServiceDeps{
		aws:      &mockAwsCredService{},
		gcp:      &mockGcpCredService{},
		alibaba:  &mockAlibabaCredService{},
		kc:       &mockKeycloakForCred{},
		userRepo: &mockUserRepoForCred{role: stdUserRole()},
		mapRepo:  &mockCspMappingRepo{mapping: buildMapping(constants.AuthMethodSecretKey, idpArn, roleArn, model.AuthMethodSecretKey, config)},
	})

	cred, err := svc.GetTemporaryCredentials(context.Background(), 1, "kc_user_id", req("azure", "SECRET_KEY"))
	require.NoError(t, err)
	assert.Equal(t, "azure_client_id", cred.AccessKeyId)
}

// TC-CRED-16: Azure OIDC — 미구현 → ErrUnsupportedAuthMethod
func TestGetTemporaryCredentials_Azure_OIDC_Unsupported(t *testing.T) {
	svc := newCredServiceWithMocks(credServiceDeps{
		aws:      &mockAwsCredService{},
		gcp:      &mockGcpCredService{},
		alibaba:  &mockAlibabaCredService{},
		kc:       &mockKeycloakForCred{},
		userRepo: &mockUserRepoForCred{role: stdUserRole()},
		mapRepo:  &mockCspMappingRepo{mapping: buildMapping(constants.AuthMethodOIDC, idpArn, roleArn, model.AuthMethodOIDC, nil)},
	})

	_, err := svc.GetTemporaryCredentials(context.Background(), 1, "kc_user_id", req("azure", "OIDC"))
	require.Error(t, err)
	assert.Equal(t, ErrUnsupportedAuthMethod, err)
}

// ── 에러 경로 ─────────────────────────────────────────────────────────────────

// TC-CRED-17: 워크스페이스에 역할 없음
func TestGetTemporaryCredentials_NoWorkspaceRole(t *testing.T) {
	svc := newCredServiceWithMocks(credServiceDeps{
		aws:      &mockAwsCredService{},
		gcp:      &mockGcpCredService{},
		alibaba:  &mockAlibabaCredService{},
		kc:       &mockKeycloakForCred{},
		userRepo: &mockUserRepoForCred{role: nil}, // 역할 없음
		mapRepo:  &mockCspMappingRepo{},
	})

	_, err := svc.GetTemporaryCredentials(context.Background(), 1, "kc_user_id", req("aws", "OIDC"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no role assigned")
}

// TC-CRED-18: CSP 역할 매핑 없음 → ErrNoCspRoleMappingFound
func TestGetTemporaryCredentials_NoCspRoleMapping(t *testing.T) {
	svc := newCredServiceWithMocks(credServiceDeps{
		aws:      &mockAwsCredService{},
		gcp:      &mockGcpCredService{},
		alibaba:  &mockAlibabaCredService{},
		kc:       &mockKeycloakForCred{},
		userRepo: &mockUserRepoForCred{role: stdUserRole()},
		mapRepo:  &mockCspMappingRepo{mapping: nil}, // 매핑 없음
	})

	_, err := svc.GetTemporaryCredentials(context.Background(), 1, "kc_user_id", req("aws", "OIDC"))
	require.Error(t, err)
	assert.Equal(t, ErrNoCspRoleMappingFound, err)
}

// TC-CRED-19: 미지원 CSP 타입 → ErrUnsupportedCspType
func TestGetTemporaryCredentials_UnsupportedCspType(t *testing.T) {
	svc := newCredServiceWithMocks(credServiceDeps{
		aws:      &mockAwsCredService{},
		gcp:      &mockGcpCredService{},
		alibaba:  &mockAlibabaCredService{},
		kc:       &mockKeycloakForCred{},
		userRepo: &mockUserRepoForCred{role: stdUserRole()},
		mapRepo:  &mockCspMappingRepo{mapping: buildMapping(constants.AuthMethodOIDC, idpArn, roleArn, model.AuthMethodOIDC, nil)},
	})

	_, err := svc.GetTemporaryCredentials(context.Background(), 1, "kc_user_id", req("unknown_csp", "OIDC"))
	require.Error(t, err)
	assert.Equal(t, ErrUnsupportedCspType, err)
}
