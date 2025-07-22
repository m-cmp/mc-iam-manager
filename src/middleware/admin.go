package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// AdminMiddleware platformAdmin 또는 admin 역할을 가진 사용자만 접근 가능한 미들웨어
func AdminMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		platformRolesInterface := c.Get("platformRoles")
		if platformRolesInterface == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token claims")
		}

		userRoles, ok := platformRolesInterface.([]string)
		if !ok {
			return echo.NewHTTPError(http.StatusInternalServerError, "Invalid platform roles format")
		}

		// platformAdmin 또는 admin 역할 확인
		isAdmin := false
		for _, role := range userRoles {
			if role == "platformAdmin" || role == "admin" {
				isAdmin = true
				break
			}
		}

		if !isAdmin {
			return echo.NewHTTPError(http.StatusForbidden, "Admin access required")
		}

		return next(c)
	}
}
