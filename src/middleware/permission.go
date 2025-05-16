package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/service"
)

// PermissionLevel은 권한 레벨을 정의합니다.
type PermissionLevel string

const (
	Read   PermissionLevel = "read"
	Write  PermissionLevel = "write"
	Manage PermissionLevel = "manage"
)

// PlatformAdminMiddleware는 플랫폼 관리자 권한이 필요한 미들웨어입니다.
func PlatformAdminMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// 플랫폼 역할 가져오기
		platformRoles, ok := c.Get("platformRoles").([]string)
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "플랫폼 역할을 가져올 수 없습니다")
		}

		// platformAdmin 역할 확인
		for _, role := range platformRoles {
			if role == "platformAdmin" {
				return next(c)
			}
		}

		return echo.NewHTTPError(http.StatusForbidden, "플랫폼 관리자 권한이 필요합니다")
	}
}

// PlatformRoleMiddleware는 플랫폼 역할 기반의 권한 체크를 수행하는 미들웨어입니다.
func PlatformRoleMiddleware(level PermissionLevel) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 플랫폼 관리자는 모든 권한을 가집니다
			if isPlatformAdmin(c) {
				return next(c)
			}

			// 플랫폼 역할 가져오기
			platformRoles, ok := c.Get("platformRoles").([]string)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "플랫폼 역할을 가져올 수 없습니다")
			}

			// 권한 체크
			hasPermission, err := checkPlatformPermission(c.Request().Context(), platformRoles, level)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "권한 체크에 실패했습니다")
			}

			if !hasPermission {
				return echo.NewHTTPError(http.StatusForbidden, "권한이 부족합니다")
			}

			return next(c)
		}
	}
}

// isPlatformAdmin는 사용자가 플랫폼 관리자인지 확인합니다.
func isPlatformAdmin(c echo.Context) bool {
	platformRoles, ok := c.Get("platformRoles").([]string)
	if !ok {
		return false
	}

	for _, role := range platformRoles {
		if role == "platformAdmin" {
			return true
		}
	}

	return false
}

// getPlatformRoles는 JWT 클레임에서 플랫폼 역할을 추출합니다.
func getPlatformRoles(claims jwt.MapClaims) ([]string, error) {
	roles, ok := claims["realm_access"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid realm_access claim")
	}

	roleList, ok := roles["roles"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid roles claim")
	}

	var platformRoles []string
	for _, role := range roleList {
		roleStr, ok := role.(string)
		if !ok {
			continue
		}
		// Platform roles don't have the "mc-iam-manager:" prefix
		if !strings.HasPrefix(roleStr, "mc-iam-manager:") {
			platformRoles = append(platformRoles, roleStr)
		}
	}

	return platformRoles, nil
}

// getWorkspaceRoles는 JWT 클레임에서 워크스페이스 역할을 추출합니다.
func getWorkspaceRoles(claims jwt.MapClaims) ([]string, error) {
	roles, ok := claims["realm_access"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid realm_access claim")
	}

	roleList, ok := roles["roles"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid roles claim")
	}

	var workspaceRoles []string
	for _, role := range roleList {
		roleStr, ok := role.(string)
		if !ok {
			continue
		}
		// Workspace roles have the "mc-iam-manager:" prefix
		if strings.HasPrefix(roleStr, "mc-iam-manager:") {
			workspaceRoles = append(workspaceRoles, roleStr)
		}
	}

	return workspaceRoles, nil
}

// checkPlatformPermission는 사용자가 필요한 플랫폼 권한을 가지고 있는지 확인합니다.
func checkPlatformPermission(ctx context.Context, userPlatformRoles []string, level PermissionLevel) (bool, error) {
	keycloakService := service.NewKeycloakService()

	// Get user's permissions from Keycloak
	permissions, err := keycloakService.GetUserPermissions(ctx, userPlatformRoles)
	if err != nil {
		return false, fmt.Errorf("failed to get user permissions: %w", err)
	}

	// Check if user has required permission level
	requiredPermission := fmt.Sprintf("platform:%s", level)
	return hasPermission(permissions, requiredPermission), nil
}

// checkWorkspacePermission는 사용자가 필요한 워크스페이스 권한을 가지고 있는지 확인합니다.
func checkWorkspacePermission(ctx context.Context, roles []string, level PermissionLevel, workspaceID string) (bool, error) {
	keycloakService := service.NewKeycloakService()

	// Get workspace ticket from context
	workspaceTicket, ok := ctx.Value("workspace_ticket").(string)
	if !ok {
		return false, fmt.Errorf("workspace ticket not found in context")
	}

	// Validate workspace ticket and get claims
	claims, err := keycloakService.ValidateTokenAndGetClaims(ctx, workspaceTicket)
	if err != nil {
		return false, fmt.Errorf("failed to validate workspace ticket: %w", err)
	}

	// Extract permissions from workspace ticket
	authorization, ok := (*claims)["authorization"].(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("authorization claim not found in workspace ticket")
	}

	permissions, ok := authorization["permissions"].([]interface{})
	if !ok || len(permissions) == 0 {
		return false, fmt.Errorf("permissions not found in workspace ticket")
	}

	// Check if the workspace ID matches and has required permission
	for _, perm := range permissions {
		permission, ok := perm.(map[string]interface{})
		if !ok {
			continue
		}

		rsid, ok := permission["rsid"].(string)
		if !ok || rsid != workspaceID {
			continue
		}

		scopes, ok := permission["scopes"].([]interface{})
		if !ok {
			continue
		}

		requiredScope := fmt.Sprintf("workspace:%s", level)
		for _, scope := range scopes {
			if scopeStr, ok := scope.(string); ok && scopeStr == requiredScope {
				return true, nil
			}
		}
	}

	return false, nil
}

// hasPermission는 사용자가 특정 권한을 가지고 있는지 확인합니다.
func hasPermission(permissions []string, requiredPermission string) bool {
	for _, permission := range permissions {
		if permission == requiredPermission {
			return true
		}
	}
	return false
}
