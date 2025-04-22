package repository

import (
	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

type WorkspaceRoleRepository struct {
	db *gorm.DB
}

func NewWorkspaceRoleRepository(db *gorm.DB) *WorkspaceRoleRepository {
	return &WorkspaceRoleRepository{
		db: db,
	}
}

func (r *WorkspaceRoleRepository) List() ([]model.WorkspaceRole, error) {
	var roles []model.WorkspaceRole
	if err := r.db.Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *WorkspaceRoleRepository) GetByID(id uint) (*model.WorkspaceRole, error) {
	var role model.WorkspaceRole
	if err := r.db.First(&role, id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *WorkspaceRoleRepository) Create(role *model.WorkspaceRole) error {
	return r.db.Create(role).Error
}

func (r *WorkspaceRoleRepository) Update(role *model.WorkspaceRole) error {
	return r.db.Save(role).Error
}

func (r *WorkspaceRoleRepository) Delete(id uint) error {
	// Consider deleting associated user roles as well, or handle via DB constraints
	return r.db.Delete(&model.WorkspaceRole{}, id).Error
}

// AssignRoleToUser 사용자에게 워크스페이스 역할 할당 (mcmp_user_workspace_roles 테이블)
func (r *WorkspaceRoleRepository) AssignRoleToUser(userID, roleID, workspaceID uint) error { // Add workspaceID parameter
	mapping := model.UserWorkspaceRole{
		UserID:          userID,
		WorkspaceID:     workspaceID, // Set WorkspaceID
		WorkspaceRoleID: roleID,
	}
	// Create the mapping record. Handle potential errors like duplicate entry.
	// Using FirstOrCreate or similar might be better to avoid errors if mapping already exists.
	// For simplicity, just Create for now. Ensure DB constraints handle duplicates.
	return r.db.Create(&mapping).Error
}

// RemoveRoleFromUser 사용자에게서 워크스페이스 역할 제거 (mcmp_user_workspace_roles 테이블)
func (r *WorkspaceRoleRepository) RemoveRoleFromUser(userID, roleID, workspaceID uint) error { // Add workspaceID parameter
	// Delete the specific mapping record using the composite key
	result := r.db.Where("user_id = ? AND workspace_id = ? AND workspace_role_id = ?", userID, workspaceID, roleID).Delete(&model.UserWorkspaceRole{})
	if result.Error != nil {
		return result.Error
	}
	// Optionally check result.RowsAffected if you need to know if a record was actually deleted
	return nil
}
