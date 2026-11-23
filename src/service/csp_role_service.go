package service

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/csp"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/m-cmp/mc-iam-manager/util"
	"gorm.io/gorm"
)

// CspRoleService CSP 역할 서비스
type CspRoleService struct {
	db                 *gorm.DB
	cspRoleRepo        *repository.CspRoleRepository
	tempCredentialRepo *repository.TempCredentialRepository
	cspIdpConfigRepo   *repository.CspIdpConfigRepository
	keycloakService    KeycloakService
	// awsRoleClient 테스트에서 AWS IAM 호출을 바꿔 끼울 때만 쓴다 (nil 이면 관리용 자격증명으로 만든 실제 클라이언트)
	awsRoleClient func(cfg *model.CspIdpConfig) (awsRoleAPI, error)
}

// NewCspRoleService 새 CspRoleService 인스턴스 생성
func NewCspRoleService(db *gorm.DB, keycloakService KeycloakService) *CspRoleService {
	return &CspRoleService{
		db:                 db,
		cspRoleRepo:        repository.NewCspRoleRepository(db),
		tempCredentialRepo: repository.NewTempCredentialRepository(db),
		cspIdpConfigRepo:   repository.NewCspIdpConfigRepository(db),
		keycloakService:    keycloakService,
	}
}

// GetAllCSPRoles 모든 CSP 역할을 조회합니다. (AWS API live 조회)
func (s *CspRoleService) GetAllCSPRoles(ctx context.Context, cspType string) ([]*model.CspRole, error) {
	roles, err := s.cspRoleRepo.FindAll()
	if err != nil {
		log.Printf("Failed to get CSP roles: %v", err)
		return nil, err
	}

	return roles, nil
}

// ListCspRolesFromDB DB에 저장된 모든 CspRole 레코드를 조회합니다.
func (s *CspRoleService) ListCspRolesFromDB() ([]*model.CspRole, error) {
	return s.cspRoleRepo.FindAllFromDB()
}

// CSP 역할 목록 중 MCIAM_ 접두사를 가진 역할만 조회합니다.
func (s *CspRoleService) GetMciamCSPRoles(ctx context.Context, cspType string) ([]*model.CspRole, error) {
	roles, err := s.cspRoleRepo.FindMciamRoleFromCsp(cspType)
	if err != nil {
		log.Printf("Failed to get CSP roles: %v", err)
		return nil, err
	}

	return roles, nil
}

// CreateCspRole CSP 역할을 생성합니다.
// CSP role(클라우드 IAM 역할) + Keycloak client 설정을 추상적으로 처리합니다.
// CSP별 구현은 private 메서드로 디스패치합니다.
func (s *CspRoleService) CreateCspRole(req *model.CreateCspRoleRequest) (*model.CspRole, error) {
	// 1. prefix 정규화
	if !strings.HasPrefix(req.CspRoleName, constants.CspRoleNamePrefix) {
		req.CspRoleName = constants.CspRoleNamePrefix + req.CspRoleName
	}

	// 관리용 Role(IdP Config role_arn)은 CspRole 로 등록하지 않는다 (409)
	if err := s.GuardNotManagementRole(req.CspRoleName, req.IamIdentifier, "registered as a CSP role"); err != nil {
		return nil, err
	}

	// 2. DB 중복 확인 — status="failed" 레코드는 재시도 허용
	existing, err := s.cspRoleRepo.GetCspRoleByName(req.CspRoleName, req.CspType)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing CSP role: %w", err)
	}
	if existing != nil && existing.Status != "failed" {
		if req.CspType != "aws" {
			return existing, nil
		}
		// AWS 는 DB 레코드만 믿지 않고 GetRole 로 다시 확인한다 (IAM-BUG-051).
		// 있으면 값을 다시 동기화하고, AWS 에서 지워졌으면(NoSuchEntity) 소속 Config 기준으로 다시 만든다.
		// 소속 Config 를 지정하지 않은 요청은 기존 레코드의 Config 를 쓴다.
		if req.CspIdpConfigID == nil && existing.CspIdpConfigID != nil {
			cfgID := *existing.CspIdpConfigID
			req.CspIdpConfigID = &cfgID
		}
	}

	// 3. 소속 IdP Config 확정 (CSP 계정 → IdP Config → CspRole)
	idpConfig, err := s.resolveCspIdpConfig(req)
	if err != nil {
		return nil, err
	}

	// 4. CSP별 역할 생성 디스패치
	var cspRole *model.CspRole
	switch req.CspType {
	case "aws":
		var api awsRoleAPI
		if api, err = s.awsRoleClientFor(idpConfig); err == nil {
			cspRole, _, err = s.ensureAwsRole(api, req, idpConfig)
		}
	default:
		// AWS 이외 CSP: DB 등록만 수행 (클라우드 역할 생성은 향후 확장)
		// idp/iam identifier는 admin이 CSP 콘솔에서 수동 생성한 리소스의 ARN/이메일을 그대로 등록하는 값이므로
		// 요청에서 받은 값을 반드시 저장해야 한다 (OI-24: 이전에는 여기서 누락되어 항상 빈 값으로 저장됐음).
		cspRole = &model.CspRole{
			Name:          req.CspRoleName,
			Description:   req.Description,
			CspType:       req.CspType,
			Status:        "created",
			IdpIdentifier: req.IdpIdentifier,
			IamIdentifier: req.IamIdentifier,
		}
		applyCspIdpConfig(cspRole, idpConfig)
		if err := s.cspRoleRepo.CreateCspRoleRecord(cspRole); err != nil {
			return nil, fmt.Errorf("failed to create CSP role record: %w", err)
		}
	}
	if err != nil {
		return nil, err
	}

	// 5. Keycloak 클라이언트 확인
	if kcErr := s.ensureKeycloakClientForCspRole(req, cspRole); kcErr != nil {
		log.Printf("Warning: Keycloak client check failed for CSP role %s: %v", cspRole.Name, kcErr)
	}

	return cspRole, nil
}

// syncAwsRoleToDb 이미 AWS에 존재하는 역할 정보를 DB에 동기화합니다. (private)
func (s *CspRoleService) syncAwsRoleToDb(newRole *model.CspRole, targetCspRole *iam.GetRoleOutput) (*model.CspRole, error) {
	newRole.Status = "created"
	newRole.IamIdentifier = *targetCspRole.Role.Arn
	newRole.CreateDate = *targetCspRole.Role.CreateDate
	newRole.Path = *targetCspRole.Role.Path
	newRole.IamRoleId = *targetCspRole.Role.RoleId
	if targetCspRole.Role.Description != nil {
		newRole.Description = *targetCspRole.Role.Description
	}
	if targetCspRole.Role.MaxSessionDuration != nil {
		newRole.MaxSessionDuration = targetCspRole.Role.MaxSessionDuration
	}
	if targetCspRole.Role.PermissionsBoundary != nil && targetCspRole.Role.PermissionsBoundary.PermissionsBoundaryArn != nil {
		newRole.PermissionsBoundary = *targetCspRole.Role.PermissionsBoundary.PermissionsBoundaryArn
	}
	if targetCspRole.Role.RoleLastUsed != nil {
		roleLastUsed := &model.RoleLastUsed{}
		if targetCspRole.Role.RoleLastUsed.LastUsedDate != nil {
			roleLastUsed.LastUsedDate = *targetCspRole.Role.RoleLastUsed.LastUsedDate
		}
		if targetCspRole.Role.RoleLastUsed.Region != nil {
			roleLastUsed.Region = *targetCspRole.Role.RoleLastUsed.Region
		}
		newRole.RoleLastUsed = roleLastUsed
	}
	if len(targetCspRole.Role.Tags) > 0 {
		tags := make([]model.Tag, len(targetCspRole.Role.Tags))
		for i, tag := range targetCspRole.Role.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[i] = model.Tag{
					Key:   *tag.Key,
					Value: *tag.Value,
				}
			}
		}
		newRole.Tags = tags
	}
	if err := s.cspRoleRepo.UpdateCspRoleRecord(newRole); err != nil {
		return nil, fmt.Errorf("failed to update CSP role in database: %v", err)
	}
	return newRole, nil
}

// ensureKeycloakClientForCspRole CSP 역할에 대응하는 Keycloak 클라이언트 설정을 확인합니다. (private)
func (s *CspRoleService) ensureKeycloakClientForCspRole(req *model.CreateCspRoleRequest, role *model.CspRole) error {
	if s.keycloakService == nil {
		return nil
	}

	ctx := context.TODO()

	// AuthMethod에 따라 SAML/OIDC 클라이언트 확인
	switch req.AuthMethod {
	case constants.AuthMethodSAML:
		// CSP별 SAML 클라이언트 ID 환경변수
		samlClientID := ""
		switch req.CspType {
		case "aws":
			samlClientID = os.Getenv("SAML_CLIENT_ID_AWS")
		case "alibaba":
			samlClientID = os.Getenv("SAML_CLIENT_ID_ALIBABA")
		}
		if samlClientID != "" {
			if _, err := s.keycloakService.CheckSAMLClientConfig(ctx, samlClientID); err != nil {
				return fmt.Errorf("SAML client check failed for %s: %w", req.CspType, err)
			}
		}
	case constants.AuthMethodOIDC:
		// OIDC 클라이언트는 공용 mciam 클라이언트를 사용하므로 별도 확인 불필요
		log.Printf("OIDC client for CSP role %s: using shared mciam OIDC client", role.Name)
	}

	return nil
}

// --- AWS 전용 헬퍼 함수 (private) ---

// awsSamlPolicyValues AWS SAML AssumeRole 정책 문서 템플릿에 채울 값
type awsSamlPolicyValues struct {
	SamlProviderArn string
}

// getAwsSamlAssumeRolePolicyDocument SAML federation용 AssumeRole 정책 문서를 반환합니다.
// SAML Provider ARN(req.IdpIdentifier, 사전에 `aws iam create-saml-provider`로 등록된 값)을
// Federated principal로 사용한다. OIDC와 달리 발급자 정보가 env var로 결정되지 않고
// 호출자가 등록한 SAML Provider 하나하나가 곧 신뢰 대상이므로 req에서 직접 받는다.
func getAwsSamlAssumeRolePolicyDocument(req *model.CreateCspRoleRequest) (string, error) {
	const policyTemplate = `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"Federated": "{{.SamlProviderArn}}"
				},
				"Action": "sts:AssumeRoleWithSAML",
				"Condition": {
					"StringEquals": {
						"SAML:aud": "https://signin.aws.amazon.com/saml"
					}
				}
			}
		]
	}`

	if req.IdpIdentifier == "" {
		return "", fmt.Errorf("idpIdentifier (SAML provider ARN) is required for SAML CSP role creation")
	}

	values := awsSamlPolicyValues{
		SamlProviderArn: req.IdpIdentifier,
	}

	tmpl, err := template.New("samlPolicy").Parse(policyTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, values)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// applyAwsRoleToModel AWS IAM GetRole 응답을 기존 CspRole 레코드에 반영합니다.
// 레코드를 새로 만들지 않으므로 소속 IdP Config·계정·ExtendedConfig 등 저장된 값은 그대로 남습니다.
func applyAwsRoleToModel(role *model.CspRole, idpIdentifier string, getRoleResult *iam.GetRoleOutput) {
	role.Name = aws.ToString(getRoleResult.Role.RoleName)
	if getRoleResult.Role.Description != nil {
		role.Description = *getRoleResult.Role.Description
	}
	if idpIdentifier != "" {
		role.IdpIdentifier = idpIdentifier
	}
	role.Status = "created"
	if getRoleResult.Role.CreateDate != nil {
		role.CreateDate = *getRoleResult.Role.CreateDate
	}
	role.Path = aws.ToString(getRoleResult.Role.Path)
	role.IamRoleId = aws.ToString(getRoleResult.Role.RoleId)
	role.IamIdentifier = aws.ToString(getRoleResult.Role.Arn)
	if getRoleResult.Role.MaxSessionDuration != nil {
		role.MaxSessionDuration = getRoleResult.Role.MaxSessionDuration
	}
	if getRoleResult.Role.PermissionsBoundary != nil && getRoleResult.Role.PermissionsBoundary.PermissionsBoundaryArn != nil {
		role.PermissionsBoundary = *getRoleResult.Role.PermissionsBoundary.PermissionsBoundaryArn
	}
	if getRoleResult.Role.RoleLastUsed != nil {
		roleLastUsed := &model.RoleLastUsed{}
		if getRoleResult.Role.RoleLastUsed.LastUsedDate != nil {
			roleLastUsed.LastUsedDate = *getRoleResult.Role.RoleLastUsed.LastUsedDate
		}
		if getRoleResult.Role.RoleLastUsed.Region != nil {
			roleLastUsed.Region = *getRoleResult.Role.RoleLastUsed.Region
		}
		role.RoleLastUsed = roleLastUsed
	}
	if len(getRoleResult.Role.Tags) > 0 {
		tags := make([]model.Tag, len(getRoleResult.Role.Tags))
		for i, tag := range getRoleResult.Role.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[i] = model.Tag{
					Key:   *tag.Key,
					Value: *tag.Value,
				}
			}
		}
		role.Tags = tags
	}
}

// findActiveAwsOidcConfig 활성 OIDC + AWS IDP Config 1건을 조회합니다. (private)
// getIAMClient(csp_iam_handler.go)와 동일한 IDP Config 기반 패턴을 사용합니다.
func (s *CspRoleService) findActiveAwsOidcConfig() (*model.CspIdpConfig, error) {
	configs, err := s.cspIdpConfigRepo.GetActiveConfigs()
	if err != nil {
		return nil, fmt.Errorf("failed to get active IDP configs: %w", err)
	}
	for _, cfg := range configs {
		if cfg.IsOIDC() && cfg.CspAccount != nil && cfg.CspAccount.CspType == "aws" {
			return cfg, nil
		}
	}
	return nil, fmt.Errorf("no active AWS OIDC IDP config found: please register an OIDC IDP config for an AWS CSP account")
}

// createNewAwsCredential 새로운 AWS 임시 자격 증명을 생성합니다. (private)
// role_arn은 IDP Config DB에서 조회합니다 (getIAMClient와 동일한 패턴).
// IDP Config 조회를 먼저 수행해 설정 누락을 빠르게 감지합니다.
func (s *CspRoleService) createNewAwsCredential(issuedBy string) (*model.TempCredential, error) {
	// 1. IDP Config DB에서 활성 AWS OIDC 설정 조회 (fast-fail)
	idpConfig, err := s.findActiveAwsOidcConfig()
	if err != nil {
		return nil, err
	}

	roleArn := idpConfig.Config["role_arn"]
	if roleArn == "" {
		return nil, fmt.Errorf("role_arn not configured in active AWS OIDC IDP config (id=%d)", idpConfig.ID)
	}

	// oidc_provider_arn은 AssumeRoleWithWebIdentity API의 파라미터가 아님 — 로깅 확인용으로만 사용
	oidcProviderArn := idpConfig.GetOidcProviderArn()
	log.Printf("createNewAwsCredential: using IDP config id=%d, role_arn=%s, oidc_provider_arn=%s", idpConfig.ID, roleArn, oidcProviderArn)

	// 2. Keycloak client credentials 토큰 획득 (WebIdentity 토큰으로 사용)
	token, err := s.keycloakService.GetClientCredentialsToken(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to get Keycloak token: %v", err)
	}

	stsClient := sts.NewFromConfig(aws.Config{
		Region: "ap-northeast-2",
	})

	input := &sts.AssumeRoleWithWebIdentityInput{
		RoleArn:          aws.String(roleArn),
		RoleSessionName:  aws.String("mcmp-iam-manager-session"),
		WebIdentityToken: aws.String(token.AccessToken),
	}

	result, err := stsClient.AssumeRoleWithWebIdentity(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to assume role with web identity: %v", err)
	}

	credential := &model.TempCredential{
		Provider:        "aws",
		AuthType:        "oidc",
		AccessKeyId:     *result.Credentials.AccessKeyId,
		SecretAccessKey: *result.Credentials.SecretAccessKey,
		SessionToken:    *result.Credentials.SessionToken,
		Region:          "ap-northeast-2",
		IssuedAt:        time.Now(),
		ExpiresAt:       *result.Credentials.Expiration,
		IsActive:        true,
		IssuedBy:        issuedBy,
	}

	return credential, nil
}

// createAwsConfigWithTempCredential 임시 자격 증명으로 AWS 설정을 생성합니다. (private)
func (s *CspRoleService) createAwsConfigWithTempCredential(credential *model.TempCredential) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load default AWS config: %v", err)
	}

	cfg.Credentials = aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
		credential.AccessKeyId,
		credential.SecretAccessKey,
		credential.SessionToken,
	))
	cfg.Region = credential.Region

	return cfg, nil
}

// --- 기존 public 메서드 (변경 없음) ---

// GetCspRoles CSP 역할 목록을 조회합니다.
func (s *CspRoleService) GetCspRoles(cspType string) ([]*model.CspRole, error) {
	return s.cspRoleRepo.FindMciamRoleFromCsp(cspType)
}

// GetCspRoleByID ID로 CSP 역할을 조회합니다.
func (s *CspRoleService) GetCspRoleByID(id uint) (*model.CspRole, error) {
	return s.cspRoleRepo.GetRoleByID(id)
}

// UpdateCspRole CSP 역할을 수정합니다.
func (s *CspRoleService) UpdateCspRole(id uint, req *model.CreateCspRoleRequest) error {
	existingRole, err := s.cspRoleRepo.GetRoleByID(id)
	if err != nil {
		return fmt.Errorf("failed to get existing role: %v", err)
	}
	// 관리용 Role 레코드는 IAM 으로 바꾸지 않고, 다른 레코드를 관리용 Role 로 바꾸는 것도 막는다 (409)
	if err := s.GuardCspRoleNotManagementRole(existingRole, "updated"); err != nil {
		return err
	}
	if err := s.GuardNotManagementRole(req.CspRoleName, req.IamIdentifier, "set as a CSP role"); err != nil {
		return err
	}
	if req.CspRoleName != "" {
		existingRole.Name = req.CspRoleName
	}
	if req.Description != "" {
		existingRole.Description = req.Description
	}
	if req.IdpIdentifier != "" {
		existingRole.IdpIdentifier = req.IdpIdentifier
	}
	if req.IamIdentifier != "" {
		existingRole.IamIdentifier = req.IamIdentifier
	}
	if req.IamRoleId != "" {
		existingRole.IamRoleId = req.IamRoleId
	}
	if req.Path != "" {
		existingRole.Path = req.Path
	}
	if len(req.ExtendedConfig) > 0 {
		existingRole.ExtendedConfig = req.ExtendedConfig
	}
	if req.CspIdpConfigID != nil {
		existingRole.CspIdpConfigID = req.CspIdpConfigID
	}
	// DB 전용 업데이트로 변경 (기존에는 CspType 무관하게 항상 AWS UpdateRoleDescription을
	// 호출해 GCP/Alibaba 등 비-AWS CspRole 수정 시 항상 실패했다 — SAML ExtendedConfig를
	// 모든 CSP에서 API로 설정 가능하게 하려면 이 경로도 함께 고쳐야 한다).
	return s.cspRoleRepo.UpdateCspRoleRecord(existingRole)
}

// DeleteCspRole CSP 역할을 삭제합니다.
// DeleteCspRole CspRole(mcmp_role_csp_roles) 레코드를 삭제합니다.
// CreateCspRole과 대칭되는 CspType 분기: AWS는 실제 클라우드 Role도 함께 삭제하고,
// 그 외에는 DB 레코드만 정리합니다. 이 레코드를 참조하는 RoleMasterCspRoleMapping은
// 호출 전에 별도로 정리되어 있어야 한다(핸들러에서 DeleteRoleCspRoleMappingsByCspRoleID 선행 호출).
func (s *CspRoleService) DeleteCspRole(id uint) error {
	role, err := s.cspRoleRepo.GetRoleByID(id)
	if err != nil {
		return fmt.Errorf("failed to find CSP role: %w", err)
	}
	// 관리용 Role 은 IAM 으로 지우지 않는다 (AWS DeleteRole 이 호출되어 IAM 기능이 멈춘다, 409)
	if err := s.GuardCspRoleNotManagementRole(role, "deleted"); err != nil {
		return err
	}

	if role.CspType == "aws" {
		if err := s.deleteAwsIamRole(role); err != nil {
			return err
		}
	}

	return s.cspRoleRepo.DeleteCspRoleRecord(id)
}

// deleteAwsIamRole AWS IAM Role을 삭제합니다. (private, AWS 전용)
func (s *CspRoleService) deleteAwsIamRole(role *model.CspRole) error {
	var cfg *model.CspIdpConfig
	if role.CspIdpConfigID != nil {
		cfg, _ = s.cspIdpConfigRepo.GetByID(*role.CspIdpConfigID)
	}
	api, err := s.awsRoleClientFor(cfg)
	if err != nil {
		return err
	}
	if _, err := api.DeleteRole(context.TODO(), &iam.DeleteRoleInput{RoleName: aws.String(role.Name)}); err != nil {
		// AWS 에서 이미 지워진 Role 이면 DB 레코드만 정리한다 (IAM-BUG-052: 404 로 실패하면 레코드를 지울 방법이 없었다)
		if isAwsNoSuchEntity(err) {
			log.Printf("[WARN] CSP role %s: AWS role does not exist — removing the DB record only", role.Name)
			return nil
		}
		return fmt.Errorf("failed to delete IAM role: %v", err)
	}
	return nil
}

// CleanupExpiredCredentials 만료된 임시 자격 증명을 정리합니다.
func (s *CspRoleService) CleanupExpiredCredentials() error {
	return s.tempCredentialRepo.DeleteExpiredCredentials()
}

// UpdateCSPRole CSP 역할 정보를 수정합니다.
func (s *CspRoleService) UpdateCSPRole(role *model.CspRole) error {
	err := s.cspRoleRepo.UpdateCSPRole(role)
	if err != nil {
		log.Printf("Failed to update CSP role: %v", err)
		return err
	}
	return nil
}

// AddPermissionsToCSPRole CSP 역할에 권한을 추가합니다.
func (s *CspRoleService) AddPermissionsToCSPRole(roleID string, permissions []string) error {
	err := s.cspRoleRepo.AddPermissionsToCSPRole(roleID, permissions)
	if err != nil {
		log.Printf("Failed to add permissions to CSP role: %v", err)
		return err
	}
	return nil
}

// RemovePermissionsFromCSPRole CSP 역할에서 권한을 제거합니다.
func (s *CspRoleService) RemovePermissionsFromCSPRole(roleID string, permissions []string) error {
	err := s.cspRoleRepo.RemovePermissionsFromCSPRole(roleID, permissions)
	if err != nil {
		log.Printf("Failed to remove permissions from CSP role: %v", err)
		return err
	}
	return nil
}

// GetCSPRolePermissions CSP 역할의 권한 목록을 조회합니다.
func (s *CspRoleService) GetCSPRolePermissions(roleID string) ([]string, error) {
	permissions, err := s.cspRoleRepo.GetCSPRolePermissions(roleID)
	if err != nil {
		log.Printf("Failed to get CSP role permissions: %v", err)
		return nil, err
	}
	return permissions, nil
}

// GetRolePolicies 역할의 정책 목록 조회
func (s *CspRoleService) GetRolePolicies(ctx context.Context, roleName string, cspType string) (*model.CspRole, error) {
	role, err := s.cspRoleRepo.GetCspRoleByName(roleName, cspType)
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	managedPolicies, err := s.cspRoleRepo.ListAttachedRolePolicies(ctx, roleName)
	if err != nil {
		return nil, fmt.Errorf("failed to list attached role policies: %w", err)
	}

	inlinePolicies, err := s.cspRoleRepo.ListRolePolicies(ctx, roleName)
	if err != nil {
		return nil, fmt.Errorf("failed to list role policies: %w", err)
	}

	role.Permissions = managedPolicies
	role.Permissions = append(role.Permissions, inlinePolicies...)
	return role, nil
}

// GetRolePolicy 역할의 특정 인라인 정책 조회
func (s *CspRoleService) GetRolePolicy(ctx context.Context, roleName string, policyName string) (*csp.RolePolicy, error) {
	return s.cspRoleRepo.GetRolePolicy(ctx, roleName, policyName)
}

// PutRolePolicy 역할에 인라인 정책 추가/수정
func (s *CspRoleService) PutRolePolicy(ctx context.Context, roleName string, policyName string, policy *csp.RolePolicy) error {
	return s.cspRoleRepo.PutRolePolicy(ctx, roleName, policyName, policy)
}

// DeleteRolePolicy 역할에서 인라인 정책 삭제
func (s *CspRoleService) DeleteRolePolicy(ctx context.Context, roleName string, policyName string) error {
	return s.cspRoleRepo.DeleteRolePolicy(ctx, roleName, policyName)
}

// CreateOrUpdateCspRole CSP 역할을 생성하거나 업데이트합니다.
func (s *CspRoleService) CreateOrUpdateCspRole(req *model.CreateCspRoleRequest) (*model.CspRole, error) {
	if req.ID == "" {
		cspRole, err := s.GetCspRoleByName(req.CspRoleName, req.CspType)
		if err != nil {
			if err != gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("failed to get CSP role by name at CreateOrUpdateCspRole: %w", err)
			}
		}
		if cspRole != nil {
			return cspRole, nil
		}
		return s.CreateCspRole(req)
	}

	id, err := util.StringToUint(req.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ID to uint: %w", err)
	}
	cspRole := &model.CspRole{
		ID:            id,
		Name:          req.CspRoleName,
		Description:   req.Description,
		CspType:       req.CspType,
		IdpIdentifier: req.IdpIdentifier,
		IamIdentifier: req.IamIdentifier,
		Status:        req.Status,
		Path:          req.Path,
		IamRoleId:     req.IamRoleId,
	}
	err = s.UpdateCSPRole(cspRole)
	if err != nil {
		return nil, err
	}
	return cspRole, nil
}

// CreateCspRoles 복수 CSP 역할을 생성합니다.
func (s *CspRoleService) CreateCspRoles(req *model.CreateCspRolesRequest) ([]*model.CspRole, error) {
	var createdRoles []*model.CspRole
	for _, cspRoleReq := range req.CspRoles {
		createdRole, err := s.CreateCspRole(&cspRoleReq)
		if err != nil {
			return nil, fmt.Errorf("failed to create CSP role '%s': %w", cspRoleReq.CspRoleName, err)
		}
		createdRoles = append(createdRoles, createdRole)
	}
	return createdRoles, nil
}

// GetCspRoleByName 이름으로 CSP 역할을 조회합니다.
func (s *CspRoleService) GetCspRoleByName(roleName string, cspType string) (*model.CspRole, error) {
	role, err := s.cspRoleRepo.GetCspRoleByName(roleName, cspType)
	if err != nil {
		return nil, fmt.Errorf("failed to get CSP role by name: %w", err)
	}
	return role, nil
}

// 이름으로 cspRole 목록을 조회합니다. 같은이름의 다른 cspType의 cspRole도 조회합니다.
func (s *CspRoleService) GetCspRolesByName(roleName string, cspType string) ([]*model.CspRole, error) {
	roles, err := s.cspRoleRepo.GetCspRolesByName(roleName)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get CSP role by name at GetCspRolesByName: %w", err)
	}
	return roles, nil
}

// ExistCspRoleByName 이름으로 CSP 역할 존재 여부를 확인합니다 (CspRole 테이블에서)
func (s *CspRoleService) ExistCspRoleByName(roleName string) (bool, error) {
	return s.cspRoleRepo.ExistCspRoleByName(roleName)
}

// ExistCspRoleByNameAndType 이름과 type으로 CSP 역할 존재 여부를 확인합니다 (CspRole 테이블에서)
func (s *CspRoleService) ExistCspRoleByNameAndType(roleName string, cspType string) (bool, error) {
	return s.cspRoleRepo.ExistCspRoleByNameAndType(roleName, cspType)
}
