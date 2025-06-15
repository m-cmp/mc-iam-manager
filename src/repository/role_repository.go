package repository

import (
	"fmt"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/util"
	"gorm.io/gorm"
)

// RoleRepository 역할 관리 레포지토리
type RoleRepository struct {
	db *gorm.DB
}

// NewRoleRepository 새 RoleRepository 인스턴스 생성
func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

// List 모든 역할 목록 조회
func (r *RoleRepository) FindRoles(roleID uint, roleType string) ([]*model.RoleMaster, error) {
	var roles []*model.RoleMaster
	query := r.db.Preload("RoleSubs")
	query = query.Joins("JOIN mcmp_role_sub ON mcmp_role_master.id = mcmp_role_sub.role_id")

	if roleID != 0 {
		query = query.Where("id = ?", roleID)
	}

	if roleType != "" {
		query = query.Where("mcmp_role_sub.role_type = ?", roleType)
	}

	if err := query.Find(&roles).Error; err != nil {
		return nil, fmt.Errorf("역할 목록 조회 실패: %w", err)
	}
	return roles, nil
}

// GetByID ID로 역할 조회
func (r *RoleRepository) FindRoleByRoleID(roleId uint, roleType string) (*model.RoleMaster, error) {
	var role model.RoleMaster

	// 쿼리 빌더를 사용하여 기본 쿼리 생성
	query := r.db.Preload("RoleSubs").Where("mcmp_role_master.id = ?", roleId)

	// roleType이 비어있지 않다면 조건 추가
	if roleType != "" {
		query = query.Joins("JOIN mcmp_role_sub ON mcmp_role_master.id = mcmp_role_sub.role_id").
			Where("mcmp_role_sub.role_type = ?", roleType)
	}

	if err := query.First(&role).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("역할 조회 실패: %w", err)
	}
	return &role, nil
}

// GetByName Name으로 역할 조회
func (r *RoleRepository) FindRoleByRoleName(roleName string, roleType string) (*model.RoleMaster, error) {
	var role model.RoleMaster

	// 쿼리 빌더를 사용하여 기본 쿼리 생성
	query := r.db.Preload("RoleSubs").Where("mcmp_role_master.name = ?", roleName)

	// roleType이 비어있지 않다면 조건 추가
	if roleType != "" {
		query = query.Joins("JOIN mcmp_role_sub ON mcmp_role_master.id = mcmp_role_sub.role_id").
			Where("mcmp_role_sub.role_type = ?", roleType)
	}

	if err := query.First(&role).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("역할 조회 실패: %w", err)
	}
	return &role, nil
}

// Create 역할 생성
func (r *RoleRepository) CreateRole(role *model.RoleMaster) error {
	return r.db.Create(role).Error
}

// Update 역할 수정
func (r *RoleRepository) UpdateRole(role *model.RoleMaster) error {
	return r.db.Save(role).Error
}

// Delete 역할 삭제
func (r *RoleRepository) DeleteRole(id uint) error {
	return r.db.Delete(&model.RoleMaster{}, id).Error
}

// CreateRoleSub 역할 서브 타입 생성
func (r *RoleRepository) CreateRoleSub(roleSub *model.RoleSub) error {
	return r.db.Create(roleSub).Error
}

// DeleteRoleSubs 역할 서브 타입들 삭제
func (r *RoleRepository) DeleteRoleSub(roleID uint) error {
	return r.db.Where("role_id = ?", roleID).Delete(&model.RoleSub{}).Error
}

// AssignPlatformRole 플랫폼 역할 할당
func (r *RoleRepository) AssignPlatformRole(userID, roleID uint) error {
	userRole := model.UserPlatformRole{
		UserID: userID,
		RoleID: roleID,
	}
	return r.db.Create(&userRole).Error
}

// RemovePlatformRole 플랫폼 역할 제거
func (r *RoleRepository) RemovePlatformRole(userID, roleID uint) error {
	return r.db.Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&model.UserPlatformRole{}).Error
}

// AssignWorkspaceRole 워크스페이스 역할 할당
func (r *RoleRepository) AssignWorkspaceRole(userID, workspaceID, roleID uint) error {
	userWorkspaceRole := model.UserWorkspaceRole{
		UserID:      userID,
		WorkspaceID: workspaceID,
		RoleID:      roleID,
	}
	return r.db.Create(&userWorkspaceRole).Error
}

// RemoveWorkspaceRole 워크스페이스 역할 제거
func (r *RoleRepository) RemoveWorkspaceRole(userID, workspaceID, roleID uint) error {
	return r.db.Where("user_id = ? AND workspace_id = ? AND role_id = ?", userID, workspaceID, roleID).
		Delete(&model.UserWorkspaceRole{}).Error
}

// GetUserWorkspaceRoles 사용자의 워크스페이스 역할 목록 조회
func (r *RoleRepository) FindUserWorkspaceRoles(userID, workspaceID uint) ([]model.UserWorkspaceRole, error) {
	var roles []model.UserWorkspaceRole
	query := r.db.
		Joins("JOIN mcmp_user_workspace_roles ON mcmp_role_master.id = mcmp_user_workspace_roles.role_id").
		Joins("JOIN mcmp_role_sub ON mcmp_role_master.id = mcmp_role_sub.role_id").
		Where("mcmp_user_workspace_roles.user_id = ? AND mcmp_role_sub.role_type = ?", userID, model.RoleTypeWorkspace)

	if workspaceID != 0 {
		query = query.Where("mcmp_user_workspace_roles.workspace_id = ?", workspaceID)
	}

	err := query.Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// GetUserPlatformRoles 사용자의 플랫폼 역할 목록 조회
func (r *RoleRepository) FindUserPlatformRoles(userID uint) ([]model.RoleMaster, error) {
	var roles []model.RoleMaster
	err := r.db.
		Joins("JOIN mcmp_user_platform_roles ON mcmp_role_master.id = mcmp_user_platform_roles.role_id").
		Joins("JOIN mcmp_role_sub ON mcmp_role_master.id = mcmp_role_sub.role_id").
		Where("mcmp_user_platform_roles.user_id = ? AND mcmp_role_sub.role_type = ?", userID, model.RoleTypePlatform).
		Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// CreateWorkspaceRoleCspRoleMapping 워크스페이스 역할-CSP 역할 매핑 생성
func (r *RoleRepository) CreateRoleMasterCspRoleMapping(mapping *model.RoleMasterCspRoleMapping) error {
	return r.db.Create(mapping).Error
}

// DeleteWorkspaceRoleCspRoleMapping 워크스페이스 역할-CSP 역할 매핑 삭제
func (r *RoleRepository) DeleteRoleCspRoleMapping(roleID uint, cspRoleID uint, cspType string) error {
	return r.db.Where("role_id = ? AND csp_role_id = ? AND csp_type = ?", roleID, cspRoleID, cspType).
		Delete(&model.RoleMasterCspRoleMapping{}).Error
}

// CreateWorkspaceRoleCspRoleMapping 워크스페이스 역할-CSP 역할 매핑 생성
func (r *RoleRepository) CreateWorkspaceRoleCspRoleMapping(mapping *model.RoleMasterCspRoleMapping) error {
	return r.db.Create(mapping).Error
}

// DeleteWorkspaceRoleCspRoleMapping 워크스페이스 역할-CSP 역할 매핑 삭제
func (r *RoleRepository) DeleteWorkspaceRoleCspRoleMapping(workspaceRoleID uint, cspRoleID uint, cspType string) error {
	return r.db.Where("workspace_role_id = ? AND csp_role_id = ? AND csp_type = ?", workspaceRoleID, cspRoleID, cspType).
		Delete(&model.RoleMasterCspRoleMapping{}).Error
}

// RoleMaster와 CSP 역할 매핑 조회
func (r *RoleRepository) FindRoleMasterCspRoleMappings(roleID uint, cspRoleID uint, cspType string) ([]*model.RoleMasterCspRoleMapping, error) {
	var mappings []*model.RoleMasterCspRoleMapping

	// 쿼리 빌더를 사용하여 기본 쿼리 생성
	query := r.db.Preload("RoleMasterCspRoleMapping")

	// workspaceRoleID가 비어있지 않다면 조건 추가
	if roleID != 0 {
		query = query.Where("role_id = ?", roleID)
	}

	// cspRoleID가 비어있지 않다면 조건 추가
	if cspRoleID != 0 {
		query = query.Where("csp_role_id = ?", cspRoleID)
	}

	// cspType이 비어있지 않다면 조건 추가
	if cspType != "" {
		query = query.Where("csp_type = ?", cspType)
	}

	if err := query.Find(&mappings).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("RoleMaster -CSP 역할 매핑 조회 실패: %w", err)
	}

	return mappings, nil
}

func (r *RoleRepository) FindWorkspaceRoleCspRoleMappings(workspaceRoleID uint, cspRoleID uint, cspType string) ([]*model.RoleMasterCspRoleMapping, error) {
	var mappings []*model.RoleMasterCspRoleMapping

	// 쿼리 빌더를 사용하여 기본 쿼리 생성
	query := r.db.Preload("WorkspaceRoleCspRoleMapping")

	// workspaceRoleID가 비어있지 않다면 조건 추가
	if workspaceRoleID != 0 {
		query = query.Where("workspace_role_id = ?", workspaceRoleID)
	}

	// cspRoleID가 비어있지 않다면 조건 추가
	if cspRoleID != 0 {
		query = query.Where("csp_role_id = ?", cspRoleID)
	}

	// cspType이 비어있지 않다면 조건 추가
	if cspType != "" {
		query = query.Where("csp_type = ?", cspType)
	}

	if err := query.Find(&mappings).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("워크스페이스 역할-CSP 역할 매핑 조회 실패: %w", err)
	}

	return mappings, nil
}

// FindUsersAndRolesByWorkspaceID 특정 워크스페이스에 속한 사용자 및 역할 목록 조회 : user 기준
func (r *RoleRepository) FindUsersAndRolesWithWorkspaces(req model.WorkspaceFilterRequest) ([]*model.UserWorkspaceRole, error) {
	var userWorkspaceRoles []*model.UserWorkspaceRole

	query := r.db.Joins("JOIN mcmp_workspace_roles ON mcmp_workspace_roles.id = mcmp_user_workspace_roles.workspace_role_id")

	if req.WorkspaceID != "" {
		workspaceIdInt, err := util.StringToUint(req.WorkspaceID)
		if err != nil {
			return nil, err
		}
		query = query.Where("mcmp_user_workspace_roles.workspace_id = ?", workspaceIdInt)
	}

	if req.RoleID != "" {
		roleIdInt, err := util.StringToUint(req.RoleID)
		if err != nil {
			return nil, err
		}
		query = query.Where("mcmp_user_workspace_roles.role_id = ?", roleIdInt)
	}

	if req.UserID != "" {
		userIdInt, err := util.StringToUint(req.UserID)
		if err != nil {
			return nil, err
		}
		query = query.Where("mcmp_user_workspace_roles.user_id = ?", userIdInt)
	}

	// Find all UserWorkspaceRole entries where the associated WorkspaceRole's WorkspaceID matches.
	// We need to join with WorkspaceRole table to filter by workspaceID.
	// Then preload User and WorkspaceRole.
	err := query.Preload("User").
		Preload("WorkspaceRole").
		Find(&userWorkspaceRoles).Error

	if err != nil {
		// Don't return ErrWorkspaceNotFound here, as an empty result is valid.
		// Return other DB errors.
		return nil, err
	}

	return userWorkspaceRoles, nil
}

// FindWorkspaceWithUsersRoles 특정 워크스페이스에 속한 사용자 및 역할 목록 조회 : workspace 기준
func (r *RoleRepository) FindWorkspaceWithUsersRoles(req model.WorkspaceFilterRequest) ([]*model.WorkspaceWithUsersAndRoles, error) {
	var userWorkspaceRoles []*model.WorkspaceWithUsersAndRoles

	query := r.db.Joins("JOIN mcmp_workspace_roles ON mcmp_workspace_roles.id = mcmp_user_workspace_roles.workspace_role_id")

	if req.WorkspaceID != "" {
		workspaceIdInt, err := util.StringToUint(req.WorkspaceID)
		if err != nil {
			return nil, err
		}
		query = query.Where("mcmp_user_workspace_roles.workspace_id = ?", workspaceIdInt)
	}

	if req.RoleID != "" {
		roleIdInt, err := util.StringToUint(req.RoleID)
		if err != nil {
			return nil, err
		}
		query = query.Where("mcmp_user_workspace_roles.role_id = ?", roleIdInt)
	}

	if req.UserID != "" {
		userIdInt, err := util.StringToUint(req.UserID)
		if err != nil {
			return nil, err
		}
		query = query.Where("mcmp_user_workspace_roles.user_id = ?", userIdInt)
	}

	// Find all UserWorkspaceRole entries where the associated WorkspaceRole's WorkspaceID matches.
	// We need to join with WorkspaceRole table to filter by workspaceID.
	// Then preload User and WorkspaceRole.
	err := query.Preload("User").
		Preload("WorkspaceRole").
		Find(&userWorkspaceRoles).Error

	if err != nil {
		// Don't return ErrWorkspaceNotFound here, as an empty result is valid.
		// Return other DB errors.
		return nil, err
	}

	return userWorkspaceRoles, nil
}

// IsAssignedPlatformRole 사용자에게 특정 플랫폼 역할이 할당되어 있는지 확인
func (r *RoleRepository) IsAssignedRole(userID uint, roleID uint, roleType string) (bool, error) {
	var count int64
	query := r.db.Model(&model.RoleMaster{}).
		Joins("JOIN mcmp_role_sub ON mcmp_role_master.id = mcmp_role_sub.role_id").
		Joins("JOIN mcmp_user_role ON mcmp_role_master.id = mcmp_user_role.role_id").
		Where("mcmp_role_master.id = ? AND mcmp_user_role.user_id = ?", roleID, userID)

	// roleType이 있는 경우에만 조건 추가
	if roleType != "" {
		query = query.Where("mcmp_role_sub.role_type = ?", roleType)
	}

	result := query.Count(&count)

	if result.Error != nil {
		return false, fmt.Errorf("역할 할당 확인 중 오류 발생: %v", result.Error)
	}

	return count > 0, nil
}

func (r *RoleRepository) IsAssignedPlatformRole(userID uint, roleID uint) (bool, error) {
	var count int64
	query := r.db.Model(&model.RoleMaster{}).
		Joins("JOIN mcmp_role_sub ON mcmp_role_master.id = mcmp_role_sub.role_id").
		Joins("JOIN mcmp_user_platform_roles ON mcmp_role_master.id = mcmp_user_platform_roles.role_id").
		Where("mcmp_role_master.id = ? AND mcmp_user_platform_roles.user_id = ?", roleID, userID).
		Where("mcmp_role_sub.role_type = ?", model.RoleTypePlatform)

	result := query.Count(&count)

	if result.Error != nil {
		return false, fmt.Errorf("P 역할 할당 확인 중 오류 발생: %v", result.Error)
	}

	return count > 0, nil
}

func (r *RoleRepository) IsAssignedWorkspaceRole(userID uint, roleID uint) (bool, error) {
	var count int64
	query := r.db.Model(&model.RoleMaster{}).
		Joins("JOIN mcmp_role_sub ON mcmp_role_master.id = mcmp_role_sub.role_id").
		Joins("JOIN mcmp_workspace_user_roles ON mcmp_role_master.id = mcmp_workspace_user_rolses.role_id").
		Where("mcmp_role_master.id = ? AND mcmp_workspace_user_roless.user_id = ?", roleID, userID).
		Where("mcmp_role_sub.role_type = ?", model.RoleTypeWorkspace)

	result := query.Count(&count)

	if result.Error != nil {
		return false, fmt.Errorf("W 역할 할당 확인 중 오류 발생: %v", result.Error)
	}

	return count > 0, nil
}
