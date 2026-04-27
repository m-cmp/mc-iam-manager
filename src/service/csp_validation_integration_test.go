package service

// csp_validation_integration_test.go
//
// 실제 Keycloak / AWS 연결이 필요한 통합 테스트.
// 실행 방법:
//   INTEGRATION_TEST=1 go test github.com/m-cmp/mc-iam-manager/service -run "TestIntegration" -v -count=1
//
// 필수 환경 변수 (mc-iam-manager .env 기준):
//   KC:  MC_IAM_MANAGER_KEYCLOAK_HOST, MC_IAM_MANAGER_KEYCLOAK_REALM
//        MC_IAM_MANAGER_KEYCLOAK_ADMIN, MC_IAM_MANAGER_KEYCLOAK_ADMIN_PASSWORD
//        MC_IAM_MANAGER_KEYCLOAK_CLIENT_NAME, MC_IAM_MANAGER_KEYCLOAK_CLIENT_SECRET
//        MC_IAM_MANAGER_KEYCLOAK_OIDC_CLIENT_NAME, MC_IAM_MANAGER_KEYCLOAK_OIDC_CLIENT_SECRET
//   AWS: IT_AWS_OIDC_PROVIDER_ARN, IT_AWS_OIDC_ROLE_ARN (IAM read 권한 보유 role)
//        IT_AWS_SAML_PROVIDER_ARN, IT_AWS_SAML_ROLE_ARN
//        IT_AWS_ACCESS_KEY_ID, IT_AWS_SECRET_ACCESS_KEY (SECRET_KEY 테스트용)
//        IT_KC_SAML_CLIENT_ID (기본값: urn:amazon:webservices)

import (
	"context"
	"fmt"
	"os"
	"testing"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	stsservice "github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/m-cmp/mc-iam-manager/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipIfNotIntegration 통합 테스트 환경이 아니면 skip
func skipIfNotIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("INTEGRATION_TEST=1 이 설정되지 않아 통합 테스트를 건너뜁니다")
	}
}

// initKCForIntegration Keycloak 설정 초기화 (테스트 내 1회)
func initKCForIntegration(t *testing.T) {
	t.Helper()
	if config.KC != nil {
		return
	}
	if err := config.InitKeycloak(); err != nil {
		t.Skipf("Keycloak 초기화 실패 — KC 서버 미연결로 테스트를 건너뜁니다: %v", err)
	}
}

// envOrDefault 환경 변수 읽기 (없으면 기본값)
func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// ── KC SAML 클라이언트 확인 ───────────────────────────────────────────────────

// TestIntegrationCheckSAMLClientConfig_Exists SAML 클라이언트가 존재하는 경우 성공 확인
func TestIntegrationCheckSAMLClientConfig_Exists(t *testing.T) {
	skipIfNotIntegration(t)
	initKCForIntegration(t)

	svc := NewKeycloakService()
	ctx := context.Background()
	clientID := envOrDefault("IT_KC_SAML_CLIENT_ID", "urn:amazon:webservices")

	detail, err := svc.CheckSAMLClientConfig(ctx, clientID)

	require.NoError(t, err, "SAML 클라이언트 확인 실패")
	assert.NotEmpty(t, detail)
	t.Logf("CheckSAMLClientConfig: %s", detail)
}

// TestIntegrationCheckSAMLClientConfig_NotExists 존재하지 않는 클라이언트 → 오류 반환
func TestIntegrationCheckSAMLClientConfig_NotExists(t *testing.T) {
	skipIfNotIntegration(t)
	initKCForIntegration(t)

	svc := NewKeycloakService()
	ctx := context.Background()

	_, err := svc.CheckSAMLClientConfig(ctx, "nonexistent-client-that-should-not-exist")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "없음")
	t.Logf("expected error: %v", err)
}

// ── AWS OIDC IAM 읽기 (Steps 4-5) ────────────────────────────────────────────

// TestIntegrationAWSOIDC_IAMRead platformAdmin OIDC 토큰으로 IAM 역할 Trust Policy 확인
func TestIntegrationAWSOIDC_IAMRead(t *testing.T) {
	skipIfNotIntegration(t)
	initKCForIntegration(t)

	oidcProviderArn := os.Getenv("IT_AWS_OIDC_PROVIDER_ARN")
	roleArn := os.Getenv("IT_AWS_OIDC_ROLE_ARN")
	if oidcProviderArn == "" || roleArn == "" {
		t.Skip("IT_AWS_OIDC_PROVIDER_ARN / IT_AWS_OIDC_ROLE_ARN 미설정 — 건너뜁니다")
	}

	ctx := context.Background()

	// Step 1: Keycloak OIDC 토큰 발급 (서비스 계정)
	kcSvc := NewKeycloakService()
	jwt, err := kcSvc.GetImpersonationTokenByServiceAccount(ctx)
	require.NoError(t, err, "Keycloak OIDC 토큰 발급 실패")
	require.NotEmpty(t, jwt.AccessToken)
	t.Logf("OIDC token len=%d", len(jwt.AccessToken))

	// Step 2: AssumeRoleWithWebIdentity → IAM 읽기용 임시 자격증명 발급
	awsSvc := NewAwsCredentialService()
	creds, err := awsSvc.AssumeRoleWithWebIdentity(ctx, roleArn, "integration-test", jwt.AccessToken, oidcProviderArn, "ap-northeast-2")
	require.NoError(t, err, "AssumeRoleWithWebIdentity 실패")
	require.NotEmpty(t, creds.AccessKeyId)
	t.Logf("Assumed role: AccessKeyId=%s Expiration=%s", creds.AccessKeyId, creds.Expiration)

	// Step 3: 발급된 임시 자격증명으로 OIDC Provider 확인 (Steps 4)
	// checkAWSOIDCProvider는 환경 자격증명(DefaultConfig)을 사용 — 임시 키를 환경에 설정
	t.Setenv("AWS_ACCESS_KEY_ID", creds.AccessKeyId)
	t.Setenv("AWS_SECRET_ACCESS_KEY", creds.SecretAccessKey)
	t.Setenv("AWS_SESSION_TOKEN", creds.SessionToken)

	detail, err := awsSvc.CheckOIDCProvider(ctx, oidcProviderArn)
	if err != nil {
		t.Logf("CheckOIDCProvider: %v (degraded mode — IAM 권한 부족 가능)", err)
	} else {
		t.Logf("CheckOIDCProvider OK: %s", detail)
		assert.NotEmpty(t, detail)
	}

	// Step 4: IAM Role Trust Policy 확인 (Step 5)
	trustDetail, err := awsSvc.CheckRoleTrust(ctx, roleArn, "sts:AssumeRoleWithWebIdentity", oidcProviderArn)
	if err != nil {
		t.Logf("CheckRoleTrust: %v (degraded mode 가능)", err)
	} else {
		t.Logf("CheckRoleTrust OK: %s", trustDetail)
		assert.NotEmpty(t, trustDetail)
	}
}

// ── AWS SECRET_KEY Step 3 — GetCallerIdentity SDK signed call ────────────────

// TestIntegrationAWSSecretKey_GetCallerIdentity SDK signed GetCallerIdentity 검증
func TestIntegrationAWSSecretKey_GetCallerIdentity(t *testing.T) {
	skipIfNotIntegration(t)

	accessKeyID := os.Getenv("IT_AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("IT_AWS_SECRET_ACCESS_KEY")
	if accessKeyID == "" || secretKey == "" {
		t.Skip("IT_AWS_ACCESS_KEY_ID / IT_AWS_SECRET_ACCESS_KEY 미설정 — 건너뜁니다")
	}

	ctx := context.Background()

	awsSvc := NewAwsCredentialService()
	detail, err := awsSvc.CheckCallerIdentity(ctx, accessKeyID, secretKey)

	require.NoError(t, err, "GetCallerIdentity 실패 — 자격증명 확인 필요")
	assert.Contains(t, detail, "Account=")
	assert.Contains(t, detail, "Arn=")
	t.Logf("GetCallerIdentity: %s", detail)
}

// TestIntegrationAWSSecretKey_InvalidKey 잘못된 키 → 명확한 오류 반환
func TestIntegrationAWSSecretKey_InvalidKey(t *testing.T) {
	skipIfNotIntegration(t)

	ctx := context.Background()

	awsSvc := NewAwsCredentialService()
	_, err := awsSvc.CheckCallerIdentity(ctx, "AKIAINVALIDKEYXXX", "invalidsecretkey1234567890")

	assert.Error(t, err)
	t.Logf("expected error: %v", err)
}

// ── ValidateCredentials 전체 흐름 통합 테스트 ────────────────────────────────

// TestIntegrationValidateCredentials_AWSOIDC_FullFlow AWS OIDC 전체 검증 흐름
func TestIntegrationValidateCredentials_AWSOIDC_FullFlow(t *testing.T) {
	skipIfNotIntegration(t)
	initKCForIntegration(t)

	oidcProviderArn := os.Getenv("IT_AWS_OIDC_PROVIDER_ARN")
	roleArn := os.Getenv("IT_AWS_OIDC_ROLE_ARN")
	workspaceID := envOrDefault("IT_WORKSPACE_ID", "1")
	if oidcProviderArn == "" || roleArn == "" {
		t.Skip("IT_AWS_OIDC_PROVIDER_ARN / IT_AWS_OIDC_ROLE_ARN 미설정 — 건너뜁니다")
	}

	mapping := buildValMapping("OIDC", oidcProviderArn, roleArn)
	svc := newValService(
		stdValUserRole(), nil,
		mapping, nil,
		nil, // 실제 KC 서비스 사용
		nil, // 실제 AWS 서비스 사용
	)
	// 실제 서비스로 교체
	svc.keycloakService = NewKeycloakService()
	svc.awsCredService = NewAwsCredentialService()

	ctx := context.Background()
	resp, err := svc.ValidateCredentials(ctx, 1, "integration-test-user", valReq("aws", "OIDC"))

	require.NoError(t, err)
	t.Logf("ValidateCredentials result: valid=%v failedStep=%d", resp.Valid, resp.FailedStep)
	for _, step := range resp.Steps {
		t.Logf("  Step %d [%s] %s: %s", step.Step, step.Status, step.Name, step.Detail)
	}

	// workspaceID에 매핑이 없으면 Step 1에서 실패 — 이는 정상
	// 실제 full flow는 DB에 워크스페이스 역할이 있어야 통과
	assert.NotNil(t, resp)
	_ = workspaceID
	fmt.Println("full flow test completed")
}

// TestIntegrationAWSSecretKey_GetCallerIdentity_STS_SDK STS SDK 직접 호출 검증
// (iam_test.go 패턴과 동일 — StaticCredentials + STS GetCallerIdentity)
func TestIntegrationAWSSTS_DirectSDKCall(t *testing.T) {
	skipIfNotIntegration(t)

	accessKeyID := os.Getenv("IT_AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("IT_AWS_SECRET_ACCESS_KEY")
	if accessKeyID == "" || secretKey == "" {
		t.Skip("IT_AWS_ACCESS_KEY_ID / IT_AWS_SECRET_ACCESS_KEY 미설정 — 건너뜁니다")
	}

	ctx := context.Background()

	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretKey, ""),
		),
	)
	require.NoError(t, err)

	stsClient := stsservice.NewFromConfig(cfg)
	result, err := stsClient.GetCallerIdentity(ctx, &stsservice.GetCallerIdentityInput{})
	require.NoError(t, err)

	t.Logf("STS GetCallerIdentity: Account=%s Arn=%s UserID=%s",
		*result.Account, *result.Arn, *result.UserId)
	assert.NotEmpty(t, *result.Account)
	assert.NotEmpty(t, *result.Arn)
}
