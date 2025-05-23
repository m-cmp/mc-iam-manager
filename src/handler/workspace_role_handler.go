package handler

import (
	"context"
	"errors" // Ensure errors is imported
	"fmt"    // Ensure fmt is imported
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm" // Import gorm
)

type WorkspaceRoleHandler struct {
	service         *service.WorkspaceRoleService
	userService     *service.UserService
	keycloakService service.KeycloakService
	// db *gorm.DB // Not needed directly in handler
}

func NewWorkspaceRoleHandler(db *gorm.DB) *WorkspaceRoleHandler {
	workspaceRoleService := service.NewWorkspaceRoleService(db)
	userService := service.NewUserService(db)
	keycloakService := service.NewKeycloakService()
	return &WorkspaceRoleHandler{
		service:         workspaceRoleService,
		userService:     userService,
		keycloakService: keycloakService,
	}
}

// GetWorkspaceIDFromToken 토큰에서 워크스페이스 ID를 추출합니다.
func (h *WorkspaceRoleHandler) GetWorkspaceIDFromToken(tokenString string) (string, error) {
	// 토큰 파싱
	claims, err := h.keycloakService.ValidateTokenAndGetClaims(context.Background(), tokenString)
	if err != nil {
		return "", fmt.Errorf("failed to validate token: %w", err)
	}

	// authorization.permissions[0].rsid에서 workspace_id 추출
	authorization, ok := (*claims)["authorization"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("authorization claim not found in token")
	}

	permissions, ok := authorization["permissions"].([]interface{})
	if !ok || len(permissions) == 0 {
		return "", fmt.Errorf("permissions not found or empty in token")
	}

	permission, ok := permissions[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid permission format in token")
	}

	workspaceID, ok := permission["rsid"].(string)
	if !ok {
		return "", fmt.Errorf("rsid not found in permission")
	}

	// rsname이 "workspace"인지 확인
	rsname, ok := permission["rsname"].(string)
	if !ok || rsname != "workspace" {
		return "", fmt.Errorf("invalid resource name in permission")
	}

	return workspaceID, nil
}

// List godoc
// @Summary 워크스페이스 역할 목록 조회
// @Description 모든 워크스페이스 역할 목록을 조회합니다.
// @Tags workspace-roles
// @Accept json
// @Produce json
// @Success 200 {array} model.WorkspaceRole
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /api/workspace-roles [get]
func (h *WorkspaceRoleHandler) List(c echo.Context) error {
	roles, err := h.service.List()
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, roles)
}

// GetByID godoc
// @Summary 워크스페이스 역할 조회
// @Description ID로 워크스페이스 역할을 조회합니다.
// @Tags workspace-roles
// @Accept json
// @Produce json
// @Param id path int true "역할 ID"
// @Success 200 {object} model.WorkspaceRole
// @Failure 400 {object} map[string]string "error: 잘못된 ID 형식"
// @Failure 404 {object} map[string]string "error: 역할을 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /api/workspace-roles/{id} [get]
func (h *WorkspaceRoleHandler) GetByID(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}
	role, err := h.service.GetByID(uint(id))
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, role)
}

// Create godoc
// @Summary 워크스페이스 역할 생성
// @Description 새로운 워크스페이스 역할을 생성합니다.
// @Tags workspace-roles
// @Accept json
// @Produce json
// @Param role body model.WorkspaceRole true "역할 정보"
// @Success 201 {object} model.WorkspaceRole
// @Failure 400 {object} map[string]string "error: 잘못된 요청 데이터"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /api/workspace-roles [post]
func (h *WorkspaceRoleHandler) Create(c echo.Context) error {
	var role model.WorkspaceRole
	if err := c.Bind(&role); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	if err := h.service.Create(&role); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, role)
}

// Update godoc
// @Summary 워크스페이스 역할 수정
// @Description 기존 워크스페이스 역할을 수정합니다.
// @Tags workspace-roles
// @Accept json
// @Produce json
// @Param id path int true "역할 ID"
// @Param role body model.WorkspaceRole true "역할 정보"
// @Success 200 {object} model.WorkspaceRole
// @Failure 400 {object} map[string]string "error: 잘못된 요청 데이터"
// @Failure 404 {object} map[string]string "error: 역할을 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /api/workspace-roles/{id} [put]
func (h *WorkspaceRoleHandler) Update(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}
	var role model.WorkspaceRole
	role.ID = uint(id)
	if err := c.Bind(&role); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	if err := h.service.Update(&role); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, role)
}

// Delete godoc
// @Summary 워크스페이스 역할 삭제
// @Description 워크스페이스 역할을 삭제합니다.
// @Tags workspace-roles
// @Accept json
// @Produce json
// @Param id path int true "역할 ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "error: 잘못된 ID 형식"
// @Failure 404 {object} map[string]string "error: 역할을 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /api/workspace-roles/{id} [delete]
func (h *WorkspaceRoleHandler) Delete(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}
	if err := h.service.Delete(uint(id)); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent) // Use http status constant
}

// AssignWorkspaceRoleToUser assigns a workspace role to a user
// @Summary Assign a workspace role to a user
// @Description Assigns a workspace role to a user by their ID or username
// @Tags workspace-roles
// @Accept json
// @Produce json
// @Param request body map[string]string true "Request body containing 'workspaceId', 'role', and either 'id' or 'username'"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/workspaces/assign/workspace-roles [post]
func (h *WorkspaceRoleHandler) AssignWorkspaceRoleToUser(c echo.Context) error {
	// 요청 본문에서 파라미터 가져오기
	var requestBody map[string]string
	if err := c.Bind(&requestBody); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body format",
		})
	}

	// 필수 파라미터 확인
	workspaceId, exists := requestBody["workspaceId"]
	if !exists {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Workspace ID is required",
		})
	}

	roleName, exists := requestBody["role"]
	if !exists {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Role name is required",
		})
	}

	var user *model.User
	var err error

	// ID 또는 username으로 사용자 찾기
	if id, exists := requestBody["id"]; exists {
		// ID를 uint로 변환
		userIDUint, err := strconv.ParseUint(id, 10, 32)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid user ID format",
			})
		}
		user, err = h.userService.GetUserByID(c.Request().Context(), uint(userIDUint))
	} else if username, exists := requestBody["username"]; exists {
		user, err = h.userService.GetUserByUsername(c.Request().Context(), username)
	} else {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Either 'id' or 'username' is required",
		})
	}

	// 사용자 찾기 오류 처리
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "User not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": fmt.Sprintf("Failed to find user: %v", err),
		})
	}

	// 워크스페이스 ID를 uint로 변환
	workspaceIDUint, err := strconv.ParseUint(workspaceId, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid workspace ID format",
		})
	}

	// 워크스페이스 역할 찾기
	workspaceRole, err := h.service.GetWorkspaceRoleByName(roleName)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"error": "Workspace role not found",
		})
	}

	// 워크스페이스 역할 할당
	err = h.service.AssignWorkspaceRoleToUser(user.ID, workspaceRole.ID, uint(workspaceIDUint))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": fmt.Sprintf("Failed to assign role: %v", err),
		})
	}

	// // Keycloak 그룹 동기화
	// groupName := fmt.Sprintf("ws_%d_%s", workspaceIDUint, workspaceRole.Name)
	// if err := h.keycloakService.EnsureGroupExistsAndAssignUser(c.Request().Context(), user.KcId, groupName); err != nil {
	// 	log.Printf("Warning: Failed to assign user %s to Keycloak group %s: %v", user.Username, groupName, err)
	// } else {
	// 	log.Printf("Successfully assigned user %s to Keycloak group %s", user.Username, groupName)
	// }

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("Successfully assigned role %s to user %s in workspace %s", roleName, user.Username, workspaceId),
	})
}

// RemoveWorkspaceRoleFromUser godoc
// @Summary 워크스페이스 사용자 워크스페이스 역할 제거
// @Description 특정 워크스페이스 내의 사용자에게서 특정 워크스페이스 역할을 제거합니다.
// @Tags workspaces, workspace-roles, users
// @Accept json
// @Produce json
// @Param workspaceId path int true "워크스페이스 ID"
// @Param username path string true "사용자 이름"
// @Param workspaceRoleName path string true "워크스페이스 역할 이름"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "error: 잘못된 ID 형식"
// @Failure 404 {object} map[string]string "error: 워크스페이스 역할 또는 워크스페이스를 찾을 수 없습니다"
// @Failure 409 {object} map[string]string "error: 워크스페이스 역할이 해당 워크스페이스에 속하지 않음"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /api/workspaces/{workspaceId}/users/{username}/roles/{workspaceRoleName} [delete]
func (h *WorkspaceRoleHandler) RemoveWorkspaceRoleFromUser(c echo.Context) error {
	workspaceID, err := strconv.ParseUint(c.Param("workspaceId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식입니다"})
	}

	username := c.Param("username")
	workspaceRoleName := c.Param("workspaceRoleName")

	// 1. 사용자명으로 DB ID 찾기
	user, err := h.userService.GetUserByUsername(c.Request().Context(), username)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "사용자를 찾을 수 없습니다"})
	}

	// 2. 워크스페이스 역할명으로 DB ID 찾기
	workspaceRole, err := h.service.GetWorkspaceRoleByName(workspaceRoleName)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "워크스페이스 역할을 찾을 수 없습니다"})
	}

	// // 3. Keycloak 그룹에서 사용자 제거
	// groupName := fmt.Sprintf("ws_%d_%s", workspaceID, workspaceRole.Name)
	// if err := h.keycloakService.RemoveUserFromGroup(c.Request().Context(), user.KcId, groupName); err != nil {
	// 	log.Printf("Warning: Failed to remove user %s from Keycloak group %s: %v", user.Username, groupName, err)
	// } else {
	// 	log.Printf("Successfully removed user %s from Keycloak group %s", user.Username, groupName)
	// }

	// 4. DB에서 워크스페이스 역할 제거
	err = h.service.RemoveWorkspaceRoleFromUser(user.ID, workspaceRole.ID, uint(workspaceID))
	if err != nil {
		if errors.Is(err, service.ErrWorkspaceRoleNotFound) || errors.Is(err, service.ErrWorkspaceNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, service.ErrWorkspaceRoleNotInWorkspace) {
			return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 역할 제거 실패: %v", err)})
	}

	return c.NoContent(http.StatusNoContent)
}

// GetUserWorkspaceRoles godoc
// @Summary 사용자의 워크스페이스 역할 목록 조회
// @Description 특정 워크스페이스에서 사용자에게 할당된 워크스페이스 역할 목록을 조회합니다.
// @Tags workspaces, workspace-roles, users
// @Accept json
// @Produce json
// @Param workspaceId path int true "워크스페이스 ID"
// @Param username path string true "사용자 이름"
// @Success 200 {array} string "워크스페이스 역할 이름 목록"
// @Failure 400 {object} map[string]string "error: 잘못된 ID 형식"
// @Failure 404 {object} map[string]string "error: 사용자를 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /api/workspaces/{workspaceId}/users/{username}/roles [get]
func (h *WorkspaceRoleHandler) GetUserWorkspaceRoles(c echo.Context) error {
	workspaceID, err := strconv.ParseUint(c.Param("workspaceId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식입니다"})
	}

	username := c.Param("username")

	// 1. 사용자명으로 DB ID 찾기
	user, err := h.userService.GetUserByUsername(c.Request().Context(), username)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "사용자를 찾을 수 없습니다"})
	}

	// 2. 사용자의 워크스페이스 역할 조회
	workspaceRoles, err := h.service.GetUserWorkspaceRoles(user.ID, uint(workspaceID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 역할 목록 조회 실패: %v", err)})
	}

	return c.JSON(http.StatusOK, workspaceRoles)
}
