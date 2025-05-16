package service

import (
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm"
)

type CspRoleService struct {
	cspRoleRepo *repository.CspRoleRepository
}

func NewCspRoleService(db *gorm.DB) *CspRoleService {
	return &CspRoleService{
		cspRoleRepo: repository.NewCspRoleRepository(db),
	}
}

// GetAllCSPRoles 모든 CSP 역할을 조회합니다.
func (s *CspRoleService) GetAllCSPRoles() ([]*model.CspRole, error) {
	return s.cspRoleRepo.FindAll()
}

// CreateCSPRole 새로운 CSP 역할을 생성합니다.
func (s *CspRoleService) CreateCSPRole(role *model.CspRole) error {
	return s.cspRoleRepo.Create(role)
}

// UpdateCSPRole CSP 역할 정보를 수정합니다.
func (s *CspRoleService) UpdateCSPRole(role *model.CspRole) error {
	return s.cspRoleRepo.Update(role)
}

// DeleteCSPRole CSP 역할을 삭제합니다.
func (s *CspRoleService) DeleteCSPRole(id string) error {
	return s.cspRoleRepo.Delete(id)
}
