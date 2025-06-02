package repository

import (
	"errors"
	"log"

	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

var (
	ErrResourceTypeNotFound      = errors.New("resource type not found")
	ErrResourceTypeAlreadyExists = errors.New("resource type with this framework_id and id already exists")
)

// ResourceTypeRepository 리소스 유형 데이터 관리
type ResourceTypeRepository struct {
	db *gorm.DB
}

// NewResourceTypeRepository 새 ResourceTypeRepository 인스턴스 생성
func NewResourceTypeRepository(db *gorm.DB) *ResourceTypeRepository {
	return &ResourceTypeRepository{db: db}
}

// Create 리소스 유형 생성
func (r *ResourceTypeRepository) CreateResourceType(resourceType *model.ResourceType) error {
	return r.db.Create(resourceType).Error
}

// List 모든 리소스 유형 조회 (프레임워크 ID로 필터링 가능)
func (r *ResourceTypeRepository) FindResourceTypes(frameworkID string) ([]model.ResourceType, error) {
	var resourceTypes []model.ResourceType
	query := r.db
	if frameworkID != "" {
		query = query.Where("framework_id = ?", frameworkID)
	}
	if err := query.Find(&resourceTypes).Error; err != nil {
		return nil, err
	}

	// SQL 쿼리 로깅
	sql := query.Statement.SQL.String()
	args := query.Statement.Vars
	log.Printf("List SQL Query: %s", sql)
	log.Printf("List SQL Args: %v", args)
	log.Printf("List Result Count: %d", len(resourceTypes))

	return resourceTypes, nil
}

// GetByID ID로 리소스 유형 조회 (FrameworkID와 ID 사용)
func (r *ResourceTypeRepository) FindResourceTypeByID(frameworkID, id string) (*model.ResourceType, error) {
	var resourceType model.ResourceType
	if err := r.db.Where("framework_id = ? AND id = ?", frameworkID, id).First(&resourceType).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrResourceTypeNotFound
		}
		return nil, err
	}

	// SQL 쿼리 로깅
	sql := r.db.Statement.SQL.String()
	args := r.db.Statement.Vars
	log.Printf("GetByID SQL Query: %s", sql)
	log.Printf("GetByID SQL Args: %v", args)

	return &resourceType, nil
}

// Update 리소스 유형 정보 부분 업데이트
func (r *ResourceTypeRepository) UpdateResourceType(frameworkID, id string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return errors.New("no fields provided for update")
	}
	// Prevent updating primary keys
	delete(updates, "id")
	delete(updates, "framework_id")
	delete(updates, "createdAt") // Prevent updating createdAt

	result := r.db.Model(&model.ResourceType{}).Where("framework_id = ? AND id = ?", frameworkID, id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrResourceTypeNotFound // Or check if record exists before update
	}

	// SQL 쿼리 로깅
	sql := result.Statement.SQL.String()
	args := result.Statement.Vars
	log.Printf("Update SQL Query: %s", sql)
	log.Printf("Update SQL Args: %v", args)
	log.Printf("Update Affected Rows: %d", result.RowsAffected)

	return nil
}

// Delete 리소스 유형 삭제
func (r *ResourceTypeRepository) DeleteResourceType(frameworkID, id string) error {
	result := r.db.Where("framework_id = ? AND id = ?", frameworkID, id).Delete(&model.ResourceType{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrResourceTypeNotFound
	}
	// Note: Associated permissions will be deleted due to CASCADE constraint in DB schema

	// SQL 쿼리 로깅
	sql := result.Statement.SQL.String()
	args := result.Statement.Vars
	log.Printf("Delete SQL Query: %s", sql)
	log.Printf("Delete SQL Args: %v", args)
	log.Printf("Delete Affected Rows: %d", result.RowsAffected)

	return nil
}
