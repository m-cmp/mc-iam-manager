package repository

import (
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
	return r.db.Delete(&model.PlatformRole{}, id).Error
}
