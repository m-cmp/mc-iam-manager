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

// WorkspaceHandler workspace management handler
type WorkspaceHandler struct {
	workspaceService  *service.WorkspaceService
	userService       *service.UserService
	permissionRepo    *repository.MciamPermissionRepository
	roleService       *service.RoleService
	workspaceRoleRepo *repository.WorkspaceRoleRepository
	roleRepo          *repository.RoleRepository
}

// NewWorkspaceHandler create new WorkspaceHandler instance
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

// Define workspace management functions.

// ListWorkspaces godoc
// @Summary List all workspaces
// @Description Retrieve a list of all workspaces.
// @Tags workspaces
// @Accept json
// @Produce json
// @Success 200 {array} model.Workspace
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/workspaces/list [post]
// @Id listWorkspaces
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
	// 	return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error occurred while checking permissions"})
	// }

	var req model.WorkspaceFilterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	var workspaces []*model.Workspace
	// if hasListAllPermission {
	// User has permission to list all workspaces
	workspaces, err := h.workspaceService.ListWorkspaces(&req)
	// } else {
	// 	// User can only list assigned workspaces
	// 	if req.UserID == "" {
	// 		return c.JSON(http.StatusBadRequest, map[string]string{"error": "User ID is required"})
	// 	}
	// 	workspaces, err = h.workspaceService.ListWorkspaces(&req) // Use the new repo method via userService
	// }

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to retrieve workspace list: %v", err)})
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
	// 		"error": "Failed to retrieve workspace list",
	// 	})
	// }
	// return c.JSON(http.StatusOK, workspaces)
}

// GetWorkspaceByID godoc
// @Summary Get workspace by ID
// @Description Retrieve workspace details by workspace ID.
// @Tags workspaces
// @Accept json
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {object} model.Workspace
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/workspaces/id/{workspaceId} [get]
// @Id getWorkspaceByID
func (h *WorkspaceHandler) GetWorkspaceByID(c echo.Context) error {

	workspaceIDInt, err := util.StringToUint(c.Param("workspaceId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid workspace ID"})
	}

	workspace, err := h.workspaceService.GetWorkspaceByID(workspaceIDInt)
	if err != nil {
		if err.Error() == "workspace not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to retrieve workspace: %v", err)})
	}
	return c.JSON(http.StatusOK, workspace)
}

// CreateWorkspace godoc
// @Summary Create new workspace
// @Description Create a new workspace with the specified information.
// @Tags workspaces
// @Accept json
// @Produce json
// @Param workspace body model.Workspace true "Workspace Info"
// @Success 201 {object} model.Workspace
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/workspaces [post]
// @Id createWorkspace
func (h *WorkspaceHandler) CreateWorkspace(c echo.Context) error {
	var workspace model.Workspace
	if err := c.Bind(&workspace); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	if err := h.workspaceService.CreateWorkspace(&workspace); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create workspace",
		})
	}

	return c.JSON(http.StatusCreated, workspace)
}

// UpdateWorkspace godoc
// @Summary Update workspace
// @Description Update the details of an existing workspace.
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
// @Id updateWorkspace
func (h *WorkspaceHandler) UpdateWorkspace(c echo.Context) error {
	idStr := c.Param("workspaceId")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid workspace ID"})
	}

	var workspace model.Workspace
	if err := c.Bind(&workspace); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	workspace.ID = uint(id)
	if err := h.workspaceService.UpdateWorkspace(&workspace); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update workspace",
		})
	}

	return c.JSON(http.StatusOK, workspace)
}

// DeleteWorkspace godoc
// @Summary Delete workspace
// @Description Delete a workspace by its ID.
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Success 204 "No Content"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/workspaces/id/{workspaceId} [delete]
// @Id deleteWorkspace
func (h *WorkspaceHandler) DeleteWorkspace(c echo.Context) error {
	idStr := c.Param("workspaceId")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid workspace ID"})
	}

	if err := h.workspaceService.DeleteWorkspace(uint(id)); err != nil {
		if err.Error() == "workspace not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to delete workspace: %v", err)})
	}

	return c.JSON(http.StatusNoContent, map[string]string{"message": "Workspace deleted successfully"})
}

// ListWorkspaceUsers godoc
// @Summary List workspace users
// @Description List users by workspace criteria
// @Tags workspaces
// @Accept json
// @Produce json
// @Success 200 {array} model.WorkspaceWithUsersAndRoles
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Security BearerAuth
// @Router /api/workspaces/users/list [post]
// @Id listWorkspaceUsers
func (h *WorkspaceHandler) ListWorkspaceUsers(c echo.Context) error {
	var req model.WorkspaceFilterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
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

	//var workspaces []*model.WorkspaceWithUsersAndRoles
	workspacesUserRoles, err := h.workspaceService.ListWorkspaceUsers(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to retrieve workspace user list: %v", err)})
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

	//return c.JSON(http.StatusOK, workspaces)
}

// GetWorkspaceByName godoc
// @Summary Get workspace by name
// @Description Retrieve specific workspace by name
// @Tags workspaces
// @Accept json
// @Produce json
// @Param workspaceName path string true "Workspace Name"
// @Success 200 {object} model.Workspace
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 404 {object} map[string]string "error: Workspace not found"
// @Security BearerAuth
// @Router /api/workspaces/name/{workspaceName} [get]
// @Id getWorkspaceByName
func (h *WorkspaceHandler) GetWorkspaceByName(c echo.Context) error {
	name := c.Param("workspaceName")

	workspace, err := h.workspaceService.GetWorkspaceByName(name)
	if err != nil {
		if err.Error() == "workspace not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to retrieve workspace: %v", err)})
	}
	return c.JSON(http.StatusOK, workspace)
}

// ListUsersAndRolesByWorkspace godoc
// @Summary List users and roles by workspace
// @Description Retrieve users and roles list belonging to workspace
// @Tags workspaces
// @Accept json
// @Produce json
// @Param workspaceId path int true "Workspace ID"
// @Success 200 {array} model.UserWorkspaceRole
// @Failure 400 {object} map[string]string "error: Invalid workspace ID"
// @Failure 404 {object} map[string]string "error: Workspace not found"
// @Failure 500 {object} map[string]string "error: Internal server error"
// @Router /api/workspaces/id/{workspaceId}/users/list [post]
// @Id listUsersAndRolesByWorkspace
func (h *WorkspaceHandler) ListUsersAndRolesByWorkspaces(c echo.Context) error {
	var req model.WorkspaceFilterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	usersWithRoles, err := h.roleService.ListUsersAndRolesWithWorkspaces(req)
	if err != nil {
		if err.Error() == "workspace not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to retrieve user and role list: %v", err)})
	}

	return c.JSON(http.StatusOK, usersWithRoles)
}

// ListWorkspaceProjects godoc
// @Summary List workspace projects
// @Description Retrieve project list belonging to specific workspace
// @Tags workspaces
// @Accept json
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {array} model.Project
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 404 {object} map[string]string "error: Workspace not found"
// @Security BearerAuth
// @Router /api/workspaces/projects/list [post]
// @Id listWorkspaceProjects
func (h *WorkspaceHandler) ListWorkspaceProjects(c echo.Context) error {
	var req model.WorkspaceFilterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	workspaceProjects, err := h.workspaceService.ListWorkspacesProjects(&req)
	if err != nil {
		// Handle not found error from service (which checks workspace existence)
		if err.Error() == "workspace not found" { // Assuming service returns this specific error
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to retrieve project list: %v", err)})
	}

	// Return empty list if no projects are associated, instead of 404

	return c.JSON(http.StatusOK, workspaceProjects)
}

// GetWorkspaceProjectsByWorkspaceId godoc
// @Summary List workspace projects
// @Description Retrieve project list belonging to specific workspace
// @Tags workspaces
// @Accept json
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {array} model.Project
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 404 {object} map[string]string "error: Workspace not found"
// @Security BearerAuth
// @Router /api/workspaces/id/{workspaceId}/projects/list [get]
// @Id getWorkspaceProjectsByWorkspaceId
func (h *WorkspaceHandler) GetWorkspaceProjectsByWorkspaceId(c echo.Context) error {
	workspaceId := c.Param("workspaceId")
	workspaceIdInt, err := util.StringToUint(workspaceId)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid workspace ID format"})
	}
	workspaceProjects, err := h.workspaceService.GetWorkspaceProjectsByWorkspaceId(workspaceIdInt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to retrieve project list: %v", err)})
	}
	return c.JSON(http.StatusOK, workspaceProjects)
}

// AddProjectToWorkspace godoc
// @Summary Add project to workspace
// @Description Add a project to a workspace
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
// @Router /api/workspaces/assign/projects [post]
// @Id addProjectToWorkspace
func (h *WorkspaceHandler) AddProjectToWorkspace(c echo.Context) error {
	var req model.WorkspaceProjectMappingRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	if req.WorkspaceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Workspace ID is required"})
	}

	workspaceIDInt, err := util.StringToUint(req.WorkspaceID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid workspace ID format"})
	}

	for _, projectID := range req.ProjectIDs {
		projectIDInt, err := util.StringToUint(projectID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid project ID format"})
		}
		// Check existence of workspace, project in AddProjectToWorkspace function
		if err := h.workspaceService.AddProjectToWorkspace(workspaceIDInt, projectIDInt); err != nil {
			if err.Error() == "workspace not found" || err.Error() == "project not found" {
				return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to assign project: %v", err)})
		}
	}
	return c.JSON(http.StatusOK, model.Response{
		Message: "Assigned project to workspace successfully",
	})
	// return c.NoContent(http.StatusNoContent)
}

// RemoveProjectFromWorkspace godoc
// @Summary Remove project from workspace
// @Description Remove a project from a workspace
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Router /api/workspaces/unassign/projects [delete]
// @Id removeProjectFromWorkspace
func (h *WorkspaceHandler) RemoveProjectFromWorkspace(c echo.Context) error {
	var req model.WorkspaceProjectMappingRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	if req.WorkspaceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Workspace ID is required"})
	}

	workspaceIDInt, err := util.StringToUint(req.WorkspaceID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid workspace ID format"})
	}

	// Cannot remove project from default workspace
	if req.WorkspaceID == "1" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Cannot remove project from default workspace"})
	}

	for _, projectID := range req.ProjectIDs {
		projectIDInt, err := util.StringToUint(projectID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid project ID format"})
		}
		if err := h.workspaceService.RemoveProjectFromWorkspace(workspaceIDInt, projectIDInt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return c.JSON(http.StatusOK, model.Response{
		Message: "Released project from workspace successfully",
	})
	// return c.NoContent(http.StatusNoContent)
}

// ListAllWorkspaceUsersAndRoles godoc
// @Summary List users and roles in workspace
// @Description Retrieve the list of users and roles assigned to the workspace.
// @Tags workspaces
// @Accept json
// @Produce json
// @Success 200 {array} model.WorkspaceWithUsersAndRoles
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 500 {object} map[string]string "error: Internal server error"
// @Security BearerAuth
// @Router /api/workspaces/users-roles/list [post]
// @Id listAllWorkspaceUsersAndRoles
func (h *WorkspaceHandler) ListWorkspaceUsersAndRoles(c echo.Context) error {

	var req model.WorkspaceFilterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	workspaceUsersRoles, err := h.roleService.ListUsersAndRolesWithWorkspaces(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve users and roles"})
	}

	return c.JSON(http.StatusOK, workspaceUsersRoles)
}

// ListWorkspaceRoles godoc
// @Summary List workspace roles
// @Description Retrieve all workspace-level roles with optional filtering
// @Tags workspaces
// @Accept json
// @Produce json
// @Param request body model.RoleFilterRequest true "Role filter parameters"
// @Success 200 {array} model.RoleMaster "Successfully retrieved workspace roles"
// @Failure 400 {object} map[string]string "error: Invalid request format"
// @Failure 500 {object} map[string]string "error: Failed to retrieve workspace roles"
// @Security BearerAuth
// @Router /api/workspaces/roles/list [post]
// @Id listWorkspaceRoles
func (h *WorkspaceHandler) ListWorkspaceRoles(c echo.Context) error {
	var req model.RoleFilterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
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
// @Router /api/workspaces/{id}/users [post]
// @Id addUserToWorkspace
func (h *WorkspaceHandler) AddUserToWorkspace(c echo.Context) error {
	var req model.AssignRoleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	if req.WorkspaceID == "" || req.UserID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Workspace ID and User ID are required"})
	}

	// 워크스페이스 역할 할당
	var workspaceID uint
	var userID uint
	var err error
	workspaceID, err = util.StringToUint(req.WorkspaceID)
	if err != nil {
		log.Printf("Workspace ID conversion error: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid workspace ID format"})
	}
	if req.UserID != "" {
		userID, err = util.StringToUint(req.UserID)
		if err != nil {
			log.Printf("User ID conversion error: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID format"})
		}
	}

	if err := h.workspaceService.AddUserToWorkspace(workspaceID, userID); err != nil {
		if err.Error() == "workspace not found" || err.Error() == "user not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to add user: %v", err)})
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
// @Router /api/workspaces/{id}/users/{userId} [delete]
// @Id removeUserFromWorkspace
func (h *WorkspaceHandler) RemoveUserFromWorkspace(c echo.Context) error {
	workspaceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid workspace ID"})
	}

	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
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
