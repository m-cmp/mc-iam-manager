package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/config"
	"github.com/m-cmp/mc-iam-manager/model/idp"
)

type AuthHandler struct {
	keycloakConfig *config.KeycloakConfig
}

func NewAuthHandler(keycloakConfig *config.KeycloakConfig) *AuthHandler {
	return &AuthHandler{
		keycloakConfig: keycloakConfig,
	}
}

// Login godoc
// @Summary 로그인
// @Description OIDC 로그인을 시작합니다
// @Tags auth
// @Accept json
// @Produce json
// @Success 302 {string} string "리다이렉트 URL"
// @Router /auth/login [get]
func (h *AuthHandler) Login(c echo.Context) error {
	// username := c.FormValue("username")
	// password := c.FormValue("password")

	// if username == "" || password == "" {
	// 	return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자명과 비밀번호가 필요합니다"})
	// }

	var userLogin idp.UserLogin
	if err := c.Bind(&userLogin); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	token, err := h.keycloakConfig.GetToken(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "인증에 실패했습니다"})
	}

	return c.JSON(http.StatusOK, token)
}

// Callback godoc
// @Summary OIDC 콜백
// @Description OIDC 인증 후 콜백을 처리합니다
// @Tags auth
// @Accept json
// @Produce json
// @Param code query string true "인증 코드"
// @Param state query string true "상태"
// @Success 200 {object} map[string]interface{}
// @Router /auth/callback [get]
// func (h *AuthHandler) Callback(c echo.Context) error {
// 	code := c.QueryParam("code")
// 	state := c.QueryParam("state")

// 	if state != "state" {
// 		return c.JSON(http.StatusBadRequest, map[string]string{
// 			"error": "잘못된 상태값",
// 		})
// 	}

// 	token, err := h.config.Exchange(c.Request().Context(), code)
// 	if err != nil {
// 		return c.JSON(http.StatusInternalServerError, map[string]string{
// 			"error": "토큰 교환 실패",
// 		})
// 	}

// 	return c.JSON(http.StatusOK, token)
// }

func (h *AuthHandler) Logout(c echo.Context) error {
	// Keycloak에서는 클라이언트 측에서 토큰을 삭제하면 됩니다
	return c.JSON(http.StatusOK, map[string]string{"message": "로그아웃되었습니다"})
}

func (h *AuthHandler) RefreshToken(c echo.Context) error {
	refreshToken := c.FormValue("refresh_token")
	if refreshToken == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "리프레시 토큰이 필요합니다"})
	}

	token, err := h.keycloakConfig.GetToken(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "토큰 갱신에 실패했습니다"})
	}

	return c.JSON(http.StatusOK, token)
}
