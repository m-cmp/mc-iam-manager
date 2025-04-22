package handler

import (
	"errors"
	"fmt"
	"net/http"

	// Needed for token type in GetUserIDFromToken
	"gorm.io/gorm" // Ensure gorm is imported

	// "github.com/golang-jwt/jwt/v5" // No longer directly needed
	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model/idp"
	"github.com/m-cmp/mc-iam-manager/repository" // Needed for error check
	"github.com/m-cmp/mc-iam-manager/service"
)

type AuthHandler struct {
	// keycloakService service.KeycloakService // Removed dependency
	userService *service.UserService // Use concrete type
	// db *gorm.DB // Not needed directly
}

// NewAuthHandler creates a new AuthHandler instance
func NewAuthHandler(db *gorm.DB) *AuthHandler { // Remove keycloakService parameter
	// Initialize UserService internally
	userService := service.NewUserService(db) // Pass only db
	return &AuthHandler{
		// keycloakService: keycloakService, // Removed
		userService: userService, // Store initialized userService
	}
}

// Login godoc
// @Summary 로그인
// @Description 사용자 ID와 비밀번호로 로그인하여 JWT 토큰을 발급받습니다.
// @Tags auth
// @Accept json
// @Produce json
// @Param login body idp.UserLogin true "로그인 정보 (Id, Password)"
// @Success 200 {object} map[string]interface{} "로그인 성공 및 토큰 정보 (gocloak.JWT 구조체와 유사)"
// @Failure 400 {object} map[string]string "error: 잘못된 요청 형식"
// @Failure 401 {object} map[string]string "error: 인증 실패 (자격 증명 오류)"
// @Failure 403 {object} map[string]string "error: 계정이 비활성화되었거나 승인 대기 중입니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류 (Keycloak 통신, DB 동기화 등)"
// @Router /auth/login [post] // Changed method to POST
func (h *AuthHandler) Login(c echo.Context) error {
	var userLogin idp.UserLogin
	if err := c.Bind(&userLogin); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	// Use Id field instead of Username
	if userLogin.Id == "" || userLogin.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID와 비밀번호를 입력해주세요"})
	}

	ctx := c.Request().Context()

	// 1. Login to Keycloak using a temporary KeycloakService instance
	ks := service.NewKeycloakService()
	token, err := ks.Login(ctx, userLogin.Id, userLogin.Password)
	if err != nil {
		// Differentiate between invalid credentials and other errors if possible
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": fmt.Sprintf("Keycloak 인증 실패: %v", err)})
	}

	// 2. Get User ID (sub) from Access Token using a temporary KeycloakService instance
	userID, err := ks.GetUserIDFromToken(ctx, token)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("토큰에서 사용자 ID 추출 실패: %v", err)})
	}

	// Old claims logic removed

	// 3. Check if user is enabled in Keycloak using a temporary KeycloakService instance
	kcUser, err := ks.GetUser(ctx, userID) // Use GetUser from KeycloakService
	if err != nil {
		// Handle not found vs other errors
		if errors.Is(err, repository.ErrUserNotFound) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "Keycloak 사용자 정보를 찾을 수 없습니다 (계정 동기화 문제 가능성)"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Keycloak 사용자 정보 조회 실패: %v", err)})
	}
	if kcUser == nil || kcUser.Enabled == nil || !*kcUser.Enabled {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "계정이 비활성화되었거나 승인 대기 중입니다"})
	}

	// 4. Sync user with local DB (Create if not exists)
	_, err = h.userService.SyncUser(ctx, userID)
	if err != nil {
		// Log the error but allow login if Keycloak auth succeeded
		fmt.Printf("Warning: Failed to sync user %s with local DB: %v\n", userID, err)
		// return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("로컬 DB 동기화 실패: %v", err)})
	}

	// 5. Return Keycloak token
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

	ctx := c.Request().Context() // Get context from request

	// Use KeycloakService to refresh token
	ks := service.NewKeycloakService()
	newToken, err := ks.RefreshToken(ctx, refreshToken)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": fmt.Sprintf("토큰 갱신 실패: %v", err)})
	}

	return c.JSON(http.StatusOK, newToken)
}
