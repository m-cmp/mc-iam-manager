package repository

import (
	"errors" // Re-add errors package

	"github.com/m-cmp/mc-iam-manager/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause" // Re-add clause package
)

// PermissionRepository 권한 관리 리포지토리
type PermissionRepository struct {
	// db *gorm.DB // db instance is now passed into methods
}

// NewPermissionRepository 권한 관리 리포지토리 생성
func NewPermissionRepository() *PermissionRepository { // Removed db parameter
	return &PermissionRepository{}
}

// Create 권한 생성
func (r *PermissionRepository) Create(tx *gorm.DB, permission *model.Permission) error {
	return tx.Create(permission).Error
}

// GetByID ID로 권한 조회
func (r *PermissionRepository) GetByID(tx *gorm.DB, id string) (*model.Permission, error) { // Changed id type to string
	var permission model.Permission
	// Use Where for string primary key
	err := tx.Where("id = ?", id).First(&permission).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Consider returning a specific error like ErrPermissionNotFound
			return nil, errors.New("permission not found")
		}
		return nil, err
	}
	return &permission, nil
}

// List 권한 목록 조회
func (r *PermissionRepository) List(tx *gorm.DB) ([]model.Permission, error) {
	var permissions []model.Permission
	err := tx.Find(&permissions).Error
	return permissions, err
}

// Update 권한 수정
func (r *PermissionRepository) Update(tx *gorm.DB, permission *model.Permission) error {
	return tx.Save(permission).Error
}

// Delete 권한 삭제
func (r *PermissionRepository) Delete(tx *gorm.DB, id string) error { // Changed id type to string
	// Use Where for string primary key
	result := tx.Where("id = ?", id).Delete(&model.Permission{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// Consider returning a specific error like ErrPermissionNotFound
		return errors.New("permission not found or already deleted")
	}
	return nil
}

// AssignRolePermission 역할에 권한 할당
func (r *PermissionRepository) AssignRolePermission(tx *gorm.DB, roleType string, roleID uint, permissionID string) error { // Added roleType, changed permissionID type
	rolePermission := model.RolePermission{
		RoleType:     roleType, // Added
		RoleID:       roleID,
		PermissionID: permissionID, // Changed type
	}
	// Use Clauses(clause.OnConflict{DoNothing: true}) to ignore if already exists
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rolePermission).Error
}

// RemoveRolePermission 역할에서 권한 제거
func (r *PermissionRepository) RemoveRolePermission(tx *gorm.DB, roleType string, roleID uint, permissionID string) error { // Added roleType, changed permissionID type
	result := tx.Where("role_type = ? AND role_id = ? AND permission_id = ?", roleType, roleID, permissionID). // Added roleType
															Delete(&model.RolePermission{})
	if result.Error != nil {
		return result.Error
	}
	// Don't return error if mapping didn't exist
	// if result.RowsAffected == 0 {
	// 	 return errors.New("role permission mapping not found")
	// }
	return nil
}

// GetRolePermissions 역할의 권한 목록 조회
func (r *PermissionRepository) GetRolePermissions(tx *gorm.DB, roleType string, roleID uint) ([]model.Permission, error) { // Added roleType
	var permissions []model.Permission
	// Use TableName() from model for joins
	err := tx.Joins("JOIN mcmp_role_permissions rp ON rp.permission_id = mcmp_permissions.id"). // Use explicit table names
													Where("rp.role_type = ? AND rp.role_id = ?", roleType, roleID). // Added roleType
													Find(&permissions).Error
	return permissions, err
}

// TODO: Add methods for Workspace Roles if needed, e.g., GetWorkspaceRolePermissions
