package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v13"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── 검증 서비스 전용 mock ─────────────────────────────────────────────────────

type mockValUserRepo struct {
	role    *model.UserWorkspaceRole
	roleErr error
}

func (m *mockValUserRepo) FindUserRoleInWorkspace(userID, workspaceID uint) (*model.UserWorkspaceRole, error) {
	return m.role, m.roleErr
}

type mockValMappingRepo struct {
	mapping    *model.RoleMasterCspRoleMapping
	mappingErr error
}

func (m *mockValMappingRepo) FindCspRoleMappingsByRoleIDAndCspType(roleID uint, cspType string, authMethod string) (*model.RoleMasterCspRoleMapping, error) {
	return m.mapping, m.mappingErr
}

type mockValKcService struct {
	mockKeycloakService          // 기본 stub 재사용
	oidcToken       *gocloak.JWT
	oidcErr         error
	samlAssertion   string
	samlErr         error
}

func (m *mockValKcService) GetImpersonationTokenByServiceAccount(ctx context.Context) (*gocloak.JWT, error) {
	return m.oidcToken, m.oidcErr
}
func (m *mockValKcService) GetSamlAssertionByServiceAccount(ctx context.Context, audience string) (string, error) {
	return m.samlAssertion, m.samlErr
}
func (m *mockValKcService) CheckSAMLClientConfig(ctx context.Context, clientID string) (string, error) {
	return "", nil
}

type mockValAwsService struct {
	oidcResult *model.CspCredentialResponse
	oidcErr    error
	samlResult *model.CspCredentialResponse
	samlErr    error
}

func (m *mockValAwsService) AssumeRoleWithWebIdentity(_ context.Context, roleArn, kcUserId, token, idpArn, region string) (*model.CspCredentialResponse, error) {
	return m.oidcResult, m.oidcErr
}
func (m *mockValAwsService) AssumeRoleWithSAML(_ context.Context, roleArn, principalArn, samlAssertion, region string) (*model.CspCredentialResponse, error) {
	return m.samlResult, m.samlErr
}
func (m *mockValAwsService) CheckOIDCProvider(_ context.Context, oidcProviderArn string) (string, error) {
	return "", nil
}
func (m *mockValAwsService) CheckSAMLProvider(_ context.Context, samlProviderArn string) (string, error) {
	return "", nil
}
func (m *mockValAwsService) CheckRoleTrust(_ context.Context, roleArn, expectedAction, expectedProviderArn string) (string, error) {
	return "", nil
}
func (m *mockValAwsService) CheckCallerIdentity(_ context.Context, accessKeyID, secretKey string) (string, error) {
	return "", nil
}

// ── 헬퍼 ─────────────────────────────────────────────────────────────────────

var (
	errValUserNotFound    = errors.New("user not found")
	errValMappingNotFound = errors.New("mapping not found")
	errValKcFail          = errors.New("keycloak unavailable")
	errValAwsFail         = errors.New("STS call failed")
)

func stdValUserRole() *model.UserWorkspaceRole {
	return &model.UserWorkspaceRole{RoleID: 1}
}

func buildValMapping(authMethod string, idpArn, roleArn string) *model.RoleMasterCspRoleMapping {
	cspRole := &model.CspRole{
		IdpIdentifier: idpArn,
		IamIdentifier: roleArn,
	}
	return &model.RoleMasterCspRoleMapping{
		RoleID:    1,
		CspRoleID: 1,
		CspRoles:  []*model.CspRole{cspRole},
	}
}

func buildValMappingWithSecretKey(accessKeyID, secretKey string) *model.RoleMasterCspRoleMapping {
	cfg := &model.CspIdpConfig{
		AuthMethod: model.AuthMethodSecretKey,
		Config:     map[string]string{"access_key_id": accessKeyID, "secret_access_key": secretKey},
	}
	cspRole := &model.CspRole{
		IdpIdentifier: "",
		IamIdentifier: "",
		CspIdpConfig:  cfg,
	}
	return &model.RoleMasterCspRoleMapping{
		RoleID:    1,
		CspRoleID: 1,
		CspRoles:  []*model.CspRole{cspRole},
	}
}

var awsValCred = &model.CspCredentialResponse{
	CspType:         "aws",
	AccessKeyId:     "ASIA_VAL_TEST",
	SecretAccessKey: "secret",
	SessionToken:    "token",
	Expiration:      time.Now().Add(1 * time.Hour),
}

func newValService(
	userRole *model.UserWorkspaceRole, userErr error,
	mapping *model.RoleMasterCspRoleMapping, mappingErr error,
	kc *mockValKcService,
	aws *mockValAwsService,
) *CspValidationService {
	if kc == nil {
		kc = &mockValKcService{}
	}
	if aws == nil {
		aws = &mockValAwsService{}
	}
	return &CspValidationService{
		userRepoIface:    &mockValUserRepo{role: userRole, roleErr: userErr},
		mappingRepoIface: &mockValMappingRepo{mapping: mapping, mappingErr: mappingErr},
		keycloakService:  kc,
		awsCredService:   aws,
	}
}

func valReq(cspType, authMethod string) *model.CspValidationRequest {
	return &model.CspValidationRequest{
		WorkspaceID: "1",
		CspType:     cspType,
		AuthMethod:  authMethod,
	}
}

// ── buildValidationSteps 단위 테스트 ─────────────────────────────────────────

// TC-VAL-STEPS-01: aws+OIDC → 6단계 반환, 초기값 skipped
func TestBuildValidationSteps_AWS_OIDC(t *testing.T) {
	steps := buildValidationSteps("aws", "OIDC")
	assert.Len(t, steps, 6)
	for i, s := range steps {
		assert.Equal(t, i+1, s.Step)
		assert.Equal(t, model.ValidationStepSkipped, s.Status)
	}
	assert.Equal(t, "DB 매핑 조회", steps[0].Name)
	assert.Equal(t, "임시자격증명 발급", steps[5].Name)
}

// TC-VAL-STEPS-02: aws+SAML → 7단계 반환
func TestBuildValidationSteps_AWS_SAML(t *testing.T) {
	steps := buildValidationSteps("aws", "SAML")
	assert.Len(t, steps, 7)
	assert.Equal(t, "Keycloak SAML 클라이언트 확인", steps[2].Name)
	assert.Equal(t, "임시자격증명 발급", steps[6].Name)
}

// TC-VAL-STEPS-03: aws+SECRET_KEY → 3단계 반환
func TestBuildValidationSteps_AWS_SecretKey(t *testing.T) {
	steps := buildValidationSteps("aws", "SECRET_KEY")
	assert.Len(t, steps, 3)
	assert.Equal(t, "DB 매핑 조회", steps[0].Name)
	assert.Equal(t, "CspIdpConfig 설정 확인", steps[1].Name)
	assert.Equal(t, "AWS 연결 확인", steps[2].Name)
}

// TC-VAL-STEPS-04: gcp+OIDC → 6단계 반환
func TestBuildValidationSteps_GCP_OIDC(t *testing.T) {
	steps := buildValidationSteps("gcp", "OIDC")
	assert.Len(t, steps, 6)
	assert.Equal(t, "GCP STS 토큰 교환", steps[3].Name)
}

// TC-VAL-STEPS-05: 미지원 조합 → 빈 슬라이스 반환
func TestBuildValidationSteps_Unsupported(t *testing.T) {
	steps := buildValidationSteps("azure", "OIDC")
	assert.Len(t, steps, 0)

	steps2 := buildValidationSteps("aws", "UNKNOWN")
	assert.Len(t, steps2, 0)
}

// ── stepRunner 단위 테스트 ────────────────────────────────────────────────────

// TC-VAL-RUNNER-01: 성공 시 ok 상태 반환
func TestStepRunner_Success(t *testing.T) {
	steps := buildValidationSteps("aws", "SECRET_KEY")
	ok := stepRunner(steps, 0, func() (string, error) {
		return "detail text", nil
	})
	assert.True(t, ok)
	assert.Equal(t, model.ValidationStepOk, steps[0].Status)
	assert.Equal(t, "detail text", steps[0].Detail)
}

// TC-VAL-RUNNER-02: 실패 시 failed 상태 + false 반환
func TestStepRunner_Failure(t *testing.T) {
	steps := buildValidationSteps("aws", "SECRET_KEY")
	ok := stepRunner(steps, 0, func() (string, error) {
		return "", errors.New("something went wrong")
	})
	assert.False(t, ok)
	assert.Equal(t, model.ValidationStepFailed, steps[0].Status)
	assert.Equal(t, "something went wrong", steps[0].Detail)
}

// ── buildFailedResponse 단위 테스트 ──────────────────────────────────────────

// TC-VAL-FAILED-01: 실패 응답 구조 검증
func TestBuildFailedResponse(t *testing.T) {
	steps := buildValidationSteps("aws", "SAML")
	steps[2].Status = model.ValidationStepFailed
	steps[2].Detail = "SAML 클라이언트 없음"

	resp := buildFailedResponse("aws", "SAML", 3, steps)

	assert.False(t, resp.Valid)
	assert.Equal(t, "aws", resp.CspType)
	assert.Equal(t, "SAML", resp.AuthMethod)
	assert.Equal(t, 3, resp.FailedStep)
	assert.Equal(t, "SAML 클라이언트 없음", resp.Error)
	assert.Len(t, resp.Steps, 7)
	assert.Nil(t, resp.Credentials)
}

// ── ValidateCredentials — 미지원 조합 ────────────────────────────────────────

// TC-VAL-001: 미지원 cspType+authMethod → error 반환
func TestValidateCredentials_UnsupportedCombination(t *testing.T) {
	svc := newValService(stdValUserRole(), nil, nil, nil, nil, nil)

	resp, err := svc.ValidateCredentials(context.Background(), 1, "kc_user", valReq("azure", "OIDC"))

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "unsupported combination")
}

// TC-VAL-002: 잘못된 workspaceId → error 반환
func TestValidateCredentials_InvalidWorkspaceID(t *testing.T) {
	svc := newValService(stdValUserRole(), nil, nil, nil, nil, nil)

	req := &model.CspValidationRequest{
		WorkspaceID: "invalid",
		CspType:     "aws",
		AuthMethod:  "OIDC",
	}
	resp, err := svc.ValidateCredentials(context.Background(), 1, "kc_user", req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// ── AWS OIDC 단계별 실패 시나리오 ─────────────────────────────────────────────

// TC-VAL-OIDC-01: Step 1 실패 — 워크스페이스 역할 없음
func TestValidateAWSWithOIDC_Step1_NoUserRole(t *testing.T) {
	svc := newValService(nil, errValUserNotFound, nil, nil, nil, nil)

	resp, err := svc.ValidateCredentials(context.Background(), 1, "kc_user", valReq("aws", "OIDC"))

	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Equal(t, 1, resp.FailedStep)
	assert.Equal(t, model.ValidationStepFailed, resp.Steps[0].Status)
	assert.Equal(t, model.ValidationStepSkipped, resp.Steps[1].Status)
}

// TC-VAL-OIDC-02: Step 1 실패 — 매핑 없음
func TestValidateAWSWithOIDC_Step1_NoMapping(t *testing.T) {
	svc := newValService(stdValUserRole(), nil, nil, errValMappingNotFound, nil, nil)

	resp, err := svc.ValidateCredentials(context.Background(), 1, "kc_user", valReq("aws", "OIDC"))

	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Equal(t, 1, resp.FailedStep)
}

// TC-VAL-OIDC-03: Step 2 실패 — CspRole idp/iam identifier 비어 있음
func TestValidateAWSWithOIDC_Step2_EmptyArn(t *testing.T) {
	mapping := buildValMapping("OIDC", "", "") // 빈 ARN
	svc := newValService(stdValUserRole(), nil, mapping, nil, nil, nil)

	resp, err := svc.ValidateCredentials(context.Background(), 1, "kc_user", valReq("aws", "OIDC"))

	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Equal(t, 2, resp.FailedStep)
}

// TC-VAL-OIDC-04: Step 3 실패 — Keycloak 토큰 발급 실패
func TestValidateAWSWithOIDC_Step3_KcFail(t *testing.T) {
	mapping := buildValMapping("OIDC",
		"arn:aws:iam::123:oidc-provider/keycloak.example.com",
		"arn:aws:iam::123:role/test-role",
	)
	kc := &mockValKcService{oidcErr: errValKcFail}
	svc := newValService(stdValUserRole(), nil, mapping, nil, kc, nil)

	resp, err := svc.ValidateCredentials(context.Background(), 1, "kc_user", valReq("aws", "OIDC"))

	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Equal(t, 3, resp.FailedStep)
	assert.Equal(t, model.ValidationStepFailed, resp.Steps[2].Status)
	// Step 4 이후는 skipped
	assert.Equal(t, model.ValidationStepSkipped, resp.Steps[3].Status)
}

// TC-VAL-OIDC-05: Step 6 실패 — AssumeRoleWithWebIdentity 실패
func TestValidateAWSWithOIDC_Step6_STSFail(t *testing.T) {
	mapping := buildValMapping("OIDC",
		"arn:aws:iam::123:oidc-provider/keycloak.example.com",
		"arn:aws:iam::123:role/test-role",
	)
	kc := &mockValKcService{oidcToken: &gocloak.JWT{AccessToken: "a_very_long_kc_access_token_that_exceeds_100_chars_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}}
	awsSvc := &mockValAwsService{oidcErr: errValAwsFail}
	svc := newValService(stdValUserRole(), nil, mapping, nil, kc, awsSvc)

	resp, err := svc.ValidateCredentials(context.Background(), 1, "kc_user", valReq("aws", "OIDC"))

	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Equal(t, 6, resp.FailedStep)
}

// ── AWS SAML 단계별 실패 시나리오 ─────────────────────────────────────────────

// TC-VAL-SAML-01: Step 1 실패 — 워크스페이스 역할 없음
func TestValidateAWSWithSAML_Step1_NoUserRole(t *testing.T) {
	svc := newValService(nil, errValUserNotFound, nil, nil, nil, nil)

	resp, err := svc.ValidateCredentials(context.Background(), 1, "kc_user", valReq("aws", "SAML"))

	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Equal(t, 1, resp.FailedStep)
	assert.Len(t, resp.Steps, 7) // SAML은 7단계
}

// TC-VAL-SAML-02: Step 2 실패 — 빈 ARN
func TestValidateAWSWithSAML_Step2_EmptyArn(t *testing.T) {
	mapping := buildValMapping("SAML", "", "")
	svc := newValService(stdValUserRole(), nil, mapping, nil, nil, nil)

	resp, err := svc.ValidateCredentials(context.Background(), 1, "kc_user", valReq("aws", "SAML"))

	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Equal(t, 2, resp.FailedStep)
}

// TC-VAL-SAML-03: Step 4 실패 — SAML Assertion 발급 실패
func TestValidateAWSWithSAML_Step4_AssertionFail(t *testing.T) {
	// Step 3(Keycloak SAML 클라이언트 확인)은 실제 Keycloak 호출이 필요하므로
	// 여기서는 assertion 발급 단계에서의 실패만 시뮬레이션
	// checkKeycloakSAMLClient는 외부 의존성이므로 Step 3은 실패할 수 있음
	// → SAML Step 4 테스트는 integration 테스트에서 수행
	t.Skip("Step 3 requires real Keycloak connection — covered in integration tests")
}

// ── AWS SECRET_KEY 단계별 시나리오 ────────────────────────────────────────────

// TC-VAL-SK-01: Step 1 실패 — 역할 없음
func TestValidateAWSWithSecretKey_Step1_NoRole(t *testing.T) {
	svc := newValService(nil, errValUserNotFound, nil, nil, nil, nil)

	resp, err := svc.ValidateCredentials(context.Background(), 1, "kc_user", valReq("aws", "SECRET_KEY"))

	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Equal(t, 1, resp.FailedStep)
	assert.Len(t, resp.Steps, 3)
}

// TC-VAL-SK-02: Step 2 실패 — CspIdpConfig 없음
func TestValidateAWSWithSecretKey_Step2_NoIdpConfig(t *testing.T) {
	mapping := buildValMapping("SECRET_KEY", "", "") // CspIdpConfig nil
	svc := newValService(stdValUserRole(), nil, mapping, nil, nil, nil)

	resp, err := svc.ValidateCredentials(context.Background(), 1, "kc_user", valReq("aws", "SECRET_KEY"))

	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Equal(t, 2, resp.FailedStep)
	assert.Contains(t, resp.Error, "CspIdpConfig 없음")
}

// TC-VAL-SK-03: Step 2 실패 — access_key_id 비어 있음
func TestValidateAWSWithSecretKey_Step2_EmptyKeyID(t *testing.T) {
	mapping := buildValMappingWithSecretKey("", "some_secret")
	svc := newValService(stdValUserRole(), nil, mapping, nil, nil, nil)

	resp, err := svc.ValidateCredentials(context.Background(), 1, "kc_user", valReq("aws", "SECRET_KEY"))

	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Equal(t, 2, resp.FailedStep)
	assert.Contains(t, resp.Error, "access_key_id 또는 secret_access_key 비어 있음")
}

// ── GCP OIDC 단계별 시나리오 ─────────────────────────────────────────────────

// TC-VAL-GCP-01: Step 1 실패 — 역할 없음
func TestValidateGCPWithOIDC_Step1_NoRole(t *testing.T) {
	svc := newValService(nil, errValUserNotFound, nil, nil, nil, nil)

	resp, err := svc.ValidateCredentials(context.Background(), 1, "kc_user", valReq("gcp", "OIDC"))

	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Equal(t, 1, resp.FailedStep)
	assert.Len(t, resp.Steps, 6)
}

// TC-VAL-GCP-02: Step 2 실패 — WIF Provider/SA 비어 있음
func TestValidateGCPWithOIDC_Step2_EmptyConfig(t *testing.T) {
	mapping := buildValMapping("OIDC", "", "")
	svc := newValService(stdValUserRole(), nil, mapping, nil, nil, nil)

	resp, err := svc.ValidateCredentials(context.Background(), 1, "kc_user", valReq("gcp", "OIDC"))

	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Equal(t, 2, resp.FailedStep)
	assert.Contains(t, resp.Error, "idp_identifier(WIF Provider) 또는 iam_identifier(SA email) 비어 있음")
}

// TC-VAL-GCP-03: Step 3 실패 — Keycloak 토큰 발급 실패
func TestValidateGCPWithOIDC_Step3_KcFail(t *testing.T) {
	mapping := buildValMapping("OIDC",
		"//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/pool/providers/kc",
		"sa@project.iam.gserviceaccount.com",
	)
	kc := &mockValKcService{oidcErr: errValKcFail}
	svc := newValService(stdValUserRole(), nil, mapping, nil, kc, nil)

	resp, err := svc.ValidateCredentials(context.Background(), 1, "kc_user", valReq("gcp", "OIDC"))

	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Equal(t, 3, resp.FailedStep)
}

// ── 응답 구조 완전성 검증 ─────────────────────────────────────────────────────

// TC-VAL-RESP-01: 실패 응답에 항상 전체 단계 포함
func TestValidateCredentials_FailResponse_AlwaysFullSteps(t *testing.T) {
	svc := newValService(nil, errValUserNotFound, nil, nil, nil, nil)

	for _, tc := range []struct {
		cspType    string
		authMethod string
		stepCount  int
	}{
		{"aws", "OIDC", 6},
		{"aws", "SAML", 7},
		{"aws", "SECRET_KEY", 3},
		{"gcp", "OIDC", 6},
	} {
		resp, err := svc.ValidateCredentials(context.Background(), 1, "kc_user", valReq(tc.cspType, tc.authMethod))
		require.NoError(t, err, "%s+%s", tc.cspType, tc.authMethod)
		assert.Len(t, resp.Steps, tc.stepCount, "%s+%s", tc.cspType, tc.authMethod)
	}
}

// TC-VAL-RESP-02: 실패 응답 credentials 필드 없음
func TestValidateCredentials_FailResponse_NoCredentials(t *testing.T) {
	svc := newValService(nil, errValUserNotFound, nil, nil, nil, nil)

	resp, err := svc.ValidateCredentials(context.Background(), 1, "kc_user", valReq("aws", "OIDC"))

	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Nil(t, resp.Credentials)
}

// TC-VAL-RESP-03: 실패 시 failedStep 이후 단계는 skipped 상태
func TestValidateCredentials_FailedStep_SubsequentSkipped(t *testing.T) {
	svc := newValService(nil, errValUserNotFound, nil, nil, nil, nil)

	resp, err := svc.ValidateCredentials(context.Background(), 1, "kc_user", valReq("aws", "SAML"))

	require.NoError(t, err)
	assert.Equal(t, 1, resp.FailedStep)
	// Step 1 failed, Step 2~7 skipped
	assert.Equal(t, model.ValidationStepFailed, resp.Steps[0].Status)
	for i := 1; i < len(resp.Steps); i++ {
		assert.Equal(t, model.ValidationStepSkipped, resp.Steps[i].Status,
			"step %d should be skipped", i+1)
	}
}
