package service

import (
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm" // Import gorm
)

// WorkspaceService 워크스페이스 관리 서비스
type WorkspaceService struct {
	workspaceRepo *repository.WorkspaceRepository
	projectRepo   *repository.ProjectRepository // Needed for association checks
	db            *gorm.DB                      // Add DB field
}

// NewWorkspaceService 새 WorkspaceService 인스턴스 생성
func NewWorkspaceService(db *gorm.DB) *WorkspaceService { // Accept only db
	// Initialize repositories internally
	workspaceRepo := repository.NewWorkspaceRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	return &WorkspaceService{
		db:            db, // Store db
		workspaceRepo: workspaceRepo,
		projectRepo:   projectRepo,
	}
}

// Create 워크스페이스 생성
func (s *WorkspaceService) Create(workspace *model.Workspace) error {
	// TODO: Add validation logic if needed
	return s.workspaceRepo.Create(workspace)
}

// List 모든 워크스페이스 조회
func (s *WorkspaceService) List() ([]model.Workspace, error) {
	return s.workspaceRepo.List()
}

// GetByID ID로 워크스페이스 조회
func (s *WorkspaceService) GetByID(id uint) (*model.Workspace, error) {
	return s.workspaceRepo.GetByID(id)
}

// GetByName 이름으로 워크스페이스 조회
func (s *WorkspaceService) GetByName(name string) (*model.Workspace, error) {
	return s.workspaceRepo.GetByName(name)
}

// Update 워크스페이스 정보 업데이트
func (s *WorkspaceService) Update(id uint, updates map[string]interface{}) error {
	// TODO: Add validation logic if needed
	// Check if workspace exists before update (optional, repo handles it)
	_, err := s.workspaceRepo.GetByID(id)
	if err != nil {
		return err // Return error if not found or other DB error
	}
	return s.workspaceRepo.Update(id, updates)
}

// Delete 워크스페이스 삭제
func (s *WorkspaceService) Delete(id uint) error {
	// Check if workspace exists
	_, err := s.workspaceRepo.GetByID(id)
	if err != nil {
		return err
	}
	return s.workspaceRepo.Delete(id)
}

// AddProjectToWorkspace 워크스페이스에 프로젝트 연결
func (s *WorkspaceService) AddProjectToWorkspace(workspaceID, projectID uint) error {
	// Check if both workspace and project exist
	_, errWs := s.workspaceRepo.GetByID(workspaceID)
	if errWs != nil {
		return errWs // Workspace not found or DB error
	}
	// TODO: Check if project exists using projectRepo
	// _, errProj := s.projectRepo.GetByID(projectID)
	// if errProj != nil {
	//     return errProj // Project not found or DB error
	// }

	// Add the association
	return s.workspaceRepo.AddProjectAssociation(workspaceID, projectID)
}

// RemoveProjectFromWorkspace 워크스페이스에서 프로젝트 연결 해제
func (s *WorkspaceService) RemoveProjectFromWorkspace(workspaceID, projectID uint) error {
	// Check if both workspace and project exist
	_, errWs := s.workspaceRepo.GetByID(workspaceID)
	if errWs != nil {
		return errWs // Workspace not found or DB error
	}
	// TODO: Check if project exists using projectRepo
	// _, errProj := s.projectRepo.GetByID(projectID)
	// if errProj != nil {
	//     return errProj // Project not found or DB error
	// }

	// Remove the association
	return s.workspaceRepo.RemoveProjectAssociation(workspaceID, projectID)
}

// GetProjectsByWorkspaceID 워크스페이스에 연결된 프로젝트 목록 조회
func (s *WorkspaceService) GetProjectsByWorkspaceID(workspaceID uint) ([]model.Project, error) {
	// First, check if the workspace exists
	_, err := s.workspaceRepo.GetByID(workspaceID)
	if err != nil {
		return nil, err // Return error if workspace not found or DB error
	}

	// Then, find the projects associated with this workspace
	projects, err := s.workspaceRepo.FindProjectsByWorkspaceID(workspaceID)
	if err != nil {
		return nil, err
	}

	// Return empty slice if no projects found
	if projects == nil {
		projects = []model.Project{}
	}
	return projects, nil
}

// UserWithRoles 워크스페이스 내 사용자 및 역할 정보를 담는 구조체
type UserWithRoles struct {
	User  model.User            `json:"user"`
	Roles []model.WorkspaceRole `json:"roles"`
}

// GetUsersAndRolesByWorkspaceID 워크스페이스에 속한 사용자와 역할 조회
func (s *WorkspaceService) GetUsersAndRolesByWorkspaceID(workspaceID uint) ([]model.UserWorkspaceRole, error) {
	// Get all user-workspace-role associations for the workspace
	var uwrs []model.UserWorkspaceRole
	if err := s.db.Where("workspace_id = ?", workspaceID).Find(&uwrs).Error; err != nil {
		return nil, err
	}

	// Preload User and WorkspaceRole for each association
	for i := range uwrs {
		if err := s.db.Model(&uwrs[i]).Association("User").Find(&uwrs[i].User); err != nil {
			return nil, err
		}
		if err := s.db.Model(&uwrs[i]).Association("WorkspaceRole").Find(&uwrs[i].WorkspaceRole); err != nil {
			return nil, err
		}
	}

	return uwrs, nil
}

// GetAllWorkspaces 모든 워크스페이스를 조회합니다.
func (s *WorkspaceService) GetAllWorkspaces() ([]*model.Workspace, error) {
	return s.workspaceRepo.FindAll()
}

// CreateWorkspace 새로운 워크스페이스를 생성합니다.
func (s *WorkspaceService) CreateWorkspace(workspace *model.Workspace) error {
	return s.workspaceRepo.Create(workspace)
}

// UpdateWorkspace 워크스페이스 정보를 수정합니다.
func (s *WorkspaceService) UpdateWorkspace(workspace *model.Workspace) error {
	updates := map[string]interface{}{
		"name":        workspace.Name,
		"description": workspace.Description,
		// 필요한 다른 필드들도 추가
	}
	return s.workspaceRepo.Update(workspace.ID, updates)
}

// DeleteWorkspace 워크스페이스를 삭제합니다.
func (s *WorkspaceService) DeleteWorkspace(id uint) error {
	return s.workspaceRepo.Delete(id)
}
