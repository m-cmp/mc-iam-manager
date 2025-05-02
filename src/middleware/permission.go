package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
)

// PermissionLevel은 권한 레벨을 정의합니다.
type PermissionLevel int

const (
	// Level1은 인증만 필요한 API를 위한 권한 레벨입니다.
	Level1 PermissionLevel = iota + 1
	// Level2는 기본 권한이 필요한 API를 위한 권한 레벨입니다.
	Level2
	// Level3는 특정 권한이 필요한 API를 위한 권한 레벨입니다.
	Level3
)

// PermissionMiddleware는 권한 체크를 위한 미들웨어를 생성합니다.
func PermissionMiddleware(db *gorm.DB, level PermissionLevel, requiredPermission string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Level 1: 인증만 필요한 API
			if level == Level1 {
				return next(c)
			}

			// Level 2 & 3: 권한 체크가 필요한 API
			// 1. Access Token 검증
			token := c.Get("access_token")
			if token == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Access token is required")
			}

			// 2. Level 2 권한 체크
			if level == Level2 {
				c.Logger().Debugf("PermissionMiddleware: Checking Level 2 permissions")

				// 기본 역할 확인
				if !checkRoleFromContext(c, []string{"admin", "platformAdmin"}) {
					c.Logger().Debugf("PermissionMiddleware: Basic role check failed")
					return echo.NewHTTPError(http.StatusForbidden, "Basic role required")
				}

				c.Logger().Debugf("PermissionMiddleware: Basic role check passed, checking basic permissions")

				// 기본 권한 확인
				hasBasicPermission, err := checkBasicPermission(c, db)
				if err != nil {
					c.Logger().Errorf("Basic permission check failed: %v", err)
					return echo.NewHTTPError(http.StatusInternalServerError, "Permission check failed")
				}
				if !hasBasicPermission {
					c.Logger().Debugf("PermissionMiddleware: Basic permission check failed")
					return echo.NewHTTPError(http.StatusForbidden, "Basic permission required")
				}

				c.Logger().Debugf("PermissionMiddleware: All checks passed")
				return next(c)
			}

			// 3. Level 3 권한 체크
			if level == Level3 {
				if requiredPermission == "" {
					return echo.NewHTTPError(http.StatusForbidden, "Permission is required")
				}

				hasPermission, err := checkPermission(c, db, requiredPermission)
				if err != nil {
					c.Logger().Errorf("Permission check failed: %v", err)
					return echo.NewHTTPError(http.StatusInternalServerError, "Permission check failed")
				}
				if !hasPermission {
					return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("Permission denied: %s", requiredPermission))
				}
			}

			return next(c)
		}
	}
}

// checkBasicPermission은 사용자가 기본 권한을 가지고 있는지 확인합니다.
func checkBasicPermission(c echo.Context, db *gorm.DB) (bool, error) {
	// 1. 사용자 ID 가져오기
	kcUserID, ok := c.Get("kcUserId").(string)
	if !ok || kcUserID == "" {
		return false, fmt.Errorf("invalid user ID")
	}

	// 2. 사용자 정보 조회
	userRepo := repository.NewUserRepository(db)
	user, err := userRepo.FindByKcID(kcUserID)
	if err != nil {
		return false, fmt.Errorf("failed to find user: %v", err)
	}
	if user == nil {
		return false, fmt.Errorf("user not found")
	}

	// 3. 사용자의 플랫폼 역할 조회
	var userRoles []model.UserPlatformRole
	if err := db.Where("user_id = ?", user.ID).Find(&userRoles).Error; err != nil {
		return false, fmt.Errorf("failed to get user roles: %v", err)
	}

	// 4. 역할별 기본 권한 확인
	permissionRepo := repository.NewMciamPermissionRepository(db)
	for _, userRole := range userRoles {
		permissions, err := permissionRepo.GetRoleMciamPermissions("platform", userRole.PlatformRoleID)
		if err != nil {
			c.Logger().Warnf("Failed to get permissions for role %d: %v", userRole.PlatformRoleID, err)
			continue
		}

		// 기본 권한 확인
		for _, permissionID := range permissions {
			// 메뉴 조회 권한
			if strings.HasPrefix(permissionID, "menu:menu:view:") {
				return true, nil
			}
			// 워크스페이스 조회 권한
			if permissionID == "mc-iam-manager:workspace:read" {
				return true, nil
			}
		}
	}

	return false, nil
}

// checkPermission은 사용자가 특정 권한을 가지고 있는지 확인합니다.
func checkPermission(c echo.Context, db *gorm.DB, requiredPermission string) (bool, error) {
	// 1. 사용자 ID 가져오기
	kcUserID, ok := c.Get("kcUserId").(string)
	if !ok || kcUserID == "" {
		return false, fmt.Errorf("invalid user ID")
	}

	// 2. 사용자 정보 조회
	userRepo := repository.NewUserRepository(db)
	user, err := userRepo.FindByKcID(kcUserID)
	if err != nil {
		return false, fmt.Errorf("failed to find user: %v", err)
	}
	if user == nil {
		return false, fmt.Errorf("user not found")
	}

	// 3. 사용자의 역할 목록 조회
	roleRepo := repository.NewPlatformRoleRepository(db)
	roles, err := roleRepo.List()
	if err != nil {
		return false, fmt.Errorf("failed to list roles: %v", err)
	}

	// 4. 역할별 권한 매핑 확인
	permissionRepo := repository.NewMciamPermissionRepository(db)
	mappingRepo := repository.NewMcmpApiPermissionActionMappingRepository(db)

	// 4.1 플랫폼 역할 권한 확인
	for _, role := range roles {
		permissions, err := permissionRepo.GetRoleMciamPermissions("platform", role.ID)
		if err != nil {
			c.Logger().Warnf("Failed to get permissions for platform role %d: %v", role.ID, err)
			continue
		}

		for _, permissionID := range permissions {
			actions, err := mappingRepo.GetActionsByPermissionID(context.Background(), permissionID)
			if err != nil {
				c.Logger().Warnf("Failed to get actions for permission %s: %v", permissionID, err)
				continue
			}

			for _, action := range actions {
				if action.ActionName == requiredPermission {
					return true, nil
				}
			}
		}
	}

	// 4.2 워크스페이스 역할 권한 확인
	workspaceID := c.Param("workspaceId")
	if workspaceID != "" {
		workspaceRoleRepo := repository.NewWorkspaceRoleRepository(db)
		workspaceRoles, err := workspaceRoleRepo.List()
		if err != nil {
			return false, fmt.Errorf("failed to list workspace roles: %v", err)
		}

		for _, role := range workspaceRoles {
			permissions, err := permissionRepo.GetRoleMciamPermissions("workspace", role.ID)
			if err != nil {
				c.Logger().Warnf("Failed to get permissions for workspace role %d: %v", role.ID, err)
				continue
			}

			for _, permissionID := range permissions {
				actions, err := mappingRepo.GetActionsByPermissionID(context.Background(), permissionID)
				if err != nil {
					c.Logger().Warnf("Failed to get actions for permission %s: %v", permissionID, err)
					continue
				}

				for _, action := range actions {
					if action.ActionName == requiredPermission {
						return true, nil
					}
				}
			}
		}
	}

	return false, nil
}

// checkRoleFromContext는 컨텍스트에서 역할을 확인합니다.
func checkRoleFromContext(c echo.Context, requiredRoles []string) bool {
	// 1. 컨텍스트에서 역할 목록 가져오기
	platformRolesValue := c.Get("platformRoles")
	if platformRolesValue == nil {
		c.Logger().Debug("No platform roles found in context")
		return false
	}

	// 2. 역할 목록 타입 확인
	platformRoles, ok := platformRolesValue.([]string)
	if !ok {
		c.Logger().Debugf("Invalid platform roles type: %T", platformRolesValue)
		return false
	}

	c.Logger().Debugf("checkRoleFromContext: User Roles from Context: %v, Required: %v", platformRoles, requiredRoles)

	// 3. 필요한 역할 확인
	for _, role := range platformRoles {
		for _, requiredRole := range requiredRoles {
			if role == requiredRole {
				c.Logger().Debugf("Found matching role: %s", role)
				return true
			}
		}
	}

	c.Logger().Debugf("No matching roles found. User roles: %v, Required roles: %v", platformRoles, requiredRoles)
	return false
}
