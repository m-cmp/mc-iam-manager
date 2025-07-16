package repository

import (
	"errors"
	"fmt"

	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

// PlatformRoleRepository platform role repository
type PlatformRoleRepository struct {
	db *gorm.DB
}

// NewPlatformRoleRepository create new PlatformRoleRepository instance
func NewPlatformRoleRepository(db *gorm.DB) *PlatformRoleRepository {
	return &PlatformRoleRepository{db: db}
}

// List retrieve all platform role list
func (r *PlatformRoleRepository) List() ([]model.RoleMaster, error) {
	var roles []model.RoleMaster
	if err := r.db.Preload("RoleSubs").
		Joins("JOIN mcmp_role_subs ON mcmp_role_masters.id = mcmp_role_subs.role_id").
		Where("mcmp_role_subs.role_type = ?", constants.RoleTypePlatform).
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// GetByID retrieve platform role by ID
func (r *PlatformRoleRepository) GetByID(id uint) (*model.RoleMaster, error) {
	var role model.RoleMaster
	if err := r.db.Preload("RoleSubs").
		Joins("JOIN mcmp_role_subs ON mcmp_role_masters.id = mcmp_role_subs.role_id").
		Where("mcmp_role_masters.id = ? AND mcmp_role_subs.role_type = ?", id, constants.RoleTypePlatform).
		First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// Create create new platform role
func (r *PlatformRoleRepository) Create(role *model.RoleMaster) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(role).Error; err != nil {
			return err
		}
		roleSub := model.RoleSub{
			RoleID:   role.ID,
			RoleType: constants.RoleTypePlatform,
		}
		return tx.Create(&roleSub).Error
	})
}

// Update modify platform role information
func (r *PlatformRoleRepository) Update(role *model.RoleMaster) error {
	return r.db.Save(role).Error
}

// Delete delete platform role
func (r *PlatformRoleRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ? AND role_type = ?", id, constants.RoleTypePlatform).Delete(&model.RoleSub{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.RoleMaster{}, id).Error
	})
}

// roleModelFactories role type-specific user-role mapping model creation factory map
// Provides functions to create corresponding mapping models for each role type (platform, workspace)
var roleModelFactories = map[constants.IAMRoleType]func(userID, roleID uint) interface{}{
	// Factory function for platform role type
	// Create and return UserPlatformRole model
	constants.RoleTypePlatform: func(userID, roleID uint) interface{} {
		return &model.UserPlatformRole{
			UserID: userID,
			RoleID: roleID,
		}
	},
	// Factory function for workspace role type
	// Create and return UserWorkspaceRole model
	constants.RoleTypeWorkspace: func(userID, roleID uint) interface{} {
		return &model.UserWorkspaceRole{
			UserID: userID,
			RoleID: roleID,
		}
	},
}

// AssignRole assign role to user
func (r *PlatformRoleRepository) AssignRole(userID, roleID uint, roleType constants.IAMRoleType) error {
	factory, ok := roleModelFactories[roleType]
	if !ok {
		return fmt.Errorf("unsupported role type: %s", roleType)
	}
	role := factory(userID, roleID)
	return r.db.Create(role).Error
}

// RemoveRole remove user's platform role
func (r *PlatformRoleRepository) RemoveRole(userID, roleID uint) error {
	return r.db.Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&model.UserPlatformRole{}).Error
}

// GetUserRoles retrieve user's platform role list
func (r *PlatformRoleRepository) GetUserRoles(userID uint) ([]model.RoleMaster, error) {
	var roles []model.RoleMaster
	if err := r.db.Preload("RoleSubs").
		Joins("JOIN mcmp_user_platform_roles ON mcmp_role_masters.id = mcmp_user_platform_roles.role_id").
		Joins("JOIN mcmp_role_subs ON mcmp_role_masters.id = mcmp_role_subs.role_id").
		Where("mcmp_user_platform_roles.user_id = ? AND mcmp_role_subs.role_type = ?", userID, constants.RoleTypePlatform).
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// GetByName retrieve platform role by name
func (r *PlatformRoleRepository) GetByName(name string) (*model.RoleMaster, error) {
	var role model.RoleMaster
	if err := r.db.Preload("RoleSubs").
		Joins("JOIN mcmp_role_subs ON mcmp_role_masters.id = mcmp_role_subs.role_id").
		Where("mcmp_role_masters.name = ? AND mcmp_role_subs.role_type = ?", name, constants.RoleTypePlatform).
		First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("platform role not found")
		}
		return nil, err
	}
	return &role, nil
}

// DB returns the underlying gorm DB instance (Helper for sync function)
func (r *PlatformRoleRepository) DB() *gorm.DB {
	return r.db
}
