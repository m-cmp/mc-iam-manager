package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/m-cmp/mc-iam-manager/util"
	"gorm.io/gorm"
)

// credUserRepo 테스트 주입을 위한 UserRepository 인터페이스
type credUserRepo interface {
	FindUserRoleInWorkspace(userID, workspaceID uint) (*model.UserWorkspaceRole, error)
}

// credMappingRepo 테스트 주입을 위한 CspMappingRepository 인터페이스
type credMappingRepo interface {
	FindCspRoleMappingsByRoleIDAndCspType(roleID uint, cspType string, authMethod string) (*model.RoleMasterCspRoleMapping, error)
}

var (
	ErrUserNotFound           = errors.New("user not found")
	ErrWorkspaceNotFound      = errors.New("workspace not found")
	ErrNoCspRoleMappingFound  = errors.New("no suitable CSP role mapping found for the user's roles in this workspace")
	ErrUnsupportedCspType     = errors.New("unsupported CSP type requested")
	ErrUnsupportedAuthMethod  = errors.New("unsupported auth method for this CSP type")
)

// CspCredentialService CSP 임시 자격 증명 발급 조율 서비스
type CspCredentialService struct {
	db                  *gorm.DB
	userRepo            *repository.UserRepository       // 프로덕션 용
	mappingRepo         *repository.CspMappingRepository // 프로덕션 용
	userRepoIface       credUserRepo                     // 테스트 주입용 (nil이면 userRepo 사용)
	mappingRepoIface    credMappingRepo                  // 테스트 주입용 (nil이면 mappingRepo 사용)
	awsCredService      AwsCredentialService
	gcpCredService      GcpCredentialService
	alibabaCredService  AlibabaCredentialService
	azureCredService    AzureCredentialService
	tencentCredService  TencentCredentialService
	ibmCredService      IbmCredentialService
	keycloakService     KeycloakService
}

// NewCspCredentialService 새 CspCredentialService 인스턴스 생성
func NewCspCredentialService(db *gorm.DB) *CspCredentialService {
	userRepo := repository.NewUserRepository(db)
	mappingRepo := repository.NewCspMappingRepository(db)
	awsCredService := NewAwsCredentialService()
	gcpCredService := NewGcpCredentialService()
	alibabaCredService := NewAlibabaCredentialService()
	azureCredService := NewAzureCredentialService()
	tencentCredService := NewTencentCredentialService()
	ibmCredService := NewIbmCredentialService()
	keycloakService := NewKeycloakService()
	return &CspCredentialService{
		db:                 db,
		userRepo:           userRepo,
		mappingRepo:        mappingRepo,
		awsCredService:     awsCredService,
		gcpCredService:     gcpCredService,
		alibabaCredService: alibabaCredService,
		azureCredService:   azureCredService,
		tencentCredService: tencentCredService,
		ibmCredService:     ibmCredService,
		keycloakService:    keycloakService,
	}
}

// resolveUserRepo 테스트 주입 우선, 없으면 프로덕션 repo 반환
func (s *CspCredentialService) resolveUserRepo() credUserRepo {
	if s.userRepoIface != nil {
		return s.userRepoIface
	}
	return s.userRepo
}

// resolveMappingRepo 테스트 주입 우선, 없으면 프로덕션 repo 반환
func (s *CspCredentialService) resolveMappingRepo() credMappingRepo {
	if s.mappingRepoIface != nil {
		return s.mappingRepoIface
	}
	return s.mappingRepo
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
	userWorkspaceRole, err := s.resolveUserRepo().FindUserRoleInWorkspace(userID, workspaceIDInt)
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

	// 2. Find the first matching CSP role mapping (authMethod 지정 시 해당 방식 매핑만 조회)
	log.Printf("[CSP_CREDENTIAL] Finding CSP role mappings for role %d, csp type %s, authMethod %s", userWorkspaceRole.RoleID, cspType, req.AuthMethod)
	targetMapping, err := s.resolveMappingRepo().FindCspRoleMappingsByRoleIDAndCspType(userWorkspaceRole.RoleID, cspType, req.AuthMethod)
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
		case "aws", "gcp", "alibaba":
			authMethod = model.AuthMethodOIDC
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
			// === 사전 체크 게이트: DB / Keycloak / CSP 설정 상태 확인 ===

			// Check 1: DB — extended_config.saml_client_id 등록 여부
			samlClientAudience := idpArn
			if extConfig, ok := targetCspRole.ExtendedConfig["saml_client_id"].(string); ok && extConfig != "" {
				samlClientAudience = extConfig
			} else {
				defaultClientID := os.Getenv("SAML_CLIENT_ID_AWS")
				return nil, fmt.Errorf("[설정 누락: DB] CspRole(id=%d)에 saml_client_id 미등록. "+
					"조치: UPDATE mcmp_role_csp_roles SET extended_config='{\"saml_client_id\":\"%s\"}' WHERE id=%d",
					targetCspRole.ID, defaultClientID, targetCspRole.ID)
			}
			log.Printf("[CSP_CREDENTIAL] Check 1 PASS: saml_client_id=%s (CspRole %d)", samlClientAudience, targetCspRole.ID)

			// Check 2: Keycloak — SAML 클라이언트 존재 확인
			if _, err := s.keycloakService.CheckSAMLClientConfig(ctx, samlClientAudience); err != nil {
				return nil, fmt.Errorf("[설정 누락: Keycloak] SAML 클라이언트 '%s' 확인 실패: %w. "+
					"조치: (1) Keycloak Clients에서 '%s' SAML 클라이언트 등록 "+
					"(2) token-exchange permission에서 mciam-oidc-Client policy 연결",
					samlClientAudience, err, samlClientAudience)
			}
			log.Printf("[CSP_CREDENTIAL] Check 2 PASS: Keycloak SAML client '%s' 확인", samlClientAudience)

			// Check 3: CSP — AWS SAML Provider 존재 확인 (IAM 읽기 권한 있을 때만 검증, 없으면 경고 후 진행)
			if _, err := s.awsCredService.CheckSAMLProvider(ctx, idpArn); err != nil {
				log.Printf("[CSP_CREDENTIAL] Check 3 WARN: AWS SAML Provider 확인 불가 (IAM 읽기 권한 미보유) — idpArn=%s, err=%v. "+
					"미등록 시 조치: AWS IAM → Identity providers에서 SAML Provider 등록 및 Keycloak metadata XML 업로드", idpArn, err)
			} else {
				log.Printf("[CSP_CREDENTIAL] Check 3 PASS: AWS SAML Provider '%s' 확인", idpArn)
			}

			// 모든 체크 통과 — SAML Assertion 발급 및 STS 호출
			samlAssertion, err := s.keycloakService.GetSamlAssertionByServiceAccount(ctx, samlClientAudience)
			if err != nil {
				log.Printf("[CSP_CREDENTIAL] Error getting SAML assertion for AWS: %v", err)
				return nil, fmt.Errorf("SAML Assertion 발급 실패: %w", err)
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
			log.Printf("[CSP_CREDENTIAL] Calling GCP WIF ExchangeTokenAndImpersonate (OIDC)...")
			// GCP WIF STS는 단일 문자열 aud를 요구하므로 Access Token(aud가 "account")이 아니라
			// ID Token(aud=OIDC 클라이언트 ID)을 사용해야 한다 — Alibaba OIDC(OI-1)와 동일한 이유.
			return s.gcpCredService.ExchangeTokenAndImpersonate(ctx, idpArn, roleArn, impersonationToken.IDToken, "jwt")
		case model.AuthMethodSAML:
			samlClientAudience := idpArn
			if extConfig, ok := targetCspRole.ExtendedConfig["saml_client_id"].(string); ok && extConfig != "" {
				samlClientAudience = extConfig
			}
			samlAssertion, err := s.keycloakService.GetSamlAssertionByServiceAccount(ctx, samlClientAudience)
			if err != nil {
				log.Printf("[CSP_CREDENTIAL] Error getting SAML assertion for GCP: %v", err)
				return nil, fmt.Errorf("failed to get SAML assertion for GCP: %w", err)
			}
			log.Printf("[CSP_CREDENTIAL] Calling GCP WIF ExchangeTokenAndImpersonate (SAML)...")
			return s.gcpCredService.ExchangeTokenAndImpersonate(ctx, idpArn, roleArn, samlAssertion, "saml2")
		case model.AuthMethodSecretKey:
			return getSecretKeyCredentials(cspType, targetCspRole.CspIdpConfig, region)
		default:
			return nil, ErrUnsupportedAuthMethod
		}
	case "alibaba":
		switch authMethod {
		case model.AuthMethodOIDC:
			impersonationToken, err := s.keycloakService.GetImpersonationTokenByServiceAccount(ctx)
			if err != nil {
				log.Printf("[CSP_CREDENTIAL] Error getting impersonation token for Alibaba: %v", err)
				return nil, fmt.Errorf("failed to get impersonation token for Alibaba: %w", err)
			}
			// Alibaba STS requires OIDC ID token (aud = single client_id), not access_token
			oidcToken := impersonationToken.IDToken
			if oidcToken == "" {
				oidcToken = impersonationToken.AccessToken
			}
			audience := ""
			if targetCspRole.CspIdpConfig != nil {
				audience = targetCspRole.CspIdpConfig.Config["audience"]
			}
			log.Printf("[CSP_CREDENTIAL] Calling Alibaba AssumeRoleWithOIDC... (audience=%s)", audience)
			return s.alibabaCredService.AssumeRoleWithOIDC(ctx, idpArn, roleArn, oidcToken, region, audience)
		case model.AuthMethodSAML:
			// === 사전 체크 게이트: DB / Keycloak / CSP 설정 상태 확인 ===

			// Check 1: DB — extended_config.saml_client_id 등록 여부
			samlClientAudience := idpArn
			if extConfig, ok := targetCspRole.ExtendedConfig["saml_client_id"].(string); ok && extConfig != "" {
				samlClientAudience = extConfig
			} else {
				defaultClientID := os.Getenv("SAML_CLIENT_ID_ALIBABA")
				return nil, fmt.Errorf("[설정 누락: DB] CspRole(id=%d)에 saml_client_id 미등록. "+
					"조치: UPDATE mcmp_role_csp_roles SET extended_config='{\"saml_client_id\":\"%s\"}' WHERE id=%d",
					targetCspRole.ID, defaultClientID, targetCspRole.ID)
			}
			log.Printf("[CSP_CREDENTIAL] Check 1 PASS: saml_client_id=%s (CspRole %d)", samlClientAudience, targetCspRole.ID)

			// Check 2: Keycloak — SAML 클라이언트 존재 확인
			if _, err := s.keycloakService.CheckSAMLClientConfig(ctx, samlClientAudience); err != nil {
				return nil, fmt.Errorf("[설정 누락: Keycloak] SAML 클라이언트 '%s' 확인 실패: %w. "+
					"조치: (1) Keycloak Clients에서 '%s' SAML 클라이언트 등록 "+
					"(2) token-exchange permission에서 mciam-oidc-Client policy 연결",
					samlClientAudience, err, samlClientAudience)
			}
			log.Printf("[CSP_CREDENTIAL] Check 2 PASS: Keycloak SAML client '%s' 확인", samlClientAudience)

			// Check 3: CSP — Alibaba SAML Provider 존재 확인 (미구현 시 경고 로그 후 진행)
			log.Printf("[CSP_CREDENTIAL] Check 3 SKIP: Alibaba SAML Provider 확인 미구현 — idpArn=%s", idpArn)

			// 모든 체크 통과 — SAML Assertion 발급 및 STS 호출
			samlAssertion, err := s.keycloakService.GetSamlAssertionByServiceAccount(ctx, samlClientAudience)
			if err != nil {
				log.Printf("[CSP_CREDENTIAL] Error getting SAML assertion for Alibaba: %v", err)
				return nil, fmt.Errorf("SAML Assertion 발급 실패: %w", err)
			}
			log.Printf("[CSP_CREDENTIAL] Calling Alibaba AssumeRoleWithSAML...")
			return s.alibabaCredService.AssumeRoleWithSAML(ctx, idpArn, roleArn, samlAssertion, region)
		case model.AuthMethodSecretKey:
			return getSecretKeyCredentials(cspType, targetCspRole.CspIdpConfig, region)
		default:
			return nil, ErrUnsupportedAuthMethod
		}
	case "azure":
		switch authMethod {
		case model.AuthMethodOIDC:
			tenantID := ""
			clientID := ""
			if targetCspRole.CspIdpConfig != nil {
				tenantID = targetCspRole.CspIdpConfig.Config["tenant_id"]
				clientID = targetCspRole.CspIdpConfig.Config["client_id"]
			}
			if tenantID == "" || clientID == "" {
				return nil, fmt.Errorf("Azure OIDC requires tenant_id and client_id in CspIdpConfig")
			}
			impersonationToken, err := s.keycloakService.GetImpersonationTokenByServiceAccount(ctx)
			if err != nil {
				log.Printf("[CSP_CREDENTIAL] Error getting impersonation token for Azure: %v", err)
				return nil, fmt.Errorf("failed to get impersonation token for Azure: %w", err)
			}
			log.Printf("[CSP_CREDENTIAL] Calling Azure GetTokenByFederatedCredential...")
			return s.azureCredService.GetTokenByFederatedCredential(ctx, tenantID, clientID, impersonationToken.AccessToken)
		case model.AuthMethodSecretKey:
			return getSecretKeyCredentials(cspType, targetCspRole.CspIdpConfig, region)
		default:
			return nil, ErrUnsupportedAuthMethod
		}
	case "tencent":
		switch authMethod {
		case model.AuthMethodOIDC:
			secretID := ""
			secretKey := ""
			if targetCspRole.CspIdpConfig != nil {
				secretID = targetCspRole.CspIdpConfig.Config["secret_id"]
				secretKey = targetCspRole.CspIdpConfig.Config["secret_key"]
			}
			// SAML 경로와 동일하게 secret_id/secret_key 설정을 요구한다. Authorization: SKIP 특성상
			// STS 호출 자체에는 필요 없을 수 있지만(SAML에서 확인된 내용), 일관성을 위해 유지 —
			// 불필요 여부 확인 및 요건 완화는 이 작업 범위 밖.
			if secretID == "" || secretKey == "" {
				return nil, fmt.Errorf("Tencent OIDC requires secret_id and secret_key in CspIdpConfig")
			}
			impersonationToken, err := s.keycloakService.GetImpersonationTokenByServiceAccount(ctx)
			if err != nil {
				log.Printf("[CSP_CREDENTIAL] Error getting impersonation token for Tencent: %v", err)
				return nil, fmt.Errorf("failed to get impersonation token for Tencent: %w", err)
			}
			// GCP/Alibaba OIDC와 동일한 이유로 AccessToken이 아니라 IDToken을 사용해야 한다 —
			// Tencent STS ProviderId="OIDC" 검증은 aud가 등록된 Client ID와 일치하는 ID Token을 요구한다.
			log.Printf("[CSP_CREDENTIAL] Calling Tencent AssumeRoleWithWebIdentity...")
			return s.tencentCredService.AssumeRoleWithWebIdentity(ctx, secretID, secretKey, roleArn, "OIDC", impersonationToken.IDToken, region)
		case model.AuthMethodSAML:
			secretID := ""
			secretKey := ""
			if targetCspRole.CspIdpConfig != nil {
				secretID = targetCspRole.CspIdpConfig.Config["secret_id"]
				secretKey = targetCspRole.CspIdpConfig.Config["secret_key"]
			}
			if secretID == "" || secretKey == "" {
				return nil, fmt.Errorf("Tencent SAML requires secret_id and secret_key in CspIdpConfig")
			}
			samlClientAudience := idpArn
			if extConfig, ok := targetCspRole.ExtendedConfig["saml_client_id"].(string); ok && extConfig != "" {
				samlClientAudience = extConfig
			}
			samlAssertion, err := s.keycloakService.GetSamlAssertionByServiceAccount(ctx, samlClientAudience)
			if err != nil {
				log.Printf("[CSP_CREDENTIAL] Error getting SAML assertion for Tencent: %v", err)
				return nil, fmt.Errorf("failed to get SAML assertion for Tencent: %w", err)
			}
			log.Printf("[CSP_CREDENTIAL] Calling Tencent AssumeRoleWithSAML...")
			return s.tencentCredService.AssumeRoleWithSAML(ctx, secretID, secretKey, roleArn, idpArn, samlAssertion, region)
		case model.AuthMethodSecretKey:
			return getSecretKeyCredentials(cspType, targetCspRole.CspIdpConfig, region)
		default:
			return nil, ErrUnsupportedAuthMethod
		}
	case "ibm":
		switch authMethod {
		case model.AuthMethodOIDC:
			profileID := ""
			if targetCspRole.CspIdpConfig != nil {
				profileID = targetCspRole.CspIdpConfig.Config["profile_id"]
			}
			if profileID == "" {
				return nil, fmt.Errorf("IBM OIDC requires profile_id in CspIdpConfig")
			}
			impersonationToken, err := s.keycloakService.GetImpersonationTokenByServiceAccount(ctx)
			if err != nil {
				log.Printf("[CSP_CREDENTIAL] Error getting impersonation token for IBM: %v", err)
				return nil, fmt.Errorf("failed to get impersonation token for IBM: %w", err)
			}
			log.Printf("[CSP_CREDENTIAL] Calling IBM GetTokenByTrustedProfile...")
			return s.ibmCredService.GetTokenByTrustedProfile(ctx, profileID, impersonationToken.AccessToken)
		case model.AuthMethodSecretKey:
			return getSecretKeyCredentials(cspType, targetCspRole.CspIdpConfig, region)
		default:
			return nil, ErrUnsupportedAuthMethod
		}
	case "ncp", "nhn", "kt", "openstack":
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
