package service

import (
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm"
)

// ResourceTypeService 리소스 유형 관리 서비스
type ResourceTypeService struct {
	repo *repository.ResourceTypeRepository
	// permissionRepo *repository.PermissionRepository // Optional: Needed if auto-creating permissions
	db *gorm.DB // Needed for potential transactions
}

// NewResourceTypeService 새 ResourceTypeService 인스턴스 생성
func NewResourceTypeService(db *gorm.DB) *ResourceTypeService {
	repo := repository.NewResourceTypeRepository(db)
	// permissionRepo := repository.NewPermissionRepository(db) // Initialize if needed
	return &ResourceTypeService{
		repo: repo,
		// permissionRepo: permissionRepo,
		db: db,
	}
}

// Create 리소스 유형 생성
func (s *ResourceTypeService) CreateResourceType(resourceType *model.ResourceType) error {
	// Basic validation can be added here if needed
	err := s.repo.CreateResourceType(resourceType)
	if err != nil {
		return err
	}

	// Optional: Automatically create default CRUD permissions for the new resource type
	// permissionsToCreate := []model.Permission{
	// 	{ID: fmt.Sprintf("%s:%s:create", resourceType.FrameworkID, resourceType.ID), FrameworkID: resourceType.FrameworkID, ResourceTypeID: resourceType.ID, Action: "create", Name: fmt.Sprintf("Create %s", resourceType.Name), Description: fmt.Sprintf("Allow creating %s", resourceType.Name)},
	// 	{ID: fmt.Sprintf("%s:%s:read", resourceType.FrameworkID, resourceType.ID), FrameworkID: resourceType.FrameworkID, ResourceTypeID: resourceType.ID, Action: "read", Name: fmt.Sprintf("Read %s", resourceType.Name), Description: fmt.Sprintf("Allow reading %s", resourceType.Name)},
	// 	{ID: fmt.Sprintf("%s:%s:update", resourceType.FrameworkID, resourceType.ID), FrameworkID: resourceType.FrameworkID, ResourceTypeID: resourceType.ID, Action: "update", Name: fmt.Sprintf("Update %s", resourceType.Name), Description: fmt.Sprintf("Allow updating %s", resourceType.Name)},
	// 	{ID: fmt.Sprintf("%s:%s:delete", resourceType.FrameworkID, resourceType.ID), FrameworkID: resourceType.FrameworkID, ResourceTypeID: resourceType.ID, Action: "delete", Name: fmt.Sprintf("Delete %s", resourceType.Name), Description: fmt.Sprintf("Allow deleting %s", resourceType.Name)},
	// }
	// for _, p := range permissionsToCreate {
	// 	if err := s.permissionRepo.Create(&p); err != nil {
	// 		// Log error but don't fail the resource type creation? Or use transaction?
	// 		log.Printf("Warning: Failed to auto-create permission %s: %v", p.ID, err)
	// 	}
	// }

	return nil
}

// List 모든 리소스 유형 조회 (프레임워크 ID로 필터링 가능)
func (s *ResourceTypeService) ListResourceTypes(frameworkID string) ([]model.ResourceType, error) {
	return s.repo.FindResourceTypes(frameworkID)
}

// GetByID ID로 리소스 유형 조회
func (s *ResourceTypeService) GetResourceTypeByID(frameworkID, resourceTypeId string) (*model.ResourceType, error) {
	return s.repo.FindResourceTypeByID(frameworkID, resourceTypeId)
}

// Update 리소스 유형 정보 부분 업데이트
func (s *ResourceTypeService) Update(frameworkID, resourceTypeId string, updates map[string]interface{}) error {
	// Add validation or business logic if needed before updating
	return s.repo.UpdateResourceType(frameworkID, resourceTypeId, updates)
}

// Delete 리소스 유형 삭제
func (s *ResourceTypeService) DeleteResourceType(frameworkID, resourceTypeId string) error {
	// Add validation or business logic if needed before deleting
	// Note: Associated permissions will be deleted by DB cascade constraint
	return s.repo.DeleteResourceType(frameworkID, resourceTypeId)
}
