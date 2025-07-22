package repository

import (
	"context"
	"fmt"
	"log"

	"github.com/m-cmp/mc-iam-manager/model/mcmpapi"
	"gorm.io/gorm"
)

type McmpApiPermissionActionMappingRepository struct {
	db *gorm.DB
}

func NewMcmpApiPermissionActionMappingRepository(db *gorm.DB) *McmpApiPermissionActionMappingRepository {
	return &McmpApiPermissionActionMappingRepository{
		db: db,
	}
}

// GetActionsByPermissionID 권한 ID에 해당하는 액션 목록 조회
func (r *McmpApiPermissionActionMappingRepository) FindActionsByPermissionID(ctx context.Context, permissionID string) ([]mcmpapi.McmpApiPermissionActionMapping, error) {
	var actions []mcmpapi.McmpApiPermissionActionMapping
	query := r.db.Where("permission_id = ?", permissionID).Find(&actions)

	if err := query.Error; err != nil {
		// 에러 발생 시에만 쿼리 로깅
		sql := query.Statement.SQL.String()
		args := query.Statement.Vars
		log.Printf("GetActionsByPermissionID SQL Query (ERROR): %s", sql)
		log.Printf("GetActionsByPermissionID SQL Args (ERROR): %v", args)
		return nil, fmt.Errorf("failed to get actions by permission ID: %w", err)
	}

	return actions, nil
}

// GetPermissionsByActionID 액션 ID에 해당하는 권한 목록 조회
func (r *McmpApiPermissionActionMappingRepository) FindPermissionsByActionID(ctx context.Context, actionID uint) ([]mcmpapi.McmpApiPermissionActionMapping, error) {
	var permissions []mcmpapi.McmpApiPermissionActionMapping
	query := r.db.Where("action_id = ?", actionID).Find(&permissions)

	if err := query.Error; err != nil {
		// 에러 발생 시에만 쿼리 로깅
		sql := query.Statement.SQL.String()
		args := query.Statement.Vars
		log.Printf("GetPermissionsByActionID SQL Query (ERROR): %s", sql)
		log.Printf("GetPermissionsByActionID SQL Args (ERROR): %v", args)
		return nil, fmt.Errorf("failed to get permissions by action ID: %w", err)
	}

	return permissions, nil
}

// CheckMappingExists 매핑 존재 여부 확인
func (r *McmpApiPermissionActionMappingRepository) CheckMappingExists(ctx context.Context, permissionID string, actionID uint) (bool, error) {
	var count int64
	query := r.db.Model(&mcmpapi.McmpApiPermissionActionMapping{}).
		Where("permission_id = ? AND action_id = ?", permissionID, actionID).
		Count(&count)

	if err := query.Error; err != nil {
		// 에러 발생 시에만 쿼리 로깅
		sql := query.Statement.SQL.String()
		args := query.Statement.Vars
		log.Printf("CheckMappingExists SQL Query (ERROR): %s", sql)
		log.Printf("CheckMappingExists SQL Args (ERROR): %v", args)
		return false, fmt.Errorf("failed to check mapping existence: %w", err)
	}

	return count > 0, nil
}

// CreateMapping 매핑 생성
func (r *McmpApiPermissionActionMappingRepository) CreateMapping(ctx context.Context, mapping *mcmpapi.McmpApiPermissionActionMapping) error {
	query := r.db.Create(mapping)

	if err := query.Error; err != nil {
		// 에러 발생 시에만 쿼리 로깅
		sql := query.Statement.SQL.String()
		args := query.Statement.Vars
		log.Printf("CreateMapping SQL Query (ERROR): %s", sql)
		log.Printf("CreateMapping SQL Args (ERROR): %v", args)
		return fmt.Errorf("failed to create mapping: %w", err)
	}

	return nil
}

// DeleteMapping 매핑 삭제
func (r *McmpApiPermissionActionMappingRepository) DeleteMapping(ctx context.Context, permissionID string, actionID uint) error {
	query := r.db.Where("permission_id = ? AND action_id = ?", permissionID, actionID).
		Delete(&mcmpapi.McmpApiPermissionActionMapping{})

	if err := query.Error; err != nil {
		// 에러 발생 시에만 쿼리 로깅
		sql := query.Statement.SQL.String()
		args := query.Statement.Vars
		log.Printf("DeleteMapping SQL Query (ERROR): %s", sql)
		log.Printf("DeleteMapping SQL Args (ERROR): %v", args)
		return fmt.Errorf("failed to delete mapping: %w", err)
	}

	return nil
}

// UpdateMapping 매핑 수정
func (r *McmpApiPermissionActionMappingRepository) UpdateMapping(ctx context.Context, mapping *mcmpapi.McmpApiPermissionActionMapping) error {
	query := r.db.Model(&mcmpapi.McmpApiPermissionActionMapping{}).
		Where("permission_id = ? AND action_id = ?", mapping.PermissionID, mapping.ActionID).
		Update("action_name", mapping.ActionName)

	if err := query.Error; err != nil {
		// 에러 발생 시에만 쿼리 로깅
		sql := query.Statement.SQL.String()
		args := query.Statement.Vars
		log.Printf("UpdateMapping SQL Query (ERROR): %s", sql)
		log.Printf("UpdateMapping SQL Args (ERROR): %v", args)
		return fmt.Errorf("failed to update mapping: %w", err)
	}

	return nil
}
