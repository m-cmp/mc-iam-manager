package service

import (
	"context"
	"errors"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/m-cmp/mc-iam-manager/util"
	"gorm.io/gorm"
)

var (
	ErrWorkspaceRoleNotFound = errors.New("워크스페이스 역할을 찾을 수 없습니다")
	ErrCspRoleNotFound       = errors.New("CSP 역할을 찾을 수 없습니다")
)

// CspMappingService CSP 매핑 서비스
type CspMappingService struct {
	cspMappingRepo    *repository.CspMappingRepository
	workspaceRoleRepo *repository.WorkspaceRoleRepository
	cspRoleRepo       *repository.CspRoleRepository
	// awsService    AwsService // Interface for AWS interactions (e.g., validation) - Define later
	// gcpService    GcpService // Interface for GCP interactions - Define later
	db *gorm.DB
}

// NewCspMappingService 새 CspMappingService 인스턴스 생성
func NewCspMappingService(
	cspMappingRepo *repository.CspMappingRepository,
	workspaceRoleRepo *repository.WorkspaceRoleRepository,
	cspRoleRepo *repository.CspRoleRepository,
	db *gorm.DB,
) *CspMappingService {
	return &CspMappingService{
		cspMappingRepo:    cspMappingRepo,
		workspaceRoleRepo: workspaceRoleRepo,
		cspRoleRepo:       cspRoleRepo,
		db:                db,
	}
}

// GetWorkspaceRoleCspRoleMappings 워크스페이스 역할의 CSP 역할 매핑 목록 조회
func (s *CspMappingService) GetWorkspaceRoleCspRoleMappings(ctx context.Context, workspaceRoleID uint) ([]*model.RoleMasterCspRoleMapping, error) {
	return s.cspMappingRepo.FindCspRoleMappingsByWorkspaceRoleID(ctx, workspaceRoleID)
}

// CreateWorkspaceRoleCspRoleMapping 워크스페이스 역할과 CSP 역할 매핑 생성
func (s *CspMappingService) CreateWorkspaceRoleCspRoleMapping(ctx context.Context, mapping *model.WorkspaceRoleCspRoleMappingRequest) error {
	var workspaceRoleID uint
	if mapping.WorkspaceRoleID != "" {
		// roleId가 있으면 uint로 변환
		roleIDInt, err := util.StringToUint(mapping.WorkspaceRoleID)
		if err != nil {
			return err
		}
		workspaceRoleID = roleIDInt
	}
	var cspRoleID uint
	if mapping.CspRoleID != "" {
		// roleId가 있으면 uint로 변환
		cspRoleIDInt, err := util.StringToUint(mapping.CspRoleID)
		if err != nil {
			return err
		}
		cspRoleID = cspRoleIDInt
	}

	// 워크스페이스 역할 존재 여부 확인
	exists, err := s.workspaceRoleRepo.ExistsWorkspaceRoleByID(workspaceRoleID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrWorkspaceRoleNotFound
	}

	// CSP 역할 존재 여부 확인
	exists, err = s.cspRoleRepo.ExistsCspRoleByID(cspRoleID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCspRoleNotFound
	}

	roleMapping := model.RoleMasterCspRoleMapping{}
	roleMapping.RoleID = workspaceRoleID
	roleMapping.CspType = mapping.CspType
	roleMapping.CspRoleID = cspRoleID

	return s.cspMappingRepo.CreateWorkspaceRoleCspRoleMapping(ctx, &roleMapping)
}

// DeleteWorkspaceRoleCspRoleMapping 워크스페이스 역할과 CSP 역할 매핑 삭제
func (s *CspMappingService) DeleteWorkspaceRoleCspRoleMapping(ctx context.Context, workspaceRoleID uint, cspType string, cspRoleID string) error {
	return s.cspMappingRepo.DeleteWorkspaceRoleCspRoleMapping(ctx, workspaceRoleID, cspType, cspRoleID)
}

// GetWorkspaceRoleCspRoleMappingsByCspType 워크스페이스 역할 ID와 CSP 타입으로 CSP 역할 매핑 목록 조회
func (s *CspMappingService) GetWorkspaceRoleCspRoleMappingsByCspType(workspaceRoleID uint, cspType string) ([]*model.RoleMasterCspRoleMapping, error) {
	return s.cspMappingRepo.FindCspRoleMappingsByWorkspaceRoleIDAndCspType(workspaceRoleID, cspType)
}

// UpdateWorkspaceRoleCspRoleMapping 워크스페이스 역할 - CSP 역할 매핑 수정
func (s *CspMappingService) UpdateWorkspaceRoleCspRoleMapping(mapping *model.RoleMasterCspRoleMapping) error {
	// 워크스페이스 역할 존재 여부 확인
	exists, err := s.workspaceRoleRepo.ExistsWorkspaceRoleByID(mapping.RoleID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrWorkspaceRoleNotFound
	}

	// CSP 역할 존재 여부 확인
	exists, err = s.cspRoleRepo.ExistsCspRoleByID(mapping.CspRoleID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCspRoleNotFound
	}

	return s.cspMappingRepo.UpdateWorkspaceRoleCspRoleMapping(mapping)
}

// Helper function placeholder for getting required CSP permissions (needs implementation)
// func (s *CspMappingService) getRequiredCspPermissions(ctx context.Context, workspaceRoleID uint) ([]string, error) {
//	 // 1. Get all mciam_permission_ids associated with workspaceRoleID from mciam_role_mciam_permissions table
//	 // 2. For each permission_id, get the required_csp_permissions JSON from mcmp_mciam_permissions table
//	 // 3. Aggregate and deduplicate all required CSP permission strings
//	 return []string{}, nil
// }
