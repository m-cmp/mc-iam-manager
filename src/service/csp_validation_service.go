package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/m-cmp/mc-iam-manager/util"
	"gorm.io/gorm"
)

// valUserRepo 테스트 주입을 위한 UserRepository 인터페이스
type valUserRepo interface {
	FindUserRoleInWorkspace(userID, workspaceID uint) (*model.UserWorkspaceRole, error)
}

// valMappingRepo 테스트 주입을 위한 CspMappingRepository 인터페이스
type valMappingRepo interface {
	FindCspRoleMappingsByRoleIDAndCspType(roleID uint, cspType string, authMethod string) (*model.RoleMasterCspRoleMapping, error)
}

// CspValidationService CSP 인증 설정 단계별 검증 서비스
type CspValidationService struct {
	db                  *gorm.DB
	userRepo            *repository.UserRepository
	mappingRepo         *repository.CspMappingRepository
	userRepoIface       valUserRepo          // 테스트 주입용 (nil이면 userRepo 사용)
	mappingRepoIface    valMappingRepo       // 테스트 주입용 (nil이면 mappingRepo 사용)
	keycloakService     KeycloakService
	awsCredService      AwsCredentialService
	gcpCredServiceIface GcpCredentialService // 테스트 주입용 (nil이면 NewGcpCredentialService() 사용)
}

// NewCspValidationService 새 CspValidationService 인스턴스 생성
func NewCspValidationService(db *gorm.DB) *CspValidationService {
	return &CspValidationService{
		db:              db,
		userRepo:        repository.NewUserRepository(db),
		mappingRepo:     repository.NewCspMappingRepository(db),
		keycloakService: NewKeycloakService(),
		awsCredService:  NewAwsCredentialService(),
	}
}

// --- AWS 검증 실패 단계별 RemediationGuide (CSP 콘솔에서의 조치 방법) ---
// 상세 절차는 mcmp-workflow/mc-iam-manager/design/CSP-ADMIN-WORKFLOW.md 참조.

// awsOidcProviderRemediation Step 4 (AWS OIDC Provider 확인) 실패 시 안내
var awsOidcProviderRemediation = &model.RemediationGuide{
	Summary: "AWS에 Keycloak을 신뢰하는 OIDC Identity Provider가 등록되어 있지 않습니다.",
	ConsoleSteps: []string{
		"AWS Console → IAM → Identity providers → Add provider",
		"Provider type: OpenID Connect",
		"Provider URL: Keycloak realm issuer URL (예: https://<keycloak-host>/realms/<realm>)",
		"Audience: mc-iam-manager Keycloak OIDC 클라이언트 ID",
		"생성 후 ARN을 CspRole.idp_identifier에 등록",
	},
	DocsRef: "mcmp-workflow/mc-iam-manager/design/CSP-ADMIN-WORKFLOW.md#step-4",
}

// awsOidcRoleTrustRemediation Step 5 (IAM Role WebIdentity Trust 확인) 실패 시 안내
var awsOidcRoleTrustRemediation = &model.RemediationGuide{
	Summary: "IAM Role의 Trust Policy에 OIDC Provider를 통한 AssumeRoleWithWebIdentity 권한이 없습니다.",
	ConsoleSteps: []string{
		"AWS Console → IAM → Roles → 대상 역할 선택 → Trust relationships → Edit trust policy",
	},
	Template: `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "<OIDC Provider ARN>"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "<keycloak-host>/realms/<realm>:aud": "<Keycloak OIDC 클라이언트 ID>"
        }
      }
    }
  ]
}`,
	DocsRef: "mcmp-workflow/mc-iam-manager/design/CSP-ADMIN-WORKFLOW.md#step-5",
}

// awsKeycloakOidcClientRemediation Step 3 (Keycloak OIDC 토큰 발급) 실패 시 안내 — Keycloak 서비스 계정/클라이언트 설정 확인
var awsKeycloakOidcClientRemediation = &model.RemediationGuide{
	Summary: "Keycloak OIDC 클라이언트(Service Account) 설정이 올바르지 않아 토큰 발급에 실패했습니다.",
	ConsoleSteps: []string{
		"Keycloak Admin Console → Clients → <OIDC 클라이언트> → Settings에서 Client authentication: On, Service accounts roles: On 확인",
		"Realm settings → Client policies에서 Token Exchange가 허용되어 있는지 확인",
		"MC_IAM_MANAGER_KEYCLOAK_OIDC_CLIENT_ID/SECRET 환경변수가 실제 Keycloak 클라이언트와 일치하는지 확인",
	},
	DocsRef: "mcmp-workflow/mc-iam-manager/design/CSP-ADMIN-WORKFLOW.md#step-1",
}

// awsSamlProviderRemediation Step 5 (AWS SAML Provider 확인) 실패 시 안내
var awsSamlProviderRemediation = &model.RemediationGuide{
	Summary: "AWS에 Keycloak SAML 메타데이터로 등록된 SAML Identity Provider가 없습니다.",
	ConsoleSteps: []string{
		"AWS Console → IAM → Identity providers → Add provider",
		"Provider type: SAML",
		"Keycloak SAML 클라이언트의 메타데이터 XML 업로드 (예: https://<keycloak-host>/realms/<realm>/protocol/saml/descriptor)",
		"생성 후 ARN을 CspRole.idp_identifier에 등록",
	},
	DocsRef: "mcmp-workflow/mc-iam-manager/design/CSP-ADMIN-WORKFLOW.md#step-4",
}

// awsSamlRoleTrustRemediation Step 6 (IAM Role SAML Trust 확인) 실패 시 안내
var awsSamlRoleTrustRemediation = &model.RemediationGuide{
	Summary: "IAM Role의 Trust Policy에 SAML Provider를 통한 AssumeRoleWithSAML 권한이 없습니다.",
	ConsoleSteps: []string{
		"AWS Console → IAM → Roles → 대상 역할 선택 → Trust relationships → Edit trust policy",
	},
	Template: `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "<SAML Provider ARN>"
      },
      "Action": "sts:AssumeRoleWithSAML",
      "Condition": {
        "StringEquals": {
          "SAML:aud": "https://signin.aws.amazon.com/saml"
        }
      }
    }
  ]
}`,
	DocsRef: "mcmp-workflow/mc-iam-manager/design/CSP-ADMIN-WORKFLOW.md#step-5",
}

// awsKeycloakSamlClientRemediation Step 3 (Keycloak SAML 클라이언트 확인) 실패 시 안내
var awsKeycloakSamlClientRemediation = &model.RemediationGuide{
	Summary: "Keycloak에 AWS SAML 연동용 클라이언트가 없거나 Role attribute mapper가 설정되어 있지 않습니다.",
	ConsoleSteps: []string{
		"Keycloak Admin Console → Clients → Create client (Client Type: SAML)",
		"Client ID: urn:amazon:webservices (AWS 기준)",
		"Client scopes → Protocol mappers에 Role List mapper 추가 (Attribute name: https://aws.amazon.com/SAML/Attributes/Role)",
	},
	DocsRef: "mcmp-workflow/mc-iam-manager/design/CSP-ADMIN-WORKFLOW.md#step-1",
}

// awsKeycloakSamlAssertionRemediation Step 4 (SAML Assertion 발급 및 검증) 실패 시 안내
var awsKeycloakSamlAssertionRemediation = &model.RemediationGuide{
	Summary: "SAML Assertion 발급에 실패했습니다 — Keycloak SAML 클라이언트/서비스 계정 설정을 확인하세요.",
	ConsoleSteps: []string{
		"Keycloak Admin Console에서 SAML 클라이언트가 활성화(Enabled)되어 있는지 확인",
		"MC_IAM_MANAGER_PLATFORMADMIN_ID/PASSWORD 환경변수로 platform admin 로그인이 가능한지 확인",
		"Realm settings → Client policies에서 Token Exchange 허용 여부 확인",
	},
	DocsRef: "mcmp-workflow/mc-iam-manager/design/CSP-ADMIN-WORKFLOW.md#step-1",
}

// awsSecretKeyConfigRemediation Step 2 (CspIdpConfig 설정 확인, SECRET_KEY) 실패 시 안내
var awsSecretKeyConfigRemediation = &model.RemediationGuide{
	Summary: "AWS Access Key/Secret Key가 CspIdpConfig에 등록되어 있지 않습니다.",
	ConsoleSteps: []string{
		"AWS Console → IAM → Users → 대상 사용자 → Security credentials → Create access key",
		"발급된 access_key_id/secret_access_key를 CspIdpConfig.Config에 등록 (POST /api/csp-idp-configs)",
	},
	DocsRef: "mcmp-workflow/mc-iam-manager/design/CSP-ADMIN-WORKFLOW.md#step-4",
}

// awsSecretKeyInvalidRemediation Step 3 (AWS 연결 확인, SECRET_KEY) 실패 시 안내
var awsSecretKeyInvalidRemediation = &model.RemediationGuide{
	Summary: "등록된 Access Key/Secret Key가 유효하지 않습니다 (AWS STS GetCallerIdentity 실패).",
	ConsoleSteps: []string{
		"AWS Console에서 해당 IAM 사용자의 Access Key가 활성 상태(Active)인지 확인",
		"키가 비활성화/삭제되었다면 새 키를 발급하여 CspIdpConfig 갱신",
	},
}

// resolveUserRepo 테스트 주입 우선, 없으면 프로덕션 repo 반환
func (s *CspValidationService) resolveUserRepo() valUserRepo {
	if s.userRepoIface != nil {
		return s.userRepoIface
	}
	return s.userRepo
}

// resolveMappingRepo 테스트 주입 우선, 없으면 프로덕션 repo 반환
func (s *CspValidationService) resolveMappingRepo() valMappingRepo {
	if s.mappingRepoIface != nil {
		return s.mappingRepoIface
	}
	return s.mappingRepo
}

// resolveGcpCredService 테스트 주입 우선, 없으면 프로덕션 서비스 반환
func (s *CspValidationService) resolveGcpCredService() GcpCredentialService {
	if s.gcpCredServiceIface != nil {
		return s.gcpCredServiceIface
	}
	return NewGcpCredentialService()
}

// --- GCP OIDC 단계별 RemediationGuide ---
// CSP 콘솔에서의 실제 조치 방법. 참고: mcmp-workflow/mc-iam-manager/design/CSP-ADMIN-WORKFLOW.md (Step 4, 5 — GCP 섹션)

// gcpStsRemediation GCP STS 토큰 교환(step 4) 실패 시 안내 — WIF Pool/Provider가 없거나
// Keycloak을 신뢰하도록 설정되어 있지 않은 경우.
var gcpStsRemediation = &model.RemediationGuide{
	Summary: "GCP Workload Identity Federation Pool/Provider가 Keycloak을 신뢰하도록 설정되어 있지 않거나 존재하지 않습니다.",
	ConsoleSteps: []string{
		"GCP Console → IAM & Admin → Workload Identity Federation → Create Pool",
		"Pool에 OIDC Provider 추가: Issuer URL = Keycloak realm issuer (예: https://<keycloak-host>/realms/<realm>)",
		"Attribute mapping 설정 (google.subject = assertion.sub 등)",
		"Attribute condition은 'in' 연산자 대신 등가 비교 권장: assertion.aud == \"<Keycloak 클라이언트 ID>\" (ID Token의 aud는 배열이 아닌 단일 문자열)",
		"생성된 리소스 이름 앞에 //iam.googleapis.com/ 접두사를 붙여 CspRole.idp_identifier에 등록 (예: //iam.googleapis.com/projects/<NUM>/locations/global/workloadIdentityPools/<POOL>/providers/<PROVIDER> — 접두사 없으면 'Unsupported audience type' 오류)",
	},
	DocsRef: "mcmp-workflow/mc-iam-manager/design/CSP-ADMIN-WORKFLOW.md#step-4",
}

// gcpSaImpersonationRemediation SA Impersonation(step 5) 실패 시 안내 — 대상 Service Account가
// 없거나 WIF Pool에 workloadIdentityUser 권한이 부여되어 있지 않은 경우.
var gcpSaImpersonationRemediation = &model.RemediationGuide{
	Summary: "대상 Service Account가 없거나, WIF Pool에서 해당 SA를 impersonate/토큰 발급할 권한이 없습니다.",
	ConsoleSteps: []string{
		"GCP Console → IAM & Admin → Service Accounts → Create Service Account (예: mcmp-<role>@<project>.iam.gserviceaccount.com)",
		"Service Account → Permissions → Grant Access → WIF Pool의 principal(또는 principalSet)에 다음 두 역할을 모두 부여: roles/iam.workloadIdentityUser + roles/iam.serviceAccountTokenCreator " +
			"(workloadIdentityUser만으로는 부족 — 실제 토큰 발급(generateAccessToken) 시 'Permission iam.serviceAccounts.getAccessToken denied'로 실패함)",
		"IAM & Admin → IAM → 해당 SA에 실제 사용할 리소스 권한 부여 (예: roles/compute.admin)",
		"SA 이메일을 CspRole.iam_identifier에 등록",
	},
	DocsRef: "mcmp-workflow/mc-iam-manager/design/CSP-ADMIN-WORKFLOW.md#step-5",
}

// buildSteps CSP×AuthMethod별 전체 단계를 skipped 초기 상태로 반환
func buildValidationSteps(cspType, authMethod string) []model.ValidationStep {
	var names []string
	switch cspType {
	case "aws":
		switch authMethod {
		case string(model.AuthMethodOIDC):
			names = []string{
				"DB 매핑 조회",
				"CspRole 설정 확인",
				"Keycloak OIDC 토큰 발급",
				"AWS OIDC Provider 확인",
				"IAM Role WebIdentity Trust 확인",
				"임시자격증명 발급",
			}
		case string(model.AuthMethodSAML):
			names = []string{
				"DB 매핑 조회",
				"CspRole 설정 확인",
				"Keycloak SAML 클라이언트 확인",
				"SAML Assertion 발급 및 검증",
				"AWS SAML Provider 확인",
				"IAM Role SAML Trust 확인",
				"임시자격증명 발급",
			}
		case string(model.AuthMethodSecretKey):
			names = []string{
				"DB 매핑 조회",
				"CspIdpConfig 설정 확인",
				"AWS 연결 확인",
			}
		}
	case "gcp":
		switch authMethod {
		case string(model.AuthMethodOIDC):
			names = []string{
				"DB 매핑 조회",
				"CspRole 설정 확인",
				"Keycloak OIDC 토큰 발급",
				"GCP STS 토큰 교환",
				"SA Impersonation",
				"임시자격증명 발급",
			}
		}
	}

	steps := make([]model.ValidationStep, len(names))
	for i, name := range names {
		steps[i] = model.ValidationStep{
			Step:   i + 1,
			Name:   name,
			Status: model.ValidationStepSkipped,
			Detail: "",
		}
	}
	return steps
}

// stepRunner 단계 실행 헬퍼 — 실패 시 false 반환. remediation은 실패 시에만 해당 단계에 첨부된다(nil이면 미첨부).
func stepRunner(steps []model.ValidationStep, idx int, remediation *model.RemediationGuide, fn func() (string, error)) bool {
	detail, err := fn()
	if err != nil {
		steps[idx].Status = model.ValidationStepFailed
		steps[idx].Detail = err.Error()
		steps[idx].Remediation = remediation
		return false
	}
	steps[idx].Status = model.ValidationStepOk
	steps[idx].Detail = detail
	return true
}

// buildFailedResponse 실패 응답 생성
func buildFailedResponse(cspType, authMethod string, failedStep int, steps []model.ValidationStep) *model.CspValidationResponse {
	return &model.CspValidationResponse{
		Valid:      false,
		CspType:    cspType,
		AuthMethod: authMethod,
		FailedStep: failedStep,
		Error:      steps[failedStep-1].Detail,
		Steps:      steps,
	}
}

// ValidateCredentials 워크스페이스 사용자 기준 CSP 인증 설정 단계별 검증
func (s *CspValidationService) ValidateCredentials(ctx context.Context, userID uint, kcUserID string, req *model.CspValidationRequest) (*model.CspValidationResponse, error) {
	cspType := req.CspType
	authMethod := req.AuthMethod

	log.Printf("[CSP_VALIDATE] Start — userID=%d workspaceID=%s csp=%s method=%s", userID, req.WorkspaceID, cspType, authMethod)

	steps := buildValidationSteps(cspType, authMethod)
	if len(steps) == 0 {
		return nil, fmt.Errorf("unsupported combination: %s+%s", cspType, authMethod)
	}

	workspaceIDInt, err := util.StringToUint(req.WorkspaceID)
	if err != nil || workspaceIDInt == 0 {
		return nil, fmt.Errorf("invalid workspaceId: %s", req.WorkspaceID)
	}

	switch cspType {
	case "aws":
		switch authMethod {
		case string(model.AuthMethodOIDC):
			return s.validateAWSWithOIDC(ctx, userID, kcUserID, workspaceIDInt, cspType, authMethod, steps)
		case string(model.AuthMethodSAML):
			return s.validateAWSWithSAML(ctx, userID, workspaceIDInt, cspType, authMethod, steps)
		case string(model.AuthMethodSecretKey):
			return s.validateAWSWithSecretKey(ctx, userID, workspaceIDInt, cspType, authMethod, steps)
		}
	case "gcp":
		switch authMethod {
		case string(model.AuthMethodOIDC):
			return s.validateGCPWithOIDC(ctx, userID, kcUserID, workspaceIDInt, cspType, authMethod, steps)
		}
	}

	return nil, fmt.Errorf("unsupported combination: %s+%s", cspType, authMethod)
}

// --- AWS OIDC (6단계) ---

func (s *CspValidationService) validateAWSWithOIDC(ctx context.Context, userID uint, kcUserID string, workspaceID uint, cspType, authMethod string, steps []model.ValidationStep) (*model.CspValidationResponse, error) {
	// Step 1: DB 매핑 조회
	var mapping *model.RoleMasterCspRoleMapping
	if !stepRunner(steps, 0, nil, func() (string, error) {
		userRole, err := s.resolveUserRepo().FindUserRoleInWorkspace(userID, workspaceID)
		if err != nil || userRole == nil {
			return "", fmt.Errorf("워크스페이스 역할 없음 — DB에 auth_method=OIDC 매핑 추가 필요")
		}
		m, err := s.resolveMappingRepo().FindCspRoleMappingsByRoleIDAndCspType(userRole.RoleID, cspType, authMethod)
		if errors.Is(err, repository.ErrAmbiguousCspRoleMapping) {
			return "", fmt.Errorf("매핑 대상 CspRole 이 2개 이상 — 하나만 남기거나 authMethod 를 지정 필요")
		}
		if err != nil || m == nil {
			return "", fmt.Errorf("OIDC 매핑 없음 — mcmp_role_csp_role_mappings에 auth_method=OIDC 레코드 추가 필요")
		}
		mapping = m
		return fmt.Sprintf("roleID=%d → cspRoleID=%d", userRole.RoleID, m.CspRoles[0].ID), nil
	}) {
		return buildFailedResponse(cspType, authMethod, 1, steps), nil
	}

	// Step 2: CspRole 설정 확인
	var idpArn, roleArn string
	if !stepRunner(steps, 1, nil, func() (string, error) {
		cspRole := mapping.CspRoles[0]
		if cspRole.IdpIdentifier == "" || cspRole.IamIdentifier == "" {
			return "", fmt.Errorf("CspRole.idp_identifier(OIDC Provider ARN) 또는 iam_identifier(Role ARN) 비어 있음")
		}
		idpArn = cspRole.IdpIdentifier
		roleArn = cspRole.IamIdentifier
		return fmt.Sprintf("idpArn=%s roleArn=%s", idpArn, roleArn), nil
	}) {
		return buildFailedResponse(cspType, authMethod, 2, steps), nil
	}

	// Step 3: Keycloak OIDC 토큰 발급
	var accessToken string
	if !stepRunner(steps, 2, awsKeycloakOidcClientRemediation, func() (string, error) {
		jwt, err := s.keycloakService.GetImpersonationTokenByServiceAccount(ctx)
		if err != nil {
			return "", fmt.Errorf("Keycloak OIDC 토큰 발급 실패: %v — Keycloak OIDC 클라이언트 설정 또는 시크릿 확인", err)
		}
		accessToken = jwt.AccessToken
		// iss/aud 간단 확인 (JWT 파싱 없이 토큰 길이 확인)
		if len(accessToken) < 100 {
			return "", fmt.Errorf("발급된 토큰이 너무 짧음 — OIDC 클라이언트 설정 확인")
		}
		return fmt.Sprintf("OIDC JWT 발급 완료 (len=%d)", len(accessToken)), nil
	}) {
		return buildFailedResponse(cspType, authMethod, 3, steps), nil
	}

	// Step 4: AWS OIDC Provider 확인
	if !stepRunner(steps, 3, awsOidcProviderRemediation, func() (string, error) {
		return s.awsCredService.CheckOIDCProvider(ctx, idpArn)
	}) {
		return buildFailedResponse(cspType, authMethod, 4, steps), nil
	}

	// Step 5: IAM Role WebIdentity Trust 확인
	if !stepRunner(steps, 4, awsOidcRoleTrustRemediation, func() (string, error) {
		return s.awsCredService.CheckRoleTrust(ctx, roleArn, "sts:AssumeRoleWithWebIdentity", idpArn)
	}) {
		return buildFailedResponse(cspType, authMethod, 5, steps), nil
	}

	// Step 6: 임시자격증명 발급
	defaultRegion := os.Getenv("AWS_REGION")
	if defaultRegion == "" {
		defaultRegion = "ap-northeast-2"
	}
	var credSummary *model.CredentialSummary
	if !stepRunner(steps, 5, nil, func() (string, error) {
		creds, err := s.awsCredService.AssumeRoleWithWebIdentity(ctx, roleArn, kcUserID, accessToken, idpArn, defaultRegion)
		if err != nil {
			return "", fmt.Errorf("AssumeRoleWithWebIdentity 실패: %v", err)
		}
		credSummary = &model.CredentialSummary{
			AccessKeyId: creds.AccessKeyId,
			Expiration:  creds.Expiration,
		}
		return fmt.Sprintf("AccessKeyId=%s Expiration=%s", creds.AccessKeyId, creds.Expiration.String()), nil
	}) {
		return buildFailedResponse(cspType, authMethod, 6, steps), nil
	}

	return &model.CspValidationResponse{
		Valid:       true,
		CspType:     cspType,
		AuthMethod:  authMethod,
		FailedStep:  0,
		Steps:       steps,
		Credentials: credSummary,
	}, nil
}

// --- AWS SAML (7단계) ---

func (s *CspValidationService) validateAWSWithSAML(ctx context.Context, userID uint, workspaceID uint, cspType, authMethod string, steps []model.ValidationStep) (*model.CspValidationResponse, error) {
	// Step 1: DB 매핑 조회
	var mapping *model.RoleMasterCspRoleMapping
	if !stepRunner(steps, 0, nil, func() (string, error) {
		userRole, err := s.resolveUserRepo().FindUserRoleInWorkspace(userID, workspaceID)
		if err != nil || userRole == nil {
			return "", fmt.Errorf("워크스페이스 역할 없음 — DB에 auth_method=SAML 매핑 추가 필요")
		}
		m, err := s.resolveMappingRepo().FindCspRoleMappingsByRoleIDAndCspType(userRole.RoleID, cspType, authMethod)
		if errors.Is(err, repository.ErrAmbiguousCspRoleMapping) {
			return "", fmt.Errorf("매핑 대상 CspRole 이 2개 이상 — 하나만 남기거나 authMethod 를 지정 필요")
		}
		if err != nil || m == nil {
			return "", fmt.Errorf("SAML 매핑 없음 — mcmp_role_csp_role_mappings에 auth_method=SAML 레코드 추가 필요")
		}
		mapping = m
		return fmt.Sprintf("roleID=%d → cspRoleID=%d", userRole.RoleID, m.CspRoles[0].ID), nil
	}) {
		return buildFailedResponse(cspType, authMethod, 1, steps), nil
	}

	// Step 2: CspRole 설정 확인
	var principalArn, roleArn, samlClientAudience string
	if !stepRunner(steps, 1, nil, func() (string, error) {
		cspRole := mapping.CspRoles[0]
		if cspRole.IdpIdentifier == "" || cspRole.IamIdentifier == "" {
			return "", fmt.Errorf("CspRole.idp_identifier(Principal ARN) 또는 iam_identifier(Role ARN) 비어 있음")
		}
		principalArn = cspRole.IdpIdentifier
		roleArn = cspRole.IamIdentifier
		samlClientAudience = principalArn
		if extConfig, ok := cspRole.ExtendedConfig["saml_client_id"].(string); ok && extConfig != "" {
			samlClientAudience = extConfig
		}
		return fmt.Sprintf("principalArn=%s roleArn=%s samlClient=%s", principalArn, roleArn, samlClientAudience), nil
	}) {
		return buildFailedResponse(cspType, authMethod, 2, steps), nil
	}

	// Step 3: Keycloak SAML 클라이언트 확인
	// AWS SAML 클라이언트 ID: extendedConfig["saml_client_id"] 우선, 없으면 SAML_CLIENT_ID_AWS 환경변수 사용
	kcSamlClientID := samlClientAudience
	if cspType == "aws" && !strings.Contains(kcSamlClientID, "urn:amazon") {
		kcSamlClientID = os.Getenv("SAML_CLIENT_ID_AWS")
	}
	if !stepRunner(steps, 2, awsKeycloakSamlClientRemediation, func() (string, error) {
		return s.keycloakService.CheckSAMLClientConfig(ctx, kcSamlClientID)
	}) {
		return buildFailedResponse(cspType, authMethod, 3, steps), nil
	}

	// Step 4: SAML Assertion 발급 및 검증
	// token exchange audience는 Keycloak 클라이언트 ID (kcSamlClientID) 사용
	var samlAssertion string
	if !stepRunner(steps, 3, awsKeycloakSamlAssertionRemediation, func() (string, error) {
		assertion, err := s.keycloakService.GetSamlAssertionByServiceAccount(ctx, kcSamlClientID)
		if err != nil {
			return "", fmt.Errorf("SAML Assertion 발급 실패: %v — Keycloak SAML 클라이언트 설정 확인", err)
		}
		samlAssertion = assertion
		// Role attribute 형식 확인 (decoded assertion에서 확인)
		detail := fmt.Sprintf("SAML Assertion 발급 완료 (len=%d)", len(assertion))
		return detail, nil
	}) {
		return buildFailedResponse(cspType, authMethod, 4, steps), nil
	}

	// Step 5: AWS SAML Provider 확인
	if !stepRunner(steps, 4, awsSamlProviderRemediation, func() (string, error) {
		return s.awsCredService.CheckSAMLProvider(ctx, principalArn)
	}) {
		return buildFailedResponse(cspType, authMethod, 5, steps), nil
	}

	// Step 6: IAM Role SAML Trust 확인
	if !stepRunner(steps, 5, awsSamlRoleTrustRemediation, func() (string, error) {
		return s.awsCredService.CheckRoleTrust(ctx, roleArn, "sts:AssumeRoleWithSAML", principalArn)
	}) {
		return buildFailedResponse(cspType, authMethod, 6, steps), nil
	}

	// Step 7: 임시자격증명 발급
	samlDefaultRegion := os.Getenv("AWS_REGION")
	if samlDefaultRegion == "" {
		samlDefaultRegion = "ap-northeast-2"
	}
	var credSummary *model.CredentialSummary
	if !stepRunner(steps, 6, nil, func() (string, error) {
		creds, err := s.awsCredService.AssumeRoleWithSAML(ctx, roleArn, principalArn, samlAssertion, samlDefaultRegion)
		if err != nil {
			return "", fmt.Errorf("AssumeRoleWithSAML 실패: %v", err)
		}
		credSummary = &model.CredentialSummary{
			AccessKeyId: creds.AccessKeyId,
			Expiration:  creds.Expiration,
		}
		return fmt.Sprintf("AccessKeyId=%s Expiration=%s", creds.AccessKeyId, creds.Expiration.String()), nil
	}) {
		return buildFailedResponse(cspType, authMethod, 7, steps), nil
	}

	return &model.CspValidationResponse{
		Valid:       true,
		CspType:     cspType,
		AuthMethod:  authMethod,
		FailedStep:  0,
		Steps:       steps,
		Credentials: credSummary,
	}, nil
}

// --- AWS SECRET_KEY (3단계) ---

func (s *CspValidationService) validateAWSWithSecretKey(ctx context.Context, userID uint, workspaceID uint, cspType, authMethod string, steps []model.ValidationStep) (*model.CspValidationResponse, error) {
	// Step 1: DB 매핑 조회
	var mapping *model.RoleMasterCspRoleMapping
	if !stepRunner(steps, 0, nil, func() (string, error) {
		userRole, err := s.resolveUserRepo().FindUserRoleInWorkspace(userID, workspaceID)
		if err != nil || userRole == nil {
			return "", fmt.Errorf("워크스페이스 역할 없음")
		}
		m, err := s.resolveMappingRepo().FindCspRoleMappingsByRoleIDAndCspType(userRole.RoleID, cspType, authMethod)
		if errors.Is(err, repository.ErrAmbiguousCspRoleMapping) {
			return "", fmt.Errorf("매핑 대상 CspRole 이 2개 이상 — 하나만 남기거나 authMethod 를 지정 필요")
		}
		if err != nil || m == nil {
			return "", fmt.Errorf("SECRET_KEY 매핑 없음 — mcmp_role_csp_role_mappings에 auth_method=SECRET_KEY 레코드 추가 필요")
		}
		mapping = m
		return fmt.Sprintf("roleID=%d → cspRoleID=%d", userRole.RoleID, m.CspRoles[0].ID), nil
	}) {
		return buildFailedResponse(cspType, authMethod, 1, steps), nil
	}

	// Step 2: CspIdpConfig 설정 확인
	var accessKeyID, secretKey string
	if !stepRunner(steps, 1, awsSecretKeyConfigRemediation, func() (string, error) {
		cspRole := mapping.CspRoles[0]
		if cspRole.CspIdpConfig == nil {
			return "", fmt.Errorf("CspIdpConfig 없음 — CspRole에 IDP 설정 연결 필요")
		}
		accessKeyID = cspRole.CspIdpConfig.GetAccessKeyID()
		secretKey = cspRole.CspIdpConfig.GetSecretAccessKey()
		if accessKeyID == "" || secretKey == "" {
			return "", fmt.Errorf("access_key_id 또는 secret_access_key 비어 있음 — CspIdpConfig 값 입력 필요")
		}
		return fmt.Sprintf("access_key_id=%s...", accessKeyID[:min(8, len(accessKeyID))]), nil
	}) {
		return buildFailedResponse(cspType, authMethod, 2, steps), nil
	}

	// Step 3: AWS 연결 확인 (GetCallerIdentity)
	if !stepRunner(steps, 2, awsSecretKeyInvalidRemediation, func() (string, error) {
		return s.awsCredService.CheckCallerIdentity(ctx, accessKeyID, secretKey)
	}) {
		return buildFailedResponse(cspType, authMethod, 3, steps), nil
	}

	return &model.CspValidationResponse{
		Valid:      true,
		CspType:    cspType,
		AuthMethod: authMethod,
		FailedStep: 0,
		Steps:      steps,
	}, nil
}

// --- GCP OIDC (6단계) ---

func (s *CspValidationService) validateGCPWithOIDC(ctx context.Context, userID uint, kcUserID string, workspaceID uint, cspType, authMethod string, steps []model.ValidationStep) (*model.CspValidationResponse, error) {
	// Step 1: DB 매핑 조회
	var mapping *model.RoleMasterCspRoleMapping
	if !stepRunner(steps, 0, nil, func() (string, error) {
		userRole, err := s.resolveUserRepo().FindUserRoleInWorkspace(userID, workspaceID)
		if err != nil || userRole == nil {
			return "", fmt.Errorf("워크스페이스 역할 없음")
		}
		m, err := s.resolveMappingRepo().FindCspRoleMappingsByRoleIDAndCspType(userRole.RoleID, cspType, authMethod)
		if errors.Is(err, repository.ErrAmbiguousCspRoleMapping) {
			return "", fmt.Errorf("매핑 대상 CspRole 이 2개 이상 — 하나만 남기거나 authMethod 를 지정 필요")
		}
		if err != nil || m == nil {
			return "", fmt.Errorf("GCP OIDC 매핑 없음 — auth_method=OIDC, csp_type=gcp 레코드 추가 필요")
		}
		mapping = m
		return fmt.Sprintf("roleID=%d → cspRoleID=%d", userRole.RoleID, m.CspRoles[0].ID), nil
	}) {
		return buildFailedResponse(cspType, authMethod, 1, steps), nil
	}

	// Step 2: CspRole 설정 확인
	var wifProvider, saEmail string
	if !stepRunner(steps, 1, nil, func() (string, error) {
		cspRole := mapping.CspRoles[0]
		if cspRole.IdpIdentifier == "" || cspRole.IamIdentifier == "" {
			return "", fmt.Errorf("idp_identifier(WIF Provider) 또는 iam_identifier(SA email) 비어 있음")
		}
		wifProvider = cspRole.IdpIdentifier
		saEmail = cspRole.IamIdentifier
		return fmt.Sprintf("wifProvider=%s saEmail=%s", wifProvider, saEmail), nil
	}) {
		return buildFailedResponse(cspType, authMethod, 2, steps), nil
	}

	// Step 3: Keycloak OIDC 토큰 발급
	// GCP WIF STS는 단일 문자열 aud를 요구하므로 Access Token(aud="account")이 아니라
	// ID Token(aud=OIDC 클라이언트 ID)을 사용해야 한다 — Alibaba OIDC(OI-1)와 동일한 이유.
	var idToken string
	if !stepRunner(steps, 2, nil, func() (string, error) {
		jwt, err := s.keycloakService.GetImpersonationTokenByServiceAccount(ctx)
		if err != nil {
			return "", fmt.Errorf("Keycloak OIDC 토큰 발급 실패: %v", err)
		}
		idToken = jwt.IDToken
		return fmt.Sprintf("OIDC JWT 발급 완료 (len=%d)", len(idToken)), nil
	}) {
		return buildFailedResponse(cspType, authMethod, 3, steps), nil
	}

	// Step 4: GCP STS 토큰 교환 (Keycloak JWT → GCP federated access token)
	gcpCredService := s.resolveGcpCredService()
	var federatedToken string
	if !stepRunner(steps, 3, gcpStsRemediation, func() (string, error) {
		token, err := gcpCredService.ExchangeToken(ctx, wifProvider, idToken, "jwt")
		if err != nil {
			return "", fmt.Errorf("GCP STS 토큰 교환 실패: %v — WIF Pool/Provider 설정 확인", err)
		}
		federatedToken = token
		return fmt.Sprintf("GCP federated token 발급 완료 (len=%d)", len(federatedToken)), nil
	}) {
		return buildFailedResponse(cspType, authMethod, 4, steps), nil
	}

	// Step 5: SA Impersonation (federated token → SA-impersonated access token)
	var creds *model.CspCredentialResponse
	if !stepRunner(steps, 4, gcpSaImpersonationRemediation, func() (string, error) {
		c, err := gcpCredService.GenerateAccessToken(ctx, saEmail, federatedToken)
		if err != nil {
			return "", fmt.Errorf("SA Impersonation 실패: %v — SA 존재 여부 및 WIF Pool의 workloadIdentityUser 권한 확인", err)
		}
		creds = c
		return fmt.Sprintf("SA Impersonation 완료 (accessToken len=%d)", len(creds.AccessToken)), nil
	}) {
		return buildFailedResponse(cspType, authMethod, 5, steps), nil
	}

	// Step 6: 임시자격증명 발급 (step 4-5에서 이미 발급된 자격증명 요약)
	var credSummary *model.CredentialSummary
	if !stepRunner(steps, 5, nil, func() (string, error) {
		credSummary = &model.CredentialSummary{
			AccessKeyId: creds.AccessToken,
			Expiration:  creds.Expiration,
		}
		return fmt.Sprintf("GCP AccessToken 발급 완료 (len=%d)", len(creds.AccessToken)), nil
	}) {
		return buildFailedResponse(cspType, authMethod, 6, steps), nil
	}

	return &model.CspValidationResponse{
		Valid:       true,
		CspType:     cspType,
		AuthMethod:  authMethod,
		FailedStep:  0,
		Steps:       steps,
		Credentials: credSummary,
	}, nil
}

// min 정수 최솟값 (Go 1.21 미만 호환)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
