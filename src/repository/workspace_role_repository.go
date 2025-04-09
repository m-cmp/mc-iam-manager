package repository

import (
	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

type WorkspaceRoleRepository struct {
	db *gorm.DB
}

func NewWorkspaceRoleRepository(db *gorm.DB) *WorkspaceRoleRepository {
	return &WorkspaceRoleRepository{
		db: db,
	}
}

func (r *WorkspaceRoleRepository) List() ([]model.WorkspaceRole, error) {
	var roles []model.WorkspaceRole
	if err := r.db.Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *WorkspaceRoleRepository) GetByID(id uint) (*model.WorkspaceRole, error) {
	var role model.WorkspaceRole
	if err := r.db.First(&role, id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *WorkspaceRoleRepository) Create(role *model.WorkspaceRole) error {
	return r.db.Create(role).Error
}

func (r *WorkspaceRoleRepository) Update(role *model.WorkspaceRole) error {
	return r.db.Save(role).Error
}

func (r *WorkspaceRoleRepository) Delete(id uint) error {
	return r.db.Delete(&model.WorkspaceRole{}, id).Error
}
