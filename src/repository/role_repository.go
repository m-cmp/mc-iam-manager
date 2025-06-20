package repository

import (
	"fmt"
	"log"

	"github.com/m-cmp/mc-iam-manager/constants"
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
func (r *RoleRepository) FindRoles(req *model.RoleFilterRequest) ([]*model.RoleMaster, error) {
	var roles []*model.RoleMaster

	// 기본 쿼리 생성 - DISTINCT를 사용하여 중복 제거
	query := r.db.Distinct("mcmp_role_masters.*").Preload("RoleSubs")

	// RoleType 필터링이 있는 경우에만 JOIN 추가
	if req.RoleTypes != nil && len(req.RoleTypes) > 0 {
		query = query.Joins("JOIN mcmp_role_subs ON mcmp_role_masters.id = mcmp_role_subs.role_id")
		query = query.Where("mcmp_role_subs.role_type IN (?)", req.RoleTypes)
	}

	if req.RoleID != "" {
		roleID, err := util.StringToUint(req.RoleID)
		if err == nil {
			query = query.Where("mcmp_role_masters.id = ?", roleID)
		}
	}

	if req.RoleName != "" {
		query = query.Where("mcmp_role_masters.name = ?", req.RoleName)
	}

	if err := query.Find(&roles).Error; err != nil {
		return nil, fmt.Errorf("역할 목록 조회 실패: %w", err)
	}

	// RoleType 필터링이 있는 경우, Preload된 RoleSubs에서도 필터링
	if req.RoleTypes != nil && len(req.RoleTypes) > 0 {
		for _, role := range roles {
			filteredRoleSubs := make([]model.RoleSub, 0)
			for _, roleSub := range role.RoleSubs {
				for _, filterType := range req.RoleTypes {
					if roleSub.RoleType == filterType {
						filteredRoleSubs = append(filteredRoleSubs, roleSub)
						break
					}
				}
			}
			role.RoleSubs = filteredRoleSubs
		}
	}

	return roles, nil
}

// GetByID ID로 역할 조회
func (r *RoleRepository) FindRoleByRoleID(roleId uint, roleType constants.IAMRoleType) (*model.RoleMaster, error) {
	var role model.RoleMaster

	// 쿼리 빌더를 사용하여 기본 쿼리 생성
	query := r.db.Preload("RoleSubs").Where("mcmp_role_masters.id = ?", roleId)

	// roleType이 비어있지 않다면 조건 추가
	if roleType != "" {
		query = query.Joins("JOIN mcmp_role_subs ON mcmp_role_masters.id = mcmp_role_subs.role_id").
			Where("mcmp_role_subs.role_type = ?", roleType)
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
func (r *RoleRepository) FindRoleByRoleName(roleName string, roleType constants.IAMRoleType) (*model.RoleMaster, error) {
	var role model.RoleMaster

	// 쿼리 빌더를 사용하여 기본 쿼리 생성
	query := r.db.Preload("RoleSubs").Where("mcmp_role_masters.name = ?", roleName)

	// roleType이 비어있지 않다면 조건 추가
	if roleType != "" {
		query = query.Joins("JOIN mcmp_role_subs ON mcmp_role_masters.id = mcmp_role_subs.role_id").
			Where("mcmp_role_subs.role_type = ?", roleType)
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

// CreateRoleSub RoleSub만 생성 (중복 체크 포함)
func (r *RoleRepository) CreateRoleSub(roleID uint, roleSub *model.RoleSub) error {
	// 1. RoleMaster가 존재하는지 확인
	var roleMaster model.RoleMaster
	if err := r.db.First(&roleMaster, roleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("role master not found with ID: %d", roleID)
		}
		return fmt.Errorf("failed to find role master: %w", err)
	}

	// 2. RoleSub가 이미 존재하는지 확인
	var existingRoleSub model.RoleSub
	if err := r.db.Where("role_id = ? AND role_type = ?", roleID, roleSub.RoleType).First(&existingRoleSub).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// RoleSub가 존재하지 않으면 생성
			roleSub.RoleID = roleID
			if err := r.db.Create(roleSub).Error; err != nil {
				return fmt.Errorf("failed to create role sub: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to check existing role sub: %w", err)
	}

	// RoleSub가 이미 존재하는 경우
	return fmt.Errorf("role sub already exists for role ID %d with type %s", roleID, roleSub.RoleType)
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
		Joins("JOIN mcmp_role_masters ON mcmp_role_masters.id = mcmp_user_workspace_roles.role_id").
		Joins("JOIN mcmp_role_subs ON mcmp_role_masters.id = mcmp_role_subs.role_id").
		Where("mcmp_user_workspace_roles.user_id = ? AND mcmp_role_subs.role_type = ?", userID, constants.RoleTypeWorkspace)

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
		Joins("JOIN mcmp_user_platform_roles ON mcmp_role_masters.id = mcmp_user_platform_roles.role_id").
		Joins("JOIN mcmp_role_subs ON mcmp_role_masters.id = mcmp_role_subs.role_id").
		Where("mcmp_user_platform_roles.user_id = ? AND mcmp_role_subs.role_type = ?", userID, constants.RoleTypePlatform).
		Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// CreateRoleCspRoleMapping 역할-CSP 역할 매핑 생성
func (r *RoleRepository) CreateRoleCspRoleMapping(req *model.RoleMasterCspRoleMappingRequest) error {
	// 문자열 ID를 uint로 변환
	roleIDInt, err := util.StringToUint(req.RoleID)
	if err != nil {
		return fmt.Errorf("잘못된 역할 ID 형식: %w", err)
	}

	cspRoleIDInt, err := util.StringToUint(req.CspRoleID)
	if err != nil {
		return fmt.Errorf("잘못된 CSP 역할 ID 형식: %w", err)
	}

	mapping := &model.RoleMasterCspRoleMapping{
		RoleID:      roleIDInt,
		CspRoleID:   cspRoleIDInt,
		AuthMethod:  req.AuthMethod,
		Description: req.Description,
	}

	return r.db.Create(mapping).Error
}

// DeleteRoleCspRoleMapping 워크스페이스 역할-CSP 역할 매핑 삭제
func (r *RoleRepository) DeleteRoleCspRoleMapping(roleID uint, cspRoleID uint, authMethod constants.AuthMethod) error {
	return r.db.Where("role_id = ? AND csp_role_id = ? AND auth_method = ?", roleID, cspRoleID, authMethod).
		Delete(&model.RoleMasterCspRoleMapping{}).Error
}

// CreateWorkspaceRoleCspRoleMapping 워크스페이스 역할-CSP 역할 매핑 생성
// RoleSub = 'workspace' 가 없으면 생성하고 RoleSub = 'csp' 가 없으면 생성
func (r *RoleRepository) CreateWorkspaceRoleCspRoleMapping(req *model.RoleMasterCspRoleMapping) error {

	workspaceRoleSub := model.RoleSub{
		RoleID:   req.RoleID,
		RoleType: constants.RoleTypeWorkspace,
	}

	cspRoleSub := model.RoleSub{
		RoleID:   req.RoleID,
		RoleType: constants.RoleTypeCSP,
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		// RoleMaster 는 먼저 생성되어 있음.

		// 2. RoleSub에 workspace 생성
		if err := tx.Create(workspaceRoleSub).Error; err != nil {
			return fmt.Errorf("failed to create role sub: %w", err)
		}

		// 3. RoleSub에 csp 생성
		if err := tx.Create(cspRoleSub).Error; err != nil {
			return fmt.Errorf("failed to create role sub: %w", err)
		}

		// 4. RoleMasterCspRoleMapping에 RoleMaster의 ID 설정
		mapping := &model.RoleMasterCspRoleMapping{
			RoleID:     req.RoleID,
			CspRoleID:  cspRoleSub.ID,
			AuthMethod: req.AuthMethod,
		}

		if err := tx.Create(mapping).Error; err != nil {
			return fmt.Errorf("failed to create role mapping: %w", err)
		}

		return nil
	})
}

// DeleteWorkspaceRoleCspRoleMapping 워크스페이스 역할-CSP 역할 매핑 삭제
// RoleSub = 'workspace' 인 경우에만 삭제
func (r *RoleRepository) DeleteWorkspaceRoleCspRoleMapping(workspaceRoleID uint, cspRoleID uint, cspType string) error {
	return r.db.Where("workspace_role_id = ? AND csp_role_id = ? AND csp_type = ?", workspaceRoleID, cspRoleID, cspType).
		Delete(&model.RoleMasterCspRoleMapping{}).Error
}

// RoleMaster와 CSP 역할 매핑 조회 : master Role 1개에 n개의 csp Role이 나온다. csp별로 존재하므로
func (r *RoleRepository) FindRoleMasterCspRoleMappings(req *model.RoleMasterCspRoleMappingRequest) ([]*model.RoleMasterCspRoleMapping, error) {
	var mappings []*model.RoleMasterCspRoleMapping

	// 쿼리 빌더를 사용하여 기본 쿼리 생성
	query := r.db.Preload("CspRole")

	query = query.Where("role_id = ?", req.RoleID) // 필수

	// 현재는 OIDC로 고정. TODO : 선택하는 로직 추가 필요
	query = query.Where("auth_method = ?", constants.AuthMethodOIDC)

	// cspRoleID가 비어있지 않다면 조건 추가
	if req.CspRoleID != "" {
		query = query.Where("csp_role_id = ?", req.CspRoleID)
	}

	if err := query.Find(&mappings).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("RoleMaster -CSP 역할 매핑 조회 실패: %w", err)
	}

	return mappings, nil
}

// WorkspaceRole과 cspRole의 매핑 조회
// cspRole에 매핑된 것들 중 RoleMaster + RoleSub(workspace) 인것만 조회
func (r *RoleRepository) FindWorkspaceRoleCspRoleMappings(req *model.RoleMasterCspRoleMappingRequest) ([]*model.RoleMasterCspRoleMapping, error) {
	var mappings []*model.RoleMasterCspRoleMapping

	// 쿼리 빌더를 사용하여 기본 쿼리 생성
	query := r.db.Preload("CspRole")

	// roleID가 비어있지 않다면 조건 추가
	if req.RoleID != "" {
		query = query.Where("role_id = ?", req.RoleID)
	}

	// cspRoleID가 비어있지 않다면 조건 추가
	if req.CspRoleID != "" {
		query = query.Where("csp_role_id = ?", req.CspRoleID)
	}

	// authMethod 는 OIDC로 고정. TODO : 선택하는 로직 추가 필요
	query = query.Where("auth_method = ?", constants.AuthMethodOIDC)

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
func (r *RoleRepository) IsAssignedRole(userID uint, roleID uint, roleType constants.IAMRoleType) (bool, error) {
	var count int64
	query := r.db.Model(&model.RoleMaster{}).
		Joins("JOIN mcmp_role_subs ON mcmp_role_masters.id = mcmp_role_subs.role_id").
		Joins("JOIN mcmp_user_role ON mcmp_role_masters.id = mcmp_user_role.role_id").
		Where("mcmp_role_masters.id = ? AND mcmp_user_role.user_id = ?", roleID, userID)

	// roleType이 있는 경우에만 조건 추가
	if roleType != "" {
		query = query.Where("mcmp_role_subs.role_type = ?", roleType)
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
		Joins("JOIN mcmp_role_subs ON mcmp_role_masters.id = mcmp_role_subs.role_id").
		Joins("JOIN mcmp_user_platform_roles ON mcmp_role_masters.id = mcmp_user_platform_roles.role_id").
		Where("mcmp_role_masters.id = ? AND mcmp_user_platform_roles.user_id = ?", roleID, userID).
		Where("mcmp_role_subs.role_type = ?", constants.RoleTypePlatform)

	result := query.Count(&count)

	if result.Error != nil {
		return false, fmt.Errorf("P 역할 할당 확인 중 오류 발생: %v", result.Error)
	}

	return count > 0, nil
}

func (r *RoleRepository) IsAssignedWorkspaceRole(userID uint, roleID uint) (bool, error) {
	var count int64
	query := r.db.Model(&model.RoleMaster{}).
		Joins("JOIN mcmp_role_subs ON mcmp_role_masters.id = mcmp_role_subs.role_id").
		Joins("JOIN mcmp_workspace_user_roles ON mcmp_role_masters.id = mcmp_workspace_user_rolses.role_id").
		Where("mcmp_role_masters.id = ? AND mcmp_workspace_user_roless.user_id = ?", roleID, userID).
		Where("mcmp_role_subs.role_type = ?", constants.RoleTypeWorkspace)

	result := query.Count(&count)

	if result.Error != nil {
		return false, fmt.Errorf("W 역할 할당 확인 중 오류 발생: %v", result.Error)
	}

	return count > 0, nil
}

// GetWorkspaceRoles 워크스페이스의 모든 역할 목록 조회 : TODO : 목록조회면 ListWorkspaceRoleById 가 맞을 듯.
func (r *RoleRepository) GetWorkspaceRoles(workspaceID uint) ([]model.RoleMaster, error) {
	var roles []model.RoleMaster
	if err := r.db.Preload("RoleSubs").
		Joins("JOIN mcmp_role_subs ON mcmp_role_masters.id = mcmp_role_subs.role_id").
		Where("mcmp_role_subs.role_type = ?", constants.RoleTypeWorkspace).
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// GetUserWorkspaceRoles 사용자의 워크스페이스 역할 목록 조회 (기존 FindUserWorkspaceRoles와 유사하지만 RoleMaster 반환)
// TODO : 목록조회면 ListUserWorkspaceRoles 가 맞을 듯.( req 객체로 조회조건 받도록 수정필요)
func (r *RoleRepository) GetUserWorkspaceRoles(userID, workspaceID uint) ([]model.RoleMaster, error) {
	var roles []model.RoleMaster
	if err := r.db.Preload("RoleSubs").
		Joins("JOIN mcmp_user_workspace_roles ON mcmp_role_masters.id = mcmp_user_workspace_roles.role_id").
		Joins("JOIN mcmp_role_subs ON mcmp_role_masters.id = mcmp_role_subs.role_id").
		Where("mcmp_user_workspace_roles.user_id = ? AND mcmp_user_workspace_roles.workspace_id = ? AND mcmp_role_subs.role_type = ?",
			userID, workspaceID, constants.RoleTypeWorkspace).
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// FindCspRoleMappingsByWorkspaceRoleIDAndCspType 역할 ID와 CSP 타입으로 CSP 역할 매핑 조회
func (r *RoleRepository) FindCspRoleMappings(req *model.RoleMappingRequest) ([]*model.RoleMasterCspRoleMapping, error) {
	var mappings []*model.RoleMasterCspRoleMapping
	err := r.db.
		Preload("CspRole").
		Where("role_id = ? AND csp_type = ?", req.RoleID, req.CspType).
		Find(&mappings).Error
	if err != nil {
		return nil, err
	}
	return mappings, err
}

// FindCspRoleMappingsByWorkspaceRoleIDAndCspType 역할 ID와 CSP 타입으로 CSP 역할 매핑 조회
func (r *RoleRepository) FindCspRoleById(cspRoleId uint) (*model.CspRole, error) {
	var cspRole *model.CspRole
	err := r.db.
		Where("id = ?", cspRoleId).
		First(&cspRole).Error
	if err != nil {
		return nil, err
	}
	return cspRole, err
}

// CreateRoleWithSubs RoleMaster와 RoleSubs를 트랜잭션으로 함께 생성
func (r *RoleRepository) CreateRoleWithSubs(role *model.RoleMaster, roleSubs []model.RoleSub) (*model.RoleMaster, error) {
	var result *model.RoleMaster
	err := r.db.Transaction(func(tx *gorm.DB) error {
		// 1. RoleMaster가 이미 존재하는지 확인
		existingRole, err := r.FindRoleByRoleName(role.Name, constants.IAMRoleType(""))
		if err != nil && err.Error() != "role not found" {
			return fmt.Errorf("역할 조회 실패: %w", err)
		}

		if existingRole != nil {
			// 이미 존재하는 역할이면 해당 역할을 사용
			role.ID = existingRole.ID
		} else {
			// 새로운 역할 생성
			if err := tx.Create(role).Error; err != nil {
				return fmt.Errorf("역할 생성 실패: %w", err)
			}
		}

		// 2. RoleSub 생성
		for _, sub := range roleSubs {
			sub.RoleID = role.ID // RoleMaster의 ID 설정

			// RoleSub가 이미 존재하는지 확인
			var existingSub model.RoleSub
			if err := tx.Where("role_id = ? AND role_type = ?", role.ID, sub.RoleType).First(&existingSub).Error; err != nil {
				if err != gorm.ErrRecordNotFound {
					return fmt.Errorf("역할 서브 타입 조회 실패: %w", err)
				}
				// RoleSub가 존재하지 않는 경우에만 생성
				if err := tx.Create(&sub).Error; err != nil {
					return fmt.Errorf("역할 서브 타입 생성 실패: %w", err)
				}
			} else {
				// RoleSub가 이미 존재하는 경우 로그만 남기고 계속 진행
				log.Printf("역할 서브 타입 (RoleID: %d, Type: %s)가 이미 존재합니다. 건너뜁니다.", role.ID, sub.RoleType)
			}
		}

		// 3. 생성된 RoleMaster와 RoleSubs를 함께 조회
		if err := tx.Preload("RoleSubs").First(role, role.ID).Error; err != nil {
			return fmt.Errorf("생성된 역할 조회 실패: %w", err)
		}

		result = role
		return nil
	})

	return result, err
}
