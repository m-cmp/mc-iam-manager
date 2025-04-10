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
