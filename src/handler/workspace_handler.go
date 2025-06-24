package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"log" // Added for logging

	"context" // Added for context passing

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/m-cmp/mc-iam-manager/service"
	"github.com/m-cmp/mc-iam-manager/util"
	"gorm.io/gorm" // Ensure gorm is imported
)

// WorkspaceHandler 워크스페이스 관리 핸들러
type WorkspaceHandler struct {
	workspaceService  *service.WorkspaceService
	userService       *service.UserService
	permissionRepo    *repository.MciamPermissionRepository
	roleService       *service.RoleService
	workspaceRoleRepo *repository.WorkspaceRoleRepository
	roleRepo          *repository.RoleRepository
}

// NewWorkspaceHandler 새 WorkspaceHandler 인스턴스 생성
func NewWorkspaceHandler(db *gorm.DB) *WorkspaceHandler {
	workspaceService := service.NewWorkspaceService(db)
	userService := service.NewUserService(db)
	permissionRepo := repository.NewMciamPermissionRepository(db)
	roleService := service.NewRoleService(db)
	workspaceRoleRepo := repository.NewWorkspaceRoleRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	return &WorkspaceHandler{
		workspaceService:  workspaceService,
		userService:       userService,
		permissionRepo:    permissionRepo,
		roleService:       roleService,
		workspaceRoleRepo: workspaceRoleRepo,
		roleRepo:          roleRepo,
	}
}

// 워크스페이스 관리 기능들을 정의함.

// ListWorkspaces godoc
// @Summary List workspaces
// @Description Get a list of all workspaces
// @Tags workspaces
// @Accept json
// @Produce json
// @Success 200 {array} model.Workspace
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/workspaces/list [post]
// @OperationId listWorkspaces
func (h *WorkspaceHandler) ListWorkspaces(c echo.Context) error {
	// // --- Permission Check ---
	// userID, platformRoles, err := getUserDbIdAndPlatformRoles(c.Request().Context(), c, h.userService) // Pass context
	// if err != nil {
	// 	log.Printf("Error getting user info for ListWorkspaces: %v", err)
	// 	return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to identify user"})
	// }

	// // Check for 'list_all' permission (Platform level)
	// hasListAllPermission, err := checkPlatformPermission(h.permissionRepo, platformRoles, "mc-iam-manager:workspace:list_all") // Use helper
	// if err != nil {
	// 	log.Printf("Error checking list_all workspace permission for user %d: %v", userID, err)
	// 	return c.JSON(http.StatusInternalServerError, map[string]string{"error": "권한 확인 중 오류 발생"})
	// }

	var req model.WorkspaceFilterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	var workspaces []*model.Workspace
	// if hasListAllPermission {
	// User has permission to list all workspaces
	workspaces, err := h.workspaceService.ListWorkspaces(&req)
	// } else {
	// 	// User can only list assigned workspaces
	// 	if req.UserID == "" {
	// 		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID가 필요합니다"})
	// 	}
	// 	workspaces, err = h.workspaceService.ListWorkspaces(&req) // Use the new repo method via userService
	// }

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 목록 조회 실패: %v", err)})
	}
	// --- End Permission Check ---

	// Return empty list if no workspaces found or accessible
	if workspaces == nil {
		workspaces = []*model.Workspace{}
	}
	return c.JSON(http.StatusOK, workspaces)

	// workspaces, err := h.workspaceService.GetAllWorkspaces()
	// if err != nil {
	// 	return c.JSON(http.StatusInternalServerError, map[string]string{
	// 		"error": "워크스페이스 목록을 가져오는데 실패했습니다",
	// 	})
	// }
	// return c.JSON(http.StatusOK, workspaces)
}

// GetWorkspaceByID godoc
// @Summary Get workspace by ID
// @Description Get workspace details by ID
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Success 200 {object} model.Workspace
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/workspaces/id/{workspaceId} [get]
// @OperationId getWorkspaceByID
func (h *WorkspaceHandler) GetWorkspaceByID(c echo.Context) error {

	workspaceIDInt, err := util.StringToUint(c.Param("workspaceId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID입니다"})
	}

	workspace, err := h.workspaceService.GetWorkspaceByID(workspaceIDInt)
	if err != nil {
		if err.Error() == "workspace not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 조회 실패: %v", err)})
	}
	return c.JSON(http.StatusOK, workspace)
}

// CreateWorkspace godoc
// @Summary Create workspace
// @Description Create a new workspace
// @Tags workspaces
// @Accept json
// @Produce json
// @Param workspace body model.Workspace true "Workspace Info"
// @Success 201 {object} model.Workspace
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/workspaces [post]
// @OperationId createWorkspace
func (h *WorkspaceHandler) CreateWorkspace(c echo.Context) error {
	var workspace model.Workspace
	if err := c.Bind(&workspace); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "잘못된 요청 형식입니다",
		})
	}

	if err := h.workspaceService.CreateWorkspace(&workspace); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "워크스페이스 생성에 실패했습니다",
		})
	}

	return c.JSON(http.StatusCreated, workspace)
}

// UpdateWorkspace godoc
// @Summary Update workspace
// @Description Update workspace information
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param workspace body model.Workspace true "Workspace Info"
// @Success 200 {object} model.Workspace
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/workspaces/id/{workspaceId} [put]
// @OperationId updateWorkspace
func (h *WorkspaceHandler) UpdateWorkspace(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID입니다"})
	}

	var workspace model.Workspace
	if err := c.Bind(&workspace); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "잘못된 요청 형식입니다",
		})
	}

	workspace.ID = uint(id)
	if err := h.workspaceService.UpdateWorkspace(&workspace); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "워크스페이스 수정에 실패했습니다",
		})
	}

	return c.JSON(http.StatusOK, workspace)
}

// DeleteWorkspace godoc
// @Summary Delete workspace
// @Description Delete a workspace
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Success 204 "No Content"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/workspaces/id/{workspaceId} [delete]
// @OperationId deleteWorkspace
func (h *WorkspaceHandler) DeleteWorkspace(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID입니다"})
	}

	if err := h.workspaceService.DeleteWorkspace(uint(id)); err != nil {
		if err.Error() == "workspace not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 삭제 실패: %v", err)})
	}

	return c.NoContent(http.StatusNoContent)
}

// ListWorkspaceUsers godoc
// @Summary 워크스페이스 목록 조회
// @Description 워크스페이스 기준으로 사용자 목록만 조회합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Success 200 {array} model.WorkspaceWithUsersAndRoles
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Security BearerAuth
// @Router /workspaces/users/list [post]
// @OperationId listWorkspaceUsers
func (h *WorkspaceHandler) ListWorkspaceUsers(c echo.Context) error {
	var req model.WorkspaceFilterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	// if req.UserID == "" {
	// 	return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID가 필요합니다"})
	// }

	// // --- Permission Check ---
	// userID, platformRoles, err := getUserDbIdAndPlatformRoles(c.Request().Context(), c, h.userService) // Pass context
	// if err != nil {
	// 	log.Printf("Error getting user info for ListWorkspaces: %v", err)
	// 	return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to identify user"})
	// }

	// // Check for 'list_all' permission (Platform level)
	// hasListAllPermission, err := checkPlatformPermission(h.permissionRepo, platformRoles, "mc-iam-manager:workspace:list_all") // Use helper
	// if err != nil {
	// 	log.Printf("Error checking list_all workspace permission for user %d: %v", userID, err)
	// 	return c.JSON(http.StatusInternalServerError, map[string]string{"error": "권한 확인 중 오류 발생"})
	// }

	// userIDInt, err := util.StringToUint(req.UserID)
	// if err != nil {
	// 	return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 사용자 ID 형식입니다"})
	// }

	var workspaces []*model.WorkspaceWithUsersAndRoles
	workspacesUserRoles, err := h.workspaceService.ListWorkspaceUsers(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 사용자 목록 조회 실패: %v", err)})
	}

	return c.JSON(http.StatusOK, workspacesUserRoles)

	// if hasListAllPermission {
	// 	// User has permission to list all workspaces
	// 	workspaces, err = h.workspaceService.ListWorkspacesByUserID(userIDInt)
	// } else {
	// 	// User can only list assigned workspaces
	// 	// TODO: Check for 'list_assigned' permission if needed for more granular control
	// 	if req.UserID == "" {
	// 		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID가 필요합니다"})
	// 	}

	// 	workspaces, err = h.workspaceService.ListWorkspacesByUserID(userIDInt) // Use the new repo method via userService
	// }

	// if err != nil {
	// 	return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 목록 조회 실패: %v", err)})
	// }
	// // --- End Permission Check ---

	return c.JSON(http.StatusOK, workspaces)
}

// GetWorkspaceByName godoc
// @Summary 워크스페이스 이름으로 조회
// @Description 특정 워크스페이스를 이름으로 조회합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param workspaceName path string true "Workspace Name"
// @Success 200 {object} model.Workspace
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 404 {object} map[string]string "error: Workspace not found"
// @Security BearerAuth
// @Router /workspaces/name/{workspaceName} [get]
// @OperationId getWorkspaceByName
func (h *WorkspaceHandler) GetWorkspaceByName(c echo.Context) error {
	name := c.Param("workspaceName")

	workspace, err := h.workspaceService.GetWorkspaceByName(name)
	if err != nil {
		if err.Error() == "workspace not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 조회 실패: %v", err)})
	}
	return c.JSON(http.StatusOK, workspace)
}

// ListUsersAndRolesByWorkspace godoc
// @Summary 워크스페이스에 속한 사용자와 역할 목록 조회
// @Description 워크스페이스에 속한 사용자와 역할 목록을 조회합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param workspaceId path int true "Workspace ID"
// @Success 200 {array} model.UserWorkspaceRole
// @Failure 400 {object} map[string]string "error: Invalid workspace ID"
// @Failure 404 {object} map[string]string "error: Workspace not found"
// @Failure 500 {object} map[string]string "error: Internal server error"
// @Router /workspaces/{workspaceId}/users/list [post]
// @OperationId listUsersAndRolesByWorkspace
func (h *WorkspaceHandler) ListUsersAndRolesByWorkspaces(c echo.Context) error {
	var req model.WorkspaceFilterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	usersWithRoles, err := h.roleService.ListUsersAndRolesWithWorkspaces(req)
	if err != nil {
		if err.Error() == "workspace not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("사용자 및 역할 목록 조회 실패: %v", err)})
	}

	return c.JSON(http.StatusOK, usersWithRoles)
}

// ListWorkspaceProjects godoc
// @Summary 워크스페이스의 프로젝트 목록 조회
// @Description 특정 워크스페이스에 속한 프로젝트 목록을 조회합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {array} model.Project
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 404 {object} map[string]string "error: Workspace not found"
// @Security BearerAuth
// @Router /workspaces/projects/list [post]
// @OperationId listWorkspaceProjects
func (h *WorkspaceHandler) ListWorkspaceProjects(c echo.Context) error {
	var req model.WorkspaceFilterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	workspaceProjects, err := h.workspaceService.ListWorkspacesProjects(&req)
	if err != nil {
		// Handle not found error from service (which checks workspace existence)
		if err.Error() == "workspace not found" { // Assuming service returns this specific error
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("프로젝트 목록 조회 실패: %v", err)})
	}

	// Return empty list if no projects are associated, instead of 404

	return c.JSON(http.StatusOK, workspaceProjects)
}

// AddProjectToWorkspace godoc
// @Summary 워크스페이스에 프로젝트 추가
// @Description 워크스페이스에 프로젝트를 추가합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Success 200 {object} model.Workspace
// @Failure 400 {object} map[string]string "error: Invalid request"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 404 {object} map[string]string "error: Workspace or Project not found"
// @Security BearerAuth
// @Router /workspaces/assign/projects [post]
// @OperationId addProjectToWorkspace
func (h *WorkspaceHandler) AddProjectToWorkspace(c echo.Context) error {
	var req model.WorkspaceProjectMappingRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}
	if req.WorkspaceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "워크스페이스 ID가 필요합니다"})
	}

	workspaceIDInt, err := util.StringToUint(req.WorkspaceID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식입니다"})
	}
	for _, projectID := range req.ProjectID {
		projectIDInt, err := util.StringToUint(projectID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 프로젝트 ID 형식입니다"})
		}
		if err := h.workspaceService.AddProjectToWorkspace(workspaceIDInt, projectIDInt); err != nil {
			if err.Error() == "workspace not found" || err.Error() == "project not found" {
				return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("프로젝트 연결 실패: %v", err)})
		}
	}
	return c.NoContent(http.StatusNoContent)
}

// RemoveProjectFromWorkspace 워크스페이스에서 프로젝트 제거
// @Summary Remove project from workspace
// @Description Remove a project from a workspace
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Router /workspaces/unassign/projects [delete]
// @OperationId removeProjectFromWorkspace
func (h *WorkspaceHandler) RemoveProjectFromWorkspace(c echo.Context) error {
	var req model.WorkspaceProjectMappingRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}
	if req.WorkspaceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "워크스페이스 ID가 필요합니다"})
	}

	workspaceIDInt, err := util.StringToUint(req.WorkspaceID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식입니다"})
	}

	for _, projectID := range req.ProjectID {
		projectIDInt, err := util.StringToUint(projectID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 프로젝트 ID 형식입니다"})
		}
		if err := h.workspaceService.RemoveProjectFromWorkspace(workspaceIDInt, projectIDInt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// ListAllWorkspaceUsersAndRoles godoc
// @Summary 모든 워크스페이스의 사용자와 역할 목록 조회
// @Description 모든 워크스페이스에 할당된 사용자와 역할 목록을 조회합니다.
// @Tags workspaces
// @Accept json
// @Produce json
// @Success 200 {array} model.WorkspaceWithUsersAndRoles
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 500 {object} map[string]string "error: Internal server error"
// @Security BearerAuth
// @Router /workspaces/all/users/list [post]
// @OperationId listAllWorkspaceUsersAndRoles
func (h *WorkspaceHandler) ListWorkspaceUsersAndRoles(c echo.Context) error {

	var req model.WorkspaceFilterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	workspaceUsersRoles, err := h.roleService.ListUsersAndRolesWithWorkspaces(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "사용자 및 역할 조회 실패"})
	}

	return c.JSON(http.StatusOK, workspaceUsersRoles)
}

// GetWorkspaceRoles 워크스페이스의 역할 목록 조회
func (h *WorkspaceHandler) ListWorkspaceRoles(c echo.Context) error {
	var req model.RoleFilterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	// workspace 역할만 조회
	if req.RoleTypes == nil {
		req.RoleTypes = []constants.IAMRoleType{constants.RoleTypeWorkspace}
	}

	roles, err := h.roleService.ListWorkspaceRoles(&req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, roles)
}

// ListWorkspaceUsers godoc
// @Summary Get workspace users
// @Description Get users in a workspace
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Success 200 {array} model.User
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /workspaces/{id}/users [get]
// @OperationId ListWorkspaceUsers
// func (h *WorkspaceHandler) ListWorkspaceUsers(c echo.Context) error {
// 	var req model.WorkspaceFilterRequest
// 	if err := c.Bind(&req); err != nil {
// 		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
// 	}

// 	workspaceUsers, err := h.workspaceService.ListWorkspaces(&req)
// 	if err != nil {
// 		if err.Error() == "workspace not found" {
// 			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
// 		}
// 		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("사용자 목록 조회 실패: %v", err)})
// 	}

// 	return c.JSON(http.StatusOK, workspaceUsers)
// }

// AddUserToWorkspace godoc
// @Summary Add user to workspace
// @Description Add a user to a workspace
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param request body model.AssignRoleRequest true "User Info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /workspaces/{id}/users [post]
// @OperationId addUserToWorkspace
func (h *WorkspaceHandler) AddUserToWorkspace(c echo.Context) error {
	var req model.AssignRoleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}
	if req.WorkspaceID == "" || req.UserID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "워크스페이스 ID와 사용자 ID가 필요합니다"})
	}

	// 워크스페이스 역할 할당
	var workspaceID uint
	var userID uint
	var err error
	workspaceID, err = util.StringToUint(req.WorkspaceID)
	if err != nil {
		log.Printf("Workspace ID 변환 오류: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 workspace ID 형식입니다"})
	}
	if req.UserID != "" {
		userID, err = util.StringToUint(req.UserID)
		if err != nil {
			log.Printf("User ID 변환 오류: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 user ID 형식입니다"})
		}
	}

	if err := h.workspaceService.AddUserToWorkspace(workspaceID, userID); err != nil {
		if err.Error() == "workspace not found" || err.Error() == "user not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("사용자 추가 실패: %v", err)})
	}
	return c.NoContent(http.StatusNoContent)
}

// RemoveUserFromWorkspace godoc
// @Summary Remove user from workspace
// @Description Remove a user from a workspace
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param userId path string true "User ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /workspaces/{id}/users/{userId} [delete]
// @OperationId removeUserFromWorkspace
func (h *WorkspaceHandler) RemoveUserFromWorkspace(c echo.Context) error {
	workspaceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID입니다"})
	}

	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 사용자 ID입니다"})
	}

	if err := h.workspaceService.RemoveUserFromWorkspace(uint(workspaceID), uint(userID)); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

// Helper function to get user DB ID and Platform Roles from context
// TODO: Move this to a shared location or middleware
func getUserDbIdAndPlatformRoles(ctx context.Context, c echo.Context, userService *service.UserService) (uint, []*model.RoleMaster, error) {
	// Assume AuthMiddleware sets kcUserId in context
	kcUserIdVal := c.Get("kcUserId")
	if kcUserIdVal == nil {
		return 0, nil, errors.New("kcUserId not found in context")
	}
	kcUserId, ok := kcUserIdVal.(string)
	if !ok || kcUserId == "" {
		return 0, nil, errors.New("invalid kcUserId in context")
	}

	// Fetch user details including roles
	user, err := userService.GetUserByKcID(ctx, kcUserId)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to find user by kcId %s: %w", kcUserId, err)
	}
	if user == nil {
		return 0, nil, fmt.Errorf("user not found for kcId %s", kcUserId)
	}

	return user.ID, user.PlatformRoles, nil
}

// Helper function to check platform role permission
// TODO: Implement proper permission check using MciamPermissionRepository
func checkPlatformPermission(permissionRepo *repository.MciamPermissionRepository, platformRoles []*model.RoleMaster, requiredPermissionID string) (bool, error) {
	if len(platformRoles) == 0 {
		return false, nil
	}
	for _, role := range platformRoles {
		hasPerm, err := permissionRepo.CheckRoleMciamPermission(constants.RoleTypePlatform, role.ID, requiredPermissionID)
		if err != nil {
			log.Printf("Error checking permission %s for platform role %d: %v", requiredPermissionID, role.ID, err)
			return false, err
		}
		if hasPerm {
			return true, nil
		}
	}
	return false, nil
}
