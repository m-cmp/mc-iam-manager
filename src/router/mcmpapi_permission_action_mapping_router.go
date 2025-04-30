package router

import (
	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/handler"
	"github.com/m-cmp/mc-iam-manager/service"
)

// RegisterMcmpApiPermissionActionMappingRoutes registers routes for permission-action mappings.
func RegisterMcmpApiPermissionActionMappingRoutes(e *echo.Echo, service *service.McmpApiPermissionActionMappingService) {
	h := handler.NewMcmpApiPermissionActionMappingHandler(service)
	g := e.Group("/api/mcmp-api-permission-action-mappings")

	// Get all API actions mapped to a specific permission
	g.GET("/permissions/:permissionId/actions", h.GetActionsByPermissionID)

	// Get all permissions mapped to a specific API action
	g.GET("/actions/:actionId/permissions", h.GetPermissionsByActionID)

	// Create a new permission-action mapping
	g.POST("", h.CreateMapping)

	// Delete a permission-action mapping
	g.DELETE("/permissions/:permissionId/actions/:actionId", h.DeleteMapping)
}
