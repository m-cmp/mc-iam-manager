package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5" // Needed for jwt.Token if used later
	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/config"
	// "github.com/m-cmp/mc-iam-manager/model/mcmpapi" // No longer needed here
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

		// Extract and set kcUserId (subject) into context
		// claimsInterface is already *jwt.MapClaims, no need for type assertion
		if claimsInterface == nil {
			fmt.Println("[Middleware] Claims are nil after decoding")
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process token claims")
		}
		kcUserId, err := (*claimsInterface).GetSubject() // Directly use claimsInterface
		if err != nil || kcUserId == "" {
			fmt.Printf("[Middleware] Failed to get subject (kcUserId) from claims: %v\n", err) // Use err from GetSubject
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get user ID from token")
		}
		c.Set("kcUserId", kcUserId) // Set kcUserId in context

		fmt.Printf("[Middleware] Successfully validated and set token_claims and kcUserId (%s)\n", kcUserId)

		return next(c)
	}
}

// McmpApiAuthMiddleware RPT 토큰을 검증하고 명시된 권한을 확인하는 미들웨어
// requiredPermission: 이 라우트에 필요한 권한 문자열 (예: "compute#create_vm", "storage#read")
func McmpApiAuthMiddleware(requiredPermission string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				return echo.NewHTTPError(http.StatusUnauthorized, "Authorization 헤더가 없거나 형식이 잘못되었습니다.")
			}
			rptToken := strings.TrimPrefix(authHeader, "Bearer ")

			// RPT 토큰 검증 (로컬 검증)
			_, claims, err := config.KC.Client.DecodeAccessToken(c.Request().Context(), rptToken, config.KC.Realm)
			if err != nil {
				log.Printf("RPT 토큰 검증 실패: %v", err)
				return echo.NewHTTPError(http.StatusUnauthorized, "유효하지 않거나 만료된 RPT 토큰입니다.")
			}

			// RPT 페이로드에서 permissions 클레임 추출
			authClaim, ok := (*claims)["authorization"].(map[string]interface{})
			if !ok {
				log.Println("RPT 토큰에 'authorization' 클레임이 없습니다.")
				return echo.NewHTTPError(http.StatusForbidden, "권한 거부: RPT에 authorization 클레임이 없습니다.")
			}
			permissionsClaim, ok := authClaim["permissions"].([]interface{})
			if !ok {
				log.Println("RPT 토큰에 'authorization.permissions' 클레임이 없습니다.")
				return echo.NewHTTPError(http.StatusForbidden, "권한 거부: RPT에 permissions 클레임이 없습니다.")
			}

			// RPT의 permissions 클레임에 필요한 권한(requiredPermission)이 있는지 확인
			// Keycloak UMA 설정에 따라 리소스 이름(rsname)과 스코프(scopes)를 조합하여 확인해야 함.
			// 예시: requiredPermission = "resourceName#scopeName"
			hasPermission := false
			requiredParts := strings.SplitN(requiredPermission, "#", 2) // Use '#' as separator
			requiredResource := requiredParts[0]
			requiredScope := ""
			if len(requiredParts) > 1 {
				requiredScope = requiredParts[1]
			} else {
				// '#' 구분자가 없는 경우, 리소스 이름만 비교하거나 스코프가 없는 것으로 간주할 수 있음
				// 또는 에러 처리 (형식이 잘못된 requiredPermission)
				log.Printf("경고: 미들웨어에 전달된 requiredPermission 형식 오류: %s", requiredPermission)
				// return echo.NewHTTPError(http.StatusInternalServerError, "서버 설정 오류: 잘못된 권한 형식")
			}

			for _, p := range permissionsClaim {
				permMap, ok := p.(map[string]interface{})
				if !ok {
					continue
				}

				// Keycloak 설정에 따라 'rsid' 또는 'rsname' 사용
				rsname, rsnameOk := permMap["rsname"].(string) // 또는 rsid
				scopes, scopesOk := permMap["scopes"].([]interface{})
				if !rsnameOk || !scopesOk {
					continue
				}

				// 리소스 이름/ID 일치 확인
				if rsname == requiredResource {
					// 스코프 목록 확인
					for _, scopeInterface := range scopes {
						scope, ok := scopeInterface.(string)
						// 필요한 스코프가 없거나(# 구분자 없는 경우) 스코프가 일치하는 경우
						if ok && (requiredScope == "" || scope == requiredScope) {
							hasPermission = true
							break
						}
					}
				}
				if hasPermission {
					break
				}
			}

			if !hasPermission {
				log.Printf("권한 거부: '%s' 필요. RPT 권한: %v", requiredPermission, permissionsClaim)
				return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("권한 거부: '%s' 권한이 필요합니다.", requiredPermission))
			}

			// 권한 확인 완료, 다음 핸들러 실행
			// 필요시 검증된 클레임이나 사용자 정보를 컨텍스트에 저장할 수 있음
			// c.Set("rpt_claims", claims)
			// c.Set("rpt_permissions", permissionsClaim)
			return next(c)
		}
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
