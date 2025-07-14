package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm"
)

// CspCredentialHandler CSP 임시 자격 증명 관련 핸들러
type CspCredentialHandler struct {
	credService     *service.CspCredentialService
	keycloakService service.KeycloakService // To get user ID from token
	userService     *service.UserService
}

// NewCspCredentialHandler 새 CspCredentialHandler 인스턴스 생성
func NewCspCredentialHandler(db *gorm.DB) *CspCredentialHandler {
	credService := service.NewCspCredentialService(db)
	keycloakService := service.NewKeycloakService() // Stateless
	userService := service.NewUserService(db)
	return &CspCredentialHandler{
		credService:     credService,
		keycloakService: keycloakService,
		userService:     userService,
	}
}

// GetTemporaryCredentials godoc
// @Summary Get temporary credentials
// @Description Get temporary credentials for CSP
// @Tags csp-credentials
// @Accept json
// @Produce json
// @Security BearerAuth
// @Router /workspaces/temporary-credentials [post]
// @Id mciamGetTemporaryCredentials
func (h *CspCredentialHandler) GetTemporaryCredentials(c echo.Context) error {
	// 1. Get values from context

	// tokenString, ok := ctx.Value(middleware.AccessTokenKey).(string)
	// if !ok {
	// 	return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Access token not found in context"})
	// }

	// kcUserId, ok := ctx.Value(middleware.KcUserIdKey).(string)
	// if !ok {
	// 	return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User ID not found in context"})
	// }

	// 2. Bind request body
	var req model.CspCredentialRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
	}

	// Validate request
	if req.WorkspaceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "workspaceId is required"})
	}
	if req.CspType == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "cspType is required"})
	}

	log.Printf("Request: %+v", req)

	kcUserId := c.Get("kcUserId").(string)

	// 1. Get User's Keycloak ID from OIDC Token
	user, err := h.userService.GetUserByKcID(c.Request().Context(), kcUserId)
	if err != nil {
		log.Printf("Error finding user by KcID %s: %v", kcUserId, err)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}
	userID := user.ID

	// 2. Call the CspCredentialService with values from context
	credentials, err := h.credService.GetTemporaryCredentials(c.Request().Context(), userID, kcUserId, &req)
	if err != nil {
		log.Printf("Error: %v", err)
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

	// 4. Return the credentials
	return c.JSON(http.StatusOK, credentials)
}

// Removed placeholder ValidateTokenAndGetClaims function from handler

// ListCredentials godoc
// @Summary CSP 인증 정보 목록 조회
// @Description 모든 CSP 인증 정보 목록을 조회합니다
// @Tags csp-credentials
// @Accept json
// @Produce json
// @Security BearerAuth
// @Router /csp-credentials [get]
// @Id mciamListCredentials
func (h *CspCredentialHandler) ListCredentials(c echo.Context) error {
	// Implementation of ListCredentials method
	return nil // Placeholder return, actual implementation needed
}

// GetCredentialByID godoc
// @Summary CSP 인증 정보 ID로 조회
// @Description 특정 CSP 인증 정보를 ID로 조회합니다
// @Tags csp-credentials
// @Accept json
// @Produce json
// @Param id path string true "Credential ID"
// @Failure 404 {object} map[string]string "error: Credential not found"
// @Security BearerAuth
// @Router /csp-credentials/{id} [get]
// @Id mciamGetCredentialByID
func (h *CspCredentialHandler) GetCredentialByID(c echo.Context) error {
	// Implementation of GetCredentialByID method
	return nil // Placeholder return, actual implementation needed
}

// CreateCredential godoc
// @Summary 새 CSP 인증 정보 생성
// @Description 새로운 CSP 인증 정보를 생성합니다
// @Tags csp-credentials
// @Accept json
// @Produce json
// @Security BearerAuth
// @Router /csp-credentials [post]
// @Id mciamCreateCredential
func (h *CspCredentialHandler) CreateCredential(c echo.Context) error {
	// Implementation of CreateCredential method
	return nil // Placeholder return, actual implementation needed
}

// UpdateCredential godoc
// @Summary CSP 인증 정보 업데이트
// @Description CSP 인증 정보를 업데이트합니다
// @Tags csp-credentials
// @Accept json
// @Produce json
// @Param id path string true "Credential ID"
// @Failure 404 {object} map[string]string "error: Credential not found"
// @Security BearerAuth
// @Router /csp-credentials/{id} [put]
// @Id mciamUpdateCredential
func (h *CspCredentialHandler) UpdateCredential(c echo.Context) error {
	// Implementation of UpdateCredential method
	return nil // Placeholder return, actual implementation needed
}

// DeleteCredential godoc
// @Summary CSP 인증 정보 삭제
// @Description CSP 인증 정보를 삭제합니다
// @Tags csp-credentials
// @Accept json
// @Produce json
// @Param id path string true "Credential ID"
// @Success 204 "No Content"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 404 {object} map[string]string "error: Credential not found"
// @Security BearerAuth
// @Router /csp-credentials/{id} [delete]
// @Id mciamDeleteCredential
func (h *CspCredentialHandler) DeleteCredential(c echo.Context) error {
	// Implementation of DeleteCredential method
	return nil // Placeholder return, actual implementation needed
}
