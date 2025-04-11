package handler

import (
	"fmt" // Ensure fmt is imported
	"net/http"

	"github.com/golang-jwt/jwt/v5" // Ensure jwt is imported
	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
	"github.com/m-cmp/mc-iam-manager/util" // Import the new util package
)

// Helper function moved to util package

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetUsers godoc
// @Summary 사용자 목록 조회 (관리자용)
// @Description 모든 사용자 목록을 조회합니다. 'admin' 또는 'platformAdmin' 역할이 필요합니다.
// @Tags users
// @Produce json
// @Success 200 {array} model.User "성공 시 사용자 목록 반환"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden (권한 부족)"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /users [get]
func (h *UserHandler) GetUsers(c echo.Context) error {
	// --- 역할 검증 (Admin or platformAdmin) ---
	hasPermission := false
	claims, err := util.GetMapClaimsFromContext(c) // Call util function
	if err == nil {
		fmt.Printf("[DEBUG] GetUsers: Claims retrieved from access_token: %+v\n", claims)
		// Check roles using map access from parsed token
		rolesValue, rolesKeyExists := claims["roles"]
		if rolesKeyExists {
			fmt.Printf("[DEBUG] GetUsers: Checking top-level 'roles': %v (Type: %T)\n", rolesValue, rolesValue)
			if rolesClaim, typeOk := rolesValue.([]interface{}); typeOk {
				for _, role := range rolesClaim {
					if roleStr, strOk := role.(string); strOk && (roleStr == "admin" || roleStr == "platformAdmin") {
						fmt.Printf("[DEBUG] GetUsers: Found matching role in top-level 'roles': %s\n", roleStr)
						hasPermission = true
						break
					}
				}
			} else {
				fmt.Printf("[DEBUG] GetUsers: Top-level 'roles' is not []interface{}. Actual type: %T\n", rolesValue)
			}
		} else {
			fmt.Printf("[DEBUG] GetUsers: Key 'roles' does not exist at top level in claims map.\n")
		}

		if !hasPermission { // Fallback check in realm_access.roles
			if realmAccessValue, realmKeyExists := claims["realm_access"]; realmKeyExists {
				fmt.Printf("[DEBUG] GetUsers: Checking 'realm_access': %+v\n", realmAccessValue)
				if realmAccessMap, mapOk := realmAccessValue.(map[string]interface{}); mapOk {
					if realmRolesValue, realmRolesKeyExists := realmAccessMap["roles"]; realmRolesKeyExists {
						fmt.Printf("[DEBUG] GetUsers: Checking 'realm_access.roles': %v (Type: %T)\n", realmRolesValue, realmRolesValue)
						if realmRolesClaim, typeOk := realmRolesValue.([]interface{}); typeOk {
							for _, role := range realmRolesClaim {
								if roleStr, strOk := role.(string); strOk && (roleStr == "admin" || roleStr == "platformAdmin") {
									fmt.Printf("[DEBUG] GetUsers: Found matching role in 'realm_access.roles': %s\n", roleStr)
									hasPermission = true
									break
								}
							}
						} else {
							fmt.Printf("[DEBUG] GetUsers: 'realm_access.roles' is not []interface{}. Actual type: %T\n", realmRolesValue)
						}
					} else {
						fmt.Printf("[DEBUG] GetUsers: Key 'roles' does not exist in 'realm_access' map.\n")
					}
				} else {
					fmt.Printf("[DEBUG] GetUsers: 'realm_access' is not map[string]interface{}. Actual type: %T\n", realmAccessValue)
				}
			} else {
				fmt.Printf("[DEBUG] GetUsers: Key 'realm_access' does not exist in claims map.\n")
			}
		}
	} else {
		fmt.Printf("[ERROR] GetUsers: Failed to get claims for permission check: %v\n", err)
		// Return Forbidden as we cannot verify roles
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden: Could not verify user roles"})
	}

	if !hasPermission {
		fmt.Printf("[DEBUG] GetUsers: Permission denied.\n")
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden: Required role not found"})
	}
	// --- 역할 검증 끝 ---

	users, err := h.userService.GetUsers(c.Request().Context())
	if err != nil {
		// Log the original error from service/repository layer for better debugging
		fmt.Printf("[ERROR] GetUsers: Error from userService.GetUsers: %v\n", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "사용자 목록을 가져오는데 실패했습니다"}) // Keep generic message for client
	}

	return c.JSON(http.StatusOK, users)
}

// GetUser returns a user by ID
// @Security BearerAuth
func (h *UserHandler) GetUserByID(c echo.Context) error {
	id := c.Param("id")
	user, err := h.userService.GetUserByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "사용자를 찾을 수 없습니다"})
	}

	return c.JSON(http.StatusOK, user)
}

// GetUserByUsername returns a user by username
// @Security BearerAuth
func (h *UserHandler) GetUserByUsername(c echo.Context) error {
	username := c.Param("username")
	user, err := h.userService.GetUserByUsername(c.Request().Context(), username)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "사용자를 찾을 수 없습니다"})
	}

	return c.JSON(http.StatusOK, user)
}

// CreateUser creates a new user (Admin only)
// @Security BearerAuth
// @Router /users [post]
func (h *UserHandler) CreateUser(c echo.Context) error {
	// --- 권한 검증 (Admin or SuperAdmin) ---
	isAdminOrSuperAdmin := false
	if u := c.Get("user"); u != nil {
		if claims, ok := u.(*jwt.Token); ok {
			if mapClaims, ok := claims.Claims.(jwt.MapClaims); ok {
				if rolesClaim, ok := mapClaims["realm_access"].(map[string]interface{}); ok {
					if roles, ok := rolesClaim["roles"].([]interface{}); ok {
						for _, role := range roles {
							if roleStr, ok := role.(string); ok && (roleStr == "admin" || roleStr == "platformadmin") { // Changed platform_superadmin to platformadmin
								isAdminOrSuperAdmin = true
								break
							}
						}
					}
				}
			}
		}
	}
	if !isAdminOrSuperAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden: Administrator access required"})
	}
	// --- 권한 검증 끝 ---

	var user model.User
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	if err := h.userService.CreateUser(c.Request().Context(), &user); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "사용자 생성에 실패했습니다"})
	}

	return c.JSON(http.StatusCreated, user)
}

// UpdateUser updates a user (Admin only)
// @Security BearerAuth
// @Router /users/{id} [put]
func (h *UserHandler) UpdateUser(c echo.Context) error {
	// --- 권한 검증 (Admin or SuperAdmin) ---
	isAdminOrSuperAdmin := false
	if u := c.Get("user"); u != nil {
		if claims, ok := u.(*jwt.Token); ok {
			if mapClaims, ok := claims.Claims.(jwt.MapClaims); ok {
				if rolesClaim, ok := mapClaims["realm_access"].(map[string]interface{}); ok {
					if roles, ok := rolesClaim["roles"].([]interface{}); ok {
						for _, role := range roles {
							if roleStr, ok := role.(string); ok && (roleStr == "admin" || roleStr == "platform_superadmin") {
								isAdminOrSuperAdmin = true
								break
							}
						}
					}
				}
			}
		}
	}
	if !isAdminOrSuperAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden: Administrator access required"})
	}
	// --- 권한 검증 끝 ---

	id := c.Param("id") // Get ID from path
	var user model.User
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "잘못된 요청입니다",
		})
	}

	user.ID = id // Set the ID from the path parameter
	if err := h.userService.UpdateUser(c.Request().Context(), &user); err != nil {
		// Handle potential "not found" errors from service/repo if needed
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "사용자 수정에 실패했습니다",
		})
	}

	// Optionally fetch and return the updated user data
	updatedUser, err := h.userService.GetUserByID(c.Request().Context(), id)
	if err != nil || updatedUser == nil {
		// Log error but return success as update itself was likely successful
		fmt.Printf("Warning: Failed to fetch updated user after update (id: %s): %v\n", id, err)
		return c.JSON(http.StatusOK, user) // Return the input data as confirmation
	}
	return c.JSON(http.StatusOK, updatedUser)
}

// DeleteUser deletes a user
// @Security BearerAuth
func (h *UserHandler) DeleteUser(c echo.Context) error {
	id := c.Param("id")
	if err := h.userService.DeleteUser(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "사용자 삭제에 실패했습니다",
		})
	}

	return c.NoContent(http.StatusNoContent)
}

// ApproveUser godoc
// @Summary 사용자 승인 (관리자용)
// @Description 지정된 사용자를 활성화하고 시스템 사용을 승인합니다. 'admin' 또는 'platformadmin' 역할이 필요합니다. // Changed platform_superadmin to platformadmin
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "사용자 Keycloak ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "error: 잘못된 사용자 ID"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden (권한 부족)"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /users/{id}/approve [post]
func (h *UserHandler) ApproveUser(c echo.Context) error {
	// --- 권한 검증 (예시: 핸들러 내에서 직접 확인) ---
	// 실제로는 미들웨어를 사용하는 것이 더 좋습니다.
	isAdminOrSuperAdmin := false
	if user := c.Get("user"); user != nil {
		if claims, ok := user.(*jwt.Token); ok {
			if mapClaims, ok := claims.Claims.(jwt.MapClaims); ok {
				if rolesClaim, ok := mapClaims["realm_access"].(map[string]interface{}); ok {
					if roles, ok := rolesClaim["roles"].([]interface{}); ok {
						for _, role := range roles {
							if roleStr, ok := role.(string); ok && (roleStr == "admin" || roleStr == "platformadmin") { // Changed platform_superadmin to platformadmin
								isAdminOrSuperAdmin = true
								break
							}
						}
					}
				}
			}
		}
	}
	if !isAdminOrSuperAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden: Administrator access required"})
	}
	// --- 권한 검증 끝 ---

	kcUserID := c.Param("id")
	if kcUserID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID가 필요합니다"})
	}

	if err := h.userService.ApproveUser(c.Request().Context(), kcUserID); err != nil {
		// Handle specific errors from service if needed (e.g., user not found in Keycloak)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("사용자 승인 실패: %v", err),
		})
	}

	return c.NoContent(http.StatusNoContent)
}
