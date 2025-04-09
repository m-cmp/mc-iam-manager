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
func (s *PermissionService) GetByID(ctx context.Context, id uint) (*model.Permission, error) {
	return s.permissionRepo.GetByID(ctx, id)
}

// List 권한 목록 조회
func (s *PermissionService) List(ctx context.Context) ([]model.Permission, error) {
	return s.permissionRepo.List(ctx)
}

// Update 권한 수정
func (s *PermissionService) Update(ctx context.Context, permission *model.Permission) error {
	existing, err := s.permissionRepo.GetByID(ctx, permission.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("권한을 찾을 수 없습니다")
	}
	return s.permissionRepo.Update(ctx, permission)
}

// Delete 권한 삭제
func (s *PermissionService) Delete(ctx context.Context, id uint) error {
	existing, err := s.permissionRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("권한을 찾을 수 없습니다")
	}
	return s.permissionRepo.Delete(ctx, id)
}

// AssignRolePermission 역할에 권한 할당
func (s *PermissionService) AssignRolePermission(ctx context.Context, roleID, permissionID uint) error {
	// 권한 존재 여부 확인
	permission, err := s.permissionRepo.GetByID(ctx, permissionID)
	if err != nil {
		return err
	}
	if permission == nil {
		return errors.New("권한을 찾을 수 없습니다")
	}
	return s.permissionRepo.AssignRolePermission(ctx, roleID, permissionID)
}

// RemoveRolePermission 역할에서 권한 제거
func (s *PermissionService) RemoveRolePermission(ctx context.Context, roleID, permissionID uint) error {
	return s.permissionRepo.RemoveRolePermission(ctx, roleID, permissionID)
}

// GetRolePermissions 역할의 권한 목록 조회
func (s *PermissionService) GetRolePermissions(ctx context.Context, roleID uint) ([]model.Permission, error) {
	return s.permissionRepo.GetRolePermissions(ctx, roleID)
}
