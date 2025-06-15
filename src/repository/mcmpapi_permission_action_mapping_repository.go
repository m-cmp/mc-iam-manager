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
		return nil, fmt.Errorf("failed to get actions by permission ID: %w", err)
	}

	// SQL 쿼리 로깅 (쿼리 실행 후)
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("GetActionsByPermissionID SQL Query: %s", sql)
	log.Printf("GetActionsByPermissionID SQL Args: %v", args)
	log.Printf("GetActionsByPermissionID Result Count: %d", len(actions))

	return actions, nil
}

// GetPermissionsByActionID 액션 ID에 해당하는 권한 목록 조회
func (r *McmpApiPermissionActionMappingRepository) FindPermissionsByActionID(ctx context.Context, actionID uint) ([]mcmpapi.McmpApiPermissionActionMapping, error) {
	var permissions []mcmpapi.McmpApiPermissionActionMapping
	query := r.db.Where("action_id = ?", actionID).Find(&permissions)

	if err := query.Error; err != nil {
		return nil, fmt.Errorf("failed to get permissions by action ID: %w", err)
	}

	// SQL 쿼리 로깅 (쿼리 실행 후)
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("GetPermissionsByActionID SQL Query: %s", sql)
	log.Printf("GetPermissionsByActionID SQL Args: %v", args)
	log.Printf("GetPermissionsByActionID Result Count: %d", len(permissions))

	return permissions, nil
}

// CheckMappingExists 매핑 존재 여부 확인
func (r *McmpApiPermissionActionMappingRepository) CheckMappingExists(ctx context.Context, permissionID string, actionID uint) (bool, error) {
	var count int64
	query := r.db.Model(&mcmpapi.McmpApiPermissionActionMapping{}).
		Where("permission_id = ? AND action_id = ?", permissionID, actionID).
		Count(&count)

	if err := query.Error; err != nil {
		return false, fmt.Errorf("failed to check mapping existence: %w", err)
	}

	// SQL 쿼리 로깅 (쿼리 실행 후)
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("CheckMappingExists SQL Query: %s", sql)
	log.Printf("CheckMappingExists SQL Args: %v", args)
	log.Printf("CheckMappingExists Result Count: %d", count)

	return count > 0, nil
}

// CreateMapping 매핑 생성
func (r *McmpApiPermissionActionMappingRepository) CreateMapping(ctx context.Context, mapping *mcmpapi.McmpApiPermissionActionMapping) error {
	query := r.db.Create(mapping)

	if err := query.Error; err != nil {
		return fmt.Errorf("failed to create mapping: %w", err)
	}

	// SQL 쿼리 로깅 (쿼리 실행 후)
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("CreateMapping SQL Query: %s", sql)
	log.Printf("CreateMapping SQL Args: %v", args)
	log.Printf("CreateMapping Created ID: %d", mapping.ID)

	return nil
}

// DeleteMapping 매핑 삭제
func (r *McmpApiPermissionActionMappingRepository) DeleteMapping(ctx context.Context, permissionID string, actionID uint) error {
	query := r.db.Where("permission_id = ? AND action_id = ?", permissionID, actionID).
		Delete(&mcmpapi.McmpApiPermissionActionMapping{})

	if err := query.Error; err != nil {
		return fmt.Errorf("failed to delete mapping: %w", err)
	}

	// SQL 쿼리 로깅 (쿼리 실행 후)
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("DeleteMapping SQL Query: %s", sql)
	log.Printf("DeleteMapping SQL Args: %v", args)
	log.Printf("DeleteMapping Affected Rows: %d", query.RowsAffected)

	return nil
}

// UpdateMapping 매핑 수정
func (r *McmpApiPermissionActionMappingRepository) UpdateMapping(ctx context.Context, mapping *mcmpapi.McmpApiPermissionActionMapping) error {
	query := r.db.Model(&mcmpapi.McmpApiPermissionActionMapping{}).
		Where("permission_id = ? AND action_id = ?", mapping.PermissionID, mapping.ActionID).
		Update("action_name", mapping.ActionName)

	if err := query.Error; err != nil {
		return fmt.Errorf("failed to update mapping: %w", err)
	}

	// SQL 쿼리 로깅 (쿼리 실행 후)
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("UpdateMapping SQL Query: %s", sql)
	log.Printf("UpdateMapping SQL Args: %v", args)
	log.Printf("UpdateMapping Affected Rows: %d", query.RowsAffected)

	return nil
}
