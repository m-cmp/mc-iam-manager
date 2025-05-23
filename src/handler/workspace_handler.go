package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"log" // Added for logging

	"context" // Added for context passing

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm" // Ensure gorm is imported
)

// Helper function to get user DB ID and Platform Roles from context
// TODO: Move this to a shared location or middleware
func getUserDbIdAndPlatformRoles(ctx context.Context, c echo.Context, userService *service.UserService) (uint, []*model.PlatformRole, error) { // Added ctx parameter
	// Assume AuthMiddleware sets kcUserId in context
	kcUserIdVal := c.Get("kcUserId") // Or however the middleware provides it
	if kcUserIdVal == nil {
		return 0, nil, errors.New("kcUserId not found in context")
	}
	kcUserId, ok := kcUserIdVal.(string)
	if !ok || kcUserId == "" {
		return 0, nil, errors.New("invalid kcUserId in context")
	}

	// Fetch user details including roles
	user, err := userService.GetUserByKcID(ctx, kcUserId) // Use GetUserByKcID from UserService, pass context
	if err != nil {
		// Handle user not found or other errors
		return 0, nil, fmt.Errorf("failed to find user by kcId %s: %w", kcUserId, err)
	}
	if user == nil { // Should not happen if FindByKcID handles errors correctly
		return 0, nil, fmt.Errorf("user not found for kcId %s", kcUserId)
	}

	return user.ID, user.PlatformRoles, nil
}

// Helper function to check platform role permission
// TODO: Implement proper permission check using MciamPermissionRepository
func checkPlatformPermission(permissionRepo *repository.MciamPermissionRepository, platformRoles []*model.PlatformRole, requiredPermissionID string) (bool, error) {
	if len(platformRoles) == 0 {
		return false, nil
	}
	for _, role := range platformRoles {
		hasPerm, err := permissionRepo.CheckRoleMciamPermission("platform", role.ID, requiredPermissionID)
		if err != nil {
			log.Printf("Error checking permission %s for platform role %d: %v", requiredPermissionID, role.ID, err)
			// Continue checking other roles in case of error? Or return error immediately?
			// Let's return the error for now.
			return false, err
		}
		if hasPerm {
			return true, nil
		}
	}
	return false, nil
}

// WorkspaceHandler 워크스페이스 관리 핸들러
type WorkspaceHandler struct {
	workspaceService     *service.WorkspaceService
	userService          *service.UserService                  // For getting user info
	permissionRepo       *repository.MciamPermissionRepository // For checking permissions directly
	workspaceRoleService *service.WorkspaceRoleService         // For assigning role on create
	workspaceRoleRepo    *repository.WorkspaceRoleRepository   // For finding 'admin' role ID
	// db *gorm.DB // Not needed directly in handler
}

// NewWorkspaceHandler 새 WorkspaceHandler 인스턴스 생성
func NewWorkspaceHandler(db *gorm.DB) *WorkspaceHandler {
	// Initialize services and repositories internally
	workspaceService := service.NewWorkspaceService(db)
	userService := service.NewUserService(db)
	permissionRepo := repository.NewMciamPermissionRepository(db)
	workspaceRoleService := service.NewWorkspaceRoleService(db)    // Corrected: Initialize WorkspaceRoleService
	workspaceRoleRepo := repository.NewWorkspaceRoleRepository(db) // Initialize WorkspaceRoleRepository
	return &WorkspaceHandler{
		workspaceService:     workspaceService,
		userService:          userService,
		permissionRepo:       permissionRepo,
		workspaceRoleService: workspaceRoleService,
		workspaceRoleRepo:    workspaceRoleRepo,
	}
}

// GetAllWorkspaces godoc
// @Summary 모든 워크스페이스 조회 (관리자용)
// @Description 관리자가 모든 워크스페이스를 조회합니다.
// @Tags workspaces
// @Accept json
// @Produce json
// @Success 200 {array} model.Workspace
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Security BearerAuth
// @Router /api/v1/admin/workspaces [get]
func (h *WorkspaceHandler) GetAllWorkspaces(c echo.Context) error {
	workspaces, err := h.workspaceService.GetAllWorkspaces()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "워크스페이스 목록을 가져오는데 실패했습니다",
		})
	}
	return c.JSON(http.StatusOK, workspaces)
}

// CreateWorkspace godoc
// @Summary 새 워크스페이스 생성
// @Description 새로운 워크스페이스를 생성합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param workspace body model.Workspace true "Workspace Info"
// @Success 201 {object} model.Workspace
// @Failure 400 {object} map[string]string "error: Invalid request"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Security BearerAuth
// @Router /api/v1/workspaces [post]
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
// @Summary 워크스페이스 정보 업데이트
// @Description 워크스페이스 정보를 업데이트합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param workspace body model.Workspace true "Workspace Info"
// @Success 200 {object} model.Workspace
// @Failure 400 {object} map[string]string "error: Invalid request"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 404 {object} map[string]string "error: Workspace not found"
// @Security BearerAuth
// @Router /api/v1/workspaces/{id} [put]
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
// @Summary 워크스페이스 삭제
// @Description 워크스페이스를 삭제합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Success 204 "No Content"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 404 {object} map[string]string "error: Workspace not found"
// @Security BearerAuth
// @Router /api/v1/workspaces/{id} [delete]
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

// ListWorkspaces godoc
// @Summary 워크스페이스 목록 조회
// @Description 모든 워크스페이스 목록을 조회합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Success 200 {array} model.Workspace
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Security BearerAuth
// @Router /api/v1/workspaces [get]
func (h *WorkspaceHandler) ListWorkspaces(c echo.Context) error {
	// --- Permission Check ---
	userID, platformRoles, err := getUserDbIdAndPlatformRoles(c.Request().Context(), c, h.userService) // Pass context
	if err != nil {
		log.Printf("Error getting user info for ListWorkspaces: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to identify user"})
	}

	// Check for 'list_all' permission (Platform level)
	hasListAllPermission, err := checkPlatformPermission(h.permissionRepo, platformRoles, "mc-iam-manager:workspace:list_all") // Use helper
	if err != nil {
		log.Printf("Error checking list_all workspace permission for user %d: %v", userID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "권한 확인 중 오류 발생"})
	}

	var workspaces []model.Workspace
	if hasListAllPermission {
		// User has permission to list all workspaces
		workspaces, err = h.workspaceService.List()
	} else {
		// User can only list assigned workspaces
		// TODO: Check for 'list_assigned' permission if needed for more granular control
		workspaces, err = h.userService.FindWorkspacesByUserID(userID) // Use the new repo method via userService
	}

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 목록 조회 실패: %v", err)})
	}
	// --- End Permission Check ---

	// Return empty list if no workspaces found or accessible
	if workspaces == nil {
		workspaces = []model.Workspace{}
	}
	return c.JSON(http.StatusOK, workspaces)
}

// GetWorkspaceByID godoc
// @Summary 워크스페이스 ID로 조회
// @Description 특정 워크스페이스를 ID로 조회합니다
// @Tags workspaces
// @Accept json
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {object} model.Workspace
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 404 {object} map[string]string "error: Workspace not found"
// @Security BearerAuth
// @Router /api/v1/workspaces/{workspaceId} [get]
func (h *WorkspaceHandler) GetWorkspaceByID(c echo.Context) error {
	idStr := c.Param("workspaceId")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID입니다"})
	}

	workspace, err := h.workspaceService.GetByID(uint(id))
	if err != nil {
		if err.Error() == "workspace not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 조회 실패: %v", err)})
	}
	return c.JSON(http.StatusOK, workspace)
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
// @Router /api/v1/workspaces/name/{workspaceName} [get]
func (h *WorkspaceHandler) GetWorkspaceByName(c echo.Context) error {
	name := c.Param("workspaceName")

	workspace, err := h.workspaceService.GetByName(name)
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
// @Router /api/v1/workspaces/{workspaceId}/users [get]
func (h *WorkspaceHandler) ListUsersAndRolesByWorkspace(c echo.Context) error {
	idStr := c.Param("workspaceId")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID입니다"})
	}

	usersWithRoles, err := h.workspaceService.GetUsersAndRolesByWorkspaceID(uint(id))
	if err != nil {
		if err.Error() == "workspace not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("사용자 및 역할 목록 조회 실패: %v", err)})
	}

	return c.JSON(http.StatusOK, usersWithRoles)
}

// ListProjectsByWorkspace godoc
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
// @Router /api/v1/workspaces/{workspaceId}/projects [get]
func (h *WorkspaceHandler) ListProjectsByWorkspace(c echo.Context) error {
	workspaceID, err := strconv.ParseUint(c.Param("workspaceId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID입니다"})
	}

	projects, err := h.workspaceService.GetProjectsByWorkspaceID(uint(workspaceID))
	if err != nil {
		// Handle not found error from service (which checks workspace existence)
		if err.Error() == "workspace not found" { // Assuming service returns this specific error
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("프로젝트 목록 조회 실패: %v", err)})
	}

	// Return empty list if no projects are associated, instead of 404
	if projects == nil {
		projects = []model.Project{} // Ensure we return [] instead of null
	}

	return c.JSON(http.StatusOK, projects)
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
// @Router /api/v1/workspaces/{id}/projects/{projectId} [post]
func (h *WorkspaceHandler) AddProjectToWorkspace(c echo.Context) error {
	workspaceID, err := strconv.ParseUint(c.Param("workspaceId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID입니다"})
	}
	projectID, err := strconv.ParseUint(c.Param("projectId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 프로젝트 ID입니다"})
	}

	if err := h.workspaceService.AddProjectToWorkspace(uint(workspaceID), uint(projectID)); err != nil {
		// Handle not found errors from service
		if err.Error() == "workspace not found" || err.Error() == "project not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		// Handle potential duplicate errors if repo doesn't ignore them
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("프로젝트 연결 실패: %v", err)})
	}
	return c.NoContent(http.StatusNoContent)
}

// RemoveProjectFromWorkspace 워크스페이스에서 프로젝트 제거
func (h *WorkspaceHandler) RemoveProjectFromWorkspace(c echo.Context) error {
	workspaceID, err := strconv.ParseUint(c.Param("workspaceId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid workspace ID"})
	}

	projectID, err := strconv.ParseUint(c.Param("projectId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid project ID"})
	}

	if err := h.workspaceService.RemoveProjectFromWorkspace(uint(workspaceID), uint(projectID)); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

// ListAllWorkspaces godoc
// @Summary 워크스페이스와 연관된 프로젝트 목록 조회
// @Description 모든 워크스페이스와 각 워크스페이스에 연관된 프로젝트 목록을 조회합니다.
// @Tags workspaces
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} model.WorkspaceWithProjects
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /api/workspaces/all [get]
func (h *WorkspaceHandler) ListAllWorkspaces(c echo.Context) error {
	workspaces, err := h.workspaceService.ListAllWorkspaces()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "워크스페이스 목록 조회에 실패했습니다",
		})
	}

	return c.JSON(http.StatusOK, workspaces)
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
// @Router /api/workspaces/all/users [get]
func (h *WorkspaceHandler) ListAllWorkspaceUsersAndRoles(c echo.Context) error {
	// 모든 워크스페이스 조회
	workspaces, err := h.workspaceService.GetAllWorkspaces()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "워크스페이스 조회 실패"})
	}

	var result []model.WorkspaceWithUsersAndRoles
	for _, workspace := range workspaces {
		// 각 워크스페이스의 사용자와 역할 조회
		usersWithRoles, err := h.workspaceService.GetUsersAndRolesByWorkspaceID(workspace.ID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "사용자 및 역할 조회 실패"})
		}

		result = append(result, model.WorkspaceWithUsersAndRoles{
			ID:          workspace.ID,
			Name:        workspace.Name,
			Description: workspace.Description,
			CreatedAt:   workspace.CreatedAt,
			UpdatedAt:   workspace.UpdatedAt,
			Users:       usersWithRoles,
		})
	}

	return c.JSON(http.StatusOK, result)
}
