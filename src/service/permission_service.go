package service

import (
	"context"
	// "errors" // Removed unused import

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm" // Import gorm
)

// MciamPermissionService MC-IAM 권한 관리 서비스 - Renamed
type MciamPermissionService struct {
	db             *gorm.DB                              // Add db field
	permissionRepo *repository.MciamPermissionRepository // Use renamed repo type
}

// NewMciamPermissionService MC-IAM 권한 관리 서비스 생성 - Renamed
func NewMciamPermissionService(db *gorm.DB) *MciamPermissionService {
	// Initialize repository internally
	permissionRepo := repository.NewMciamPermissionRepository(db) // Use renamed constructor
	return &MciamPermissionService{
		db:             db, // Store db
		permissionRepo: permissionRepo,
	}
}

// Create MC-IAM 권한 생성 - Renamed
func (s *MciamPermissionService) Create(ctx context.Context, permission *model.MciamPermission) error { // Use renamed model
	// Add validation for permission ID format if needed
	// e.g., parts := strings.Split(permission.ID, ":"); if len(parts) != 3 { ... }
	return s.permissionRepo.Create(permission)
}

// GetByID ID로 MC-IAM 권한 조회 - Renamed
func (s *MciamPermissionService) GetByID(ctx context.Context, id string) (*model.MciamPermission, error) { // Use renamed model
	return s.permissionRepo.GetByID(id)
}

// List MC-IAM 권한 목록 조회 (필터 추가) - Renamed
func (s *MciamPermissionService) List(ctx context.Context, frameworkID, resourceTypeID string) ([]model.MciamPermission, error) { // Use renamed model
	return s.permissionRepo.List(frameworkID, resourceTypeID)
}

// Update MC-IAM 권한 정보 부분 업데이트 - Renamed
func (s *MciamPermissionService) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	// Add validation for updates map if needed (e.g., allowed fields)
	// The repository already prevents updating PKs and createdAt
	return s.permissionRepo.Update(id, updates)
}

// Delete MC-IAM 권한 삭제 - Renamed
func (s *MciamPermissionService) Delete(ctx context.Context, id string) error {
	// Add business logic if needed before deleting
	return s.permissionRepo.Delete(id)
}

// AssignMciamPermissionToRole 역할에 MC-IAM 권한 할당 - Renamed
func (s *MciamPermissionService) AssignMciamPermissionToRole(ctx context.Context, roleType string, roleID uint, permissionID string) error {
	// 권한 존재 여부 확인
	_, err := s.permissionRepo.GetByID(permissionID)
	if err != nil {
		// Handle specific "not found" error from repo
		return err // Return ErrPermissionNotFound or other DB error
	}

	// TODO: 역할 존재 여부 확인 (Platform or Workspace) - Requires Role Repositories
	// Example:
	// if roleType == "platform" {
	//  if _, err := s.platformRoleRepo.GetByID(roleID); err != nil { return err }
	// } else if roleType == "workspace" {
	//  if _, err := s.workspaceRoleRepo.GetByID(roleID); err != nil { return err }
	// } else {
	//  return errors.New("invalid role type")
	// }

	return s.permissionRepo.AssignMciamPermissionToRole(roleType, roleID, permissionID) // Use renamed repo method
}

// RemoveMciamPermissionFromRole 역할에서 MC-IAM 권한 제거 - Renamed
func (s *MciamPermissionService) RemoveMciamPermissionFromRole(ctx context.Context, roleType string, roleID uint, permissionID string) error {
	// No need to check existence first, repo handles it gracefully
	return s.permissionRepo.RemoveMciamPermissionFromRole(roleType, roleID, permissionID) // Use renamed repo method
}

// GetRoleMciamPermissions 역할의 MC-IAM 권한 ID 목록 조회 - Renamed
func (s *MciamPermissionService) GetRoleMciamPermissions(ctx context.Context, roleType string, roleID uint) ([]string, error) { // Return []string
	// TODO: 역할 존재 여부 확인 (Platform or Workspace)
	return s.permissionRepo.GetRoleMciamPermissions(roleType, roleID) // Use renamed repo method
}

// Note: Need similar service for CSP permissions and role-csp mappings later.
