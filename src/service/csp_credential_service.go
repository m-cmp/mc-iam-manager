package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm"
)

var (
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
func (s *CspCredentialService) GetTemporaryCredentials(ctx context.Context, kcUserId string, rawOidcToken string, workspaceIDStr string, cspType string, region string) (*model.CspCredentialResponse, error) {
	// 1. Get User's Keycloak ID from OIDC Token
	user, err := s.userRepo.FindByKcID(kcUserId)
	if err != nil {
		log.Printf("Error finding user by KcID %s: %v", kcUserId, err)
		return nil, ErrUserNotFound
	}
	userID := user.ID

	// 2. Parse Workspace ID
	workspaceID_uint64, err := strconv.ParseUint(workspaceIDStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace ID format: %w", err)
	}
	workspaceID := uint(workspaceID_uint64)

	// 3. Get User's Roles for the specified Workspace
	userWorkspaceRoles, err := s.userRepo.GetUserRolesInWorkspace(userID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	// 4. Find the first matching CSP role mapping
	var targetMapping *model.WorkspaceRoleCspRoleMapping
	for _, role := range userWorkspaceRoles {
		mappings, err := s.mappingRepo.FindByRoleAndCspType(role.RoleID, cspType)
		if err != nil {
			log.Printf("Error finding CSP role mapping for role %d: %v", role.RoleID, err)
			continue
		}
		if len(mappings) > 0 {
			targetMapping = &mappings[0]
			break
		}
	}

	if targetMapping == nil {
		return nil, ErrNoCspRoleMappingFound
	}

	// 5. Get impersonation token for the OIDC client
	impersonationToken, err := s.keycloakService.GetImpersonationTokenByServiceAccount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get impersonation token: %w", err)
	}
	log.Printf("GetImpersonationToken : %v", impersonationToken)

	// 6. Use the impersonation token to get AWS credentials
	switch cspType {
	case "aws":
		if s.awsCredService == nil {
			return nil, fmt.Errorf("AWS credential service is not initialized")
		}
		return s.awsCredService.AssumeRoleWithWebIdentity(ctx, targetMapping.CspRoleArn, kcUserId, impersonationToken.AccessToken, targetMapping.IdpIdentifier, region)
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
