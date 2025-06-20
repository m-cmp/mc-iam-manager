package service

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/csp"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm"
)

type CspRoleService struct {
	db          *gorm.DB
	cspRoleRepo *repository.CspRoleRepository
	roleService *RoleService
}

func NewCspRoleService(db *gorm.DB) *CspRoleService {
	cspRoleRepo := repository.NewCspRoleRepository(db)
	roleService := NewRoleService(db)

	return &CspRoleService{
		cspRoleRepo: cspRoleRepo,
		roleService: roleService,
		db:          db,
	}
}

// GetAllCSPRoles 모든 CSP 역할을 조회합니다.
func (s *CspRoleService) GetAllCSPRoles(ctx context.Context, cspType string) ([]*model.CspRole, error) {
	roles, err := s.cspRoleRepo.FindAll()
	if err != nil {
		log.Printf("Failed to get CSP roles: %v", err)
		return nil, err
	}

	return roles, nil
}

// CSP 역할 목록 중 MCIAM_ 접두사를 가진 역할만 조회합니다.
func (s *CspRoleService) GetMciamCSPRoles(ctx context.Context, cspType string) ([]*model.CspRole, error) {
	roles, err := s.cspRoleRepo.FindMciamRoleFromCsp(cspType)
	if err != nil {
		log.Printf("Failed to get CSP roles: %v", err)
		return nil, err
	}

	return roles, nil
}

// CreateCSPRole CSP 역할을 생성하고 RoleMaster와 매핑합니다.
func (s *CspRoleService) CreateCSPRole(req *model.CreateCspRoleRequest) (*model.CspRole, error) {
	// 1. RoleMaster와 RoleSub 처리 (먼저 처리)
	roleMasterID, err := s.handleRoleMasterAndSub(req)
	if err != nil {
		return nil, fmt.Errorf("failed to handle role master and sub: %w", err)
	}

	// 2. CSP 역할 처리 (독립적으로 처리)
	cspRole, err := s.handleCspRole(req)
	if err != nil {
		return nil, fmt.Errorf("failed to handle CSP role: %w", err)
	}

	// 3. RoleMaster와 CSP Role 매핑
	err = s.createRoleMapping(roleMasterID, cspRole.ID, req.Description)
	if err != nil {
		return nil, fmt.Errorf("failed to create role mapping: %w", err)
	}

	return cspRole, nil
}

// handleCspRole CSP 역할을 처리합니다.
// 1-1. CSP 역할이 있으면 대상 CSP에서 역할 조회하여 정보를 CspRole 테이블에 저장
// 1-2. CSP 역할이 없으면 대상 CSP에 역할을 추가하고 5초간 대기한 후 조회해서 정보를 저장
func (s *CspRoleService) handleCspRole(req *model.CreateCspRoleRequest) (*model.CspRole, error) {
	// CSP 역할 존재 여부 확인 (DB에서)
	exists, err := s.ExistCspRoleByName(req.RoleName)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing CSP role: %w", err)
	}

	if exists {
		// 1-1. CSP 역할이 있으면 대상 CSP에서 역할 조회하여 정보를 CspRole 테이블에 저장
		existingCspRole, err := s.cspRoleRepo.GetRoleByName(req.RoleName)
		if err != nil {
			return nil, fmt.Errorf("failed to get existing CSP role: %w", err)
		}

		// 실제 CSP에서 최신 정보를 조회하여 업데이트
		updatedCspRole, err := s.syncCspRoleFromCloud(existingCspRole)
		if err != nil {
			log.Printf("Warning: failed to sync CSP role from cloud: %v", err)
			// 동기화 실패해도 기존 정보로 계속 진행
			return existingCspRole, nil
		}

		log.Printf("CSP role already exists and synced: %s (ID: %d)", req.RoleName, updatedCspRole.ID)
		return updatedCspRole, nil
	} else {
		// 1-2. CSP 역할이 없으면 대상 CSP에 역할을 추가하고 5초간 대기한 후 조회해서 정보를 저장
		cspRole, err := s.cspRoleRepo.CreateCSPRole(req)
		if err != nil {
			log.Printf("Failed to create CSP role: %v", err)
			return nil, err
		}
		return cspRole, nil
	}
}

// syncCspRoleFromCloud 실제 CSP에서 역할 정보를 조회하여 DB를 업데이트합니다.
func (s *CspRoleService) syncCspRoleFromCloud(cspRole *model.CspRole) (*model.CspRole, error) {
	// repository를 통해 실제 CSP에서 최신 역할 정보 조회 및 DB 업데이트
	updatedRole, err := s.cspRoleRepo.SyncCspRoleFromCloud(cspRole.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to sync CSP role from cloud: %w", err)
	}

	return updatedRole, nil
}

// handleRoleMasterAndSub RoleMaster와 RoleSub를 처리합니다.
// 2-1. 없는 경우에는 RoleMaster와 RoleSub에 저장하고 해당 RoleMaster.ID를 보관
// 2-2. RoleMaster에만 있고 RoleSub에 없는 경우 RoleSub에 추가하고 RoleMaster.ID를 보관
// 2-3. RoleMaster, RoleSub에 모두 있으면 RoleMaster.ID를 보관
func (s *CspRoleService) handleRoleMasterAndSub(req *model.CreateCspRoleRequest) (uint, error) {
	// RoleMaster 존재 여부 확인
	roleExists, err := s.roleService.ExistRoleByName(req.RoleName, constants.RoleTypeCSP)
	if err != nil {
		return 0, fmt.Errorf("failed to check existing role: %w", err)
	}

	if roleExists {
		// 기존 역할이 있는 경우 해당 ID 조회
		existingRole, err := s.roleService.GetRoleByName(req.RoleName, constants.RoleTypeCSP)
		if err != nil {
			return 0, fmt.Errorf("failed to get existing role: %w", err)
		}
		return existingRole.ID, nil
	} else {
		// 2-1. RoleMaster와 RoleSub가 없는 경우 새로 생성
		roleMaster := model.RoleMaster{
			Name:        req.RoleName,
			Description: req.Description,
			Predefined:  false, // CSP 역할은 기본적으로 predefined가 false
		}

		roleSubs := []model.RoleSub{
			{
				RoleType: constants.RoleTypeCSP, // CSP 역할은 항상 "csp" 타입
			},
		}

		// RoleMaster와 RoleSubs 함께 생성
		createdRole, err := s.roleService.CreateRoleWithSubs(&roleMaster, roleSubs)
		if err != nil {
			return 0, fmt.Errorf("역할 서브 타입 생성 실패: %w", err)
		}
		return createdRole.ID, nil
	}
}

// createRoleMapping RoleMaster와 CSP Role 매핑을 생성합니다.
// 중복 체크 후 저장하고, 이미 있는 경우는 로그만 남깁니다.
func (s *CspRoleService) createRoleMapping(roleMasterID uint, cspRoleID uint, description string) error {
	// 매핑 요청 객체 생성
	mappingReq := &model.RoleMasterCspRoleMappingRequest{
		RoleID:      fmt.Sprintf("%d", roleMasterID),
		CspRoleID:   fmt.Sprintf("%d", cspRoleID),
		AuthMethod:  constants.AuthMethodOIDC,
		Description: description,
	}

	// repository를 통해 매핑 생성 (중복 체크 포함)
	err := s.roleService.CreateRoleCspRoleMapping(mappingReq)
	if err != nil {
		// 중복 에러인 경우 로그만 남기고 계속 진행
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "already exists") {
			log.Printf("Role mapping already exists for role_id=%d, csp_role_id=%d", roleMasterID, cspRoleID)
			return nil
		}
		return fmt.Errorf("failed to create role mapping: %w", err)
	}

	return nil
}

// UpdateCSPRole CSP 역할 정보를 수정합니다.
func (s *CspRoleService) UpdateCSPRole(role *model.CspRole) error {
	err := s.cspRoleRepo.UpdateCSPRole(role)
	if err != nil {
		log.Printf("Failed to update CSP role: %v", err)
		return err
	}

	return nil
}

// DeleteCSPRole CSP 역할을 삭제합니다.
func (s *CspRoleService) DeleteCSPRole(id string) error {
	err := s.cspRoleRepo.DeleteCSPRole(id)
	if err != nil {
		log.Printf("Failed to delete CSP role: %v", err)
		return err
	}

	return nil
}

// AddPermissionsToCSPRole CSP 역할에 권한을 추가합니다.
func (s *CspRoleService) AddPermissionsToCSPRole(roleID string, permissions []string) error {
	err := s.cspRoleRepo.AddPermissionsToCSPRole(roleID, permissions)
	if err != nil {
		log.Printf("Failed to add permissions to CSP role: %v", err)
		return err
	}
	return nil
}

// RemovePermissionsFromCSPRole CSP 역할에서 권한을 제거합니다.
func (s *CspRoleService) RemovePermissionsFromCSPRole(roleID string, permissions []string) error {
	err := s.cspRoleRepo.RemovePermissionsFromCSPRole(roleID, permissions)
	if err != nil {
		log.Printf("Failed to remove permissions from CSP role: %v", err)
		return err
	}
	return nil
}

// GetCSPRolePermissions CSP 역할의 권한 목록을 조회합니다.
func (s *CspRoleService) GetCSPRolePermissions(roleID string) ([]string, error) {
	permissions, err := s.cspRoleRepo.GetCSPRolePermissions(roleID)
	if err != nil {
		log.Printf("Failed to get CSP role permissions: %v", err)
		return nil, err
	}
	return permissions, nil
}

// GetRolePolicies 역할의 정책 목록 조회
func (s *CspRoleService) GetRolePolicies(ctx context.Context, roleName string) (*model.CspRole, error) {
	// 1. 역할 존재 여부 확인
	role, err := s.cspRoleRepo.GetRoleByName(roleName)
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	// 2. 관리형 정책 목록 조회
	managedPolicies, err := s.cspRoleRepo.ListAttachedRolePolicies(ctx, roleName)
	if err != nil {
		return nil, fmt.Errorf("failed to list attached role policies: %w", err)
	}

	// 3. 인라인 정책 목록 조회
	inlinePolicies, err := s.cspRoleRepo.ListRolePolicies(ctx, roleName)
	if err != nil {
		return nil, fmt.Errorf("failed to list role policies: %w", err)
	}

	role.Permissions = managedPolicies
	role.Permissions = append(role.Permissions, inlinePolicies...)

	return role, nil
}

// GetRolePolicy 역할의 특정 인라인 정책 조회
func (s *CspRoleService) GetRolePolicy(ctx context.Context, roleName string, policyName string) (*csp.RolePolicy, error) {
	return s.cspRoleRepo.GetRolePolicy(ctx, roleName, policyName)
}

// PutRolePolicy 역할에 인라인 정책 추가/수정
func (s *CspRoleService) PutRolePolicy(ctx context.Context, roleName string, policyName string, policy *csp.RolePolicy) error {
	return s.cspRoleRepo.PutRolePolicy(ctx, roleName, policyName, policy)
}

// DeleteRolePolicy 역할에서 인라인 정책 삭제
func (s *CspRoleService) DeleteRolePolicy(ctx context.Context, roleName string, policyName string) error {
	return s.cspRoleRepo.DeleteRolePolicy(ctx, roleName, policyName)
}

// CreateOrUpdateCspRole CSP 역할을 생성하거나 업데이트합니다.
// ID가 비어있으면 새로 생성하고, ID가 있으면 기존 것을 업데이트합니다.
func (s *CspRoleService) CreateOrUpdateCspRole(cspRole *model.CspRole) (*model.CspRole, error) {
	if cspRole.ID == 0 {
		// ID가 비어있으면 새로 생성
		req := &model.CreateCspRoleRequest{
			RoleName:      cspRole.Name,
			Description:   cspRole.Description,
			CspType:       cspRole.CspType,
			IdpIdentifier: cspRole.IdpIdentifier,
			IamIdentifier: cspRole.IamIdentifier,
			Status:        cspRole.Status,
			Path:          cspRole.Path,
			IamRoleId:     cspRole.IamRoleId,
		}
		return s.CreateCSPRole(req)
	} else {
		// ID가 있으면 업데이트
		err := s.UpdateCSPRole(cspRole)
		if err != nil {
			return nil, err
		}
		return cspRole, nil
	}
}

// CreateCspRoles 복수 CSP 역할을 생성합니다.
func (s *CspRoleService) CreateCspRoles(req *model.CreateCspRolesRequest) ([]*model.CspRole, error) {
	var createdRoles []*model.CspRole

	for _, cspRoleReq := range req.CspRoles {
		createdRole, err := s.CreateCSPRole(&cspRoleReq)
		if err != nil {
			return nil, fmt.Errorf("failed to create CSP role '%s': %w", cspRoleReq.RoleName, err)
		}
		createdRoles = append(createdRoles, createdRole)
	}

	return createdRoles, nil
}

// GetCspRoleByName 이름으로 CSP 역할을 조회합니다.
func (s *CspRoleService) GetCspRoleByName(roleName string) (*model.CspRole, error) {
	role, err := s.cspRoleRepo.GetRoleByName(roleName)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 역할이 존재하지 않음
		}
		return nil, fmt.Errorf("failed to get CSP role by name: %w", err)
	}
	return role, nil
}

// ExistCspRoleByName 이름으로 CSP 역할 존재 여부를 확인합니다 (CspRole 테이블에서)
func (s *CspRoleService) ExistCspRoleByName(roleName string) (bool, error) {
	return s.cspRoleRepo.ExistCspRoleByName(roleName)
}
