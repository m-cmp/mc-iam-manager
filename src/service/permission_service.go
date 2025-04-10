package service

import (
	"context"
	"errors"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
)

// PermissionService 권한 관리 서비스
type PermissionService struct {
	permissionRepo *repository.PermissionRepository
}

// NewPermissionService 권한 관리 서비스 생성
func NewPermissionService(permissionRepo *repository.PermissionRepository) *PermissionService {
	return &PermissionService{
		permissionRepo: permissionRepo,
	}
}

// Create 권한 생성
func (s *PermissionService) Create(ctx context.Context, permission *model.Permission) error {
	return s.permissionRepo.Create(ctx, permission)
}

// GetByID ID로 권한 조회
func (s *PermissionService) GetByID(ctx context.Context, id string) (*model.Permission, error) { // Changed id type to string
	return s.permissionRepo.GetByID(ctx, id)
}

// List 권한 목록 조회
func (s *PermissionService) List(ctx context.Context) ([]model.Permission, error) {
	return s.permissionRepo.List(ctx)
}

// Update 권한 수정
func (s *PermissionService) Update(ctx context.Context, permission *model.Permission) error {
	// permission.ID is already string, no change needed for GetByID call
	existing, err := s.permissionRepo.GetByID(ctx, permission.ID)
	if err != nil {
		return err // Propagate error (e.g., DB connection issue)
	}
	if existing == nil {
		return errors.New("권한을 찾을 수 없습니다")
	}
	return s.permissionRepo.Update(ctx, permission)
}

// Delete 권한 삭제
func (s *PermissionService) Delete(ctx context.Context, id string) error { // Changed id type to string
	// Check existence before deleting (optional, as repo Delete handles it)
	// existing, err := s.permissionRepo.GetByID(ctx, id)
	// if err != nil { return err }
	// if existing == nil { return errors.New("권한을 찾을 수 없습니다") }
	return s.permissionRepo.Delete(ctx, id) // Pass string id
}

// AssignRolePermission 역할에 권한 할당
func (s *PermissionService) AssignRolePermission(ctx context.Context, roleType string, roleID uint, permissionID string) error { // Added roleType, changed permissionID type
	// 권한 존재 여부 확인
	permission, err := s.permissionRepo.GetByID(ctx, permissionID) // Pass string permissionID
	if err != nil {
		// Handle specific "not found" error from repo if needed
		return err
	}
	if permission == nil {
		// This case might be covered by the error check above if repo returns specific error
		return errors.New("권한을 찾을 수 없습니다")
	}
	// TODO: 역할 존재 여부 확인 (Platform or Workspace)
	return s.permissionRepo.AssignRolePermission(ctx, roleType, roleID, permissionID) // Pass all args
}

// RemoveRolePermission 역할에서 권한 제거
func (s *PermissionService) RemoveRolePermission(ctx context.Context, roleType string, roleID uint, permissionID string) error { // Added roleType, changed permissionID type
	// No need to check existence first, repo handles it gracefully
	return s.permissionRepo.RemoveRolePermission(ctx, roleType, roleID, permissionID) // Pass all args
}

// GetRolePermissions 역할의 권한 목록 조회
func (s *PermissionService) GetRolePermissions(ctx context.Context, roleType string, roleID uint) ([]model.Permission, error) { // Added roleType
	// TODO: 역할 존재 여부 확인 (Platform or Workspace)
	return s.permissionRepo.GetRolePermissions(ctx, roleType, roleID) // Pass all args
}
