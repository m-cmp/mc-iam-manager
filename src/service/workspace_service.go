package service

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm" // Import gorm
)

// WorkspaceService 워크스페이스 관리 서비스
type WorkspaceService struct {
	workspaceRepo     *repository.WorkspaceRepository
	workspaceRoleRepo *repository.WorkspaceRoleRepository
	projectRepo       *repository.ProjectRepository
	roleRepo          *repository.RoleRepository
	userRepo          *repository.UserRepository
	db                *gorm.DB
}

// NewWorkspaceService 새 WorkspaceService 인스턴스 생성
func NewWorkspaceService(db *gorm.DB) *WorkspaceService {
	// Initialize repositories internally
	workspaceRepo := repository.NewWorkspaceRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	workspaceRoleRepo := repository.NewWorkspaceRoleRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	userRepo := repository.NewUserRepository(db)
	return &WorkspaceService{
		db:                db,
		workspaceRepo:     workspaceRepo,
		projectRepo:       projectRepo,
		workspaceRoleRepo: workspaceRoleRepo,
		roleRepo:          roleRepo,
		userRepo:          userRepo,
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
	// Check if workspace exists
	workspace, err := s.workspaceRepo.FindWorkspaceProjectsByWorkspaceID(workspaceID) // 실제 존재하는 workspace인지 확인
	if err != nil {
		return err
	}

	// 워크스페이스에 연결된 프로젝트가 있는지 확인
	if len(workspace.Projects) > 0 {
		return errors.New("워크스페이스에 연결된 프로젝트가 있습니다")
	}

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
func (s *WorkspaceService) ListUsersByWorkspaceID(req model.WorkspaceFilterRequest) ([]*model.UserWorkspaceRole, error) {
	// 워크스페이스 존재 여부 확인
	workspaceUsers, err := s.roleRepo.FindUsersAndRolesWithWorkspaces(req)
	if err != nil {
		return nil, err
	}

	return workspaceUsers, nil
}

// AddProjectToWorkspace 워크스페이스에 프로젝트 연결
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
	// 프로젝트가 다른 워크스페이스에 할당되어 있는지 확인
	assignedWorkspaces, err := s.projectRepo.FindAssignedWorkspaces(projectID)
	if err != nil {
		return fmt.Errorf("워크스페이스 할당 정보를 가져오는데 실패했습니다: %v", err)
	}

	// 현재 워크스페이스에서만 할당되어 있는 경우에만 기본 워크스페이스에 할당
	if len(assignedWorkspaces) == 1 && assignedWorkspaces[0].ID == workspaceID {
		// 기본 워크스페이스 조회
		defaultWsName := os.Getenv("DEFAULT_WORKSPACE_NAME")
		if defaultWsName == "" {
			defaultWsName = "default"
		}
		defaultWs, err := s.workspaceRepo.FindWorkspaceByName(defaultWsName)
		if err != nil {
			if err.Error() == "workspace not found" {
				// 기본 워크스페이스가 없으면 생성
				newWorkspace := &model.Workspace{
					Name:        defaultWsName,
					Description: "Default workspace for automatically synced projects",
				}
				if err := s.workspaceRepo.CreateWorkspace(newWorkspace); err != nil {
					return fmt.Errorf("기본 워크스페이스 생성에 실패했습니다: %v", err)
				}
				defaultWs = newWorkspace
			} else {
				return fmt.Errorf("기본 워크스페이스를 찾는데 실패했습니다: %v", err)
			}
		}

		// 기존 워크스페이스에서 제거
		if err := s.workspaceRepo.RemoveProjectAssociation(workspaceID, projectID); err != nil {
			return fmt.Errorf("워크스페이스 연결 제거에 실패했습니다: %v", err)
		}

		// 기본 워크스페이스에 할당
		if err := s.workspaceRepo.AddProjectAssociation(defaultWs.ID, projectID); err != nil {
			return fmt.Errorf("기본 워크스페이스 할당에 실패했습니다: %v", err)
		}
	} else {
		// 다른 워크스페이스에도 할당되어 있는 경우 단순히 현재 워크스페이스에서만 제거
		if err := s.workspaceRepo.RemoveProjectAssociation(workspaceID, projectID); err != nil {
			return fmt.Errorf("워크스페이스 연결 제거에 실패했습니다: %v", err)
		}
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
	role, err := s.roleRepo.FindRoleByRoleID(roleID, model.RoleTypeWorkspace)
	if err != nil {
		return fmt.Errorf("역할을 찾을 수 없습니다: %w", err)
	}
	if role == nil {
		return fmt.Errorf("역할을 찾을 수 없습니다")
	}

	// 워크스페이스 역할 타입 검증
	var roleSub model.RoleSub
	if err := s.db.Where("role_id = ? AND role_type = ?", roleID, model.RoleTypeWorkspace).First(&roleSub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("워크스페이스 역할이 아닙니다")
		}
		return fmt.Errorf("역할 타입 확인 중 오류 발생: %w", err)
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
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	// 이미 워크스페이스에 속해있는지 확인
	var count int64
	if err := s.db.Model(&model.UserWorkspaceRole{}).
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("user already in workspace")
	}

	// 기본 워크스페이스 역할 조회
	var defaultRole model.RoleMaster
	if err := s.db.Where("role_type = ? AND name = ?", model.RoleTypeWorkspace, "workspace_user").
		First(&defaultRole).Error; err != nil {
		return err
	}

	// 사용자를 워크스페이스에 추가
	uwr := model.UserWorkspaceRole{
		UserID:      userID,
		WorkspaceID: workspaceID,
		RoleID:      defaultRole.ID,
	}
	if err := s.db.Create(&uwr).Error; err != nil {
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
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	// 워크스페이스에서 사용자 제거
	if err := s.db.Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Delete(&model.UserWorkspaceRole{}).Error; err != nil {
		return err
	}

	return nil
}

// TODO : move to workspace service
// GetUserWorkspaceAndWorkspaceRoles 사용자의 워크스페이스와 역할 정보를 조회합니다.
func (s *WorkspaceService) GetUserWorkspaceAndWorkspaceRoles(ctx context.Context, userID uint) ([]*model.UserWorkspaceRole, error) {
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

// TODO : move to workspace service
// GetUserWorkspaceRoles 사용자의 워크스페이스 역할 목록을 조회합니다.
func (s *UserService) GetUserWorkspaceRoles(userID uint) ([]*model.UserWorkspaceRole, error) {
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

func (s *WorkspaceService) ListProjectsByWorkspaceID(workspaceID uint) ([]*model.Project, error) {
	return s.workspaceRepo.FindProjectsByWorkspaceID(workspaceID)
}
