package repository

import (
	"errors"
	"log"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/util"
	"gorm.io/gorm"
)

var (
	ErrWorkspaceNotFound = errors.New("workspace not found")
)

// WorkspaceRepository workspace data management
type WorkspaceRepository struct {
	db *gorm.DB
}

// NewWorkspaceRepository create new WorkspaceRepository instance
func NewWorkspaceRepository(db *gorm.DB) *WorkspaceRepository {
	return &WorkspaceRepository{db: db}
}

// Create workspace
func (r *WorkspaceRepository) CreateWorkspace(workspace *model.Workspace) error {
	return r.db.Create(workspace).Error
}

// Update partial workspace information update
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

// Delete workspace
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

// Find retrieve all workspaces
func (r *WorkspaceRepository) FindWorkspaces(req *model.WorkspaceFilterRequest) ([]*model.Workspace, error) {
	var workspaces []*model.Workspace

	// If filter conditions exist, retrieve workspaces that match the conditions
	// Use query builder to create basic query
	query := r.db.Model(&model.Workspace{})
	log.Printf("req", req)

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

	// Filter by ProjectID
	if req.ProjectID != "" {
		projectIdInt, err := util.StringToUint(req.ProjectID)
		if err != nil {
			return nil, err
		}
		query = query.Joins("JOIN mcmp_workspace_projects ON mcmp_workspaces.id = mcmp_workspace_projects.workspace_id").
			Where("mcmp_workspace_projects.project_id = ?", projectIdInt)
	}

	// Filter by UserID
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

// FindByID retrieve workspace by ID. Single record query
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

// FindWorkspaceByName retrieve workspace by name (including project information)
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

// FindWorkspacesProjects retrieve all workspaces (including project information)
// If filtered by WorkspaceID, returns single record, otherwise returns multiple records
func (r *WorkspaceRepository) FindWorkspacesProjects(req *model.WorkspaceFilterRequest) ([]*model.WorkspaceWithProjects, error) {
	var workspacesProjects []*model.WorkspaceWithProjects
	query := r.db.Model(&model.WorkspaceWithProjects{}).
		Preload("Projects")

	// Single record query when filtering by WorkspaceID
	if req.WorkspaceID != "" {
		workspaceIdInt, err := util.StringToUint(req.WorkspaceID)
		if err != nil {
			return nil, err
		}
		var workspace model.WorkspaceWithProjects
		if err := query.Where("id = ?", workspaceIdInt).First(&workspace).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return []*model.WorkspaceWithProjects{}, nil // Return empty array
			}
			return nil, err
		}
		return []*model.WorkspaceWithProjects{&workspace}, nil
	}

	if req.ProjectID != "" {
		projectIdInt, err := util.StringToUint(req.ProjectID)
		if err != nil {
			return nil, err
		}
		query = query.Joins("JOIN mcmp_workspace_projects ON mcmp_workspaces.id = mcmp_workspace_projects.workspace_id").
			Where("mcmp_workspace_projects.project_id = ?", projectIdInt)
	}

	if err := query.Find(&workspacesProjects).Error; err != nil {
		return nil, err
	}
	return workspacesProjects, nil
}

// FindWorkspaceProjectsByWorkspaceID retrieve workspace by ID (including project information)
func (r *WorkspaceRepository) FindWorkspaceProjectsByWorkspaceID(id uint) (*model.WorkspaceWithProjects, error) {
	var workspaceProjects model.WorkspaceWithProjects
	// Preload Projects to fetch associated projects using many2many relationship
	if err := r.db.Preload("Projects").Where("id = ?", id).First(&workspaceProjects).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWorkspaceNotFound
		}
		return nil, err
	}
	return &workspaceProjects, nil
}

// AddProjectAssociation add project association to workspace
func (r *WorkspaceRepository) AddProjectAssociation(workspaceID, projectID uint) error {
	// Remove all existing workspace associations for this project (1:N relationship enforcement)
	// This ensures a project can only belong to one workspace at a time
	result := r.db.Where("project_id = ?", projectID).
		Delete(&model.WorkspaceProject{})
	if result.Error != nil {
		return result.Error
	}

	// Add new workspace association
	workspaceProject := &model.WorkspaceProject{
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
	}

	// Save directly to mcmp_workspace_projects table
	err := r.db.Save(workspaceProject).Error
	if err != nil {
		return err
	}
	return nil
}

// RemoveProjectAssociation remove project association from workspace
func (r *WorkspaceRepository) RemoveProjectAssociation(workspaceID, projectID uint) error {
	// Delete directly from mcmp_workspace_projects table
	// 기본 workspace 포함 모든 workspace에서 제거 가능
	result := r.db.Where("workspace_id = ? AND project_id = ?", workspaceID, projectID).
		Delete(&model.WorkspaceProject{})

	if result.Error != nil {
		return result.Error
	}

	// 기본 workspace로 재할당하지 않음
	// 프로젝트가 미할당 상태가 될 수 있음

	// Do not treat as error even if no records were deleted (relationship may not have existed)
	return nil
}

// FindProjectsByWorkspaceID retrieve project list connected to specific workspace (returns only Project list)
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
