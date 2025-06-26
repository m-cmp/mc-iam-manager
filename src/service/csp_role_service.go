package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/m-cmp/mc-iam-manager/csp"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
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
func (s *CspRoleService) CreateCspRole(req *model.CreateCspRoleRequest) (*model.CspRole, error) {
	// 1. 유효한 임시 자격 증명이 있는지 확인하고, 없으면 새로 생성
	issuedBy := "system" // cspRole 생성은 system만 한다.
	credential, err := s.tempCredentialRepo.GetOrCreateValidCredential("aws", "oidc", "ap-northeast-2", nil, issuedBy, func() (*model.TempCredential, error) {
		return s.createNewAwsCredential(issuedBy)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get or create valid credential: %v", err)
	}

	// 2. 임시 자격 증명으로 AWS IAM 클라이언트 생성
	awsCfg, err := s.createAwsConfigWithTempCredential(credential)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config with temp credential: %v", err)
	}
	iamClient := iam.NewFromConfig(awsCfg)

	// 3. CspRoleRepository의 내부 메서드 호출 (임시 자격 증명 관리 로직 제거)
	return s.cspRoleRepo.CreateCspRoleWithIamClient(req, iamClient)
}

// createNewAwsCredential 새로운 AWS 임시 자격 증명을 생성합니다.
func (s *CspRoleService) createNewAwsCredential(issuedBy string) (*model.TempCredential, error) {
	// Keycloak에서 OIDC 토큰 획득
	token, err := s.keycloakService.GetClientCredentialsToken(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to get Keycloak token: %v", err)
	}

	// AWS STS AssumeRoleWithWebIdentity 호출
	stsClient := sts.NewFromConfig(aws.Config{
		Region: "ap-northeast-2",
	})

	// AWS Identity Provider ARN (환경 변수에서 가져오기)
	identityProviderArn := os.Getenv("IDENTITY_PROVIDER_ARN_AWS")
	if identityProviderArn == "" {
		return nil, fmt.Errorf("IDENTITY_PROVIDER_ARN_AWS environment variable is not set")
	}

	// Role ARN (환경 변수에서 가져오기)
	roleArn := os.Getenv("IDENTITY_ROLE_ARN_AWS")
	if roleArn == "" {
		return nil, fmt.Errorf("IDENTITY_ROLE_ARN_AWS environment variable is not set")
	}

	// AssumeRoleWithWebIdentity 호출
	input := &sts.AssumeRoleWithWebIdentityInput{
		RoleArn:          aws.String(roleArn),
		RoleSessionName:  aws.String("mcmp-iam-manager-session"),
		WebIdentityToken: aws.String(token.AccessToken),
	}

	result, err := stsClient.AssumeRoleWithWebIdentity(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to assume role with web identity: %v", err)
	}

	// 임시 자격 증명 모델 생성
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

// createAwsConfigWithTempCredential 임시 자격 증명으로 AWS 설정을 생성합니다.
func (s *CspRoleService) createAwsConfigWithTempCredential(credential *model.TempCredential) (aws.Config, error) {
	// AWS SDK v2 설정 생성
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load default AWS config: %v", err)
	}

	// 임시 자격 증명으로 설정 업데이트
	cfg.Credentials = aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
		credential.AccessKeyId,
		credential.SecretAccessKey,
		credential.SessionToken,
	))

	// 리전 설정
	cfg.Region = credential.Region

	return cfg, nil
}

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
	// ID로 기존 역할 조회
	existingRole, err := s.cspRoleRepo.GetRoleByID(id)
	if err != nil {
		return fmt.Errorf("failed to get existing role: %v", err)
	}

	// 요청 데이터로 업데이트
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
func (s *CspRoleService) GetRolePolicies(ctx context.Context, roleName string) (*model.CspRole, error) {
	// 1. 역할 존재 여부 확인
	role, err := s.cspRoleRepo.GetRoleByName(roleName)
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	// 2. 관리형 정책 목록 조회
	managedPolicies, err := s.cspRoleRepo.ListAttachedRolePolicies(ctx, roleName)
	if err != nil {
		return nil, fmt.Errorf("failed to list attached role policies: %w", err)
	}

	// 3. 인라인 정책 목록 조회
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
// ID가 비어있으면 새로 생성하고, ID가 있으면 기존 것을 업데이트합니다.
func (s *CspRoleService) CreateOrUpdateCspRole(req *model.CreateCspRoleRequest) (*model.CspRole, error) {
	if req.ID == 0 {
		// ID가 비어있으면 새로 생성

		// if constants.CSPTypeAWS == constants.CSPType(req.CspType) {
		// 	//req.RoleName = constants.CspRoleNamePrefix + req.RoleName

		// 	idpIdentifier := "arn:aws:iam::050864702683:oidc-provider/mciambase.onecloudcon.com/realms/mciam-demo"
		// 	iamIdentifier := "arn:aws:iam::050864702683:role/" + constants.CspRoleNamePrefix + req.RoleName

		// 	req.IdpIdentifier = idpIdentifier
		// 	req.IamIdentifier = iamIdentifier
		// }

		return s.CreateCspRole(req)
	} else {
		// ID가 있으면 업데이트
		cspRole := &model.CspRole{
			ID:            req.ID,
			Name:          req.CspRoleName,
			Description:   req.Description,
			CspType:       req.CspType,
			IdpIdentifier: req.IdpIdentifier,
			IamIdentifier: req.IamIdentifier,
			Status:        req.Status,
			Path:          req.Path,
			IamRoleId:     req.IamRoleId,
		}
		err := s.UpdateCSPRole(cspRole)
		if err != nil {
			return nil, err
		}
		return cspRole, nil
	}
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
func (s *CspRoleService) GetCspRoleByName(roleName string) (*model.CspRole, error) {
	role, err := s.cspRoleRepo.GetRoleByName(roleName)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 역할이 존재하지 않음
		}
		return nil, fmt.Errorf("failed to get CSP role by name: %w", err)
	}
	return role, nil
}

// ExistCspRoleByName 이름으로 CSP 역할 존재 여부를 확인합니다 (CspRole 테이블에서)
func (s *CspRoleService) ExistCspRoleByName(roleName string) (bool, error) {
	return s.cspRoleRepo.ExistCspRoleByName(roleName)
}
