package handler

import (
	"errors" // Ensure errors is imported
	"fmt"    // Ensure fmt is imported
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm" // Import gorm
)

type WorkspaceRoleHandler struct {
	service *service.WorkspaceRoleService
	// db *gorm.DB // Not needed directly in handler
}

func NewWorkspaceRoleHandler(db *gorm.DB) *WorkspaceRoleHandler { // Accept db, remove service param
	// Initialize service internally
	workspaceRoleService := service.NewWorkspaceRoleService(db)
	return &WorkspaceRoleHandler{
		service: workspaceRoleService,
	}
}

func (h *WorkspaceRoleHandler) List(c echo.Context) error {
	roles, err := h.service.List()
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, roles)
}

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

// AssignRoleToUser godoc
// @Summary 워크스페이스 사용자에게 역할 할당
// @Description 특정 워크스페이스 내의 사용자에게 특정 워크스페이스 역할을 할당합니다.
// @Tags workspaces, roles, users
// @Accept json
// @Produce json
// @Param workspaceId path int true "워크스페이스 ID"
// @Param userId path int true "사용자 DB ID (db_id)"
// @Param roleId path int true "워크스페이스 역할 ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "error: 잘못된 ID 형식"
// @Failure 404 {object} map[string]string "error: 사용자, 역할 또는 워크스페이스를 찾을 수 없습니다"
// @Failure 409 {object} map[string]string "error: 역할이 해당 워크스페이스에 속하지 않음"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspaces/{workspaceId}/users/{userId}/roles/{roleId} [post]
func (h *WorkspaceRoleHandler) AssignRoleToUser(c echo.Context) error {
	workspaceID, err := strconv.ParseUint(c.Param("workspaceId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식입니다"})
	}
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32) // Assuming userId is the DB ID (uint)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 사용자 ID 형식입니다"})
	}
	roleID, err := strconv.ParseUint(c.Param("roleId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID 형식입니다"})
	}

	err = h.service.AssignRoleToUser(uint(userID), uint(roleID), uint(workspaceID))
	if err != nil {
		// Handle specific errors from service
		if errors.Is(err, service.ErrUserNotFound) || errors.Is(err, service.ErrRoleNotFound) || errors.Is(err, service.ErrWorkspaceNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, service.ErrRoleNotInWorkspace) {
			return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()}) // 409 Conflict might be appropriate
		}
		// Handle potential duplicate assignment errors from DB if not handled by repo/service
		// if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		// 	return c.JSON(http.StatusConflict, map[string]string{"error": "User already has this role in the workspace"})
		// }
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 할당 실패: %v", err)})
	}

	return c.NoContent(http.StatusNoContent)
}

// RemoveRoleFromUser godoc
// @Summary 워크스페이스 사용자 역할 제거
// @Description 특정 워크스페이스 내의 사용자에게서 특정 워크스페이스 역할을 제거합니다.
// @Tags workspaces, roles, users
// @Accept json
// @Produce json
// @Param workspaceId path int true "워크스페이스 ID"
// @Param userId path int true "사용자 DB ID (db_id)"
// @Param roleId path int true "워크스페이스 역할 ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "error: 잘못된 ID 형식"
// @Failure 404 {object} map[string]string "error: 역할 또는 워크스페이스를 찾을 수 없습니다" // User existence check is optional here
// @Failure 409 {object} map[string]string "error: 역할이 해당 워크스페이스에 속하지 않음"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspaces/{workspaceId}/users/{userId}/roles/{roleId} [delete]
func (h *WorkspaceRoleHandler) RemoveRoleFromUser(c echo.Context) error {
	workspaceID, err := strconv.ParseUint(c.Param("workspaceId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식입니다"})
	}
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 사용자 ID 형식입니다"})
	}
	roleID, err := strconv.ParseUint(c.Param("roleId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID 형식입니다"})
	}

	err = h.service.RemoveRoleFromUser(uint(userID), uint(roleID), uint(workspaceID))
	if err != nil {
		// Handle specific errors from service
		if errors.Is(err, service.ErrRoleNotFound) || errors.Is(err, service.ErrWorkspaceNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, service.ErrRoleNotInWorkspace) {
			return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 제거 실패: %v", err)})
	}

	return c.NoContent(http.StatusNoContent)
}
