package aws

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/m-cmp/mc-iam-manager/csp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// skipIfNotAWSIntegration 실제 AWS STS/IAM 연동이 필요한 테스트는 명시적 opt-in이 없으면 건너뜀.
// (INTEGRATION_TEST 게이트(csp_credential_integration_test.go)와 동일한 취지의 관례를 따르되,
//  DB뿐 아니라 실제 AWS 자격증명/토큰이 필요하다는 점을 명확히 하기 위해 별도 env var 사용)
func skipIfNotAWSIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("AWS_INTEGRATION_TEST") != "1" {
		t.Skip("AWS_INTEGRATION_TEST=1 이 설정되지 않아 실제 AWS STS/IAM 연동 테스트를 건너뜁니다")
	}
}

// envOrDefault 환경 변수 읽기 (없으면 기본값)
func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// WorkspaceRole 테스트용 워크스페이스 역할 모델
type WorkspaceRole struct {
	ID          uint   `gorm:"primaryKey"`
	WorkspaceID string `gorm:"column:workspace_id"`
}

// WorkspaceRoleCSPRoleMapping 테스트용 워크스페이스 역할-CSP 역할 매핑 모델
type WorkspaceRoleCSPRoleMapping struct {
	ID              uint   `gorm:"primaryKey"`
	WorkspaceRoleID uint   `gorm:"column:workspace_role_id"`
	CSPType         string `gorm:"column:csp_type"`
	RoleARN         string `gorm:"column:role_arn"`
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 테스트용 테이블 생성
	err = db.AutoMigrate(&WorkspaceRole{}, &WorkspaceRoleCSPRoleMapping{})
	require.NoError(t, err)

	// 테스트 데이터 삽입
	workspaceRole := &WorkspaceRole{
		ID:          1,
		WorkspaceID: "e2464ed3-8d96-4c5e-84ad-22c22de38576", // JWT의 sub 값과 일치
	}
	err = db.Create(workspaceRole).Error
	require.NoError(t, err)

	cspRoleMapping := &WorkspaceRoleCSPRoleMapping{
		ID:              1,
		WorkspaceRoleID: 1,
		CSPType:         "AWS",
		RoleARN:         "arn:aws:iam::050864702683:role/mciam_viewer",
	}
	err = db.Create(cspRoleMapping).Error
	require.NoError(t, err)

	return db
}

// TestAWSIAMClient 실제 AWS STS(AssumeRoleWithWebIdentity) + IAM(GetRole) 연동 통합 테스트.
//
// 이 테스트는 순수 단위 테스트가 아니라 실제 AWS 계정에 대한 종단 검증이다:
//   - NewAWSIAMClient 는 내부적으로 getSecurityToken() 을 호출해 실제 AWS STS 엔드포인트에
//     AssumeRoleWithWebIdentity 요청을 보낸다 (WebIdentityToken 이 만료/무효면 여기서 실패한다).
//   - GetRole 은 실제 AWS IAM API를 호출하며, 대상 계정에 "mciam-csp-role-manager" 라는
//     이름의 역할이 실제로 존재해야 한다.
//
// 실행 방법:
//
//	AWS_INTEGRATION_TEST=1 go test ./csp/aws/... -run TestAWSIAMClient -v
//
// 유효한 자격증명으로 실행하려면 AWS_TEST_ROLE_ARN 환경변수로 RoleARN을 오버라이드한다
// (WebIdentityToken은 아래 하드코딩된 만료 샘플 그대로 — 실제 실행 시 이 상수를 직접 교체할 것.
// 반드시 STS 인증 실패로 끝난다 — 게이트가 실제 경로를 타는지 확인하는 용도로는 충분하다):
//
//	AWS_TEST_ROLE_ARN : AssumeRoleWithWebIdentity 대상 역할 ARN
func TestAWSIAMClient(t *testing.T) {
	skipIfNotAWSIntegration(t)

	roleARN := envOrDefault("AWS_TEST_ROLE_ARN", "arn:aws:iam::050864702683:role/mciam_viewer")

	config := &csp.IAMClientConfig{
		Region:           "ap-northeast-2",
		RoleARN:          roleARN,
		WebIdentityToken: "eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJmRmtqcWltT1FhSEs0ZlQ2YWVlV3VZUi1KOFk3TjdEWmV2ZFJQZjRNNmNJIn0.eyJleHAiOjE3NDczOTU4OTAsImlhdCI6MTc0NzM1OTg5NSwianRpIjoiNjI0OGY0NzYtZDU4MC00MGFjLWEyZDgtMTE5MzNjZGNiZmMwIiwiaXNzIjoiaHR0cHM6Ly9tY2lhbS5vbmVjbG91ZGNvbi5jb20vcmVhbG1zL21jaWFtLWRlbXAzIiwiYXVkIjoib2lkYy1zZXJ2aWNlQWNjb3VudENsaWVudCIsInN1YiI6ImEzNDQ1Y2VjLTBjZjQtNGFkNi1hNjk0LTUyMDdjYmMxMWIzOSIsInR5cCI6IkJlYXJlciIsImF6cCI6Im1jaWFtQ2xpZW50Iiwic2Vzc2lvbl9zdGF0ZSI6ImQzNDE1Y2U3LTA0MjItNGQ1ZS1iNTlhLWVhMDIzNmI5Njk4ZCIsImFjciI6IjEiLCJhbGxvd2VkLW9yaWdpbnMiOlsiLyoiXSwicmVhbG1fYWNjZXNzIjp7InJvbGVzIjpbIm9mZmxpbmVfYWNjZXNzIiwidW1hX2F1dGhvcml6YXRpb24iLCJkZWZhdWx0LXJvbGVzLW1jaWFtLWRlbXAzIiwicGxhdGZvcm1BZG1pbiJdfSwicmVzb3VyY2VfYWNjZXNzIjp7Im1jaWFtLW9pZGMtY2xpZW50LWRlbXAzIjp7InJvbGVzIjpbIm9pZGMtYXNzdW1lLXJvbGUiXX0sImFjY291bnQiOnsicm9sZXMiOlsibWFuYWdlLWFjY291bnQiLCJtYW5hZ2UtYWNjb3VudC1saW5rcyIsInZpZXctcHJvZmlsZSJdfX0sImF1dGhvcml6YXRpb24iOnsicGVybWlzc2lvbnMiOlt7InJzaWQiOiIwMTVmZTBhNC0xZjA0LTQzY2UtYWE4Yy00N2NiMjg5ZWVhNjMiLCJyc25hbWUiOiJEZWZhdWx0IFJlc291cmNlIn1dfSwic2NvcGUiOiJwcm9maWxlIGVtYWlsIiwic2lkIjoiZDM0MTVjZTctMDQyMi00ZDVlLWI1OWEtZWEwMjM2Yjk2OThkIiwiZW1haWxfdmVyaWZpZWQiOnRydWUsIm5hbWUiOiJsZWUgbWFuIiwicHJlZmVycmVkX3VzZXJuYW1lIjoibGVlbWFuIiwiZ2l2ZW5fbmFtZSI6ImxlZSIsImZhbWlseV9uYW1lIjoibWFuIiwiZW1haWwiOiJsZWVtYW5AdGVzdC5jb20ifQ.WZPH_0VvUhJNm7S5E21BFnK0kfH30Wevheoy9IvZtmUR-vWA4-8Jv7Ilu_o9BINJm47GNodK71R-PxC3MfTwjFJ7apT-m5r2AKtrzvIUE7HFuzaykcrqQ2kmBnUHqguFcSoo0GlZ7vfqO9fYMv5Gyn62bKk25ITleItahTFaY198M0p0je_xwLbm0wf7Wj7WfvqHVIyNgIZetksoe00ikNXCHeLjU1lLTPO-zT-QvT46CV0ETnPO9tIRiPSe2utP96GQ1Tav2YIW9zXE-hGb2pGWDCA5AK-ohABoOMedwm9ajReDDclGKE0SUAePs2GP1eAcke8IJ3NcGHTPyhClww",
		WorkspaceTicket:  "eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJmRmtqcWltT1FhSEs0ZlQ2YWVlV3VZUi1KOFk3TjdEWmV2ZFJQZjRNNmNJIn0.eyJleHAiOjE3NDczOTU4OTAsImlhdCI6MTc0NzM1OTg5NSwianRpIjoiNjI0OGY0NzYtZDU4MC00MGFjLWEyZDgtMTE5MzNjZGNiZmMwIiwiaXNzIjoiaHR0cHM6Ly9tY2lhbS5vbmVjbG91ZGNvbi5jb20vcmVhbG1zL21jaWFtLWRlbXAzIiwiYXVkIjoib2lkYy1zZXJ2aWNlQWNjb3VudENsaWVudCIsInN1YiI6ImEzNDQ1Y2VjLTBjZjQtNGFkNi1hNjk0LTUyMDdjYmMxMWIzOSIsInR5cCI6IkJlYXJlciIsImF6cCI6Im1jaWFtQ2xpZW50Iiwic2Vzc2lvbl9zdGF0ZSI6ImQzNDE1Y2U3LTA0MjItNGQ1ZS1iNTlhLWVhMDIzNmI5Njk4ZCIsImFjciI6IjEiLCJhbGxvd2VkLW9yaWdpbnMiOlsiLyoiXSwicmVhbG1fYWNjZXNzIjp7InJvbGVzIjpbIm9mZmxpbmVfYWNjZXNzIiwidW1hX2F1dGhvcml6YXRpb24iLCJkZWZhdWx0LXJvbGVzLW1jaWFtLWRlbXAzIiwicGxhdGZvcm1BZG1pbiJdfSwicmVzb3VyY2VfYWNjZXNzIjp7Im1jaWFtLW9pZGMtY2xpZW50LWRlbXAzIjp7InJvbGVzIjpbIm9pZGMtYXNzdW1lLXJvbGUiXX0sImFjY291bnQiOnsicm9sZXMiOlsibWFuYWdlLWFjY291bnQiLCJtYW5hZ2UtYWNjb3VudC1saW5rcyIsInZpZXctcHJvZmlsZSJdfX0sImF1dGhvcml6YXRpb24iOnsicGVybWlzc2lvbnMiOlt7InJzaWQiOiIwMTVmZTBhNC0xZjA0LTQzY2UtYWE4Yy00N2NiMjg5ZWVhNjMiLCJyc25hbWUiOiJEZWZhdWx0IFJlc291cmNlIn1dfSwic2NvcGUiOiJwcm9maWxlIGVtYWlsIiwic2lkIjoiZDM0MTVjZTctMDQyMi00ZDVlLWI1OWEtZWEwMjM2Yjk2OThkIiwiZW1haWxfdmVyaWZpZWQiOnRydWUsIm5hbWUiOiJsZWUgbWFuIiwicHJlZmVycmVkX3VzZXJuYW1lIjoibGVlbWFuIiwiZ2l2ZW5fbmFtZSI6ImxlZSIsImZhbWlseV9uYW1lIjoibWFuIiwiZW1haWwiOiJsZWVtYW5AdGVzdC5jb20ifQ.WZPH_0VvUhJNm7S5E21BFnK0kfH30Wevheoy9IvZtmUR-vWA4-8Jv7Ilu_o9BINJm47GNodK71R-PxC3MfTwjFJ7apT-m5r2AKtrzvIUE7HFuzaykcrqQ2kmBnUHqguFcSoo0GlZ7vfqO9fYMv5Gyn62bKk25ITleItahTFaY198M0p0je_xwLbm0wf7Wj7WfvqHVIyNgIZetksoe00ikNXCHeLjU1lLTPO-zT-QvT46CV0ETnPO9tIRiPSe2utP96GQ1Tav2YIW9zXE-hGb2pGWDCA5AK-ohABoOMedwm9ajReDDclGKE0SUAePs2GP1eAcke8IJ3NcGHTPyhClww",
		Timeout:          300 * time.Second,
	}

	// db 는 NewAWSIAMClient/AWSIAMClient의 어떤 메서드에서도 참조되지 않고 구조체에 저장만 된다
	// (grep 결과 c.db 사용처 없음). WorkspaceRoleCSPRoleMapping SQLite 픽스처는 죽은 셋업이었으므로
	// 제거하고 nil 을 전달한다.
	client, err := NewAWSIAMClient(config, nil)
	require.NoError(t, err)
	require.NotNil(t, client)

	ctx := context.Background()

	t.Run("GetRole", func(t *testing.T) {
		role, err := client.GetRole(ctx, "mciam-csp-role-manager")
		assert.NoError(t, err)
		assert.NotNil(t, role)
	})
}

// TC-IAM-VALIDATE-01: RoleARN 미설정 시 오류 반환
func TestNewAWSIAMClient_MissingRoleARN(t *testing.T) {
	db := setupTestDB(t)
	cfg := &csp.IAMClientConfig{
		Region:           "ap-northeast-2",
		WebIdentityToken: "dummy-token",
		RoleARN:          "",
	}
	_, err := NewAWSIAMClient(cfg, db)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "role ARN is required")
}

// TC-IAM-VALIDATE-02: WebIdentityToken 미설정 시 오류 반환
func TestNewAWSIAMClient_MissingWebIdentityToken(t *testing.T) {
	db := setupTestDB(t)
	cfg := &csp.IAMClientConfig{
		Region:           "ap-northeast-2",
		WebIdentityToken: "",
		RoleARN:          "arn:aws:iam::123456789012:role/test-role",
	}
	_, err := NewAWSIAMClient(cfg, db)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "web identity token is required")
}
