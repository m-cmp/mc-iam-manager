package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Nerzal/gocloak/v13" // Import gocloak
	// "github.com/m-cmp/mc-iam-manager/config" // Removed duplicate import
	// "github.com/m-cmp/mc-iam-manager/model"  // Removed duplicate import

	// Needed for token type in GetUserIDFromToken
	"gorm.io/gorm" // Ensure gorm is imported

	// "github.com/golang-jwt/jwt/v5" // No longer directly needed
	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/config" // Keep this one
	"github.com/m-cmp/mc-iam-manager/model"  // Keep this one
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
// @Success 200 {object} model.JWTResponse "로그인 성공 및 토큰 정보" // Updated response type for Swagger
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

// RefreshToken godoc
// @Summary 토큰 갱신
// @Description 리프레시 토큰을 사용하여 새로운 액세스 토큰 및 리프레시 토큰을 발급받습니다.
// @Tags auth
// @Accept x-www-form-urlencoded
// @Produce json
// @Param refresh_token formData string true "리프레시 토큰"
// @Success 200 {object} model.JWTResponse "토큰 갱신 성공 및 새 토큰 정보" // Added response type for Swagger
// @Failure 400 {object} map[string]string "error: 리프레시 토큰 누락"
// @Failure 401 {object} map[string]string "error: 토큰 갱신 실패 (유효하지 않은 토큰 등)"
// @Router /auth/refresh [post]
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
// @Summary 워크스페이스 티켓(RPT) 발급
// @Description 사용자가 선택한 워크스페이스 및 요청 권한에 대한 RPT(Requesting Party Token)를 Keycloak으로부터 발급받습니다.
// @Tags auth
// @Accept json
// @Produce json
// @Param ticketRequest body model.WorkspaceTicketRequest true "워크스페이스 ID 및 요청 권한"
// @Success 200 {object} model.JWTResponse "발급된 RPT 토큰" // Updated response type for Swagger
// @Failure 400 {object} map[string]string "error: 잘못된 요청 형식"
// @Failure 401 {object} map[string]string "error: 인증 실패 또는 유효하지 않은 토큰"
// @Failure 403 {object} map[string]string "error: 권한 부족 (Keycloak 정책 평가 실패)"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류 또는 Keycloak 통신 실패"
// @Security BearerAuth
// @Router /auth/workspace-ticket [post]
func (h *AuthHandler) WorkspaceTicket(c echo.Context) error {
	// 1. Get original Access Token from header
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authorization header is missing or invalid"})
	}
	accessToken := strings.TrimPrefix(authHeader, "Bearer ")

	// 2. Bind request body
	var req model.WorkspaceTicketRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
	}
	// Comment out c.Validate(&req) call as requested, validator is not registered in main.go
	// if err := c.Validate(&req); err != nil {
	// 	return c.JSON(http.StatusBadRequest, map[string]string{"error": "Validation failed: " + err.Error()})
	// }

	// 3. Prepare permissions for Keycloak UMA request
	// The request model expects permissions like "resource#scope".
	// We need to convert this to gocloak.Permission objects.
	// This assumes Keycloak resources are named like "resource_vm" and scopes like "scope_create".
	// The exact format depends heavily on how resources/scopes are defined in Keycloak.
	var permissions []string // Permissions should be a slice of strings like "resource_id#scope_name"
	for _, p := range req.Permissions {
		// Basic format validation (contains '#')
		if strings.Contains(p, "#") {
			permissions = append(permissions, p)
		} else {
			log.Printf("Warning: Invalid permission format in request: %s", p)
			// Optionally return bad request here
		}
	}
	if len(permissions) == 0 && len(req.Permissions) > 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "No valid permissions found in request after parsing"})
	}

	// 4. Prepare options for GetRequestingPartyToken
	// The Audience should likely be the client ID itself or a specific resource server ID
	// WorkspaceID from the request might be used to construct resource IDs or passed differently if needed by policies
	options := gocloak.RequestingPartyTokenOptions{
		GrantType:   gocloak.StringP("urn:ietf:params:oauth:grant-type:uma-ticket"),
		Permissions: &permissions,
		Audience:    &config.KC.ClientID, // Requesting token for our client
		// We might need to pass workspaceId differently, e.g., via ClaimToken if using claims-based policies
		// ClaimToken: &claimToken,
		// ClaimTokenFormat: gocloak.StringP("urn:ietf:params:oauth:token-type:jwt"),
	}

	// 5. Call KeycloakService to get RPT
	ks := service.NewKeycloakService() // Instantiate KeycloakService locally
	rpt, err := ks.GetRequestingPartyToken(c.Request().Context(), accessToken, options)
	if err != nil {
		// Check for specific Keycloak errors (e.g., 403 Forbidden)
		if strings.Contains(err.Error(), "403") {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "Permission denied by Keycloak policy"})
		}
		log.Printf("Error getting RPT: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get workspace ticket from Keycloak"})
	}

	// 6. Return the RPT
	return c.JSON(http.StatusOK, rpt)
}
