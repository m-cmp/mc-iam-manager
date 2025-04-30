package service

import (
	"context"
	"fmt"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm"
)

// CspMappingService 역할-CSP 역할 매핑 관리 서비스
type CspMappingService struct {
	repo     *repository.CspMappingRepository
	roleRepo *repository.WorkspaceRoleRepository // Needed for validation
	// awsService    AwsService // Interface for AWS interactions (e.g., validation) - Define later
	// gcpService    GcpService // Interface for GCP interactions - Define later
	db *gorm.DB
}

// NewCspMappingService 새 CspMappingService 인스턴스 생성
func NewCspMappingService(db *gorm.DB) *CspMappingService {
	repo := repository.NewCspMappingRepository(db)
	roleRepo := repository.NewWorkspaceRoleRepository(db)
	return &CspMappingService{
		repo:     repo,
		roleRepo: roleRepo,
		db:       db,
	}
}

// Create 역할-CSP 역할 매핑 생성
func (s *CspMappingService) Create(ctx context.Context, mapping *model.WorkspaceRoleCspRoleMapping) error {
	// 1. Validate if workspace role exists
	if _, err := s.roleRepo.GetByID(mapping.WorkspaceRoleID); err != nil {
		return fmt.Errorf("failed to find workspace role %d: %w", mapping.WorkspaceRoleID, err)
	}

	// 2. TODO: Validate CSP Role ARN existence and potentially required permissions
	//    This requires CSP-specific logic and SDK integration.
	//    Example for AWS:
	//    if mapping.CspType == "aws" {
	//        isValid, err := s.awsService.ValidateRoleArn(ctx, mapping.CspRoleArn)
	//        if err != nil || !isValid {
	//            return fmt.Errorf("invalid AWS Role ARN %s: %w", mapping.CspRoleArn, err)
	//        }
	//        // Optionally check if the role has permissions required by the workspace role's mciam permissions
	//        requiredPermissions, _ := s.getRequiredCspPermissions(ctx, mapping.WorkspaceRoleID) // Need to implement this helper
	//        hasPermissions, err := s.awsService.CheckRolePermissions(ctx, mapping.CspRoleArn, requiredPermissions)
	// 		 if err != nil || !hasPermissions { ... return error/warning ...}
	//    }

	// 3. Create the mapping
	return s.repo.Create(mapping)
}

// ListByWorkspaceRole 워크스페이스 역할 ID로 매핑 목록 조회
func (s *CspMappingService) ListByWorkspaceRole(ctx context.Context, workspaceRoleID uint) ([]model.WorkspaceRoleCspRoleMapping, error) {
	return s.repo.ListByWorkspaceRole(workspaceRoleID)
}

// Get 역할-CSP 역할 매핑 조회
func (s *CspMappingService) Get(ctx context.Context, workspaceRoleID uint, cspType string, cspRoleArn string) (*model.WorkspaceRoleCspRoleMapping, error) {
	return s.repo.Get(workspaceRoleID, cspType, cspRoleArn)
}

// FindByRoleAndCspType 워크스페이스 역할 ID와 CSP 타입으로 매핑 목록 조회
func (s *CspMappingService) FindByRoleAndCspType(ctx context.Context, workspaceRoleID uint, cspType string) ([]model.WorkspaceRoleCspRoleMapping, error) {
	return s.repo.FindByRoleAndCspType(workspaceRoleID, cspType)
}

// Update 역할-CSP 역할 매핑 수정 (Description, IdpIdentifier)
func (s *CspMappingService) Update(ctx context.Context, workspaceRoleID uint, cspType string, cspRoleArn string, updates map[string]interface{}) error {
	// 1. Check if mapping exists
	if _, err := s.repo.Get(workspaceRoleID, cspType, cspRoleArn); err != nil {
		return err // Returns ErrCspMappingNotFound if not found
	}

	// 2. TODO: If IdpIdentifier is updated, potentially validate it?

	// 3. Perform update
	return s.repo.Update(workspaceRoleID, cspType, cspRoleArn, updates)
}

// Delete 역할-CSP 역할 매핑 삭제
func (s *CspMappingService) Delete(ctx context.Context, workspaceRoleID uint, cspType string, cspRoleArn string) error {
	// Check existence before delete is handled by repo
	return s.repo.Delete(workspaceRoleID, cspType, cspRoleArn)
}

// Helper function placeholder for getting required CSP permissions (needs implementation)
// func (s *CspMappingService) getRequiredCspPermissions(ctx context.Context, workspaceRoleID uint) ([]string, error) {
//	 // 1. Get all mciam_permission_ids associated with workspaceRoleID from mciam_role_mciam_permissions table
//	 // 2. For each permission_id, get the required_csp_permissions JSON from mcmp_mciam_permissions table
//	 // 3. Aggregate and deduplicate all required CSP permission strings
//	 return []string{}, nil
// }
