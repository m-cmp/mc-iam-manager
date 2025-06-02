package repository

import (
	"errors"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/util"
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
func (r *WorkspaceRepository) CreateWorkspace(workspace *model.Workspace) error {
	return r.db.Create(workspace).Error
}

// Update 워크스페이스 정보 부분 업데이트
func (r *WorkspaceRepository) UpdateWorkspace(id uint, updates map[string]interface{}) error {
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
func (r *WorkspaceRepository) DeleteWorkspace(id uint) error {
	// GORM will automatically handle deleting associations in the join table
	// due to the ON DELETE CASCADE constraint in the DB schema.
	result := r.db.Delete(&model.Workspace{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrWorkspaceNotFound
	}
	return nil
}

// Find 모든 워크스페이스를 조회합니다. workspace 목록만 return
func (r *WorkspaceRepository) FindWorkspaces(req *model.WorkspaceFilterRequest) ([]*model.Workspace, error) {
	var workspaces []*model.Workspace

	// filter 조건이 있으면 조건에 맞는 워크스페이스 조회
	// 쿼리 빌더를 사용하여 기본 쿼리 생성
	query := r.db.Model(&model.Workspace{})

	if req.WorkspaceID != "" {
		workspaceIdInt, err := util.StringToUint(req.WorkspaceID)
		if err != nil {
			return nil, err
		}
		query = query.Where("id = ?", workspaceIdInt)
	}

	if req.WorkspaceName != "" {
		query = query.Where("name = ?", req.WorkspaceName)
	}

	// ProjectID로 필터링
	if req.ProjectID != "" {
		projectIdInt, err := util.StringToUint(req.ProjectID)
		if err != nil {
			return nil, err
		}
		query = query.Joins("JOIN mcmp_workspace_projects ON mcmp_workspaces.id = mcmp_workspace_projects.workspace_id").
			Where("mcmp_workspace_projects.project_id = ?", projectIdInt)
	}

	// UserID로 필터링
	if req.UserID != "" {
		userIdInt, err := util.StringToUint(req.UserID)
		if err != nil {
			return nil, err
		}
		query = query.Joins("JOIN mcmp_user_workspace_roles ON mcmp_workspaces.id = mcmp_user_workspace_roles.workspace_id").
			Where("mcmp_user_workspace_roles.user_id = ?", userIdInt)
	}

	if err := query.Find(&workspaces).Error; err != nil {
		return nil, err
	}
	return workspaces, nil
}

// FindByID 워크스페이스를 ID로 조회. 단건조회
func (r *WorkspaceRepository) FindWorkspaceByID(workspaceId uint) (*model.Workspace, error) {
	var workspace model.Workspace
	if err := r.db.First(&workspace, workspaceId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &workspace, nil
}

// FindWorkspaceByName 이름으로 워크스페이스 조회 (프로젝트 정보 포함)
func (r *WorkspaceRepository) FindWorkspaceByName(workspaceName string) (*model.Workspace, error) {
	var workspace *model.Workspace
	// Preload Projects and find by name
	if err := r.db.Preload("Projects").Where("name = ?", workspaceName).First(&workspace).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkspaceNotFound
		}
		return nil, err
	}
	return workspace, nil
}

// FindWorkspacesProjects 모든 워크스페이스 조회 (프로젝트 정보 포함)
func (r *WorkspaceRepository) FindWorkspacesProjects(req model.WorkspaceProjectFilterRequest) ([]*model.WorkspaceWithProjects, error) {
	var workspacesProjects []*model.WorkspaceWithProjects
	// Preload Projects to fetch associated projects
	query := r.db.Model(&model.Workspace{})

	if req.WorkspaceID != "" {
		workspaceIdInt, err := util.StringToUint(req.WorkspaceID)
		if err != nil {
			return nil, err
		}
		query = query.Where("id = ?", workspaceIdInt)
	}

	if req.ProjectID != "" {
		projectIdInt, err := util.StringToUint(req.ProjectID)
		if err != nil {
			return nil, err
		}
		query = query.Joins("JOIN mcmp_workspace_projects ON mcmp_workspaces.id = mcmp_workspace_projects.workspace_id").
			Where("mcmp_workspace_projects.project_id = ?", projectIdInt)
	}

	if err := query.Preload("Projects").Find(&workspacesProjects).Error; err != nil {
		return nil, err
	}
	return workspacesProjects, nil
}

// FindWorkspaceProjectsByWorkspaceID ID로 워크스페이스 조회 (프로젝트 정보 포함)
func (r *WorkspaceRepository) FindWorkspaceProjectsByWorkspaceID(id uint) (*model.WorkspaceWithProjects, error) {
	var workspace *model.WorkspaceWithProjects
	// Preload Projects to fetch associated projects
	if err := r.db.Preload("Projects").First(&workspace, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkspaceNotFound
		}
		return nil, err
	}
	return workspace, nil
}

// AddProjectAssociation 워크스페이스에 프로젝트 연결 추가
func (r *WorkspaceRepository) AddProjectAssociation(workspaceID, projectID uint) error {
	workspace := &model.Workspace{ID: workspaceID}
	project := &model.Project{ID: projectID}

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
	workspace := &model.Workspace{ID: workspaceID}
	project := &model.Project{ID: projectID}

	// Delete the association
	err := r.db.Model(&workspace).Association("Projects").Delete(&project)
	if err != nil {
		return err
	}
	// GORM's Delete association might not return error if association didn't exist
	return nil
}

// FindProjectsByWorkspaceID 특정 워크스페이스에 연결된 프로젝트 목록 조회 ( Project 목록만 return)
func (r *WorkspaceRepository) FindProjectsByWorkspaceID(workspaceID uint) ([]*model.Project, error) {
	workspace := &model.Workspace{ID: workspaceID}
	var projects []*model.Project

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
