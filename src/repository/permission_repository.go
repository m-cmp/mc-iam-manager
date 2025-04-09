package repository

import (
	"context"

	"github.com/m-cmp/mc-iam-manager/model"

	"gorm.io/gorm"
)

// PermissionRepository 권한 관리 리포지토리
type PermissionRepository struct {
	db *gorm.DB
}

// NewPermissionRepository 권한 관리 리포지토리 생성
func NewPermissionRepository(db *gorm.DB) *PermissionRepository {
	return &PermissionRepository{db: db}
}

// Create 권한 생성
func (r *PermissionRepository) Create(ctx context.Context, permission *model.Permission) error {
	return r.db.WithContext(ctx).Create(permission).Error
}

// GetByID ID로 권한 조회
func (r *PermissionRepository) GetByID(ctx context.Context, id uint) (*model.Permission, error) {
	var permission model.Permission
	err := r.db.WithContext(ctx).First(&permission, id).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

// List 권한 목록 조회
func (r *PermissionRepository) List(ctx context.Context) ([]model.Permission, error) {
	var permissions []model.Permission
	err := r.db.WithContext(ctx).Find(&permissions).Error
	return permissions, err
}

// Update 권한 수정
func (r *PermissionRepository) Update(ctx context.Context, permission *model.Permission) error {
	return r.db.WithContext(ctx).Save(permission).Error
}

// Delete 권한 삭제
func (r *PermissionRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Permission{}, id).Error
}

// AssignRolePermission 역할에 권한 할당
func (r *PermissionRepository) AssignRolePermission(ctx context.Context, roleID, permissionID uint) error {
	rolePermission := model.RolePermission{
		RoleID:       roleID,
		PermissionID: permissionID,
	}
	return r.db.WithContext(ctx).Create(&rolePermission).Error
}

// RemoveRolePermission 역할에서 권한 제거
func (r *PermissionRepository) RemoveRolePermission(ctx context.Context, roleID, permissionID uint) error {
	return r.db.WithContext(ctx).
		Where("role_id = ? AND permission_id = ?", roleID, permissionID).
		Delete(&model.RolePermission{}).Error
}

// GetRolePermissions 역할의 권한 목록 조회
func (r *PermissionRepository) GetRolePermissions(ctx context.Context, roleID uint) ([]model.Permission, error) {
	var permissions []model.Permission
	err := r.db.WithContext(ctx).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID).
		Find(&permissions).Error
	return permissions, err
}
