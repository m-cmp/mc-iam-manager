package repository

import (
	"errors"

	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

var (
	ErrWorkspaceNotFound = errors.New("workspace not found")
)

// WorkspaceRepository 워크스페이스 데이터 관리
type WorkspaceRepository struct {
	db *gorm.DB
}

// NewWorkspaceRepository 새 WorkspaceRepository 인스턴스 생성
func NewWorkspaceRepository(db *gorm.DB) *WorkspaceRepository {
	return &WorkspaceRepository{db: db}
}

// Create 워크스페이스 생성
func (r *WorkspaceRepository) Create(workspace *model.Workspace) error {
	return r.db.Create(workspace).Error
}

// List 모든 워크스페이스 조회 (프로젝트 정보 포함)
func (r *WorkspaceRepository) List() ([]model.Workspace, error) {
	var workspaces []model.Workspace
	// Preload Projects to fetch associated projects
	if err := r.db.Preload("Projects").Find(&workspaces).Error; err != nil {
		return nil, err
	}
	return workspaces, nil
}

// GetByID ID로 워크스페이스 조회 (프로젝트 정보 포함)
func (r *WorkspaceRepository) GetByID(id uint) (*model.Workspace, error) {
	var workspace model.Workspace
	// Preload Projects to fetch associated projects
	if err := r.db.Preload("Projects").First(&workspace, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkspaceNotFound
		}
		return nil, err
	}
	return &workspace, nil
}

// GetByName 이름으로 워크스페이스 조회 (프로젝트 정보 포함)
func (r *WorkspaceRepository) GetByName(name string) (*model.Workspace, error) {
	var workspace model.Workspace
	// Preload Projects and find by name
	if err := r.db.Preload("Projects").Where("name = ?", name).First(&workspace).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkspaceNotFound
		}
		return nil, err
	}
	return &workspace, nil
}

// Update 워크스페이스 정보 부분 업데이트
func (r *WorkspaceRepository) Update(id uint, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return errors.New("no fields provided for update")
	}
	result := r.db.Model(&model.Workspace{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrWorkspaceNotFound
	}
	return nil
}

// Delete 워크스페이스 삭제
func (r *WorkspaceRepository) Delete(id uint) error {
	// GORM will automatically handle deleting associations in the join table
	// due to the ON DELETE CASCADE constraint in the DB schema.
	result := r.db.Delete(&model.Workspace{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrWorkspaceNotFound
	}
	return nil
}

// AddProjectAssociation 워크스페이스에 프로젝트 연결 추가
func (r *WorkspaceRepository) AddProjectAssociation(workspaceID, projectID uint) error {
	// GORM's Association Append handles M:N relationships
	var workspace model.Workspace
	workspace.ID = workspaceID // Need the ID to specify the workspace
	var project model.Project
	project.ID = projectID // Need the ID of the project to associate

	// Check if workspace and project exist first (optional but recommended)
	// ...

	// Append the association
	err := r.db.Model(&workspace).Association("Projects").Append(&project)
	if err != nil {
		// Handle potential errors like duplicate entry if association already exists
		// GORM might handle duplicates gracefully depending on DB driver
		return err
	}
	return nil
}

// RemoveProjectAssociation 워크스페이스에서 프로젝트 연결 제거
func (r *WorkspaceRepository) RemoveProjectAssociation(workspaceID, projectID uint) error {
	var workspace model.Workspace
	workspace.ID = workspaceID
	var project model.Project
	project.ID = projectID

	// Delete the association
	err := r.db.Model(&workspace).Association("Projects").Delete(&project)
	if err != nil {
		return err
	}
	// GORM's Delete association might not return error if association didn't exist
	return nil
}

// FindProjectsByWorkspaceID 특정 워크스페이스에 연결된 프로젝트 목록 조회
func (r *WorkspaceRepository) FindProjectsByWorkspaceID(workspaceID uint) ([]model.Project, error) {
	var workspace model.Workspace
	workspace.ID = workspaceID
	var projects []model.Project

	// Use GORM's Association to find related projects
	err := r.db.Model(&workspace).Association("Projects").Find(&projects)
	if err != nil {
		// Check if the error is because the workspace itself wasn't found
		// Although the service layer already checks this, double-checking might be useful
		// Or just return the error as is.
		return nil, err
	}

	// If the workspace exists but has no projects, Find will return an empty slice and nil error.
	return projects, nil
}

// FindUsersAndRolesByWorkspaceID 특정 워크스페이스에 속한 사용자 및 역할 목록 조회
func (r *WorkspaceRepository) FindUsersAndRolesByWorkspaceID(workspaceID uint) ([]model.UserWorkspaceRole, error) {
	var userWorkspaceRoles []model.UserWorkspaceRole

	// Find all UserWorkspaceRole entries where the associated WorkspaceRole's WorkspaceID matches.
	// We need to join with WorkspaceRole table to filter by workspaceID.
	// Then preload User and WorkspaceRole.
	err := r.db.Joins("JOIN mcmp_workspace_roles ON mcmp_workspace_roles.id = mcmp_user_workspace_roles.workspace_role_id").
		Where("mcmp_user_workspace_roles.workspace_id = ?", workspaceID).
		Preload("User").          // Preload the User associated with the mapping
		Preload("WorkspaceRole"). // Preload the WorkspaceRole associated with the mapping
		Find(&userWorkspaceRoles).Error

	if err != nil {
		// Don't return ErrWorkspaceNotFound here, as an empty result is valid.
		// Return other DB errors.
		return nil, err
	}

	return userWorkspaceRoles, nil
}
