package repository

import (
	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

// MenuMappingRepository 메뉴 매핑 데이터 관리
type MenuMappingRepository struct {
	db *gorm.DB
}

// NewMenuMappingRepository 새 MenuMappingRepository 인스턴스 생성
func NewMenuMappingRepository(db *gorm.DB) *MenuMappingRepository {
	return &MenuMappingRepository{db: db}
}

// GetMappedMenuIDs 플랫폼 역할에 매핑된 메뉴 ID 목록 조회
func (r *MenuMappingRepository) GetMappedMenuIDs(platformRole string) ([]string, error) {
	var menuIDs []string
	query := r.db.Model(&model.PlatformRoleMenuMapping{}).
		Where("platform_role = ?", platformRole).
		Pluck("menu_id", &menuIDs)
	if err := query.Error; err != nil {
		return nil, err
	}

	return menuIDs, nil
}

// CreateMapping 플랫폼 역할-메뉴 매핑 생성
func (r *MenuMappingRepository) CreateMapping(platformRole string, menuID string) error {
	mapping := model.PlatformRoleMenuMapping{
		PlatformRole: platformRole,
		MenuID:       menuID,
	}
	query := r.db.Create(&mapping)
	return query.Error
}

// DeleteMapping 플랫폼 역할-메뉴 매핑 삭제
func (r *MenuMappingRepository) DeleteMapping(platformRole string, menuID string) error {
	query := r.db.Where("platform_role = ? AND menu_id = ?", platformRole, menuID).
		Delete(&model.PlatformRoleMenuMapping{})
	return query.Error
}

// FindMappedMenusWithParents 플랫폼 역할에 매핑된 메뉴와 그 상위 메뉴들을 포함한 메뉴 트리 조회
func (r *MenuMappingRepository) FindMappedMenusWithParents(platformRole string) ([]*model.Menu, error) {
	var menus []*model.Menu

	// 1. 매핑된 메뉴 ID 목록 조회
	var mappedMenuIDs []string
	query := r.db.Model(&model.PlatformRoleMenuMapping{}).
		Where("platform_role = ?", platformRole).
		Pluck("menu_id", &mappedMenuIDs)
	if err := query.Error; err != nil {
		return nil, err
	}

	// 2. 매핑된 메뉴와 그 상위 메뉴들을 재귀적으로 조회
	query = r.db.Where("id IN ? OR parent_id IN (SELECT parent_id FROM mcmp_menu WHERE id IN ?)", mappedMenuIDs, mappedMenuIDs).
		Find(&menus)
	if err := query.Error; err != nil {
		return nil, err
	}

	return menus, nil
}

func (r *MenuMappingRepository) Create(mapping *model.PlatformRoleMenuMapping) error {
	return r.db.Create(mapping).Error
}

func (r *MenuMappingRepository) GetByMenuID(menuID uint) ([]model.PlatformRoleMenuMapping, error) {
	var mappings []model.PlatformRoleMenuMapping
	err := r.db.Where("menu_id = ?", menuID).Find(&mappings).Error
	return mappings, err
}

func (r *MenuMappingRepository) GetByPermissionID(permissionID uint) ([]model.PlatformRoleMenuMapping, error) {
	var mappings []model.PlatformRoleMenuMapping
	err := r.db.Where("permission_id = ?", permissionID).Find(&mappings).Error
	return mappings, err
}

func (r *MenuMappingRepository) DeleteByMenuID(menuID uint) error {
	return r.db.Where("menu_id = ?", menuID).Delete(&model.PlatformRoleMenuMapping{}).Error
}

func (r *MenuMappingRepository) DeleteByPermissionID(permissionID uint) error {
	return r.db.Where("permission_id = ?", permissionID).Delete(&model.PlatformRoleMenuMapping{}).Error
}

func (r *MenuMappingRepository) DeleteByMenuIDAndPermissionID(menuID, permissionID uint) error {
	return r.db.Where("menu_id = ? AND permission_id = ?", menuID, permissionID).Delete(&model.PlatformRoleMenuMapping{}).Error
}

// FindMappedMenusByRole returns menu IDs mapped to the given platform role
func (r *MenuMappingRepository) FindMappedMenusByRole(platformRole string) ([]string, error) {
	var menuIDs []string
	err := r.db.Model(&model.PlatformRoleMenuMapping{}).
		Where("platform_role = ?", platformRole).
		Pluck("menu_id", &menuIDs).Error
	return menuIDs, err
}
