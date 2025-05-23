package service

import (
	"context"
	"log"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
)

type CspRoleService struct {
	cspRoleRepo *repository.CspRoleRepository
}

func NewCspRoleService() *CspRoleService {
	cspRoleRepo, _ := repository.NewCspRoleRepository()

	return &CspRoleService{
		cspRoleRepo: cspRoleRepo,
	}
}

// GetAllCSPRoles 모든 CSP 역할을 조회합니다.
func (s *CspRoleService) GetAllCSPRoles(ctx context.Context, cspType string) ([]*model.CspRole, error) {
	roles, err := s.cspRoleRepo.FindAll()
	if err != nil {
		log.Printf("Failed to get CSP roles: %v", err)
		return nil, err
	}

	return roles, nil
}

// CSP 역할 목록 중 MCMP_ 접두사를 가진 역할만 조회합니다.
func (s *CspRoleService) GetCSPRoles(ctx context.Context, cspType string) ([]*model.CspRole, error) {
	roles, err := s.cspRoleRepo.FindByCspType(cspType)
	if err != nil {
		log.Printf("Failed to get CSP roles: %v", err)
		return nil, err
	}

	return roles, nil
}

// CreateCSPRole 새로운 CSP 역할을 생성합니다.
func (s *CspRoleService) CreateCSPRole(role *model.CspRole) error {
	err := s.cspRoleRepo.Create(role)
	if err != nil {
		log.Printf("Failed to create CSP role: %v", err)
		return err
	}

	return nil
}

// UpdateCSPRole CSP 역할 정보를 수정합니다.
func (s *CspRoleService) UpdateCSPRole(role *model.CspRole) error {
	err := s.cspRoleRepo.Update(role)
	if err != nil {
		log.Printf("Failed to update CSP role: %v", err)
		return err
	}

	return nil
}

// DeleteCSPRole CSP 역할을 삭제합니다.
func (s *CspRoleService) DeleteCSPRole(id string) error {
	err := s.cspRoleRepo.Delete(id)
	if err != nil {
		log.Printf("Failed to delete CSP role: %v", err)
		return err
	}

	return nil
}
