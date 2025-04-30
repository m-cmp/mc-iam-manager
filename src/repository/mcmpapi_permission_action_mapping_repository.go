package repository

import (
	"context"
	"errors"

	"github.com/m-cmp/mc-iam-manager/model/mcmpapi"
	"gorm.io/gorm"
)

// McmpApiPermissionActionMappingRepository handles database operations for permission-action mappings.
type McmpApiPermissionActionMappingRepository struct {
	db *gorm.DB
}

// NewMcmpApiPermissionActionMappingRepository creates a new repository instance.
func NewMcmpApiPermissionActionMappingRepository(db *gorm.DB) *McmpApiPermissionActionMappingRepository {
	return &McmpApiPermissionActionMappingRepository{db: db}
}

// GetActionsByPermissionID returns all API actions mapped to a specific permission.
func (r *McmpApiPermissionActionMappingRepository) GetActionsByPermissionID(ctx context.Context, permissionID string) ([]mcmpapi.McmpApiAction, error) {
	var actions []mcmpapi.McmpApiAction
	err := r.db.WithContext(ctx).
		Joins("JOIN mcmp_mciam_permission_action_mappings m ON m.action_id = mcmp_api_actions.id").
		Where("m.permission_id = ?", permissionID).
		Find(&actions).Error
	if err != nil {
		return nil, err
	}
	return actions, nil
}

// GetPermissionsByActionID returns all permissions mapped to a specific API action.
func (r *McmpApiPermissionActionMappingRepository) GetPermissionsByActionID(ctx context.Context, actionID uint) ([]string, error) {
	var permissionIDs []string
	err := r.db.WithContext(ctx).
		Model(&mcmpapi.McmpApiPermissionActionMapping{}).
		Where("action_id = ?", actionID).
		Pluck("permission_id", &permissionIDs).Error
	if err != nil {
		return nil, err
	}
	return permissionIDs, nil
}

// CreateMapping creates a new permission-action mapping.
func (r *McmpApiPermissionActionMappingRepository) CreateMapping(ctx context.Context, permissionID string, actionID uint) error {
	mapping := &mcmpapi.McmpApiPermissionActionMapping{
		PermissionID: permissionID,
		ActionID:     actionID,
	}
	err := r.db.WithContext(ctx).Create(mapping).Error
	if err != nil {
		return err
	}
	return nil
}

// DeleteMapping deletes a permission-action mapping.
func (r *McmpApiPermissionActionMappingRepository) DeleteMapping(ctx context.Context, permissionID string, actionID uint) error {
	result := r.db.WithContext(ctx).
		Where("permission_id = ? AND action_id = ?", permissionID, actionID).
		Delete(&mcmpapi.McmpApiPermissionActionMapping{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("mapping not found")
	}
	return nil
}

// CheckMappingExists checks if a permission-action mapping exists.
func (r *McmpApiPermissionActionMappingRepository) CheckMappingExists(ctx context.Context, permissionID string, actionID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&mcmpapi.McmpApiPermissionActionMapping{}).
		Where("permission_id = ? AND action_id = ?", permissionID, actionID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
