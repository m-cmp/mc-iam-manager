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
	ErrUserNotFound           = errors.New("user not found")
	ErrWorkspaceNotFound      = errors.New("workspace not found")
	ErrNoCspRoleMappingFound  = errors.New("no suitable CSP role mapping found for the user's roles in this workspace")
	ErrUnsupportedCspType     = errors.New("unsupported CSP type requested")
	ErrUnsupportedAuthMethod  = errors.New("unsupported auth method for this CSP type")
)

// CspCredentialService CSP 임시 자격 증명 발급 조율 서비스
type CspCredentialService struct {
	db                     *gorm.DB
	userRepo               *repository.UserRepository       // To get user roles
	mappingRepo            *repository.CspMappingRepository // To get CSP role mapping
	awsCredService         AwsCredentialService             // To call AWS STS
	gcpCredService         GcpCredentialService             // To call GCP WIF
	alibabaCredService     AlibabaCredentialService         // To call Alibaba RAM STS
	keycloakService        KeycloakService                  // To get KcId from token
}

// NewCspCredentialService 새 CspCredentialService 인스턴스 생성
func NewCspCredentialService(db *gorm.DB) *CspCredentialService {
	userRepo := repository.NewUserRepository(db)
	mappingRepo := repository.NewCspMappingRepository(db)
	awsCredService := NewAwsCredentialService()
	gcpCredService := NewGcpCredentialService()
	alibabaCredService := NewAlibabaCredentialService()
	keycloakService := NewKeycloakService()
	return &CspCredentialService{
		db:                 db,
		userRepo:           userRepo,
		mappingRepo:        mappingRepo,
		awsCredService:     awsCredService,
		gcpCredService:     gcpCredService,
		alibabaCredService: alibabaCredService,
		keycloakService:    keycloakService,
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
	if cspType == "" {
		log.Printf("[CSP_CREDENTIAL] Error: csp type is empty")
		return nil, fmt.Errorf("csp type is required")
	}
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
	targetMapping, err := s.mappingRepo.FindCspRoleMappingsByRoleIDAndCspType(userWorkspaceRole.RoleID, cspType)
	if err != nil {
		log.Printf("[CSP_CREDENTIAL] Error finding CSP role mapping for role %d: %v", userWorkspaceRole.RoleID, err)
	}

	if targetMapping == nil {
		log.Printf("[CSP_CREDENTIAL] Error: No CSP role mappings found for role %d and csp type %s", userWorkspaceRole.RoleID, cspType)
		return nil, ErrNoCspRoleMappingFound
	}

	// CspRoles 배열에서 첫 번째 요소를 사용
	if len(targetMapping.CspRoles) == 0 {
		log.Printf("[CSP_CREDENTIAL] Error: No CSP roles found in mapping")
		return nil, fmt.Errorf("CSP 역할 정보가 없습니다")
	}
	targetCspRole := targetMapping.CspRoles[0]
	log.Printf("[CSP_CREDENTIAL] Found CSP role mapping - RoleID: %d, CspRoleID: %d", targetMapping.RoleID, targetCspRole.ID)

	// 3. IDP ARN 가져오기
	if targetCspRole == nil {
		log.Printf("[CSP_CREDENTIAL] Error: CSP role information is nil")
		return nil, fmt.Errorf("CSP 역할 정보가 없습니다")
	}
	idpArn := targetCspRole.IdpIdentifier
	if idpArn == "" {
		log.Printf("[CSP_CREDENTIAL] Error: IDP ARN is empty")
		return nil, fmt.Errorf("IDP ARN이 설정되지 않았습니다")
	}
	log.Printf("[CSP_CREDENTIAL] IDP ARN: %s", idpArn)

	// 4. Role ARN 가져오기
	roleArn := targetCspRole.IamIdentifier
	if roleArn == "" {
		log.Printf("[CSP_CREDENTIAL] Error: Role ARN is empty")
		return nil, fmt.Errorf("Role ARN이 설정되지 않았습니다")
	}
	log.Printf("[CSP_CREDENTIAL] Role ARN: %s", roleArn)

	// 5. Determine auth method from CspIdpConfig (with backward-compat defaults)
	authMethod := model.AuthMethodType("")
	if targetCspRole.CspIdpConfig != nil {
		authMethod = targetCspRole.CspIdpConfig.AuthMethod
	}
	if authMethod == "" {
		switch cspType {
		case "aws", "gcp":
			authMethod = model.AuthMethodOIDC
		case "alibaba":
			authMethod = model.AuthMethodSAML
		}
	}
	log.Printf("[CSP_CREDENTIAL] Auth method resolved: cspType=%s, authMethod=%s", cspType, authMethod)

	// 6. Dispatch by (cspType, authMethod)
	switch cspType {
	case "aws":
		switch authMethod {
		case model.AuthMethodOIDC:
			impersonationToken, err := s.keycloakService.GetImpersonationTokenByServiceAccount(ctx)
			if err != nil {
				log.Printf("[CSP_CREDENTIAL] Error getting impersonation token: %v", err)
				return nil, fmt.Errorf("failed to get impersonation token: %w", err)
			}
			log.Printf("[CSP_CREDENTIAL] Calling AWS AssumeRoleWithWebIdentity...")
			return s.awsCredService.AssumeRoleWithWebIdentity(ctx, roleArn, kcUserId, impersonationToken.AccessToken, idpArn, region)
		case model.AuthMethodSAML:
			samlClientAudience := idpArn
			if extConfig, ok := targetCspRole.ExtendedConfig["saml_client_id"].(string); ok && extConfig != "" {
				samlClientAudience = extConfig
			}
			samlAssertion, err := s.keycloakService.GetSamlAssertionByServiceAccount(ctx, samlClientAudience)
			if err != nil {
				log.Printf("[CSP_CREDENTIAL] Error getting SAML assertion for AWS: %v", err)
				return nil, fmt.Errorf("failed to get SAML assertion for AWS: %w", err)
			}
			log.Printf("[CSP_CREDENTIAL] Calling AWS AssumeRoleWithSAML...")
			return s.awsCredService.AssumeRoleWithSAML(ctx, roleArn, idpArn, samlAssertion, region)
		case model.AuthMethodSecretKey:
			return getSecretKeyCredentials(cspType, targetCspRole.CspIdpConfig, region)
		default:
			return nil, ErrUnsupportedAuthMethod
		}
	case "gcp":
		switch authMethod {
		case model.AuthMethodOIDC:
			impersonationToken, err := s.keycloakService.GetImpersonationTokenByServiceAccount(ctx)
			if err != nil {
				log.Printf("[CSP_CREDENTIAL] Error getting impersonation token: %v", err)
				return nil, fmt.Errorf("failed to get impersonation token: %w", err)
			}
			log.Printf("[CSP_CREDENTIAL] Calling GCP WIF ExchangeTokenAndImpersonate...")
			return s.gcpCredService.ExchangeTokenAndImpersonate(ctx, idpArn, roleArn, impersonationToken.AccessToken)
		case model.AuthMethodSecretKey:
			return getSecretKeyCredentials(cspType, targetCspRole.CspIdpConfig, region)
		default:
			return nil, ErrUnsupportedAuthMethod
		}
	case "alibaba":
		switch authMethod {
		case model.AuthMethodSAML:
			samlClientAudience := idpArn
			if extConfig, ok := targetCspRole.ExtendedConfig["saml_client_id"].(string); ok && extConfig != "" {
				samlClientAudience = extConfig
			}
			samlAssertion, err := s.keycloakService.GetSamlAssertionByServiceAccount(ctx, samlClientAudience)
			if err != nil {
				log.Printf("[CSP_CREDENTIAL] Error getting SAML assertion for Alibaba: %v", err)
				return nil, fmt.Errorf("failed to get SAML assertion for Alibaba: %w", err)
			}
			log.Printf("[CSP_CREDENTIAL] Calling Alibaba AssumeRoleWithSAML...")
			return s.alibabaCredService.AssumeRoleWithSAML(ctx, idpArn, roleArn, samlAssertion, region)
		case model.AuthMethodSecretKey:
			return getSecretKeyCredentials(cspType, targetCspRole.CspIdpConfig, region)
		default:
			return nil, ErrUnsupportedAuthMethod
		}
	case "azure", "tencent", "ibm", "ncp", "nhn", "kt", "openstack":
		switch authMethod {
		case model.AuthMethodSecretKey:
			return getSecretKeyCredentials(cspType, targetCspRole.CspIdpConfig, region)
		default:
			log.Printf("[CSP_CREDENTIAL] %s: federation not yet implemented (authMethod=%s)", cspType, authMethod)
			return nil, ErrUnsupportedAuthMethod
		}
	default:
		log.Printf("[CSP_CREDENTIAL] Error: Unsupported CSP type: %s", cspType)
		return nil, ErrUnsupportedCspType
	}
}

// getSecretKeyCredentials SECRET_KEY 방식: CspIdpConfig에 저장된 키를 직접 반환
func getSecretKeyCredentials(cspType string, idpConfig *model.CspIdpConfig, region string) (*model.CspCredentialResponse, error) {
	if idpConfig == nil {
		return nil, fmt.Errorf("IDP config is not set for SECRET_KEY authentication")
	}
	accessKeyID := idpConfig.GetAccessKeyID()
	secretAccessKey := idpConfig.GetSecretAccessKey()
	if accessKeyID == "" || secretAccessKey == "" {
		return nil, fmt.Errorf("access_key_id or secret_access_key is not configured in IDP config")
	}
	return &model.CspCredentialResponse{
		CspType:         cspType,
		AccessKeyId:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		Region:          region,
	}, nil
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
