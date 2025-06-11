package repository

import (
	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

// WorkspaceRoleRepository 워크스페이스 역할 레포지토리
type WorkspaceRoleRepository struct {
	db *gorm.DB
}

// NewWorkspaceRoleRepository 새 WorkspaceRoleRepository 인스턴스 생성
func NewWorkspaceRoleRepository(db *gorm.DB) *WorkspaceRoleRepository {
	return &WorkspaceRoleRepository{db: db}
}

// List 모든 워크스페이스 역할 목록 조회
func (r *WorkspaceRoleRepository) List() ([]model.RoleMaster, error) {
	var roles []model.RoleMaster
	if err := r.db.Preload("RoleSubs").
		Joins("JOIN mcmp_role_sub ON mcmp_role_master.id = mcmp_role_sub.role_id").
		Where("mcmp_role_sub.role_type = ?", model.RoleTypeWorkspace).
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// GetByID ID로 워크스페이스 역할 조회
func (r *WorkspaceRoleRepository) GetByID(id uint) (*model.RoleMaster, error) {
	var role model.RoleMaster
	if err := r.db.Preload("RoleSubs").
		Joins("JOIN mcmp_role_sub ON mcmp_role_master.id = mcmp_role_sub.role_id").
		Where("mcmp_role_master.id = ? AND mcmp_role_sub.role_type = ?", id, model.RoleTypeWorkspace).
		First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// Create 새 워크스페이스 역할 생성
func (r *WorkspaceRoleRepository) Create(role *model.RoleMaster) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(role).Error; err != nil {
			return err
		}
		roleSub := model.RoleSub{
			RoleID:   role.ID,
			RoleType: model.RoleTypeWorkspace,
		}
		return tx.Create(&roleSub).Error
	})
}

// Update 워크스페이스 역할 정보 수정
func (r *WorkspaceRoleRepository) Update(role *model.RoleMaster) error {
	return r.db.Save(role).Error
}

// Delete 워크스페이스 역할 삭제
func (r *WorkspaceRoleRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ? AND role_type = ?", id, model.RoleTypeWorkspace).Delete(&model.RoleSub{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.RoleMaster{}, id).Error
	})
}

// AssignRole 사용자에게 워크스페이스 역할 할당
func (r *WorkspaceRoleRepository) AssignRole(userID, workspaceID, roleID uint) error {
	userWorkspaceRole := model.UserWorkspaceRole{
		UserID:      userID,
		WorkspaceID: workspaceID,
		RoleID:      roleID,
	}
	return r.db.Create(&userWorkspaceRole).Error
}

// RemoveRole 사용자의 워크스페이스 역할 제거
func (r *WorkspaceRoleRepository) RemoveRole(userID, workspaceID, roleID uint) error {
	return r.db.Where("user_id = ? AND workspace_id = ? AND role_id = ?", userID, workspaceID, roleID).
		Delete(&model.UserWorkspaceRole{}).Error
}

// GetUserRoles 사용자의 워크스페이스 역할 목록 조회
func (r *WorkspaceRoleRepository) GetUserRoles(userID, workspaceID uint) ([]model.RoleMaster, error) {
	var roles []model.RoleMaster
	if err := r.db.Preload("RoleSubs").
		Joins("JOIN mcmp_user_workspace_roles ON mcmp_role_master.id = mcmp_user_workspace_roles.role_id").
		Joins("JOIN mcmp_role_sub ON mcmp_role_master.id = mcmp_role_sub.role_id").
		Where("mcmp_user_workspace_roles.user_id = ? AND mcmp_user_workspace_roles.workspace_id = ? AND mcmp_role_sub.role_type = ?",
			userID, workspaceID, model.RoleTypeWorkspace).
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// GetWorkspaceRoles 워크스페이스의 모든 역할 목록 조회
func (r *WorkspaceRoleRepository) GetWorkspaceRoles(workspaceID uint) ([]*model.RoleMaster, error) {
	var roles []*model.RoleMaster
	if err := r.db.Preload("RoleSubs").
		Joins("JOIN mcmp_role_sub ON mcmp_role_master.id = mcmp_role_sub.role_id").
		Where("mcmp_role_sub.role_type = ?", model.RoleTypeWorkspace).
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// DB returns the underlying gorm.DB instance
func (r *WorkspaceRoleRepository) DB() *gorm.DB {
	return r.db
}

// ExistsWorkspaceRoleByID ID로 워크스페이스 역할 존재 여부 확인
func (r *WorkspaceRoleRepository) ExistsWorkspaceRoleByID(id uint) (bool, error) {
	var count int64
	if err := r.db.Model(&model.RoleMaster{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
