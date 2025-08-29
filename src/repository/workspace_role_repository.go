package repository

import (
	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

// WorkspaceRoleRepository workspace role repository
type WorkspaceRoleRepository struct {
	db *gorm.DB
}

// NewWorkspaceRoleRepository create new WorkspaceRoleRepository instance
func NewWorkspaceRoleRepository(db *gorm.DB) *WorkspaceRoleRepository {
	return &WorkspaceRoleRepository{db: db}
}

// List retrieve all workspace role list
func (r *WorkspaceRoleRepository) List() ([]model.RoleMaster, error) {
	var roles []model.RoleMaster
	if err := r.db.Preload("RoleSubs").
		Joins("JOIN mcmp_role_subs ON mcmp_role_masters.id = mcmp_role_subs.role_id").
		Where("mcmp_role_subs.role_type = ?", constants.RoleTypeWorkspace).
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// GetByID retrieve workspace role by ID
func (r *WorkspaceRoleRepository) GetByID(id uint) (*model.RoleMaster, error) {
	var role model.RoleMaster
	if err := r.db.Preload("RoleSubs").
		Joins("JOIN mcmp_role_subs ON mcmp_role_masters.id = mcmp_role_subs.role_id").
		Where("mcmp_role_masters.id = ? AND mcmp_role_subs.role_type = ?", id, constants.RoleTypeWorkspace).
		First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// Create create new workspace role
func (r *WorkspaceRoleRepository) Create(role *model.RoleMaster) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(role).Error; err != nil {
			return err
		}
		roleSub := model.RoleSub{
			RoleID:   role.ID,
			RoleType: constants.RoleTypeWorkspace,
		}
		return tx.Create(&roleSub).Error
	})
}

// Update modify workspace role information
func (r *WorkspaceRoleRepository) Update(role *model.RoleMaster) error {
	return r.db.Save(role).Error
}

// Delete delete workspace role
func (r *WorkspaceRoleRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ? AND role_type = ?", id, constants.RoleTypeWorkspace).Delete(&model.RoleSub{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.RoleMaster{}, id).Error
	})
}

// AssignRole assign workspace role to user
func (r *WorkspaceRoleRepository) AssignRole(userID, workspaceID, roleID uint) error {
	userWorkspaceRole := model.UserWorkspaceRole{
		UserID:      userID,
		WorkspaceID: workspaceID,
		RoleID:      roleID,
	}
	return r.db.Create(&userWorkspaceRole).Error
}

// RemoveRole remove user's workspace role
func (r *WorkspaceRoleRepository) RemoveRole(userID, workspaceID, roleID uint) error {
	return r.db.Where("user_id = ? AND workspace_id = ? AND role_id = ?", userID, workspaceID, roleID).
		Delete(&model.UserWorkspaceRole{}).Error
}

// GetUserRoles retrieve user's workspace role list
func (r *WorkspaceRoleRepository) GetUserRoles(userID, workspaceID uint) ([]model.RoleMaster, error) {
	var roles []model.RoleMaster
	if err := r.db.Preload("RoleSubs").
		Joins("JOIN mcmp_user_workspace_roles ON mcmp_role_masters.id = mcmp_user_workspace_roles.role_id").
		Joins("JOIN mcmp_role_subs ON mcmp_role_masters.id = mcmp_role_subs.role_id").
		Where("mcmp_user_workspace_roles.user_id = ? AND mcmp_user_workspace_roles.workspace_id = ? AND mcmp_role_subs.role_type = ?",
			userID, workspaceID, constants.RoleTypeWorkspace).
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// GetWorkspaceRoles retrieve all role list of workspace
func (r *WorkspaceRoleRepository) GetWorkspaceRoles(workspaceID uint) ([]*model.RoleMaster, error) {
	var roles []*model.RoleMaster
	if err := r.db.Preload("RoleSubs").
		Joins("JOIN mcmp_role_subs ON mcmp_role_masters.id = mcmp_role_subs.role_id").
		Where("mcmp_role_subs.role_type = ?", constants.RoleTypeWorkspace).
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// DB returns the underlying gorm.DB instance
func (r *WorkspaceRoleRepository) DB() *gorm.DB {
	return r.db
}

// ExistsWorkspaceRoleByID check if workspace role exists by ID
func (r *WorkspaceRoleRepository) ExistsWorkspaceRoleByID(id uint) (bool, error) {
	var count int64
	if err := r.db.Model(&model.RoleMaster{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
