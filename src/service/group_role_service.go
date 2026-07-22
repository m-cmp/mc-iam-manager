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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return repository.ErrRoleMasterNotFound
		}
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

// GetAvailablePlatformRoles 그룹에 할당되지 않은 플랫폼 역할 목록 조회
func (s *GroupRoleService) GetAvailablePlatformRoles(groupID uint) ([]model.RoleMaster, error) {
	return s.groupRoleRepo.FindAvailablePlatformRoles(groupID)
}

// ListGroupsByPlatformRole 특정 platform role이 부여된 그룹 목록 조회 (역할→그룹 역방향 조회)
func (s *GroupRoleService) ListGroupsByPlatformRole(roleID uint) ([]model.GroupPlatformRoleResponse, error) {
	return s.groupRoleRepo.FindGroupsByPlatformRoleID(roleID)
}

// ListGroupsByWorkspaceRole 특정 workspace role이 부여된 그룹 목록 조회 (역할→그룹 역방향 조회)
func (s *GroupRoleService) ListGroupsByWorkspaceRole(roleID uint) ([]model.GroupWorkspaceRoleResponse, error) {
	return s.groupRoleRepo.FindGroupsByWorkspaceRoleID(roleID)
}

// GetAvailableWorkspaces 그룹에 매핑되지 않은 워크스페이스 목록 조회
func (s *GroupRoleService) GetAvailableWorkspaces(groupID uint) ([]model.Workspace, error) {
	return s.groupRoleRepo.FindAvailableWorkspaces(groupID)
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return repository.ErrRoleMasterNotFound
		}
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
// workspace_id, role_id 존재 여부 pre-validation 포함
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

// AssignUsersToGroup 그룹에 사용자 일괄 할당 (그룹 입장, DB + Keycloak 동기화)
func (s *GroupRoleService) AssignUsersToGroup(ctx context.Context, groupID uint, userIDs []uint) error {
	// 그룹 존재 확인
	org, err := s.orgRepo.FindByID(groupID)
	if err != nil {
		return err
	}

	for _, userID := range userIDs {
		// 사용자 KC ID 조회
		var user model.User
		if err := s.db.First(&user, userID).Error; err != nil {
			return fmt.Errorf("user not found: %d", userID)
		}

		// DB 저장
		if err := s.orgRepo.AssignUserToOrganizations(userID, []uint{groupID}); err != nil {
			return fmt.Errorf("failed to assign user %d to group in DB: %w", userID, err)
		}

		// Keycloak 동기화
		if user.KcId != "" {
			if err := s.kcService.EnsureGroupExistsAndAssignUser(ctx, user.KcId, org.Name); err != nil {
				return fmt.Errorf("failed to assign user %d to keycloak group '%s': %w", userID, org.Name, err)
			}
		}
	}
	return nil
}

// RemoveUsersFromGroup 그룹에서 사용자 일괄 제거 (그룹 입장, DB + Keycloak 동기화)
func (s *GroupRoleService) RemoveUsersFromGroup(ctx context.Context, groupID uint, userIDs []uint) error {
	// 그룹 존재 확인
	org, err := s.orgRepo.FindByID(groupID)
	if err != nil {
		return err
	}

	for _, userID := range userIDs {
		// 사용자 KC ID 조회
		var user model.User
		if err := s.db.First(&user, userID).Error; err != nil {
			return fmt.Errorf("user not found: %d", userID)
		}

		// DB 삭제
		if err := s.orgRepo.RemoveUserFromOrganization(userID, groupID); err != nil {
			return fmt.Errorf("failed to remove user %d from group in DB: %w", userID, err)
		}

		// Keycloak 동기화
		if user.KcId != "" {
			if err := s.kcService.RemoveUserFromGroup(ctx, user.KcId, org.Name); err != nil {
				return fmt.Errorf("keycloak group removal failed for user %d (DB already updated): %w", userID, err)
			}
		}
	}
	return nil
}

// GetEffectivePlatformRoles 사용자의 유효 플랫폼 역할 목록 조회 (직접 + 그룹 상속, 중복 제거)
func (s *GroupRoleService) GetEffectivePlatformRoles(userID uint) ([]model.EffectivePlatformRoleItem, error) {
	return s.groupRoleRepo.FindEffectivePlatformRolesByUserID(userID)
}

// GetUserAccessSummary 사용자 접근 권한 요약 조회 (직접 역할 + 그룹 + 그룹 기반 역할)
func (s *GroupRoleService) GetUserAccessSummary(userID uint) (*model.UserAccessSummaryResponse, error) {
	directRoles, err := s.groupRoleRepo.FindDirectPlatformRolesByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get direct roles: %w", err)
	}

	groups, err := s.groupRoleRepo.FindUserGroupsWithRoles(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups with roles: %w", err)
	}

	return &model.UserAccessSummaryResponse{
		UserID:      userID,
		DirectRoles: directRoles,
		Groups:      groups,
	}, nil
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
