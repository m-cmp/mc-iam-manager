package service

import (
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
)

type WorkspaceRoleService struct {
	repo *repository.WorkspaceRoleRepository
}

func NewWorkspaceRoleService(repo *repository.WorkspaceRoleRepository) *WorkspaceRoleService {
	return &WorkspaceRoleService{
		repo: repo,
	}
}

func (s *WorkspaceRoleService) List() ([]model.WorkspaceRole, error) {
	return s.repo.List()
}

func (s *WorkspaceRoleService) GetByID(id uint) (*model.WorkspaceRole, error) {
	return s.repo.GetByID(id)
}

func (s *WorkspaceRoleService) Create(role *model.WorkspaceRole) error {
	return s.repo.Create(role)
}

func (s *WorkspaceRoleService) Update(role *model.WorkspaceRole) error {
	return s.repo.Update(role)
}

func (s *WorkspaceRoleService) Delete(id uint) error {
	return s.repo.Delete(id)
}
