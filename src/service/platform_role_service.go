package service

import (
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm" // Import gorm
)

type PlatformRoleService struct {
	repo *repository.PlatformRoleRepository
	// db *gorm.DB // Not needed directly in service methods for now
}

func NewPlatformRoleService(db *gorm.DB) *PlatformRoleService { // Accept db
	// Initialize repository internally
	repo := repository.NewPlatformRoleRepository(db) // Pass db to repo constructor
	return &PlatformRoleService{
		repo: repo,
		// db: db, // Store db if needed later
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
