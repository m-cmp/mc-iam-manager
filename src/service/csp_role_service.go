package service

import (
	"context"
	"fmt"
	"log"

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

// CreateCSPRole 새로운 CSP 역할을 생성합니다.
func (s *CspRoleService) CreateCSPRole(req *model.CreateCspRoleRequest) (*model.CspRole, error) {
	// 1. Role Master에서 역할 존재 여부 확인
	existingRole, err := s.roleService.GetRoleByName(req.RoleName, constants.RoleTypeCSP)
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Printf("Failed to check existing role: %v", err)
		return nil, err
	}

	var roleMasterID uint
	if existingRole != nil {
		// 기존 역할이 있는 경우 해당 ID 사용
		roleMasterID = existingRole.ID
	} else {
		// 1. RoleMaster 생성
		roleMaster := model.RoleMaster{
			Name:        req.RoleName,
			Description: req.Description,
			Predefined:  false, // CSP 역할은 기본적으로 predefined가 false
		}

		// 2. RoleSub 생성
		roleSubs := []model.RoleSub{
			{
				RoleType: constants.RoleTypeCSP, // CSP 역할은 항상 "csp" 타입
			},
		}

		// 3. RoleMaster와 RoleSubs 함께 생성
		createdRole, err := s.roleService.CreateRoleWithSubs(&roleMaster, roleSubs)
		if err != nil {
			return nil, fmt.Errorf("역할 서브 타입 생성 실패: %w", err)
		}
		roleMasterID = createdRole.ID
	}

	// 2. CSP Role 생성 (AWS IAM에서 역할 존재 여부 확인 및 상태 업데이트)
	cspRole, err := s.cspRoleRepo.CreateCSPRole(req)
	if err != nil {
		log.Printf("Failed to create CSP role: %v", err)
		return cspRole, err
	}

	// 3. RoleMaster와 CSP Role 매핑
	roleMapping := model.RoleMasterCspRoleMapping{
		RoleID:      roleMasterID,
		AuthMethod:  constants.AuthMethodOIDC,
		CspRoleID:   cspRole.ID,
		Description: req.Description,
	}

	// 중복 체크 후 생성
	var existingMapping model.RoleMasterCspRoleMapping
	if err := s.db.Where("role_id = ? AND auth_method = ? AND csp_role_id = ?",
		roleMapping.RoleID, roleMapping.AuthMethod, roleMapping.CspRoleID).
		First(&existingMapping).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 매핑이 존재하지 않으면 생성
			if err := s.db.Create(&roleMapping).Error; err != nil {
				return nil, fmt.Errorf("failed to create role mapping: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to check existing role mapping: %w", err)
		}
	} else {
		// 매핑이 이미 존재하면 로그만 남기고 계속 진행
		log.Printf("Role mapping already exists for role_id=%d, auth_method=%s, csp_role_id=%d",
			roleMapping.RoleID, roleMapping.AuthMethod, roleMapping.CspRoleID)
	}

	return cspRole, nil
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
	role, err := s.cspRoleRepo.GetRole(roleName)
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
