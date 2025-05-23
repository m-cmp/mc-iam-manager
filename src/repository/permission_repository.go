package repository

import (
	"errors"
	"fmt"
	"log"

	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

var (
	ErrPermissionNotFound      = errors.New("permission not found")
	ErrPermissionAlreadyExists = errors.New("permission with this id already exists")
)

// MciamPermissionRepository MC-IAM 권한 데이터 관리 - Renamed
type MciamPermissionRepository struct {
	db *gorm.DB
}

// NewMciamPermissionRepository 새 MciamPermissionRepository 인스턴스 생성 - Renamed
func NewMciamPermissionRepository(db *gorm.DB) *MciamPermissionRepository {
	return &MciamPermissionRepository{db: db}
}

// Create MC-IAM 권한 생성 - Renamed
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

// List MC-IAM 권한 목록 조회 - Renamed
func (r *MciamPermissionRepository) List(frameworkID, resourceTypeID string) ([]model.MciamPermission, error) {
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

// GetByID ID로 MC-IAM 권한 조회 - Renamed
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

// Update MC-IAM 권한 정보 부분 업데이트 - Renamed
func (r *MciamPermissionRepository) Update(id string, updates map[string]interface{}) error {
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

// Delete MC-IAM 권한 삭제 - Renamed
func (r *MciamPermissionRepository) Delete(id string) error {
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

// AssignMciamPermissionToRole 역할에 MC-IAM 권한 할당 - Renamed
func (r *MciamPermissionRepository) AssignMciamPermissionToRole(roleType string, roleID uint, permissionID string) error {
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

// RemoveMciamPermissionFromRole 역할에서 MC-IAM 권한 제거 - Renamed
func (r *MciamPermissionRepository) RemoveMciamPermissionFromRole(roleType string, roleID uint, permissionID string) error {
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

// GetPlatformRolePermissions 플랫폼 역할이 가진 모든 MC-IAM 권한 ID 조회
func (r *MciamPermissionRepository) GetPlatformRolePermissions(platformRole string) ([]string, error) {
	log.Printf("GetPlatformRolePermissions platformRole: %s", platformRole)

	// 먼저 platform role의 ID를 조회
	var platformRoleID uint
	err := r.db.Model(&model.RoleMaster{}).
		Joins("JOIN mcmp_role_sub ON mcmp_role_master.id = mcmp_role_sub.role_id").
		Where("mcmp_role_master.name = ? AND mcmp_role_sub.role_type = ?", platformRole, "platform").
		Pluck("mcmp_role_master.id", &platformRoleID).Error
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

// GetRoleMciamPermissions 워크스페이스 역할이 가진 모든 MC-IAM 권한 ID 조회
func (r *MciamPermissionRepository) GetRoleMciamPermissions(roleType string, workspaceRoleID uint) ([]string, error) {
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

// CheckRoleMciamPermission 역할이 특정 MC-IAM 권한을 가지고 있는지 확인 - Renamed
func (r *MciamPermissionRepository) CheckRoleMciamPermission(roleType string, roleID uint, permissionID string) (bool, error) {
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
