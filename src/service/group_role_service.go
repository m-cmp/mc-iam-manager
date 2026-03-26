package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm"
)

// GroupRoleService 그룹 역할 관리 서비스
type GroupRoleService struct {
	db            *gorm.DB
	groupRoleRepo *repository.GroupRoleRepository
	orgRepo       *repository.OrganizationRepository
	kcService     KeycloakService
}

// NewGroupRoleService GroupRoleService 생성자
func NewGroupRoleService(db *gorm.DB) *GroupRoleService {
	return &GroupRoleService{
		db:            db,
		groupRoleRepo: repository.NewGroupRoleRepository(db),
		orgRepo:       repository.NewOrganizationRepository(db),
		kcService:     NewKeycloakService(),
	}
}

// --- Platform Role ---

// AssignGroupPlatformRole 그룹에 platform role 할당 (DB + Keycloak)
func (s *GroupRoleService) AssignGroupPlatformRole(ctx context.Context, groupID, roleID uint) error {
	// 1. 그룹 조회 (KC 그룹 이름으로 사용)
	org, err := s.orgRepo.FindByID(groupID)
	if err != nil {
		return err
	}

	// 2. Role 이름 조회
	var roleMaster model.RoleMaster
	if err := s.db.First(&roleMaster, roleID).Error; err != nil {
		return fmt.Errorf("role not found: %w", err)
	}

	// 3. DB에 저장
	if err := s.groupRoleRepo.CreateGroupPlatformRole(groupID, roleID); err != nil {
		return err
	}

	// 4. Keycloak: 그룹에 realm role 추가
	if err := s.kcService.AddRealmRoleToGroup(ctx, org.Name, roleMaster.Name); err != nil {
		// DB rollback
		_ = s.groupRoleRepo.DeleteGroupPlatformRole(groupID, roleID)
		return fmt.Errorf("failed to assign role to keycloak group: %w", err)
	}

	return nil
}

// GetGroupPlatformRoles 그룹의 platform role 목록 조회
func (s *GroupRoleService) GetGroupPlatformRoles(groupID uint) ([]model.GroupPlatformRoleResponse, error) {
	return s.groupRoleRepo.FindGroupPlatformRoles(groupID)
}

// RemoveGroupPlatformRole 그룹에서 platform role 해제 (DB + Keycloak)
func (s *GroupRoleService) RemoveGroupPlatformRole(ctx context.Context, groupID, roleID uint) error {
	// 1. 그룹 조회
	org, err := s.orgRepo.FindByID(groupID)
	if err != nil {
		return err
	}

	// 2. Role 이름 조회
	var roleMaster model.RoleMaster
	if err := s.db.First(&roleMaster, roleID).Error; err != nil {
		return fmt.Errorf("role not found: %w", err)
	}

	// 3. DB에서 삭제
	if err := s.groupRoleRepo.DeleteGroupPlatformRole(groupID, roleID); err != nil {
		return err
	}

	// 4. Keycloak에서 제거
	if err := s.kcService.RemoveRealmRoleFromGroup(ctx, org.Name, roleMaster.Name); err != nil {
		return fmt.Errorf("keycloak role removal failed (DB already updated): %w", err)
	}

	return nil
}

// --- Workspace Role ---

// AssignGroupWorkspace 그룹-워크스페이스 매핑 생성 (DB 전용)
func (s *GroupRoleService) AssignGroupWorkspace(groupID, workspaceID, roleID uint) error {
	// workspace 존재 여부 확인
	var workspace model.Workspace
	if err := s.db.First(&workspace, workspaceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return repository.ErrWorkspaceNotFound
		}
		return fmt.Errorf("error checking workspace: %w", err)
	}
	// role 존재 여부 확인
	var roleMaster model.RoleMaster
	if err := s.db.First(&roleMaster, roleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return repository.ErrRoleMasterNotFound
		}
		return fmt.Errorf("error checking role: %w", err)
	}
	return s.groupRoleRepo.CreateGroupWorkspaceRole(groupID, workspaceID, roleID)
}

// GetGroupWorkspaces 그룹의 워크스페이스 매핑 목록 조회
func (s *GroupRoleService) GetGroupWorkspaces(groupID uint) ([]model.GroupWorkspaceRoleResponse, error) {
	return s.groupRoleRepo.FindGroupWorkspaceRoles(groupID)
}

// UpdateGroupWorkspaceRole 그룹-워크스페이스 역할 변경
func (s *GroupRoleService) UpdateGroupWorkspaceRole(groupID, workspaceID, roleID uint) error {
	return s.groupRoleRepo.UpdateGroupWorkspaceRole(groupID, workspaceID, roleID)
}

// RemoveGroupWorkspaceRole 그룹-워크스페이스 매핑 제거
func (s *GroupRoleService) RemoveGroupWorkspaceRole(groupID, workspaceID uint) error {
	return s.groupRoleRepo.DeleteGroupWorkspaceRole(groupID, workspaceID)
}

// GetAvailableGroupWorkspaces 그룹에 미매핑된 워크스페이스 목록 조회
func (s *GroupRoleService) GetAvailableGroupWorkspaces(groupID uint) ([]*model.Workspace, error) {
	return s.groupRoleRepo.FindAvailableWorkspacesForGroup(groupID)
}

// --- User-Group (with Keycloak sync) ---

// AssignUserToGroups 사용자를 그룹에 할당 (DB + Keycloak 동기화)
func (s *GroupRoleService) AssignUserToGroups(ctx context.Context, userID uint, groupIDs []uint, kcUserID string) error {
	for _, groupID := range groupIDs {
		// 그룹 조회
		org, err := s.orgRepo.FindByID(groupID)
		if err != nil {
			return fmt.Errorf("group not found: %d", groupID)
		}

		// DB 저장 (이미 소속이면 skip - FirstOrCreate 패턴은 repository에서)
		if err := s.orgRepo.AssignUserToOrganizations(userID, []uint{groupID}); err != nil {
			return fmt.Errorf("failed to assign user to group %d in DB: %w", groupID, err)
		}

		// Keycloak 그룹 동기화
		if kcUserID != "" {
			if err := s.kcService.EnsureGroupExistsAndAssignUser(ctx, kcUserID, org.Name); err != nil {
				return fmt.Errorf("failed to assign user to keycloak group '%s': %w", org.Name, err)
			}
		}
	}
	return nil
}

// RemoveUserFromGroup 사용자를 그룹에서 제거 (DB + Keycloak 동기화)
func (s *GroupRoleService) RemoveUserFromGroup(ctx context.Context, userID, groupID uint, kcUserID string) error {
	// 그룹 조회
	org, err := s.orgRepo.FindByID(groupID)
	if err != nil {
		return err
	}

	// DB 삭제
	if err := s.orgRepo.RemoveUserFromOrganization(userID, groupID); err != nil {
		return err
	}

	// Keycloak 그룹에서 제거
	if kcUserID != "" {
		if err := s.kcService.RemoveUserFromGroup(ctx, kcUserID, org.Name); err != nil {
			return fmt.Errorf("keycloak group removal failed (DB already updated): %w", err)
		}
	}
	return nil
}
