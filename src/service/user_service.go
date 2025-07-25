package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	// Add strings import for error checking
	// "github.com/Nerzal/gocloak/v13" // No longer needed directly
	// "github.com/m-cmp/mc-iam-manager/config" // No longer needed directly
	"github.com/Nerzal/gocloak/v13"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm"
)

// Use repository's error
// var (
// 	ErrUserNotFound = errors.New("user not found")
// )

type UserService struct {
	db                *gorm.DB
	userRepo          *repository.UserRepository
	roleRepo          *repository.RoleRepository
	workspaceRoleRepo *repository.WorkspaceRoleRepository
	workspaceRepo     *repository.WorkspaceRepository
	tokenRepo         *repository.TokenRepository
	// keycloakService   KeycloakService // Removed dependency
}

// NewUserService constructor initializes repositories internally
func NewUserService(
	db *gorm.DB, // Add db parameter
	// keycloakService KeycloakService, // Removed KeycloakService parameter
) *UserService {
	// Initialize repositories internally
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	workspaceRepo := repository.NewWorkspaceRepository(db)
	workspaceRoleRepo := repository.NewWorkspaceRoleRepository(db) // Initialize needed repo
	tokenRepo := repository.NewTokenRepository(db)                 // Initialize needed repo

	return &UserService{
		db:                db,
		userRepo:          userRepo,
		roleRepo:          roleRepo,
		workspaceRepo:     workspaceRepo,
		workspaceRoleRepo: workspaceRoleRepo, // Store initialized repo
		tokenRepo:         tokenRepo,         // Store initialized repo
		// keycloakService:   keycloakService, // Removed KeycloakService field
	}
}

// --- Helper methods for Keycloak interaction --- // REMOVED

// SyncUser ensures a user record exists in the local DB for the given Keycloak ID.
func (s *UserService) SyncUser(ctx context.Context, kcUserID string) (*model.User, error) {
	dbUser, err := s.userRepo.FindByKcID(kcUserID)
	if err == nil && dbUser != nil {
		// User found, enrich with Keycloak data before returning
		ks := NewKeycloakService() // Create KeycloakService instance when needed
		kcUser, kcErr := ks.GetUser(ctx, kcUserID)
		if kcErr != nil {
			log.Printf("Warning: Found user in DB but failed to get Keycloak details for %s: %v", kcUserID, kcErr)
		} else if kcUser != nil {
			dbUser.Email = *kcUser.Email
			dbUser.FirstName = *kcUser.FirstName
			dbUser.LastName = *kcUser.LastName
			dbUser.Enabled = *kcUser.Enabled
			if dbUser.Username != *kcUser.Username {
				log.Printf("Warning: Username mismatch for user KcId %s (DB: %s, KC: %s). Updating DB.", kcUserID, dbUser.Username, *kcUser.Username)
				dbUser.Username = *kcUser.Username
				updateErr := s.userRepo.Update(dbUser)
				if updateErr != nil {
					log.Printf("Warning: Failed to update username in DB for KcId %s: %v", kcUserID, updateErr)
				}
			}
		}
		return dbUser, nil
	}
	// Handle DB errors other than "not found" (which is nil, nil from FindByKcID)
	if err != nil {
		return nil, fmt.Errorf("error checking user in local db (kc_id: %s): %w", kcUserID, err)
	}

	// Not found in DB, fetch from Keycloak and create
	log.Printf("User '%s' not found in local DB, syncing from Keycloak...", kcUserID)
	ks := NewKeycloakService() // Create KeycloakService instance when needed
	kcUser, err := ks.GetUser(ctx, kcUserID)
	if err != nil {
		// Handle specific "not found" error from keycloakService
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, fmt.Errorf("user %s not found in keycloak during sync: %w", kcUserID, err)
		}
		return nil, fmt.Errorf("failed to get user details from keycloak for sync (id: %s): %w", kcUserID, err)
	}
	if kcUser == nil {
		return nil, fmt.Errorf("user %s not found in keycloak detail fetch", kcUserID)
	}

	newUser := &model.User{
		KcId:        *kcUser.ID,
		Username:    *kcUser.Username,
		Description: "", // Default description
	}
	createdDbUser, createErr := s.userRepo.Create(newUser)
	if createErr != nil {
		return nil, fmt.Errorf("failed to create user in local db during sync (kc_id: %s): %w", kcUserID, createErr)
	}

	log.Printf("User '%s' synced and created in local DB.", kcUserID)
	// Merge transient Keycloak info
	createdDbUser.Email = *kcUser.Email
	createdDbUser.FirstName = *kcUser.FirstName
	createdDbUser.LastName = *kcUser.LastName
	createdDbUser.Enabled = *kcUser.Enabled
	return createdDbUser, nil
}

func (s *UserService) SetupInitialAdmin(ctx context.Context, user *model.User, adminToken *gocloak.JWT) (string, error) {
	ks := NewKeycloakService() // Create KeycloakService instance when needed
	kcId, err := ks.SetupInitialKeycloakAdmin(ctx, adminToken)
	if err != nil {
		return kcId, err // Propagate error (e.g., user exists)
	}

	user.KcId = kcId
	_, err = s.userRepo.Create(user)
	if err != nil {
		log.Printf("CRITICAL: Failed to create user in DB after Keycloak creation (kcId: %s). Manual cleanup needed. Error: %v", kcId, err)
		// TODO: Compensation - delete user from Keycloak?
		return kcId, fmt.Errorf("failed to create user in DB after Keycloak: %w", err)
	}
	return kcId, nil
}

// CreateUser creates a user in Keycloak and the local DB.
func (s *UserService) CreateUser(ctx context.Context, user *model.User) error {
	ks := NewKeycloakService() // Create KeycloakService instance when needed
	kcId, err := ks.CreateUser(ctx, user)
	if err != nil {
		return err // Propagate error (e.g., user exists)
	}
	user.KcId = kcId
	_, err = s.userRepo.Create(user)
	if err != nil {
		log.Printf("CRITICAL: Failed to create user in DB after Keycloak creation (kcId: %s). Manual cleanup needed. Error: %v", kcId, err)
		// TODO: Compensation - delete user from Keycloak?
		return fmt.Errorf("failed to create user in DB after Keycloak: %w", err)
	}
	return nil
}

// CreateUser creates a user in Keycloak and the local DB.
// Keycloak 에 있는 유저가 DB에 등록되어 있지 않은 경우
func (s *UserService) SyncUserByKeycloak(ctx context.Context, user *model.User) error {
	if user.KcId == "" {
		return fmt.Errorf("user kcId is empty")
	}
	dbUser, err := s.userRepo.FindByKcID(user.KcId)
	if dbUser != nil {
		return fmt.Errorf("user already exists in DB")
	}

	_, err = s.userRepo.Create(user)
	if err != nil {
		log.Printf("CRITICAL: Failed to create user in DB after Keycloak creation (kcId: %s). Manual cleanup needed. Error: %v", user.KcId, err)
		// TODO: Compensation - delete user from Keycloak?
		return fmt.Errorf("failed to create user in DB after Keycloak: %w", err)
	}
	return nil
}

// UpdateUser updates a user in Keycloak and the local DB.
func (s *UserService) UpdateUser(ctx context.Context, user *model.User) error {
	if user.ID == 0 {
		return errors.New("user ID must be provided for update")
	}
	dbUser, err := s.userRepo.FindUserByID(user.ID)
	if err != nil {
		return err
	}
	if dbUser.KcId == "" {
		return fmt.Errorf("cannot update user in keycloak: KcId missing for DB user ID %d", user.ID)
	}
	user.KcId = dbUser.KcId

	ks := NewKeycloakService() // Create KeycloakService instance when needed
	err = ks.UpdateUser(ctx, user)
	if err != nil {
		return err // Propagate error
	}
	err = s.userRepo.Update(user)
	if err != nil {
		log.Printf("Warning: Keycloak user updated, but DB update failed for ID %d: %v", user.ID, err)
	}
	return nil
}

// --- Public Service Methods ---

// ListUsers retrieves all users, merging data from Keycloak and the local DB.
func (s *UserService) ListUsers(ctx context.Context) ([]model.User, error) {
	ks := NewKeycloakService() // Create KeycloakService instance when needed
	kcUsers, err := ks.GetUsers(ctx)
	if err != nil {
		return nil, err
	}
	if len(kcUsers) == 0 {
		return []model.User{}, nil
	}

	kcIDs := make([]string, 0, len(kcUsers))
	keycloakUserMap := make(map[string]*gocloak.User, len(kcUsers))
	for _, u := range kcUsers {
		if u != nil && u.ID != nil {
			kcIDs = append(kcIDs, *u.ID)
			keycloakUserMap[*u.ID] = u
		}
	}
	if len(kcIDs) == 0 {
		return []model.User{}, nil
	}

	users, err := s.userRepo.GetUsersByKcIDs(kcIDs)
	if err != nil {
		log.Printf("Warning: Failed to get DB user details for some users: %v. Returning potentially incomplete data.", err)
		var result []model.User
		for _, kcUser := range kcUsers {
			if kcUser != nil && kcUser.ID != nil {
				result = append(result, model.User{
					KcId:      *kcUser.ID,
					Username:  *kcUser.Username,
					Email:     *kcUser.Email,
					FirstName: *kcUser.FirstName,
					LastName:  *kcUser.LastName,
					Enabled:   *kcUser.Enabled,
				})
			}
		}
		return result, nil
	}

	userMap := make(map[string]*model.User, len(users))
	for i := range users {
		userMap[users[i].KcId] = &users[i]
	}

	var result []model.User
	for _, kcUser := range kcUsers {
		if kcUser == nil || kcUser.ID == nil {
			continue
		}
		kcID := *kcUser.ID

		mergedUser := model.User{
			KcId:      kcID,
			Username:  *kcUser.Username,
			Email:     *kcUser.Email,
			FirstName: *kcUser.FirstName,
			LastName:  *kcUser.LastName,
			Enabled:   *kcUser.Enabled,
		}

		if dbUser, dbExists := userMap[kcID]; dbExists {
			mergedUser.ID = dbUser.ID
			mergedUser.Description = dbUser.Description
			mergedUser.CreatedAt = dbUser.CreatedAt
			mergedUser.UpdatedAt = dbUser.UpdatedAt
			mergedUser.PlatformRoles = dbUser.PlatformRoles
			mergedUser.WorkspaceRoles = dbUser.WorkspaceRoles
		} else {
			fmt.Printf("Warning: User found in Keycloak but not in local db (kc_id: %s)\n", kcID)
		}
		result = append(result, mergedUser)
	}

	return result, nil
}

// GetUserByID retrieves user details by DB ID.
func (s *UserService) GetUserByID(ctx context.Context, id uint) (*model.User, error) {
	dbUser, err := s.userRepo.FindUserByID(id)
	if err != nil {
		return nil, err
	}
	if dbUser.KcId == "" {
		log.Printf("Warning: User with DB ID %d has empty kc_id.", id)
		return dbUser, nil
	}
	ks := NewKeycloakService() // Create KeycloakService instance when needed
	kcUser, err := ks.GetUser(ctx, dbUser.KcId)
	if err != nil {
		log.Printf("Warning: failed to get Keycloak details for user id %d (kcId: %s): %v. Returning DB data only.", id, dbUser.KcId, err)
		// If user not found in Keycloak, maybe return DB data but log inconsistency?
		return dbUser, nil
	}
	dbUser.Email = *kcUser.Email
	dbUser.FirstName = *kcUser.FirstName
	dbUser.LastName = *kcUser.LastName
	dbUser.Enabled = *kcUser.Enabled
	if dbUser.Username != *kcUser.Username {
		log.Printf("Warning: Username mismatch for user ID %d (DB: %s, KC: %s)", id, dbUser.Username, *kcUser.Username)
	}
	return dbUser, nil
}

// GetUserByKcID retrieves user details by Keycloak ID.
func (s *UserService) GetUserByKcID(ctx context.Context, kcId string) (*model.User, error) {
	ks := NewKeycloakService() // Create KeycloakService instance when needed
	kcUser, err := ks.GetUser(ctx, kcId)
	if err != nil {
		return nil, err // Propagate error (e.g., user not found)
	}
	resultUser := &model.User{
		KcId:      *kcUser.ID,
		Username:  *kcUser.Username,
		Email:     *kcUser.Email,
		FirstName: *kcUser.FirstName,
		LastName:  *kcUser.LastName,
		Enabled:   *kcUser.Enabled,
	}
	dbUser, err := s.userRepo.FindByKcID(kcId)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) && err != nil {
		log.Printf("Error fetching user details from local db (kc_id: %s): %v\n", resultUser.KcId, err)
		return resultUser, nil
	}
	if dbUser != nil {
		resultUser.ID = dbUser.ID
		resultUser.Description = dbUser.Description
		resultUser.CreatedAt = dbUser.CreatedAt
		resultUser.UpdatedAt = dbUser.UpdatedAt
		resultUser.PlatformRoles = dbUser.PlatformRoles
		resultUser.WorkspaceRoles = dbUser.WorkspaceRoles
	} else {
		fmt.Printf("Warning: User found in Keycloak but not in local db (kc_id: %s)\n", resultUser.KcId)
	}
	return resultUser, nil
}

// GetUserByUsername retrieves user details by username.
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {

	dbUser, err := s.userRepo.FindByUsername(username)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) && err != nil {
		log.Printf("Error fetching user details from local db (username: %s): %v\n", username, err)
		return nil, err
	}

	return dbUser, nil
}

func (s *UserService) GetUserByKeycloakUsername(ctx context.Context, username string) (*model.User, error) {
	ks := NewKeycloakService() // Create KeycloakService instance when needed
	kcUser, err := ks.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err // Propagate error (e.g., user not found)
	}
	resultUser := &model.User{
		KcId:      *kcUser.ID,
		Username:  *kcUser.Username,
		Email:     *kcUser.Email,
		FirstName: *kcUser.FirstName,
		LastName:  *kcUser.LastName,
		Enabled:   *kcUser.Enabled,
	}
	dbUser, err := s.userRepo.FindByKcID(resultUser.KcId)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) && err != nil {
		log.Printf("Error fetching user details from local db (kc_id: %s): %v\n", resultUser.KcId, err)
		return resultUser, nil
	}
	if dbUser != nil {
		resultUser.ID = dbUser.ID
		resultUser.Description = dbUser.Description
		resultUser.CreatedAt = dbUser.CreatedAt
		resultUser.UpdatedAt = dbUser.UpdatedAt
		resultUser.PlatformRoles = dbUser.PlatformRoles
		resultUser.WorkspaceRoles = dbUser.WorkspaceRoles
	} else {
		fmt.Printf("Warning: User found in Keycloak but not in local db (kc_id: %s)\n", resultUser.KcId)
	}
	return resultUser, nil
}

// // FindWorkspacesByUserID 사용자가 속한 워크스페이스 목록 조회. workspacerepository 에서 처리
// func (s *UserService) FindWorkspacesByUserID(userID uint) ([]*model.Workspace, error) {
// 	return s.userRepo.FindWorkspacesByUserID(userID)
// }

// // GetUserRolesInWorkspace 특정 워크스페이스에서 유저가 가진 롤만 반환. workspacerepository 에서 처리
// func (s *UserService) GetUserRolesInWorkspace(userID, workspaceID uint) ([]*model.UserWorkspaceRole, error) {
// 	userWorkspaceRoles, err := s.userRepo.FindUserRolesInWorkspace(userID, workspaceID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return userWorkspaceRoles, nil
// }

// DeleteUser deletes a user from Keycloak and the local DB using the DB ID.
func (s *UserService) DeleteUser(ctx context.Context, id uint) error {
	dbUser, err := s.userRepo.FindUserByID(id)
	if err != nil {
		return err
	}
	kcId := dbUser.KcId

	if kcId != "" {
		ks := NewKeycloakService() // Create KeycloakService instance when needed
		err = ks.DeleteUser(ctx, kcId)
		if err != nil {
			// Log warning but continue with DB deletion attempt
			log.Printf("Warning: Failed to delete user %s from Keycloak: %v. Proceeding with DB deletion.", kcId, err)
		}
	} else {
		log.Printf("Warning: User with DB ID %d has no KcId. Skipping Keycloak deletion.", id)
	}

	err = s.userRepo.Delete(id)
	if err != nil {
		log.Printf("CRITICAL: Failed to delete user from DB (ID: %d) after Keycloak deletion attempt. Manual cleanup needed. Error: %v", id, err)
		return fmt.Errorf("failed to delete user from DB: %w", err)
	}
	return nil
}

// ApproveUser enables a user in Keycloak and ensures they exist in the local DB.
func (s *UserService) ApproveUser(ctx context.Context, kcUserID string) error {
	ks := NewKeycloakService() // Create KeycloakService instance when needed
	err := ks.EnableUser(ctx, kcUserID)
	if err != nil {
		return fmt.Errorf("failed to enable user in keycloak: %w", err)
	}
	_, err = s.SyncUser(ctx, kcUserID)
	if err != nil {
		fmt.Printf("Warning: User %s enabled in Keycloak, but failed to sync/create in local DB: %v\n", kcUserID, err)
	}
	return nil
}

// GetUserIDByKcID finds the local database ID for a given Keycloak User ID.
func (s *UserService) GetUserIDByKcID(ctx context.Context, kcUserID string) (uint, error) { // Uncomment function
	dbUser, err := s.userRepo.FindByKcID(kcUserID) // Use correct repo method name
	if err == nil && dbUser != nil {
		return dbUser.ID, nil // User found in DB
	}
	// Check if the error is specifically gorm.ErrRecordNotFound or our nil,nil case
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, fmt.Errorf("error checking user db_id by kc_id %s: %w", kcUserID, err)
	}
	// If err is nil but dbUser is nil (FindByKcID returns nil,nil for not found)
	if err == nil && dbUser == nil {
		// Proceed to sync
	} else if !errors.Is(err, gorm.ErrRecordNotFound) { // Handle other errors
		return 0, fmt.Errorf("error checking user db_id by kc_id %s: %w", kcUserID, err)
	}

	// User not found in DB, sync from Keycloak
	syncedUser, syncErr := s.SyncUser(ctx, kcUserID) // Call SyncUser method
	if syncErr != nil {
		return 0, fmt.Errorf("failed to sync user to get db_id (kc_id: %s): %w", kcUserID, syncErr)
	}
	if syncedUser == nil || syncedUser.ID == 0 {
		return 0, fmt.Errorf("failed to retrieve db_id after syncing user (kc_id: %s)", kcUserID)
	}

	return syncedUser.ID, nil
}

// getValidToken (Keep as is, assuming tokenRepo is initialized if needed)
// func (s *UserService) getValidToken(ctx context.Context) (string, error) {
// 	// ... (Implementation) ...
// }
