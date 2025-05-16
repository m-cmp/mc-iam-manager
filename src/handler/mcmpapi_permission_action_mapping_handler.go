package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm"
)

// McmpApiPermissionActionMappingHandler handles MCMP API permission action mappings
type McmpApiPermissionActionMappingHandler struct {
	service        *service.McmpApiPermissionActionMappingService
	mcmpApiService service.McmpApiService
}

// NewMcmpApiPermissionActionMappingHandler creates a new McmpApiPermissionActionMappingHandler
func NewMcmpApiPermissionActionMappingHandler(db *gorm.DB) *McmpApiPermissionActionMappingHandler {
	return &McmpApiPermissionActionMappingHandler{
		service:        service.NewMcmpApiPermissionActionMappingService(db),
		mcmpApiService: service.NewMcmpApiService(db),
	}
}

// GetPlatformActionsByPermissionID 플랫폼 권한 ID에 해당하는 액션 목록 조회
// @Summary Get platform actions by permission ID
// @Description Returns all platform actions mapped to a specific permission
// @Tags mcmp-api-permission-action-mappings
// @Accept json
// @Produce json
// @Param permissionId path string true "Permission ID"
// @Success 200 {array} mcmpapi.McmpApiAction
// @Router /api/mcmp-api-permission-action-mappings/platforms/{permissionId}/actions [get]
func (h *McmpApiPermissionActionMappingHandler) GetPlatformActionsByPermissionID(c echo.Context) error {
	permissionID := c.Param("permission_id")
	if permissionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "permission_id is required"})
	}

	actions, err := h.service.GetPlatformActionsByPermissionID(c.Request().Context(), permissionID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, actions)
}

// GetWorkspaceActionsByPermissionID 워크스페이스 권한 ID에 해당하는 액션 목록 조회
// @Summary Get workspace actions by permission ID
// @Description Returns all workspace actions mapped to a specific permission
// @Tags mcmp-api-permission-action-mappings
// @Accept json
// @Produce json
// @Param permissionId path string true "Permission ID"
// @Success 200 {array} mcmpapi.McmpApiAction
// @Router /api/mcmp-api-permission-action-mappings/workspaces/{permissionId}/actions [get]
func (h *McmpApiPermissionActionMappingHandler) GetWorkspaceActionsByPermissionID(c echo.Context) error {
	permissionID := c.Param("permission_id")
	if permissionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "permission_id is required"})
	}

	actions, err := h.service.GetWorkspaceActionsByPermissionID(c.Request().Context(), permissionID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, actions)
}

// GetPermissionsByActionID 액션 ID에 해당하는 권한 목록 조회
// @Summary Get permissions by action ID
// @Description Returns all permissions mapped to a specific API action
// @Tags mcmp-api-permission-action-mappings
// @Accept json
// @Produce json
// @Param actionId path int true "Action ID"
// @Success 200 {array} string
// @Router /api/mcmp-api-permission-action-mappings/actions/{actionId}/permissions [get]
func (h *McmpApiPermissionActionMappingHandler) GetPermissionsByActionID(c echo.Context) error {
	actionIDStr := c.Param("action_id")
	if actionIDStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "action_id is required"})
	}

	actionID, err := strconv.Atoi(actionIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid action_id"})
	}

	permissions, err := h.service.GetPermissionsByActionID(c.Request().Context(), actionID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, permissions)
}

// CreateMapping 매핑 생성
// @Summary Create permission-action mapping
// @Description Creates a new mapping between a permission and an API action
// @Tags mcmp-api-permission-action-mappings
// @Accept json
// @Produce json
// @Param mapping body mcmpapi.McmpApiPermissionActionMapping true "Mapping to create"
// @Success 204 "No Content"
// @Router /api/mcmp-api-permission-action-mappings [post]
func (h *McmpApiPermissionActionMappingHandler) CreateMapping(c echo.Context) error {
	var request struct {
		PermissionID string `json:"permission_id"`
		ActionID     int    `json:"action_id"`
		ActionName   string `json:"action_name"`
	}

	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if request.PermissionID == "" || request.ActionID == 0 || request.ActionName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "permission_id, action_id, and action_name are required"})
	}

	err := h.service.CreateMapping(c.Request().Context(), request.PermissionID, request.ActionID, request.ActionName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]string{"message": "mapping created successfully"})
}

// DeleteMapping 매핑 삭제
// @Summary Delete permission-action mapping
// @Description Deletes a mapping between a permission and an API action
// @Tags mcmp-api-permission-action-mappings
// @Accept json
// @Produce json
// @Param permissionId path string true "Permission ID"
// @Param actionId path int true "Action ID"
// @Success 204 "No Content"
// @Router /api/mcmp-api-permission-action-mappings/permissions/{permissionId}/actions/{actionId} [delete]
func (h *McmpApiPermissionActionMappingHandler) DeleteMapping(c echo.Context) error {
	permissionID := c.Param("permission_id")
	actionIDStr := c.Param("action_id")

	if permissionID == "" || actionIDStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "permission_id and action_id are required"})
	}

	actionID, err := strconv.Atoi(actionIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid action_id"})
	}

	err = h.service.DeleteMapping(c.Request().Context(), permissionID, actionID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "mapping deleted successfully"})
}

// UpdateMapping 매핑 수정
// @Summary Update permission-action mapping
// @Description Updates an existing mapping between a permission and an API action
// @Tags mcmp-api-permission-action-mappings
// @Accept json
// @Produce json
// @Param permissionId path string true "Permission ID"
// @Param actionId path int true "Action ID"
// @Param mapping body mcmpapi.McmpApiPermissionActionMapping true "Updated mapping"
// @Success 200 {object} map[string]string
// @Router /api/mcmp-api-permission-action-mappings/permissions/{permissionId}/actions/{actionId} [put]
func (h *McmpApiPermissionActionMappingHandler) UpdateMapping(c echo.Context) error {
	permissionID := c.Param("permission_id")
	actionIDStr := c.Param("action_id")

	if permissionID == "" || actionIDStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "permission_id and action_id are required"})
	}

	actionID, err := strconv.Atoi(actionIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid action_id"})
	}

	var request struct {
		ActionName string `json:"action_name"`
	}

	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if request.ActionName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "action_name is required"})
	}

	err = h.service.UpdateMapping(c.Request().Context(), permissionID, actionID, request.ActionName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "mapping updated successfully"})
}

// ListMappings godoc
// @Summary MCMP API 권한-액션 매핑 목록 조회
// @Description 모든 MCMP API 권한-액션 매핑 목록을 조회합니다
// @Tags mcmp-api-permission-action-mappings
// @Accept json
// @Produce json
// @Success 200 {array} model.MCMPAPIPermissionActionMapping
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Security BearerAuth
// @Router /api/v1/mcmp-api-permission-action-mappings [get]

// GetMappingByID godoc
// @Summary MCMP API 권한-액션 매핑 ID로 조회
// @Description 특정 MCMP API 권한-액션 매핑을 ID로 조회합니다
// @Tags mcmp-api-permission-action-mappings
// @Accept json
// @Produce json
// @Param id path string true "Mapping ID"
// @Success 200 {object} model.MCMPAPIPermissionActionMapping
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 404 {object} map[string]string "error: Mapping not found"
// @Security BearerAuth
// @Router /api/v1/mcmp-api-permission-action-mappings/{id} [get]

// CreateMapping godoc
// @Summary 새 MCMP API 권한-액션 매핑 생성
// @Description 새로운 MCMP API 권한-액션 매핑을 생성합니다
// @Tags mcmp-api-permission-action-mappings
// @Accept json
// @Produce json
// @Param mapping body model.MCMPAPIPermissionActionMapping true "Mapping Info"
// @Success 201 {object} model.MCMPAPIPermissionActionMapping
// @Failure 400 {object} map[string]string "error: Invalid request"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Security BearerAuth
// @Router /api/v1/mcmp-api-permission-action-mappings [post]

// UpdateMapping godoc
// @Summary MCMP API 권한-액션 매핑 업데이트
// @Description MCMP API 권한-액션 매핑 정보를 업데이트합니다
// @Tags mcmp-api-permission-action-mappings
// @Accept json
// @Produce json
// @Param id path string true "Mapping ID"
// @Param mapping body model.MCMPAPIPermissionActionMapping true "Mapping Info"
// @Success 200 {object} model.MCMPAPIPermissionActionMapping
// @Failure 400 {object} map[string]string "error: Invalid request"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 404 {object} map[string]string "error: Mapping not found"
// @Security BearerAuth
// @Router /api/v1/mcmp-api-permission-action-mappings/{id} [put]

// DeleteMapping godoc
// @Summary MCMP API 권한-액션 매핑 삭제
// @Description MCMP API 권한-액션 매핑을 삭제합니다
// @Tags mcmp-api-permission-action-mappings
// @Accept json
// @Produce json
// @Param id path string true "Mapping ID"
// @Success 204 "No Content"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 404 {object} map[string]string "error: Mapping not found"
// @Security BearerAuth
// @Router /api/v1/mcmp-api-permission-action-mappings/{id} [delete]
