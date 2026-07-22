package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/Nerzal/gocloak/v13"
	"github.com/golang-jwt/jwt/v5"
	"github.com/m-cmp/mc-iam-manager/config"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository" // For ErrUserNotFound potentially
)

// KeycloakService defines operations related to Keycloak interaction.
type KeycloakService interface {
	KeycloakAdminLogin(ctx context.Context) (*gocloak.JWT, error)
	GetUser(ctx context.Context, kcId string) (*gocloak.User, error)
	GetUserByUsername(ctx context.Context, username string) (*gocloak.User, error)
	GetUsers(ctx context.Context, enabled *bool) ([]*gocloak.User, error)
	CreateUser(ctx context.Context, user *model.User) (string, error)
	UpdateUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, kcId string) error
	EnableUser(ctx context.Context, kcUserID string) error
	DisableUser(ctx context.Context, kcUserID string) error
	CheckAdminLogin(ctx context.Context) (bool, error)
	CheckRealm(ctx context.Context) (bool, error)
	CreateRealm(ctx context.Context, accessToken string) (bool, error)
	ExistRealm(ctx context.Context, accessToken string) (bool, error)
	ExistClient(ctx context.Context, accessToken string) (bool, error)
	CheckClient(ctx context.Context) (bool, error)
	CreateClient(ctx context.Context, accessToken string) (bool, error)
	GetCerts(ctx context.Context) (*gocloak.CertResponse, error)
	GetUserIDFromToken(ctx context.Context, token *gocloak.JWT) (string, error)
	Login(ctx context.Context, username, password string) (*gocloak.JWT, error)
	RefreshToken(ctx context.Context, refreshToken string) (*gocloak.JWT, error)
	// Methods for group synchronization
	EnsureGroupExistsAndAssignUser(ctx context.Context, kcUserId, groupName string) error
	RemoveUserFromGroup(ctx context.Context, kcUserId, groupName string) error
	// Method for UMA RPT Token
	GetRequestingPartyToken(ctx context.Context, accessToken string, options gocloak.RequestingPartyTokenOptions) (*gocloak.JWT, error)
	// Method to validate token and get claims
	ValidateTokenAndGetClaims(ctx context.Context, token string) (*jwt.MapClaims, error)
	// SetupInitialKeycloakAdmin creates the initial platform admin user and sets up necessary permissions
	SetupInitialKeycloakAdmin(ctx context.Context, adminToken *gocloak.JWT) (string, error)
	// CheckUserRoles checks and logs all roles assigned to a user
	CheckUserRoles(ctx context.Context, username string) error
	// GetUserPermissions gets all permissions for the given roles
	GetUserPermissions(ctx context.Context, roles []string) ([]string, error)
	// GetImpersonationToken gets an impersonation token for a user
	GetImpersonationToken(ctx context.Context) (*gocloak.JWT, error)
	// GetImpersonationTokenByAdminToken: adminToken을 이용해 특정 사용자의 impersonation 토큰을 발급
	GetImpersonationTokenByAdminToken(ctx context.Context, userID string, targetClientID string) (string, error)
	// GetImpersonationTokenByServiceAccount: 서비스 계정을 이용해 특정 클라이언트에 로그인한 토큰을 발급
	GetImpersonationTokenByServiceAccount(ctx context.Context) (*gocloak.JWT, error)
	// GetSamlAssertionByServiceAccount: RFC 8693 토큰 교환으로 SAML2 assertion을 발급 (Alibaba SAML 연동용)
	GetSamlAssertionByServiceAccount(ctx context.Context, samlClientAudience string) (string, error)
	// AssignRealmRoleToUser assigns a realm role to a user
	AssignRealmRoleToUser(ctx context.Context, kcUserId, roleName string) error
	// CheckRealmRoleExists checks if a realm role exists
	CheckRealmRoleExists(ctx context.Context, roleName string) (bool, error)
	// CreateRealmRole creates a realm role
	CreateRealmRole(ctx context.Context, roleName string) error
	// CreateRealmRoleAndWait creates a realm role and waits for it to be available
	CreateRealmRoleAndWait(ctx context.Context, roleName string) error
	// RemoveRealmRoleFromUser removes a realm role from a user
	RemoveRealmRoleFromUser(ctx context.Context, kcUserId, roleName string) error
	// IsRealmRoleAssignedToUser checks if a realm role is already assigned to a user
	IsRealmRoleAssignedToUser(ctx context.Context, kcUserId, roleName string) (bool, error)
	// IssueWorkspaceTicket 워크스페이스 티켓을 발행합니다.
	IssueWorkspaceTicket(ctx context.Context, kcUserId string, workspaceID uint) (string, map[string]interface{}, error)
	// 기본 Role 정의
	SetupPredefinedRoles(ctx context.Context, accessToken string) error
	// GetClientCredentialsToken 클라이언트 자격 증명으로 토큰을 발급받습니다.
	GetClientCredentialsToken(ctx context.Context) (*gocloak.JWT, error)
	// CreatePendingUser creates a user in pending state (enabled=false) with password
	CreatePendingUser(ctx context.Context, req *model.SignupRequest) (string, error)
	// ResetPassword resets a user's password
	ResetPassword(ctx context.Context, kcUserID, newPassword string) error
	// AddRealmRoleToGroup adds a realm role to a Keycloak group (creates group if not exists)
	AddRealmRoleToGroup(ctx context.Context, groupName, roleName string) error
	// RemoveRealmRoleFromGroup removes a realm role from a Keycloak group
	RemoveRealmRoleFromGroup(ctx context.Context, groupName, roleName string) error
	// DeleteGroup deletes a Keycloak group by name (no-op if the group doesn't exist)
	DeleteGroup(ctx context.Context, groupName string) error
	// CheckSAMLClientConfig Keycloak SAML 클라이언트 존재 및 protocol mapper 구성 확인
	CheckSAMLClientConfig(ctx context.Context, clientID string) (string, error)
}

// keycloakService is now stateless, methods directly use config.KC
type keycloakService struct {
}

// NewKeycloakService creates a new stateless KeycloakService.
func NewKeycloakService() KeycloakService {

	return &keycloakService{}
}

func (s *keycloakService) KeycloakAdminLogin(ctx context.Context) (*gocloak.JWT, error) {
	// 1. Admin 로그인
	log.Printf("[DEBUG] Attempting to login as admin")
	adminToken, err := config.KC.LoginAdmin(ctx)
	if err != nil {
		return nil, fmt.Errorf("admin login failed: %w", err)
	}

	return adminToken, nil
}

// GetUser retrieves a user from Keycloak by their Keycloak ID.
func (s *keycloakService) GetUser(ctx context.Context, kcId string) (*gocloak.User, error) {
	if config.KC == nil || config.KC.Client == nil {
		return nil, fmt.Errorf("keycloak configuration not initialized")
	}
	token, err := config.KC.GetAdminToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin token: %w", err)
	}
	user, err := config.KC.Client.GetUserByID(ctx, token.AccessToken, config.KC.Realm, kcId)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, fmt.Errorf("user not found in keycloak (kcId: %s): %w", kcId, repository.ErrUserNotFound)
		}
		return nil, fmt.Errorf("failed to get user from keycloak (kcId: %s): %w", kcId, err)
	}
	return user, nil
}

// GetUserByUsername retrieves a user from Keycloak by username.
func (s *keycloakService) GetUserByUsername(ctx context.Context, username string) (*gocloak.User, error) {
	if config.KC == nil || config.KC.Client == nil {
		return nil, fmt.Errorf("keycloak configuration not initialized")
	}
	token, err := config.KC.GetAdminToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin token: %v", err)
	}
	users, err := config.KC.Client.GetUsers(ctx, token.AccessToken, config.KC.Realm, gocloak.GetUsersParams{
		Username: gocloak.StringP(username),
		Exact:    gocloak.BoolP(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user from keycloak by username %s: %w", username, err)
	}
	if len(users) == 0 {
		return nil, repository.ErrUserNotFound
	}
	if len(users) > 1 {
		log.Printf("Warning: Found multiple users with username %s in Keycloak", username)
	}
	return users[0], nil
}

// GetUsers retrieves users from Keycloak, optionally filtered by enabled status.
// gocloak.GetUsersParams.Enabled uses json:"omitempty" which drops false values,
// so we pass enabled as a manual query param to avoid the omitempty bug.
func (s *keycloakService) GetUsers(ctx context.Context, enabled *bool) ([]*gocloak.User, error) {
	if config.KC == nil || config.KC.Client == nil {
		return nil, fmt.Errorf("keycloak configuration not initialized")
	}
	token, err := config.KC.LoginAdmin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin token: %w", err)
	}

	if enabled == nil {
		kcUsers, err := config.KC.Client.GetUsers(ctx, token.AccessToken, config.KC.Realm, gocloak.GetUsersParams{})
		if err != nil {
			return nil, fmt.Errorf("failed to get users from keycloak: %w", err)
		}
		return kcUsers, nil
	}

	// enabled 값이 false일 때 omitempty로 쿼리 파라미터가 누락되는 gocloak 버그 우회:
	// Keycloak Admin REST API를 직접 호출하여 ?enabled=false 명시 전송
	var result []*gocloak.User
	url := fmt.Sprintf("%s/admin/realms/%s/users?enabled=%v", config.KC.Host, config.KC.Realm, *enabled)
	resp, err := config.KC.Client.GetRequestWithBearerAuth(ctx, token.AccessToken).
		SetResult(&result).
		Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get users from keycloak: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("keycloak returned error %d for GetUsers", resp.StatusCode())
	}
	return result, nil
}

// CreateUser creates a user in Keycloak.
func (s *keycloakService) CreateUser(ctx context.Context, user *model.User) (string, error) {
	// Directly use config.KC
	if config.KC == nil || config.KC.Client == nil {
		return "", fmt.Errorf("keycloak configuration not initialized")
	}
	token, err := config.KC.LoginAdmin(ctx) // Use admin token
	if err != nil {
		return "", fmt.Errorf("failed to get admin token: %w", err)
	}
	keycloakUser := gocloak.User{
		Username:      &user.Username,
		Email:         &user.Email,
		FirstName:     &user.FirstName,
		LastName:      &user.LastName,
		Enabled:       gocloak.BoolP(true), // Typically created enabled by admin
		EmailVerified: gocloak.BoolP(true), // Assume verified if created by admin
	}
	// Add password if provided - requires temporary password handling
	// if user.Password != "" {
	// 	keycloakUser.Credentials = &[]gocloak.Credential{
	// 		{Type: gocloak.StringP("password"), Value: &user.Password, Temporary: gocloak.BoolP(false)},
	// 	}
	// }

	kcId, err := config.KC.Client.CreateUser(ctx, token.AccessToken, config.KC.Realm, keycloakUser)
	if err != nil {
		// Check for conflict (user exists)
		if strings.Contains(err.Error(), "409") {
			return "", fmt.Errorf("user with username '%s' or email '%s' already exists in keycloak", user.Username, user.Email)
		}
		return "", fmt.Errorf("failed to create user in keycloak: %w", err)
	}

	// If password was set, might need to reset required actions if not temporary
	// if user.Password != "" {
	// 	err = s.client.ExecuteActionsEmail(ctx, token.AccessToken, s.config.Realm, kcId, &[]string{"UPDATE_PASSWORD"})
	// 	if err != nil {
	// 		log.Printf("Warning: Failed to set required action UPDATE_PASSWORD for new user %s: %v", kcId, err)
	// 	}
	// }

	return kcId, nil
}

// CreatePendingUser creates a user in pending state (enabled=false) with password
func (s *keycloakService) CreatePendingUser(ctx context.Context, req *model.SignupRequest) (string, error) {
	if config.KC == nil || config.KC.Client == nil {
		return "", fmt.Errorf("keycloak configuration not initialized")
	}

	token, err := config.KC.LoginAdmin(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get admin token: %w", err)
	}

	// Generate username from email (before @)
	username := strings.Split(req.Email, "@")[0]

	keycloakUser := gocloak.User{
		Username:      &username,
		Email:         &req.Email,
		FirstName:     &req.FirstName,
		LastName:      &req.LastName,
		Enabled:       gocloak.BoolP(false),       // 승인 대기 상태
		EmailVerified: gocloak.BoolP(false),       // 이메일 미확인
		Attributes: &map[string][]string{
			"organization": {req.Organization},    // 조직 정보 저장
		},
	}

	kcId, err := config.KC.Client.CreateUser(ctx, token.AccessToken, config.KC.Realm, keycloakUser)
	if err != nil {
		if strings.Contains(err.Error(), "409") {
			return "", fmt.Errorf("Email already in use")
		}
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	// 비밀번호 설정
	err = config.KC.Client.SetPassword(ctx, token.AccessToken, kcId, config.KC.Realm, req.Password, false)
	if err != nil {
		// 사용자 생성은 성공했으나 비밀번호 설정 실패 - 사용자 삭제
		config.KC.Client.DeleteUser(ctx, token.AccessToken, config.KC.Realm, kcId)
		return "", fmt.Errorf("failed to set password: %w", err)
	}

	return kcId, nil
}

// ResetPassword resets a user's password
func (s *keycloakService) ResetPassword(ctx context.Context, kcUserID, newPassword string) error {
	if config.KC == nil || config.KC.Client == nil {
		return fmt.Errorf("keycloak configuration not initialized")
	}

	// Admin 토큰 획득
	adminToken, err := config.KC.LoginAdmin(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin token: %w", err)
	}

	// 사용자 존재 확인
	existingUser, err := config.KC.Client.GetUserByID(ctx, adminToken.AccessToken, config.KC.Realm, kcUserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if existingUser == nil {
		return fmt.Errorf("user not found: %s", kcUserID)
	}

	// 비밀번호 재설정 (temporary=false: 영구 비밀번호)
	err = config.KC.Client.SetPassword(ctx, adminToken.AccessToken, kcUserID, config.KC.Realm, newPassword, false)
	if err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}

	log.Printf("[INFO] Password reset successfully for user: %s", kcUserID)
	return nil
}

// UpdateUser updates a user in Keycloak.
func (s *keycloakService) UpdateUser(ctx context.Context, user *model.User) error {
	// Directly use config.KC
	if config.KC == nil || config.KC.Client == nil {
		return fmt.Errorf("keycloak configuration not initialized")
	}
	if user.KcId == "" {
		return fmt.Errorf("cannot update keycloak user without KcId")
	}
	token, err := config.KC.LoginAdmin(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin token for keycloak update: %w", err)
	}
	// Fetch existing user to only update provided fields? Or just send update payload.
	// Sending only updated fields is generally safer.
	keycloakUser := gocloak.User{
		ID:        &user.KcId, // ID is needed to identify the user
		Username:  &user.Username,
		Email:     &user.Email,
		FirstName: &user.FirstName,
		LastName:  &user.LastName,
		Enabled:   &user.Enabled,
		// Attributes: &user.Attributes, // If attributes are managed
	}
	err = config.KC.Client.UpdateUser(ctx, token.AccessToken, config.KC.Realm, keycloakUser)
	if err != nil {
		return fmt.Errorf("failed to update user in keycloak (kcId: %s): %w", user.KcId, err)
	}
	return nil
}

// GetRequestingPartyToken requests an RPT token from Keycloak using provided options.
func (s *keycloakService) GetRequestingPartyToken(ctx context.Context, accessToken string, options gocloak.RequestingPartyTokenOptions) (*gocloak.JWT, error) {
	if config.KC == nil || config.KC.Client == nil {
		return nil, fmt.Errorf("keycloak configuration not initialized")
	}

	// Ensure GrantType is set correctly for RPT request
	if options.GrantType == nil || *options.GrantType != "urn:ietf:params:oauth:grant-type:uma-ticket" {
		// It's often required to get a permission ticket first and include it in options.Ticket
		// For simplicity, we assume the caller provides the correct options including the ticket if needed.
		log.Println("Warning: GrantType in RequestingPartyTokenOptions should typically be 'urn:ietf:params:oauth:grant-type:uma-ticket'")
		// Setting it here just in case, but the caller should ideally set it.
		options.GrantType = gocloak.StringP("urn:ietf:params:oauth:grant-type:uma-ticket")
	}

	// The Audience for RPT is often the resource server (client acting as resource server)
	if options.Audience == nil {
		options.Audience = &config.KC.ClientName
	}

	// Call the gocloak function to get the RPT
	rpt, err := config.KC.Client.GetRequestingPartyToken(ctx, accessToken, config.KC.Realm, options)
	if err != nil {
		// Handle specific errors, e.g., 403 Forbidden if permissions are denied
		return nil, fmt.Errorf("failed to get requesting party token: %w", err)
	}
	if rpt == nil {
		// gocloak might return nil token and nil error in some cases? Unlikely but check.
		return nil, fmt.Errorf("received nil RPT token from keycloak without an error")
	}

	return rpt, nil
}

// ValidateTokenAndGetClaims validates the token signature/expiry and returns claims.
func (s *keycloakService) ValidateTokenAndGetClaims(ctx context.Context, token string) (*jwt.MapClaims, error) { // Changed return type
	if config.KC == nil || config.KC.Client == nil {
		return nil, fmt.Errorf("keycloak configuration not initialized")
	}
	// DecodeAccessToken performs local validation (signature, expiry) based on realm keys
	_, claims, err := config.KC.Client.DecodeAccessToken(ctx, token, config.KC.Realm)
	if err != nil {
		return nil, fmt.Errorf("token validation/decoding failed: %w", err)
	}
	if claims == nil {
		return nil, fmt.Errorf("token claims are nil after decoding")
	}
	// Note: For stricter validation, especially for RPTs or if Keycloak settings change,
	// using IntrospectToken might be preferred, but requires an extra network call.
	return claims, nil
}

// DeleteUser deletes a user from Keycloak.
func (s *keycloakService) DeleteUser(ctx context.Context, kcId string) error {
	// Directly use config.KC
	if config.KC == nil || config.KC.Client == nil {
		return fmt.Errorf("keycloak configuration not initialized")
	}
	token, err := config.KC.LoginAdmin(ctx)
	if err != nil {
		// Log warning but maybe don't fail the whole operation? Depends on desired behavior.
		log.Printf("Warning: failed to get admin token to delete user %s from keycloak: %v.", kcId, err)
		return fmt.Errorf("failed to get admin token for keycloak delete: %w", err)
	}
	err = config.KC.Client.DeleteUser(ctx, token.AccessToken, config.KC.Realm, kcId)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			log.Printf("User %s not found in Keycloak for deletion (already deleted?).", kcId)
			return nil // Not necessarily an error in this context
		}
		log.Printf("Error deleting user from keycloak (kc_id: %s): %v.", kcId, err)
		return fmt.Errorf("failed to delete user from keycloak: %w", err)
	}
	log.Printf("Successfully deleted user from Keycloak (kc_id: %s)", kcId)
	return nil
}

// EnableUser enables a user in Keycloak.
func (s *keycloakService) EnableUser(ctx context.Context, kcUserID string) error {
	// Directly use config.KC
	if config.KC == nil || config.KC.Client == nil {
		return fmt.Errorf("keycloak configuration not initialized")
	}
	adminToken, err := config.KC.LoginAdmin(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin token to enable user: %w", err)
	}
	// Get user first to ensure they exist
	user, err := config.KC.Client.GetUserByID(ctx, adminToken.AccessToken, config.KC.Realm, kcUserID)
	if err != nil {
		return fmt.Errorf("failed to get user %s from keycloak before enabling: %w", kcUserID, err)
	}
	if user == nil {
		return fmt.Errorf("user %s not found in keycloak", kcUserID)
	}
	// Update only the enabled flag
	userToUpdate := gocloak.User{
		ID:      &kcUserID,
		Enabled: gocloak.BoolP(true),
	}
	err = config.KC.Client.UpdateUser(ctx, adminToken.AccessToken, config.KC.Realm, userToUpdate)
	if err != nil {
		return fmt.Errorf("failed to enable user %s in keycloak: %w", kcUserID, err)
	}
	log.Printf("User '%s' enabled in Keycloak.", kcUserID)
	return nil
}

// DisableUser disables a user in Keycloak.
func (s *keycloakService) DisableUser(ctx context.Context, kcUserID string) error {
	if config.KC == nil || config.KC.Client == nil {
		return fmt.Errorf("keycloak configuration not initialized")
	}
	adminToken, err := config.KC.LoginAdmin(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin token to disable user: %w", err)
	}
	user, err := config.KC.Client.GetUserByID(ctx, adminToken.AccessToken, config.KC.Realm, kcUserID)
	if err != nil {
		return fmt.Errorf("failed to get user %s from keycloak before disabling: %w", kcUserID, err)
	}
	if user == nil {
		return fmt.Errorf("user %s not found in keycloak", kcUserID)
	}
	userToUpdate := gocloak.User{
		ID:      &kcUserID,
		Enabled: gocloak.BoolP(false),
	}
	err = config.KC.Client.UpdateUser(ctx, adminToken.AccessToken, config.KC.Realm, userToUpdate)
	if err != nil {
		return fmt.Errorf("failed to disable user %s in keycloak: %w", kcUserID, err)
	}
	log.Printf("User '%s' disabled in Keycloak.", kcUserID)
	return nil
}

// CheckAdminLogin checks if admin login to Keycloak is successful.
func (s *keycloakService) CheckAdminLogin(ctx context.Context) (bool, error) {
	// Directly use config.KC
	if config.KC == nil {
		return false, fmt.Errorf("keycloak configuration not initialized")
	}
	_, err := config.KC.LoginAdmin(ctx)
	if err != nil {
		return false, err
	}
	return true, nil
}

// CheckRealm checks if the configured realm exists. Requires admin token.
func (s *keycloakService) CheckRealm(ctx context.Context) (bool, error) {
	// Directly use config.KC
	if config.KC == nil || config.KC.Client == nil {
		log.Printf("[DEBUG] Keycloak configuration not initialized")
		return false, fmt.Errorf("keycloak configuration not initialized")
	}
	log.Printf("[DEBUG] Attempting to login as admin to check realm")
	token, err := config.KC.LoginAdmin(ctx)
	if err != nil {
		log.Printf("[DEBUG] Admin login failed: %v", err)
		return false, fmt.Errorf("admin login failed, cannot check realm: %w", err)
	}
	log.Printf("[DEBUG] Admin login successful, token obtained")

	// Check client permissions
	log.Printf("[DEBUG] Checking client permissions for realm management")
	clients, err := config.KC.Client.GetClients(ctx, token.AccessToken, config.KC.Realm, gocloak.GetClientsParams{
		ClientID: &config.KC.ClientName, // 클라이언트 이름으로 조회
	})
	if err != nil {
		log.Printf("[DEBUG] Failed to get client info: %v", err)
		return false, fmt.Errorf("failed to get client info: %w", err)
	}
	if len(clients) == 0 {
		log.Printf("[DEBUG] Client '%s' not found", config.KC.ClientName)
		return false, fmt.Errorf("client '%s' not found", config.KC.ClientName)
	}

	// Log required permissions
	log.Printf("[DEBUG] Required permissions for realm check:")
	log.Printf("[DEBUG] - realm-management:manage-realm")
	log.Printf("[DEBUG] - realm-management:query-realm")
	log.Printf("[DEBUG] - realm-management:view-clients")
	log.Printf("[DEBUG] - realm-management:query-clients")
	log.Printf("[DEBUG] - realm-management:manage-clients")

	// Get all realms first
	log.Printf("[DEBUG] Fetching all realms from Keycloak")
	realms, err := config.KC.Client.GetRealms(ctx, token.AccessToken)
	if err != nil {
		log.Printf("[DEBUG] Failed to get realms list: %v", err)
		log.Printf("[DEBUG] This might be due to missing realm-management permissions")
		return false, fmt.Errorf("failed to get realms list: %w", err)
	}

	// Log all available realms
	log.Printf("[DEBUG] Available realms count: %d", len(realms))
	for i, r := range realms {
		if r.Realm != nil {
			log.Printf("[DEBUG] Realm %d: %s", i+1, *r.Realm)
		} else {
			log.Printf("[DEBUG] Realm %d: <nil>", i+1)
		}
	}

	// Check if our realm exists
	log.Printf("[DEBUG] Checking if realm '%s' exists", config.KC.Realm)
	_, err = config.KC.Client.GetRealm(ctx, token.AccessToken, config.KC.Realm)
	if err != nil {
		log.Printf("[DEBUG] Failed to get realm '%s': %v", config.KC.Realm, err)
		log.Printf("[DEBUG] This might be due to missing realm-management permissions")
		return false, fmt.Errorf("failed to get realm '%s': %w", config.KC.Realm, err)
	}
	log.Printf("[DEBUG] Realm '%s' exists and is accessible", config.KC.Realm)
	return true, nil
}

func (s *keycloakService) CreateRealm(ctx context.Context, accessToken string) (bool, error) {
	lifespanSec := config.AccessTokenLifespanSec()
	newRealm := gocloak.RealmRepresentation{
		Realm:               gocloak.StringP(config.KC.Realm),
		Enabled:             gocloak.BoolP(true),
		DisplayName:         gocloak.StringP(config.KC.Realm),
		AccessTokenLifespan: gocloak.IntP(lifespanSec),
	}
	realmInfo, err := config.KC.Client.CreateRealm(ctx, accessToken, newRealm)
	if err != nil {
		return false, fmt.Errorf("failed to create realm '%s': %w", config.KC.Realm, err)
	}
	log.Printf("[DEBUG] Realm '%s' created successfully", realmInfo)
	return true, nil
}

func (s *keycloakService) ensureAccessTokenLifespan(ctx context.Context, accessToken string) error {
	lifespanSec := config.AccessTokenLifespanSec()
	realm, err := config.KC.Client.GetRealm(ctx, accessToken, config.KC.Realm)
	if err != nil {
		return fmt.Errorf("failed to get realm '%s': %w", config.KC.Realm, err)
	}
	if realm.AccessTokenLifespan != nil && *realm.AccessTokenLifespan == lifespanSec {
		return nil
	}
	realm.AccessTokenLifespan = gocloak.IntP(lifespanSec)
	if err := config.KC.Client.UpdateRealm(ctx, accessToken, *realm); err != nil {
		return fmt.Errorf("failed to update access token lifespan for realm '%s': %w", config.KC.Realm, err)
	}
	log.Printf("[INFO] Keycloak realm '%s' access token lifespan set to %ds", config.KC.Realm, lifespanSec)
	return nil
}

// CheckRealm checks if the configured realm exists. Requires admin token.
func (s *keycloakService) ExistRealm(ctx context.Context, accessToken string) (bool, error) {

	// Check if our realm exists
	log.Printf("[DEBUG] Checking if realm '%s' exists", config.KC.Realm)
	realmInfo, err := config.KC.Client.GetRealm(ctx, accessToken, config.KC.Realm)
	if err != nil {
		log.Printf("[DEBUG] Failed to get realm '%s': %v", config.KC.Realm, err)
		log.Printf("[DEBUG] This might be due to missing realm-management permissions")
		return false, fmt.Errorf("failed to get realm '%s': %w", config.KC.Realm, err)
	}
	log.Printf("[DEBUG] Realm '%s' exists and is accessible", realmInfo)
	return true, nil
}

// CheckClient checks if the configured client ID exists within the realm. Requires admin token.
func (s *keycloakService) CheckClient(ctx context.Context) (bool, error) {
	// Directly use config.KC
	if config.KC == nil || config.KC.Client == nil {
		return false, fmt.Errorf("keycloak configuration not initialized")
	}
	token, err := config.KC.LoginAdmin(ctx)
	if err != nil {
		return false, fmt.Errorf("admin login failed, cannot check client: %w", err)
	}
	clients, err := config.KC.Client.GetClients(ctx, token.AccessToken, config.KC.Realm, gocloak.GetClientsParams{ClientID: &config.KC.ClientName})
	if err != nil {
		return false, fmt.Errorf("failed to get client '%s': %w", config.KC.ClientName, err)
	}
	if len(clients) == 0 {
		return false, fmt.Errorf("client '%s' not found", config.KC.ClientName)
	}
	return true, nil
}

// CheckClient checks if the configured client ID exists within the realm. Requires admin token.
func (s *keycloakService) ExistClient(ctx context.Context, accessToken string) (bool, error) {

	clients, err := config.KC.Client.GetClients(ctx, accessToken, config.KC.Realm, gocloak.GetClientsParams{ClientID: &config.KC.ClientName})
	if err != nil {
		return false, fmt.Errorf("failed to get client '%s': %w", config.KC.ClientName, err)
	}
	if len(clients) == 0 {
		return false, fmt.Errorf("client '%s' not found", config.KC.ClientName)
	}
	return true, nil
}

func (s *keycloakService) CreateClient(ctx context.Context, accessToken string) (bool, error) {
	newClient := gocloak.Client{
		ClientID:                  gocloak.StringP(config.KC.ClientName),
		Secret:                    gocloak.StringP(config.KC.ClientSecret),
		Enabled:                   gocloak.BoolP(true),
		PublicClient:              gocloak.BoolP(false), // 'false'는 confidential client (비밀번호 필요)
		ServiceAccountsEnabled:    gocloak.BoolP(true),  // 서비스 계정 활성화
		DirectAccessGrantsEnabled: gocloak.BoolP(true),  // Direct access grants 활성화
	}
	clientInfo, err := config.KC.Client.CreateClient(ctx, accessToken, config.KC.Realm, newClient)
	if err != nil {
		return false, fmt.Errorf("failed to create client '%s': %w", config.KC.ClientName, err)
	}
	log.Printf("[DEBUG] Client '%s' created successfully", clientInfo)
	return true, nil
}

// GetUserIDFromToken extracts the user ID (subject) from a JWT token.
func (s *keycloakService) GetUserIDFromToken(ctx context.Context, token *gocloak.JWT) (string, error) {
	// Directly use config.KC
	if config.KC == nil || config.KC.Client == nil {
		return "", fmt.Errorf("keycloak configuration not initialized")
	}
	if token == nil {
		return "", fmt.Errorf("provided token is nil")
	}
	_, claims, err := config.KC.Client.DecodeAccessToken(ctx, token.AccessToken, config.KC.Realm)
	if err != nil {
		return "", fmt.Errorf("failed to decode access token: %w", err)
	}
	if claims == nil {
		return "", fmt.Errorf("token claims are nil")
	}
	userID, ok := (*claims)["sub"].(string)
	if !ok || userID == "" {
		return "", fmt.Errorf("token 'sub' claim not found or empty")
	}
	return userID, nil
}

// Login performs user login via Keycloak using username and password.
func (s *keycloakService) Login(ctx context.Context, username, password string) (*gocloak.JWT, error) {
	// Directly use config.KC
	if config.KC == nil || config.KC.Client == nil {
		return nil, fmt.Errorf("keycloak configuration not initialized")
	}

	// Add debug logging for Keycloak configuration
	log.Printf("[DEBUG] Keycloak Login Configuration:")
	log.Printf("[DEBUG] - Host: %s", config.KC.Host)
	log.Printf("[DEBUG] - Realm: %s", config.KC.Realm)
	log.Printf("[DEBUG] - ClientID: %s", config.KC.ClientName)
	log.Printf("[DEBUG] - ClientSecret: %s", config.KC.ClientSecret)
	log.Printf("[DEBUG] - Username: %s", username)

	token, err := config.KC.Client.Login(ctx, config.KC.ClientName, config.KC.ClientSecret, config.KC.Realm, username, password)
	if err != nil {
		// Consider more specific error handling for invalid credentials vs other errors
		return nil, fmt.Errorf("keycloak login failed: %w", err)
	}

	// 사용자 정보 가져오기
	userInfo, err := config.KC.Client.GetUserInfo(ctx, token.AccessToken, config.KC.Realm)
	if err != nil {
		log.Printf("[DEBUG] 사용자 정보 조회 실패: %v", err)
		return token, nil // 토큰은 성공했으므로 반환
	}

	// 로컬 DB에 사용자 동기화
	if userInfo.Sub != nil {
		log.Printf("[DEBUG] 로컬 DB 동기화 시작 (kc_id: %s)", *userInfo.Sub)
		// 여기서 로컬 DB 동기화 로직을 호출
		// 중복 키 에러는 무시하고 계속 진행
	}

	return token, nil
}

// RefreshToken refreshes the JWT token using a refresh token.
func (s *keycloakService) RefreshToken(ctx context.Context, refreshToken string) (*gocloak.JWT, error) {
	// Directly use config.KC
	if config.KC == nil || config.KC.Client == nil {
		return nil, fmt.Errorf("keycloak configuration not initialized")
	}
	newToken, err := config.KC.Client.RefreshToken(ctx, refreshToken, config.KC.ClientName, config.KC.ClientSecret, config.KC.Realm)
	if err != nil {
		return nil, fmt.Errorf("keycloak token refresh failed: %w", err)
	}
	return newToken, nil
}

// --- Group Synchronization Methods ---

// findGroupByName finds a group by name and returns its ID. Returns empty string if not found.
func (s *keycloakService) findGroupByName(ctx context.Context, token, groupName string) (string, error) {
	groups, err := config.KC.Client.GetGroups(ctx, token, config.KC.Realm, gocloak.GetGroupsParams{
		Search: &groupName,
		Exact:  gocloak.BoolP(true),
	})
	if err != nil {
		return "", fmt.Errorf("failed to search for group '%s': %w", groupName, err)
	}
	if len(groups) == 0 {
		return "", nil // Not found
	}
	if len(groups) > 1 {
		log.Printf("Warning: Found multiple groups named '%s'. Using the first one.", groupName)
	}
	if groups[0].ID == nil {
		return "", fmt.Errorf("found group '%s' but its ID is nil", groupName)
	}
	return *groups[0].ID, nil
}

// EnsureGroupExistsAndAssignUser ensures a group exists and assigns a user to it.
func (s *keycloakService) EnsureGroupExistsAndAssignUser(ctx context.Context, kcUserId, groupName string) error {
	if config.KC == nil || config.KC.Client == nil {
		return fmt.Errorf("keycloak configuration not initialized")
	}
	adminToken, err := config.KC.LoginAdmin(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin token for group operation: %w", err)
	}

	// 1. Find group by name
	groupID, err := s.findGroupByName(ctx, adminToken.AccessToken, groupName)
	if err != nil {
		return err // Error during search
	}

	// 2. Create group if not found
	if groupID == "" {
		log.Printf("Keycloak group '%s' not found, creating it.", groupName)
		newGroup := gocloak.Group{Name: &groupName}
		groupID, err = config.KC.Client.CreateGroup(ctx, adminToken.AccessToken, config.KC.Realm, newGroup)
		if err != nil {
			// Handle potential conflict if group was created concurrently
			if strings.Contains(err.Error(), "409") {
				log.Printf("Group '%s' likely created concurrently, attempting to find it again.", groupName)
				groupID, err = s.findGroupByName(ctx, adminToken.AccessToken, groupName)
				if err != nil {
					return err
				}
				if groupID == "" {
					return fmt.Errorf("failed to create or find group '%s' after conflict", groupName)
				}
			} else {
				return fmt.Errorf("failed to create keycloak group '%s': %w", groupName, err)
			}
		}
		log.Printf("Successfully created Keycloak group '%s' (ID: %s)", groupName, groupID)
	}

	// 3. Assign user to group
	err = config.KC.Client.AddUserToGroup(ctx, adminToken.AccessToken, config.KC.Realm, kcUserId, groupID)
	if err != nil {
		// Handle potential errors like user already in group (might not be an error depending on gocloak)
		// Or user not found (should have been checked before calling this service method ideally)
		return fmt.Errorf("failed to add user '%s' to keycloak group '%s' (ID: %s): %w", kcUserId, groupName, groupID, err)
	}

	return nil
}

// RemoveUserFromGroup removes a user from a specific group.
func (s *keycloakService) RemoveUserFromGroup(ctx context.Context, kcUserId, groupName string) error {
	if config.KC == nil || config.KC.Client == nil {
		return fmt.Errorf("keycloak configuration not initialized")
	}
	adminToken, err := config.KC.LoginAdmin(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin token for group operation: %w", err)
	}

	// 1. Find group by name
	groupID, err := s.findGroupByName(ctx, adminToken.AccessToken, groupName)
	if err != nil {
		return err // Error during search
	}

	// 2. If group doesn't exist, nothing to remove from
	if groupID == "" {
		log.Printf("Keycloak group '%s' not found, cannot remove user '%s'.", groupName, kcUserId)
		return nil // Or return an error? Let's consider it not an error for idempotency.
	}

	// 3. Remove user from group
	err = config.KC.Client.DeleteUserFromGroup(ctx, adminToken.AccessToken, config.KC.Realm, kcUserId, groupID)
	if err != nil {
		// Handle potential errors like user not in group (might not be an error) or user not found
		if strings.Contains(err.Error(), "404") {
			log.Printf("User '%s' or Group '%s' (ID: %s) not found during removal, or user not in group.", kcUserId, groupName, groupID)
			return nil // Consider it not an error
		}
		return fmt.Errorf("failed to remove user '%s' from keycloak group '%s' (ID: %s): %w", kcUserId, groupName, groupID, err)
	}

	return nil
}

// SetupInitialAdmin creates the initial platform admin user and sets up necessary permissions
// 초기 관리자 생성 및 권한 설정
// 1. KC 관리자 로그인 -> token 발급
// 2. KC Realm 생성
// 3. KC Client 생성
// 4. KC User 생성 for platformAdmin at .env defined user info
// 5. Releam Role 생성 : platformAdmin by default
// 6. KC Role 할당 : platformAdmin to user
func (s *keycloakService) SetupInitialKeycloakAdmin(ctx context.Context, adminToken *gocloak.JWT) (string, error) {
	if config.KC == nil || config.KC.Client == nil {
		return "", fmt.Errorf("keycloak configuration not initialized")
	}

	log.Printf("[DEBUG] adminToken: %s", adminToken.AccessToken)

	existRealm, err := s.ExistRealm(ctx, adminToken.AccessToken)
	if err != nil {

		if !existRealm {
			createRealm, err := s.CreateRealm(ctx, adminToken.AccessToken)
			if err != nil {
				return "", fmt.Errorf("failed to create realm: %w", err)
			}
			log.Print("[DEBUG] createRealm ", createRealm)
		}
	} else if err := s.ensureAccessTokenLifespan(ctx, adminToken.AccessToken); err != nil {
		log.Printf("[WARN] failed to ensure access token lifespan: %v", err)
	}

	existClient, err := s.ExistClient(ctx, adminToken.AccessToken)
	if err != nil {
		if !existClient {
			createClient, err := s.CreateClient(ctx, adminToken.AccessToken)
			if err != nil {
				return "", fmt.Errorf("failed to create client: %w", err)
			}
			log.Print("[DEBUG] createClient ", createClient)
		}
	}

	// 2. platformAdmin 사용자 생성
	platformAdminID := os.Getenv("MC_IAM_MANAGER_PLATFORMADMIN_ID")
	platformAdminPassword := os.Getenv("MC_IAM_MANAGER_PLATFORMADMIN_PASSWORD")
	platformAdminFirstName := os.Getenv("MC_IAM_MANAGER_PLATFORMADMIN_FIRSTNAME")
	platformAdminLastName := os.Getenv("MC_IAM_MANAGER_PLATFORMADMIN_LASTNAME")
	platformAdminEmail := os.Getenv("MC_IAM_MANAGER_PLATFORMADMIN_EMAIL")

	if platformAdminID == "" {
		return "", fmt.Errorf("MC_IAM_MANAGER_PLATFORMADMIN_ID not set in environment variables")
	}
	if platformAdminPassword == "" {
		return "", fmt.Errorf("MC_IAM_MANAGER_PLATFORMADMIN_PASSWORD not set in environment variables")
	}
	if platformAdminFirstName == "" {
		return "", fmt.Errorf("MC_IAM_MANAGER_PLATFORMADMIN_FIRSTNAME not set in environment variables")
	}
	if platformAdminLastName == "" {
		return "", fmt.Errorf("MC_IAM_MANAGER_PLATFORMADMIN_LASTNAME not set in environment variables")
	}
	if platformAdminEmail == "" {
		return "", fmt.Errorf("MC_IAM_MANAGER_PLATFORMADMIN_EMAIL not set in environment variables")
	}

	log.Printf("[DEBUG] Creating platform admin user: %s", platformAdminID)
	user := gocloak.User{
		Username:        &platformAdminID,
		FirstName:       &platformAdminFirstName,
		LastName:        &platformAdminLastName,
		Email:           &platformAdminEmail,
		Enabled:         gocloak.BoolP(true),
		EmailVerified:   gocloak.BoolP(true),
		RequiredActions: &[]string{""},
	}
	kcUsers, err := config.KC.Client.GetUsers(ctx, adminToken.AccessToken, config.KC.Realm, gocloak.GetUsersParams{
		Username: gocloak.StringP(platformAdminID),
		Exact:    gocloak.BoolP(true),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get user from keycloak by username %s: %w", platformAdminID, err)
	}

	kcUser := gocloak.User{}
	existUser := false
	if len(kcUsers) == 0 || len(kcUsers) > 1 {

	}
	if len(kcUsers) == 1 {
		existUser = true
		kcUser = *kcUsers[0]
	}

	userID := ""
	if existUser {
		userID = *kcUser.ID
		log.Printf("[DEBUG] User exists : %s", userID)
	} else {
		kcId, err := config.KC.Client.CreateUser(ctx, adminToken.AccessToken, config.KC.Realm, user)
		if err != nil {
			return "", fmt.Errorf("failed to create user: %w", err)
		}
		log.Printf("[DEBUG] User created with ID: %s, kc id: %s", userID, kcId)

		// 5초 대기
		time.Sleep(5 * time.Second)
		userID = kcId

		// 비밀번호 설정
		err = config.KC.Client.SetPassword(ctx, adminToken.AccessToken, userID, config.KC.Realm, platformAdminPassword, false)
		if err != nil {
			return "", fmt.Errorf("failed to set password: %w", err)
		}
		log.Printf("[DEBUG] Password set successfully")

	}

	// if !platformRoleExists {
	// 	log.Printf("[DEBUG] PlatformAdmin role not exists")
	// }
	// //if len(rolesToAssign) > 0 {
	// 	err = config.KC.Client.AddRealmRoleToUser(ctx, adminToken.AccessToken, config.KC.Realm, userID, rolesToAssign)
	// 	if err != nil {
	// 		log.Printf("failed to assign default roles %s", rolesToAssign[0].Name)
	// 		return nil, fmt.Errorf("failed to assign default roles: %w", err)
	// 	}
	// 	log.Printf("[DEBUG] Default roles assigned")
	// }

	// 3. platformAdmin 역할 할당. patformAdmin 역할이 없으면 생성
	log.Printf("[DEBUG] Setting platformAdmin role")
	platformAdminRoleName := "platformAdmin"
	var platformAdminRole gocloak.Role
	realmRole, err := config.KC.Client.GetRealmRole(ctx, adminToken.AccessToken, config.KC.Realm, platformAdminRoleName)
	if err != nil {
		log.Printf("failed to get platformAdmin role: %v", err)
		newRole := gocloak.Role{
			Name:        gocloak.StringP(platformAdminRoleName),
			Description: gocloak.StringP("Predefined platform role"),
		}
		result, err := config.KC.Client.CreateRealmRole(ctx, adminToken.AccessToken, config.KC.Realm, newRole)
		if err != nil {
			log.Printf("Failed to create realm role %s, %s: %v", platformAdminRoleName, result, err)
			return "", fmt.Errorf("failed to create platformAdmin role: %w", err)
		}
		log.Printf("platformAdminRole created: %v", result)

		// 5초 대기 : 만들자마자 조회하면 안됨
		time.Sleep(5 * time.Second)

		// 다시 조회
		realmRoleResult, err := config.KC.Client.GetRealmRole(ctx, adminToken.AccessToken, config.KC.Realm, platformAdminRoleName)
		if err != nil {
			log.Printf("failed to get platformAdmin role again: %v", err)
			return userID, fmt.Errorf("failed to get platformAdmin role again: %w", err)
		}
		platformAdminRole = *realmRoleResult
	} else {
		platformAdminRole = *realmRole
	}
	log.Printf("platformAdminRole: %+v", platformAdminRole)
	// platformAdminRole, err := config.KC.Client.GetRealmRole(ctx, adminToken.AccessToken, config.KC.Realm, "platformAdmin")
	// if err != nil {
	// 	log.Printf("failed to get platformAdmin role: %v", err)
	// 	return nil, fmt.Errorf("failed to get platformAdmin role: %w", err)
	// }

	err = config.KC.Client.AddRealmRoleToUser(ctx, adminToken.AccessToken, config.KC.Realm, userID, []gocloak.Role{platformAdminRole})
	if err != nil {
		log.Printf("failed to assign platformAdmin role: %v", err)
		return userID, fmt.Errorf("failed to assign platformAdmin role: %w", err)
	}
	log.Printf("[DEBUG] PlatformAdmin role assigned")

	return userID, nil
}

// CheckUserRoles checks and logs all roles assigned to a user
func (s *keycloakService) CheckUserRoles(ctx context.Context, username string) error {
	if config.KC == nil || config.KC.Client == nil {
		return fmt.Errorf("keycloak configuration not initialized")
	}

	// Admin 로그인
	log.Printf("[DEBUG] === Keycloak 설정 정보 ===")
	log.Printf("[DEBUG] Realm: %s", config.KC.Realm)
	log.Printf("[DEBUG] ClientName: %s", config.KC.ClientName)
	log.Printf("[DEBUG] Host: %s", config.KC.Host)
	log.Printf("[DEBUG] ======================")

	adminToken, err := config.KC.LoginAdmin(ctx)
	if err != nil {
		return fmt.Errorf("admin login failed: %w", err)
	}
	log.Printf("[DEBUG] Admin 로그인 성공")

	// 모든 사용자 목록 가져오기
	users, err := config.KC.Client.GetUsers(ctx, adminToken.AccessToken, config.KC.Realm, gocloak.GetUsersParams{})
	if err != nil {
		log.Printf("[DEBUG] 사용자 목록 조회 실패: %v", err)
		return fmt.Errorf("failed to get users: %w", err)
	}

	log.Printf("[DEBUG] === 사용자 목록 (%d명) ===", len(users))
	for _, user := range users {
		if user.Username != nil {
			log.Printf("[DEBUG] 사용자: %s (ID: %s)", *user.Username, *user.ID)
		}
	}
	log.Printf("[DEBUG] =======================")

	// 대소문자 구분 없이 사용자 찾기
	var targetUser *gocloak.User
	for _, user := range users {
		if user.Username != nil && strings.EqualFold(*user.Username, username) {
			targetUser = user
			break
		}
	}

	if targetUser == nil {
		log.Printf("[DEBUG] 사용자를 찾을 수 없음: %s", username)
		return fmt.Errorf("사용자를 찾을 수 없습니다: %s", username)
	}

	userID := *targetUser.ID
	log.Printf("[DEBUG] 찾은 사용자 ID: %s", userID)

	// Realm 역할 확인
	realmRoles, err := config.KC.Client.GetRealmRolesByUserID(ctx, adminToken.AccessToken, config.KC.Realm, userID)
	if err != nil {
		log.Printf("[DEBUG] Realm 역할 조회 실패: %v", err)
	} else {
		log.Printf("[DEBUG] === Realm 역할 목록 (%d개) ===", len(realmRoles))
		for _, role := range realmRoles {
			if role.Name != nil {
				log.Printf("[DEBUG] - %s", *role.Name)
			}
		}
		log.Printf("[DEBUG] =======================")
	}

	// 클라이언트 정보 가져오기
	clients, err := config.KC.Client.GetClients(ctx, adminToken.AccessToken, config.KC.Realm, gocloak.GetClientsParams{
		ClientID: &config.KC.ClientName,
	})
	if err != nil {
		log.Printf("[DEBUG] 클라이언트 목록 조회 실패: %v", err)
		return fmt.Errorf("클라이언트 정보 조회 실패: %w", err)
	}
	if len(clients) == 0 {
		log.Printf("[DEBUG] 클라이언트를 찾을 수 없음: %s", config.KC.ClientName)
		return fmt.Errorf("클라이언트를 찾을 수 없습니다: %s", config.KC.ClientName)
	}
	clientID := *clients[0].ID
	log.Printf("[DEBUG] 클라이언트 ID: %s", clientID)

	// 클라이언트 역할 확인
	clientRoles, err := config.KC.Client.GetClientRolesByUserID(ctx, adminToken.AccessToken, config.KC.Realm, clientID, userID)
	if err != nil {
		log.Printf("[DEBUG] 클라이언트 역할 조회 실패: %v", err)
	} else {
		log.Printf("[DEBUG] === 클라이언트 역할 목록 (%d개) ===", len(clientRoles))
		for _, role := range clientRoles {
			if role.Name != nil {
				log.Printf("[DEBUG] - %s", *role.Name)
			}
		}
		log.Printf("[DEBUG] ==========================")
	}

	// 기본 역할 확인
	defaultRoles, err := config.KC.Client.GetRealmRoles(ctx, adminToken.AccessToken, config.KC.Realm, gocloak.GetRoleParams{
		Search: gocloak.StringP("default"),
	})
	if err != nil {
		log.Printf("[DEBUG] 기본 역할 조회 실패: %v", err)
	} else {
		log.Printf("[DEBUG] === 사용 가능한 기본 역할 목록 (%d개) ===", len(defaultRoles))
		for _, role := range defaultRoles {
			if role.Name != nil {
				log.Printf("[DEBUG] - %s", *role.Name)
			}
		}
		log.Printf("[DEBUG] ==============================")
	}

	return nil
}

// GetUserPermissions gets all permissions for the given roles
func (s *keycloakService) GetUserPermissions(ctx context.Context, roles []string) ([]string, error) {
	if config.KC == nil || config.KC.Client == nil {
		return nil, fmt.Errorf("keycloak configuration not initialized")
	}

	// Get admin token
	token, err := config.KC.GetAdminToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin token: %w", err)
	}

	// Get client ID
	clients, err := config.KC.Client.GetClients(ctx, token.AccessToken, config.KC.Realm, gocloak.GetClientsParams{
		ClientID: &config.KC.ClientName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	if len(clients) == 0 {
		return nil, fmt.Errorf("client not found")
	}
	clientID := *clients[0].ID

	// Get all permissions for the roles
	var allPermissions []string
	for _, role := range roles {
		// Get role details
		roleDetails, err := config.KC.Client.GetClientRole(ctx, token.AccessToken, config.KC.Realm, clientID, role)
		if err != nil {
			log.Printf("Warning: Failed to get role details for %s: %v", role, err)
			continue
		}

		// Add the role itself as a permission
		if roleDetails.Name != nil {
			allPermissions = append(allPermissions, *roleDetails.Name)
		}

		// Add role attributes as permissions
		if roleDetails.Attributes != nil {
			for key, values := range *roleDetails.Attributes {
				for _, value := range values {
					allPermissions = append(allPermissions, fmt.Sprintf("%s:%s", key, value))
				}
			}
		}
	}

	return allPermissions, nil
}

// GetImpersonationToken gets an impersonation token for a user
func (s *keycloakService) GetImpersonationToken(ctx context.Context) (*gocloak.JWT, error) {
	if config.KC == nil || config.KC.Client == nil {
		return nil, fmt.Errorf("keycloak configuration not initialized")
	}

	// Get user's access token from context
	accessToken, ok := ctx.Value("access_token").(string)
	if !ok || accessToken == "" {
		return nil, fmt.Errorf("access token not found in context")
	}

	// Get user ID from context
	kcUserId, ok := ctx.Value("kcUserId").(string)
	if !ok || kcUserId == "" {
		return nil, fmt.Errorf("user ID not found in context")
	}

	// adminToken, err := config.KC.GetAdminToken(ctx)
	// if err != nil {
	// 	return nil, fmt.Errorf("admin login failed: %w", err)
	// }
	// log.Printf("[DEBUG] adminToken: %s", adminToken.AccessToken)
	//stsClientID := "aws-sts-client"
	username := "leeman"
	// Set up token exchange options
	tokenOptions := gocloak.TokenOptions{
		//GrantType:          gocloak.StringP("urn:ietf:params:oauth:grant-type:token-exchange"),
		GrantType: gocloak.StringP("urn:ietf:params:oauth:grant-type:token-exchange"),

		SubjectToken: gocloak.StringP(accessToken),
		// SubjectToken:       gocloak.StringP(adminToken.AccessToken),
		RequestedTokenType: gocloak.StringP("urn:ietf:params:oauth:token-type:refresh_token"),
		ClientID:           &config.KC.OIDCClientID,
		ClientSecret:       &config.KC.OIDCClientSecret,
		RequestedSubject:   &kcUserId,
		Username:           &username,
	}
	log.Printf("[DEBUG] adminToken: %s", accessToken)
	// Get impersonation token using TokenExchange
	token, err := config.KC.Client.GetToken(ctx, config.KC.Realm, tokenOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get impersonation token: %w", err)
	}

	return token, nil
}

// GetImpersonationTokenByAdminToken: adminToken을 이용해 특정 사용자의 impersonation 토큰을 발급
func (s *keycloakService) GetImpersonationTokenByAdminToken(ctx context.Context, userID string, targetClientID string) (string, error) {

	// 1. admin 계정으로 로그인
	if config.KC == nil || config.KC.Client == nil {
		return "", fmt.Errorf("keycloak configuration not initialized")
	}
	adminToken, err := config.KC.GetAdminToken(ctx)
	if err != nil {
		return "", fmt.Errorf("admin login failed: %w", err)
	}
	//user, err := config.KC.Client.GetUserByID(ctx, adminToken.AccessToken, config.KC.Realm, kcId)
	// var result User
	// resp, err := g.GetRequestWithBearerAuth(ctx, accessToken).
	// 	SetResult(&result).
	// 	Get(g.getAdminRealmURL(realm, "users", userID))

	// if err := checkForError(resp, err, errMessage); err != nil {
	// 	return nil, err
	// }

	log.Printf("[DEBUG] adminToken: %s", adminToken.AccessToken)
	// 2. Keycloak REST API로 impersonation 요청
	url := fmt.Sprintf("%s/admin/realms/%s/users/%s/impersonation", config.KC.Host, config.KC.Realm, userID)
	body := map[string]interface{}{}
	// if targetClientID != "" {
	// 	body["client_id"] = targetClientID
	// }

	// 환경 변수에서 OIDC 클라이언트 ID 가져오기
	log.Printf("[DEBUG] Attempting to get KEYCLOAK_OIDC_CLIENT_ID from environment")
	oidcClientID := os.Getenv("KEYCLOAK_OIDC_CLIENT_ID")
	log.Printf("[DEBUG] KEYCLOAK_OIDC_CLIENT_ID value: '%s'", oidcClientID)
	if oidcClientID == "" {
		log.Printf("[DEBUG] KEYCLOAK_OIDC_CLIENT_ID is empty, checking alternative environment variables")
		// 대안 환경 변수들 확인
		alt1 := os.Getenv("KEYCLOAK_OIDC_CLIENT_NAME")
		log.Printf("[DEBUG] KEYCLOAK_OIDC_CLIENT_NAME value: '%s'", alt1)
		alt2 := os.Getenv("KEYCLOAK_CLIENT_NAME")
		log.Printf("[DEBUG] KEYCLOAK_CLIENT_NAME value: '%s'", alt2)
		return "", fmt.Errorf("KEYCLOAK_OIDC_CLIENT_ID environment variable is not set")
	}
	body["client_id"] = oidcClientID // 하드코딩된 값 대신 환경 변수 사용
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create impersonation request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+adminToken.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("impersonation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("impersonation failed: %s", string(respBody))
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode impersonation response: %w", err)
	}

	log.Printf("[DEBUG] Impersonation resp.StatusCode : %d", resp.StatusCode) //if result.Token == "" {
	// resp.Body를 다시 읽기 위해 전체 바이트로 읽음
	respBody, _ := ioutil.ReadAll(resp.Body)
	log.Printf("[DEBUG] Impersonation response body: %s", string(respBody))
	//}

	log.Printf("[DEBUG] Impersonation token: %s", result.Token)

	return result.Token, nil
}

// GetImpersonationTokenByServiceAccount: 서비스 계정을 이용해 특정 클라이언트에 로그인한 토큰을 발급
func (s *keycloakService) GetImpersonationTokenByServiceAccount(ctx context.Context) (*gocloak.JWT, error) {
	if config.KC == nil || config.KC.Client == nil {
		return nil, fmt.Errorf("keycloak configuration not initialized")
	}

	// KeycloakConfig에서 OIDC 클라이언트 ID와 시크릿 가져오기
	clientID := config.KC.OIDCClientID
	clientName := config.KC.OIDCClientName
	clientSecret := config.KC.OIDCClientSecret

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("OIDC client ID or secret not configured in KeycloakConfig")
	}

	log.Printf("[DEBUG] Impersonation clientID: %s", clientID)
	log.Printf("[DEBUG] Impersonation clientName: %s", clientName)
	log.Printf("[DEBUG] Impersonation clientSecret: %s", clientSecret)
	log.Printf("[DEBUG] Impersonation realm: %s", config.KC.Realm)

	// 서비스 계정으로 로그인 (openid scope 포함 → id_token 발급)
	// Alibaba STS AssumeRoleWithOIDC는 단일 aud 문자열의 id_token을 요구함
	token, err := config.KC.Client.LoginClient(ctx, clientName, clientSecret, config.KC.Realm, "openid")
	if err != nil {
		return nil, fmt.Errorf("failed to login with service account: %w", err)
	}

	return token, nil
}

// GetSamlAssertionByServiceAccount: RFC 8693 토큰 교환으로 SAML2 assertion을 발급하고
// AWS STS AssumeRoleWithSAML에 전달할 수 있는 base64-encoded SAMLResponse를 반환한다.
//
// 내부 흐름:
//  1. platform admin 계정으로 password grant → UserSession 포함 JWT 발급
//  2. RFC 8693 token exchange → base64url-encoded SAML Assertion 획득
//  3. Assertion XML을 SAMLResponse로 래핑
//  4. standard base64 인코딩하여 반환
//
// samlClientAudience: Keycloak에 등록된 SAML 클라이언트 ID (e.g., "urn:amazon:webservices")
func (s *keycloakService) GetSamlAssertionByServiceAccount(ctx context.Context, samlClientAudience string) (string, error) {
	if config.KC == nil || config.KC.Client == nil {
		return "", fmt.Errorf("keycloak configuration not initialized")
	}

	clientName := config.KC.OIDCClientName
	clientSecret := config.KC.OIDCClientSecret
	if clientName == "" || clientSecret == "" {
		return "", fmt.Errorf("OIDC client credentials not configured")
	}

	// Step 1: password grant으로 UserSession 포함 token 발급
	// client_credentials는 UserSession을 생성하지 않아 SAML token exchange가 실패함.
	platformAdminID := os.Getenv("MC_IAM_MANAGER_PLATFORMADMIN_ID")
	platformAdminPW := os.Getenv("MC_IAM_MANAGER_PLATFORMADMIN_PASSWORD")
	if platformAdminID == "" || platformAdminPW == "" {
		return "", fmt.Errorf("platform admin credentials not configured (MC_IAM_MANAGER_PLATFORMADMIN_ID/PASSWORD required for SAML exchange)")
	}

	userToken, err := config.KC.Client.Login(ctx, clientName, clientSecret, config.KC.Realm, platformAdminID, platformAdminPW)
	if err != nil {
		return "", fmt.Errorf("failed to get user token for SAML exchange: %w", err)
	}

	// Step 2: RFC 8693 토큰 교환 — access token → SAML2 assertion (base64url)
	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", config.KC.Host, config.KC.Realm)

	formData := url.Values{}
	formData.Set("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange")
	formData.Set("client_id", clientName)
	formData.Set("client_secret", clientSecret)
	formData.Set("subject_token", userToken.AccessToken)
	formData.Set("subject_token_type", "urn:ietf:params:oauth:token-type:access_token")
	formData.Set("requested_token_type", "urn:ietf:params:oauth:token-type:saml2")
	formData.Set("audience", samlClientAudience)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create SAML exchange request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("SAML token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read SAML exchange response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Keycloak SAML exchange returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken     string `json:"access_token"`
		IssuedTokenType string `json:"issued_token_type"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse SAML exchange response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("Keycloak returned empty SAML assertion")
	}

	// Step 3: base64url assertion → XML 디코딩
	assertionXML, err := decodeBase64URLToString(tokenResp.AccessToken)
	if err != nil {
		return "", fmt.Errorf("failed to decode SAML assertion from Keycloak: %w", err)
	}

	// Step 4: SAMLResponse 래핑 + standard base64 인코딩
	// AWS/Alibaba STS AssumeRoleWithSAML은 SAMLResponse 래퍼가 포함된 base64를 요구함
	// destination: Keycloak SAML 클라이언트의 ACS URL (assertions의 Recipient과 일치)
	destination := extractRecipientFromAssertion(assertionXML)
	samlResponseXML := buildSAMLResponse(assertionXML, destination)
	samlResponseB64 := base64.StdEncoding.EncodeToString([]byte(samlResponseXML))

	log.Printf("[KEYCLOAK] SAML assertion exchange succeeded for audience: %s", samlClientAudience)
	return samlResponseB64, nil
}

// extractRecipientFromAssertion extracts the Recipient URL from the SAML assertion's
// SubjectConfirmationData element. Returns empty string if not found.
func extractRecipientFromAssertion(assertionXML string) string {
	const recipientAttr = `Recipient="`
	idx := strings.Index(assertionXML, recipientAttr)
	if idx < 0 {
		return ""
	}
	start := idx + len(recipientAttr)
	end := strings.Index(assertionXML[start:], `"`)
	if end < 0 {
		return ""
	}
	return assertionXML[start : start+end]
}

// decodeBase64URLToString base64url (Keycloak token exchange 반환값) → UTF-8 문자열 디코딩
func decodeBase64URLToString(b64url string) (string, error) {
	// base64url → standard base64
	b64 := strings.ReplaceAll(b64url, "-", "+")
	b64 = strings.ReplaceAll(b64, "_", "/")
	// 패딩 추가
	switch len(b64) % 4 {
	case 2:
		b64 += "=="
	case 3:
		b64 += "="
	}
	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %w", err)
	}
	return string(decoded), nil
}

// buildSAMLResponse Keycloak SAML Assertion XML을 SAMLResponse로 래핑
// AWS/Alibaba STS AssumeRoleWithSAML은 samlp:Response 래퍼가 포함된 base64를 요구함
// - Destination: ACS URL (Alibaba: https://signin.aliyun.com/saml-role/sso)
// - xmlns:saml 은 내부 Assertion에서 선언하므로 Response에서 제외 (서명 c14n 안정성)
func buildSAMLResponse(assertionXML string, destination string) string {
	dest := ""
	if destination != "" {
		dest = ` Destination="` + destination + `"`
	}
	return `<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" ID="_samlresponse" Version="2.0" IssueInstant="` +
		time.Now().UTC().Format("2006-01-02T15:04:05Z") + `"` + dest +
		`><samlp:Status><samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/></samlp:Status>` +
		assertionXML +
		`</samlp:Response>`
}

// AssignRealmRoleToUser assigns a realm role to a user
func (s *keycloakService) AssignRealmRoleToUser(ctx context.Context, kcUserId, roleName string) error {
	if config.KC == nil || config.KC.Client == nil {
		return fmt.Errorf("keycloak configuration not initialized")
	}

	token, err := config.KC.GetAdminToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin token: %w", err)
	}

	// Get the role by name
	roles, err := config.KC.Client.GetRealmRoles(ctx, token.AccessToken, config.KC.Realm, gocloak.GetRoleParams{
		Search: &roleName,
	})
	if err != nil {
		return fmt.Errorf("failed to get realm role %s: %w", roleName, err)
	}
	if len(roles) == 0 {
		return fmt.Errorf("realm role %s not found", roleName)
	}
	if len(roles) > 1 {
		log.Printf("Warning: Found multiple roles matching '%s'. Using the first one.", roleName)
	}

	// Assign the role to the user
	err = config.KC.Client.AddRealmRoleToUser(ctx, token.AccessToken, config.KC.Realm, kcUserId, []gocloak.Role{*roles[0]})
	if err != nil {
		return fmt.Errorf("failed to assign realm role %s to user %s: %w", roleName, kcUserId, err)
	}

	log.Printf("Successfully assigned realm role %s to user %s", roleName, kcUserId)
	return nil
}

// IssueWorkspaceTicket 워크스페이스 티켓을 발행합니다.
func (s *keycloakService) IssueWorkspaceTicket(ctx context.Context, kcUserId string, workspaceID uint) (string, map[string]interface{}, error) {
	// Keycloak에 워크스페이스 티켓 발행 요청
	// TODO: 실제 Keycloak API 호출 구현
	ticket := fmt.Sprintf("workspace_ticket_%s_%d", kcUserId, workspaceID)

	// 임시 권한 정보 생성
	permissions := map[string]interface{}{
		"workspace_id": workspaceID,
		"kc_user_id":   kcUserId,
		"roles":        []string{"admin"}, // TODO: 실제 사용자 역할 조회
	}

	return ticket, permissions, nil
}

// SetupPredefinedRoles retrieves all realm roles for a specific realm
// 특정 Realm의 모든 RealmRole 목록을 조회합니다.
func (s *keycloakService) SetupPredefinedRoles(ctx context.Context, accessToken string) error {
	if config.KC == nil || config.KC.Client == nil {
		return fmt.Errorf("keycloak client is not initialized")
	}

	// Get all realm roles
	realmRoles, err := config.KC.Client.GetRealmRoles(ctx, accessToken, config.KC.Realm, gocloak.GetRoleParams{})
	if err != nil {
		log.Printf("[DEBUG] Get Realm roles failed : %v", err)
		return fmt.Errorf("failed to get realm roles: %w", err)
	}

	// MC_IAM_MANAGER_PREDEFINED_PLATFORM_ROLE에 정의된 역할들이 없으면 생성
	predefinedRoles := strings.Split(os.Getenv("MC_IAM_MANAGER_PREDEFINED_PLATFORM_ROLE"), ",")
	for _, roleName := range predefinedRoles {
		roleName = strings.TrimSpace(roleName)
		if roleName == "" {
			continue
		}

		// 역할이 이미 존재하는지 확인
		roleExists := false
		for _, role := range realmRoles {
			//log.Printf("default role: %s, predefined role: %s", *role.Name, roleName)
			if role.Name != nil && *role.Name == roleName {
				roleExists = true
				log.Printf("role: %s exists", roleName)
				break
			}
		}

		// 역할이 없으면 생성
		if !roleExists {
			log.Printf("Creating predefined role: %s", roleName)
			log.Printf("target realm %s, client %s", config.KC.Realm, config.KC.ClientID)
			newRole := gocloak.Role{
				Name:        &roleName,
				Description: gocloak.StringP("Predefined platform role"),
			}

			result, err := config.KC.Client.CreateRealmRole(ctx, accessToken, config.KC.Realm, newRole)
			if err != nil {
				log.Printf("Failed to create realm role %s, %s: %v", roleName, result, err)
				continue
			}
			// _, err := config.KC.Client.CreateClientRole(ctx, adminToken.AccessToken, config.KC.Realm, config.KC.ClientID, newRole)
			// if err != nil {
			// 	log.Printf("Failed to create role %s: %v", roleName, err)
			// 	continue
			// }

			// // 생성된 역할을 할당 목록에 추가
			// log.Println("result : ", result)

			// if *newRole.Name == "platformAdmin" {
			// 	rolesToAssign = append(rolesToAssign, newRole)
			// }
		}
	}

	return nil
}

func (s *keycloakService) GetCerts(ctx context.Context) (*gocloak.CertResponse, error) {

	cert, err := config.KC.Client.GetCerts(ctx, config.KC.Realm)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return cert, nil
}

// GetClientCredentialsToken 클라이언트 자격 증명으로 토큰을 발급받습니다.
func (s *keycloakService) GetClientCredentialsToken(ctx context.Context) (*gocloak.JWT, error) {
	if config.KC == nil || config.KC.Client == nil {
		return nil, fmt.Errorf("keycloak configuration not initialized")
	}

	// Get client credentials
	realm := config.KC.Realm
	oidcClientID := config.KC.OIDCClientID
	oidcClientName := config.KC.OIDCClientName
	oidcClientSecret := config.KC.OIDCClientSecret

	if oidcClientID == "" || oidcClientSecret == "" {
		return nil, fmt.Errorf("OIDC client ID or secret not configured in KeycloakConfig")
	}

	log.Printf("[DEBUG] Impersonation realm: %s", realm)
	log.Printf("[DEBUG] Impersonation clientID: %s", oidcClientID)
	log.Printf("[DEBUG] Impersonation clientName: %s", oidcClientName)
	log.Printf("[DEBUG] Impersonation clientSecret: %s", oidcClientSecret)

	// Login with client credentials
	token, err := config.KC.Client.LoginClient(ctx, oidcClientName, oidcClientSecret, realm)
	if err != nil {
		log.Printf("[DEBUG] KC.Client.LoginClient failed : %s", err)
		return nil, fmt.Errorf("failed to login with client credentials: %w", err)
	}
	//log.Printf("[DEBUG] client credentials token: %s", token)
	return token, nil
}

// CheckRealmRoleExists checks if a realm role exists
func (s *keycloakService) CheckRealmRoleExists(ctx context.Context, roleName string) (bool, error) {
	if config.KC == nil || config.KC.Client == nil {
		return false, fmt.Errorf("keycloak configuration not initialized")
	}

	token, err := config.KC.GetAdminToken(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get admin token: %w", err)
	}

	_, err = config.KC.Client.GetRealmRole(ctx, token.AccessToken, config.KC.Realm, roleName)
	if err != nil {
		// Role not found
		return false, nil
	}
	return true, nil
}

// CreateRealmRole creates a realm role
func (s *keycloakService) CreateRealmRole(ctx context.Context, roleName string) error {
	if config.KC == nil || config.KC.Client == nil {
		return fmt.Errorf("keycloak configuration not initialized")
	}

	token, err := config.KC.GetAdminToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin token: %w", err)
	}

	newRole := gocloak.Role{
		Name:        &roleName,
		Description: gocloak.StringP("Platform role"),
	}

	result, err := config.KC.Client.CreateRealmRole(ctx, token.AccessToken, config.KC.Realm, newRole)
	if err != nil {
		return fmt.Errorf("failed to create realm role %s: %w", roleName, err)
	}

	log.Printf("Successfully created realm role: %s, result: %s", roleName, result)
	return nil
}

// RemoveRealmRoleFromUser removes a realm role from a user
func (s *keycloakService) RemoveRealmRoleFromUser(ctx context.Context, kcUserId, roleName string) error {
	if config.KC == nil || config.KC.Client == nil {
		return fmt.Errorf("keycloak configuration not initialized")
	}

	token, err := config.KC.GetAdminToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin token: %w", err)
	}

	// Get the role by name
	roles, err := config.KC.Client.GetRealmRoles(ctx, token.AccessToken, config.KC.Realm, gocloak.GetRoleParams{
		Search: &roleName,
	})
	if err != nil {
		return fmt.Errorf("failed to get realm role %s: %w", roleName, err)
	}
	if len(roles) == 0 {
		log.Printf("Realm role %s not found, skipping removal", roleName)
		return nil
	}
	if len(roles) > 1 {
		log.Printf("Warning: Found multiple roles matching '%s'. Using the first one.", roleName)
	}

	// Remove the role from the user
	err = config.KC.Client.DeleteRealmRoleFromUser(ctx, token.AccessToken, config.KC.Realm, kcUserId, []gocloak.Role{*roles[0]})
	if err != nil {
		return fmt.Errorf("failed to remove realm role %s from user %s: %w", roleName, kcUserId, err)
	}

	log.Printf("Successfully removed realm role %s from user %s", roleName, kcUserId)
	return nil
}

// CreateRealmRoleAndWait creates a realm role and waits for it to be available
func (s *keycloakService) CreateRealmRoleAndWait(ctx context.Context, roleName string) error {
	if config.KC == nil || config.KC.Client == nil {
		return fmt.Errorf("keycloak configuration not initialized")
	}

	token, err := config.KC.GetAdminToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin token: %w", err)
	}

	newRole := gocloak.Role{
		Name:        &roleName,
		Description: gocloak.StringP("Platform role"),
	}

	result, err := config.KC.Client.CreateRealmRole(ctx, token.AccessToken, config.KC.Realm, newRole)
	if err != nil {
		return fmt.Errorf("failed to create realm role %s: %w", roleName, err)
	}

	log.Printf("Realm role creation initiated: %s, result: %s", roleName, result)

	// 1초마다 최대 20번 시도하여 role 생성 확인
	maxRetries := 20
	for i := 0; i < maxRetries; i++ {
		time.Sleep(1 * time.Second)

		exists, err := s.CheckRealmRoleExists(ctx, roleName)
		if err != nil {
			log.Printf("Failed to check realm role existence (attempt %d/%d): %v", i+1, maxRetries, err)
			continue
		}

		if exists {
			log.Printf("Realm role %s successfully created and available (attempt %d/%d)", roleName, i+1, maxRetries)
			return nil
		}

		log.Printf("Realm role %s not yet available, waiting... (attempt %d/%d)", roleName, i+1, maxRetries)
	}

	return fmt.Errorf("realm role %s was not available after %d attempts", roleName, maxRetries)
}

// IsRealmRoleAssignedToUser checks if a specific realm role is assigned to the given user
func (s *keycloakService) IsRealmRoleAssignedToUser(ctx context.Context, kcUserId, roleName string) (bool, error) {
	if config.KC == nil || config.KC.Client == nil {
		return false, fmt.Errorf("keycloak configuration not initialized")
	}

	token, err := config.KC.GetAdminToken(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get admin token: %w", err)
	}

	roles, err := config.KC.Client.GetRealmRolesByUserID(ctx, token.AccessToken, config.KC.Realm, kcUserId)
	if err != nil {
		return false, fmt.Errorf("failed to get realm roles for user %s: %w", kcUserId, err)
	}

	for _, role := range roles {
		if role.Name != nil && *role.Name == roleName {
			return true, nil
		}
	}
	return false, nil
}

// AddRealmRoleToGroup adds a realm role to a Keycloak group (creates group if not exists)
func (s *keycloakService) AddRealmRoleToGroup(ctx context.Context, groupName, roleName string) error {
	if config.KC == nil || config.KC.Client == nil {
		return fmt.Errorf("keycloak configuration not initialized")
	}

	token, err := config.KC.GetAdminToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin token: %w", err)
	}

	// Find or create KC group
	groupID, err := s.findGroupByName(ctx, token.AccessToken, groupName)
	if err != nil {
		return err
	}
	if groupID == "" {
		log.Printf("Keycloak group '%s' not found, creating it for role assignment.", groupName)
		newGroup := gocloak.Group{Name: &groupName}
		groupID, err = config.KC.Client.CreateGroup(ctx, token.AccessToken, config.KC.Realm, newGroup)
		if err != nil {
			if strings.Contains(err.Error(), "409") {
				groupID, err = s.findGroupByName(ctx, token.AccessToken, groupName)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("failed to create keycloak group '%s': %w", groupName, err)
			}
		}
		log.Printf("Created Keycloak group '%s' (ID: %s)", groupName, groupID)
	}

	// Get realm role by name
	roles, err := config.KC.Client.GetRealmRoles(ctx, token.AccessToken, config.KC.Realm, gocloak.GetRoleParams{
		Search: &roleName,
	})
	if err != nil {
		return fmt.Errorf("failed to get realm role '%s': %w", roleName, err)
	}
	if len(roles) == 0 {
		return fmt.Errorf("realm role '%s' not found in Keycloak", roleName)
	}

	// Add role to group
	if err := config.KC.Client.AddRealmRoleToGroup(ctx, token.AccessToken, config.KC.Realm, groupID, []gocloak.Role{*roles[0]}); err != nil {
		return fmt.Errorf("failed to add realm role '%s' to group '%s': %w", roleName, groupName, err)
	}

	log.Printf("Successfully added realm role '%s' to Keycloak group '%s'", roleName, groupName)
	return nil
}

// RemoveRealmRoleFromGroup removes a realm role from a Keycloak group
func (s *keycloakService) RemoveRealmRoleFromGroup(ctx context.Context, groupName, roleName string) error {
	if config.KC == nil || config.KC.Client == nil {
		return fmt.Errorf("keycloak configuration not initialized")
	}

	token, err := config.KC.GetAdminToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin token: %w", err)
	}

	// Find KC group
	groupID, err := s.findGroupByName(ctx, token.AccessToken, groupName)
	if err != nil {
		return err
	}
	if groupID == "" {
		log.Printf("Keycloak group '%s' not found, skipping role removal", groupName)
		return nil
	}

	// Get realm role by name
	roles, err := config.KC.Client.GetRealmRoles(ctx, token.AccessToken, config.KC.Realm, gocloak.GetRoleParams{
		Search: &roleName,
	})
	if err != nil {
		return fmt.Errorf("failed to get realm role '%s': %w", roleName, err)
	}
	if len(roles) == 0 {
		log.Printf("Realm role '%s' not found in Keycloak, skipping removal", roleName)
		return nil
	}

	// Remove role from group
	if err := config.KC.Client.DeleteRealmRoleFromGroup(ctx, token.AccessToken, config.KC.Realm, groupID, []gocloak.Role{*roles[0]}); err != nil {
		return fmt.Errorf("failed to remove realm role '%s' from group '%s': %w", roleName, groupName, err)
	}

	log.Printf("Successfully removed realm role '%s' from Keycloak group '%s'", roleName, groupName)
	return nil
}

// DeleteGroup deletes a Keycloak group by name. No-op (not an error) if the group doesn't exist.
func (s *keycloakService) DeleteGroup(ctx context.Context, groupName string) error {
	if config.KC == nil || config.KC.Client == nil {
		return fmt.Errorf("keycloak configuration not initialized")
	}

	token, err := config.KC.GetAdminToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin token: %w", err)
	}

	groupID, err := s.findGroupByName(ctx, token.AccessToken, groupName)
	if err != nil {
		return err
	}
	if groupID == "" {
		log.Printf("Keycloak group '%s' not found, skipping deletion", groupName)
		return nil
	}

	if err := config.KC.Client.DeleteGroup(ctx, token.AccessToken, config.KC.Realm, groupID); err != nil {
		return fmt.Errorf("failed to delete keycloak group '%s': %w", groupName, err)
	}

	log.Printf("Successfully deleted Keycloak group '%s'", groupName)
	return nil
}

// kcProtocolMapperForService Keycloak protocol mapper 응답 구조체 (service 패키지 내)
type kcProtocolMapperForService struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	ProtocolMapper string            `json:"protocolMapper"`
	Config         map[string]string `json:"config"`
}

// CheckSAMLClientConfig Keycloak SAML 클라이언트 존재 및 protocol mapper 구성 확인
// AWS SAML 연동에 필요한 클라이언트와 Role attribute mapper가 설정되어 있는지 검증한다.
func (s *keycloakService) CheckSAMLClientConfig(ctx context.Context, clientID string) (string, error) {
	adminToken, err := config.KC.LoginAdmin(ctx)
	if err != nil {
		return "", fmt.Errorf("Keycloak admin 로그인 실패: %v", err)
	}

	realm := config.KC.Realm
	kcHost := config.KC.Host

	// 1. 클라이언트 존재 확인
	clientsURL := fmt.Sprintf("%s/admin/realms/%s/clients?clientId=%s", kcHost, realm, clientID)
	clientsResp, err := kcAdminGetRequest(ctx, clientsURL, adminToken.AccessToken)
	if err != nil {
		return "", fmt.Errorf("Keycloak 클라이언트 조회 실패: %v", err)
	}

	var clients []map[string]interface{}
	if err := json.Unmarshal(clientsResp, &clients); err != nil || len(clients) == 0 {
		return "", fmt.Errorf("SAML 클라이언트 '%s' 없음 — Keycloak에 SAML 클라이언트 생성 필요 (KEYCLOAK-AWS-SAML-SETUP.md 참조)", clientID)
	}
	kcClientID, ok := clients[0]["id"].(string)
	if !ok || kcClientID == "" {
		return "", fmt.Errorf("SAML 클라이언트 ID 파싱 실패")
	}

	// 2. Protocol mappers 확인
	mappersURL := fmt.Sprintf("%s/admin/realms/%s/clients/%s/protocol-mappers/models", kcHost, realm, kcClientID)
	mappersResp, err := kcAdminGetRequest(ctx, mappersURL, adminToken.AccessToken)
	if err != nil {
		return "", fmt.Errorf("Protocol mapper 조회 실패: %v", err)
	}

	var mappers []kcProtocolMapperForService
	if err := json.Unmarshal(mappersResp, &mappers); err != nil {
		return "", fmt.Errorf("Protocol mapper 파싱 실패: %v", err)
	}

	// Role attribute mapper 확인 (CSP SAML Role 전달용)
	hasRoleMapper := false
	for _, m := range mappers {
		if m.ProtocolMapper == "saml-role-list-mapper" || m.ProtocolMapper == "saml-hardcode-attribute-mapper" {
			hasRoleMapper = true
			break
		}
	}
	if !hasRoleMapper {
		return "", fmt.Errorf("Role attribute mapper 없음 — saml-role-list-mapper 또는 saml-hardcode-attribute-mapper 설정 필요")
	}

	mapperNames := make([]string, 0, len(mappers))
	for _, m := range mappers {
		mapperNames = append(mapperNames, m.Name)
	}
	return fmt.Sprintf("클라이언트 '%s' 존재, mappers=%v", clientID, mapperNames), nil
}

// kcAdminGetRequest Keycloak Admin API GET 요청 헬퍼 (service 패키지 내)
func kcAdminGetRequest(ctx context.Context, url, token string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
