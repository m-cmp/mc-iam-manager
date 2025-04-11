package service

import (
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
)

// WorkspaceService 워크스페이스 관리 서비스
type WorkspaceService struct {
	workspaceRepo *repository.WorkspaceRepository
	projectRepo   *repository.ProjectRepository // Needed for association checks
}

// NewWorkspaceService 새 WorkspaceService 인스턴스 생성
func NewWorkspaceService(workspaceRepo *repository.WorkspaceRepository, projectRepo *repository.ProjectRepository) *WorkspaceService {
	return &WorkspaceService{
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

// Update 워크스페이스 정보 부분 업데이트
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
	// Check if workspace exists before delete (optional, repo handles it)
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
	_, errPr := s.projectRepo.GetByID(projectID)
	if errPr != nil {
		return errPr // Project not found or DB error
	}

	return s.workspaceRepo.AddProjectAssociation(workspaceID, projectID)
}

// RemoveProjectFromWorkspace 워크스페이스에서 프로젝트 연결 제거
func (s *WorkspaceService) RemoveProjectFromWorkspace(workspaceID, projectID uint) error {
	// Existence checks are optional as repo method might handle it gracefully
	return s.workspaceRepo.RemoveProjectAssociation(workspaceID, projectID)
}
