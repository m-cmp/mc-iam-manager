package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	// Import gocloak
	// "github.com/m-cmp/mc-iam-manager/config" // Removed duplicate import
	// "github.com/m-cmp/mc-iam-manager/model"  // Removed duplicate import

	// Needed for token type in GetUserIDFromToken
	// Ensure gorm is imported

	// "github.com/golang-jwt/jwt/v5" // No longer directly needed

	"github.com/Nerzal/gocloak/v13"
	"github.com/labstack/echo/v4" // Keep this one
	"github.com/m-cmp/mc-iam-manager/config"
	"github.com/m-cmp/mc-iam-manager/model" // Keep this one
	"github.com/m-cmp/mc-iam-manager/model/idp"
	"github.com/m-cmp/mc-iam-manager/repository" // Needed for error check
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm"
)

// 사용자 인증관련 기능들을 정의함.

type AuthHandler struct {
	userService     *service.UserService
	keycloakService service.KeycloakService
}

// NewAuthHandler creates a new AuthHandler instance
func NewAuthHandler(db *gorm.DB) *AuthHandler {
	userService := service.NewUserService(db)
	keycloakService := service.NewKeycloakService()
	return &AuthHandler{
		userService:     userService,
		keycloakService: keycloakService,
	}
}

// Login godoc
// @Summary Login
// @Description Authenticate user and get token
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body model.LoginRequest true "Login Credentials"
// @Success 200 {object} model.LoginResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/auth/login [post]
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

// Logout godoc
// @Summary Logout
// @Description Logout user and invalidate token
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /api/auth/logout [post]
func (h *AuthHandler) Logout(c echo.Context) error {
	// Keycloak에서는 클라이언트 측에서 토큰을 삭제하면 됩니다
	return c.JSON(http.StatusOK, map[string]string{"message": "로그아웃되었습니다"})
}

// RefreshToken godoc
// @Summary Refresh token
// @Description Get new access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param refresh body model.RefreshTokenRequest true "Refresh Token"
// @Success 200 {object} model.LoginResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/auth/refresh [post]
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

// WorkspaceTicket godoc
// @Summary 워크스페이스 티켓 설정
// @Description 워크스페이스 티켓 설정
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "message: Workspace ticket set successfully"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Security BearerAuth
// @Router /workspaces/workspace-ticket [post]
func (h *AuthHandler) WorkspaceTicket(c echo.Context) error {
	// 1. 기존 액세스 토큰 확인
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authorization header is missing or invalid"})
	}
	accessToken := strings.TrimPrefix(authHeader, "Bearer ")

	// 2. 요청 바인딩
	var req model.WorkspaceTicketRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
	}

	// 3. 워크스페이스 ID 유효성 검사
	if _, err := strconv.ParseUint(req.WorkspaceID, 10, 32); err != nil {
		log.Printf("워크스페이스 ID 형식 오류: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid workspace ID"})
	}

	// 4. 사용자의 워크스페이스 역할 확인
	userID, err := h.userService.GetUserIDByKcID(c.Request().Context(), c.Get("kcUserId").(string))
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User not found"})
	}

	// 5. 워크스페이스 권한 조회
	workspaceRoles, err := h.userService.GetUserWorkspaceRoles(userID)
	if err != nil {
		log.Printf("워크스페이스 권한 조회 실패: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get workspace permissions"})
	}

	// 권한을 scope 형식으로 변환
	scopes := make([]string, 0)
	for _, role := range workspaceRoles {
		scopes = append(scopes, fmt.Sprintf("workspace:%s", role))
	}

	// 6. KeycloakService를 사용하여 RPT 발급
	ks := service.NewKeycloakService()

	// authorization.permissions 형식으로 클레임 토큰 생성
	claimToken := fmt.Sprintf(`{
		"authorization": {
			"permissions": [
				{
					"rsid": "%s",
					"rsname": "workspace",
					"scopes": %v
				}
			]
		}
	}`, req.WorkspaceID, scopes)

	rptOptions := gocloak.RequestingPartyTokenOptions{
		GrantType:  gocloak.StringP("urn:ietf:params:oauth:grant-type:uma-ticket"),
		Audience:   gocloak.StringP(config.KC.OIDCClientID),
		ClaimToken: gocloak.StringP(claimToken),
	}
	rpt, err := ks.GetRequestingPartyToken(c.Request().Context(), accessToken, rptOptions)
	if err != nil {
		log.Printf("RPT 발급 실패: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get RPT"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"rpt": rpt})
}

// 인증서 전달. 클라이언트에서 인증서를 parsing 해서 사용할 수 있도록 함.
func (h *AuthHandler) AuthCerts(c echo.Context) error {
	cert, err := h.keycloakService.GetCerts(c.Request().Context())
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, cert)
}
