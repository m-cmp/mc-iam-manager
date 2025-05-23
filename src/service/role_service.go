package service

import (
	"fmt"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm"
)

// RoleService 역할 관리 서비스
type RoleService struct {
	db         *gorm.DB
	repository *repository.RoleRepository
}

// NewRoleService 새 RoleService 인스턴스 생성
func NewRoleService(db *gorm.DB) *RoleService {
	return &RoleService{
		db:         db,
		repository: repository.NewRoleRepository(db),
	}
}

// List 역할 목록 조회
func (s *RoleService) List(roleType string) ([]model.RoleMaster, error) {
	return s.repository.List(roleType)
}

// GetByID ID로 역할 조회
func (s *RoleService) GetByID(id uint) (*model.RoleMaster, error) {
	return s.repository.GetByID(id)
}

// CreateRoleWithSubs 역할과 역할 서브 타입들을 함께 생성
func (s *RoleService) CreateRoleWithSubs(role model.RoleMaster, roleTypes []string) (*model.RoleMaster, error) {
	var createdRole *model.RoleMaster
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 역할 마스터 생성
		if err := tx.Create(&role).Error; err != nil {
			return fmt.Errorf("역할 생성 실패: %w", err)
		}

		// 2. 역할 서브 타입들 생성
		for _, roleType := range roleTypes {
			roleSub := model.RoleSub{
				RoleID:   role.ID,
				RoleType: roleType,
			}
			if err := tx.Create(&roleSub).Error; err != nil {
				return fmt.Errorf("역할 서브 타입 생성 실패: %w", err)
			}
		}

		// 3. 생성된 역할 정보 조회 (서브 타입 포함)
		if err := tx.Preload("RoleSubs").First(&createdRole, role.ID).Error; err != nil {
			return fmt.Errorf("생성된 역할 조회 실패: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return createdRole, nil
}

// UpdateRoleWithSubs 역할과 역할 서브 타입들을 함께 수정
func (s *RoleService) UpdateRoleWithSubs(role model.RoleMaster, roleTypes []string) (*model.RoleMaster, error) {
	var updatedRole *model.RoleMaster
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 역할 마스터 수정
		if err := tx.Save(&role).Error; err != nil {
			return fmt.Errorf("역할 수정 실패: %w", err)
		}

		// 2. 기존 역할 서브 타입들 삭제
		if err := tx.Where("role_id = ?", role.ID).Delete(&model.RoleSub{}).Error; err != nil {
			return fmt.Errorf("기존 역할 서브 타입 삭제 실패: %w", err)
		}

		// 3. 새로운 역할 서브 타입들 생성
		for _, roleType := range roleTypes {
			roleSub := model.RoleSub{
				RoleID:   role.ID,
				RoleType: roleType,
			}
			if err := tx.Create(&roleSub).Error; err != nil {
				return fmt.Errorf("역할 서브 타입 생성 실패: %w", err)
			}
		}

		// 4. 수정된 역할 정보 조회 (서브 타입 포함)
		if err := tx.Preload("RoleSubs").First(&updatedRole, role.ID).Error; err != nil {
			return fmt.Errorf("수정된 역할 조회 실패: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return updatedRole, nil
}

// DeleteRoleWithSubs 역할과 관련된 모든 서브 타입을 함께 삭제
func (s *RoleService) DeleteRoleWithSubs(roleID uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 역할 서브 타입들 삭제
		if err := tx.Where("role_id = ?", roleID).Delete(&model.RoleSub{}).Error; err != nil {
			return fmt.Errorf("역할 서브 타입 삭제 실패: %w", err)
		}

		// 2. 역할 마스터 삭제
		if err := tx.Delete(&model.RoleMaster{}, roleID).Error; err != nil {
			return fmt.Errorf("역할 삭제 실패: %w", err)
		}

		return nil
	})
}

// AssignPlatformRole 플랫폼 역할 할당
func (s *RoleService) AssignPlatformRole(userID, roleID uint) error {
	// 1. 역할이 존재하는지 확인
	role, err := s.repository.GetByID(roleID)
	if err != nil {
		return fmt.Errorf("역할 조회 실패: %w", err)
	}
	if role == nil {
		return fmt.Errorf("역할을 찾을 수 없습니다")
	}

	// 2. 역할이 platform 타입인지 확인
	isPlatformRole := false
	for _, sub := range role.RoleSubs {
		if sub.RoleType == "platform" {
			isPlatformRole = true
			break
		}
	}
	if !isPlatformRole {
		return fmt.Errorf("플랫폼 역할이 아닙니다")
	}

	// 3. 역할 할당
	return s.repository.AssignPlatformRole(userID, roleID)
}

// RemovePlatformRole 플랫폼 역할 제거
func (s *RoleService) RemovePlatformRole(userID, roleID uint) error {
	return s.repository.RemovePlatformRole(userID, roleID)
}

// AssignWorkspaceRole 워크스페이스 역할 할당
func (s *RoleService) AssignWorkspaceRole(userID, workspaceID, roleID uint) error {
	// 1. 역할이 존재하는지 확인
	role, err := s.repository.GetByID(roleID)
	if err != nil {
		return fmt.Errorf("역할 조회 실패: %w", err)
	}
	if role == nil {
		return fmt.Errorf("역할을 찾을 수 없습니다")
	}

	// 2. 역할이 workspace 타입인지 확인
	isWorkspaceRole := false
	for _, sub := range role.RoleSubs {
		if sub.RoleType == "workspace" {
			isWorkspaceRole = true
			break
		}
	}
	if !isWorkspaceRole {
		return fmt.Errorf("워크스페이스 역할이 아닙니다")
	}

	// 3. 역할 할당
	return s.repository.AssignWorkspaceRole(userID, workspaceID, roleID)
}

// RemoveWorkspaceRole 워크스페이스 역할 제거
func (s *RoleService) RemoveWorkspaceRole(userID, workspaceID, roleID uint) error {
	return s.repository.RemoveWorkspaceRole(userID, workspaceID, roleID)
}

// GetUserWorkspaceRoles 사용자의 워크스페이스 역할 목록 조회
func (s *RoleService) GetUserWorkspaceRoles(userID, workspaceID uint) ([]model.RoleMaster, error) {
	return s.repository.GetUserWorkspaceRoles(userID, workspaceID)
}

// GetUserPlatformRoles 사용자의 플랫폼 역할 목록 조회
func (s *RoleService) GetUserPlatformRoles(userID uint) ([]model.RoleMaster, error) {
	return s.repository.GetUserPlatformRoles(userID)
}

// GetByName 이름으로 역할 조회
func (s *RoleService) GetByName(name string) (*model.RoleMaster, error) {
	var role model.RoleMaster
	if err := s.db.Preload("RoleSubs").Where("name = ?", name).First(&role).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("역할 조회 실패: %w", err)
	}
	return &role, nil
}
