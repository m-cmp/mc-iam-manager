package service

import (
	"context"

	"github.com/m-cmp/mc-iam-manager/model/mcmpapi"
	"github.com/m-cmp/mc-iam-manager/repository"
)

// McmpApiPermissionActionMappingService handles business logic for permission-action mappings.
type McmpApiPermissionActionMappingService struct {
	repo *repository.McmpApiPermissionActionMappingRepository
}

// NewMcmpApiPermissionActionMappingService creates a new service instance.
func NewMcmpApiPermissionActionMappingService(repo *repository.McmpApiPermissionActionMappingRepository) *McmpApiPermissionActionMappingService {
	return &McmpApiPermissionActionMappingService{repo: repo}
}

// GetActionsByPermissionID returns all API actions mapped to a specific permission.
func (s *McmpApiPermissionActionMappingService) GetActionsByPermissionID(ctx context.Context, permissionID string) ([]mcmpapi.McmpApiAction, error) {
	return s.repo.GetActionsByPermissionID(ctx, permissionID)
}

// GetPermissionsByActionID returns all permissions mapped to a specific API action.
func (s *McmpApiPermissionActionMappingService) GetPermissionsByActionID(ctx context.Context, actionID uint) ([]string, error) {
	return s.repo.GetPermissionsByActionID(ctx, actionID)
}

// CreateMapping creates a new permission-action mapping.
func (s *McmpApiPermissionActionMappingService) CreateMapping(ctx context.Context, permissionID string, actionID uint) error {
	exists, err := s.repo.CheckMappingExists(ctx, permissionID, actionID)
	if err != nil {
		return err
	}
	if exists {
		return nil // 이미 존재하는 매핑은 무시
	}
	return s.repo.CreateMapping(ctx, permissionID, actionID)
}

// DeleteMapping deletes a permission-action mapping.
func (s *McmpApiPermissionActionMappingService) DeleteMapping(ctx context.Context, permissionID string, actionID uint) error {
	return s.repo.DeleteMapping(ctx, permissionID, actionID)
}

// CheckPermissionForAction checks if a permission has access to a specific API action.
func (s *McmpApiPermissionActionMappingService) CheckPermissionForAction(ctx context.Context, permissionID string, actionID uint) (bool, error) {
	return s.repo.CheckMappingExists(ctx, permissionID, actionID)
}
