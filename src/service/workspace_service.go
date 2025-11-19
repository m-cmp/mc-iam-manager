package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm"
)

// WorkspaceService 워크스페이스 관리 서비스
type WorkspaceService struct {
	db                *gorm.DB
	workspaceRepo     *repository.WorkspaceRepository
	roleRepo          *repository.RoleRepository
	userRepo          *repository.UserRepository
	workspaceRoleRepo *repository.WorkspaceRoleRepository
	projectRepo       *repository.ProjectRepository
}

// NewWorkspaceService 새 WorkspaceService 인스턴스 생성
func NewWorkspaceService(db *gorm.DB) *WorkspaceService {
	// Initialize repositories internally
	workspaceRepo := repository.NewWorkspaceRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	userRepo := repository.NewUserRepository(db)
	workspaceRoleRepo := repository.NewWorkspaceRoleRepository(db)
	return &WorkspaceService{
		db:                db,
		workspaceRepo:     workspaceRepo,
		projectRepo:       projectRepo,
		roleRepo:          roleRepo,
		userRepo:          userRepo,
		workspaceRoleRepo: workspaceRoleRepo,
	}
}

// Create 워크스페이스 생성
func (s *WorkspaceService) CreateWorkspace(workspace *model.Workspace) error {
	// 이름 중복 체크
	existingWorkspace, err := s.workspaceRepo.FindWorkspaceByName(workspace.Name)
	if err == nil && existingWorkspace != nil {
		return fmt.Errorf("workspace with name '%s' already exists", workspace.Name)
	}
	if err != nil && err.Error() != "workspace not found" {
		return err
	}

	return s.workspaceRepo.CreateWorkspace(workspace)
}

// UpdateWorkspace 워크스페이스 정보를 수정합니다.
func (s *WorkspaceService) UpdateWorkspace(workspace *model.Workspace) error {
	_, err := s.workspaceRepo.FindWorkspaceByID(workspace.ID) // 실제 존재하는 workspace인지 확인
	if err != nil {
		return err // Return error if not found or other DB error
	}
	updates := map[string]interface{}{}
	if workspace.Name != "" {
		updates["name"] = workspace.Name
	}
	if workspace.Description != "" {
		updates["description"] = workspace.Description
	}

	return s.workspaceRepo.UpdateWorkspace(workspace.ID, updates)
}

// Delete 워크스페이스 삭제
func (s *WorkspaceService) DeleteWorkspace(workspaceID uint) error {
	// 1. 워크스페이스 존재 확인
	workspace, err := s.workspaceRepo.FindWorkspaceProjectsByWorkspaceID(workspaceID)
	if err != nil {
		return err
	}

	// 2. 할당된 프로젝트 확인
	if len(workspace.Projects) > 0 {
		projectNames := make([]string, len(workspace.Projects))
		for i, p := range workspace.Projects {
			projectNames[i] = p.Name
		}
		return fmt.Errorf("워크스페이스에 연결된 프로젝트가 있습니다: %s. 먼저 모든 프로젝트를 제거하세요",
			strings.Join(projectNames, ", "))
	}

	// 3. 삭제 실행
	return s.workspaceRepo.DeleteWorkspace(workspaceID)
}

// List 모든 워크스페이스 조회 workspace 목록만 return
func (s *WorkspaceService) ListWorkspaces(req *model.WorkspaceFilterRequest) ([]*model.Workspace, error) {
	return s.workspaceRepo.FindWorkspaces(req)
}

// ListWorkspacesProjects 모든 워크스페이스와 연관된 프로젝트 목록을 조회합니다.
func (s *WorkspaceService) ListWorkspacesProjects(req *model.WorkspaceFilterRequest) ([]*model.WorkspaceWithProjects, error) {
	workspaces, err := s.workspaceRepo.FindWorkspacesProjects(req)
	if err != nil {
		return nil, err
	}

	// var result []model.WorkspaceWithProjects
	// for _, workspace := range workspaces {
	// 	projects, err := s.workspaceRepo.FindProjectsByWorkspaceID(workspace.ID)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	result = append(result, model.WorkspaceWithProjects{
	// 		ID:          workspace.ID,
	// 		Name:        workspace.Name,
	// 		Description: workspace.Description,
	// 		CreatedAt:   workspace.CreatedAt,
	// 		UpdatedAt:   workspace.UpdatedAt,
	// 		Projects:    projects,
	// 	})
	// }

	// return result, nil
	return workspaces, nil
}

func (s *WorkspaceService) GetWorkspaceProjectsByWorkspaceId(workspaceID uint) (*model.WorkspaceWithProjects, error) {
	return s.workspaceRepo.FindWorkspaceProjectsByWorkspaceID(workspaceID)
}

// GetByID ID로 워크스페이스 조회
func (s *WorkspaceService) GetWorkspaceByID(workspaceID uint) (*model.Workspace, error) {
	return s.workspaceRepo.FindWorkspaceByID(workspaceID)
}

// GetByName 이름으로 워크스페이스 조회
func (s *WorkspaceService) GetWorkspaceByName(workspaceName string) (*model.Workspace, error) {
	return s.workspaceRepo.FindWorkspaceByName(workspaceName)
}

// 유저에게 할당된 워크스페이스 목록
func (s *WorkspaceService) ListWorkspacesByUserID(userID uint) ([]*model.WorkspaceWithUsersAndRoles, error) {
	return s.userRepo.FindWorkspacesByUserID(userID)
}

// GetUsersByWorkspaceID 워크스페이스에 속한 사용자 목록을 조회합니다.
func (s *WorkspaceService) ListWorkspaceUsers(req model.WorkspaceFilterRequest) ([]*model.WorkspaceWithUsersAndRoles, error) {
	// 워크스페이스 존재 여부 확인
	workspaceUsers, err := s.roleRepo.FindWorkspaceWithUsersRoles(req)
	if err != nil {
		return nil, err
	}

	return workspaceUsers, nil
}

// GetUsersByWorkspaceID 워크스페이스에 속한 사용자와 역할 목록을 조회합니다.
func (s *WorkspaceService) ListWorkspaceUsersAndRoles(req model.WorkspaceFilterRequest) ([]*model.UserWorkspaceRole, error) {
	// 워크스페이스 존재 여부 확인
	workspaceUserRoles, err := s.roleRepo.FindUsersAndRolesWithWorkspaces(req)
	if err != nil {
		return nil, err
	}

	return workspaceUserRoles, nil
}

// AddProjectToWorkspace 워크스페이스에 프로젝트 연결
// 워크스페이스와 프로젝트가 존재하는지 확인하고 연결
func (s *WorkspaceService) AddProjectToWorkspace(workspaceID, projectID uint) error {
	// Check if both workspace and project exist
	_, errWs := s.workspaceRepo.FindWorkspaceByID(workspaceID)
	if errWs != nil {
		return errWs // Workspace not found or DB error
	}

	_, errProj := s.projectRepo.FindProjectByProjectID(projectID)
	if errProj != nil {
		return errProj // Project not found or DB error
	}

	// Add the association
	return s.workspaceRepo.AddProjectAssociation(workspaceID, projectID)
}

// RemoveProjectFromWorkspace 워크스페이스에서 프로젝트 제거
func (s *WorkspaceService) RemoveProjectFromWorkspace(workspaceID, projectID uint) error {
	// 워크스페이스에서 프로젝트 제거
	// 기본 workspace 포함 모든 workspace에서 제거 가능
	// 프로젝트는 미할당 상태가 될 수 있음
	if err := s.workspaceRepo.RemoveProjectAssociation(workspaceID, projectID); err != nil {
		return fmt.Errorf("워크스페이스 연결 제거에 실패했습니다: %v", err)
	}

	return nil
}

// AssignRole 워크스페이스에 사용자 역할 할당
func (s *WorkspaceService) AssignWorkspaceRole(userID, workspaceID, roleID uint) error {
	// 워크스페이스 존재 여부 확인
	workspace, err := s.workspaceRepo.FindWorkspaceByID(workspaceID)
	if err != nil {
		return fmt.Errorf("워크스페이스를 찾을 수 없습니다: %w", err)
	}
	if workspace == nil {
		return fmt.Errorf("워크스페이스를 찾을 수 없습니다")
	}

	// 역할 존재 여부 확인
	role, err := s.roleRepo.FindRoleByRoleID(roleID, constants.RoleTypeWorkspace)
	if err != nil {
		return fmt.Errorf("역할을 찾을 수 없습니다: %w", err)
	}
	if role == nil {
		return fmt.Errorf("역할을 찾을 수 없습니다")
	}

	// 워크스페이스 역할 타입 검증
	roleSub, err := s.roleRepo.FindRoleSubByRoleIDAndType(roleID, constants.RoleTypeWorkspace)
	if err != nil {
		return fmt.Errorf("워크스페이스 역할이 아닙니다")
	}
	if roleSub == nil {
		return fmt.Errorf("워크스페이스 역할이 아닙니다")
	}

	// 역할 할당
	return s.roleRepo.AssignWorkspaceRole(userID, workspaceID, roleID)
}

// AddUserToWorkspace 워크스페이스에 사용자를 추가합니다.
func (s *WorkspaceService) AddUserToWorkspace(workspaceID, userID uint) error {
	// 워크스페이스 존재 여부 확인
	_, err := s.workspaceRepo.FindWorkspaceByID(workspaceID)
	if err != nil {
		return err
	}

	// 사용자 존재 여부 확인
	user, err := s.userRepo.FindUserByID(userID)
	if err != nil {
		return errors.New("user not found")
	}
	if user == nil {
		return errors.New("user not found")
	}

	// 이미 워크스페이스에 속해있는지 확인
	isInWorkspace, err := s.roleRepo.CheckUserInWorkspace(workspaceID, userID)
	if err != nil {
		return err
	}
	if isInWorkspace {
		return errors.New("user already in workspace")
	}

	// 기본 워크스페이스 역할 조회
	defaultRole, err := s.roleRepo.FindDefaultWorkspaceRole()
	if err != nil {
		return err
	}

	// 사용자를 워크스페이스에 추가
	uwr := model.UserWorkspaceRole{
		UserID:      userID,
		WorkspaceID: workspaceID,
		RoleID:      defaultRole.ID,
	}
	if err := s.userRepo.CreateUserWorkspaceRole(&uwr); err != nil {
		return err
	}

	return nil
}

// RemoveUserFromWorkspace 워크스페이스에서 사용자를 제거합니다.
func (s *WorkspaceService) RemoveUserFromWorkspace(workspaceID, userID uint) error {
	// 워크스페이스 존재 여부 확인
	_, err := s.workspaceRepo.FindWorkspaceByID(workspaceID)
	if err != nil {
		return err
	}

	// 사용자 존재 여부 확인
	user, err := s.userRepo.FindUserByID(userID)
	if err != nil {
		return errors.New("user not found")
	}
	if user == nil {
		return errors.New("user not found")
	}

	// 워크스페이스에서 사용자 제거
	if err := s.userRepo.DeleteUserWorkspaceRole(workspaceID, userID); err != nil {
		return err
	}

	return nil
}

// GetUserWorkspaceAndWorkspaceRoles 사용자의 워크스페이스와 역할 정보를 조회합니다.
func (s *WorkspaceService) ListUserWorkspaceAndWorkspaceRoles(ctx context.Context, userID uint) ([]*model.UserWorkspaceRole, error) {
	// 사용자 존재 여부 확인
	_, err := s.userRepo.FindUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("사용자를 찾을 수 없습니다: %w", err)
	}

	// 사용자의 워크스페이스 역할 조회
	userWorkspaceRoles, err := s.userRepo.FindWorkspaceAndWorkspaceRolesByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("워크스페이스 역할 조회 실패: %w", err)
	}

	return userWorkspaceRoles, nil
}

// GetUserWorkspaceRoles 사용자의 워크스페이스 역할 목록을 조회합니다.
func (s *UserService) ListUserWorkspaceRoles(userID uint) ([]*model.UserWorkspaceRole, error) {
	// 사용자 존재 여부 확인
	_, err := s.userRepo.FindUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("사용자를 찾을 수 없습니다: %w", err)
	}

	// 사용자의 워크스페이스 역할 조회
	userWorkspaceRoles, err := s.userRepo.FindWorkspaceAndWorkspaceRolesByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("워크스페이스 역할 조회 실패: %w", err)
	}

	return userWorkspaceRoles, nil
}

// 한 유저는 1개의 워크스페이스에서 1개의 역할만 가질 수 있음.
func (s *UserService) GetUserWorkspaceRoleByWorkspaceID(ctx context.Context, userID uint, workspaceID uint) (*model.UserWorkspaceRole, error) {
	// 사용자 존재 여부 확인
	_, err := s.userRepo.FindUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("사용자를 찾을 수 없습니다: %w", err)
	}

	// 사용자의 워크스페이스 역할 조회
	userWorkspaceRole, err := s.userRepo.FindUserRoleInWorkspace(userID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("워크스페이스 역할 조회 실패: %w", err)
	}

	return userWorkspaceRole, nil
}

func (s *WorkspaceService) ListProjectsByWorkspaceID(workspaceID uint) ([]*model.Project, error) {
	return s.workspaceRepo.FindProjectsByWorkspaceID(workspaceID)
}
