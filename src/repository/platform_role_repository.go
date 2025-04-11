package repository

import (
	"errors"

	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

type PlatformRoleRepository struct {
	db *gorm.DB
}

func NewPlatformRoleRepository(db *gorm.DB) *PlatformRoleRepository {
	return &PlatformRoleRepository{
		db: db,
	}
}

func (r *PlatformRoleRepository) List() ([]model.PlatformRole, error) {
	var roles []model.PlatformRole
	if err := r.db.Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *PlatformRoleRepository) GetByID(id uint) (*model.PlatformRole, error) {
	var role model.PlatformRole
	if err := r.db.First(&role, id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *PlatformRoleRepository) Create(role *model.PlatformRole) error {
	return r.db.Create(role).Error
}

func (r *PlatformRoleRepository) Update(role *model.PlatformRole) error {
	return r.db.Save(role).Error
}

func (r *PlatformRoleRepository) Delete(id uint) error {
	result := r.db.Delete(&model.PlatformRole{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// Consider returning a specific error like ErrPlatformRoleNotFound
		return errors.New("platform role not found or already deleted")
	}
	return nil
}

// GetByName finds a platform role by its name.
func (r *PlatformRoleRepository) GetByName(name string) (*model.PlatformRole, error) {
	var role model.PlatformRole
	if err := r.db.Where("name = ?", name).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Consider returning a specific error like ErrPlatformRoleNotFound
			return nil, errors.New("platform role not found")
		}
		return nil, err
	}
	return &role, nil
}

// DB returns the underlying gorm DB instance (Helper for sync function)
func (r *PlatformRoleRepository) DB() *gorm.DB {
	return r.db
}
