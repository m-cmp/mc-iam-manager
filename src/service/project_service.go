package service

import (
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
)

// ProjectService 프로젝트 관리 서비스
type ProjectService struct {
	projectRepo   *repository.ProjectRepository
	workspaceRepo *repository.WorkspaceRepository // Needed for association checks
}

// NewProjectService 새 ProjectService 인스턴스 생성
func NewProjectService(projectRepo *repository.ProjectRepository, workspaceRepo *repository.WorkspaceRepository) *ProjectService {
	return &ProjectService{
		projectRepo:   projectRepo,
		workspaceRepo: workspaceRepo,
	}
}

// Create 프로젝트 생성
func (s *ProjectService) Create(project *model.Project) error {
	// TODO: Add validation logic if needed (e.g., check NsId uniqueness within workspace if required)
	return s.projectRepo.Create(project)
}

// List 모든 프로젝트 조회
func (s *ProjectService) List() ([]model.Project, error) {
	return s.projectRepo.List()
}

// GetByID ID로 프로젝트 조회
func (s *ProjectService) GetByID(id uint) (*model.Project, error) {
	return s.projectRepo.GetByID(id)
}

// GetByName 이름으로 프로젝트 조회
func (s *ProjectService) GetByName(name string) (*model.Project, error) {
	return s.projectRepo.GetByName(name)
}

// Update 프로젝트 정보 부분 업데이트
func (s *ProjectService) Update(id uint, updates map[string]interface{}) error {
	// TODO: Add validation logic if needed
	_, err := s.projectRepo.GetByID(id)
	if err != nil {
		return err
	}
	return s.projectRepo.Update(id, updates)
}

// Delete 프로젝트 삭제
func (s *ProjectService) Delete(id uint) error {
	_, err := s.projectRepo.GetByID(id)
	if err != nil {
		return err
	}
	return s.projectRepo.Delete(id)
}

// AddWorkspaceToProject 프로젝트에 워크스페이스 연결
func (s *ProjectService) AddWorkspaceToProject(projectID, workspaceID uint) error {
	_, errPr := s.projectRepo.GetByID(projectID)
	if errPr != nil {
		return errPr
	}
	_, errWs := s.workspaceRepo.GetByID(workspaceID)
	if errWs != nil {
		return errWs
	}
	return s.projectRepo.AddWorkspaceAssociation(projectID, workspaceID)
}

// RemoveWorkspaceFromProject 프로젝트에서 워크스페이스 연결 제거
func (s *ProjectService) RemoveWorkspaceFromProject(projectID, workspaceID uint) error {
	return s.projectRepo.RemoveWorkspaceAssociation(projectID, workspaceID)
}
