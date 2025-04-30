package service

import (
	"context"
	"errors" // Import errors package for custom errors
	"fmt"
	"log"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm"
	// Remove duplicate imports below
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrRoleNotFound       = errors.New("workspace role not found")
	ErrWorkspaceNotFound  = errors.New("workspace not found") // Assuming this might be needed
	ErrRoleNotInWorkspace = errors.New("role does not belong to the specified workspace")
)

type WorkspaceRoleService struct {
	roleRepo        *repository.WorkspaceRoleRepository
	userRepo        *repository.UserRepository      // Add UserRepository dependency
	workspaceRepo   *repository.WorkspaceRepository // Add WorkspaceRepository dependency
	keycloakService KeycloakService                 // Change to interface type
	db              *gorm.DB                        // Add DB field
}

// Modify NewWorkspaceRoleService to initialize repositories internally
func NewWorkspaceRoleService(db *gorm.DB) *WorkspaceRoleService {
	// Initialize repositories internally
	roleRepo := repository.NewWorkspaceRoleRepository(db)
	userRepo := repository.NewUserRepository(db)
	// userRepo := repository.NewUserRepository(db) // Remove duplicate declaration
	workspaceRepo := repository.NewWorkspaceRepository(db)
	keycloakService := NewKeycloakService() // Initialize KeycloakService (returns interface)
	return &WorkspaceRoleService{
		db:              db, // Store db
		roleRepo:        roleRepo,
		userRepo:        userRepo,
		workspaceRepo:   workspaceRepo,
		keycloakService: keycloakService, // Store interface value
	}
}

func (s *WorkspaceRoleService) List() ([]model.WorkspaceRole, error) {
	return s.roleRepo.List()
}

func (s *WorkspaceRoleService) GetByID(id uint) (*model.WorkspaceRole, error) {
	role, err := s.roleRepo.GetByID(id)
	if err != nil {
		// Assuming repo returns gorm.ErrRecordNotFound or similar
		return nil, ErrRoleNotFound // Return custom error
	}
	return role, nil
}

func (s *WorkspaceRoleService) Create(role *model.WorkspaceRole) error {
	// WorkspaceID is removed from WorkspaceRole model.
	// No need to validate workspace existence here.
	// Add validation for role name uniqueness if needed (DB constraint should handle it).
	return s.roleRepo.Create(role)
}

func (s *WorkspaceRoleService) Update(role *model.WorkspaceRole) error {
	// Validate if the role exists first
	_, err := s.roleRepo.GetByID(role.ID)
	if err != nil {
		return ErrRoleNotFound
	}
	// WorkspaceID is removed from WorkspaceRole model.
	// No need to validate workspace changes here.

	// Ensure Name is not being updated to an existing name (handled by DB unique constraint)
	return s.roleRepo.Update(role) // Update should handle partial updates based on fields provided
}

func (s *WorkspaceRoleService) Delete(id uint) error {
	// Check if role exists before delete
	_, err := s.roleRepo.GetByID(id)
	if err != nil {
		return ErrRoleNotFound
	}
	return s.roleRepo.Delete(id)
}

// AssignRoleToUser 사용자에게 워크스페이스 역할 할당
func (s *WorkspaceRoleService) AssignRoleToUser(userID, roleID, workspaceID uint) error {
	// 1. Check if user exists using the local DB ID
	_, err := s.userRepo.FindByID(userID) // Use FindByID
	if err != nil {
		// Handle user not found error (assuming FindByDbID returns a specific error)
		return ErrUserNotFound // Define this error
	}

	// 2. Check if role exists
	_, err = s.roleRepo.GetByID(roleID) // Use '=' to assign to existing err variable
	if err != nil {
		return ErrRoleNotFound
	}

	// 3. WorkspaceID check is removed as WorkspaceRole is now independent.
	//    The association itself links user, workspace, and role via UserWorkspaceRole table.

	// 4. Assign the role in DB first
	if err := s.roleRepo.AssignRoleToUser(userID, roleID, workspaceID); err != nil {
		// Handle potential DB errors (e.g., duplicate entry if not handled by repo)
		return err
	}

	// 5. Sync with Keycloak Group
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		// Log warning but don't fail the operation as DB part succeeded
		log.Printf("Warning: Could not find user %d for Keycloak group assignment: %v", userID, err)
		return nil
	}
	role, err := s.roleRepo.GetByID(roleID)
	if err != nil {
		// Log warning
		log.Printf("Warning: Could not find role %d for Keycloak group assignment: %v", roleID, err)
		return nil
	}

	// Construct group name (e.g., ws_1_admin)
	groupName := fmt.Sprintf("ws_%d_%s", workspaceID, role.Name)

	// Call Keycloak service to ensure group exists and assign user
	// Assuming KeycloakService has a method like this (implementation needed)
	// It should handle finding group by name, creating if not exists, and adding user by kc_id
	if err := s.keycloakService.EnsureGroupExistsAndAssignUser(context.Background(), user.KcId, groupName); err != nil { // Corrected user.KcId
		// Log warning if Keycloak sync fails
		log.Printf("Warning: Failed to assign user %s to Keycloak group %s: %v", user.Username, groupName, err)
	} else {
		log.Printf("Successfully assigned user %s to Keycloak group %s", user.Username, groupName)
	}

	return nil // DB assignment was successful
}

// RemoveRoleFromUser 사용자에게서 워크스페이스 역할 제거
func (s *WorkspaceRoleService) RemoveRoleFromUser(userID, roleID, workspaceID uint) error {
	// 1. Check if user exists (optional, depends on desired behavior)
	// _, err := s.userRepo.FindByID(userID)
	// if err != nil { return ErrUserNotFound }

	// 2. Check if role exists (optional, repo delete might handle non-existence)
	var err error                       // Declare err variable for this scope
	_, err = s.roleRepo.GetByID(roleID) // Use '=' to assign to existing err variable
	if err != nil {
		return ErrRoleNotFound // Role must exist to be removed
	}

	// 3. WorkspaceID check is removed. We just need to ensure the role exists before attempting removal.

	// 4. Sync with Keycloak Group (Remove user from group first)
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		log.Printf("Warning: Could not find user %d for Keycloak group removal: %v", userID, err)
		// Continue to DB removal attempt
	}
	role, err := s.roleRepo.GetByID(roleID)
	if err != nil {
		log.Printf("Warning: Could not find role %d for Keycloak group removal: %v", roleID, err)
		// Continue to DB removal attempt
	}

	if user != nil && role != nil {
		groupName := fmt.Sprintf("ws_%d_%s", workspaceID, role.Name)
		// Call Keycloak service to remove user from group
		// Assuming KeycloakService has a method like this (implementation needed)
		if err := s.keycloakService.RemoveUserFromGroup(context.Background(), user.KcId, groupName); err != nil { // Corrected user.KcId
			// Log warning if Keycloak sync fails
			log.Printf("Warning: Failed to remove user %s from Keycloak group %s: %v", user.Username, groupName, err)
		} else {
			log.Printf("Successfully removed user %s from Keycloak group %s", user.Username, groupName)
		}
	}

	// 5. Remove the role assignment from DB
	return s.roleRepo.RemoveRoleFromUser(userID, roleID, workspaceID)
}
