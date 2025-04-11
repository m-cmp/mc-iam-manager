package repository

import (
	"context"
	"errors" // Ensure errors package is imported
	"fmt"
	"log"

	"github.com/m-cmp/mc-iam-manager/config"
	"github.com/m-cmp/mc-iam-manager/model"

	"github.com/Nerzal/gocloak/v13"
	"gorm.io/gorm" // Add GORM import
)

type UserRepository struct {
	db             *gorm.DB // Changed to *gorm.DB
	keycloakConfig *config.KeycloakConfig
	keycloakClient *gocloak.GoCloak
}

func NewUserRepository(db *gorm.DB, keycloakConfig *config.KeycloakConfig, keycloakClient *gocloak.GoCloak) *UserRepository { // Changed db type
	return &UserRepository{
		db:             db, // Assign gorm.DB
		keycloakConfig: keycloakConfig,
		keycloakClient: keycloakClient,
	}
}

func (r *UserRepository) GetUsers(ctx context.Context) ([]model.User, error) {
	log.Println("[DEBUG] GetUsers: Attempting to get Keycloak token...") // Log: Start token fetch
	token, err := r.keycloakConfig.GetToken(ctx)
	if err != nil {
		log.Printf("[ERROR] GetUsers: Failed to get Keycloak token: %v\n", err) // Log: Token fetch error
		return nil, fmt.Errorf("failed to get token: %w", err)                  // Wrap error
	}
	log.Println("[DEBUG] GetUsers: Successfully obtained Keycloak token.") // Log: Token fetch success

	log.Printf("[DEBUG] GetUsers: Attempting to get users from Keycloak realm '%s'...\n", r.keycloakConfig.Realm) // Log: Start KC user fetch
	getUsersParams := gocloak.GetUsersParams{}                                                                    // Define params for logging if needed
	users, err := r.keycloakClient.GetUsers(ctx, token.AccessToken, r.keycloakConfig.Realm, getUsersParams)
	if err != nil {
		log.Printf("[ERROR] GetUsers: Failed to get users from Keycloak: %v\n", err) // Log: KC user fetch error
		return nil, fmt.Errorf("failed to get users: %w", err)                       // Wrap error
	}
	log.Printf("[DEBUG] GetUsers: Successfully retrieved %d users from Keycloak.\n", len(users)) // Log: KC user fetch success

	if len(users) == 0 {
		return []model.User{}, nil
	}

	// Extract Keycloak IDs
	kcIDs := make([]string, 0, len(users))
	keycloakUserMap := make(map[string]*gocloak.User, len(users))
	for _, u := range users {
		if u != nil && u.ID != nil {
			kcIDs = append(kcIDs, *u.ID)
			keycloakUserMap[*u.ID] = u // Map for easy lookup
		}
	}

	if len(kcIDs) == 0 {
		return []model.User{}, nil
	}

	// Fetch corresponding users from DB using KcIds
	var dbUsers []model.User
	// Use Preload to fetch roles efficiently
	log.Printf("[DEBUG] GetUsers: Attempting to fetch %d users from local DB by KcIDs...\n", len(kcIDs)) // Log: Start DB fetch
	if errDb := r.db.Preload("PlatformRoles").Preload("WorkspaceRoles").Where("kc_id IN ?", kcIDs).Find(&dbUsers).Error; errDb != nil {
		// Log error but potentially return only Keycloak data
		log.Printf("[ERROR] GetUsers: Error fetching user details from local db for multiple users: %v\n", errDb) // Log: DB fetch error
		// Fallback to returning only Keycloak data
		var result []model.User
		for _, kcID := range kcIDs {
			if u, ok := keycloakUserMap[kcID]; ok {
				result = append(result, model.User{
					ID:        *u.ID,
					KcId:      *u.ID,
					Username:  *u.Username,
					Email:     *u.Email,
					FirstName: *u.FirstName,
					LastName:  *u.LastName,
					Enabled:   *u.Enabled,
				})
			}
		}
		return result, nil // Return partial data
	}

	// Create a map of DB users for efficient merging
	dbUserMap := make(map[string]*model.User, len(dbUsers))
	for i := range dbUsers {
		dbUserMap[dbUsers[i].KcId] = &dbUsers[i]
	}

	// Merge Keycloak data with DB data
	var result []model.User
	for _, kcID := range kcIDs {
		kcUser, kcExists := keycloakUserMap[kcID]
		if !kcExists {
			continue // Should not happen if kcIDs were built correctly
		}

		mergedUser := model.User{
			ID:        *kcUser.ID,
			KcId:      *kcUser.ID,
			Username:  *kcUser.Username,
			Email:     *kcUser.Email,
			FirstName: *kcUser.FirstName,
			LastName:  *kcUser.LastName,
			Enabled:   *kcUser.Enabled,
		}

		if dbUser, dbExists := dbUserMap[kcID]; dbExists {
			mergedUser.DbId = dbUser.DbId
			mergedUser.Description = dbUser.Description
			mergedUser.CreatedAt = dbUser.CreatedAt
			mergedUser.UpdatedAt = dbUser.UpdatedAt
			mergedUser.PlatformRoles = dbUser.PlatformRoles
			mergedUser.WorkspaceRoles = dbUser.WorkspaceRoles
		} else {
			// User exists in Keycloak but not DB (log warning)
			fmt.Printf("Warning: User found in Keycloak but not in local db (kc_id: %s)\n", kcID)
		}
		result = append(result, mergedUser)
	}

	return result, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	token, err := r.keycloakConfig.LoginAdmin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %v", err)
	}

	user, err := r.keycloakClient.GetUserByID(ctx, token.AccessToken, r.keycloakConfig.Realm, id)
	if err != nil {
		// Handle specific Keycloak errors if needed (e.g., 404 Not Found)
		return nil, fmt.Errorf("failed to get user from keycloak: %w", err)
	}

	// Create the base user model from Keycloak data
	resultUser := &model.User{
		ID:        *user.ID, // Keycloak ID
		KcId:      *user.ID, // Store KcId as well
		Username:  *user.Username,
		Email:     *user.Email,
		FirstName: *user.FirstName,
		LastName:  *user.LastName,
		Enabled:   *user.Enabled,
	}

	// Find corresponding user in local DB using KcId to get DbId and Description
	var dbUser model.User
	// Preload roles along with finding the user
	if errDb := r.db.Preload("PlatformRoles").Preload("WorkspaceRoles").Where("kc_id = ?", resultUser.KcId).First(&dbUser).Error; errDb != nil {
		// If user not found in DB, log warning but return Keycloak data
		// This handles cases where DB sync might be lagging or failed previously
		if errors.Is(errDb, gorm.ErrRecordNotFound) {
			fmt.Printf("Warning: User found in Keycloak but not in local db (kc_id: %s)\n", resultUser.KcId)
			// Return user data from Keycloak without roles/description
			return resultUser, nil
		}
		// For other DB errors, log and potentially return error
		fmt.Printf("Error fetching user details from local db (kc_id: %s): %v\n", resultUser.KcId, errDb)
		// Depending on requirements, you might return partial data or an error
		// return nil, fmt.Errorf("error fetching user details from db: %w", errDb)
		return resultUser, nil // Return Keycloak data even if DB fetch failed
	}

	// Populate additional fields from DB record
	resultUser.DbId = dbUser.DbId
	resultUser.Description = dbUser.Description
	resultUser.CreatedAt = dbUser.CreatedAt // Populate DB timestamps
	resultUser.UpdatedAt = dbUser.UpdatedAt
	resultUser.PlatformRoles = dbUser.PlatformRoles   // Assign preloaded roles
	resultUser.WorkspaceRoles = dbUser.WorkspaceRoles // Assign preloaded roles

	return resultUser, nil
}

func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	token, err := r.keycloakConfig.LoginAdmin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %v", err)
	}

	users, err := r.keycloakClient.GetUsers(ctx, token.AccessToken, r.keycloakConfig.Realm, gocloak.GetUsersParams{
		Username: gocloak.StringP(username),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	kcUser := users[0] // Use different variable name

	// Create the base user model from Keycloak data
	resultUser := &model.User{
		ID:        *kcUser.ID, // Keycloak ID
		KcId:      *kcUser.ID, // Store KcId as well
		Username:  *kcUser.Username,
		Email:     *kcUser.Email,
		FirstName: *kcUser.FirstName,
		LastName:  *kcUser.LastName,
		Enabled:   *kcUser.Enabled,
	}

	// Find corresponding user in local DB using KcId
	var dbUser model.User
	if errDb := r.db.Preload("PlatformRoles").Preload("WorkspaceRoles").Where("kc_id = ?", resultUser.KcId).First(&dbUser).Error; errDb != nil {
		if errors.Is(errDb, gorm.ErrRecordNotFound) {
			fmt.Printf("Warning: User found in Keycloak but not in local db (kc_id: %s)\n", resultUser.KcId)
			return resultUser, nil
		}
		fmt.Printf("Error fetching user details from local db (kc_id: %s): %v\n", resultUser.KcId, errDb)
		return resultUser, nil // Return Keycloak data even if DB fetch failed
	}

	// Populate additional fields from DB record
	resultUser.DbId = dbUser.DbId
	resultUser.Description = dbUser.Description
	resultUser.CreatedAt = dbUser.CreatedAt
	resultUser.UpdatedAt = dbUser.UpdatedAt
	resultUser.PlatformRoles = dbUser.PlatformRoles
	resultUser.WorkspaceRoles = dbUser.WorkspaceRoles

	return resultUser, nil
}

func (r *UserRepository) CreateUser(ctx context.Context, user *model.User) error {
	token, err := r.keycloakConfig.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get token: %v", err)
	}

	keycloakUser := gocloak.User{
		Username:      &user.Username,
		Email:         &user.Email,
		FirstName:     &user.FirstName,
		LastName:      &user.LastName,
		Enabled:       gocloak.BoolP(true),
		EmailVerified: gocloak.BoolP(true),
	}

	userID, err := r.keycloakClient.CreateUser(ctx, token.AccessToken, r.keycloakConfig.Realm, keycloakUser)
	if err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}

	user.ID = userID // Keycloak ID

	// Save user info (including KcId) to local DB using map
	userDataToCreate := map[string]interface{}{
		"kc_id":    userID,
		"username": user.Username,
		// "email":       user.Email,    // Removed based on schema change
		// "first_name":  user.FirstName, // Removed
		// "last_name":   user.LastName, // Removed
		"description": user.Description,
	}
	log.Printf("[DEBUG] Attempting to create user data in CreateUser (map): %+v", userDataToCreate)
	if err := r.db.Model(&model.User{}).Create(userDataToCreate).Error; err != nil {
		// TODO: Consider rollback or compensation logic for Keycloak user creation if DB save fails
		return fmt.Errorf("failed to save user to local db after keycloak creation: %w", err)
	}
	// Need to fetch the DbId after creation if needed elsewhere
	// For now, we don't assign it back to the input 'user' model

	return nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, user *model.User) error {
	token, err := r.keycloakConfig.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get token: %v", err)
	}

	keycloakUser := gocloak.User{
		ID:            &user.ID,
		Username:      &user.Username,
		Email:         &user.Email,
		FirstName:     &user.FirstName,
		LastName:      &user.LastName,
		Enabled:       gocloak.BoolP(true),
		EmailVerified: gocloak.BoolP(true),
	}

	err = r.keycloakClient.UpdateUser(ctx, token.AccessToken, r.keycloakConfig.Realm, keycloakUser)
	if err != nil {
		return fmt.Errorf("failed to update user in keycloak: %w", err)
	}

	// Update user info in local DB (e.g., description) based on KcId
	// We need to find the user by KcId first to get the DbId for update
	var dbUser model.User
	if errDb := r.db.Where("kc_id = ?", user.ID).First(&dbUser).Error; errDb != nil {
		// Log error but potentially continue if Keycloak update succeeded
		// Or return error depending on desired consistency level
		fmt.Printf("Warning: failed to find user in local db (kc_id: %s) for update: %v\n", user.ID, errDb)
		// return fmt.Errorf("failed to find user in local db for update: %w", errDb)
	} else {
		// Update only specific fields, e.g., Description
		if errDbUpdate := r.db.Model(&dbUser).Update("description", user.Description).Error; errDbUpdate != nil {
			// Log error but potentially continue
			fmt.Printf("Warning: failed to update user description in local db (kc_id: %s): %v\n", user.ID, errDbUpdate)
			// return fmt.Errorf("failed to update user in local db: %w", errDbUpdate)
		}
	}

	return nil
}

func (r *UserRepository) DeleteUser(ctx context.Context, id string) error {
	token, err := r.keycloakConfig.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get token: %v", err)
	}

	err = r.keycloakClient.DeleteUser(ctx, token.AccessToken, r.keycloakConfig.Realm, id)
	if err != nil {
		// Log error but potentially continue if Keycloak delete succeeded,
		// or return error depending on desired consistency.
		// If Keycloak user doesn't exist, gocloak might return an error.
		// We might want to proceed with DB deletion even if Keycloak deletion fails or user not found there.
		fmt.Printf("Warning: failed to delete user from keycloak (id: %s): %v. Attempting to delete from local db.\n", id, err)
		// return fmt.Errorf("failed to delete user from keycloak: %w", err)
	}

	// Delete user from local DB based on KcId
	result := r.db.Where("kc_id = ?", id).Delete(&model.User{})
	if result.Error != nil {
		// Log error but potentially consider the overall operation successful if Keycloak delete worked.
		fmt.Printf("Warning: failed to delete user from local db (kc_id: %s): %v\n", id, result.Error)
		// return fmt.Errorf("failed to delete user from local db: %w", result.Error)
	}
	// We might not want to return an error if the user was already gone from the local DB.
	// if result.RowsAffected == 0 {
	// 	 fmt.Printf("Info: User not found in local db for deletion (kc_id: %s)\n", id)
	// }

	return nil // Return nil even if DB deletion had issues, assuming Keycloak is primary
}

// DB returns the underlying gorm DB instance (Helper for sync function)
func (r *UserRepository) DB() *gorm.DB {
	return r.db
}

// SyncUser ensures a user record exists in the local DB for the given Keycloak ID.
// If not found, it fetches info from Keycloak and creates the local record.
// It returns the local DB user model (or error).
func (r *UserRepository) SyncUser(ctx context.Context, kcUserID string) (*model.User, error) {
	var dbUser model.User
	err := r.db.Where("kc_id = ?", kcUserID).First(&dbUser).Error

	if err == nil {
		// User found in DB, return it
		return &dbUser, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Other DB error occurred
		return nil, fmt.Errorf("error checking user in local db (kc_id: %s): %w", kcUserID, err)
	}

	// User not found in DB, need to create entry
	log.Printf("User '%s' not found in local DB, syncing from Keycloak...", kcUserID)

	// Fetch user details from Keycloak (requires admin token)
	token, tokenErr := r.keycloakConfig.LoginAdmin(ctx)
	if tokenErr != nil {
		return nil, fmt.Errorf("failed to get admin token for user sync: %w", tokenErr)
	}
	kcUser, kcErr := r.keycloakClient.GetUserByID(ctx, token.AccessToken, r.keycloakConfig.Realm, kcUserID)
	if kcErr != nil {
		return nil, fmt.Errorf("failed to get user details from keycloak for sync (id: %s): %w", kcUserID, kcErr)
	}
	if kcUser == nil { // Should be covered by error, but double check
		return nil, fmt.Errorf("user %s exists in token but not found in keycloak detail fetch", kcUserID)
	}

	// Create local DB record using map
	userDataToCreate := map[string]interface{}{
		"kc_id":    *kcUser.ID,
		"username": *kcUser.Username,
		// "email":       *kcUser.Email, // Removed based on schema change
		// "first_name":  *kcUser.FirstName, // Removed
		// "last_name":   *kcUser.LastName, // Removed
		"description": "", // Add default or fetch from attributes if needed
	}
	log.Printf("[DEBUG] Attempting to create user data in SyncUser (map): %+v", userDataToCreate)
	if errCreate := r.db.Model(&model.User{}).Create(userDataToCreate).Error; errCreate != nil {
		return nil, fmt.Errorf("failed to create user in local db during sync (kc_id: %s): %w", kcUserID, errCreate)
	}

	// Fetch the newly created record to return it (including DbId)
	var newlyCreatedUser model.User
	if errFetch := r.db.Where("kc_id = ?", kcUserID).First(&newlyCreatedUser).Error; errFetch != nil {
		log.Printf("Warning: Failed to fetch newly created user %s from local DB: %v", kcUserID, errFetch)
		// Return a minimal user object based on Keycloak data as fallback
		return &model.User{ID: *kcUser.ID, KcId: *kcUser.ID, Username: *kcUser.Username}, nil
	}

	log.Printf("User '%s' synced and created in local DB.", kcUserID)
	return &newlyCreatedUser, nil
}

// EnableUserInKeycloak sets the enabled status of a Keycloak user to true.
func (r *UserRepository) EnableUserInKeycloak(ctx context.Context, kcUserID string) error {
	adminToken, err := r.keycloakConfig.LoginAdmin(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin token to enable user: %w", err)
	}

	// Get the user first to ensure we have the latest representation
	user, err := r.keycloakClient.GetUserByID(ctx, adminToken.AccessToken, r.keycloakConfig.Realm, kcUserID)
	if err != nil {
		return fmt.Errorf("failed to get user %s from keycloak before enabling: %w", kcUserID, err)
	}
	if user == nil {
		return fmt.Errorf("user %s not found in keycloak", kcUserID)
	}

	// Set Enabled to true
	user.Enabled = gocloak.BoolP(true)

	// Update the user in Keycloak
	err = r.keycloakClient.UpdateUser(ctx, adminToken.AccessToken, r.keycloakConfig.Realm, *user)
	if err != nil {
		return fmt.Errorf("failed to enable user %s in keycloak: %w", kcUserID, err)
	}

	log.Printf("User '%s' enabled in Keycloak.", kcUserID)
	return nil
}
