package repository

import (
	"errors"
	"fmt"
	"log"

	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

var (
	ErrPermissionNotFound      = errors.New("permission not found")
	ErrPermissionAlreadyExists = errors.New("permission with this id already exists")
)

// MciamPermissionRepository MC-IAM permission data management - Renamed
type MciamPermissionRepository struct {
	db *gorm.DB
}

// NewMciamPermissionRepository create new MciamPermissionRepository instance - Renamed
func NewMciamPermissionRepository(db *gorm.DB) *MciamPermissionRepository {
	return &MciamPermissionRepository{db: db}
}

// Create MC-IAM permission creation - Renamed
func (r *MciamPermissionRepository) Create(permission *model.MciamPermission) error {
	// Check if already exists
	var existing model.MciamPermission
	if err := r.db.Where("id = ?", permission.ID).First(&existing).Error; err == nil {
		return ErrPermissionAlreadyExists // Keep error name for now? Or rename? Let's keep it.
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err // Other DB error
	}
	// Create if not found
	return r.db.Create(permission).Error
}

// List MC-IAM permission list retrieval - Renamed
func (r *MciamPermissionRepository) ListMcIamPermissions(frameworkID, resourceTypeID string) ([]model.MciamPermission, error) {
	var permissions []model.MciamPermission
	query := r.db
	if frameworkID != "" {
		query = query.Where("framework_id = ?", frameworkID)
	}
	if resourceTypeID != "" {
		query = query.Where("resource_type_id = ?", resourceTypeID)
	}
	if err := query.Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

// GetByID retrieve MC-IAM permission by ID - Renamed
func (r *MciamPermissionRepository) GetByID(id string) (*model.MciamPermission, error) {
	var permission model.MciamPermission
	if err := r.db.Where("id = ?", id).First(&permission).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPermissionNotFound
		}
		return nil, err
	}
	return &permission, nil
}

// Update partial MC-IAM permission information update - Renamed
func (r *MciamPermissionRepository) UpdateMcIamPermission(id string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return errors.New("no fields provided for update")
	}
	// Prevent updating primary key or immutable fields
	delete(updates, "id")
	delete(updates, "framework_id")
	delete(updates, "resource_type_id")
	delete(updates, "action")
	delete(updates, "createdAt")

	result := r.db.Model(&model.MciamPermission{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPermissionNotFound
	}
	return nil
}

// Delete MC-IAM permission deletion - Renamed
func (r *MciamPermissionRepository) DeleteMcIamPermission(id string) error {
	result := r.db.Where("id = ?", id).Delete(&model.MciamPermission{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPermissionNotFound
	}
	// Note: Associated role_permissions will be deleted due to CASCADE constraint in DB schema
	return nil
}

// --- Role MC-IAM Permission Mappings ---

// AssignMciamPermissionToRole assign MC-IAM permission to role - Renamed
func (r *MciamPermissionRepository) AssignMciamPermissionToRole(roleType constants.IAMRoleType, roleID uint, permissionID string) error {
	// Check if permission exists
	if _, err := r.GetByID(permissionID); err != nil {
		return err // Return ErrPermissionNotFound or other DB error
	}

	mapping := model.MciamRoleMciamPermission{ // Use new model name
		RoleType:     roleType,
		RoleID:       roleID,
		PermissionID: permissionID,
	}
	// Use FirstOrCreate or similar to avoid duplicate errors, or handle constraint violation error
	return r.db.Create(&mapping).Error // Simple create for now
}

// RemoveMciamPermissionFromRole remove MC-IAM permission from role - Renamed
func (r *MciamPermissionRepository) RemoveMciamPermissionFromRole(roleType constants.IAMRoleType, roleID uint, permissionID string) error {
	result := r.db.Where("role_type = ? AND role_id = ? AND permission_id = ?", roleType, roleID, permissionID).Delete(&model.MciamRoleMciamPermission{}) // Use new model name and column name
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// Consider if this should be an error (mapping not found) or just success
		return errors.New("role mciam permission mapping not found")
	}
	return nil
}

// GetPlatformRolePermissions retrieve all MC-IAM permission IDs that platform role has
func (r *MciamPermissionRepository) GetPlatformRolePermissions(platformRole string) ([]string, error) {
	log.Printf("GetPlatformRolePermissions platformRole: %s", platformRole)

	// First retrieve platform role ID
	var platformRoleID uint
	err := r.db.Model(&model.RoleMaster{}).
		Joins("JOIN mcmp_role_subs ON mcmp_role_masters.id = mcmp_role_subs.role_id").
		Where("mcmp_role_masters.name = ? AND mcmp_role_subs.role_type = ?", platformRole, constants.RoleTypePlatform).
		Pluck("mcmp_role_masters.id", &platformRoleID).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get platform role ID: %w", err)
	}

	var permissionIDs []string
	err = r.db.Model(&model.MciamRoleMciamPermission{}).
		Where("role_type = 'platform' AND role_id = ?", platformRoleID).
		Pluck("permission_id", &permissionIDs).Error
	if err != nil {
		return nil, err
	}
	return permissionIDs, nil
}

// GetRoleMciamPermissions retrieve all MC-IAM permission IDs that workspace role has
func (r *MciamPermissionRepository) GetRoleMciamPermissions(roleType constants.IAMRoleType, workspaceRoleID uint) ([]string, error) {
	log.Printf("GetRoleMciamPermissions roleType: %s RoleID: %d", roleType, workspaceRoleID)

	var permissionIDs []string
	err := r.db.Model(&model.MciamRoleMciamPermission{}).
		Where("role_type = ? AND role_id = ?", roleType, workspaceRoleID).
		Pluck("permission_id", &permissionIDs).Error
	if err != nil {
		return nil, err
	}
	return permissionIDs, nil
}

// CheckRoleMciamPermission check if role has specific MC-IAM permission - Renamed
func (r *MciamPermissionRepository) CheckRoleMciamPermission(roleType constants.IAMRoleType, roleID uint, permissionID string) (bool, error) {
	var count int64
	err := r.db.Model(&model.MciamRoleMciamPermission{}). // Use new model name
								Where("role_type = ? AND role_id = ? AND permission_id = ?", roleType, roleID, permissionID). // Use new column name
								Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Note: Need similar functions for mcmp_csp_permissions and mciam_role_csp_permissions later.
