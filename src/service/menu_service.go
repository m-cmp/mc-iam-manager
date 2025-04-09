package service

import (
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
)

type MenuService struct {
	menuRepo *repository.MenuRepository
}

func NewMenuService(menuRepo *repository.MenuRepository) *MenuService {
	return &MenuService{
		menuRepo: menuRepo,
	}
}

// GetMenus 모든 메뉴 조회
func (s *MenuService) GetMenus() ([]model.Menu, error) {
	return s.menuRepo.GetMenus()
}

// GetByID 메뉴 ID로 조회
func (s *MenuService) GetByID(id string) (*model.Menu, error) {
	return s.menuRepo.GetByID(id)
}

// Create 새 메뉴 생성
func (s *MenuService) Create(menu *model.Menu) error {
	return s.menuRepo.Create(menu)
}

// Update 메뉴 정보 업데이트
func (s *MenuService) Update(menu *model.Menu) error {
	return s.menuRepo.Update(menu)
}

// Delete 메뉴 삭제
func (s *MenuService) Delete(id string) error {
	return s.menuRepo.Delete(id)
}
