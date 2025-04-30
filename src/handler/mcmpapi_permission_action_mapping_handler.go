package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model/mcmpapi"
	"github.com/m-cmp/mc-iam-manager/service"
)

// McmpApiPermissionActionMappingHandler handles HTTP requests for permission-action mappings.
type McmpApiPermissionActionMappingHandler struct {
	service *service.McmpApiPermissionActionMappingService
}

// NewMcmpApiPermissionActionMappingHandler creates a new handler instance.
func NewMcmpApiPermissionActionMappingHandler(service *service.McmpApiPermissionActionMappingService) *McmpApiPermissionActionMappingHandler {
	return &McmpApiPermissionActionMappingHandler{service: service}
}

// GetActionsByPermissionID returns all API actions mapped to a specific permission.
// @Summary Get API actions by permission ID
// @Description Returns all API actions mapped to a specific permission
// @Tags mcmp-api-permission-action-mappings
// @Accept json
// @Produce json
// @Param permissionId path string true "Permission ID"
// @Success 200 {array} mcmpapi.McmpApiAction
// @Router /api/mcmp-api-permission-action-mappings/permissions/{permissionId}/actions [get]
func (h *McmpApiPermissionActionMappingHandler) GetActionsByPermissionID(c echo.Context) error {
	permissionID := c.Param("permissionId")
	actions, err := h.service.GetActionsByPermissionID(c.Request().Context(), permissionID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get actions"})
	}
	return c.JSON(http.StatusOK, actions)
}

// GetPermissionsByActionID returns all permissions mapped to a specific API action.
// @Summary Get permissions by action ID
// @Description Returns all permissions mapped to a specific API action
// @Tags mcmp-api-permission-action-mappings
// @Accept json
// @Produce json
// @Param actionId path int true "Action ID"
// @Success 200 {array} string
// @Router /api/mcmp-api-permission-action-mappings/actions/{actionId}/permissions [get]
func (h *McmpApiPermissionActionMappingHandler) GetPermissionsByActionID(c echo.Context) error {
	actionID, err := strconv.ParseUint(c.Param("actionId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid action ID"})
	}
	permissions, err := h.service.GetPermissionsByActionID(c.Request().Context(), uint(actionID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get permissions"})
	}
	return c.JSON(http.StatusOK, permissions)
}

// CreateMapping creates a new permission-action mapping.
// @Summary Create permission-action mapping
// @Description Creates a new mapping between a permission and an API action
// @Tags mcmp-api-permission-action-mappings
// @Accept json
// @Produce json
// @Param mapping body mcmpapi.McmpApiPermissionActionMapping true "Mapping to create"
// @Success 204 "No Content"
// @Router /api/mcmp-api-permission-action-mappings [post]
func (h *McmpApiPermissionActionMappingHandler) CreateMapping(c echo.Context) error {
	var mapping mcmpapi.McmpApiPermissionActionMapping
	if err := c.Bind(&mapping); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	if err := h.service.CreateMapping(c.Request().Context(), mapping.PermissionID, mapping.ActionID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create mapping"})
	}
	return c.NoContent(http.StatusNoContent)
}

// DeleteMapping deletes a permission-action mapping.
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
	permissionID := c.Param("permissionId")
	actionID, err := strconv.ParseUint(c.Param("actionId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid action ID"})
	}
	if err := h.service.DeleteMapping(c.Request().Context(), permissionID, uint(actionID)); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete mapping"})
	}
	return c.NoContent(http.StatusNoContent)
}
