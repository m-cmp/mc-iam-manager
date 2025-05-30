package handler

import (
	"fmt"
	"net/http"

	// Ensure jwt is imported
	"github.com/labstack/echo/v4"

	// "github.com/m-cmp/mc-iam-manager/config" // Removed unused import
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
	"github.com/m-cmp/mc-iam-manager/util"
	"gorm.io/gorm" // Ensure gorm is imported
)

// Helper function to check roles directly from context (assuming middleware stores them)
func checkRoleFromContext(c echo.Context, requiredRoles []string) bool {
	platformRolesIntf := c.Get("platformRoles")

	allUserRoles := []string{}
	if platformRoles, ok := platformRolesIntf.([]string); ok {
		allUserRoles = append(allUserRoles, platformRoles...)
	}

	c.Logger().Debugf("checkRoleFromContext: User Roles from Context: %v, Required: %v", allUserRoles, requiredRoles)

	for _, userRole := range allUserRoles {
		for _, reqRole := range requiredRoles {
			if userRole == reqRole {
				c.Logger().Debugf("checkRoleFromContext: Found matching role: %s", userRole)
				return true
			}
		}
	}
	c.Logger().Debugf("checkRoleFromContext: No matching role found.")
	return false
}

// 사용자 관리 기능들을 정의함.

// --- User Handler ---

type UserHandler struct {
	userService *service.UserService
	// db *gorm.DB // Not needed directly
	// keycloakConfig *config.KeycloakConfig // Not needed directly
	// keycloakClient *gocloak.GoCloak // Not needed directly
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	userService := service.NewUserService(db)
	return &UserHandler{
		userService: userService,
	}
}

// ListUsers godoc
// @Summary Get all users
// @Description Get a list of all users
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {array} model.User
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/users/list [post]
func (h *UserHandler) ListUsers(c echo.Context) error {
	// --- 역할 검증 (Admin or platformAdmin) ---
	requiredRoles := []string{"admin", "platformAdmin"} // todo : middleware에서 체크되지 않나?
	// Use the helper function that reads roles from context
	if !checkRoleFromContext(c, requiredRoles) {
		fmt.Printf("[INFO] ListUsers: Permission denied. User does not have required roles: %v\n", requiredRoles)
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden: Required role not found"})
	}
	fmt.Printf("[DEBUG] ListUsers: Permission granted.\n")
	// --- 역할 검증 끝 ---

	users, err := h.userService.ListUsers(c.Request().Context())
	if err != nil {
		fmt.Printf("[ERROR] ListUsers: Error from userService.ListUsers: %v\n", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "사용자 목록을 가져오는데 실패했습니다"})
	}

	return c.JSON(http.StatusOK, users)
}

// GetUserByID godoc
// @Summary Get user by ID
// @Description Get user details by ID
// @Tags users
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} model.User
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/users/id/{userId} [get]
func (h *UserHandler) GetUserByID(c echo.Context) error {
	// Note: Add role check if needed for this endpoint as well
	kcId := c.Param("userId")                                             // Parameter is Keycloak ID (string)
	user, err := h.userService.GetUserByKcID(c.Request().Context(), kcId) // Call renamed service method
	if err != nil {
		// Consider checking for specific errors (e.g., not found)
		fmt.Printf("[ERROR] GetUserByID: Error fetching user by KcID %s: %v\n", kcId, err)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "사용자를 찾을 수 없습니다"})
	}
	return c.JSON(http.StatusOK, user)
}

// GetUserByUsername godoc
// @Summary Get user by username
// @Description Get user details by username
// @Tags users
// @Accept json
// @Produce json
// @Param username path string true "Username"
// @Success 200 {object} model.User
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/users/name/{username} [get]
func (h *UserHandler) GetUserByUsername(c echo.Context) error {
	// Note: Add role check if needed for this endpoint as well
	username := c.Param("username")
	user, err := h.userService.GetUserByUsername(c.Request().Context(), username)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "사용자를 찾을 수 없습니다"})
	}
	return c.JSON(http.StatusOK, user)
}

// CreateUser godoc
// @Summary Create user
// @Description Create a new user
// @Tags users
// @Accept json
// @Produce json
// @Param user body model.User true "User Info"
// @Success 201 {object} model.User
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/users/createUser [post]
func (h *UserHandler) CreateUser(c echo.Context) error {
	// --- 역할 검증 (Admin or platformAdmin) ---
	requiredRoles := []string{"admin", "platformAdmin"}
	if !checkRoleFromContext(c, requiredRoles) {
		fmt.Printf("[INFO] CreateUser: Permission denied. User does not have required roles: %v\n", requiredRoles)
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden: Administrator access required"})
	}
	fmt.Printf("[DEBUG] CreateUser: Permission granted.\n")
	// --- 역할 검증 끝 ---

	var user model.User
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	// Password is handled by Keycloak, not directly in this model/handler typically.
	// Assume userService.CreateUser only returns an error
	err := h.userService.CreateUser(c.Request().Context(), &user) // Assign error to a new variable 'err'
	if err != nil {
		fmt.Printf("[ERROR] CreateUser: Error from userService.CreateUser: %v\n", err)
		// Provide more specific error if possible (e.g., user exists)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "사용자 생성에 실패했습니다"})
	}

	// Return the user data from the request body as confirmation,
	// as the service might not return the created user object directly.
	// Ensure sensitive data like password isn't returned if it were present.
	return c.JSON(http.StatusCreated, user)
}

// UpdateUser godoc
// @Summary Update user
// @Description Update user information
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param user body model.User true "User Info"
// @Success 200 {object} model.User
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/users/id/{id} [put]
func (h *UserHandler) UpdateUser(c echo.Context) error {
	// --- 역할 검증 (Admin or platformAdmin) ---
	requiredRoles := []string{"admin", "platformAdmin"}
	if !checkRoleFromContext(c, requiredRoles) {
		fmt.Printf("[INFO] UpdateUser: Permission denied. User does not have required roles: %v\n", requiredRoles)
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden: Administrator access required"})
	}
	fmt.Printf("[DEBUG] UpdateUser: Permission granted.\n")
	// --- 역할 검증 끝 ---

	// Parse DB ID (uint) from path parameter
	userIDInt, err := util.StringToUint(c.Param("userId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 user ID 형식입니다"})
	}

	var user model.User
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 본문입니다"})
	}

	user.ID = userIDInt // Set the DB ID from the path parameter

	// Call service method (assuming it now expects user object with DB ID)
	err = h.userService.UpdateUser(c.Request().Context(), &user)
	if err != nil {
		fmt.Printf("[ERROR] UpdateUser: Error from userService.UpdateUser for DB ID %d: %v\n", userIDInt, err)
		// Handle potential "not found" errors from service/repo if needed
		// Consider returning 404 if user with dbId not found
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "사용자 수정에 실패했습니다"})
	}

	// Update successful, fetch the updated user to return it (using KcId, need to get it first)
	// Or simply return 200 OK with the updated data from request?
	// Fetching by DB ID might be better if GetUserByDbID exists.
	// For now, let's return the input data as confirmation.
	// TODO: Revisit fetching logic if GetUserByID needs KcId.
	// updatedUserResponse, fetchErr := h.userService.GetUserByID(c.Request().Context(), id) // This needs KcId
	// For simplicity, return the input user data (without sensitive info if any)
	return c.JSON(http.StatusOK, user)
}

// DeleteUser godoc
// @Summary Delete user
// @Description Delete a user
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 204 "No Content"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/users/id/{id} [delete]
func (h *UserHandler) DeleteUser(c echo.Context) error {
	// --- 역할 검증 (Admin or platformAdmin) ---
	requiredRoles := []string{"admin", "platformAdmin"}
	if !checkRoleFromContext(c, requiredRoles) {
		fmt.Printf("[INFO] DeleteUser: Permission denied. User does not have required roles: %v\n", requiredRoles)
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden: Administrator access required"})
	}
	fmt.Printf("[DEBUG] DeleteUser: Permission granted.\n")
	// --- 역할 검증 끝 ---

	// Parse DB ID (uint) from path parameter
	userIDInt, err := util.StringToUint(c.Param("userId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 user ID 형식입니다"})
	}

	// Call service method with DB ID
	err = h.userService.DeleteUser(c.Request().Context(), userIDInt) // Pass uint ID
	if err != nil {
		fmt.Printf("[ERROR] DeleteUser: Error from userService.DeleteUser for DB ID %d: %v\n", userIDInt, err)
		// Consider returning 404 if user with dbId not found
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "사용자 삭제에 실패했습니다"})
	}

	return c.NoContent(http.StatusNoContent)
}

// UpdateUserStatus godoc
// @Summary Update user status
// @Description Update user status
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/users/id/{id}/status [post]
func (h *UserHandler) UpdateUserStatus(c echo.Context) error {

	// --- 역할 검증 (Admin or platformAdmin) ---
	requiredRoles := []string{"admin", "platformAdmin"}
	if !checkRoleFromContext(c, requiredRoles) {
		fmt.Printf("[INFO] ApproveUser: Permission denied. User does not have required roles: %v\n", requiredRoles)
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden: Administrator access required"})
	}
	fmt.Printf("[DEBUG] ApproveUser: Permission granted.\n")
	// --- 역할 검증 끝 ---
	userIDInt, err := util.StringToUint(c.Param("userId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 user ID 형식입니다"})
	}
	// kcUserID := c.Param("id")
	// if kcUserID == "" {
	// 	return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID가 필요합니다"})
	// }

	var updateUser model.UserStatusRequest
	if err := c.Bind(&updateUser); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 본문입니다"})
	}

	user, err := h.userService.GetUserByID(c.Request().Context(), userIDInt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "사용자를 찾을 수 없습니다"})
	}

	if updateUser.Status == "approved" {

		err := h.userService.ApproveUser(c.Request().Context(), user.KcId) // Assign error to a new variable 'err'
		if err != nil {
			fmt.Printf("[ERROR] ApproveUser: Error from userService.ApproveUser: %v\n", err)
			// Handle specific errors from service if needed (e.g., user not found in Keycloak)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("사용자 승인 실패: %v", err)})
		}
	}

	// TODO : 사용자 활성화 및 비활성화 기능 추가
	// if updateUser.Status == "active" {

	// 	err := h.userService.ApproveUser(c.Request().Context(), user.KcId) // Assign error to a new variable 'err'
	// 	if err != nil {
	// 		fmt.Printf("[ERROR] ApproveUser: Error from userService.ApproveUser: %v\n", err)
	// 		// Handle specific errors from service if needed (e.g., user not found in Keycloak)
	// 		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("사용자 승인 실패: %v", err)})
	// 	}
	// }

	// if updateUser.Status == "inactive" {

	// 	err := h.userService.ApproveUser(c.Request().Context(), user.KcId) // Assign error to a new variable 'err'
	// 	if err != nil {
	// 		fmt.Printf("[ERROR] ApproveUser: Error from userService.ApproveUser: %v\n", err)
	// 		// Handle specific errors from service if needed (e.g., user not found in Keycloak)
	// 		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("사용자 승인 실패: %v", err)})
	// 	}
	// }

	return c.NoContent(http.StatusNoContent)
}

// GetUserWorkspaceAndWorkspaceRoles godoc
// @Summary Get user workspace and roles
// @Description Get workspaces and roles for the current user
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {array} service.WorkspaceRoleInfo
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/users/workspaces [get]
func (h *UserHandler) GetUserWorkspaceAndWorkspaceRoles(c echo.Context) error {
	// // 1. Get user claims from context
	// claimsIntf := c.Get("token_claims")
	// if claimsIntf == nil {
	// 	return c.JSON(http.StatusUnauthorized, map[string]string{"error": "token_claims not found in context"})
	// }
	// mapClaimsPtr, ok := claimsIntf.(*jwt.MapClaims) // Assert to pointer type
	// if !ok || mapClaimsPtr == nil {
	// 	fmt.Printf("[ERROR] GetUserWorkspaceAndWorkspaceRoles: Failed to assert token_claims to *jwt.MapClaims. Actual type: %T\n", claimsIntf) // Updated log prefix
	// 	return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to process user claims"})
	// }
	// mapClaims := *mapClaimsPtr // Dereference

	// // 2. Get Keycloak User ID (subject) from claims
	// kcUserID, err := mapClaims.GetSubject() // Use GetSubject() method
	// if err != nil {
	// 	fmt.Printf("[ERROR] GetUserWorkspaceAndWorkspaceRoles: Failed to get subject (kcUserID) from claims: %v\n", err) // Updated log prefix
	// 	return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get user ID from token"})
	// }
	// if kcUserID == "" {
	// 	return c.JSON(http.StatusInternalServerError, map[string]string{"error": "User ID (sub) is empty in token"})
	// }
	kcUserIdVal := c.Get("kcUserId")
	if kcUserIdVal == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get user at GetUserWorkspaceAndWorkspaceRoles"})
	}
	kcUserID, ok := kcUserIdVal.(string)
	if !ok || kcUserID == "" {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "invalid kcUserId in context at GetUserWorkspaceAndWorkspaceRoles"})
	}

	// 3. Get local DB User ID (db_id) using the service method
	//    This method handles syncing if the user isn't in the local DB yet.
	localUserID, err := h.userService.GetUserIDByKcID(c.Request().Context(), kcUserID) // Correct function name
	if err != nil {
		fmt.Printf("[ERROR] GetUserWorkspaceAndWorkspaceRoles: Failed to get local DB ID for user (kcID: %s): %v\n", kcUserID, err) // Updated log prefix
		// Handle potential errors like user not found in Keycloak (if GetUserIDByKcID propagates it)
		// or DB errors during sync.
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve user database ID"})
	}
	if localUserID == 0 {
		// Should not happen if GetUserIDByKcID works correctly, but safeguard.
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Retrieved invalid local user database ID (0)"})
	}

	workspaceUserRole := model.WorkspaceWithUsersAndRolesRequest{}
	if err := c.Bind(&workspaceUserRole); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}
	// 4. Call the service function with the local DB User ID
	workspaceRoles, err := h.roleService.GetUserWorkspaceRoles(localUserID, workspaceUserRole.WorkspaceID) // Correct service method name
	if err != nil {
		fmt.Printf("[ERROR] GetUserWorkspaceAndWorkspaceRoles: Error from service: %v\n", err) // Updated log prefix
		// Handle specific errors like UserNotFound if necessary, though GetUserIDByKcID should prevent this
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "워크스페이스 및 역할 정보를 가져오는데 실패했습니다"})
	}

	// Return the result
	if workspaceRoles == nil {
		workspaceRoles = []service.WorkspaceRoleInfo{} // Return empty array instead of null
	}
	return c.JSON(http.StatusOK, workspaceRoles)
}
