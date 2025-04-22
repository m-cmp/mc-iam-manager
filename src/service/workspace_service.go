package service

import (
	"fmt"

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

// GetProjectsByWorkspaceID 특정 워크스페이스에 연결된 프로젝트 목록 조회
func (s *WorkspaceService) GetProjectsByWorkspaceID(workspaceID uint) ([]model.Project, error) {
	// First, check if the workspace exists
	_, err := s.workspaceRepo.GetByID(workspaceID)
	if err != nil {
		return nil, err // Return error if workspace not found or DB error
	}
	// Call the repository method to get associated projects
	return s.workspaceRepo.FindProjectsByWorkspaceID(workspaceID)
}

// UserWithRoles 워크스페이스 내 사용자 및 역할 정보를 담는 구조체
type UserWithRoles struct {
	User  model.User            `json:"user"`
	Roles []model.WorkspaceRole `json:"roles"`
}

// GetUsersAndRolesByWorkspaceID 특정 워크스페이스에 속한 사용자 및 역할 목록 조회
func (s *WorkspaceService) GetUsersAndRolesByWorkspaceID(workspaceID uint) ([]UserWithRoles, error) {
	// 1. Check if workspace exists
	_, err := s.workspaceRepo.GetByID(workspaceID)
	if err != nil {
		return nil, err // Return ErrWorkspaceNotFound or other DB error
	}

	// 2. Get the raw mapping data from the repository
	userWorkspaceRoles, err := s.workspaceRepo.FindUsersAndRolesByWorkspaceID(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to find users and roles for workspace %d: %w", workspaceID, err)
	}

	// 3. Process the data into the desired output format (group roles by user)
	userRolesMap := make(map[uint]*UserWithRoles) // Key: User DB ID

	for _, uwr := range userWorkspaceRoles {
		// Ensure User and WorkspaceRole were preloaded correctly
		if uwr.User.ID == 0 || uwr.WorkspaceRole.ID == 0 {
			// Log a warning or skip this entry if data is incomplete
			fmt.Printf("Warning: Incomplete data in UserWorkspaceRole mapping (UserID: %d, RoleID: %d)\n", uwr.UserID, uwr.WorkspaceRoleID)
			continue
		}

		userID := uwr.User.ID
		if _, exists := userRolesMap[userID]; !exists {
			userRolesMap[userID] = &UserWithRoles{
				User:  uwr.User, // Assumes User object is fully populated by preload
				Roles: []model.WorkspaceRole{},
			}
		}
		userRolesMap[userID].Roles = append(userRolesMap[userID].Roles, uwr.WorkspaceRole)
	}

	// Convert map to slice
	result := make([]UserWithRoles, 0, len(userRolesMap))
	for _, userInfo := range userRolesMap {
		result = append(result, *userInfo)
	}

	return result, nil
}
