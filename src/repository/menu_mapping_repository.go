package repository

import (
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/util"
	"gorm.io/gorm"
)

// MenuMappingRepository menu mapping data management
type MenuMappingRepository struct {
	db *gorm.DB
}

// NewMenuMappingRepository create new MenuMappingRepository instance
func NewMenuMappingRepository(db *gorm.DB) *MenuMappingRepository {
	return &MenuMappingRepository{db: db}
}

// GetMappedMenuIDs retrieve menu ID list mapped to role
func (r *MenuMappingRepository) GetMappedMenuIDs(roleID uint) ([]string, error) {
	var menuIDs []string
	query := r.db.Model(&model.RoleMenuMapping{}).
		Where("role_id = ?", roleID).
		Pluck("menu_id", &menuIDs)
	if err := query.Error; err != nil {
		return nil, err
	}
	return menuIDs, nil
}

// FindMappedMenusWithParents retrieve menu tree including menus mapped to role and their parent menus
func (r *MenuMappingRepository) FindMappedMenusWithParents(req *model.MenuMappingFilterRequest) ([]*model.Menu, error) {
	var menus []*model.Menu

	// 1. Retrieve mapped menu ID list
	var mappedMenuIDs []string
	query := r.db.Model(&model.RoleMenuMapping{}).
		Where("role_id in ?", req.RoleIDs).
		Pluck("menu_id", &mappedMenuIDs)
	if err := query.Error; err != nil {
		return nil, err
	}

	// 2. Recursively retrieve mapped menus and their parent menus
	query = r.db.Where("id IN ? OR parent_id IN (SELECT parent_id FROM mcmp_menu WHERE id IN ?)", mappedMenuIDs, mappedMenuIDs).
		Find(&menus)
	if err := query.Error; err != nil {
		return nil, err
	}

	return menus, nil
}

// Retrieve menu list mapped to the role
func (r *MenuMappingRepository) FindMappedMenus(roleID uint) ([]*model.Menu, error) {
	var menus []*model.Menu

	// 1. Retrieve mapped menu ID list
	var mappedMenuIDs []string
	query := r.db.Model(&model.RoleMenuMapping{}).
		Where("role_id = ?", roleID).
		Pluck("menu_id", &mappedMenuIDs)
	if err := query.Error; err != nil {
		return nil, err
	}

	query = r.db.Where("id IN ?", mappedMenuIDs).
		Find(&menus)
	if err := query.Error; err != nil {
		return nil, err
	}

	return menus, nil
}

func (r *MenuMappingRepository) Create(mapping *model.RoleMenuMapping) error {
	return r.db.Create(mapping).Error
}

func (r *MenuMappingRepository) GetByMenuID(menuID uint) ([]model.RoleMenuMapping, error) {
	var mappings []model.RoleMenuMapping
	err := r.db.Where("menu_id = ?", menuID).Find(&mappings).Error
	return mappings, err
}

func (r *MenuMappingRepository) GetByPermissionID(permissionID uint) ([]model.RoleMenuMapping, error) {
	var mappings []model.RoleMenuMapping
	err := r.db.Where("permission_id = ?", permissionID).Find(&mappings).Error
	return mappings, err
}

func (r *MenuMappingRepository) DeleteByMenuID(menuID uint) error {
	return r.db.Where("menu_id = ?", menuID).Delete(&model.RoleMenuMapping{}).Error
}

func (r *MenuMappingRepository) DeleteByPermissionID(permissionID uint) error {
	return r.db.Where("permission_id = ?", permissionID).Delete(&model.RoleMenuMapping{}).Error
}

func (r *MenuMappingRepository) DeleteByMenuIDAndPermissionID(menuID, permissionID uint) error {
	return r.db.Where("menu_id = ? AND permission_id = ?", menuID, permissionID).Delete(&model.RoleMenuMapping{}).Error
}

// FindMappedMenusByRole returns menu IDs mapped to the given role
func (r *MenuMappingRepository) FindMappedMenuIDs(req *model.MenuMappingFilterRequest) ([]*string, error) {
	var mappings []*string

	query := r.db.Model(&model.RoleMenuMapping{})

	roleIDs := []uint{}
	if len(req.RoleIDs) > 0 {
		for _, roleID := range req.RoleIDs {
			roleIDInt, err := util.StringToUint(roleID)
			if err != nil {
				return nil, err
			}
			roleIDs = append(roleIDs, roleIDInt)
		}

		query = query.Where("role_id in ?", roleIDs)
	}
	// for _, roleID := range req.RoleID {
	// 	if roleID != "" {
	// 		query = query.Where("role_id = ?", roleID)
	// 	}
	// }
	if req.MenuID != "" {
		query = query.Where("menu_id = ?", req.MenuID)
	}

	err := query.Pluck("menu_id", &mappings).Error
	return mappings, err
}
