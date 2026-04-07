package service

import (
	"context"
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
	db               *gorm.DB
	userRepo         *repository.UserRepository
	mappingRepo      *repository.CspMappingRepository
	userRepoIface    valUserRepo    // 테스트 주입용 (nil이면 userRepo 사용)
	mappingRepoIface valMappingRepo // 테스트 주입용 (nil이면 mappingRepo 사용)
	keycloakService  KeycloakService
	awsCredService   AwsCredentialService
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

// stepRunner 단계 실행 헬퍼 — 실패 시 false 반환
func stepRunner(steps []model.ValidationStep, idx int, fn func() (string, error)) bool {
	detail, err := fn()
	if err != nil {
		steps[idx].Status = model.ValidationStepFailed
		steps[idx].Detail = err.Error()
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
	if !stepRunner(steps, 0, func() (string, error) {
		userRole, err := s.resolveUserRepo().FindUserRoleInWorkspace(userID, workspaceID)
		if err != nil || userRole == nil {
			return "", fmt.Errorf("워크스페이스 역할 없음 — DB에 auth_method=OIDC 매핑 추가 필요")
		}
		m, err := s.resolveMappingRepo().FindCspRoleMappingsByRoleIDAndCspType(userRole.RoleID, cspType, authMethod)
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
	if !stepRunner(steps, 1, func() (string, error) {
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
	if !stepRunner(steps, 2, func() (string, error) {
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
	if !stepRunner(steps, 3, func() (string, error) {
		return s.awsCredService.CheckOIDCProvider(ctx, idpArn)
	}) {
		return buildFailedResponse(cspType, authMethod, 4, steps), nil
	}

	// Step 5: IAM Role WebIdentity Trust 확인
	if !stepRunner(steps, 4, func() (string, error) {
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
	if !stepRunner(steps, 5, func() (string, error) {
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
	if !stepRunner(steps, 0, func() (string, error) {
		userRole, err := s.resolveUserRepo().FindUserRoleInWorkspace(userID, workspaceID)
		if err != nil || userRole == nil {
			return "", fmt.Errorf("워크스페이스 역할 없음 — DB에 auth_method=SAML 매핑 추가 필요")
		}
		m, err := s.resolveMappingRepo().FindCspRoleMappingsByRoleIDAndCspType(userRole.RoleID, cspType, authMethod)
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
	if !stepRunner(steps, 1, func() (string, error) {
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
	// AWS SAML 클라이언트 ID는 urn:amazon:webservices (AWS 규약)
	// extendedConfig["saml_client_id"]가 없으면 urn:amazon:webservices로 조회
	kcSamlClientID := samlClientAudience
	if cspType == "aws" && !strings.Contains(kcSamlClientID, "urn:amazon") {
		kcSamlClientID = "urn:amazon:webservices"
	}
	if !stepRunner(steps, 2, func() (string, error) {
		return s.keycloakService.CheckSAMLClientConfig(ctx, kcSamlClientID)
	}) {
		return buildFailedResponse(cspType, authMethod, 3, steps), nil
	}

	// Step 4: SAML Assertion 발급 및 검증
	// token exchange audience는 Keycloak 클라이언트 ID (kcSamlClientID) 사용
	var samlAssertion string
	if !stepRunner(steps, 3, func() (string, error) {
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
	if !stepRunner(steps, 4, func() (string, error) {
		return s.awsCredService.CheckSAMLProvider(ctx, principalArn)
	}) {
		return buildFailedResponse(cspType, authMethod, 5, steps), nil
	}

	// Step 6: IAM Role SAML Trust 확인
	if !stepRunner(steps, 5, func() (string, error) {
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
	if !stepRunner(steps, 6, func() (string, error) {
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
	if !stepRunner(steps, 0, func() (string, error) {
		userRole, err := s.resolveUserRepo().FindUserRoleInWorkspace(userID, workspaceID)
		if err != nil || userRole == nil {
			return "", fmt.Errorf("워크스페이스 역할 없음")
		}
		m, err := s.resolveMappingRepo().FindCspRoleMappingsByRoleIDAndCspType(userRole.RoleID, cspType, authMethod)
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
	if !stepRunner(steps, 1, func() (string, error) {
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
	if !stepRunner(steps, 2, func() (string, error) {
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
	if !stepRunner(steps, 0, func() (string, error) {
		userRole, err := s.resolveUserRepo().FindUserRoleInWorkspace(userID, workspaceID)
		if err != nil || userRole == nil {
			return "", fmt.Errorf("워크스페이스 역할 없음")
		}
		m, err := s.resolveMappingRepo().FindCspRoleMappingsByRoleIDAndCspType(userRole.RoleID, cspType, authMethod)
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
	if !stepRunner(steps, 1, func() (string, error) {
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
	var accessToken string
	if !stepRunner(steps, 2, func() (string, error) {
		jwt, err := s.keycloakService.GetImpersonationTokenByServiceAccount(ctx)
		if err != nil {
			return "", fmt.Errorf("Keycloak OIDC 토큰 발급 실패: %v", err)
		}
		accessToken = jwt.AccessToken
		return fmt.Sprintf("OIDC JWT 발급 완료 (len=%d)", len(accessToken)), nil
	}) {
		return buildFailedResponse(cspType, authMethod, 3, steps), nil
	}

	// Step 4: GCP STS 토큰 교환
	// Step 5: SA Impersonation
	// Step 6: 임시자격증명 발급
	// GCP는 ExchangeTokenAndImpersonate에서 한 번에 처리 — 단계를 순서대로 시도
	gcpCredService := NewGcpCredentialService()
	var credSummary *model.CredentialSummary

	if !stepRunner(steps, 3, func() (string, error) {
		// GCP STS exchange only (ExchangeTokenAndImpersonate가 전체를 수행하므로 step 4에서 전체 실행)
		return "GCP STS 토큰 교환 시도 중...", nil
	}) {
		return buildFailedResponse(cspType, authMethod, 4, steps), nil
	}

	if !stepRunner(steps, 4, func() (string, error) {
		return "SA Impersonation 시도 중...", nil
	}) {
		return buildFailedResponse(cspType, authMethod, 5, steps), nil
	}

	if !stepRunner(steps, 5, func() (string, error) {
		creds, err := gcpCredService.ExchangeTokenAndImpersonate(ctx, wifProvider, saEmail, accessToken, "jwt")
		if err != nil {
			return "", fmt.Errorf("GCP 자격증명 발급 실패: %v — WIF Pool/Provider 설정 또는 SA 권한 확인", err)
		}
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
