package service

import (
	"context"
	"errors"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm" // Import gorm
)

// PermissionService 권한 관리 서비스
type PermissionService struct {
	db             *gorm.DB // Add db field
	permissionRepo *repository.PermissionRepository
}

// NewPermissionService 권한 관리 서비스 생성
func NewPermissionService(db *gorm.DB) *PermissionService { // Removed permissionRepo parameter
	// Initialize repository internally
	permissionRepo := repository.NewPermissionRepository()
	return &PermissionService{
		db:             db, // Store db
		permissionRepo: permissionRepo,
	}
}

// Create 권한 생성
func (s *PermissionService) Create(ctx context.Context, permission *model.Permission) error {
	tx := s.db.WithContext(ctx)                    // Create transaction with context
	return s.permissionRepo.Create(tx, permission) // Pass tx
}

// GetByID ID로 권한 조회
func (s *PermissionService) GetByID(ctx context.Context, id string) (*model.Permission, error) { // Changed id type to string
	tx := s.db.WithContext(ctx)             // Create transaction with context
	return s.permissionRepo.GetByID(tx, id) // Pass tx
}

// List 권한 목록 조회
func (s *PermissionService) List(ctx context.Context) ([]model.Permission, error) {
	tx := s.db.WithContext(ctx)      // Create transaction with context
	return s.permissionRepo.List(tx) // Pass tx
}

// Update 권한 수정
func (s *PermissionService) Update(ctx context.Context, permission *model.Permission) error {
	tx := s.db.WithContext(ctx) // Create transaction with context
	// permission.ID is already string, no change needed for GetByID call
	existing, err := s.permissionRepo.GetByID(tx, permission.ID) // Pass tx
	if err != nil {
		return err // Propagate error (e.g., DB connection issue)
	}
	if existing == nil {
		return errors.New("권한을 찾을 수 없습니다")
	}
	return s.permissionRepo.Update(tx, permission) // Pass tx
}

// Delete 권한 삭제
func (s *PermissionService) Delete(ctx context.Context, id string) error { // Changed id type to string
	tx := s.db.WithContext(ctx) // Create transaction with context
	// Check existence before deleting (optional, as repo Delete handles it)
	// existing, err := s.permissionRepo.GetByID(tx, id) // Pass tx if uncommented
	// if err != nil { return err }
	// if existing == nil { return errors.New("권한을 찾을 수 없습니다") }
	return s.permissionRepo.Delete(tx, id) // Pass tx, Pass string id
}

// AssignRolePermission 역할에 권한 할당
func (s *PermissionService) AssignRolePermission(ctx context.Context, roleType string, roleID uint, permissionID string) error { // Added roleType, changed permissionID type
	tx := s.db.WithContext(ctx) // Create transaction with context
	// 권한 존재 여부 확인
	permission, err := s.permissionRepo.GetByID(tx, permissionID) // Pass tx, Pass string permissionID
	if err != nil {
		// Handle specific "not found" error from repo if needed
		return err
	}
	if permission == nil {
		// This case might be covered by the error check above if repo returns specific error
		return errors.New("권한을 찾을 수 없습니다")
	}
	// TODO: 역할 존재 여부 확인 (Platform or Workspace)
	return s.permissionRepo.AssignRolePermission(tx, roleType, roleID, permissionID) // Pass tx, Pass all args
}

// RemoveRolePermission 역할에서 권한 제거
func (s *PermissionService) RemoveRolePermission(ctx context.Context, roleType string, roleID uint, permissionID string) error { // Added roleType, changed permissionID type
	tx := s.db.WithContext(ctx) // Create transaction with context
	// No need to check existence first, repo handles it gracefully
	return s.permissionRepo.RemoveRolePermission(tx, roleType, roleID, permissionID) // Pass tx, Pass all args
}

// GetRolePermissions 역할의 권한 목록 조회
func (s *PermissionService) GetRolePermissions(ctx context.Context, roleType string, roleID uint) ([]model.Permission, error) { // Added roleType
	tx := s.db.WithContext(ctx) // Create transaction with context
	// TODO: 역할 존재 여부 확인 (Platform or Workspace)
	return s.permissionRepo.GetRolePermissions(tx, roleType, roleID) // Pass tx, Pass all args
}
