package service

import (
	"errors" // Import errors package for custom errors
	"fmt"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm"
	// Remove duplicate imports below
)

var (
	ErrUserNotFound                = errors.New("user not found")
	ErrWorkspaceRoleNotFound       = errors.New("workspace role not found")
	ErrWorkspaceNotFound           = errors.New("workspace not found")
	ErrWorkspaceRoleNotInWorkspace = errors.New("workspace role does not belong to the specified workspace")
)

type WorkspaceRoleService struct {
	roleRepo      *repository.WorkspaceRoleRepository
	userRepo      *repository.UserRepository
	workspaceRepo *repository.WorkspaceRepository
	db            *gorm.DB
}

// NewWorkspaceRoleService 새로운 WorkspaceRoleService 인스턴스 생성
func NewWorkspaceRoleService(db *gorm.DB) *WorkspaceRoleService {
	return &WorkspaceRoleService{
		db:            db,
		roleRepo:      repository.NewWorkspaceRoleRepository(db),
		userRepo:      repository.NewUserRepository(db),
		workspaceRepo: repository.NewWorkspaceRepository(db),
	}
}

func (s *WorkspaceRoleService) List() ([]model.WorkspaceRole, error) {
	return s.roleRepo.List()
}

func (s *WorkspaceRoleService) GetByID(id uint) (*model.WorkspaceRole, error) {
	role, err := s.roleRepo.GetByID(id)
	if err != nil {
		// Assuming repo returns gorm.ErrRecordNotFound or similar
		return nil, ErrWorkspaceRoleNotFound // Return custom error
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
		return ErrWorkspaceRoleNotFound
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
		return ErrWorkspaceRoleNotFound
	}
	return s.roleRepo.Delete(id)
}

// AssignWorkspaceRoleToUser 사용자에게 워크스페이스 역할 할당
func (s *WorkspaceRoleService) AssignWorkspaceRoleToUser(userID, workspaceRoleID, workspaceID uint) error {
	// 1. Check if user exists using the local DB ID
	_, err := s.userRepo.FindByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	// 2. Check if workspace role exists
	_, err = s.roleRepo.GetByID(workspaceRoleID)
	if err != nil {
		return ErrWorkspaceRoleNotFound
	}

	// 3. Assign the workspace role in DB
	return s.roleRepo.AssignRoleToUser(userID, workspaceRoleID, workspaceID)
}

// RemoveWorkspaceRoleFromUser 사용자에게서 워크스페이스 역할 제거
func (s *WorkspaceRoleService) RemoveWorkspaceRoleFromUser(userID, workspaceRoleID, workspaceID uint) error {
	// 1. Check if workspace role exists
	_, err := s.roleRepo.GetByID(workspaceRoleID)
	if err != nil {
		return ErrWorkspaceRoleNotFound
	}

	// 2. Remove the workspace role assignment from DB
	return s.roleRepo.RemoveRoleFromUser(userID, workspaceRoleID, workspaceID)
}

// GetUserWorkspaceRoles 사용자의 워크스페이스 역할 목록을 조회합니다.
func (s *WorkspaceRoleService) GetUserWorkspaceRoles(userID uint, workspaceID uint) ([]string, error) {
	// 사용자의 워크스페이스 역할 조회
	userWorkspaceRoles, err := s.userRepo.GetUserRolesInWorkspace(userID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user workspace roles: %w", err)
	}

	// 워크스페이스 역할 이름 목록 추출
	workspaceRoles := make([]string, 0, len(userWorkspaceRoles))
	for _, uwr := range userWorkspaceRoles {
		if uwr.WorkspaceRole.Name != "" {
			workspaceRoles = append(workspaceRoles, uwr.WorkspaceRole.Name)
		}
	}

	return workspaceRoles, nil
}

// GetUserByID 사용자 ID로 사용자 정보를 조회합니다.
func (s *WorkspaceRoleService) GetUserByID(userID uint) (*model.User, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// GetWorkspaceRoleByName 워크스페이스 역할 이름으로 역할을 조회합니다.
func (s *WorkspaceRoleService) GetWorkspaceRoleByName(name string) (*model.WorkspaceRole, error) {
	var workspaceRole model.WorkspaceRole
	if err := s.db.Where("name = ?", name).First(&workspaceRole).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkspaceRoleNotFound
		}
		return nil, fmt.Errorf("워크스페이스 역할 조회 실패: %w", err)
	}
	return &workspaceRole, nil
}
