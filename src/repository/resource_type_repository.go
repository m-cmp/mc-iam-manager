package repository

import (
	"errors"

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
func (r *ResourceTypeRepository) Create(resourceType *model.ResourceType) error {
	// Check if already exists
	var existing model.ResourceType
	if err := r.db.Where("framework_id = ? AND id = ?", resourceType.FrameworkID, resourceType.ID).First(&existing).Error; err == nil {
		return ErrResourceTypeAlreadyExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err // Other DB error
	}
	// Create if not found
	return r.db.Create(resourceType).Error
}

// List 모든 리소스 유형 조회 (프레임워크 ID로 필터링 가능)
func (r *ResourceTypeRepository) List(frameworkID string) ([]model.ResourceType, error) {
	var resourceTypes []model.ResourceType
	query := r.db
	if frameworkID != "" {
		query = query.Where("framework_id = ?", frameworkID)
	}
	if err := query.Find(&resourceTypes).Error; err != nil {
		return nil, err
	}
	return resourceTypes, nil
}

// GetByID ID로 리소스 유형 조회 (FrameworkID와 ID 사용)
func (r *ResourceTypeRepository) GetByID(frameworkID, id string) (*model.ResourceType, error) {
	var resourceType model.ResourceType
	if err := r.db.Where("framework_id = ? AND id = ?", frameworkID, id).First(&resourceType).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrResourceTypeNotFound
		}
		return nil, err
	}
	return &resourceType, nil
}

// Update 리소스 유형 정보 부분 업데이트
func (r *ResourceTypeRepository) Update(frameworkID, id string, updates map[string]interface{}) error {
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
	return nil
}

// Delete 리소스 유형 삭제
func (r *ResourceTypeRepository) Delete(frameworkID, id string) error {
	result := r.db.Where("framework_id = ? AND id = ?", frameworkID, id).Delete(&model.ResourceType{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrResourceTypeNotFound
	}
	// Note: Associated permissions will be deleted due to CASCADE constraint in DB schema
	return nil
}
