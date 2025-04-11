package service

import (
	"context" // Ensure errors is imported
	"fmt"
	"log" // Added for logging in SyncPlatformAdmin
	"os"
	"time"

	"github.com/Nerzal/gocloak/v13"
	"github.com/m-cmp/mc-iam-manager/config"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	// Added for gorm.ErrRecordNotFound
)

type UserService struct {
	userRepo          *repository.UserRepository
	platformRoleRepo  *repository.PlatformRoleRepository // Added dependency
	workspaceRoleRepo *repository.WorkspaceRoleRepository
	tokenRepo         *repository.TokenRepository // Assuming this exists and is needed for getValidToken
	keycloakClient    *gocloak.GoCloak
	keycloakConfig    *config.KeycloakConfig
}

// NewUserService constructor needs to accept new dependencies
func NewUserService(
	userRepo *repository.UserRepository,
	platformRoleRepo *repository.PlatformRoleRepository, // Added
	// workspaceRoleRepo *repository.WorkspaceRoleRepository, // Add if needed
	// tokenRepo *repository.TokenRepository, // Add if needed
	keycloakConfig *config.KeycloakConfig,
	keycloakClient *gocloak.GoCloak,
) *UserService {
	return &UserService{
		userRepo:          userRepo,
		platformRoleRepo:  platformRoleRepo, // Initialize
		workspaceRoleRepo: nil,              // Initialize if needed
		tokenRepo:         nil,              // Initialize if needed
		keycloakClient:    keycloakClient,
		keycloakConfig:    keycloakConfig,
	}
}

// SyncPlatformAdmin ensures the platform superadmin from .env exists and has the correct role
func (s *UserService) SyncPlatformAdmin(ctx context.Context) error {
	adminUsername := os.Getenv("MCIAMMANAGER_PLATFORMADMIN_ID")
	if adminUsername == "" {
		log.Println("Warning: MCIAMMANAGER_PLATFORMADMIN_ID not set in .env, skipping superadmin sync.")
		return nil // Not a fatal error if not set
	}

	log.Printf("Syncing platform superadmin: %s", adminUsername)

	// 1. Find user in Keycloak by username
	kcUser, err := s.userRepo.GetUserByUsername(ctx, adminUsername)
	if err != nil {
		log.Printf("Error finding superadmin '%s' in Keycloak: %v. Please ensure the user exists in Keycloak.", adminUsername, err)
		return fmt.Errorf("superadmin user '%s' must exist in Keycloak", adminUsername) // Make it an error to stop startup?
	}
	if kcUser == nil { // Should be covered by err check, but double-check
		log.Printf("Superadmin user '%s' not found in Keycloak. Please ensure the user exists.", adminUsername)
		return fmt.Errorf("superadmin user '%s' must exist in Keycloak", adminUsername)
	}

	// 2. Ensure user exists in local DB (mcmp_users) using Count()
	var count int64
	var dbUser model.User // Declare dbUser here for later use
	err = s.userRepo.DB().Model(&model.User{}).Where("kc_id = ?", kcUser.ID).Count(&count).Error
	if err != nil {
		log.Printf("Error counting superadmin '%s' in local DB: %v", adminUsername, err)
		return fmt.Errorf("failed to count superadmin in local DB: %w", err)
	}

	if count == 0 {
		// User not found, create entry
		log.Printf("Superadmin '%s' not found in local DB, creating entry...", adminUsername)
		// Log kcUser details before creating dbUserToCreate
		log.Printf("[DEBUG] kcUser data before creating local record: %+v", kcUser)
		dbUserToCreate := model.User{
			KcId:     kcUser.ID,
			Username: kcUser.Username, // Ensure Username is included
			// Email:       kcUser.Email,    // Email is ignored by gorm:"-" in model
			FirstName:   kcUser.FirstName,
			LastName:    kcUser.LastName,
			Description: kcUser.Description, // Include Description
		}
		// Log the data being sent to DB Create
		log.Printf("[DEBUG] Attempting to create dbUserToCreate: %+v", dbUserToCreate)
		// Use map to explicitly specify columns for Create, ensuring username is included
		userDataToCreate := map[string]interface{}{
			"kc_id":    dbUserToCreate.KcId,
			"username": dbUserToCreate.Username,
			// "email":       dbUserToCreate.Email, // Remove email as column doesn't exist
			//"first_name":  dbUserToCreate.FirstName,
			//"last_name":   dbUserToCreate.LastName,
			"description": dbUserToCreate.Description,
		}
		log.Printf("[DEBUG] Data being passed to GORM Create (map): %+v", userDataToCreate)
		// Create using map on the specific model type
		if errCreate := s.userRepo.DB().Model(&model.User{}).Create(userDataToCreate).Error; errCreate != nil {
			// Corrected log format string and added missing argument
			log.Printf("Error creating superadmin '%s' in local DB: %v", adminUsername, errCreate)
			return fmt.Errorf("failed to create superadmin in local DB: %w", errCreate)
		}
		// Need to fetch the created user again to get the DbId
		err = s.userRepo.DB().Where("kc_id = ?", kcUser.ID).First(&dbUser).Error
		if err != nil {
			log.Printf("Error fetching newly created superadmin '%s' from local DB: %v", adminUsername, err)
			return fmt.Errorf("failed to fetch newly created superadmin from local DB: %w", err)
		}
		log.Printf("Superadmin '%s' entry created in local DB. DB ID: %d", adminUsername, dbUser.DbId)
	} else {
		// User exists, fetch the full record including DbId
		err = s.userRepo.DB().Where("kc_id = ?", kcUser.ID).First(&dbUser).Error
		if err != nil {
			log.Printf("Error fetching existing superadmin '%s' from local DB: %v", adminUsername, err)
			return fmt.Errorf("failed to fetch existing superadmin from local DB: %w", err)
		}
	}

	// 3. Ensure 'platformadmin' role exists
	adminRoleName := "platformadmin"                         // Changed variable name and value
	role, err := s.platformRoleRepo.GetByName(adminRoleName) // Use PlatformRoleRepository with new name
	if err != nil {
		// Assuming GetByName returns specific error string for not found
		if err.Error() == "platform role not found" { // Check error string
			log.Printf("Error: '%s' role not found in mcmp_platform_roles. Run migration 000010 first.", adminRoleName)
		} else {
			log.Printf("Error fetching '%s' role: %v", adminRoleName, err)
		}
		return fmt.Errorf("failed to find '%s' role: %w", adminRoleName, err)
	}

	// 4. Assign 'platformadmin' role to the user if not already assigned
	if dbUser.DbId == 0 {
		// Fetch again to ensure DbId is populated if it was just created
		err = s.userRepo.DB().Where("kc_id = ?", kcUser.ID).First(&dbUser).Error
		if err != nil || dbUser.DbId == 0 {
			log.Printf("Error: Could not get valid DB ID for admin '%s' after creation/check. Cannot assign role.", adminUsername) // Changed log message
			return fmt.Errorf("could not get valid DB ID for admin '%s'", adminUsername)                                           // Changed error message
		}
	}

	// Check if association already exists
	var currentRoles []model.PlatformRole
	// Use userRepo's DB method and check association
	if err := s.userRepo.DB().Model(&dbUser).Association("PlatformRoles").Find(&currentRoles); err != nil {
		log.Printf("Error checking existing roles for admin '%s': %v", adminUsername, err) // Changed log message
		return fmt.Errorf("failed to check existing roles for admin: %w", err)             // Changed error message
	}

	hasAdminRole := false // Changed variable name
	for _, r := range currentRoles {
		if r.ID == role.ID {
			hasAdminRole = true // Changed variable name
			break
		}
	}

	if !hasAdminRole { // Changed variable name
		log.Printf("Assigning '%s' role to admin '%s'...", adminRoleName, adminUsername) // Changed log message
		// Use userRepo's DB method to append association
		if err := s.userRepo.DB().Model(&dbUser).Association("PlatformRoles").Append(role); err != nil {
			log.Printf("Error assigning '%s' role to admin '%s': %v", adminRoleName, adminUsername, err) // Changed log message
			return fmt.Errorf("failed to assign '%s' role: %w", adminRoleName, err)                      // Changed error message
		}
		log.Printf("Successfully assigned '%s' role to admin '%s'.", adminRoleName, adminUsername) // Changed log message
	} else {
		log.Printf("Admin '%s' already has '%s' role.", adminUsername, adminRoleName) // Changed log message
	}

	return nil
}

// --- Existing UserService methods below ---

// getValidToken (Keep as is, assuming tokenRepo is initialized if needed)
func (s *UserService) getValidToken(ctx context.Context) (string, error) {
	if s.tokenRepo == nil {
		// If tokenRepo is not essential for all UserService operations, handle its absence.
		// Otherwise, ensure it's initialized in NewUserService.
		// For now, assume it might be needed elsewhere or refactor if not.
		// Fallback to direct login if token repo is unavailable.
		tokenResponse, err := s.keycloakClient.LoginClient(ctx, s.keycloakConfig.ClientID, s.keycloakConfig.ClientSecret, s.keycloakConfig.Realm)
		if err != nil {
			return "", fmt.Errorf("failed to get token via login: %v", err)
		}
		return tokenResponse.AccessToken, nil
	}

	token, err := s.tokenRepo.GetTokenByUserID(s.keycloakConfig.ClientID)
	if err == nil {
		if time.Until(token.ExpiresAt) > 10*time.Minute {
			return token.Token, nil
		}
	}

	tokenResponse, err := s.keycloakClient.LoginClient(ctx, s.keycloakConfig.ClientID, s.keycloakConfig.ClientSecret, s.keycloakConfig.Realm)
	if err != nil {
		return "", fmt.Errorf("failed to get token: %v", err)
	}

	if err := s.tokenRepo.SaveToken(s.keycloakConfig.ClientID, tokenResponse.AccessToken, int64(tokenResponse.ExpiresIn)); err != nil {
		// Log error but return token anyway
		fmt.Printf("Warning: failed to save token: %v\n", err)
		// return "", fmt.Errorf("failed to save token: %v", err)
	}

	return tokenResponse.AccessToken, nil
}

// GetUsers returns a list of users
func (s *UserService) GetUsers(ctx context.Context) ([]model.User, error) {
	// This method likely needs updating to use userRepo.GetUsers which combines KC and DB data
	return s.userRepo.GetUsers(ctx) // Delegate to repository
}

// GetUserByID returns a user by ID
func (s *UserService) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	return s.userRepo.GetUserByID(ctx, id)
}

// GetUserByUsername returns a user by username from Keycloak
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	return s.userRepo.GetUserByUsername(ctx, username)
}

// CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, user *model.User) error {
	// Add potential validation or business logic here
	return s.userRepo.CreateUser(ctx, user)
}

// UpdateUser updates an existing user
func (s *UserService) UpdateUser(ctx context.Context, user *model.User) error {
	// Add potential validation or business logic here
	return s.userRepo.UpdateUser(ctx, user)
}

// DeleteUser deletes a user
func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	// Add potential validation or business logic here
	return s.userRepo.DeleteUser(ctx, id)
}

// GetUser returns a user by ID (Duplicate of GetUserByID?)
// func (s *UserService) GetUser(ctx context.Context, id string) (*model.User, error) {
// 	return s.userRepo.GetUserByID(ctx, id)
// }

// ApproveUser enables a user in Keycloak and ensures they exist in the local DB.
func (s *UserService) ApproveUser(ctx context.Context, kcUserID string) error {
	// 1. Enable user in Keycloak
	err := s.userRepo.EnableUserInKeycloak(ctx, kcUserID)
	if err != nil {
		return fmt.Errorf("failed to enable user in keycloak: %w", err)
	}

	// 2. Ensure user exists in local DB (sync)
	_, err = s.userRepo.SyncUser(ctx, kcUserID)
	if err != nil {
		// Log warning but consider approval successful if Keycloak enable worked
		fmt.Printf("Warning: User %s enabled in Keycloak, but failed to sync/create in local DB: %v\n", kcUserID, err)
		// return fmt.Errorf("failed to sync user to local db after enabling: %w", err)
	}

	// TODO: Assign default role(s) if needed upon approval

	return nil
}
