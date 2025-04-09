package service

import (
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
)

type PlatformRoleService struct {
	repo *repository.PlatformRoleRepository
}

func NewPlatformRoleService(repo *repository.PlatformRoleRepository) *PlatformRoleService {
	return &PlatformRoleService{
		repo: repo,
	}
}

func (s *PlatformRoleService) List() ([]model.PlatformRole, error) {
	return s.repo.List()
}

func (s *PlatformRoleService) GetByID(id uint) (*model.PlatformRole, error) {
	return s.repo.GetByID(id)
}

func (s *PlatformRoleService) Create(role *model.PlatformRole) error {
	return s.repo.Create(role)
}

func (s *PlatformRoleService) Update(role *model.PlatformRole) error {
	return s.repo.Update(role)
}

func (s *PlatformRoleService) Delete(id uint) error {
	return s.repo.Delete(id)
}
