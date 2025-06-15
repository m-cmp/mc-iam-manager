package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/m-cmp/mc-iam-manager/model/mcmpapi"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm"
)

// McmpApiPermissionActionMappingService handles business logic for permission-action mappings.
type McmpApiPermissionActionMappingService struct {
	db                          *gorm.DB // Add db field
	permissionActionMappingRepo *repository.McmpApiPermissionActionMappingRepository
}

// NewMcmpApiPermissionActionMappingService creates a new service instance.
func NewMcmpApiPermissionActionMappingService(db *gorm.DB) *McmpApiPermissionActionMappingService {

	repo := repository.NewMcmpApiPermissionActionMappingRepository(db)
	return &McmpApiPermissionActionMappingService{
		permissionActionMappingRepo: repo,
	}
}

// GetPlatformActionsByPermissionID returns all API actions mapped to a specific permission.
func (s *McmpApiPermissionActionMappingService) GetPlatformActionsByPermissionID(ctx context.Context, permissionID string) ([]mcmpapi.McmpApiPermissionActionMapping, error) {
	// 플랫폼 권한은 'mc-iam-manager:' 접두사를 가지지 않는 권한
	if strings.HasPrefix(permissionID, "mc-iam-manager:") {
		return nil, fmt.Errorf("invalid platform permission ID format")
	}
	return s.permissionActionMappingRepo.FindActionsByPermissionID(ctx, permissionID)
}

// GetWorkspaceActionsByPermissionID returns all API actions mapped to a specific permission.
func (s *McmpApiPermissionActionMappingService) ListWorkspaceActionsByPermissionID(ctx context.Context, permissionID string) ([]mcmpapi.McmpApiPermissionActionMapping, error) {
	// 워크스페이스 권한은 'mc-iam-manager:' 접두사를 가진 권한
	if !strings.HasPrefix(permissionID, "mc-iam-manager:") {
		return nil, fmt.Errorf("invalid workspace permission ID format")
	}
	return s.permissionActionMappingRepo.FindActionsByPermissionID(ctx, permissionID)
}

// GetPermissionsByActionID returns all permissions mapped to a specific API action.
func (s *McmpApiPermissionActionMappingService) GetPermissionsByActionID(ctx context.Context, actionID uint) ([]mcmpapi.McmpApiPermissionActionMapping, error) {
	return s.permissionActionMappingRepo.FindPermissionsByActionID(ctx, actionID)
}

// CreateMapping creates a new permission-action mapping.
func (s *McmpApiPermissionActionMappingService) CreateMapping(ctx context.Context, permissionID string, actionID uint, actionName string) error {
	// 매핑 존재 여부 확인
	exists, err := s.permissionActionMappingRepo.CheckMappingExists(ctx, permissionID, actionID)
	if err != nil {
		return fmt.Errorf("failed to check mapping existence: %w", err)
	}
	if exists {
		return fmt.Errorf("mapping already exists")
	}

	// 매핑 생성
	mapping := &mcmpapi.McmpApiPermissionActionMapping{
		PermissionID: permissionID,
		ActionID:     actionID,
		ActionName:   actionName,
	}
	return s.permissionActionMappingRepo.CreateMapping(ctx, mapping)
}

// DeleteMapping deletes a permission-action mapping.
func (s *McmpApiPermissionActionMappingService) DeleteMapping(ctx context.Context, permissionID string, actionID uint) error {
	// 매핑 존재 여부 확인
	exists, err := s.permissionActionMappingRepo.CheckMappingExists(ctx, permissionID, actionID)
	if err != nil {
		return fmt.Errorf("failed to check mapping existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("mapping does not exist")
	}

	return s.permissionActionMappingRepo.DeleteMapping(ctx, permissionID, actionID)
}

// CheckPermissionForAction checks if a permission has access to a specific API action.
func (s *McmpApiPermissionActionMappingService) CheckPermissionForAction(ctx context.Context, permissionID string, actionID uint) (bool, error) {
	return s.permissionActionMappingRepo.CheckMappingExists(ctx, permissionID, actionID)
}

// UpdateMapping updates an existing permission-action mapping.
func (s *McmpApiPermissionActionMappingService) UpdateMapping(ctx context.Context, permissionID string, actionID uint, actionName string) error {
	// 매핑 존재 여부 확인
	exists, err := s.permissionActionMappingRepo.CheckMappingExists(ctx, permissionID, actionID)
	if err != nil {
		return fmt.Errorf("failed to check mapping existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("mapping does not exist")
	}

	// 매핑 업데이트
	mapping := &mcmpapi.McmpApiPermissionActionMapping{
		PermissionID: permissionID,
		ActionID:     actionID,
		ActionName:   actionName,
	}
	return s.permissionActionMappingRepo.UpdateMapping(ctx, mapping)
}

// SyncMcmpAPIsFromYAML YAML 파일에서 MCMP API 동기화
func (s *McmpApiPermissionActionMappingService) SyncMcmpAPIsFromYAML(ctx context.Context) error {
	// 1. YAML 파일에서 API 정보 읽기
	// 2. 기존 DB의 API 정보와 비교
	// 3. 새로운 API 추가
	// 4. 삭제된 API 제거
	// 5. 변경된 API 업데이트
	// 6. API와 permission-action 매핑 생성
	return nil
}
