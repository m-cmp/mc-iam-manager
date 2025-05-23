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

// WorkspaceRoleService 워크스페이스 역할 서비스
type WorkspaceRoleService struct {
	repo *repository.WorkspaceRoleRepository
}

// NewWorkspaceRoleService 새 WorkspaceRoleService 인스턴스 생성
func NewWorkspaceRoleService(repo *repository.WorkspaceRoleRepository) *WorkspaceRoleService {
	return &WorkspaceRoleService{repo: repo}
}

// List 모든 워크스페이스 역할 목록 조회
func (s *WorkspaceRoleService) List() ([]model.RoleMaster, error) {
	return s.repo.List()
}

// GetByID ID로 워크스페이스 역할 조회
func (s *WorkspaceRoleService) GetByID(id uint) (*model.RoleMaster, error) {
	return s.repo.GetByID(id)
}

// Create 새 워크스페이스 역할 생성
func (s *WorkspaceRoleService) Create(role *model.RoleMaster) error {
	return s.repo.Create(role)
}

// Update 워크스페이스 역할 정보 수정
func (s *WorkspaceRoleService) Update(role *model.RoleMaster) error {
	return s.repo.Update(role)
}

// Delete 워크스페이스 역할 삭제
func (s *WorkspaceRoleService) Delete(id uint) error {
	return s.repo.Delete(id)
}

// AssignRole 사용자에게 워크스페이스 역할 할당
func (s *WorkspaceRoleService) AssignRole(userID, workspaceID, roleID uint) error {
	return s.repo.AssignRole(userID, workspaceID, roleID)
}

// RemoveRole 사용자의 워크스페이스 역할 제거
func (s *WorkspaceRoleService) RemoveRole(userID, workspaceID, roleID uint) error {
	return s.repo.RemoveRole(userID, workspaceID, roleID)
}

// GetUserRoles 사용자의 워크스페이스 역할 목록 조회
func (s *WorkspaceRoleService) GetUserRoles(userID, workspaceID uint) ([]model.RoleMaster, error) {
	return s.repo.GetUserRoles(userID, workspaceID)
}

// GetWorkspaceRoles 워크스페이스의 모든 역할 목록 조회
func (s *WorkspaceRoleService) GetWorkspaceRoles(workspaceID uint) ([]model.RoleMaster, error) {
	return s.repo.GetWorkspaceRoles(workspaceID)
}

// GetUserWorkspaceRoles 사용자의 워크스페이스 역할 목록을 조회합니다.
func (s *WorkspaceRoleService) GetUserWorkspaceRoles(userID uint, workspaceID uint) ([]string, error) {
	// 사용자의 워크스페이스 역할 조회
	userWorkspaceRoles, err := s.repo.GetUserRoles(userID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user workspace roles: %w", err)
	}

	// 워크스페이스 역할 이름 목록 추출
	workspaceRoles := make([]string, 0, len(userWorkspaceRoles))
	for _, uwr := range userWorkspaceRoles {
		if uwr.Name != "" {
			workspaceRoles = append(workspaceRoles, uwr.Name)
		}
	}

	return workspaceRoles, nil
}

// GetUserByID 사용자 ID로 사용자 정보를 조회합니다.
func (s *WorkspaceRoleService) GetUserByID(userID uint) (*model.User, error) {
	userRepo := repository.NewUserRepository(s.repo.DB())
	user, err := userRepo.FindByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// GetWorkspaceRoleByName 워크스페이스 역할 이름으로 역할을 조회합니다.
func (s *WorkspaceRoleService) GetWorkspaceRoleByName(name string) (*model.RoleMaster, error) {
	var role model.RoleMaster
	if err := s.repo.DB().Preload("RoleSubs").
		Joins("JOIN mcmp_role_sub ON mcmp_role_master.id = mcmp_role_sub.role_id").
		Where("mcmp_role_master.name = ? AND mcmp_role_sub.role_type = ?", name, "workspace").
		First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkspaceRoleNotFound
		}
		return nil, fmt.Errorf("워크스페이스 역할 조회 실패: %w", err)
	}
	return &role, nil
}
