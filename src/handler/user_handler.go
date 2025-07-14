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
		return c.JSON(http.StatusNotFound, map[string]string{"error": "사용자를 찾을 수 없습니다"})
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
		return c.JSON(http.StatusNotFound, map[string]string{"error": "사용자를 찾을 수 없습니다"})
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
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 workspace ID 형식입니다"})
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

// 특정 유저에게 할당된 workspace 목록 조회
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

// 사용자에게 할당된 workspace 와 역할 목록 조회
func (h *UserHandler) GetUserWorkspaceAndWorkspaceRolesByUserID(c echo.Context) error {
	userId := c.Param("userId")
	userIdInt, err := util.StringToUint(userId)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 user ID 형식입니다"})
	}

	// 사용자의 workspace 목록 조회
	workspaces, err := h.workspaceService.ListWorkspaces(&model.WorkspaceFilterRequest{
		UserID: userId,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to get user workspaces: %v", err)})
	}

	// workspaces가 nil이면 빈 배열로 초기화
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

func (h *UserHandler) GetUserWorkspaceAndWorkspaceRolesByUserIDAndWorkspaceID(c echo.Context) error {
	userId := c.Param("userId")
	workspaceId := c.Param("workspaceId")
	userIdInt, err := util.StringToUint(userId)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 user ID 형식입니다"})
	}
	workspaceIdInt, err := util.StringToUint(workspaceId)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 workspace ID 형식입니다"})
	}

	workspaceRoles, err := h.roleService.GetUserWorkspaceRoles(userIdInt, workspaceIdInt)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to get user workspace roles: %v", err)})
	}

	return c.JSON(http.StatusOK, workspaceRoles)
}
