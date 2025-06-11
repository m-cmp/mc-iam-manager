package service

import (
	"fmt"
	"log"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm"
)

// RoleService 역할 관리 서비스
type RoleService struct {
	db             *gorm.DB
	roleRepository *repository.RoleRepository
}

// NewRoleService 새 RoleService 인스턴스 생성
func NewRoleService(db *gorm.DB) *RoleService {
	return &RoleService{
		db:             db,
		roleRepository: repository.NewRoleRepository(db),
	}
}

// List 역할 목록 조회
func (s *RoleService) ListRoles(roleType string) ([]*model.RoleMaster, error) {
	return s.roleRepository.FindRoles(0, roleType)
}

// GetByID ID로 역할 조회
func (s *RoleService) GetRoleByID(roleId uint, roleType string) (*model.RoleMaster, error) {
	return s.roleRepository.FindRoleByRoleID(roleId, roleType)
}

// GetByName Name으로 역할 조회
func (s *RoleService) GetRoleByName(roleName string, roleType string) (*model.RoleMaster, error) {
	return s.roleRepository.FindRoleByRoleName(roleName, roleType)
}

// CreateRoleWithSubs 역할과 서브 타입을 함께 생성합니다.
func (s *RoleService) CreateRoleWithSubs(role *model.RoleMaster, roleSubs []model.RoleSub) (*model.RoleMaster, error) {
	// 트랜잭션 시작
	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("트랜잭션 시작 실패: %w", tx.Error)
	}

	// 롤백을 위한 defer
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. RoleMaster 생성
	// 먼저 RoleMaster가 이미 존재하는지 확인
	existingRole, err := s.roleRepository.FindRoleByRoleName(role.Name, "") // roleType은 빈 문자열로 전달
	if err != nil && err.Error() != "role not found" {
		tx.Rollback()
		return nil, fmt.Errorf("역할 조회 실패: %w", err)
	}

	if existingRole != nil {
		// 이미 존재하는 역할이면 해당 역할을 사용
		role = existingRole
	} else {
		// 새로운 역할 생성
		if err := tx.Create(role).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("역할 생성 실패: %w", err)
		}
	}

	// 2. RoleSub 생성
	for _, sub := range roleSubs {
		sub.RoleID = role.ID // RoleMaster의 ID 설정

		// RoleSub가 이미 존재하는지 확인
		var existingSub model.RoleSub
		if err := tx.Where("role_id = ? AND role_type = ?", role.ID, sub.RoleType).First(&existingSub).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				tx.Rollback()
				return role, fmt.Errorf("역할 서브 타입 조회 실패: %w", err)
			}
			// RoleSub가 존재하지 않는 경우에만 생성
			if err := tx.Create(&sub).Error; err != nil {
				tx.Rollback()
				return role, fmt.Errorf("역할 서브 타입 생성 실패: %w", err)
			}
		} else {
			// RoleSub가 이미 존재하는 경우 로그만 남기고 계속 진행
			log.Printf("역할 서브 타입 (RoleID: %d, Type: %s)가 이미 존재합니다. 건너뜁니다.", role.ID, sub.RoleType)
		}
	}

	// 3. 생성된 RoleMaster와 RoleSubs를 함께 조회
	var createdRole model.RoleMaster
	if err := tx.Preload("RoleSubs").First(&createdRole, role.ID).Error; err != nil {
		tx.Rollback()
		return &createdRole, fmt.Errorf("생성된 역할 조회 실패: %w", err)
	}

	// 트랜잭션 커밋
	if err := tx.Commit().Error; err != nil {
		return &createdRole, fmt.Errorf("트랜잭션 커밋 실패: %w", err)
	}

	return &createdRole, nil
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
	role, err := s.roleRepository.FindRoleByRoleID(roleID, model.RoleTypePlatform)
	if err != nil {
		return fmt.Errorf("역할 조회 실패: %w", err)
	}
	if role == nil {
		return fmt.Errorf("역할을 찾을 수 없습니다")
	}

	// 2. 역할이 platform 타입인지 확인
	isPlatformRole := false
	for _, sub := range role.RoleSubs {
		if sub.RoleType == model.RoleTypePlatform {
			isPlatformRole = true
			break
		}
	}
	if !isPlatformRole {
		return fmt.Errorf("플랫폼 역할이 아닙니다")
	}

	// 3. 역할 할당
	return s.roleRepository.AssignPlatformRole(userID, roleID)
}

// AssignWorkspaceRole 워크스페이스 역할 할당
func (s *RoleService) AssignWorkspaceRole(userID, workspaceID, roleID uint) error {
	// 1. 역할이 존재하는지 확인
	role, err := s.roleRepository.FindRoleByRoleID(roleID, model.RoleTypeWorkspace)
	if err != nil {
		return fmt.Errorf("역할 조회 실패: %w", err)
	}
	if role == nil {
		return fmt.Errorf("역할을 찾을 수 없습니다")
	}

	// 2. 역할이 workspace 타입인지 확인
	isWorkspaceRole := false
	for _, sub := range role.RoleSubs {
		if sub.RoleType == model.RoleTypeWorkspace {
			isWorkspaceRole = true
			break
		}
	}
	if !isWorkspaceRole {
		return fmt.Errorf("워크스페이스 역할이 아닙니다")
	}

	// 3. 역할 할당
	return s.roleRepository.AssignWorkspaceRole(userID, workspaceID, roleID)
}

// AssignRole 역할 할당 (플랫폼/워크스페이스)
func (s *RoleService) AssignRole(userID, workspaceID, roleID uint) error {
	// 1. 역할이 존재하는지 확인
	role, err := s.roleRepository.FindRoleByRoleID(roleID, "")
	if err != nil {
		return fmt.Errorf("역할 조회 실패: %w", err)
	}
	if role == nil {
		return fmt.Errorf("역할을 찾을 수 없습니다")
	}

	// 2. 역할 타입 확인
	isWorkspaceRole := false
	isPlatformRole := false
	for _, sub := range role.RoleSubs {
		if sub.RoleType == model.RoleTypeWorkspace {
			isWorkspaceRole = true
		}
		if sub.RoleType == model.RoleTypePlatform {
			isPlatformRole = true
		}
	}

	// 3. 역할 타입에 따라 할당
	if isWorkspaceRole {
		if workspaceID == 0 {
			return fmt.Errorf("워크스페이스 역할 할당을 위해 워크스페이스 ID가 필요합니다")
		}
		return s.roleRepository.AssignWorkspaceRole(userID, workspaceID, roleID)
	} else if isPlatformRole {
		return s.roleRepository.AssignPlatformRole(userID, roleID)
	} else {
		return fmt.Errorf("지원하지 않는 역할 타입입니다")
	}
}

// RemovePlatformRole 플랫폼 역할 제거
func (s *RoleService) RemovePlatformRole(userID, roleID uint) error {
	return s.roleRepository.RemovePlatformRole(userID, roleID)
}

// RemoveWorkspaceRole 워크스페이스 역할 제거
func (s *RoleService) RemoveWorkspaceRole(userID, workspaceID, roleID uint) error {
	return s.roleRepository.RemoveWorkspaceRole(userID, workspaceID, roleID)
}

// GetUserWorkspaceRoles 사용자의 워크스페이스 역할 목록 조회
func (s *RoleService) GetUserWorkspaceRoles(userID, workspaceID uint) ([]model.RoleMaster, error) {
	return s.roleRepository.FindUserWorkspaceRoles(userID, workspaceID)
}

// GetUserPlatformRoles 사용자의 플랫폼 역할 목록 조회
func (s *RoleService) GetUserPlatformRoles(userID uint) ([]model.RoleMaster, error) {
	return s.roleRepository.FindUserPlatformRoles(userID)
}

// CreateWorkspaceRoleCspRoleMapping 워크스페이스 역할-CSP 역할 매핑 생성
func (s *RoleService) CreateWorkspaceRoleCspRoleMapping(mapping model.RoleMasterCspRoleMapping) (*model.RoleMasterCspRoleMapping, error) {
	// 1. 워크스페이스 역할이 존재하는지 확인
	workspaceRole, err := s.roleRepository.FindRoleByRoleID(mapping.RoleID, model.RoleTypeWorkspace)
	if err != nil {
		return nil, fmt.Errorf("워크스페이스 역할 조회 실패: %w", err)
	}
	if workspaceRole == nil {
		return nil, fmt.Errorf("워크스페이스 역할을 찾을 수 없습니다")
	}

	// 2. CSP 역할이 존재하는지 확인
	cspRole, err := s.roleRepository.FindRoleByRoleID(mapping.CspRoleID, model.RoleTypeCSP)
	if err != nil {
		return nil, fmt.Errorf("CSP 역할 조회 실패: %w", err)
	}
	if cspRole == nil {
		return nil, fmt.Errorf("CSP 역할을 찾을 수 없습니다")
	}

	// 3. 매핑 생성
	err = s.roleRepository.CreateWorkspaceRoleCspRoleMapping(&mapping)
	if err != nil {
		return nil, fmt.Errorf("매핑 생성 실패: %w", err)
	}

	return &mapping, nil
}

// DeleteWorkspaceRoleCspRoleMapping 워크스페이스 역할-CSP 역할 매핑 삭제
func (s *RoleService) DeleteWorkspaceRoleCspRoleMapping(workspaceRoleID uint, cspRoleID uint, cspType string) error {
	return s.roleRepository.DeleteWorkspaceRoleCspRoleMapping(workspaceRoleID, cspRoleID, cspType)
}

// GetWorkspaceRoleCspRoleMappings 워크스페이스 역할-CSP 역할 매핑 목록 조회
func (s *RoleService) ListWorkspaceRoleCspRoleMappings(workspaceRoleID uint, cspRoleID uint, cspType string) ([]*model.RoleMasterCspRoleMapping, error) {
	return s.roleRepository.FindWorkspaceRoleCspRoleMappings(workspaceRoleID, cspRoleID, cspType)
}

// GetWorkspaceRoleCspRoleMappings 역할-CSP 역할 매핑 목록 조회
func (s *RoleService) GetRoleCspRoleMappings(roleID uint, cspRoleID uint, cspType string) ([]*model.RoleMasterCspRoleMapping, error) {
	return s.roleRepository.FindRoleMasterCspRoleMappings(roleID, cspRoleID, cspType)
}

// GetUsersByWorkspaceID 워크스페이스에 속한 사용자 목록을 조회합니다.
func (s *RoleService) ListWorkspaceUsersAndRoles(req model.WorkspaceFilterRequest) ([]*model.WorkspaceWithUsersAndRoles, error) {
	// 워크스페이스 존재 여부 확인
	workspaceUsers, err := s.roleRepository.FindWorkspaceWithUsersRoles(req)
	if err != nil {
		return nil, err
	}

	return workspaceUsers, nil
}

// GetUsersAndRolesByWorkspaceID 워크스페이스에 속한 사용자와 역할 조회 : 사용자기준
func (s *RoleService) ListUsersAndRolesWithWorkspaces(req model.WorkspaceFilterRequest) ([]*model.UserWorkspaceRole, error) {

	userRoleWorkspaces, err := s.roleRepository.FindUsersAndRolesWithWorkspaces(req)
	if err != nil {
		return nil, err
	}

	return userRoleWorkspaces, nil
}

// GetWorkspaceRoles 워크스페이스의 모든 역할 목록 조회 Role만
func (s *RoleService) ListWorkspaceRoles(workspaceID uint) ([]*model.RoleMaster, error) {
	roleType := model.RoleTypeWorkspace
	roles, err := s.roleRepository.FindRoles(workspaceID, roleType)
	if err != nil {
		return nil, err
	}

	return roles, nil
}

// IsAssignedPlatformRole 사용자에게 특정 플랫폼 역할이 할당되어 있는지 확인
func (s *RoleService) IsAssignedPlatformRole(userID uint, roleID uint) (bool, error) {
	return s.roleRepository.IsAssignedRole(userID, roleID, model.RoleTypePlatform)
}
func (s *RoleService) IsAssignedWorkspaceRole(userID uint, roleID uint) (bool, error) {
	return s.roleRepository.IsAssignedRole(userID, roleID, model.RoleTypeWorkspace)
}
func (s *RoleService) IsAssignedRole(userID uint, roleID uint, roleType string) (bool, error) {
	return s.roleRepository.IsAssignedRole(userID, roleID, roleType)
}
