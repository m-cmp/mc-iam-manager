package service

import (
	"bytes"
	"context"
	"encoding/json"
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
	mciamConfig "github.com/m-cmp/mc-iam-manager/config"
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
	keycloakService    KeycloakService
}

// NewCspRoleService 새 CspRoleService 인스턴스 생성
func NewCspRoleService(db *gorm.DB, keycloakService KeycloakService) *CspRoleService {
	return &CspRoleService{
		db:                 db,
		cspRoleRepo:        repository.NewCspRoleRepository(db),
		tempCredentialRepo: repository.NewTempCredentialRepository(db),
		keycloakService:    keycloakService,
	}
}

// GetAllCSPRoles 모든 CSP 역할을 조회합니다.
func (s *CspRoleService) GetAllCSPRoles(ctx context.Context, cspType string) ([]*model.CspRole, error) {
	roles, err := s.cspRoleRepo.FindAll()
	if err != nil {
		log.Printf("Failed to get CSP roles: %v", err)
		return nil, err
	}

	return roles, nil
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

	// 2. DB 중복 확인
	existing, err := s.cspRoleRepo.GetCspRoleByName(req.CspRoleName, req.CspType)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing CSP role: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	// 3. CSP별 역할 생성 디스패치
	var cspRole *model.CspRole
	switch req.CspType {
	case "aws":
		cspRole, err = s.createAwsIamRole(req)
	default:
		// AWS 이외 CSP: DB 등록만 수행 (클라우드 역할 생성은 향후 확장)
		cspRole = &model.CspRole{
			Name:        req.CspRoleName,
			Description: req.Description,
			CspType:     req.CspType,
			Status:      "created",
		}
		if err := s.cspRoleRepo.CreateCspRoleRecord(cspRole); err != nil {
			return nil, fmt.Errorf("failed to create CSP role record: %w", err)
		}
	}
	if err != nil {
		return nil, err
	}

	// 4. Keycloak 클라이언트 확인
	if kcErr := s.ensureKeycloakClientForCspRole(req, cspRole); kcErr != nil {
		log.Printf("Warning: Keycloak client check failed for CSP role %s: %v", cspRole.Name, kcErr)
	}

	return cspRole, nil
}

// createAwsIamRole AWS IAM 역할을 생성합니다. (private, AWS 전용)
func (s *CspRoleService) createAwsIamRole(req *model.CreateCspRoleRequest) (*model.CspRole, error) {
	// 1. 임시 자격 증명 획득
	issuedBy := "system"
	credential, err := s.tempCredentialRepo.GetOrCreateValidCredential("aws", "oidc", "ap-northeast-2", nil, issuedBy, func() (*model.TempCredential, error) {
		return s.createNewAwsCredential(issuedBy)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get or create valid credential: %v", err)
	}

	// 2. AWS IAM 클라이언트 생성
	awsCfg, err := s.createAwsConfigWithTempCredential(credential)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config with temp credential: %v", err)
	}
	awsIamClient := iam.NewFromConfig(awsCfg)

	// 3. DB에 초기 레코드 생성
	newRole := &model.CspRole{
		Name:    req.CspRoleName,
		CspType: req.CspType,
	}

	// AWS에서 역할 존재 여부 확인
	getRoleInput := &iam.GetRoleInput{RoleName: aws.String(req.CspRoleName)}
	var targetCspRole *iam.GetRoleOutput
	if cspRoleResponse, getErr := awsIamClient.GetRole(context.TODO(), getRoleInput); getErr == nil {
		newRole.Status = "created"
		targetCspRole = cspRoleResponse
	} else {
		newRole.Status = "creating"
	}

	if err := s.cspRoleRepo.CreateCspRoleRecord(newRole); err != nil {
		return nil, fmt.Errorf("failed to create CSP role in database: %v", err)
	}

	if newRole.Status == "creating" {
		// 4. AssumeRolePolicyDocument 생성 및 AWS IAM Role 생성
		return s.createAndPollAwsIamRole(awsIamClient, newRole, req)
	}

	// 이미 AWS에 존재하는 경우 정보 동기화
	return s.syncAwsRoleToDb(newRole, targetCspRole)
}

// createAndPollAwsIamRole AWS IAM Role을 생성하고 polling으로 완료를 대기합니다. (private)
func (s *CspRoleService) createAndPollAwsIamRole(awsIamClient *iam.Client, newRole *model.CspRole, req *model.CreateCspRoleRequest) (*model.CspRole, error) {
	idpIdentifier := ""

	assumeRolePolicyDocument, err := getAwsAssumeRolePolicyDocument(newRole)
	if err != nil {
		newRole.Status = "failed"
		s.cspRoleRepo.UpdateCspRoleRecord(newRole)
		return nil, fmt.Errorf("failed to generate assume role policy document: %v", err)
	}

	// IdP identifier 추출
	var policyDoc map[string]interface{}
	if err := json.Unmarshal([]byte(assumeRolePolicyDocument), &policyDoc); err != nil {
		newRole.Status = "failed"
		s.cspRoleRepo.UpdateCspRoleRecord(newRole)
		return nil, fmt.Errorf("failed to parse assume role policy document: %v", err)
	}
	if statements, ok := policyDoc["Statement"].([]interface{}); ok && len(statements) > 0 {
		if statement, ok := statements[0].(map[string]interface{}); ok {
			if principal, ok := statement["Principal"].(map[string]interface{}); ok {
				if federated, ok := principal["Federated"].(string); ok {
					idpIdentifier = federated
				}
			}
		}
	}

	// AWS IAM Role 생성
	input := &iam.CreateRoleInput{
		RoleName:                 aws.String(req.CspRoleName),
		AssumeRolePolicyDocument: aws.String(assumeRolePolicyDocument),
		Description:              aws.String(newRole.Description),
	}
	_, err = awsIamClient.CreateRole(context.TODO(), input)
	if err != nil {
		newRole.Status = "failed"
		s.cspRoleRepo.UpdateCspRoleRecord(newRole)
		return nil, fmt.Errorf("failed to create IAM role: %v", err)
	}

	// Polling: 역할 생성 완료 대기
	getRoleInput := &iam.GetRoleInput{RoleName: aws.String(req.CspRoleName)}
	for i := 0; i < 30; i++ {
		time.Sleep(1 * time.Second)
		getRoleResult, err := awsIamClient.GetRole(context.TODO(), getRoleInput)
		if err == nil && getRoleResult != nil && getRoleResult.Role != nil {
			createdRole := mapAwsRoleToModel(newRole.ID, newRole.CspType, idpIdentifier, getRoleResult)
			if err := s.cspRoleRepo.UpdateCspRoleRecord(createdRole); err != nil {
				return nil, fmt.Errorf("failed to update CSP role in database: %v", err)
			}
			return createdRole, nil
		}
	}

	newRole.Status = "failed"
	s.cspRoleRepo.UpdateCspRoleRecord(newRole)
	return nil, fmt.Errorf("failed to verify IAM role creation after 30 attempts")
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

type awsPolicyValues struct {
	AccountID        string
	KeycloakHostname string
	Subject          string
	Audience         string
}

// getAwsAssumeRolePolicyDocument 플랫폼 관리자용 AssumeRole 정책 문서를 반환합니다.
func getAwsAssumeRolePolicyDocument(role *model.CspRole) (string, error) {
	const policyTemplate = `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::{{.AccountID}}:oidc-provider/{{.KeycloakHostname}}"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"{{.KeycloakHostname}}:aud": "{{.Audience}}"
					}
				}
			}
		]
	}`

	oidcClientID := os.Getenv("KEYCLOAK_OIDC_CLIENT_ID")
	if oidcClientID == "" {
		return "", fmt.Errorf("KEYCLOAK_OIDC_CLIENT environment variable is not set")
	}

	values := awsPolicyValues{
		AccountID:        "050864702683",
		KeycloakHostname: mciamConfig.KC.Host,
		Audience:         oidcClientID,
	}

	tmpl, err := template.New("policy").Parse(policyTemplate)
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

// mapAwsRoleToModel AWS IAM GetRole 응답을 CspRole 모델로 매핑합니다.
func mapAwsRoleToModel(id uint, cspType string, idpIdentifier string, getRoleResult *iam.GetRoleOutput) *model.CspRole {
	createdRole := &model.CspRole{
		ID:            id,
		Name:          *getRoleResult.Role.RoleName,
		Description:   *getRoleResult.Role.Description,
		CspType:       cspType,
		IdpIdentifier: idpIdentifier,
		Status:        "created",
		CreateDate:    *getRoleResult.Role.CreateDate,
		Path:          *getRoleResult.Role.Path,
		IamRoleId:     *getRoleResult.Role.RoleId,
		IamIdentifier: *getRoleResult.Role.Arn,
	}
	if getRoleResult.Role.MaxSessionDuration != nil {
		createdRole.MaxSessionDuration = getRoleResult.Role.MaxSessionDuration
	}
	if getRoleResult.Role.PermissionsBoundary != nil && getRoleResult.Role.PermissionsBoundary.PermissionsBoundaryArn != nil {
		createdRole.PermissionsBoundary = *getRoleResult.Role.PermissionsBoundary.PermissionsBoundaryArn
	}
	if getRoleResult.Role.RoleLastUsed != nil {
		roleLastUsed := &model.RoleLastUsed{}
		if getRoleResult.Role.RoleLastUsed.LastUsedDate != nil {
			roleLastUsed.LastUsedDate = *getRoleResult.Role.RoleLastUsed.LastUsedDate
		}
		if getRoleResult.Role.RoleLastUsed.Region != nil {
			roleLastUsed.Region = *getRoleResult.Role.RoleLastUsed.Region
		}
		createdRole.RoleLastUsed = roleLastUsed
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
		createdRole.Tags = tags
	}
	return createdRole
}

// createNewAwsCredential 새로운 AWS 임시 자격 증명을 생성합니다. (private)
func (s *CspRoleService) createNewAwsCredential(issuedBy string) (*model.TempCredential, error) {
	token, err := s.keycloakService.GetClientCredentialsToken(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to get Keycloak token: %v", err)
	}

	stsClient := sts.NewFromConfig(aws.Config{
		Region: "ap-northeast-2",
	})

	identityProviderArn := os.Getenv("IDENTITY_PROVIDER_ARN_AWS")
	if identityProviderArn == "" {
		return nil, fmt.Errorf("IDENTITY_PROVIDER_ARN_AWS environment variable is not set")
	}

	roleArn := os.Getenv("IDENTITY_ROLE_ARN_AWS")
	if roleArn == "" {
		return nil, fmt.Errorf("IDENTITY_ROLE_ARN_AWS environment variable is not set")
	}

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
	existingRole.Name = req.CspRoleName
	existingRole.Description = req.Description
	return s.cspRoleRepo.UpdateCSPRole(existingRole)
}

// DeleteCspRole CSP 역할을 삭제합니다.
func (s *CspRoleService) DeleteCspRole(id uint) error {
	return s.cspRoleRepo.DeleteCSPRole(fmt.Sprintf("%d", id))
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

// DeleteCSPRole CSP 역할을 삭제합니다.
func (s *CspRoleService) DeleteCSPRole(id string) error {
	err := s.cspRoleRepo.DeleteCSPRole(id)
	if err != nil {
		log.Printf("Failed to delete CSP role: %v", err)
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
