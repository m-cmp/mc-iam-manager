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

func KeycloakAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "인증 토큰이 필요합니다"})
		}

		// Bearer 토큰 추출
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "잘못된 인증 형식입니다"})
		}

		token := tokenParts[1]
		fmt.Printf("Received token: %s\n", token)

		// 토큰 검증
		claims, err := config.KC.ValidateToken(c.Request().Context(), token)
		if err != nil {
			fmt.Printf("Token validation error: %v\n", err)
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "유효하지 않은 토큰입니다"})
		}

		if !*claims.Active {
			fmt.Println("Token is not active")
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "토큰이 만료되었습니다"})
		}

		// 컨텍스트에 토큰 정보 저장
		c.Set("token_claims", claims)
		c.Set("access_token", token)

		return next(c)
	}
}
