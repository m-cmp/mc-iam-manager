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
	roleRepo          *repository.RoleRepository
	workspaceRoleRepo *repository.WorkspaceRoleRepository
	cspRoleRepo       *repository.CspRoleRepository
	cspMappingRepo    *repository.CspMappingRepository
	// awsService    AwsService // Interface for AWS interactions (e.g., validation) - Define later
	// gcpService    GcpService // Interface for GCP interactions - Define later
	db *gorm.DB
}

// NewCspMappingService 새 CspMappingService 인스턴스 생성
func NewCspMappingService(
	db *gorm.DB,
) *CspMappingService {
	roleRepo := repository.NewRoleRepository(db)
	workspaceRoleRepo := repository.NewWorkspaceRoleRepository(db)
	cspRoleRepo := repository.NewCspRoleRepository(db)
	cspMappingRepo := repository.NewCspMappingRepository(db)
	return &CspMappingService{
		roleRepo:          roleRepo,
		workspaceRoleRepo: workspaceRoleRepo,
		cspRoleRepo:       cspRoleRepo,
		cspMappingRepo:    cspMappingRepo,
		db:                db,
	}
}

// CreateWorkspaceRoleCspRoleMapping 워크스페이스 역할과 CSP 역할 매핑 생성
func (s *CspMappingService) CreateRoleCspRoleMapping(ctx context.Context, mapping *model.CreateRoleCspRoleMappingRequest) error {
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

	roleMapping := model.CreateCspRolesMappingRequest{}
	roleMapping.RoleID = mapping.WorkspaceRoleID
	roleMapping.AuthMethod = mapping.AuthMethod
	roleMapping.CspRoles = mapping.CspRoles

	return s.roleRepo.CreateWorkspaceRoleCspRoleMapping(&roleMapping)
}

// DeleteWorkspaceRoleCspRoleMapping 워크스페이스 역할과 CSP 역할 매핑 삭제
func (s *CspMappingService) DeleteWorkspaceRoleCspRoleMapping(ctx context.Context, workspaceRoleID uint, cspRoleID uint, cspType string) error {
	return s.roleRepo.DeleteWorkspaceRoleCspRoleMapping(workspaceRoleID, cspRoleID, cspType)
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

	// CSP 역할 존재 여부 확인 (CspRoles 배열에서 첫 번째 요소 사용)
	if len(mapping.CspRoles) == 0 {
		return ErrCspRoleNotFound
	}
	exists, err = s.cspRoleRepo.ExistsCspRoleByID(mapping.CspRoles[0].ID)
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
