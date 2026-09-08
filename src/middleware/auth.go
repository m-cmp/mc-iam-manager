package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5" // Needed for jwt.Token if used later
	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/config"
	"github.com/m-cmp/mc-iam-manager/util"
	// "github.com/m-cmp/mc-iam-manager/model/mcmpapi" // No longer needed here
	// gocloak import might not be needed here if types aren't directly used
)

// ContextKey 타입 정의
type ContextKey string

const (
	AccessTokenKey ContextKey = "access_token"
	KcUserIdKey    ContextKey = "kc_user_id"
)

// Main middleware used in routes
func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// 1. Authorization 헤더에서 토큰 추출
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "Authorization header is required")
		}

		// Bearer 토큰 형식 확인
		parts := strings.Split(authHeader, " ")
		c.Logger().Debug("authHeader: ", authHeader)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid authorization header format")
		}

		accessToken := parts[1]
		if accessToken == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "Access token is required")
		}

		c.Set("access_token", accessToken)

		// 2. 토큰 검증
		claimsInterface, err := util.ValidateToken(accessToken)
		if err != nil {
			c.Logger().Debugf("Token validation failed: %v", err)
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
		}

		if claimsInterface == nil {
			c.Logger().Debug("Claims are nil after decoding")
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process token claims")
		}

		// 토큰 클레임을 컨텍스트에 설정
		c.Set("token_claims", claimsInterface)

		kcUserId, err := (*claimsInterface).GetSubject()
		if err != nil || kcUserId == "" {
			c.Logger().Debugf("Failed to get subject (kcUserId) from claims: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get user ID from token")
		}
		c.Set("kcUserId", kcUserId)

		// Store token and user ID in context
		ctx := context.WithValue(c.Request().Context(), AccessTokenKey, accessToken)
		ctx = context.WithValue(ctx, KcUserIdKey, kcUserId)
		c.SetRequest(c.Request().WithContext(ctx))

		// 3. 역할 추출 및 설정
		var roleStrings []string

		// 3.1 top-level roles 확인
		if roles, ok := (*claimsInterface)["roles"].([]interface{}); ok {
			for _, role := range roles {
				if roleStr, ok := role.(string); ok {
					roleStrings = append(roleStrings, roleStr)
				}
			}
			c.Logger().Debugf("Extracted platform roles from top-level: %v", roleStrings)
		}

		// 3.2 realm_access.roles == platform role 확인
		realm := config.KC.Realm
		if realm == "" {
			realm = "mcmp-demo" // 기본값 설정
		}
		clientDefaultRoleName := "default-roles-" + realm

		if realmAccess, ok := (*claimsInterface)["realm_access"].(map[string]interface{}); ok {
			if roles, ok := realmAccess["roles"].([]interface{}); ok {
				excludedRoles := map[string]bool{
					"offline_access":      true,
					"uma_authorization":   true,
					clientDefaultRoleName: true, // Releam에 따른 기본 Role 제외
				}
				for _, role := range roles {
					if roleStr, ok := role.(string); ok {
						if !excludedRoles[roleStr] {
							roleStrings = append(roleStrings, roleStr)
						}
					}
				}
				c.Logger().Debugf("Extracted platform roles from realm_access: %v", roleStrings)
			}
		}

		// 중복 제거
		uniqueRoles := make(map[string]bool)
		var finalRoles []string
		for _, role := range roleStrings {
			if !uniqueRoles[role] {
				uniqueRoles[role] = true
				finalRoles = append(finalRoles, role)
			}
		}

		c.Set("platformRoles", finalRoles)
		c.Logger().Debugf("Final platform roles set in context: %v", finalRoles)

		return next(c)
	}
}

// --- Keep the helper function for use in the handler later ---
// Helper function to extract roles from claims map
func getRolesFromClaims(claims jwt.MapClaims) (realmRoles []string, resourceRoles map[string][]string) {
	resourceRoles = make(map[string][]string)
	// Extract Realm Roles
	if realmAccess, ok := claims["realm_access"].(map[string]interface{}); ok {
		if roles, ok := realmAccess["roles"].([]interface{}); ok {
			for _, role := range roles {
				if rStr, ok := role.(string); ok {
					realmRoles = append(realmRoles, rStr)
				}
			}
		}
	}
	// Extract Resource Roles
	if resourceAccess, ok := claims["resource_access"].(map[string]interface{}); ok {
		for client, clientAccess := range resourceAccess {
			if caMap, ok := clientAccess.(map[string]interface{}); ok {
				if roles, ok := caMap["roles"].([]interface{}); ok {
					var clientRoleList []string
					for _, role := range roles {
						if rStr, ok := role.(string); ok {
							clientRoleList = append(clientRoleList, rStr)
						}
					}
					resourceRoles[client] = clientRoleList
				}
			}
		}
	}
	return realmRoles, resourceRoles
}

// --- Remove or comment out unused/problematic old middleware ---
/*
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Nerzal/gocloak/v13"
	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/config"
)

func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		accessToken := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")
		if accessToken == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "인증 토큰이 없습니다"})
		}

		// 토큰 검증
		result, err := config.KC.ValidateToken(context.Background(), accessToken)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "유효하지 않은 토큰입니다"})
		}

		// 사용자 정보 가져오기
		userInfo, err := config.KC.GetUserInfo(context.Background(), accessToken)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "사용자 정보를 가져올 수 없습니다"})
		}

		// 컨텍스트에 토큰과 사용자 정보 저장
		c.Set("access_token", accessToken)
		c.Set("user_info", userInfo)
		c.Set("token_claims", result)

		return next(c)
	}
}

func RequireRole(role string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := c.Get("token_claims").(*gocloak.IntroSpectTokenResult)
			if claims == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "토큰 정보가 없습니다"})
			}

			// 토큰이 활성화되어 있는지 확인
			if !*claims.Active {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "토큰이 만료되었습니다"})
			}

			// 현재는 역할 체크를 건너뛰고 활성화된 토큰만 허용
			return next(c)
		}
	}
}
*/
