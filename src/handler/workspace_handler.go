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

// CreateWorkspace godoc
// @Summary 워크스페이스 생성
// @Description 새로운 워크스페이스를 생성합니다.
// @Tags workspaces
// @Accept json
// @Produce json
// @Param workspace body model.Workspace true "워크스페이스 정보 (ID, CreatedAt, UpdatedAt, Projects 제외)"
// @Success 201 {object} model.Workspace
// @Failure 400 {object} map[string]string "error: 잘못된 요청 형식"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Failure 403 {object} map[string]string "error: 권한 부족"
// @Security BearerAuth
// @Router /workspaces [post]
func (h *WorkspaceHandler) CreateWorkspace(c echo.Context) error {
	// --- Permission Check ---
	userID, platformRoles, err := getUserDbIdAndPlatformRoles(c.Request().Context(), c, h.userService) // Pass context
	if err != nil {
		log.Printf("Error getting user info for CreateWorkspace: %v", err)
		// Fallback or default behavior if user info is missing? For now, deny.
		// This indicates an issue with the AuthMiddleware setup.
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to identify user"})
	}

	// Check for 'create' permission (Platform level)
	hasPermission, err := checkPlatformPermission(h.permissionRepo, platformRoles, "mc-iam-manager:workspace:create") // Use helper
	if err != nil {
		log.Printf("Error checking create workspace permission for user %d: %v", userID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "권한 확인 중 오류 발생"})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "워크스페이스 생성 권한이 없습니다"})
	}
	// --- End Permission Check ---

	var workspace model.Workspace
	if err := c.Bind(&workspace); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	// Ensure ID is not set by client
	workspace.ID = 0

	if err := h.workspaceService.Create(&workspace); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 생성 실패: %v", err)})
	}

	// --- Assign creator as admin ---
	adminRole, err := h.workspaceRoleRepo.GetByName("admin") // Find the 'admin' role
	if err != nil {
		log.Printf("Error finding 'admin' workspace role: %v. Cannot auto-assign creator.", err)
		// Continue without assigning role, but log the issue
	} else {
		// Assign the role (ignore error for now, just log it)
		// Note: AssignRoleToUser expects DB User ID
		errAssign := h.workspaceRoleService.AssignRoleToUser(userID, adminRole.ID, workspace.ID)
		if errAssign != nil {
			log.Printf("Warning: Failed to auto-assign admin role to creator (User DB ID: %d) for new workspace %d: %v", userID, workspace.ID, errAssign)
		} else {
			log.Printf("Auto-assigned admin role to creator (User DB ID: %d) for new workspace %d", userID, workspace.ID)
		}
	}
	// --- End Assign creator ---

	// Return the created workspace, including the generated ID
	return c.JSON(http.StatusCreated, workspace)
}

// ListWorkspaces godoc
// @Summary 모든 워크스페이스 조회
// @Description 모든 워크스페이스 목록을 조회합니다 (연결된 프로젝트 정보 포함).
// @Tags workspaces
// @Accept json
// @Produce json
// @Success 200 {array} model.Workspace
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Failure 403 {object} map[string]string "error: 권한 부족"
// @Security BearerAuth
// @Router /workspaces [get]
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
// @Summary ID로 워크스페이스 조회
// @Description ID로 특정 워크스페이스를 조회합니다 (연결된 프로젝트 정보 포함).
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path int true "워크스페이스 ID"
// @Success 200 {object} model.Workspace
// @Failure 400 {object} map[string]string "error: 잘못된 워크스페이스 ID"
// @Failure 404 {object} map[string]string "error: 워크스페이스를 찾을 수 없습니다"
// @Failure 403 {object} map[string]string "error: 접근 권한 없음"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspaces/{id} [get]
func (h *WorkspaceHandler) GetWorkspaceByID(c echo.Context) error {
	// --- Permission Check ---
	userID, platformRoles, err := getUserDbIdAndPlatformRoles(c.Request().Context(), c, h.userService) // Pass context
	if err != nil {
		log.Printf("Error getting user info for GetWorkspaceByID: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to identify user"})
	}

	workspaceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID입니다"})
	}

	// Check for 'list_all' platform permission first
	hasListAllPermission, err := checkPlatformPermission(h.permissionRepo, platformRoles, "mc-iam-manager:workspace:list_all") // Use helper
	if err != nil {
		log.Printf("Error checking list_all workspace permission for user %d: %v", userID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "권한 확인 중 오류 발생"})
	}
	hasReadPermission := false

	if !hasListAllPermission {
		// If no list_all, check specific read permission for this workspace
		userRolesInWs, err := h.userService.GetUserRolesInWorkspace(userID, uint(workspaceID))
		if err != nil {
			log.Printf("Error getting user roles in workspace %d for permission check: %v", workspaceID, err)
			// Don't expose internal error, treat as forbidden
			return c.JSON(http.StatusForbidden, map[string]string{"error": "워크스페이스 접근 권한이 없습니다"})
		}
		if len(userRolesInWs) == 0 {
			// User has no roles in this workspace
			return c.JSON(http.StatusForbidden, map[string]string{"error": "워크스페이스 접근 권한이 없습니다"})
		}

		// Check if any of the user's roles in this workspace have the 'read' permission
		requiredPermissionID := "mc-iam-manager:workspace:read"
		for _, userRole := range userRolesInWs {
			hasPerm, errCheck := h.permissionRepo.CheckRoleMciamPermission("workspace", userRole.WorkspaceRoleID, requiredPermissionID) // Use actual repo method
			if errCheck != nil {
				log.Printf("Error checking permission %s for role %d: %v", requiredPermissionID, userRole.WorkspaceRoleID, errCheck)
				// Return error on check failure
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "권한 확인 중 오류 발생"})
			}
			if hasPerm {
				hasReadPermission = true
				break
			}
		}
	}

	// If user doesn't have list_all and doesn't have specific read permission
	if !hasListAllPermission && !hasReadPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "워크스페이스 접근 권한이 없습니다"})
	}
	// --- End Permission Check ---

	workspace, err := h.workspaceService.GetByID(uint(workspaceID))
	if err != nil {
		// Check for specific "not found" error (should be handled by service now)
		if err.Error() == "workspace not found" { // Assuming service returns this error string
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 조회 실패: %v", err)})
	}
	return c.JSON(http.StatusOK, workspace)
}

// GetWorkspaceByName godoc
// @Summary 이름으로 워크스페이스 조회
// @Description 이름으로 특정 워크스페이스를 조회합니다 (연결된 프로젝트 정보 포함).
// @Tags workspaces
// @Accept json
// @Produce json
// @Param name path string true "워크스페이스 이름"
// @Success 200 {object} model.Workspace
// @Failure 404 {object} map[string]string "error: 워크스페이스를 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspaces/name/{name} [get]
func (h *WorkspaceHandler) GetWorkspaceByName(c echo.Context) error {
	name := c.Param("name")

	workspace, err := h.workspaceService.GetByName(name)
	if err != nil {
		if err.Error() == "workspace not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 조회 실패: %v", err)})
	}
	return c.JSON(http.StatusOK, workspace)
}

// UpdateWorkspace godoc
// @Summary 워크스페이스 수정
// @Description 기존 워크스페이스 정보를 부분적으로 수정합니다.
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path int true "워크스페이스 ID"
// @Param updates body object true "수정할 필드와 값 (예: {\"name\": \"New Name\", \"description\": \"New Desc\"})"
// @Success 200 {object} model.Workspace "업데이트된 워크스페이스 정보"
// @Failure 400 {object} map[string]string "error: 잘못된 요청 형식 또는 ID"
// @Failure 404 {object} map[string]string "error: 워크스페이스를 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspaces/{id} [put]
func (h *WorkspaceHandler) UpdateWorkspace(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID입니다"})
	}

	updates := make(map[string]interface{})
	if err := c.Bind(&updates); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("잘못된 요청 형식입니다: %v", err)})
	}

	// Prevent updating ID or CreatedAt/UpdatedAt via map
	delete(updates, "id")
	delete(updates, "created_at")
	delete(updates, "updated_at")
	delete(updates, "projects") // Prevent direct update of associations

	if len(updates) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "업데이트할 필드가 없습니다"})
	}

	if err := h.workspaceService.Update(uint(id), updates); err != nil {
		if err.Error() == "workspace not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 업데이트 실패: %v", err)})
	}

	// Return updated workspace
	updatedWorkspace, err := h.workspaceService.GetByID(uint(id))
	if err != nil {
		// Log error but return success as update itself was successful
		fmt.Printf("Warning: Failed to fetch updated workspace (id: %d): %v\n", id, err)
		return c.JSON(http.StatusOK, updates) // Return updates map as confirmation
	}
	return c.JSON(http.StatusOK, updatedWorkspace)
}

// DeleteWorkspace godoc
// @Summary 워크스페이스 삭제
// @Description 워크스페이스를 삭제합니다. 연결된 프로젝트와의 관계도 해제됩니다.
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path int true "워크스페이스 ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "error: 잘못된 워크스페이스 ID"
// @Failure 404 {object} map[string]string "error: 워크스페이스를 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspaces/{id} [delete]
func (h *WorkspaceHandler) DeleteWorkspace(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID입니다"})
	}

	if err := h.workspaceService.Delete(uint(id)); err != nil {
		if err.Error() == "workspace not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 삭제 실패: %v", err)})
	}
	return c.NoContent(http.StatusNoContent)
}

// ListUsersAndRolesByWorkspace godoc
// @Summary 워크스페이스 사용자 및 역할 목록 조회
// @Description 특정 워크스페이스에 속한 모든 사용자와 각 사용자의 역할을 조회합니다.
// @Tags workspaces, users, roles
// @Accept json
// @Produce json
// @Param id path int true "워크스페이스 ID"
// @Success 200 {array} service.UserWithRoles "성공 시 사용자 및 역할 목록 반환"
// @Failure 400 {object} map[string]string "error: 잘못된 워크스페이스 ID"
// @Failure 404 {object} map[string]string "error: 워크스페이스를 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspaces/{id}/users [get]
func (h *WorkspaceHandler) ListUsersAndRolesByWorkspace(c echo.Context) error {
	workspaceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID입니다"})
	}

	usersWithRoles, err := h.workspaceService.GetUsersAndRolesByWorkspaceID(uint(workspaceID))
	if err != nil {
		// Handle not found error from service (which checks workspace existence)
		if errors.Is(err, repository.ErrWorkspaceNotFound) { // Assuming service passes this error up
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("사용자 및 역할 목록 조회 실패: %v", err)})
	}

	// Return empty list if no users are associated
	if usersWithRoles == nil {
		usersWithRoles = []service.UserWithRoles{} // Ensure we return [] instead of null
	}

	return c.JSON(http.StatusOK, usersWithRoles)
}

// ListProjectsByWorkspace godoc
// @Summary 워크스페이스에 연결된 프로젝트 목록 조회
// @Description 특정 워크스페이스 ID에 연결된 모든 프로젝트 목록을 조회합니다.
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path int true "워크스페이스 ID"
// @Success 200 {array} model.Project "성공 시 프로젝트 목록 반환"
// @Failure 400 {object} map[string]string "error: 잘못된 워크스페이스 ID"
// @Failure 404 {object} map[string]string "error: 워크스페이스를 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspaces/{id}/projects [get]
func (h *WorkspaceHandler) ListProjectsByWorkspace(c echo.Context) error {
	workspaceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
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
// @Summary 워크스페이스에 프로젝트 연결
// @Description 특정 워크스페이스에 프로젝트를 연결합니다.
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path int true "워크스페이스 ID"
// @Param projectId path int true "프로젝트 ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "error: 잘못된 ID 형식"
// @Failure 404 {object} map[string]string "error: 워크스페이스 또는 프로젝트를 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspaces/{id}/projects/{projectId} [post]
func (h *WorkspaceHandler) AddProjectToWorkspace(c echo.Context) error {
	workspaceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
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

// RemoveProjectFromWorkspace godoc
// @Summary 워크스페이스에서 프로젝트 연결 해제
// @Description 특정 워크스페이스에서 프로젝트 연결을 해제합니다.
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path int true "워크스페이스 ID"
// @Param projectId path int true "프로젝트 ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "error: 잘못된 ID 형식"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspaces/{id}/projects/{projectId} [delete]
func (h *WorkspaceHandler) RemoveProjectFromWorkspace(c echo.Context) error {
	workspaceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID입니다"})
	}
	projectID, err := strconv.ParseUint(c.Param("projectId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 프로젝트 ID입니다"})
	}

	if err := h.workspaceService.RemoveProjectFromWorkspace(uint(workspaceID), uint(projectID)); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("프로젝트 연결 해제 실패: %v", err)})
	}
	return c.NoContent(http.StatusNoContent)
}
