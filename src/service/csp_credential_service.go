package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound          = errors.New("user not found")
	ErrWorkspaceNotFound     = errors.New("workspace not found")
	ErrNoCspRoleMappingFound = errors.New("no suitable CSP role mapping found for the user's roles in this workspace")
	ErrUnsupportedCspType    = errors.New("unsupported CSP type requested")
)

// CspCredentialService CSP 임시 자격 증명 발급 조율 서비스
type CspCredentialService struct {
	db             *gorm.DB
	userRepo       *repository.UserRepository       // To get user roles
	mappingRepo    *repository.CspMappingRepository // To get CSP role mapping
	awsCredService AwsCredentialService             // To call AWS STS
	// gcpCredService GcpCredentialService             // For future GCP support
	// azureCredService AzureCredentialService           // For future Azure support
	keycloakService KeycloakService // To get KcId from token
}

// NewCspCredentialService 새 CspCredentialService 인스턴스 생성
func NewCspCredentialService(db *gorm.DB) *CspCredentialService {
	userRepo := repository.NewUserRepository(db)
	mappingRepo := repository.NewCspMappingRepository(db)
	awsCredService := NewAwsCredentialService()
	keycloakService := NewKeycloakService()
	return &CspCredentialService{
		db:              db,
		userRepo:        userRepo,
		mappingRepo:     mappingRepo,
		awsCredService:  awsCredService,
		keycloakService: keycloakService,
	}
}

// GetTemporaryCredentials 사용자의 워크스페이스 역할에 기반하여 CSP 임시 자격 증명 발급
func (s *CspCredentialService) GetTemporaryCredentials(ctx context.Context, userID uint, kcUserId string, req *model.CspCredentialRequest) (*model.CspCredentialResponse, error) {

	// 1. Get User's Roles for the specified Workspace
	userWorkspaceRole, err := s.userRepo.FindUserRoleInWorkspace(userID, req.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	// 2. Find the first matching CSP role mapping
	var targetMapping *model.RoleMasterCspRoleMapping
	cspRoleMappings, err := s.mappingRepo.FindCspRoleMappingsByWorkspaceRoleIDAndCspType(userWorkspaceRole.RoleID, req.CspType)
	if err != nil {
		log.Printf("Error finding CSP role mapping for role %d: %v", userWorkspaceRole.RoleID, err)
	}

	if len(cspRoleMappings) == 0 {
		return nil, ErrNoCspRoleMappingFound
	}
	targetMapping = cspRoleMappings[0]

	// 3. IDP ARN 가져오기
	if targetMapping.CspRole == nil {
		return nil, fmt.Errorf("CSP 역할 정보가 없습니다")
	}
	idpArn := targetMapping.CspRole.IdpIdentifier
	if idpArn == "" {
		return nil, fmt.Errorf("IDP ARN이 설정되지 않았습니다")
	}

	// 4. Role ARN 가져오기
	roleArn := targetMapping.CspRole.IamIdentifier
	if roleArn == "" {
		return nil, fmt.Errorf("Role ARN이 설정되지 않았습니다")
	}

	// 5. Get impersonation token for the OIDC client
	impersonationToken, err := s.keycloakService.GetImpersonationTokenByServiceAccount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get impersonation token: %w", err)
	}
	log.Printf("GetImpersonationToken : %v", impersonationToken)

	// 6. Use the impersonation token to get AWS credentials
	switch req.CspType {
	case "aws":
		if s.awsCredService == nil {
			return nil, fmt.Errorf("AWS credential service is not initialized")
		}
		return s.awsCredService.AssumeRoleWithWebIdentity(ctx, roleArn, kcUserId, impersonationToken.AccessToken, idpArn, req.Region)
	case "gcp":
		// TODO: Implement GCP credential logic using gcpCredService
		return nil, ErrUnsupportedCspType
	case "azure":
		// TODO: Implement Azure credential logic using azureCredService
		return nil, ErrUnsupportedCspType
	default:
		return nil, ErrUnsupportedCspType
	}
}

// GetUserWorkspaceRoles 사용자의 워크스페이스 역할 목록 조회
func (s *CspCredentialService) GetUserWorkspaceRoles(userID, workspaceID uint) ([]model.UserWorkspaceRole, error) {
	var roles []model.UserWorkspaceRole
	if err := s.db.Where("user_id = ? AND workspace_id = ?", userID, workspaceID).Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// GetUserWorkspaceRoleIDs 사용자의 워크스페이스 역할 ID 목록 조회
func (s *CspCredentialService) GetUserWorkspaceRoleIDs(userID uint) ([]uint, error) {
	var userWorkspaceRoles []model.UserWorkspaceRole
	if err := s.db.Where("user_id = ?", userID).Find(&userWorkspaceRoles).Error; err != nil {
		return nil, fmt.Errorf("failed to get user workspace roles: %w", err)
	}

	roleIDs := make([]uint, len(userWorkspaceRoles))
	for i, role := range userWorkspaceRoles {
		roleIDs[i] = role.RoleID
	}
	return roleIDs, nil
}

// GetUserWorkspaceRoleNames 사용자의 워크스페이스 역할 이름 목록 조회
func (s *CspCredentialService) GetUserWorkspaceRoleNames(userID uint) ([]string, error) {
	var userWorkspaceRoles []model.UserWorkspaceRole
	if err := s.db.Preload("Role").Where("user_id = ?", userID).Find(&userWorkspaceRoles).Error; err != nil {
		return nil, fmt.Errorf("failed to get user workspace roles: %w", err)
	}

	roleNames := make([]string, len(userWorkspaceRoles))
	for i, role := range userWorkspaceRoles {
		roleNames[i] = role.Role.Name
	}
	return roleNames, nil
}

// GetUserPlatformRoleIDs 사용자의 플랫폼 역할 ID 목록 조회
func (s *CspCredentialService) GetUserPlatformRoleIDs(userID uint) ([]uint, error) {
	var userRoles []model.UserPlatformRole
	if err := s.db.Where("user_id = ?", userID).Find(&userRoles).Error; err != nil {
		return nil, fmt.Errorf("failed to get user platform roles: %w", err)
	}

	roleIDs := make([]uint, len(userRoles))
	for i, role := range userRoles {
		roleIDs[i] = role.RoleID
	}
	return roleIDs, nil
}

// GetUserPlatformRoleNames 사용자의 플랫폼 역할 이름 목록 조회
func (s *CspCredentialService) GetUserPlatformRoleNames(userID uint) ([]string, error) {
	var userRoles []model.UserPlatformRole
	if err := s.db.Preload("Role").Where("user_id = ?", userID).Find(&userRoles).Error; err != nil {
		return nil, fmt.Errorf("failed to get user platform roles: %w", err)
	}

	roleNames := make([]string, len(userRoles))
	for i, role := range userRoles {
		roleNames[i] = role.Role.Name
	}
	return roleNames, nil
}
