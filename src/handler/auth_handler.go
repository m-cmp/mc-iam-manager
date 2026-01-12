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

// Define user authentication related functions.

type AuthHandler struct {
	userService     *service.UserService
	keycloakService service.KeycloakService
	roleService     *service.RoleService
}

// NewAuthHandler creates a new AuthHandler instance
func NewAuthHandler(db *gorm.DB) *AuthHandler {
	userService := service.NewUserService(db)
	keycloakService := service.NewKeycloakService()
	roleService := service.NewRoleService(db)
	return &AuthHandler{
		userService:     userService,
		keycloakService: keycloakService,
		roleService:     roleService,
	}
}

// Login godoc
// @Summary User login
// @Description Authenticate user and issue JWT token.
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body idp.UserLogin true "Login Credentials"
// @Router /api/auth/login [post]
// @Id mciamLogin
func (h *AuthHandler) Login(c echo.Context) error {
	var userLogin idp.UserLogin
	if err := c.Bind(&userLogin); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	// Use Id field instead of Username
	if userLogin.Id == "" || userLogin.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Please enter user ID and password"})
	}

	ctx := c.Request().Context()

	// 1. Login to Keycloak using a temporary KeycloakService instance
	ks := service.NewKeycloakService()
	token, err := ks.Login(ctx, userLogin.Id, userLogin.Password)
	if err != nil {
		// Differentiate between invalid credentials and other errors if possible
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": fmt.Sprintf("Authentication failed: %v", err)})
	}

	// 2. Get User ID (sub) from Access Token using a temporary KeycloakService instance
	userID, err := ks.GetUserIDFromToken(ctx, token)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to extract user ID from token: %v", err)})
	}

	// Old claims logic removed

	// 3. Check if user is enabled in Keycloak using a temporary KeycloakService instance
	kcUser, err := ks.GetUser(ctx, userID) // Use GetUser from KeycloakService
	if err != nil {
		// Handle not found vs other errors
		if errors.Is(err, repository.ErrUserNotFound) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "Keycloak user information not found (possible account synchronization issue)"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to retrieve Keycloak user information: %v", err)})
	}
	if kcUser == nil || kcUser.Enabled == nil || !*kcUser.Enabled {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Account is disabled or pending approval"})
	}

	// 4. Sync user with local DB (Create if not exists)
	_, err = h.userService.SyncUser(ctx, userID)
	if err != nil {
		// Log the error but allow login if Keycloak auth succeeded
		fmt.Printf("Warning: Failed to sync user %s with local DB: %v\n", userID, err)
		// return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Local DB synchronization failed: %v", err)})
	}

	// 5. Return Keycloak token
	return c.JSON(http.StatusOK, token)
}


// Logout godoc
// @Summary Logout user
// @Description Invalidate the user's refresh token and log out.
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /api/auth/logout [post]
// @Id mciamLogout
func (h *AuthHandler) Logout(c echo.Context) error {
	// In Keycloak, tokens can be deleted on the client side
	return c.JSON(http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Refresh JWT access token using a valid refresh token.
// @Tags auth
// @Accept json
// @Produce json
// @Param refresh_token body string true "Refresh token"
// @Success 200 {object} map[string]interface{} "New token information"
// @Failure 400 {object} map[string]string "error: Bad Request"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Router /api/auth/refresh [post]
// @Id mciamRefreshToken
func (h *AuthHandler) RefreshToken(c echo.Context) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if req.RefreshToken == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Refresh token is required"})
	}

	ctx := c.Request().Context() // Get context from request

	// Use KeycloakService to refresh token
	ks := service.NewKeycloakService()
	newToken, err := ks.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": fmt.Sprintf("Token refresh failed: %v", err)})
	}

	return c.JSON(http.StatusOK, newToken)
}

// WorkspaceTicket godoc
// @Summary Set workspace ticket
// @Description Set workspace ticket
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "message: Workspace ticket set successfully"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Security BearerAuth
// @Router /api/workspaces/workspace-ticket [post]
// @Id mciamWorkspaceTicket
func (h *AuthHandler) WorkspaceTicket(c echo.Context) error {
	// 1. Check existing access token
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authorization header is missing or invalid"})
	}
	accessToken := strings.TrimPrefix(authHeader, "Bearer ")

	// 2. Request binding
	var req model.WorkspaceTicketRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
	}

	// 3. Validate workspace ID
	if _, err := strconv.ParseUint(req.WorkspaceID, 10, 32); err != nil {
		log.Printf("Workspace ID format error: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid workspace ID"})
	}

	// 4. Check user's workspace role
	userID, err := h.userService.GetUserIDByKcID(c.Request().Context(), c.Get("kcUserId").(string))
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User not found"})
	}

	// 5. Get workspace permissions
	workspaceRoles, err := h.roleService.GetUserWorkspaceRoles(userID, 0)
	if err != nil {
		log.Printf("Failed to get workspace permissions: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get workspace permissions"})
	}

	// Convert permissions to scope format
	scopes := make([]string, 0)
	for _, role := range workspaceRoles {
		scopes = append(scopes, fmt.Sprintf("workspace:%s", role))
	}

	// 6. Use KeycloakService to issue RPT
	ks := service.NewKeycloakService()

	// Create claim token in authorization.permissions format
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
		GrantType: gocloak.StringP("urn:ietf:params:oauth:grant-type:uma-ticket"),
		Audience:  gocloak.StringP(config.KC.OIDCClientName),
		//Audience:   gocloak.StringP(config.KC.OIDCClientID),
		ClaimToken: gocloak.StringP(claimToken),
	}
	rpt, err := ks.GetRequestingPartyToken(c.Request().Context(), accessToken, rptOptions)
	if err != nil {
		//log.Printf("Audience: %v", config.KC.OIDCClientID)
		log.Printf("Audience: %v", config.KC.OIDCClientName)
		log.Printf("ClaimToken: %v", claimToken)
		log.Printf("Failed to issue RPT: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get RPT"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"rpt": rpt})
}

// AuthCerts godoc
// @Summary Get authentication certificates
// @Description Retrieve authentication certificates for MC-IAM-Manager to be used in target frameworks for token validation.
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /api/auth/certs [get]
// @Id mciamAuthCerts
func (h *AuthHandler) AuthCerts(c echo.Context) error {
	cert, err := h.keycloakService.GetCerts(c.Request().Context())
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, cert)
}

// GetTempCredentialCsp godoc
// @Summary Get temporary credential CSP information
// @Description Get temporary credential provider information for AWS and GCP
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "CSP temporary credential information"
// @Router /api/auth/temp-credential-csps [get]
// @Id mciamGetTempCredentialProviders
func (h *AuthHandler) GetTempCredentialProviders(c echo.Context) error {
	// TODO : Need to manage and provide as reference data: need to provide both platform-provided list and user-selected list

	// CSP temporary credential provision method information
	cspInfo := []map[string]interface{}{
		{
			"provider": "aws",
			"methods":  []string{"oidc", "saml"},
		},
		{
			"provider": "gcp",
			"methods":  []string{"oidc"},
		},
		{
			"provider": "alibaba",
			"methods":  []string{"oidc"},
		},
	}

	return c.JSON(http.StatusOK, cspInfo)
}

// Validate godoc
// @Summary Validate access token
// @Description Validate the current access token and refresh if expired
// @Tags auth
// @Accept json
// @Produce json
// @Param refresh_token body string true "Refresh token"
// @Success 200 {object} map[string]interface{} "Token validation result with new token if refreshed"
// @Failure 400 {object} map[string]string "error: Bad Request"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Security BearerAuth
// @Router /api/auth/validate [post]
// @Id mciamValidateToken
func (h *AuthHandler) Validate(c echo.Context) error {
	// 1. Get access token from Authorization header
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authorization header is missing or invalid"})
	}
	accessToken := strings.TrimPrefix(authHeader, "Bearer ")

	// 2. Get refresh token from request body (same as RefreshToken handler)
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if req.RefreshToken == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Refresh token is required"})
	}

	ctx := c.Request().Context()
	ks := service.NewKeycloakService()

	// 3. Validate access token
	jwtToken := &gocloak.JWT{AccessToken: accessToken}
	userID, err := ks.GetUserIDFromToken(ctx, jwtToken)
	if err != nil {
		// Token is invalid or expired, try to refresh
		log.Printf("Access token validation failed: %v", err)

		// 4. Try to refresh the token
		newToken, refreshErr := ks.RefreshToken(ctx, req.RefreshToken)
		if refreshErr != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error":   "Token validation failed and refresh failed",
				"details": fmt.Sprintf("validation: %v, refresh: %v", err, refreshErr),
			})
		}

		// 5. Return new token
		return c.JSON(http.StatusOK, map[string]interface{}{
			"valid":   false,
			"message": "Token was expired and has been refreshed",
			"token":   newToken,
		})
	}

	// 6. Token is valid, return success
	return c.JSON(http.StatusOK, map[string]interface{}{
		"valid":   true,
		"message": "Token is valid",
		"user_id": userID,
	})
}
