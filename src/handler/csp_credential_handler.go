package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service" // Import service package
	"gorm.io/gorm"
)

// CspCredentialHandler CSP 임시 자격 증명 관련 핸들러
type CspCredentialHandler struct {
	credService     *service.CspCredentialService
	keycloakService service.KeycloakService // To get user ID from token
}

// NewCspCredentialHandler 새 CspCredentialHandler 인스턴스 생성
func NewCspCredentialHandler(db *gorm.DB) *CspCredentialHandler {
	credService := service.NewCspCredentialService(db)
	keycloakService := service.NewKeycloakService() // Stateless
	return &CspCredentialHandler{
		credService:     credService,
		keycloakService: keycloakService,
	}
}

// GetTemporaryCredentials godoc
// @Summary CSP 임시 자격 증명 발급
// @Description 사용자의 워크스페이스 역할에 매핑된 CSP 역할을 Assume하여 임시 자격 증명을 발급받습니다.
// @Tags csp-credentials
// @Accept json
// @Produce json
// @Param credentialRequest body model.CspCredentialRequest true "워크스페이스 ID 및 CSP 타입"
// @Success 200 {object} model.CspCredentialResponse "발급된 임시 자격 증명 (현재 AWS만 지원)"
// @Failure 400 {object} map[string]string "error: 잘못된 요청 형식 또는 지원하지 않는 CSP 타입"
// @Failure 401 {object} map[string]string "error: 인증 실패 또는 유효하지 않은 토큰"
// @Failure 403 {object} map[string]string "error: 해당 워크스페이스에 역할이 없거나 매핑된 CSP 역할이 없음"
// @Failure 404 {object} map[string]string "error: 사용자 또는 워크스페이스를 찾을 수 없음"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류 또는 CSP 통신 실패"
// @Security BearerAuth
// @Router /csp/credentials [post]
func (h *CspCredentialHandler) GetTemporaryCredentials(c echo.Context) error {
	// 1. Get original OIDC Access Token from header
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authorization header is missing or invalid"})
	}
	oidcTokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// 2. Bind request body
	var req model.CspCredentialRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Validation failed: " + err.Error()})
	}

	// 3. Get Keycloak User ID (Subject) from the token
	// We need the gocloak.JWT object for GetUserIDFromToken, but we only have the string.
	// We should ideally validate the token *and* get claims here.
	// Let's reuse the logic from AuthMiddleware or call KeycloakService's validation/decode method.
	// For simplicity here, assume we can get kcUserId (this needs proper implementation).
	// THIS IS A PLACEHOLDER - Replace with actual token validation and user ID extraction
	claims, err := h.keycloakService.ValidateTokenAndGetClaims(c.Request().Context(), oidcTokenString) // Call the actual service method
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid or expired OIDC token: " + err.Error()})
	}
	kcUserId, ok := (*claims)["sub"].(string) // Extract sub claim correctly
	if !ok || kcUserId == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Cannot extract user ID from token"})
	}

	// 4. Call the CspCredentialService with correct arguments
	credentials, err := h.credService.GetTemporaryCredentials(c.Request().Context(), kcUserId, oidcTokenString, req.WorkspaceID, req.CspType)
	if err != nil {
		// Handle specific errors from the service
		if errors.Is(err, service.ErrUserNotFound) || errors.Is(err, service.ErrWorkspaceNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, service.ErrNoCspRoleMappingFound) || strings.Contains(err.Error(), "user has no roles") {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "No suitable CSP role mapping found for user in this workspace: " + err.Error()})
		}
		if errors.Is(err, service.ErrUnsupportedCspType) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		// Handle STS specific errors (e.g., access denied by AWS)
		if strings.Contains(err.Error(), "failed to assume AWS role") {
			// Provide a more generic error to the client for security
			return c.JSON(http.StatusForbidden, map[string]string{"error": "Failed to assume target CSP role. Check IAM policies and mappings."})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to get temporary credentials: %v", err)})
	}

	// 5. Return the credentials
	return c.JSON(http.StatusOK, credentials)
}

// Removed placeholder ValidateTokenAndGetClaims function from handler
