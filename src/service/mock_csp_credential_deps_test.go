package service

import (
	"context"
	"errors"

	"github.com/Nerzal/gocloak/v13"
	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
)

// ── AWS ──────────────────────────────────────────────────────────────────────

type mockAwsCredService struct {
	oidcResult *model.CspCredentialResponse
	oidcErr    error
	samlResult *model.CspCredentialResponse
	samlErr    error
}

func (m *mockAwsCredService) AssumeRoleWithWebIdentity(_ context.Context, roleArn, kcUserId, token, idpArn, region string) (*model.CspCredentialResponse, error) {
	return m.oidcResult, m.oidcErr
}
func (m *mockAwsCredService) AssumeRoleWithSAML(_ context.Context, roleArn, principalArn, samlAssertion, region string) (*model.CspCredentialResponse, error) {
	return m.samlResult, m.samlErr
}
func (m *mockAwsCredService) CheckOIDCProvider(_ context.Context, oidcProviderArn string) (string, error) {
	return "", nil
}
func (m *mockAwsCredService) CheckSAMLProvider(_ context.Context, samlProviderArn string) (string, error) {
	return "", nil
}
func (m *mockAwsCredService) CheckRoleTrust(_ context.Context, roleArn, expectedAction, expectedProviderArn string) (string, error) {
	return "", nil
}
func (m *mockAwsCredService) CheckCallerIdentity(_ context.Context, accessKeyID, secretKey string) (string, error) {
	return "", nil
}

// ── GCP ──────────────────────────────────────────────────────────────────────

type mockGcpCredService struct {
	result *model.CspCredentialResponse
	err    error
}

func (m *mockGcpCredService) ExchangeTokenAndImpersonate(_ context.Context, wif, sa, token, tokenType string) (*model.CspCredentialResponse, error) {
	return m.result, m.err
}

// ── Alibaba ──────────────────────────────────────────────────────────────────

type mockAlibabaCredService struct {
	result    *model.CspCredentialResponse
	err       error
	oidcResult *model.CspCredentialResponse
	oidcErr    error
}

func (m *mockAlibabaCredService) AssumeRoleWithSAML(_ context.Context, samlProviderArn, roleArn, samlAssertion, region string) (*model.CspCredentialResponse, error) {
	return m.result, m.err
}

func (m *mockAlibabaCredService) AssumeRoleWithOIDC(_ context.Context, oidcProviderArn, roleArn, oidcToken, region string) (*model.CspCredentialResponse, error) {
	if m.oidcResult != nil || m.oidcErr != nil {
		return m.oidcResult, m.oidcErr
	}
	return m.result, m.err
}

// ── Azure ─────────────────────────────────────────────────────────────────────

type mockAzureCredService struct {
	result *model.CspCredentialResponse
	err    error
}

func (m *mockAzureCredService) GetTokenByFederatedCredential(_ context.Context, tenantID, clientID, keycloakJWT string) (*model.CspCredentialResponse, error) {
	return m.result, m.err
}

// ── Tencent ───────────────────────────────────────────────────────────────────

type mockTencentCredService struct {
	result *model.CspCredentialResponse
	err    error
}

func (m *mockTencentCredService) AssumeRoleWithSAML(_ context.Context, secretID, secretKey, roleArn, principalArn, samlAssertion, region string) (*model.CspCredentialResponse, error) {
	return m.result, m.err
}

// ── IBM ───────────────────────────────────────────────────────────────────────

type mockIbmCredService struct {
	result *model.CspCredentialResponse
	err    error
}

func (m *mockIbmCredService) GetTokenByTrustedProfile(_ context.Context, profileID, crToken string) (*model.CspCredentialResponse, error) {
	return m.result, m.err
}

// ── UserRepository (필요한 메서드만) ─────────────────────────────────────────

type mockUserRepoForCred struct {
	role    *model.UserWorkspaceRole
	roleErr error
}

func (m *mockUserRepoForCred) FindUserRoleInWorkspace(userID, workspaceID uint) (*model.UserWorkspaceRole, error) {
	return m.role, m.roleErr
}

// ── CspMappingRepository ─────────────────────────────────────────────────────

type mockCspMappingRepo struct {
	mapping    *model.RoleMasterCspRoleMapping
	mappingErr error
}

func (m *mockCspMappingRepo) FindCspRoleMappingsByRoleIDAndCspType(roleID uint, cspType string, authMethod string) (*model.RoleMasterCspRoleMapping, error) {
	return m.mapping, m.mappingErr
}

// ── 헬퍼: 표준 응답값 ─────────────────────────────────────────────────────────

var awsOidcCred = &model.CspCredentialResponse{
	CspType:         "aws",
	AccessKeyId:     "ASIA_OIDC",
	SecretAccessKey: "secret_oidc",
	SessionToken:    "token_oidc",
}

var awsSamlCred = &model.CspCredentialResponse{
	CspType:         "aws",
	AccessKeyId:     "ASIA_SAML",
	SecretAccessKey: "secret_saml",
	SessionToken:    "token_saml",
}

var gcpOidcCred = &model.CspCredentialResponse{
	CspType:     "gcp",
	AccessToken: "gcp_access_token",
	TokenType:   "Bearer",
}

var gcpSamlCred = &model.CspCredentialResponse{
	CspType:     "gcp",
	AccessToken: "gcp_saml_access_token",
	TokenType:   "Bearer",
}

var alibabaSamlCred = &model.CspCredentialResponse{
	CspType:         "alibaba",
	AccessKeyId:     "STS_ALIBABA",
	SecretAccessKey: "alibaba_secret",
	SecurityToken:   "alibaba_token",
}

var alibabaOidcCred = &model.CspCredentialResponse{
	CspType:         "alibaba",
	AccessKeyId:     "STS_ALIBABA_OIDC",
	SecretAccessKey: "alibaba_oidc_secret",
	SecurityToken:   "alibaba_oidc_token",
}

var azureOidcCred = &model.CspCredentialResponse{
	CspType:     "azure",
	AccessToken: "azure_access_token",
	TokenType:   "Bearer",
}

var tencentSamlCred = &model.CspCredentialResponse{
	CspType:         "tencent",
	AccessKeyId:     "STS_TENCENT",
	SecretAccessKey: "tencent_secret",
	SessionToken:    "tencent_token",
}

var ibmOidcCred = &model.CspCredentialResponse{
	CspType:     "ibm",
	AccessToken: "ibm_access_token",
	TokenType:   "Bearer",
}

// ── 헬퍼: CspCredentialService 생성 ──────────────────────────────────────────

type credServiceDeps struct {
	aws      *mockAwsCredService
	gcp      *mockGcpCredService
	alibaba  *mockAlibabaCredService
	azure    *mockAzureCredService
	tencent  *mockTencentCredService
	ibm      *mockIbmCredService
	kc       KeycloakService // 인터페이스 — mockKeycloakService 또는 mockKeycloakForCred 모두 허용
	userRepo *mockUserRepoForCred
	mapRepo  *mockCspMappingRepo
}

func newCredServiceWithMocks(deps credServiceDeps) *CspCredentialService {
	return &CspCredentialService{
		awsCredService:     deps.aws,
		gcpCredService:     deps.gcp,
		alibabaCredService: deps.alibaba,
		azureCredService:   deps.azure,
		tencentCredService: deps.tencent,
		ibmCredService:     deps.ibm,
		keycloakService:    deps.kc,
		userRepoIface:      deps.userRepo,
		mappingRepoIface:   deps.mapRepo,
	}
}

// ── 헬퍼: 표준 매핑 빌더 ─────────────────────────────────────────────────────

func buildMapping(authMethod constants.AuthMethod, idpArn, roleArn string, idpConfigAuthMethod model.AuthMethodType, extraConfig map[string]string) *model.RoleMasterCspRoleMapping {
	cfg := &model.CspIdpConfig{
		AuthMethod: idpConfigAuthMethod,
		Config:     extraConfig,
	}
	cspRole := &model.CspRole{
		IdpIdentifier: idpArn,
		IamIdentifier: roleArn,
		CspIdpConfig:  cfg,
	}
	return &model.RoleMasterCspRoleMapping{
		RoleID:     1,
		AuthMethod: authMethod,
		CspRoleID:  1,
		CspRoles:   []*model.CspRole{cspRole},
	}
}

var errKeycloakFail = errors.New("keycloak unavailable")
var errStsFail = errors.New("STS call failed")

// ── 제어 가능한 Keycloak mock (credential 테스트 전용) ─────────────────────────

type mockKeycloakForCred struct {
	mockKeycloakService                    // 나머지 메서드는 기존 stub 재사용
	oidcToken   *gocloak.JWT
	oidcErr     error
	samlAssertion string
	samlErr     error
}

func (m *mockKeycloakForCred) GetImpersonationTokenByServiceAccount(ctx context.Context) (*gocloak.JWT, error) {
	return m.oidcToken, m.oidcErr
}
func (m *mockKeycloakForCred) GetSamlAssertionByServiceAccount(ctx context.Context, audience string) (string, error) {
	return m.samlAssertion, m.samlErr
}
