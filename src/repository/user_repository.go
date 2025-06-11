package repository

import (
	// "context" // Removed context import

	"errors"
	"fmt"
	"log"

	// "github.com/m-cmp/mc-iam-manager/config" // Removed Keycloak config dependency
	"github.com/m-cmp/mc-iam-manager/model"

	// "github.com/Nerzal/gocloak/v13" // Removed Keycloak client dependency
	"gorm.io/gorm"
)

// UserRepository handles database operations for users.
type UserRepository struct {
	db *gorm.DB
}

var (
	ErrUserNotFound = errors.New("user not found")
)

// DB returns the underlying gorm DB instance (Helper for sync function)
func (r *UserRepository) DB() *gorm.DB {
	return r.db
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *gorm.DB) *UserRepository {
	// Auto Migrate the schema
	if err := db.AutoMigrate(&model.User{}); err != nil {
		log.Printf("Failed to migrate user table: %v", err)
	}
	return &UserRepository{db: db}
}

// FindByID finds a user by their local database primary key (id column).
func (r *UserRepository) FindUserByID(id uint) (*model.User, error) {
	var dbUser model.User
	// Preload roles when fetching by ID
	if err := r.db.Preload("PlatformRoles").Preload("WorkspaceRoles").First(&dbUser, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("error finding user by id %d: %w", id, err)
	}
	return &dbUser, nil
}

// FindByKcID finds a user by their Keycloak ID (kc_id column).
// Returns nil, nil if not found.
func (r *UserRepository) FindByKcID(kcId string) (*model.User, error) {
	var dbUser model.User
	// Preload roles when fetching by KcId
	if err := r.db.Preload("PlatformRoles").Preload("WorkspaceRoles").Where("kc_id = ?", kcId).First(&dbUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil, nil for not found, service layer handles sync
		}
		return nil, fmt.Errorf("error finding user by kc_id %s: %w", kcId, err)
	}
	return &dbUser, nil
}

// FindByUsername finds a user by their username (username column). : db에서 조회
func (r *UserRepository) FindByUsername(username string) (*model.User, error) {
	var dbUser model.User

	query := r.db.Table("mcmp_users")

	// Find user by username
	if err := query.Where("username = ?", username).First(&dbUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("error finding user by username %s: %w", username, err)
	}
	log.Printf("[DEBUG] GetDbUser: %+v", &dbUser)
	return &dbUser, nil
}

// GetDbUsersByKcIDs retrieves users from the local DB based on a list of Keycloak IDs, preloading roles.
func (r *UserRepository) GetUsersByKcIDs(kcIDs []string) ([]model.User, error) {
	if len(kcIDs) == 0 {
		return []model.User{}, nil
	}
	var dbUsers []model.User
	log.Printf("[DEBUG] GetDbUsersByKcIDs: Attempting to fetch %d users from local DB by KcIDs...\n", len(kcIDs))
	if errDb := r.db.Preload("PlatformRoles").Preload("WorkspaceRoles").Where("kc_id IN ?", kcIDs).Find(&dbUsers).Error; errDb != nil {
		log.Printf("[ERROR] GetDbUsersByKcIDs: Error fetching user details from local db: %v\n", errDb)
		return nil, fmt.Errorf("error fetching users from db by kc_id list: %w", errDb)
	}
	return dbUsers, nil
}

// CreateDbUser creates a new user record in the local database.
func (r *UserRepository) Create(user *model.User) (*model.User, error) {
	// Ensure ID is not set, let DB generate it
	user.ID = 0
	// Use map to explicitly specify columns, especially if model has fields not in DB
	userDataToCreate := map[string]interface{}{
		"kc_id":       user.KcId,
		"username":    user.Username,
		"description": user.Description,
	}
	log.Printf("[DEBUG] Attempting to create user data in CreateDbUser (map): %+v", userDataToCreate)
	// Create using the map
	result := r.db.Model(&model.User{}).Create(userDataToCreate)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to save user to local db: %w", result.Error)
	}
	// Fetch the newly created record to get the generated ID
	var createdUser model.User
	// Use the unique kc_id to fetch the record reliably
	if err := r.db.Where("kc_id = ?", user.KcId).First(&createdUser).Error; err != nil {
		log.Printf("Warning: Failed to fetch newly created user %s from local DB after creation: %v", user.KcId, err)
		// Return the input user but ID might be 0
		return user, nil
	}
	return &createdUser, nil
}

// UpdateDbUser updates an existing user record in the local database using the DB ID.
func (r *UserRepository) Update(user *model.User) error {
	if user.ID == 0 {
		return errors.New("cannot update user without DB ID")
	}
	// Update only specific fields (e.g., description, username if allowed)
	updateData := map[string]interface{}{
		"description": user.Description,
		"username":    user.Username, // Assuming username can be updated in DB
		// Add other updatable DB fields here
	}
	result := r.db.Model(&model.User{}).Where("id = ?", user.ID).Updates(updateData)
	if result.Error != nil {
		return fmt.Errorf("failed to update user in local db (id: %d): %w", user.ID, result.Error)
	}
	if result.RowsAffected == 0 {
		log.Printf("Warning: User with DB ID %d not found during DB update.", user.ID)
		return ErrUserNotFound
	}
	return nil
}

// DeleteDbUserByID deletes a user record from the local database using the DB ID.
func (r *UserRepository) Delete(id uint) error {
	result := r.db.Delete(&model.User{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete user from local db (id: %d): %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		log.Printf("Warning: User with DB ID %d not found during DB deletion.", id)
		return ErrUserNotFound
	}
	log.Printf("Successfully deleted user from local DB (id: %d)", id)
	return nil
}

// FindWorkspaceAndWorkspaceRolesByUserID finds all workspace roles assigned to a user.
// It expects the user's local database ID (id column).
func (r *UserRepository) FindWorkspaceAndWorkspaceRolesByUserID(userID uint) ([]*model.UserWorkspaceRole, error) {
	var userWorkspaceRoles []*model.UserWorkspaceRole
	err := r.db.Where("user_id = ?", userID).
		Preload("User").
		Preload("Workspace").
		Preload("Role").
		Find(&userWorkspaceRoles).Error
	if err != nil {
		return nil, fmt.Errorf("error finding workspace roles for user %d: %w", userID, err)
	}
	return userWorkspaceRoles, nil
}

// FindWorkspacesByUserID finds all workspaces a user is assigned to (has any role in).
func (r *UserRepository) FindWorkspacesByUserID(userID uint) ([]*model.WorkspaceWithUsersAndRoles, error) {
	var workspaces []*model.WorkspaceWithUsersAndRoles
	// Select distinct workspaces associated with the user through the join table
	err := r.db.Joins("JOIN mcmp_user_workspace_roles uwr ON uwr.workspace_id = mcmp_workspaces.id").
		Where("uwr.user_id = ?", userID).
		Distinct("mcmp_workspaces.*").           // Select distinct workspace fields
		Preload("Users", "user_id = ?", userID). // Preload users for the specific user
		Preload("Users.User").                   // Preload user details
		Preload("Users.Role").                   // Preload role details
		Find(&workspaces).Error
	if err != nil {
		return nil, fmt.Errorf("error finding workspaces for user %d: %w", userID, err)
	}
	return workspaces, nil
}

// GetUserRolesInWorkspace finds all roles assigned to a user within a specific workspace.
func (r *UserRepository) FindUserRolesInWorkspace(userID, workspaceID uint) ([]*model.UserWorkspaceRole, error) {
	var userWorkspaceRoles []*model.UserWorkspaceRole
	err := r.db.Where("user_id = ? AND workspace_id = ?", userID, workspaceID).
		Preload("Workspace").
		Preload("Role").
		Find(&userWorkspaceRoles).Error
	if err != nil {
		return nil, err
	}
	return userWorkspaceRoles, nil
}
