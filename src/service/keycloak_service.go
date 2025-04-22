package service

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Nerzal/gocloak/v13"
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
	Login(ctx context.Context, username, password string) (*gocloak.JWT, error)  // Added Login method
	RefreshToken(ctx context.Context, refreshToken string) (*gocloak.JWT, error) // Added RefreshToken method
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
		return false, fmt.Errorf("keycloak configuration not initialized")
	}
	token, err := config.KC.LoginAdmin(ctx)
	if err != nil {
		return false, fmt.Errorf("admin login failed, cannot check realm: %w", err)
	}
	_, err = config.KC.Client.GetRealm(ctx, token.AccessToken, config.KC.Realm)
	if err != nil {
		return false, fmt.Errorf("failed to get realm '%s': %w", config.KC.Realm, err)
	}
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
