package middleware

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm"
)

// WorkspaceRoleMiddleware 워크스페이스 역할 기반 접근 제어 미들웨어
func WorkspaceRoleMiddleware(db *gorm.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 1. Get workspace ticket from X-Workspace-Ticket header
			workspaceTicket := c.Request().Header.Get("X-Workspace-Ticket")
			if workspaceTicket == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "X-Workspace-Ticket header is required")
			}

			// 2. Parse workspace ticket and extract information
			keycloakService := service.NewKeycloakService()
			claims, err := keycloakService.ValidateTokenAndGetClaims(c.Request().Context(), workspaceTicket)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Failed to validate workspace ticket: %v", err))
			}

			// 3. Extract authorization information
			authorization, ok := (*claims)["authorization"].(map[string]interface{})
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "authorization claim not found in workspace ticket")
			}

			permissions, ok := authorization["permissions"].([]interface{})
			if !ok || len(permissions) == 0 {
				return echo.NewHTTPError(http.StatusUnauthorized, "permissions not found in workspace ticket")
			}

			permission, ok := permissions[0].(map[string]interface{})
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid permission format in workspace ticket")
			}

			// 4. Extract and validate workspace ID
			workspaceID, ok := permission["rsid"].(string)
			if !ok || workspaceID == "" {
				return echo.NewHTTPError(http.StatusBadRequest, "workspace_id not found in ticket")
			}

			// 5. Validate resource name
			rsname, ok := permission["rsname"].(string)
			if !ok || rsname != "workspace" {
				return echo.NewHTTPError(http.StatusBadRequest, "invalid resource name in permission")
			}

			// 6. Store all necessary information in context
			c.Set("workspace_id", workspaceID)
			c.Set("workspace_ticket", workspaceTicket)
			c.Set("workspace_permissions", permission)

			return next(c)
		}
	}
}
