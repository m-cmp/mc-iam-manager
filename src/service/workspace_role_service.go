package service

import (
	"errors" // Import errors package for custom errors

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
	roleRepo      *repository.WorkspaceRoleRepository
	userRepo      *repository.UserRepository      // Add UserRepository dependency
	workspaceRepo *repository.WorkspaceRepository // Add WorkspaceRepository dependency
	db            *gorm.DB                        // Add DB field
}

// Modify NewWorkspaceRoleService to initialize repositories internally
func NewWorkspaceRoleService(db *gorm.DB) *WorkspaceRoleService { // Accept only db
	// Initialize repositories internally
	roleRepo := repository.NewWorkspaceRoleRepository(db)
	userRepo := repository.NewUserRepository(db)
	workspaceRepo := repository.NewWorkspaceRepository(db)
	return &WorkspaceRoleService{
		db:            db, // Store db
		roleRepo:      roleRepo,
		userRepo:      userRepo,
		workspaceRepo: workspaceRepo,
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

	// 4. Assign the role (create mapping in UserWorkspaceRole table)
	return s.roleRepo.AssignRoleToUser(userID, roleID, workspaceID) // Pass workspaceID
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

	// 4. Remove the role assignment (delete mapping from UserWorkspaceRole table)
	return s.roleRepo.RemoveRoleFromUser(userID, roleID, workspaceID) // Pass workspaceID
}
