package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

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
func (s *CspCredentialService) GetTemporaryCredentials(ctx context.Context, kcUserId string, rawOidcToken string, workspaceIDStr string, cspType string) (*model.CspCredentialResponse, error) { // Updated signature

	// 1. Get User's Keycloak ID from OIDC Token - Now passed as argument (kcUserId)
	// We still need the rawOidcToken for the STS call
	// Assuming the handler passes the raw token string
	// Let's adjust the signature later if needed, for now assume we have kcUserId
	// For now, let's assume we get kcUserId directly or via another service call using the token
	// This part needs refinement based on how the token is passed and validated upstream.
	// Placeholder: Get kcUserId (subject) from the validated token claims passed via context?
	// claims := ctx.Value("token_claims").(*jwt.MapClaims) // Example if claims are in context
	// kcUserId := (*claims)["sub"].(string)
	// For now, let's assume kcUserId is passed directly for simplicity of this function's logic
	// We need to get the DB user ID first based on kcUserId
	// Let's refine this: Handler should get kcUserId and pass it here.
	// Let's modify the function signature for now.

	// --- Refined Function Signature (Example) ---
	// func (s *CspCredentialService) GetTemporaryCredentials(ctx context.Context, kcUserId string, rawOidcToken string, workspaceIDStr string, cspType string) (*model.CspCredentialResponse, error) {

	// --- Assuming Handler passes kcUserId and rawOidcToken ---

	// 2. Find local user DB ID
	user, err := s.userRepo.FindByKcID(kcUserId)
	if err != nil {
		log.Printf("Error finding user by KcID %s: %v", kcUserId, err)
		return nil, ErrUserNotFound
	}
	userID := user.ID

	// 3. Parse Workspace ID
	workspaceID_uint64, err := strconv.ParseUint(workspaceIDStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace ID format: %w", err)
	}
	workspaceID := uint(workspaceID_uint64)

	// 4. Get User's Roles for the specified Workspace
	userRoles, err := s.userRepo.GetUserRolesInWorkspace(userID, workspaceID)
	if err != nil {
		log.Printf("Error getting user %d roles in workspace %d: %v", userID, workspaceID, err)
		return nil, fmt.Errorf("failed to get user roles for workspace: %w", err)
	}
	if len(userRoles) == 0 {
		log.Printf("User %d has no roles assigned in workspace %d", userID, workspaceID)
		return nil, fmt.Errorf("user has no roles in the specified workspace")
	}

	// 5. Find CSP Role Mapping for the user's roles and requested CSP type
	//    Strategy: Find the first mapping associated with any of the user's roles.
	//    Alternative: Define priority or allow client to specify which role mapping to use.
	var targetMapping *model.WorkspaceRoleCspRoleMapping
	for _, role := range userRoles {
		mappings, err := s.mappingRepo.FindByRoleAndCspType(role.WorkspaceRoleID, cspType)
		if err != nil {
			log.Printf("Error finding CSP mapping for role %d, csp %s: %v", role.WorkspaceRoleID, cspType, err)
			continue // Try next role
		}
		if len(mappings) > 0 {
			targetMapping = &mappings[0] // Use the first mapping found
			log.Printf("Found CSP mapping for user %d, workspace %d, role %d: %s", userID, workspaceID, role.WorkspaceRoleID, targetMapping.CspRoleArn)
			break
		}
	}

	if targetMapping == nil {
		log.Printf("No CSP mapping found for user %d in workspace %d for CSP type %s", userID, workspaceID, cspType)
		return nil, ErrNoCspRoleMappingFound
	}

	// 6. Call the appropriate CSP Credential Service
	switch cspType {
	case "aws":
		if s.awsCredService == nil {
			return nil, fmt.Errorf("AWS credential service is not initialized")
		}
		// RoleSessionName needs to be unique per session, using user ID + timestamp
		roleSessionName := fmt.Sprintf("mciam-%d-%d", userID, time.Now().Unix())
		if len(roleSessionName) > 64 {
			roleSessionName = roleSessionName[:64]
		}
		return s.awsCredService.AssumeRoleWithWebIdentity(ctx, targetMapping.CspRoleArn, kcUserId, rawOidcToken, targetMapping.IdpIdentifier)
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
