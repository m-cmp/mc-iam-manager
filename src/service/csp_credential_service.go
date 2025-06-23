package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/m-cmp/mc-iam-manager/util"
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
	log.Printf("[CSP_CREDENTIAL] Starting GetTemporaryCredentials - UserID: %d, WorkspaceID: %s, CspType: %s", userID, req.WorkspaceID, req.CspType)

	workspaceIDInt, err := util.StringToUint(req.WorkspaceID)
	if err != nil {
		log.Printf("[CSP_CREDENTIAL] Error converting workspace ID: %v", err)
		return nil, fmt.Errorf("invalid workspace ID: %w", err)
	}
	if workspaceIDInt == 0 {
		log.Printf("[CSP_CREDENTIAL] Error: workspace ID is 0")
		return nil, fmt.Errorf("workspace ID is required")
	}

	cspType := req.CspType
	region := req.Region
	log.Printf("[CSP_CREDENTIAL] Parameters - WorkspaceID: %d, CspType: %s, Region: %s", workspaceIDInt, cspType, region)

	// 1. Get User's Roles for the specified Workspace
	log.Printf("[CSP_CREDENTIAL] Getting user roles for workspace...")
	userWorkspaceRole, err := s.userRepo.FindUserRoleInWorkspace(userID, workspaceIDInt)
	if err != nil {
		log.Printf("[CSP_CREDENTIAL] Error finding user role in workspace: %v", err)
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	// Check if userWorkspaceRole is nil (user has no role in this workspace)
	if userWorkspaceRole == nil {
		log.Printf("[CSP_CREDENTIAL] Error: user has no role assigned in workspace %d", workspaceIDInt)
		return nil, fmt.Errorf("user has no role assigned in the specified workspace")
	}
	log.Printf("[CSP_CREDENTIAL] Found user workspace role - RoleID: %d", userWorkspaceRole.RoleID)

	// 2. Find the first matching CSP role mapping
	log.Printf("[CSP_CREDENTIAL] Finding CSP role mappings for role %d and csp type %s", userWorkspaceRole.RoleID, cspType)
	var targetMapping *model.RoleMasterCspRoleMapping
	cspRoleMappings, err := s.mappingRepo.FindCspRoleMappingsByWorkspaceRoleIDAndCspType(userWorkspaceRole.RoleID, cspType)
	if err != nil {
		log.Printf("[CSP_CREDENTIAL] Error finding CSP role mapping for role %d: %v", userWorkspaceRole.RoleID, err)
	}

	if len(cspRoleMappings) == 0 {
		log.Printf("[CSP_CREDENTIAL] Error: No CSP role mappings found for role %d and csp type %s", userWorkspaceRole.RoleID, cspType)
		return nil, ErrNoCspRoleMappingFound
	}
	targetMapping = cspRoleMappings[0]
	log.Printf("[CSP_CREDENTIAL] Found CSP role mapping - RoleID: %d, CspRoleID: %d", targetMapping.RoleID, targetMapping.CspRoleID)

	// 3. IDP ARN 가져오기
	if targetMapping.CspRole == nil {
		log.Printf("[CSP_CREDENTIAL] Error: CSP role information is nil")
		return nil, fmt.Errorf("CSP 역할 정보가 없습니다")
	}
	idpArn := targetMapping.CspRole.IdpIdentifier
	if idpArn == "" {
		log.Printf("[CSP_CREDENTIAL] Error: IDP ARN is empty")
		return nil, fmt.Errorf("IDP ARN이 설정되지 않았습니다")
	}
	log.Printf("[CSP_CREDENTIAL] IDP ARN: %s", idpArn)

	// 4. Role ARN 가져오기
	roleArn := targetMapping.CspRole.IamIdentifier
	if roleArn == "" {
		log.Printf("[CSP_CREDENTIAL] Error: Role ARN is empty")
		return nil, fmt.Errorf("Role ARN이 설정되지 않았습니다")
	}
	log.Printf("[CSP_CREDENTIAL] Role ARN: %s", roleArn)

	// 5. Get impersonation token for the OIDC client
	log.Printf("[CSP_CREDENTIAL] Getting impersonation token...")
	impersonationToken, err := s.keycloakService.GetImpersonationTokenByServiceAccount(ctx)
	if err != nil {
		log.Printf("[CSP_CREDENTIAL] Error getting impersonation token: %v", err)
		return nil, fmt.Errorf("failed to get impersonation token: %w", err)
	}
	log.Printf("[CSP_CREDENTIAL] Successfully got impersonation token: %v", impersonationToken)

	// 6. Use the impersonation token to get AWS credentials
	log.Printf("[CSP_CREDENTIAL] Processing credentials for CSP type: %s", cspType)
	switch cspType {
	case "aws":
		if s.awsCredService == nil {
			log.Printf("[CSP_CREDENTIAL] Error: AWS credential service is nil")
			return nil, fmt.Errorf("AWS credential service is not initialized")
		}
		log.Printf("[CSP_CREDENTIAL] Calling AWS AssumeRoleWithWebIdentity...")
		log.Printf("[CSP_CREDENTIAL] Role ARN: %s", roleArn)
		log.Printf("[CSP_CREDENTIAL] IDP ARN: %s", idpArn)
		log.Printf("[CSP_CREDENTIAL] Region: %s", region)
		log.Printf("[CSP_CREDENTIAL] KcUserId: %s", kcUserId)
		log.Printf("[CSP_CREDENTIAL] Impersonation Token: %s", impersonationToken.AccessToken)
		return s.awsCredService.AssumeRoleWithWebIdentity(ctx, roleArn, kcUserId, impersonationToken.AccessToken, idpArn, region)
	case "gcp":
		log.Printf("[CSP_CREDENTIAL] Error: GCP not supported yet")
		return nil, ErrUnsupportedCspType
	case "azure":
		log.Printf("[CSP_CREDENTIAL] Error: Azure not supported yet")
		// TODO: Implement Azure credential logic using azureCredService
		return nil, ErrUnsupportedCspType
	default:
		log.Printf("[CSP_CREDENTIAL] Error: Unsupported CSP type: %s", cspType)
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
