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

// Define user management functions.

// --- User Handler ---

type UserHandler struct {
	userService      *service.UserService
	roleService      *service.RoleService
	workspaceService *service.WorkspaceService
	// db *gorm.DB // Not needed directly
	// keycloakConfig *config.KeycloakConfig // Not needed directly
	// keycloakClient *gocloak.GoCloak // Not needed directly
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	userService := service.NewUserService(db)
	roleService := service.NewRoleService(db)
	workspaceService := service.NewWorkspaceService(db)
	return &UserHandler{
		userService:      userService,
		roleService:      roleService,
		workspaceService: workspaceService,
	}
}

// ListUsers godoc
// @Summary List all users
// @Description Retrieve a list of all users.
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {array} model.User
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/users/list [post]
// @Id listUsers
func (h *UserHandler) ListUsers(c echo.Context) error {
	// --- Role validation (Admin or platformAdmin) ---
	requiredRoles := []string{"admin", "platformAdmin"} // todo : shouldn't this be checked in middleware?
	// Use the helper function that reads roles from context
	if !checkRoleFromContext(c, requiredRoles) {
		fmt.Printf("[INFO] ListUsers: Permission denied. User does not have required roles: %v\n", requiredRoles)
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden: Required role not found"})
	}
	fmt.Printf("[DEBUG] ListUsers: Permission granted.\n")
	// --- Role validation end ---

	users, err := h.userService.ListUsers(c.Request().Context())
	if err != nil {
		fmt.Printf("[ERROR] ListUsers: Error from userService.ListUsers: %v\n", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve user list"})
	}

	return c.JSON(http.StatusOK, users)
}

// GetUserByKcID godoc
// @Summary Get user by KcID
// @Description Get user details by KcID
// @Tags users
// @Accept json
// @Produce json
// @Param kcUserId path string true "User KcID"
// @Success 200 {object} model.User
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/users/kc/{kcUserId} [get]
// @Id getUserByKcID
func (h *UserHandler) GetUserByKcID(c echo.Context) error {
	// Note: Add role check if needed for this endpoint as well
	kcId := c.Param("kcUserId")                                           // Parameter is Keycloak ID (string)
	user, err := h.userService.GetUserByKcID(c.Request().Context(), kcId) // Call renamed service method
	if err != nil {
		// Consider checking for specific errors (e.g., not found)
		fmt.Printf("[ERROR] GetUserByKcID: Error fetching user by KcID %s: %v\n", kcId, err)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}
	return c.JSON(http.StatusOK, user)
}

// GetUserByID godoc
// @Summary Get user by ID
// @Description Retrieve user details by user ID.
// @Tags users
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} model.User
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/users/id/{userId} [get]
// @Id getUserByID
func (h *UserHandler) GetUserByID(c echo.Context) error {
	// Note: Add role check if needed for this endpoint as well
	userId := c.Param("userId") // Parameter is Keycloak ID (string)
	userIdInt, err := util.StringToUint(userId)

	user, err := h.userService.GetUserByID(c.Request().Context(), userIdInt)
	if err != nil {
		// Consider checking for specific errors (e.g., not found)
		fmt.Printf("[ERROR] GetUserByID: Error fetching user by Id %s: %v\n", userId, err)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}
	return c.JSON(http.StatusOK, user)
}

// GetUserByUsername godoc
// @Summary Get user by username
// @Description Get user details by username
// @Tags users
// @Accept json
// @Produce json
// @Param name path string true "Username"
// @Success 200 {object} model.User
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/users/name/{username} [get]
// @Id getUserByUsername
func (h *UserHandler) GetUserByUsername(c echo.Context) error {
	// Note: Add role check if needed for this endpoint as well
	username := c.Param("username")
	user, err := h.userService.GetUserByUsername(c.Request().Context(), username)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}
	return c.JSON(http.StatusOK, user)
}

// CreateUser godoc
// @Summary Create new user
// @Description Create a new user with the specified information.
// @Tags users
// @Accept json
// @Produce json
// @Param user body model.User true "User Info"
// @Success 201 {object} model.User
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/users [post]
// @Id createUser
func (h *UserHandler) CreateUser(c echo.Context) error {
	// --- Role validation (Admin or platformAdmin) ---
	requiredRoles := []string{"admin", "platformAdmin"}
	if !checkRoleFromContext(c, requiredRoles) {
		fmt.Printf("[INFO] CreateUser: Permission denied. User does not have required roles: %v\n", requiredRoles)
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden: Administrator access required"})
	}
	fmt.Printf("[DEBUG] CreateUser: Permission granted.\n")
	// --- Role validation end ---

	var user model.User
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	// Password is handled by Keycloak, not directly in this model/handler typically.
	// Assume userService.CreateUser only returns an error
	err := h.userService.CreateUser(c.Request().Context(), &user) // Assign error to a new variable 'err'
	if err != nil {
		fmt.Printf("[ERROR] CreateUser: Error from userService.CreateUser: %v\n", err)
		// Provide more specific error if possible (e.g., user exists)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create user"})
	}

	// Return the user data from the request body as confirmation,
	// as the service might not return the created user object directly.
	// Ensure sensitive data like password isn't returned if it were present.
	return c.JSON(http.StatusCreated, user)
}

// UpdateUser godoc
// @Summary Update user
// @Description Update the details of an existing user.
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
// @Router /api/users/{id} [put]
// @Id updateUser
func (h *UserHandler) UpdateUser(c echo.Context) error {
	// --- Role validation (Admin or platformAdmin) ---
	requiredRoles := []string{"admin", "platformAdmin"}
	if !checkRoleFromContext(c, requiredRoles) {
		fmt.Printf("[INFO] UpdateUser: Permission denied. User does not have required roles: %v\n", requiredRoles)
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden: Administrator access required"})
	}
	fmt.Printf("[DEBUG] UpdateUser: Permission granted.\n")
	// --- Role validation end ---

	// Parse DB ID (uint) from path parameter
	userIDInt, err := util.StringToUint(c.Param("userId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID format"})
	}

	var user model.User
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	user.ID = userIDInt // Set the DB ID from the path parameter

	// Call service method (assuming it now expects user object with DB ID)
	err = h.userService.UpdateUser(c.Request().Context(), &user)
	if err != nil {
		fmt.Printf("[ERROR] UpdateUser: Error from userService.UpdateUser for DB ID %d: %v\n", userIDInt, err)
		// Handle potential "not found" errors from service/repo if needed
		// Consider returning 404 if user with dbId not found
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update user"})
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
// @Description Delete a user by their ID.
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 204 "No Content"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/users/{id} [delete]
// @Id deleteUser
func (h *UserHandler) DeleteUser(c echo.Context) error {
	// --- Role validation (Admin or platformAdmin) ---
	requiredRoles := []string{"admin", "platformAdmin"}
	if !checkRoleFromContext(c, requiredRoles) {
		fmt.Printf("[INFO] DeleteUser: Permission denied. User does not have required roles: %v\n", requiredRoles)
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden: Administrator access required"})
	}
	fmt.Printf("[DEBUG] DeleteUser: Permission granted.\n")
	// --- Role validation end ---

	// Parse DB ID (uint) from path parameter
	userIDInt, err := util.StringToUint(c.Param("userId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID format"})
	}

	// Call service method with DB ID
	err = h.userService.DeleteUser(c.Request().Context(), userIDInt) // Pass uint ID
	if err != nil {
		fmt.Printf("[ERROR] DeleteUser: Error from userService.DeleteUser for DB ID %d: %v\n", userIDInt, err)
		// Consider returning 404 if user with dbId not found
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete user"})
	}

	return c.NoContent(http.StatusNoContent)
}

// UpdateUserStatus godoc
// @Summary Update user status
// @Description Update user status (active/inactive)
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param status body model.UserStatusRequest true "User Status"
// @Success 200 {object} model.User
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/users/id/{userId}/status [post]
// @Id updateUserStatus
func (h *UserHandler) UpdateUserStatus(c echo.Context) error {

	// --- Role validation (Admin or platformAdmin) ---
	requiredRoles := []string{"admin", "platformAdmin"}
	if !checkRoleFromContext(c, requiredRoles) {
		fmt.Printf("[INFO] ApproveUser: Permission denied. User does not have required roles: %v\n", requiredRoles)
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden: Administrator access required"})
	}
	fmt.Printf("[DEBUG] ApproveUser: Permission granted.\n")
	// --- Role validation end ---
	userIDInt, err := util.StringToUint(c.Param("userId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID format"})
	}
	// kcUserID := c.Param("id")
	// if kcUserID == "" {
	// 	return c.JSON(http.StatusBadRequest, map[string]string{"error": "User ID is required"})
	// }

	var updateUser model.UserStatusRequest
	if err := c.Bind(&updateUser); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	user, err := h.userService.GetUserByID(c.Request().Context(), userIDInt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "User not found"})
	}

	if updateUser.Status == "approved" {

		err := h.userService.ApproveUser(c.Request().Context(), user.KcId) // Assign error to a new variable 'err'
		if err != nil {
			fmt.Printf("[ERROR] ApproveUser: Error from userService.ApproveUser: %v\n", err)
			// Handle specific errors from service if needed (e.g., user not found in Keycloak)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to approve user: %v", err)})
		}
	}

	// TODO : Add user activation and deactivation functionality
	// if updateUser.Status == "active" {

	// 	err := h.userService.ApproveUser(c.Request().Context(), user.KcId) // Assign error to a new variable 'err'
	// 	if err != nil {
	// 		fmt.Printf("[ERROR] ApproveUser: Error from userService.ApproveUser: %v\n", err)
	// 		// Handle specific errors from service if needed (e.g., user not found in Keycloak)
	// 		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to approve user: %v", err)})
	// 	}
	// }

	// if updateUser.Status == "inactive" {

	// 	err := h.userService.ApproveUser(c.Request().Context(), user.KcId) // Assign error to a new variable 'err'
	// 	if err != nil {
	// 		fmt.Printf("[ERROR] ApproveUser: Error from userService.ApproveUser: %v\n", err)
	// 		// Handle specific errors from service if needed (e.g., user not found in Keycloak)
	// 		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to approve user: %v", err)})
	// 	}
	// }

	return c.NoContent(http.StatusNoContent)
}

// ListUserWorkspaceAndWorkspaceRoles godoc
// @Summary List user workspace and roles
// @Description List workspaces and roles for the current user
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {array} model.RoleMaster
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/users/workspaces/roles/list [post]
// @Id listUserWorkspaceAndWorkspaceRoles
func (h *UserHandler) ListUserWorkspaceAndWorkspaceRoles(c echo.Context) error {
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

	req := model.WorkspaceWithUsersAndRolesRequest{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	var workspaceID uint
	if req.WorkspaceID != "" {
		workspaceID, err = util.StringToUint(req.WorkspaceID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 workspace ID 형식입니다"})
		}
	}

	// Get user's workspace roles
	var workspaceRoles []model.UserWorkspaceRole
	workspaceRoles, err = h.roleService.GetUserWorkspaceRoles(localUserID, workspaceID)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to get user workspace roles: %v", err)})
	}

	return c.JSON(http.StatusOK, workspaceRoles)
}

// ListUserWorkspaces godoc
// @Summary List user workspaces
// @Description List workspaces for the current user
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {array} model.Workspace
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/users/workspaces/list [post]
// @Id listUserWorkspaces
func (h *UserHandler) ListUserWorkspaces(c echo.Context) error {
	// 1. Get platform roles from context
	userPlatformRoles, ok := c.Get("platformRoles").([]string)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "Unable to retrieve platform roles")
	}

	// 2. Check if user has platformAdmin role
	isPlatformAdmin := false
	for _, role := range userPlatformRoles {
		if role == "platformAdmin" {
			isPlatformAdmin = true
			break
		}
	}

	// 3. If platformAdmin, return all workspaces
	if isPlatformAdmin {
		// Get all workspaces without user filter
		allWorkspacesFilter := &model.WorkspaceFilterRequest{}
		workspaces, err := h.workspaceService.ListWorkspaces(allWorkspacesFilter)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to get all workspaces: %v", err)})
		}

		// Get projects in each workspace
		workspacesProjects := make([]*model.WorkspaceWithProjects, 0)
		for _, workspace := range workspaces {
			aWorkspacesProject, err := h.workspaceService.ListWorkspacesProjects(&model.WorkspaceFilterRequest{
				WorkspaceID: fmt.Sprintf("%d", workspace.ID),
			})
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to get workspace projects: %v", err)})
			}
			workspacesProjects = append(workspacesProjects, aWorkspacesProject...)
		}
		return c.JSON(http.StatusOK, workspacesProjects)
	}

	// 4. If not platformAdmin, use existing logic for user-specific workspaces
	// Get Keycloak User ID
	kcUserIdVal := c.Get("kcUserId")
	if kcUserIdVal == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get user at GetUserWorkspaceAndWorkspaceRoles"})
	}
	kcUserID, ok := kcUserIdVal.(string)
	if !ok || kcUserID == "" {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "invalid kcUserId in context at GetUserWorkspaceAndWorkspaceRoles"})
	}

	// Get local DB User ID (db_id) using the service method
	localUserID, err := h.userService.GetUserIDByKcID(c.Request().Context(), kcUserID)
	if err != nil {
		fmt.Printf("[ERROR] GetUserWorkspaceAndWorkspaceRoles: Failed to get local DB ID for user (kcID: %s): %v\n", kcUserID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve user database ID"})
	}
	if localUserID == 0 {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Retrieved invalid local user database ID (0)"})
	}

	WorkspaceFilterRequest := &model.WorkspaceFilterRequest{
		UserID: fmt.Sprintf("%d", localUserID),
	}

	// Get user's workspace
	var workspaces []*model.Workspace
	workspaces, err = h.workspaceService.ListWorkspaces(WorkspaceFilterRequest)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to get user workspace roles: %v", err)})
	}

	// Get projects in each workspace
	workspacesProjects := make([]*model.WorkspaceWithProjects, 0)
	for _, workspace := range workspaces {
		aWorkspacesProject, err := h.workspaceService.ListWorkspacesProjects(&model.WorkspaceFilterRequest{
			WorkspaceID: fmt.Sprintf("%d", workspace.ID),
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to get user workspace roles: %v", err)})
		}
		workspacesProjects = append(workspacesProjects, aWorkspacesProject...)
	}
	return c.JSON(http.StatusOK, workspacesProjects)
}

// ListUserProjectsByWorkspace godoc
// @Summary List user projects by workspace
// @Description List projects for the current user
// @Tags users
// @Accept json
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {array} model.Project
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/users/workspaces/id/{workspaceId}/projects/list [get]
// @Id listUserProjectsByWorkspace
func (h *UserHandler) ListUserProjectsByWorkspace(c echo.Context) error {
	workspaceIdInt, err := util.StringToUint(c.Param("workspaceId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid workspace ID format"})
	}

	// 1. Get Keycloak User ID
	kcUserIdVal := c.Get("kcUserId")
	if kcUserIdVal == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get user at GetUserWorkspaceAndWorkspaceRoles"})
	}
	kcUserID, ok := kcUserIdVal.(string)
	if !ok || kcUserID == "" {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "invalid kcUserId in context at GetUserWorkspaceAndWorkspaceRoles"})
	}

	// 2. Get local DB User ID (db_id) using the service method
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

	// Get user's workspace
	//ListWorkspacesProjects
	//workspaces, err := h.workspaceService.ListWorkspaces(&model.WorkspaceFilterRequest{
	workspaces, err := h.workspaceService.ListWorkspacesProjects(&model.WorkspaceFilterRequest{
		WorkspaceID: fmt.Sprintf("%d", workspaceIdInt),
	})
	//workspace, err := h.workspaceService.GetWorkspaceByID(workspaceIdInt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to get user workspace roles: %v", err)})
	}

	if len(workspaces) > 0 {
		return c.JSON(http.StatusOK, workspaces[0])
	}
	return c.JSON(http.StatusOK, model.Workspace{})
}

// Retrieve workspace list assigned to specific user
// GetUserWorkspacesByUserID godoc
// @Summary Get user workspaces by user ID
// @Description Get workspaces for a specific user
// @Tags users
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {array} model.Workspace
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/users/id/{userId}/workspaces/list [get]
// @Id getUserWorkspacesByUserID
func (h *UserHandler) GetUserWorkspacesByUserID(c echo.Context) error {
	userId := c.Param("userId")

	WorkspaceFilterRequest := &model.WorkspaceFilterRequest{
		UserID: userId,
	}

	// Get user's workspace
	workspaces, err := h.workspaceService.ListWorkspaces(WorkspaceFilterRequest)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to get user workspace roles: %v", err)})
	}

	// // Get projects in each workspace
	// var workspacesProjects []*model.WorkspaceWithProjects
	// for _, workspace := range workspaces {
	// 	aWorkspacesProject, err := h.workspaceService.ListWorkspacesProjects(&model.WorkspaceFilterRequest{
	// 		WorkspaceID: fmt.Sprintf("%d", workspace.ID),
	// 	})
	// 	if err != nil {
	// 		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to get user workspace roles: %v", err)})
	// 	}
	// 	workspace.Projects = aWorkspacesProject
	// }
	return c.JSON(http.StatusOK, workspaces)
}

// Retrieve workspace and role list assigned to user
// @Summary Get user workspace and workspace roles by user ID
// @Description Get workspaces and roles for a specific user
// @Tags users
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {array} model.UserWorkspaceRole
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/users/id/{userId}/workspaces/roles/list [get]
// @Id getUserWorkspaceAndWorkspaceRolesByUserID
func (h *UserHandler) GetUserWorkspaceAndWorkspaceRolesByUserID(c echo.Context) error {
	userId := c.Param("userId")
	userIdInt, err := util.StringToUint(userId)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID format"})
	}

	// Retrieve user's workspace list
	workspaces, err := h.workspaceService.ListWorkspaces(&model.WorkspaceFilterRequest{
		UserID: userId,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to get user workspaces: %v", err)})
	}

	// Initialize as empty array if workspaces is nil
	if workspaces == nil {
		workspaces = make([]*model.Workspace, 0)
	}

	userWorkspaceRoles := make([]model.UserWorkspaceRole, 0)
	for _, workspace := range workspaces {
		// Get user's workspace roles
		workspaceRoles, err := h.roleService.GetUserWorkspaceRoles(userIdInt, workspace.ID)

		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to get user workspace roles: %v", err)})
		}
		userWorkspaceRoles = append(userWorkspaceRoles, workspaceRoles...)
	}

	return c.JSON(http.StatusOK, userWorkspaceRoles)
}

// @Summary Get user workspace and workspace roles by user ID and workspace ID
// @Description Get workspaces and roles for a specific user and workspace
// @Tags users
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {array} model.UserWorkspaceRole
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/users/id/{userId}/workspaces/id/{workspaceId}/roles/list [get]
// @Id getUserWorkspaceAndWorkspaceRolesByUserIDAndWorkspaceID
func (h *UserHandler) GetUserWorkspaceAndWorkspaceRolesByUserIDAndWorkspaceID(c echo.Context) error {
	userId := c.Param("userId")
	workspaceId := c.Param("workspaceId")
	userIdInt, err := util.StringToUint(userId)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID format"})
	}
	workspaceIdInt, err := util.StringToUint(workspaceId)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid workspace ID format"})
	}

	workspaceRoles, err := h.roleService.GetUserWorkspaceRoles(userIdInt, workspaceIdInt)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to get user workspace roles: %v", err)})
	}

	return c.JSON(http.StatusOK, workspaceRoles)
}
