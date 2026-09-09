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

// UpdateStatus updates the status field of a user in the local DB.
func (r *UserRepository) UpdateStatus(id uint, status model.UserStatus) error {
	result := r.db.Model(&model.User{}).Where("id = ?", id).Update("status", string(status))
	if result.Error != nil {
		return fmt.Errorf("failed to update user status (id: %d): %w", id, result.Error)
	}
	return nil
}

// DeleteAllRoleMappings removes all platform roles, workspace roles, and organization mappings for a user.
func (r *UserRepository) DeleteAllRoleMappings(userID uint) error {
	if err := r.db.Where("user_id = ?", userID).Delete(&model.UserPlatformRole{}).Error; err != nil {
		return fmt.Errorf("failed to delete platform roles for user %d: %w", userID, err)
	}
	if err := r.db.Where("user_id = ?", userID).Delete(&model.UserWorkspaceRole{}).Error; err != nil {
		return fmt.Errorf("failed to delete workspace roles for user %d: %w", userID, err)
	}
	if err := r.db.Where("user_id = ?", userID).Delete(&model.UserOrganization{}).Error; err != nil {
		return fmt.Errorf("failed to delete organization mappings for user %d: %w", userID, err)
	}
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
func (r *UserRepository) FindUserRoleInWorkspace(userID, workspaceID uint) (*model.UserWorkspaceRole, error) {
	var userWorkspaceRole model.UserWorkspaceRole
	err := r.db.Where("user_id = ? AND workspace_id = ?", userID, workspaceID).
		Preload("Workspace").
		Preload("Role").
		First(&userWorkspaceRole).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user %d has no role assigned in workspace %d", userID, workspaceID)
		}
		return nil, fmt.Errorf("error finding user role in workspace: %w", err)
	}
	return &userWorkspaceRole, nil
}

// FindEffectiveUserRoleInWorkspace 사용자의 워크스페이스 유효 역할 단건 조회 (직접 우선, 없으면 그룹 상속).
// 직접 배정이 있으면 그것을 쓰고, 없으면 소속 그룹의 워크스페이스 역할을 본다(P1-3/OPEN-016: 자동 부여 +
// Union+dedup 직접 우선 결정). 여러 그룹이 같은 워크스페이스에 서로 다른 역할을 부여하는 경우(다중 매핑
// 선택 규칙은 P4-3 미결)는 role_id 오름차순 첫 값으로 결정적으로 선택한다. 아무 역할도 없으면 (nil, nil).
func (r *UserRepository) FindEffectiveUserRoleInWorkspace(userID, workspaceID uint) (*model.EffectiveWorkspaceRole, error) {
	var direct model.EffectiveWorkspaceRole
	err := r.db.Raw(`
		SELECT uwr.workspace_id, w.name AS workspace_name, uwr.role_id, rm.name AS role_name
		FROM mcmp_user_workspace_roles uwr
		JOIN mcmp_workspaces w ON w.id = uwr.workspace_id
		JOIN mcmp_role_masters rm ON rm.id = uwr.role_id
		WHERE uwr.user_id = ? AND uwr.workspace_id = ?
	`, userID, workspaceID).Scan(&direct).Error
	if err != nil {
		return nil, fmt.Errorf("error finding direct workspace role for user %d in workspace %d: %w", userID, workspaceID, err)
	}
	if direct.RoleID != 0 {
		return &direct, nil
	}

	var groupRoles []model.EffectiveWorkspaceRole
	err = r.db.Raw(`
		SELECT DISTINCT gwr.workspace_id, w.name AS workspace_name, gwr.role_id, rm.name AS role_name
		FROM mcmp_group_workspace_roles gwr
		JOIN mcmp_user_organizations uo ON uo.organization_id = gwr.group_id
		JOIN mcmp_workspaces w ON w.id = gwr.workspace_id
		JOIN mcmp_role_masters rm ON rm.id = gwr.role_id
		WHERE uo.user_id = ? AND gwr.workspace_id = ?
		ORDER BY gwr.role_id ASC
	`, userID, workspaceID).Scan(&groupRoles).Error
	if err != nil {
		return nil, fmt.Errorf("error finding group-inherited workspace role for user %d in workspace %d: %w", userID, workspaceID, err)
	}
	if len(groupRoles) == 0 {
		return nil, nil
	}
	return &groupRoles[0], nil
}

// CreateUserWorkspaceRole 사용자를 워크스페이스에 추가
func (r *UserRepository) CreateUserWorkspaceRole(userWorkspaceRole *model.UserWorkspaceRole) error {
	return r.db.Create(userWorkspaceRole).Error
}

// DeleteUserWorkspaceRole 워크스페이스에서 사용자 제거
func (r *UserRepository) DeleteUserWorkspaceRole(workspaceID, userID uint) error {
	result := r.db.Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Delete(&model.UserWorkspaceRole{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete user from workspace: %w", result.Error)
	}
	return nil
}
