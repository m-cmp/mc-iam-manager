package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5" // Needed for jwt.Token if used later
	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/config"
	// gocloak import might not be needed here if types aren't directly used
)

// Main middleware used in routes
func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "인증 토큰이 필요합니다"})
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "잘못된 인증 형식입니다"})
		}

		token := tokenParts[1]

		// Decode the token using DecodeAccessToken
		// Assuming it returns (*jwt.Token, interface{}, error) based on previous findings
		jwtTokenDecoded, claimsInterface, err := config.KC.Client.DecodeAccessToken(c.Request().Context(), token, config.KC.Realm)
		if err != nil {
			fmt.Printf("[Middleware] Token decode/validation error: %v\n", err)
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "유효하지 않거나 만료된 토큰입니다"})
		}
		if !jwtTokenDecoded.Valid {
			fmt.Println("[Middleware] Token is decoded but reported as invalid")
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "유효하지 않은 토큰입니다 (invalid)"})
		}
		fmt.Println("[Middleware] claimsInterface ", claimsInterface)

		// Store the decoded claims (interface{}) and token in context
		c.Set("token_claims", claimsInterface) // Store the raw claims object
		c.Set("access_token", token)
		fmt.Printf("[Middleware] Successfully validated and set token_claims (type: %T)\n", claimsInterface)

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
