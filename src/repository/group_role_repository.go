package repository

import (
	"errors"
	"fmt"
	"strings"

	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

var (
	ErrGroupPlatformRoleNotFound   = errors.New("group platform role mapping not found")
	ErrGroupPlatformRoleDuplicate  = errors.New("group platform role mapping already exists")
	ErrGroupWorkspaceRoleNotFound  = errors.New("group workspace role mapping not found")
	ErrGroupWorkspaceRoleDuplicate = errors.New("group workspace role mapping already exists")
	ErrRoleMasterNotFound          = errors.New("role not found")
)

// GroupRoleRepository 그룹 역할 매핑 데이터 관리
type GroupRoleRepository struct {
	db *gorm.DB
}

// NewGroupRoleRepository GroupRoleRepository 생성자
func NewGroupRoleRepository(db *gorm.DB) *GroupRoleRepository {
	return &GroupRoleRepository{db: db}
}

// --- GroupPlatformRole ---

// CreateGroupPlatformRole 그룹-플랫폼 역할 매핑 생성
func (r *GroupRoleRepository) CreateGroupPlatformRole(groupID, roleID uint) error {
	record := &model.GroupPlatformRole{
		GroupID: groupID,
		RoleID:  roleID,
	}
	if err := r.db.Create(record).Error; err != nil {
		if isGroupDuplicateError(err) {
			return ErrGroupPlatformRoleDuplicate
		}
		return fmt.Errorf("error creating group platform role: %w", err)
	}
	return nil
}

// FindGroupPlatformRoles 그룹의 플랫폼 역할 목록 조회
func (r *GroupRoleRepository) FindGroupPlatformRoles(groupID uint) ([]model.GroupPlatformRoleResponse, error) {
	results := make([]model.GroupPlatformRoleResponse, 0)
	err := r.db.Table("mcmp_group_platform_roles gpr").
		Select("gpr.group_id, o.name as group_name, gpr.role_id, rm.name as role_name, gpr.created_at").
		Joins("JOIN mcmp_organizations o ON o.id = gpr.group_id").
		Joins("JOIN mcmp_role_masters rm ON rm.id = gpr.role_id").
		Where("gpr.group_id = ?", groupID).
		Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("error finding group platform roles: %w", err)
	}
	return results, nil
}

// FindGroupPlatformRoleByRoleID 특정 그룹-역할 매핑 조회
func (r *GroupRoleRepository) FindGroupPlatformRoleByRoleID(groupID, roleID uint) (*model.GroupPlatformRole, error) {
	var record model.GroupPlatformRole
	if err := r.db.Where("group_id = ? AND role_id = ?", groupID, roleID).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGroupPlatformRoleNotFound
		}
		return nil, err
	}
	return &record, nil
}

// FindAvailableGroupPlatformRoles 그룹에 미할당된 플랫폼 역할 목록 조회
func (r *GroupRoleRepository) FindAvailableGroupPlatformRoles(groupID uint) ([]model.AvailablePlatformRoleResponse, error) {
	results := make([]model.AvailablePlatformRoleResponse, 0)
	err := r.db.Table("mcmp_role_masters rm").
		Select("rm.id as role_id, rm.name as role_name, rm.description").
		Where("rm.role_type = 'platform'").
		Where("rm.id NOT IN (?)",
			r.db.Table("mcmp_group_platform_roles").
				Select("role_id").
				Where("group_id = ?", groupID),
		).
		Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("error finding available platform roles: %w", err)
	}
	return results, nil
}

// DeleteGroupPlatformRole 그룹-플랫폼 역할 매핑 삭제
func (r *GroupRoleRepository) DeleteGroupPlatformRole(groupID, roleID uint) error {
	result := r.db.Where("group_id = ? AND role_id = ?", groupID, roleID).Delete(&model.GroupPlatformRole{})
	if result.Error != nil {
		return fmt.Errorf("error deleting group platform role: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrGroupPlatformRoleNotFound
	}
	return nil
}

// --- GroupWorkspaceRole ---

// CreateGroupWorkspaceRole 그룹-워크스페이스-역할 매핑 생성
func (r *GroupRoleRepository) CreateGroupWorkspaceRole(groupID, workspaceID, roleID uint) error {
	record := &model.GroupWorkspaceRole{
		GroupID:     groupID,
		WorkspaceID: workspaceID,
		RoleID:      roleID,
	}
	if err := r.db.Create(record).Error; err != nil {
		if isGroupDuplicateError(err) {
			return ErrGroupWorkspaceRoleDuplicate
		}
		return fmt.Errorf("error creating group workspace role: %w", err)
	}
	return nil
}

// FindGroupWorkspaceRoles 그룹의 워크스페이스 매핑 목록 조회
func (r *GroupRoleRepository) FindGroupWorkspaceRoles(groupID uint) ([]model.GroupWorkspaceRoleResponse, error) {
	results := make([]model.GroupWorkspaceRoleResponse, 0)
	err := r.db.Table("mcmp_group_workspace_roles gwr").
		Select("gwr.group_id, o.name as group_name, gwr.workspace_id, w.name as workspace_name, gwr.role_id, rm.name as role_name, gwr.created_at").
		Joins("JOIN mcmp_organizations o ON o.id = gwr.group_id").
		Joins("JOIN mcmp_workspaces w ON w.id = gwr.workspace_id").
		Joins("JOIN mcmp_role_masters rm ON rm.id = gwr.role_id").
		Where("gwr.group_id = ?", groupID).
		Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("error finding group workspace roles: %w", err)
	}
	return results, nil
}

// UpdateGroupWorkspaceRole 그룹-워크스페이스 역할 변경
func (r *GroupRoleRepository) UpdateGroupWorkspaceRole(groupID, workspaceID, roleID uint) error {
	result := r.db.Model(&model.GroupWorkspaceRole{}).
		Where("group_id = ? AND workspace_id = ?", groupID, workspaceID).
		Update("role_id", roleID)
	if result.Error != nil {
		return fmt.Errorf("error updating group workspace role: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrGroupWorkspaceRoleNotFound
	}
	return nil
}

// FindAvailableWorkspacesForGroup 그룹에 미매핑된 워크스페이스 목록 조회
func (r *GroupRoleRepository) FindAvailableWorkspacesForGroup(groupID uint) ([]*model.Workspace, error) {
	var workspaces []*model.Workspace
	err := r.db.Where("id NOT IN (?)",
		r.db.Table("mcmp_group_workspace_roles").
			Select("workspace_id").
			Where("group_id = ?", groupID),
	).Find(&workspaces).Error
	if err != nil {
		return nil, fmt.Errorf("error finding available workspaces: %w", err)
	}
	return workspaces, nil
}

// DeleteGroupWorkspaceRole 그룹-워크스페이스 매핑 삭제
func (r *GroupRoleRepository) DeleteGroupWorkspaceRole(groupID, workspaceID uint) error {
	result := r.db.Where("group_id = ? AND workspace_id = ?", groupID, workspaceID).Delete(&model.GroupWorkspaceRole{})
	if result.Error != nil {
		return fmt.Errorf("error deleting group workspace role: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrGroupWorkspaceRoleNotFound
	}
	return nil
}

// FindAvailablePlatformRoles 그룹에 할당되지 않은 플랫폼 역할 목록 조회
func (r *GroupRoleRepository) FindAvailablePlatformRoles(groupID uint) ([]model.RoleMaster, error) {
	var roles []model.RoleMaster
	err := r.db.Where("role_type = 'platform'").
		Where("id NOT IN (?)",
			r.db.Table("mcmp_group_platform_roles").
				Select("role_id").
				Where("group_id = ?", groupID),
		).
		Order("name ASC").
		Find(&roles).Error
	if err != nil {
		return nil, fmt.Errorf("error finding available platform roles for group %d: %w", groupID, err)
	}
	return roles, nil
}

// FindAvailableWorkspaces 그룹에 매핑되지 않은 워크스페이스 목록 조회
func (r *GroupRoleRepository) FindAvailableWorkspaces(groupID uint) ([]model.Workspace, error) {
	var workspaces []model.Workspace
	err := r.db.Where("id NOT IN (?)",
		r.db.Table("mcmp_group_workspace_roles").
			Select("workspace_id").
			Where("group_id = ?", groupID),
	).
		Order("name ASC").
		Find(&workspaces).Error
	if err != nil {
		return nil, fmt.Errorf("error finding available workspaces for group %d: %w", groupID, err)
	}
	return workspaces, nil
}

// isGroupDuplicateError unique constraint 위반 여부 확인 (PostgreSQL: 23505, SQLite: UNIQUE constraint failed)
func isGroupDuplicateError(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "23505") ||
		strings.Contains(err.Error(), "UNIQUE constraint failed"))
}
