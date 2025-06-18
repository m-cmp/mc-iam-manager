package repository

import (
	"errors"
	"fmt"

	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

// PlatformRoleRepository 플랫폼 역할 레포지토리
type PlatformRoleRepository struct {
	db *gorm.DB
}

// NewPlatformRoleRepository 새 PlatformRoleRepository 인스턴스 생성
func NewPlatformRoleRepository(db *gorm.DB) *PlatformRoleRepository {
	return &PlatformRoleRepository{db: db}
}

// List 모든 플랫폼 역할 목록 조회
func (r *PlatformRoleRepository) List() ([]model.RoleMaster, error) {
	var roles []model.RoleMaster
	if err := r.db.Preload("RoleSubs").
		Joins("JOIN mcmp_role_sub ON mcmp_role_master.id = mcmp_role_sub.role_id").
		Where("mcmp_role_sub.role_type = ?", constants.RoleTypePlatform).
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// GetByID ID로 플랫폼 역할 조회
func (r *PlatformRoleRepository) GetByID(id uint) (*model.RoleMaster, error) {
	var role model.RoleMaster
	if err := r.db.Preload("RoleSubs").
		Joins("JOIN mcmp_role_sub ON mcmp_role_master.id = mcmp_role_sub.role_id").
		Where("mcmp_role_master.id = ? AND mcmp_role_sub.role_type = ?", id, constants.RoleTypePlatform).
		First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// Create 새 플랫폼 역할 생성
func (r *PlatformRoleRepository) Create(role *model.RoleMaster) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(role).Error; err != nil {
			return err
		}
		roleSub := model.RoleSub{
			RoleID:   role.ID,
			RoleType: constants.RoleTypePlatform,
		}
		return tx.Create(&roleSub).Error
	})
}

// Update 플랫폼 역할 정보 수정
func (r *PlatformRoleRepository) Update(role *model.RoleMaster) error {
	return r.db.Save(role).Error
}

// Delete 플랫폼 역할 삭제
func (r *PlatformRoleRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ? AND role_type = ?", id, constants.RoleTypePlatform).Delete(&model.RoleSub{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.RoleMaster{}, id).Error
	})
}

// roleModelFactories 역할 타입별 사용자-역할 매핑 모델 생성 팩토리 맵
// 각 역할 타입(platform, workspace)에 대해 해당하는 매핑 모델을 생성하는 함수를 제공
var roleModelFactories = map[string]func(userID, roleID uint) interface{}{
	// 플랫폼 역할 타입에 대한 팩토리 함수
	// UserPlatformRole 모델을 생성하여 반환
	constants.RoleTypePlatform: func(userID, roleID uint) interface{} {
		return &model.UserPlatformRole{
			UserID: userID,
			RoleID: roleID,
		}
	},
	// 워크스페이스 역할 타입에 대한 팩토리 함수
	// UserWorkspaceRole 모델을 생성하여 반환
	constants.RoleTypeWorkspace: func(userID, roleID uint) interface{} {
		return &model.UserWorkspaceRole{
			UserID: userID,
			RoleID: roleID,
		}
	},
}

// AssignRole 사용자에게 역할 할당
func (r *PlatformRoleRepository) AssignRole(userID, roleID uint, roleType string) error {
	factory, ok := roleModelFactories[roleType]
	if !ok {
		return fmt.Errorf("unsupported role type: %s", roleType)
	}
	role := factory(userID, roleID)
	return r.db.Create(role).Error
}

// RemoveRole 사용자의 플랫폼 역할 제거
func (r *PlatformRoleRepository) RemoveRole(userID, roleID uint) error {
	return r.db.Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&model.UserPlatformRole{}).Error
}

// GetUserRoles 사용자의 플랫폼 역할 목록 조회
func (r *PlatformRoleRepository) GetUserRoles(userID uint) ([]model.RoleMaster, error) {
	var roles []model.RoleMaster
	if err := r.db.Preload("RoleSubs").
		Joins("JOIN mcmp_user_platform_roles ON mcmp_role_master.id = mcmp_user_platform_roles.role_id").
		Joins("JOIN mcmp_role_sub ON mcmp_role_master.id = mcmp_role_sub.role_id").
		Where("mcmp_user_platform_roles.user_id = ? AND mcmp_role_sub.role_type = ?", userID, constants.RoleTypePlatform).
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// GetByName 이름으로 플랫폼 역할 조회
func (r *PlatformRoleRepository) GetByName(name string) (*model.RoleMaster, error) {
	var role model.RoleMaster
	if err := r.db.Preload("RoleSubs").
		Joins("JOIN mcmp_role_sub ON mcmp_role_master.id = mcmp_role_sub.role_id").
		Where("mcmp_role_master.name = ? AND mcmp_role_sub.role_type = ?", name, constants.RoleTypePlatform).
		First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
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
