package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Nerzal/gocloak/v13"
	"github.com/golang-jwt/jwt/v5"
	"github.com/m-cmp/mc-iam-manager/config"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository" // For ErrUserNotFound potentially
)

// KeycloakService defines operations related to Keycloak interaction.
type KeycloakService interface {
	GetUser(ctx context.Context, kcId string) (*gocloak.User, error)
	GetUserByUsername(ctx context.Context, username string) (*gocloak.User, error)
	GetUsers(ctx context.Context) ([]*gocloak.User, error)
	CreateUser(ctx context.Context, user *model.User) (string, error)
	UpdateUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, kcId string) error
	EnableUser(ctx context.Context, kcUserID string) error
	CheckAdminLogin(ctx context.Context) (bool, error)
	CheckRealm(ctx context.Context) (bool, error)
	CheckClient(ctx context.Context) (bool, error)
	GetUserIDFromToken(ctx context.Context, token *gocloak.JWT) (string, error)
	Login(ctx context.Context, username, password string) (*gocloak.JWT, error)
	RefreshToken(ctx context.Context, refreshToken string) (*gocloak.JWT, error)
	// Methods for group synchronization
	EnsureGroupExistsAndAssignUser(ctx context.Context, kcUserId, groupName string) error
	RemoveUserFromGroup(ctx context.Context, kcUserId, groupName string) error
	// Method for UMA RPT Token
	GetRequestingPartyToken(ctx context.Context, accessToken string, options gocloak.RequestingPartyTokenOptions) (*gocloak.JWT, error)
	// Method to validate token and get claims
	ValidateTokenAndGetClaims(ctx context.Context, token string) (*jwt.MapClaims, error) // Changed return type
	// SetupInitialAdmin creates the initial platform admin user and sets up necessary permissions
	SetupInitialAdmin(ctx context.Context) error
}

// keycloakService is now stateless, methods directly use config.KC
type keycloakService struct{}

// NewKeycloakService creates a new stateless KeycloakService.
func NewKeycloakService() KeycloakService {
	return &keycloakService{}
}

// GetUser retrieves a user from Keycloak by their Keycloak ID.
func (s *keycloakService) GetUser(ctx context.Context, kcId string) (*gocloak.User, error) {
	// Directly use config.KC
	if config.KC == nil || config.KC.Client == nil {
		return nil, fmt.Errorf("keycloak configuration not initialized")
	}
	token, err := config.KC.LoginAdmin(ctx) // Use admin token for broad access
	if err != nil {
		return nil, fmt.Errorf("failed to get admin token: %w", err)
	}
	user, err := config.KC.Client.GetUserByID(ctx, token.AccessToken, config.KC.Realm, kcId)
	if err != nil {
		// Check for 404 specifically
		if strings.Contains(err.Error(), "404") {
			// Consider returning a more specific error, maybe repository.ErrUserNotFound if appropriate
			return nil, fmt.Errorf("user not found in keycloak (kcId: %s): %w", kcId, repository.ErrUserNotFound)
		}
		return nil, fmt.Errorf("failed to get user from keycloak (kcId: %s): %w", kcId, err)
	}
	return user, nil
}

// GetUserByUsername retrieves a user from Keycloak by username.
func (s *keycloakService) GetUserByUsername(ctx context.Context, username string) (*gocloak.User, error) {
	// Directly use config.KC
	if config.KC == nil || config.KC.Client == nil {
		return nil, fmt.Errorf("keycloak configuration not initialized")
	}
	token, err := config.KC.LoginAdmin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin token: %v", err)
	}
	users, err := config.KC.Client.GetUsers(ctx, token.AccessToken, config.KC.Realm, gocloak.GetUsersParams{
		Username: gocloak.StringP(username),
		Exact:    gocloak.BoolP(true), // Ensure exact match
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user from keycloak by username %s: %w", username, err)
	}
	if len(users) == 0 {
		return nil, repository.ErrUserNotFound // Use repository error
	}
	if len(users) > 1 {
		log.Printf("Warning: Found multiple users with username %s in Keycloak", username)
	}
	return users[0], nil
}

// GetUsers retrieves all users from Keycloak.
func (s *keycloakService) GetUsers(ctx context.Context) ([]*gocloak.User, error) {
	// Directly use config.KC
	if config.KC == nil || config.KC.Client == nil {
		return nil, fmt.Errorf("keycloak configuration not initialized")
	}
	token, err := config.KC.LoginAdmin(ctx) // Use admin token
	if err != nil {
		return nil, fmt.Errorf("failed to get admin token: %w", err)
	}
	// Consider pagination for large numbers of users
	getUsersParams := gocloak.GetUsersParams{}
	kcUsers, err := config.KC.Client.GetUsers(ctx, token.AccessToken, config.KC.Realm, getUsersParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get users from keycloak: %w", err)
	}
	return kcUsers, nil
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
		options.Audience = &config.KC.ClientID
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
		ClientID: &config.KC.ClientID,
	})
	if err != nil {
		log.Printf("[DEBUG] Failed to get client info: %v", err)
		return false, fmt.Errorf("failed to get client info: %w", err)
	}
	if len(clients) == 0 {
		log.Printf("[DEBUG] Client '%s' not found", config.KC.ClientID)
		return false, fmt.Errorf("client '%s' not found", config.KC.ClientID)
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
	clients, err := config.KC.Client.GetClients(ctx, token.AccessToken, config.KC.Realm, gocloak.GetClientsParams{ClientID: &config.KC.ClientID})
	if err != nil {
		return false, fmt.Errorf("failed to get client '%s': %w", config.KC.ClientID, err)
	}
	if len(clients) == 0 {
		return false, fmt.Errorf("client '%s' not found", config.KC.ClientID)
	}
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
	log.Printf("[DEBUG] - ClientID: %s", config.KC.ClientID)
	log.Printf("[DEBUG] - ClientSecret: %s", config.KC.ClientSecret)
	log.Printf("[DEBUG] - Username: %s", username)
	log.Printf("[DEBUG] - password: %s", password)

	token, err := config.KC.Client.Login(ctx, config.KC.ClientID, config.KC.ClientSecret, config.KC.Realm, username, password)
	if err != nil {
		// Consider more specific error handling for invalid credentials vs other errors
		return nil, fmt.Errorf("keycloak login failed: %w", err)
	}
	return token, nil
}

// RefreshToken refreshes the JWT token using a refresh token.
func (s *keycloakService) RefreshToken(ctx context.Context, refreshToken string) (*gocloak.JWT, error) {
	// Directly use config.KC
	if config.KC == nil || config.KC.Client == nil {
		return nil, fmt.Errorf("keycloak configuration not initialized")
	}
	newToken, err := config.KC.Client.RefreshToken(ctx, refreshToken, config.KC.ClientID, config.KC.ClientSecret, config.KC.Realm)
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
func (s *keycloakService) SetupInitialAdmin(ctx context.Context) error {
	if config.KC == nil || config.KC.Client == nil {
		return fmt.Errorf("keycloak configuration not initialized")
	}

	// 1. Admin 로그인
	log.Printf("[DEBUG] Attempting to login as admin")
	adminToken, err := config.KC.LoginAdmin(ctx)
	if err != nil {
		return fmt.Errorf("admin login failed: %w", err)
	}
	log.Printf("[DEBUG] Admin login successful")

	// 2. platformAdmin 사용자 생성
	platformAdminID := os.Getenv("MCIAMMANAGER_PLATFORMADMIN_ID")
	platformAdminPassword := os.Getenv("MCIAMMANAGER_PLATFORMADMIN_PASSWORD")
	platformAdminFirstName := os.Getenv("MCIAMMANAGER_PLATFORMADMIN_FIRSTNAME")
	platformAdminLastName := os.Getenv("MCIAMMANAGER_PLATFORMADMIN_LASTNAME")
	platformAdminEmail := os.Getenv("MCIAMMANAGER_PLATFORMADMIN_EMAIL")

	if platformAdminID == "" {
		return fmt.Errorf("MCIAMMANAGER_PLATFORMADMIN_ID not set in environment variables")
	}
	if platformAdminPassword == "" {
		return fmt.Errorf("MCIAMMANAGER_PLATFORMADMIN_PASSWORD not set in environment variables")
	}
	if platformAdminFirstName == "" {
		return fmt.Errorf("MCIAMMANAGER_PLATFORMADMIN_FIRSTNAME not set in environment variables")
	}
	if platformAdminLastName == "" {
		return fmt.Errorf("MCIAMMANAGER_PLATFORMADMIN_LASTNAME not set in environment variables")
	}
	if platformAdminEmail == "" {
		return fmt.Errorf("MCIAMMANAGER_PLATFORMADMIN_EMAIL not set in environment variables")
	}

	log.Printf("[DEBUG] Creating platform admin user: %s", platformAdminID)
	user := gocloak.User{
		Username:      &platformAdminID,
		FirstName:     &platformAdminFirstName,
		LastName:      &platformAdminLastName,
		Email:         &platformAdminEmail,
		Enabled:       gocloak.BoolP(true),
		EmailVerified: gocloak.BoolP(true),
	}

	userID, err := config.KC.Client.CreateUser(ctx, adminToken.AccessToken, config.KC.Realm, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	log.Printf("[DEBUG] User created with ID: %s", userID)

	// 비밀번호 설정
	err = config.KC.Client.SetPassword(ctx, adminToken.AccessToken, userID, config.KC.Realm, platformAdminPassword, false)
	if err != nil {
		return fmt.Errorf("failed to set password: %w", err)
	}
	log.Printf("[DEBUG] Password set successfully")

	// 3. platformAdmin 역할을 기본 역할로 설정
	log.Printf("[DEBUG] Setting platformAdmin as default role")
	platformAdminRole, err := config.KC.Client.GetRealmRole(ctx, adminToken.AccessToken, config.KC.Realm, "platformAdmin")
	if err != nil {
		return fmt.Errorf("failed to get platformAdmin role: %w", err)
	}

	// platformAdmin 역할을 기본 역할로 설정
	err = config.KC.Client.AddRealmRoleToUser(ctx, adminToken.AccessToken, config.KC.Realm, userID, []gocloak.Role{*platformAdminRole})
	if err != nil {
		return fmt.Errorf("failed to assign platformAdmin role: %w", err)
	}
	log.Printf("[DEBUG] PlatformAdmin role assigned as default role")

	// 4. 필요한 권한 부여 (Realm 내부 관리자 권한)
	requiredPermissions := []string{
		"view-users",
		"view-identity-providers",
		"view-clients",
		"view-events",
		"view-realm",
		"view-authorization",
		"manage-users",
		"manage-identity-providers",
		"manage-clients",
		"manage-events",
		"manage-realm",
		"manage-authorization",
		"impersonation",
		"create-client",
		"manage-account",
		"manage-account-links",
		"view-profile",
	}

	// 클라이언트 정보 가져오기
	clients, err := config.KC.Client.GetClients(ctx, adminToken.AccessToken, config.KC.Realm, gocloak.GetClientsParams{
		ClientID: &config.KC.ClientID,
	})
	if err != nil {
		return fmt.Errorf("failed to get client info: %w", err)
	}
	if len(clients) == 0 {
		return fmt.Errorf("client '%s' not found", config.KC.ClientID)
	}
	clientID := *clients[0].ID

	// 각 권한에 대해 부여
	for _, permission := range requiredPermissions {
		log.Printf("[DEBUG] Assigning permission: %s", permission)

		// 권한을 클라이언트 역할로 변환하여 부여
		clientRole := gocloak.Role{
			Name: &permission,
		}

		err = config.KC.Client.AddClientRoleToUser(ctx, adminToken.AccessToken, config.KC.Realm, clientID, userID, []gocloak.Role{clientRole})
		if err != nil {
			log.Printf("[DEBUG] Failed to assign permission %s: %v", permission, err)
			continue
		}

		log.Printf("[DEBUG] Successfully assigned permission: %s", permission)
	}

	return nil
}
