package repository

import (
	"errors"
	"fmt"

	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

// RoleRepository 역할 관리 레포지토리
type RoleRepository struct {
	db *gorm.DB
}

// NewRoleRepository 새 RoleRepository 인스턴스 생성
func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

// List 모든 역할 목록 조회
func (r *RoleRepository) List(roleType string) ([]model.RoleMaster, error) {
	var roles []model.RoleMaster
	query := r.db.Preload("RoleSubs")

	if roleType != "" {
		query = query.Joins("JOIN mcmp_role_sub ON mcmp_role_master.id = mcmp_role_sub.role_id").
			Where("mcmp_role_sub.role_type = ?", roleType)
	}

	if err := query.Find(&roles).Error; err != nil {
		return nil, fmt.Errorf("역할 목록 조회 실패: %w", err)
	}
	return roles, nil
}

// GetByID ID로 역할 조회
func (r *RoleRepository) GetByID(id uint) (*model.RoleMaster, error) {
	var role model.RoleMaster
	if err := r.db.Preload("RoleSubs").First(&role, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("역할 조회 실패: %w", err)
	}
	return &role, nil
}

// Create 역할 생성
func (r *RoleRepository) Create(role *model.RoleMaster) error {
	return r.db.Create(role).Error
}

// Update 역할 수정
func (r *RoleRepository) Update(role *model.RoleMaster) error {
	return r.db.Save(role).Error
}

// Delete 역할 삭제
func (r *RoleRepository) Delete(id uint) error {
	return r.db.Delete(&model.RoleMaster{}, id).Error
}

// CreateRoleSub 역할 서브 타입 생성
func (r *RoleRepository) CreateRoleSub(roleSub *model.RoleSub) error {
	return r.db.Create(roleSub).Error
}

// DeleteRoleSubs 역할 서브 타입들 삭제
func (r *RoleRepository) DeleteRoleSubs(roleID uint) error {
	return r.db.Where("role_id = ?", roleID).Delete(&model.RoleSub{}).Error
}

// AssignPlatformRole 플랫폼 역할 할당
func (r *RoleRepository) AssignPlatformRole(userID, roleID uint) error {
	userRole := model.UserPlatformRole{
		UserID: userID,
		RoleID: roleID,
	}
	return r.db.Create(&userRole).Error
}

// RemovePlatformRole 플랫폼 역할 제거
func (r *RoleRepository) RemovePlatformRole(userID, roleID uint) error {
	return r.db.Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&model.UserPlatformRole{}).Error
}

// AssignWorkspaceRole 워크스페이스 역할 할당
func (r *RoleRepository) AssignWorkspaceRole(userID, workspaceID, roleID uint) error {
	userWorkspaceRole := model.UserWorkspaceRole{
		UserID:      userID,
		WorkspaceID: workspaceID,
		RoleID:      roleID,
	}
	return r.db.Create(&userWorkspaceRole).Error
}

// RemoveWorkspaceRole 워크스페이스 역할 제거
func (r *RoleRepository) RemoveWorkspaceRole(userID, workspaceID, roleID uint) error {
	return r.db.Where("user_id = ? AND workspace_id = ? AND role_id = ?", userID, workspaceID, roleID).
		Delete(&model.UserWorkspaceRole{}).Error
}

// GetUserWorkspaceRoles 사용자의 워크스페이스 역할 목록 조회
func (r *RoleRepository) GetUserWorkspaceRoles(userID, workspaceID uint) ([]model.RoleMaster, error) {
	var roles []model.RoleMaster
	err := r.db.
		Joins("JOIN mcmp_user_workspace_roles ON mcmp_role_master.id = mcmp_user_workspace_roles.role_id").
		Joins("JOIN mcmp_role_sub ON mcmp_role_master.id = mcmp_role_sub.role_id").
		Where("mcmp_user_workspace_roles.user_id = ? AND mcmp_user_workspace_roles.workspace_id = ? AND mcmp_role_sub.role_type = ?", userID, workspaceID, "workspace").
		Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// GetUserPlatformRoles 사용자의 플랫폼 역할 목록 조회
func (r *RoleRepository) GetUserPlatformRoles(userID uint) ([]model.RoleMaster, error) {
	var roles []model.RoleMaster
	err := r.db.
		Joins("JOIN mcmp_user_platform_roles ON mcmp_role_master.id = mcmp_user_platform_roles.role_id").
		Joins("JOIN mcmp_role_sub ON mcmp_role_master.id = mcmp_role_sub.role_id").
		Where("mcmp_user_platform_roles.user_id = ? AND mcmp_role_sub.role_type = ?", userID, "platform").
		Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// FindByID 역할을 ID로 조회
func (r *RoleRepository) FindByID(id uint) (*model.RoleMaster, error) {
	var role model.RoleMaster
	if err := r.db.First(&role, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}
