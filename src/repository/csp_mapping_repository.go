package repository

import (
	"errors"

	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

var (
	ErrCspMappingNotFound      = errors.New("CSP role mapping not found")
	ErrCspMappingAlreadyExists = errors.New("CSP role mapping for this workspace role and CSP type already exists (or combination is duplicate)")
)

// CspMappingRepository 역할-CSP 역할 매핑 데이터 관리
type CspMappingRepository struct {
	db *gorm.DB
}

// NewCspMappingRepository 새 CspMappingRepository 인스턴스 생성
func NewCspMappingRepository(db *gorm.DB) *CspMappingRepository {
	return &CspMappingRepository{db: db}
}

// Create 역할-CSP 역할 매핑 생성
func (r *CspMappingRepository) Create(mapping *model.WorkspaceRoleCspRoleMapping) error {
	return r.db.Create(mapping).Error
}

// ListByWorkspaceRole 워크스페이스 역할 ID로 매핑 목록 조회
func (r *CspMappingRepository) ListByWorkspaceRole(workspaceRoleID uint) ([]model.WorkspaceRoleCspRoleMapping, error) {
	var mappings []model.WorkspaceRoleCspRoleMapping
	if err := r.db.Where("workspace_role_id = ?", workspaceRoleID).Find(&mappings).Error; err != nil {
		return nil, err
	}
	return mappings, nil
}

// Get 역할-CSP 역할 매핑 조회 (복합 키 사용)
func (r *CspMappingRepository) Get(workspaceRoleID uint, cspType string, cspRoleArn string) (*model.WorkspaceRoleCspRoleMapping, error) {
	var mapping model.WorkspaceRoleCspRoleMapping
	err := r.db.Where("workspace_role_id = ? AND csp_type = ? AND csp_role_arn = ?", workspaceRoleID, cspType, cspRoleArn).First(&mapping).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCspMappingNotFound
		}
		return nil, err
	}
	return &mapping, nil
}

// FindByRoleAndCspType 워크스페이스 역할 ID와 CSP 타입으로 매핑 목록 조회 (특정 타입의 모든 매핑 조회 시)
func (r *CspMappingRepository) FindByRoleAndCspType(workspaceRoleID uint, cspType string) ([]model.WorkspaceRoleCspRoleMapping, error) {
	var mappings []model.WorkspaceRoleCspRoleMapping
	if err := r.db.Where("workspace_role_id = ? AND csp_type = ?", workspaceRoleID, cspType).Find(&mappings).Error; err != nil {
		return nil, err
	}
	return mappings, nil
}

// Update 역할-CSP 역할 매핑 수정 (Description, IdpIdentifier 등)
// PK는 변경 불가하므로, 다른 필드 업데이트용.
func (r *CspMappingRepository) Update(workspaceRoleID uint, cspType string, cspRoleArn string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return errors.New("no fields provided for update")
	}
	// Prevent updating primary keys or createdAt
	delete(updates, "workspace_role_id")
	delete(updates, "csp_type")
	delete(updates, "csp_role_arn")
	delete(updates, "createdAt")

	result := r.db.Model(&model.WorkspaceRoleCspRoleMapping{}).
		Where("workspace_role_id = ? AND csp_type = ? AND csp_role_arn = ?", workspaceRoleID, cspType, cspRoleArn).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCspMappingNotFound
	}
	return nil
}

// Delete 역할-CSP 역할 매핑 삭제
func (r *CspMappingRepository) Delete(workspaceRoleID uint, cspType string, cspRoleArn string) error {
	query := r.db.Where("workspace_role_id = ? AND csp_type = ? AND csp_role_arn = ?", workspaceRoleID, cspType, cspRoleArn).Delete(&model.WorkspaceRoleCspRoleMapping{})
	if err := query.Error; err != nil {
		return err
	}

	if query.RowsAffected == 0 {
		return ErrCspMappingNotFound
	}
	return nil
}

// GetByCspID CSP ID로 매핑 조회
func (r *CspMappingRepository) GetByCspID(cspID uint) ([]model.WorkspaceRoleCspRoleMapping, error) {
	var mappings []model.WorkspaceRoleCspRoleMapping
	err := r.db.Where("csp_id = ?", cspID).Find(&mappings).Error
	return mappings, err
}

// GetByPermissionID 권한 ID로 매핑 조회
func (r *CspMappingRepository) GetByPermissionID(permissionID uint) ([]model.WorkspaceRoleCspRoleMapping, error) {
	var mappings []model.WorkspaceRoleCspRoleMapping
	err := r.db.Where("permission_id = ?", permissionID).Find(&mappings).Error
	return mappings, err
}

// DeleteByCspID CSP ID로 매핑 삭제
func (r *CspMappingRepository) DeleteByCspID(cspID uint) error {
	return r.db.Where("csp_id = ?", cspID).Delete(&model.WorkspaceRoleCspRoleMapping{}).Error
}

// DeleteByPermissionID 권한 ID로 매핑 삭제
func (r *CspMappingRepository) DeleteByPermissionID(permissionID uint) error {
	return r.db.Where("permission_id = ?", permissionID).Delete(&model.WorkspaceRoleCspRoleMapping{}).Error
}

// DeleteByCspIDAndPermissionID CSP ID와 권한 ID로 매핑 삭제
func (r *CspMappingRepository) DeleteByCspIDAndPermissionID(cspID, permissionID uint) error {
	return r.db.Where("csp_id = ? AND permission_id = ?", cspID, permissionID).Delete(&model.WorkspaceRoleCspRoleMapping{}).Error
}
