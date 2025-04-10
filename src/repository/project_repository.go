package repository

import (
	"errors"

	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

var (
	ErrProjectNotFound = errors.New("project not found")
)

// ProjectRepository 프로젝트 데이터 관리
type ProjectRepository struct {
	db *gorm.DB
}

// NewProjectRepository 새 ProjectRepository 인스턴스 생성
func NewProjectRepository(db *gorm.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Create 프로젝트 생성
func (r *ProjectRepository) Create(project *model.Project) error {
	return r.db.Create(project).Error
}

// List 모든 프로젝트 조회 (워크스페이스 정보 포함)
func (r *ProjectRepository) List() ([]model.Project, error) {
	var projects []model.Project
	// Preload Workspaces to fetch associated workspaces
	if err := r.db.Preload("Workspaces").Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

// GetByID ID로 프로젝트 조회 (워크스페이스 정보 포함)
func (r *ProjectRepository) GetByID(id uint) (*model.Project, error) {
	var project model.Project
	// Preload Workspaces to fetch associated workspaces
	if err := r.db.Preload("Workspaces").First(&project, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	return &project, nil
}

// Update 프로젝트 정보 부분 업데이트
func (r *ProjectRepository) Update(id uint, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return errors.New("no fields provided for update")
	}
	// Prevent updating the ID via map
	delete(updates, "id")
	result := r.db.Model(&model.Project{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProjectNotFound
	}
	return nil
}

// Delete 프로젝트 삭제
func (r *ProjectRepository) Delete(id uint) error {
	// GORM will automatically handle deleting associations in the join table
	result := r.db.Delete(&model.Project{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProjectNotFound
	}
	return nil
}

// AddWorkspaceAssociation 프로젝트에 워크스페이스 연결 추가
func (r *ProjectRepository) AddWorkspaceAssociation(projectID, workspaceID uint) error {
	var project model.Project
	project.ID = projectID
	var workspace model.Workspace
	workspace.ID = workspaceID

	err := r.db.Model(&project).Association("Workspaces").Append(&workspace)
	if err != nil {
		return err
	}
	return nil
}

// RemoveWorkspaceAssociation 프로젝트에서 워크스페이스 연결 제거
func (r *ProjectRepository) RemoveWorkspaceAssociation(projectID, workspaceID uint) error {
	var project model.Project
	project.ID = projectID
	var workspace model.Workspace
	workspace.ID = workspaceID

	err := r.db.Model(&project).Association("Workspaces").Delete(&workspace)
	if err != nil {
		return err
	}
	return nil
}
