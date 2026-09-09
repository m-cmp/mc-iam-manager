package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
	"github.com/m-cmp/mc-iam-manager/util"
	"gorm.io/gorm"
)

// RoleHandler role management handler
type RoleHandler struct {
	roleService     *service.RoleService
	userService     *service.UserService
	keycloakService service.KeycloakService
	menuService     *service.MenuService
	cspRoleService  *service.CspRoleService
}

// NewRoleHandler create new RoleHandler instance
func NewRoleHandler(db *gorm.DB) *RoleHandler {
	roleService := service.NewRoleService(db)
	userService := service.NewUserService(db)
	keycloakService := service.NewKeycloakService()
	menuService := service.NewMenuService(db)
	cspRoleService := service.NewCspRoleService(db, keycloakService)
	return &RoleHandler{
		roleService:     roleService,
		userService:     userService,
		keycloakService: keycloakService,
		menuService:     menuService,
		cspRoleService:  cspRoleService,
	}
}

// @Summary List all roles
// @Description Retrieve a list of all roles.
// @Tags roles
// @Accept json
// @Produce json
// @Success 200 {array} model.RoleMaster
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/list [post]
// @Id listRoles
func (h *RoleHandler) ListRoles(c echo.Context) error {
	var req model.RoleFilterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	roles, err := h.roleService.ListRoles(&req)
	if err != nil {
		log.Printf("Failed to retrieve role list: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to retrieve role list: %v", err)})
	}

	log.Printf("Successfully retrieved role list - number of roles: %d", len(roles))
	return c.JSON(http.StatusOK, roles)
}

// @Summary Create role
// @Description Create a new role
// @Tags roles
// @Accept json
// @Produce json
// @Param role body model.CreateRoleRequest true "Role Info"
// @Success 201 {object} model.RoleMaster
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles [post]
// @Id createRole
func (h *RoleHandler) CreateRole(c echo.Context) error {
	var req model.CreateRoleRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Failed to bind role creation request: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	log.Printf("Role creation request - name: %s, description: %s, parentID: %d, roleTypes: %v",
		req.Name, req.Description, req.ParentID, req.RoleTypes)

	// Input validation
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Create RoleMaster
	role := &model.RoleMaster{
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
	}

	roleSubs := make([]model.RoleSub, 0)
	for _, roleType := range req.RoleTypes {
		roleSubs = append(roleSubs, model.RoleSub{
			RoleType: roleType,
		})
	}

	// 1. Create CSP roles first (external API calls required, so handle outside transaction)
	createdCspRoles := make([]model.CreateCspRoleRequest, 0)
	if len(req.CspRoles) > 0 {
		log.Printf("cspRoles requested: %v", req.CspRoles)
		for _, cspRole := range req.CspRoles {
			// TODO: Extract CSPRoleName logic to a function
			if cspRole.CspRoleName == "" {
				cspRoleName := constants.CspRoleNamePrefix + req.Name // Create cspRoleName by adding prefix to roleName
				cspRole.CspRoleName = cspRoleName
			}

			if !strings.HasPrefix(cspRole.CspRoleName, constants.CspRoleNamePrefix) {
				cspRole.CspRoleName = constants.CspRoleNamePrefix + cspRole.CspRoleName
			}

			log.Printf("cspRole requested: %v", cspRole)
			// Create or update CSP role
			createdCspRole, err := h.cspRoleService.CreateOrUpdateCspRole(&cspRole)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to create/update CSP role: %v", err)})
			}

			// Store created CSP role information
			// AuthMethod는 요청 원본(cspRole)에서 그대로 전달 — 누락 시 CreateRoleWithAllDependencies가
			// OIDC로 기본 반영하지만, SAML 등으로 생성된 CSP 역할은 그 값이 매핑에도 반영되어야 한다.
			createdCspRoles = append(createdCspRoles, model.CreateCspRoleRequest{
				ID:          util.UintToString(createdCspRole.ID),
				CspRoleName: createdCspRole.Name,
				CspType:     createdCspRole.CspType,
				AuthMethod:  cspRole.AuthMethod,
			})
		}
	}

	// // Convert MenuIDs from string to uint
	// menuIDs := make([]uint, 0)
	// for _, menuIDStr := range req.MenuIDs {
	// 	menuID, err := util.StringToUint(menuIDStr)
	// 	if err != nil {
	// 		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Invalid menu ID format: %s", menuIDStr)})
	// 	}
	// 	menuIDs = append(menuIDs, menuID)
	// }

	// 2. Create role and all dependencies together in transaction
	createdRole, err := h.roleService.CreateRoleWithAllDependencies(role, roleSubs, req.MenuIDs, createdCspRoles, req.Description)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, createdRole)
}

// @Summary Get role by ID
// @Description Get role details by ID
// @Tags roles
// @Accept json
// @Produce json
// @Param id path string true "Role ID"
// @Success 200 {object} model.RoleMaster
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/id/{roleId} [get]
// @Id getRoleByRoleID
func (h *RoleHandler) GetRoleByRoleID(c echo.Context) error {
	roleTypeStr := c.QueryParam("roleType")
	roleType := constants.IAMRoleType(roleTypeStr)
	log.Printf("Role list retrieval request - type: %s", roleType)

	id, err := strconv.ParseUint(c.Param("roleId"), 10, 32)
	if err != nil {
		log.Printf("Invalid role ID format: %s", c.Param("roleId"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid roleId ID format"})
	}

	log.Printf("Role retrieval request - ID: %d", id)

	role, err := h.roleService.GetRoleByID(uint(id), roleType)
	if err != nil {
		log.Printf("Failed to retrieve role - ID: %d, error: %v", id, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to retrieve role: %v", err)})
	}

	if role == nil {
		log.Printf("Role not found - ID: %d", id)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Role with the specified ID not found"})
	}

	log.Printf("Successfully retrieved role - ID: %d", id)
	return c.JSON(http.StatusOK, role)
}

// @Summary Get role by Name
// @Description Retrieve role details by role name.
// @Tags roles
// @Accept json
// @Produce json
// @Param name path string true "Role name"
// @Success 200 {object} model.RoleMaster
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/name/{roleName} [get]
// @Id getRoleByRoleName
func (h *RoleHandler) GetRoleByRoleName(c echo.Context) error {
	roleTypeStr := c.QueryParam("roleType")
	roleType := constants.IAMRoleType(roleTypeStr)
	log.Printf("Role list retrieval request - type: %s", roleType)

	roleName := c.Param("roleName")

	role, err := h.roleService.GetRoleByName(roleName, roleType)
	if err != nil {
		log.Printf("Failed to retrieve role - Name: %s, error: %v", roleName, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to retrieve role: %v", err)})
	}

	if role == nil {
		log.Printf("Role not found - ID: %v", roleName)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Role with the specified ID not found"})
	}

	return c.JSON(http.StatusOK, role)
}

// @Summary Update role
// @Description Update the details of an existing role.
// @Tags roles
// @Accept json
// @Produce json
// @Param id path string true "Role ID"
// @Param role body model.CreateRoleRequest true "Role Info"
// @Success 200 {object} model.RoleMaster
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/id/{roleId} [put]
// @Id updateRole
func (h *RoleHandler) UpdateRole(c echo.Context) error {
	roleId := c.Param("roleId")

	roleIdInt, err := util.StringToUint(roleId)
	if err != nil {
		log.Printf("Role ID conversion error: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid role ID format"})
	}

	var req model.CreateRoleRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Failed to bind role update request - ID: %d, error: %v", roleIdInt, err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	log.Printf("Role update request - ID: %d, name: %s, description: %s, parentID: %d, roleTypes: %v",
		roleIdInt, req.Name, req.Description, req.ParentID, req.RoleTypes)

	if err := c.Validate(&req); err != nil {
		log.Printf("Role update input validation failed - ID: %d, error: %v", roleIdInt, err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Input validation failed: %v", err)})
	}

	role := model.RoleMaster{
		ID:          roleIdInt,
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
	}

	updatedRole, err := h.roleService.UpdateRoleWithSubs(role, req.RoleTypes)
	if err != nil {
		log.Printf("Failed to update role - ID: %d, error: %v", roleIdInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to update role: %v", err)})
	}

	if len(req.MenuIDs) > 0 {
		// Delete existing mappings
		if err := h.menuService.DeleteRoleMenuMappingsByRoleID(roleIdInt); err != nil {
			log.Printf("Failed to delete role menu mappings - ID: %d, error: %v", roleIdInt, err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to delete role menu mappings: %v", err)})
		}
		// Create new mappings
		mappings := make([]*model.RoleMenuMapping, 0)
		for _, menuID := range req.MenuIDs {
			mapping := &model.RoleMenuMapping{
				RoleID: roleIdInt,
				MenuID: menuID,
			}
			mappings = append(mappings, mapping)
		}
		if err := h.menuService.CreateRoleMenuMappings(mappings); err != nil {
			log.Printf("Failed to create role menu mappings - ID: %d, error: %v", roleIdInt, err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to create role menu mappings: %v", err)})
		}
	}

	// TODO : Same logic exists in createRole. Remove duplicate code.
	if len(req.CspRoles) > 0 {
		// Delete existing mappings
		if err := h.roleService.DeleteRoleCspRoleMappingsByRoleId(roleIdInt); err != nil {
			log.Printf("Failed to delete role csp mappings - ID: %d, error: %v", roleIdInt, err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to delete role csp mappings: %v", err)})
		}

		for _, cspRole := range req.CspRoles {
			// Use existing csp role if available, otherwise create new one
			if cspRole.CspRoleName == "" {
				cspRoleName := constants.CspRoleNamePrefix + req.Name // Create cspRoleName by adding prefix to roleName
				cspRole.CspRoleName = cspRoleName
			}

			if !strings.HasPrefix(cspRole.CspRoleName, constants.CspRoleNamePrefix) {
				cspRole.CspRoleName = constants.CspRoleNamePrefix + cspRole.CspRoleName
			}

			// Check if CSP role exists
			exists, err := h.cspRoleService.ExistCspRoleByNameAndType(cspRole.CspRoleName, cspRole.CspType)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to check CSP role existence: %v", err)})
			}

			if exists {
				// Get ID if existing CSP role exists
				existingCspRole, err := h.cspRoleService.GetCspRoleByName(cspRole.CspRoleName, cspRole.CspType)
				if err != nil {
					return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to retrieve CSP role: %v", err)})
				}
				cspRole.ID = util.UintToString(existingCspRole.ID)
			} else {
				log.Printf("cspRole requested: %v", cspRole)
				// Create or update CSP role (mapping is handled together)
				newCspRole, err := h.cspRoleService.CreateOrUpdateCspRole(&cspRole)
				if err != nil {
					return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to update CSP role: %v", err)})
				}
				cspRole.ID = util.UintToString(newCspRole.ID)
			}

			// Create new mapping
			// AuthMethod가 비어있으면 CreateRoleCspRoleMapping(repository)이 OIDC로 기본 반영한다.
			roleCspRoleMappingRequest := &model.CreateRoleMasterCspRoleMappingRequest{
				RoleID:      roleId,
				CspRoleID:   cspRole.ID,
				AuthMethod:  cspRole.AuthMethod,
				Description: req.Description,
			}

			err = h.roleService.AddCspRolesMapping(roleCspRoleMappingRequest)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to assign role: %v", err)})
			}
		}
		// if err := h.roleService.UpdateRoleCspRoleMappings(updatedRole.ID, req.CspRoles); err != nil {
		// 	log.Printf("Failed to update role csp mappings - ID: %d, error: %v", id, err)
		// 	return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to update role csp mappings: %v", err)})
		// }
	}

	log.Printf("Successfully updated role - ID: %d", roleIdInt)
	return c.JSON(http.StatusOK, updatedRole)
}

// @Summary Delete role
// @Description Delete a role by its name.
// @Tags roles
// @Accept json
// @Produce json
// @Param id path string true "Role ID"
// @Success 204 "No Content"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/id/{roleId} [delete]
// @Id deleteRole
func (h *RoleHandler) DeleteRole(c echo.Context) error {
	roleIdInt, err := util.StringToUint(c.Param("roleId"))
	if err != nil {
		log.Printf("Invalid role ID format: %s", c.Param("id"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid Role ID format"})
	}
	log.Printf("Role deletion request - ID: %d", roleIdInt)
	var req model.AssignRoleRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Failed to bind role deletion request: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	reqRoleType := constants.IAMRoleType(req.RoleType)
	// Retrieve role
	role, err := h.roleService.GetRoleByID(roleIdInt, reqRoleType)
	if err != nil {
		log.Printf("Failed to retrieve role - ID: %d, error: %v", roleIdInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to retrieve role: %v", err)})
	}

	if role == nil {
		log.Printf("Role not found - ID: %d", roleIdInt)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Role with the specified ID not found"})
	}

	// Predefined roles cannot be deleted
	if role.Predefined {
		log.Printf("Attempted to delete predefined role - ID: %d", roleIdInt)
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Predefined roles cannot be deleted"})
	}

	// This endpoint deletes the entire role (role_master + every role_sub it has —
	// platform/workspace/csp), regardless of reqRoleType; reqRoleType only scopes
	// the existence lookup above. See TC-IAM-TECH-025-03: a request scoped to one
	// type must still clean up a realm role tied to a platform sub on the same
	// role_id. DeleteRoleWithSubsAndMappings itself only cascades to role_master
	// once no role_sub is left, so passing all types here is what makes that
	// happen for this "delete everything" caller.
	targetRoleTypes := []constants.IAMRoleType{constants.RoleTypePlatform, constants.RoleTypeWorkspace, constants.RoleTypeCSP}

	// Check if users are assigned to this role -> if yes, cannot delete.
	assigned, err := h.roleService.IsRoleAssignedToUsers(roleIdInt, targetRoleTypes)
	if err != nil {
		log.Printf("Failed to check role assignment - ID: %d, error: %v", roleIdInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to check role assignment: %v", err)})
	}
	if assigned {
		log.Printf("Attempted to delete role assigned to users - ID: %d", roleIdInt)
		return c.JSON(http.StatusConflict, map[string]string{"error": "Role is assigned to one or more users and cannot be deleted"})
	}

	// role.RoleSubs above may be preloaded scoped to reqRoleType only (see
	// FindRoleByRoleID), which would hide a platform-type sub on this same
	// role_id when a different roleType was requested, so the Keycloak realm
	// role check below looks this up independently of that scoping.
	isPlatformRole := false
	if platformSub, err := h.roleService.GetRoleByID(roleIdInt, constants.RoleTypePlatform); err != nil {
		log.Printf("Failed to check platform role sub - ID: %d, error: %v", roleIdInt, err)
	} else {
		isPlatformRole = platformSub != nil
	}

	// Delete every role sub this role has and their mappings (menu, CSP)
	// atomically, cascading to role_master since no sub is left for it
	// afterward. A mid-way failure (e.g. an FK violation from a stale assignment
	// check race) rolls the whole transaction back, leaving no orphaned data.
	if err := h.roleService.DeleteRoleWithSubsAndMappings(roleIdInt, targetRoleTypes); err != nil {
		log.Printf("Failed to delete role - ID: %d, error: %v", roleIdInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to delete role: %v", err)})
	}

	// Keycloak realm role cleanup (best-effort). Platform role assignment (user or
	// group) is the only path that creates a realm role for this role name
	// (AssignRole/AssignGroupPlatformRole), so only attempt this for platform roles.
	// role.Name has a DB-level unique constraint, so no other role_master can still
	// reference this realm role name.
	if isPlatformRole {
		if err := h.keycloakService.DeleteRealmRole(c.Request().Context(), role.Name); err != nil {
			log.Printf("Failed to delete keycloak realm role for role - ID: %d, name: %s, error: %v", roleIdInt, role.Name, err)
		}
	}

	log.Printf("Successfully deleted role - ID: %d", roleIdInt)
	//return c.NoContent(http.StatusNoContent)
	return c.JSON(http.StatusOK, map[string]string{"message": "Role deleted successfully"})
}

// @Summary Assign role
// @Description Assign a role to a user
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.AssignRoleRequest true "Role Assignment Info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/id/{roleId}/assign [post]
// @Id assignRole
func (h *RoleHandler) AssignRole(c echo.Context) error {
	var req model.AssignRoleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	reqRoleType := constants.IAMRoleType(req.RoleType)

	// Log request object contents
	log.Printf("[DEBUG] AssignRole Request: UserID=%s, Username=%s, RoleID=%s, RoleName=%s, RoleType=%s, WorkspaceID=%s",
		req.UserID, req.Username, req.RoleID, req.RoleName, reqRoleType, req.WorkspaceID)

	// Request validation
	if req.RoleID == "" && req.RoleName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Role ID or role name is required"})
	}
	if req.UserID == "" && req.Username == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "User ID or username is required"})
	}

	log.Printf("Role assignment request - UserID: %s, Username: %s, RoleID: %s, RoleType: %s, WorkspaceID: %s",
		req.UserID, req.Username, req.RoleID, reqRoleType, req.WorkspaceID)

	if err := c.Validate(&req); err != nil {
		log.Printf("Role assignment input validation failed: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Input validation failed: %v", err)})
	}

	// Process user ID
	var userID uint
	var err error
	if req.UserID != "" {
		userID, err = util.StringToUint(req.UserID)
		if err != nil {
			log.Printf("User ID conversion error: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID format"})
		}
	} else if req.Username != "" {
		// Retrieve user by username
		user, err := h.userService.GetUserByUsername(c.Request().Context(), req.Username)
		if err != nil {
			log.Printf("Failed to retrieve user - username: %s, error: %v", req.Username, err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to retrieve user: %v", err)})
		}
		if user == nil {
			log.Printf("User not found - username: %s", req.Username)
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User with the specified username not found"})
		}
		userID = user.ID
	} else {
		log.Printf("User identifier missing - UserID: %s, Username: %s", req.UserID, req.Username)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "User ID or username is required"})
	}

	var roleID uint
	if req.RoleID != "" {
		roleID, err = util.StringToUint(req.RoleID)
		if err != nil {
			log.Printf("Role ID conversion error: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid role ID format"})
		}
	}
	// Check if role exists
	role, err := h.roleService.GetRoleByID(roleID, reqRoleType)
	if err != nil {
		log.Printf("Failed to retrieve role - roleID: %s, error: %v", req.RoleID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to retrieve role: %v", err)})
	}
	if role == nil {
		log.Printf("Role not found - roleID: %s", req.RoleID)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Role with the specified ID not found"})
	}

	// Check if role type is supported
	var hasRoleType bool
	for _, sub := range role.RoleSubs {
		if sub.RoleType == reqRoleType {
			hasRoleType = true
			break
		}
	}
	if !hasRoleType {
		log.Printf("Unsupported role type - roleID: %s, requested type: %s", req.RoleID, reqRoleType)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("This role does not support %s type", reqRoleType)})
	}

	var workspaceID uint
	if req.WorkspaceID != "" {
		workspaceID, err = util.StringToUint(req.WorkspaceID)
		if err != nil {
			log.Printf("Workspace ID conversion error: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid workspace ID format"})
		}
	}

	// Assign role
	if reqRoleType == constants.RoleTypePlatform {
		if err := h.roleService.AssignPlatformRole(userID, roleID); err != nil {
			log.Printf("Failed to assign platform role - userID: %d, roleID: %s, error: %v", userID, req.RoleID, err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to assign platform role: %v", err)})
		}
	} else if reqRoleType == constants.RoleTypeWorkspace {
		if workspaceID == 0 {
			log.Printf("Workspace ID missing - userID: %d, roleID: %s", userID, req.RoleID)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Workspace ID is required"})
		}
		if err := h.roleService.AssignWorkspaceRole(userID, workspaceID, roleID); err != nil {
			log.Printf("Failed to assign workspace role - userID: %d, workspaceID: %d, roleID: %s, error: %v",
				userID, workspaceID, req.RoleID, err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to assign workspace role: %v", err)})
		}
	}

	log.Printf("Successfully assigned role - userID: %d, roleID: %s, roleType: %s", userID, req.RoleID, reqRoleType)
	return c.JSON(http.StatusOK, map[string]string{"message": fmt.Sprintf("%s role assigned successfully", reqRoleType)})
}

// @Summary Remove role
// @Description Remove a role from a user
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.AssignRoleRequest true "Role Removal Info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/id/{roleId}/unassign [delete]
// @Id removeRole
func (h *RoleHandler) RemoveRole(c echo.Context) error {
	var req model.AssignRoleRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Error RemoveRole : %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}
	reqRoleType := constants.IAMRoleType(req.RoleType)

	// 사용자 ID 처리
	var userID uint
	if req.UserID != "" {
		// userId가 있으면 uint로 변환
		userIDInt, err := util.StringToUint(req.UserID)
		if err != nil {
			log.Printf("Error req.UserID : %v", req.UserID)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 사용자 ID 형식입니다"})
		}
		userID = userIDInt
	} else if req.Username != "" {
		// userId가 없고 username이 있으면 사용자 조회
		user, err := h.userService.GetUserByUsername(c.Request().Context(), req.Username)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("사용자 조회 실패: %v", err)})
		}
		if user == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 사용자명의 사용자를 찾을 수 없습니다"})
		}
		userID = user.ID
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID 또는 사용자명이 필요합니다"})
	}

	// 역할 ID 처리
	var roleID uint
	if req.RoleID != "" {
		// roleId가 있으면 uint로 변환
		roleIDInt, err := util.StringToUint(req.RoleID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID 형식입니다"})
		}
		roleID = roleIDInt
	} else if req.RoleName != "" {
		// roleId가 없고 roleName이 있으면 역할 조회
		role, err := h.roleService.GetRoleByName(req.RoleName, constants.IAMRoleType(req.RoleType))
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
		}
		if role == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 역할명의 역할을 찾을 수 없습니다"})
		}
		roleID = role.ID
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "역할 ID 또는 역할명이 필요합니다"})
	}

	// 역할 타입 확인
	if reqRoleType == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "역할 타입이 필요합니다"})
	}

	// 역할 제거
	if reqRoleType == constants.RoleTypePlatform {
		if err := h.roleService.RemovePlatformRole(userID, roleID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("플랫폼 역할 제거 실패: %v", err)})
		}
	} else if reqRoleType == constants.RoleTypeWorkspace {
		var workspaceID uint
		if req.WorkspaceID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "워크스페이스 ID가 필요합니다"})
		}
		// WorkspaceID가 있으면 uint로 변환
		workspaceIDInt, err := util.StringToUint(req.WorkspaceID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID 형식입니다"})
		}
		workspaceID = workspaceIDInt

		if err := h.roleService.RemoveWorkspaceRole(userID, workspaceID, roleID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 역할 제거 실패: %v", err)})
		}
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "지원하지 않는 역할 타입입니다"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": fmt.Sprintf("%s 역할이 성공적으로 제거되었습니다", req.RoleType)})
}

// @Summary List menu roles
// @Description Get a list of all menu roles
// @Tags roles
// @Accept json
// @Produce json
// @Success 200 {array} model.RoleMaster
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/platform-roles/list [post]
// @Id listPlatformRoles
func (h *RoleHandler) ListPlatformRoles(c echo.Context) error {
	var req model.RoleFilterRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Error ListPlatformRoles : %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	// workspace 역할만 조회
	if req.RoleTypes == nil {
		req.RoleTypes = []constants.IAMRoleType{constants.RoleTypePlatform}
	}

	roles, err := h.roleService.ListRoles(&req)
	if err != nil {
		log.Printf("역할 목록 조회 실패: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 목록 조회 실패: %v", err)})
	}

	log.Printf("역할 목록 조회 성공 - 조회된 역할 수: %d", len(roles))
	return c.JSON(http.StatusOK, roles)
}

// @Summary List workspace roles
// @Description Get a list of all workspace roles
// @Tags roles
// @Accept json
// @Produce json
// @Success 200 {array} model.RoleMaster
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/workspace-roles/list [post]
// @Id listRolesOfWorkspaceType
func (h *RoleHandler) ListWorkspaceRoles(c echo.Context) error {
	var req model.RoleFilterRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Error ListWorkspaceRoles : %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	// Only query workspace roles
	if req.RoleTypes == nil {
		req.RoleTypes = []constants.IAMRoleType{constants.RoleTypeWorkspace}
	}
	log.Printf("req ListWorkspaceRoles : %v", req)
	roles, err := h.roleService.ListWorkspaceRoles(&req)
	if err != nil {
		log.Printf("Failed to retrieve role list: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to retrieve role list: %v", err)})
	}

	log.Printf("Successfully retrieved role list - number of roles: %d", len(roles))
	return c.JSON(http.StatusOK, roles)
}

// @Summary List csp roles
// @Description Get a list of all csp roles stored in the database
// @Tags roles
// @Accept json
// @Produce json
// @Success 200 {array} model.CspRole
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/csp/list [post]
// @Id listCSPRoles
func (h *RoleHandler) ListCSPRoles(c echo.Context) error {
	roles, err := h.cspRoleService.ListCspRolesFromDB()
	if err != nil {
		log.Printf("Failed to retrieve CSP role list: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to retrieve CSP role list: %v", err)})
	}

	log.Printf("Successfully retrieved CSP role list - number of roles: %d", len(roles))
	return c.JSON(http.StatusOK, roles)
}

// @Summary Create menu role
// @Description Create a new menu role
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.CreateRoleRequest true "Menu Role Creation Info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/platform-roles [post]
// @Id createPlatformRole
func (h *RoleHandler) CreatePlatformRole(c echo.Context) error {
	var req model.CreateRoleRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Failed to bind role creation request: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	log.Printf("Role creation request - name: %s, description: %s, parentID: %d, roleTypes: %v",
		req.Name, req.Description, req.ParentID, req.RoleTypes)

	// Input validation
	if err := c.Validate(&req); err != nil {
		log.Printf("Role creation input validation failed: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Input validation failed: %v", err)})
	}

	// Create RoleMaster
	role := &model.RoleMaster{
		Name:        req.Name,
		Description: req.Description,
		Predefined:  false,
	}

	// Create RoleSubs
	roleSubs := make([]model.RoleSub, len(req.RoleTypes))
	for i, roleType := range req.RoleTypes {
		log.Printf("roleType: %s", roleType)
		roleSubs[i] = model.RoleSub{
			RoleID:   role.ID,
			RoleType: constants.RoleTypePlatform,
		}
	}

	// Create role and subtypes
	createdRole, err := h.roleService.CreateRoleWithSubs(role, roleSubs)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, createdRole)
}

// @Summary Create workspace role
// @Description Create a new workspace role
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.CreateRoleRequest true "Workspace Role Creation Info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/workspace-roles [post]
// @Id createWorkspaceRole
func (h *RoleHandler) CreateWorkspaceRole(c echo.Context) error {
	var req model.CreateRoleRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Failed to bind role creation request: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	log.Printf("Role creation request - name: %s, description: %s, parentID: %d, roleTypes: %v",
		req.Name, req.Description, req.ParentID, req.RoleTypes)

	// Input validation
	if err := c.Validate(&req); err != nil {
		log.Printf("Role creation input validation failed: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Input validation failed: %v", err)})
	}

	// Create RoleMaster
	role := &model.RoleMaster{
		Name:        req.Name,
		Description: req.Description,
		Predefined:  false,
	}

	// Create RoleSubs
	roleSubs := make([]model.RoleSub, len(req.RoleTypes))
	for i, roleType := range req.RoleTypes {
		log.Printf("roleType: %s", roleType)
		roleSubs[i] = model.RoleSub{
			RoleID:   role.ID,
			RoleType: constants.RoleTypeWorkspace,
		}
	}

	// Create role and subtypes
	createdRole, err := h.roleService.CreateRoleWithSubs(role, roleSubs)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	log.Printf("Role creation successful - ID: %d", createdRole.ID)
	return c.JSON(http.StatusCreated, createdRole)
}

// @Summary Create csp role
// @Description Create a new csp role
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.CreateRoleRequest true "CSP Role Creation Info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/csp [post]
// @Id createCspRole
func (h *RoleHandler) CreateCspRole(c echo.Context) error {
	var req model.CreateCspRoleRequest

	if err := c.Bind(&req); err != nil {
		c.Logger().Debugf("Bind error: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다_"})
	}

	if err := c.Validate(&req); err != nil {
		c.Logger().Debugf("Validate error: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	createdRole, err := h.cspRoleService.CreateCspRole(&req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, createdRole)
}

// @Summary Update CSP role
// @Description CspRole(mcmp_role_csp_roles) 레코드를 수정합니다. Name/Description/ExtendedConfig를 반영합니다.
// @Description AWS의 경우 클라우드 측 리소스는 건드리지 않고 DB만 갱신합니다(SAML saml_client_id 등 설정용).
// @Tags roles
// @Accept json
// @Produce json
// @Param roleId path string true "CSP Role ID"
// @Param role body model.CreateCspRoleRequest true "CSP Role Info"
// @Success 200 {object} model.CspRole
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/csp/id/{roleId} [put]
// @Id updateCspRole
func (h *RoleHandler) UpdateCspRole(c echo.Context) error {
	roleIDInt, err := util.StringToUint(c.Param("roleId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 csp 역할 ID 형식입니다"})
	}

	var req model.CreateCspRoleRequest
	if err := c.Bind(&req); err != nil {
		c.Logger().Debugf("Bind error: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	if err := h.cspRoleService.UpdateCspRole(roleIDInt, &req); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	updatedRole, err := h.cspRoleService.GetCspRoleByID(roleIDInt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, updatedRole)
}

// @Summary Get platform role by ID
// @Description Get platform role details by ID
// @Tags roles
// @Accept json
// @Produce json
// @Param id path string true "Platform Role ID"
// @Success 200 {object} model.RoleMaster
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/platform-roles/id/{roleId} [get]
// @Id getPlatformRoleByID
func (h *RoleHandler) GetPlatformRoleByID(c echo.Context) error {
	roleType := constants.RoleTypePlatform
	log.Printf("Role list retrieval request - type: %s", roleType)

	roleIDInt, err := util.StringToUint(c.Param("roleId"))
	if err != nil {
		log.Printf("Invalid role ID format: %s", c.Param("roleId"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid Platform Role ID format"})
	}

	log.Printf("Role retrieval request - ID: %d", roleIDInt)

	role, err := h.roleService.GetRoleByID(roleIDInt, roleType)
	if err != nil {
		log.Printf("Failed to retrieve role - ID: %d, error: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to retrieve role: %v", err)})
	}

	if role == nil {
		log.Printf("Role not found - ID: %d", roleIDInt)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Role with the specified ID not found"})
	}

	log.Printf("Successfully retrieved role - ID: %d", roleIDInt)
	return c.JSON(http.StatusOK, role)
}

// @Summary Get menu role by Name
// @Description Get menu role details by Name
// @Tags roles
// @Accept json
// @Produce json
// @Param name path string true "Menu Role Name"
// @Success 200 {object} model.RoleMaster
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/platform-roles/name/{roleName} [get]
// @Id getPlatformRoleByName
func (h *RoleHandler) GetPlatformRoleByName(c echo.Context) error {
	roleType := constants.RoleTypePlatform
	log.Printf("역할 목록 조회 요청 - 타입: %s", roleType)

	roleName := c.Param("roleName")

	role, err := h.roleService.GetRoleByName(roleName, roleType)
	if err != nil {
		log.Printf("역할 조회 실패 - Name: %s, 에러: %v", roleName, err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 Menu ID 형식입니다"})
	}

	if role == nil {
		log.Printf("역할을 찾을 수 없음 - Name: %v", roleName)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 역할을 찾을 수 없습니다"})
	}

	log.Printf("역할 조회 성공 - Name: %v", roleName)
	return c.JSON(http.StatusOK, role)
}

// @Summary Get workspace role by ID
// @Description Get workspace role details by ID
// @Tags roles
// @Accept json
// @Produce json
// @Param id path string true "Workspace Role ID"
// @Success 200 {object} model.RoleMaster
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/workspace-roles/id/{roleId} [get]
// @Id getWorkspaceRoleByID
func (h *RoleHandler) GetWorkspaceRoleByID(c echo.Context) error {
	roleType := constants.RoleTypeWorkspace
	log.Printf("역할 목록 조회 요청 - 타입: %s", roleType)

	roleIDInt, err := util.StringToUint(c.Param("roleId"))
	if err != nil {
		log.Printf("잘못된 역할 ID 형식: %s", c.Param("roleId"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 Role ID 형식입니다"})
	}

	log.Printf("역할 조회 요청 - ID: %d", roleIDInt)

	role, err := h.roleService.GetRoleByID(roleIDInt, roleType)
	if err != nil {
		log.Printf("역할 조회 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
	}

	if role == nil {
		log.Printf("역할을 찾을 수 없음 - ID: %d", roleIDInt)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 역할을 찾을 수 없습니다"})
	}

	log.Printf("역할 조회 성공 - ID: %d", roleIDInt)
	return c.JSON(http.StatusOK, role)
}

// @Summary Get workspace role by Name
// @Description Get workspace role details by Name
// @Tags roles
// @Accept json
// @Produce json
// @Param name path string true "Workspace Role Name"
// @Success 200 {object} model.RoleMaster
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/workspace-roles/name/{roleName} [get]
// @Id getWorkspaceRoleByName
func (h *RoleHandler) GetWorkspaceRoleByName(c echo.Context) error {
	roleType := constants.RoleTypeWorkspace
	log.Printf("역할 목록 조회 요청 - 타입: %s", roleType)

	roleName := c.Param("roleName")

	role, err := h.roleService.GetRoleByName(roleName, roleType)
	if err != nil {
		log.Printf("역할 조회 실패 - Name: %s, 에러: %v", roleName, err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 Role Name 형식입니다"})
	}

	if role == nil {
		log.Printf("역할을 찾을 수 없음 - Name: %v", roleName)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 역할을 찾을 수 없습니다"})
	}

	log.Printf("역할 조회 성공 - Name: %v", roleName)
	return c.JSON(http.StatusOK, role)
}

// @Summary Get csp role by ID
// @Description Get csp role details by ID
// @Tags roles
// @Accept json
// @Produce json
// @Param id path string true "CSP Role ID"
// @Success 200 {object} model.RoleMaster
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/csp/id/{roleId} [get]
// @Id getCspRoleByID
func (h *RoleHandler) GetCspRoleByID(c echo.Context) error {
	roleType := constants.RoleTypeCSP
	log.Printf("역할 목록 조회 요청 - 타입: %s", roleType)

	roleIDInt, err := util.StringToUint(c.Param("roleId"))
	if err != nil {
		log.Printf("잘못된 역할 ID 형식: %s", c.Param("roleId"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 RoleID 형식입니다 for csp role"})
	}

	log.Printf("역할 조회 요청 - ID: %d", roleIDInt)

	role, err := h.roleService.GetCspRoleByID(roleIDInt)
	if err != nil {
		log.Printf("역할 조회 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
	}

	if role == nil {
		log.Printf("역할을 찾을 수 없음 - ID: %d", roleIDInt)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 역할을 찾을 수 없습니다"})
	}

	log.Printf("역할 조회 성공 - ID: %d", roleIDInt)
	return c.JSON(http.StatusOK, role)
}

// @Summary Get csp role by Name
// @Description Get csp role details by Name
// @Tags roles
// @Accept json
// @Produce json
// @Param name path string true "CSP Role Name"
// @Success 200 {object} model.RoleMaster
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/csp/name/{roleName} [get]
// @Id getCspRoleByName
func (h *RoleHandler) GetCspRoleByName(c echo.Context) error {
	roleType := constants.RoleTypeCSP
	log.Printf("csp 역할 목록 조회 요청 - 타입: %s", roleType)

	roleName := c.Param("roleName")

	role, err := h.roleService.GetCspRoleByName(roleName)
	if err != nil {
		log.Printf("csp 역할 조회 실패 - Name: %s, 에러: %v", roleName, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "csp 역할 조회 실패"})
	}

	if role == nil {
		log.Printf("csp 역할을 찾을 수 없음 - Name: %v", roleName)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 역할을 찾을 수 없습니다"})
	}

	log.Printf("csp 역할 조회 성공 - Name: %v", roleName)
	return c.JSON(http.StatusOK, role)
}

// @Summary Update platform role
// @Description Update the details of an existing platform role (RoleMaster: name/description/parentId, RoleSub type list)
// @Tags roles
// @Accept json
// @Produce json
// @Param roleId path string true "Platform Role ID"
// @Param role body model.CreateRoleRequest true "Platform Role Info"
// @Success 200 {object} model.RoleMaster
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/platform-roles/id/{roleId} [put]
// @Id updatePlatformRole
func (h *RoleHandler) UpdatePlatformRole(c echo.Context) error {
	roleIDInt, err := util.StringToUint(c.Param("roleId"))
	if err != nil {
		log.Printf("잘못된 platform 역할 ID 형식: %s", c.Param("roleId"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID 형식입니다"})
	}

	// RoleSub는 RoleMaster와 별도 테이블이라 요청 바디가 아닌 DB의 기존 RoleSub를 기준으로
	// 이 roleId가 실제로 platform 타입인지 검증한다 (요청 바디의 RoleTypes만 보면 다른 타입을
	// 함께 지정해 다른 엔드포인트의 역할까지 건드릴 수 있음).
	existing, err := h.roleService.GetRoleByID(roleIDInt, constants.RoleTypePlatform)
	if err != nil {
		log.Printf("platform 역할 조회 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
	}
	if existing == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "platform 타입의 해당 역할을 찾을 수 없습니다"})
	}

	var req model.CreateRoleRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("platform 역할 수정 요청 바인딩 실패 - 에러: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	if err := c.Validate(&req); err != nil {
		log.Printf("platform 역할 수정 입력값 검증 실패 - 에러: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("입력값 검증 실패: %v", err)})
	}

	role := model.RoleMaster{
		ID:          roleIDInt,
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
	}

	updatedRole, err := h.roleService.UpdateRoleWithSubs(role, req.RoleTypes)
	if err != nil {
		log.Printf("platform 역할 수정 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 수정 실패: %v", err)})
	}

	log.Printf("platform 역할 수정 성공 - ID: %d", roleIDInt)
	return c.JSON(http.StatusOK, updatedRole)
}

// @Summary Update workspace role
// @Description Update the details of an existing workspace role (RoleMaster: name/description/parentId, RoleSub type list)
// @Tags roles
// @Accept json
// @Produce json
// @Param roleId path string true "Workspace Role ID"
// @Param role body model.CreateRoleRequest true "Workspace Role Info"
// @Success 200 {object} model.RoleMaster
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/workspace-roles/id/{roleId} [put]
// @Id updateWorkspaceRole
func (h *RoleHandler) UpdateWorkspaceRole(c echo.Context) error {
	roleIDInt, err := util.StringToUint(c.Param("roleId"))
	if err != nil {
		log.Printf("잘못된 workspace 역할 ID 형식: %s", c.Param("roleId"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID 형식입니다"})
	}

	existing, err := h.roleService.GetRoleByID(roleIDInt, constants.RoleTypeWorkspace)
	if err != nil {
		log.Printf("workspace 역할 조회 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
	}
	if existing == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "workspace 타입의 해당 역할을 찾을 수 없습니다"})
	}

	var req model.CreateRoleRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("workspace 역할 수정 요청 바인딩 실패 - 에러: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	if err := c.Validate(&req); err != nil {
		log.Printf("workspace 역할 수정 입력값 검증 실패 - 에러: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("입력값 검증 실패: %v", err)})
	}

	role := model.RoleMaster{
		ID:          roleIDInt,
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
	}

	updatedRole, err := h.roleService.UpdateRoleWithSubs(role, req.RoleTypes)
	if err != nil {
		log.Printf("workspace 역할 수정 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 수정 실패: %v", err)})
	}

	log.Printf("workspace 역할 수정 성공 - ID: %d", roleIDInt)
	return c.JSON(http.StatusOK, updatedRole)
}

// @Summary Update CSP-type platform role (RoleMaster)
// @Description RoleSub.RoleType이 csp인 RoleMaster(플랫폼 역할 정의: name/description/parentId)를 수정합니다.
// @Description CSP 클라우드 측 역할 레코드(mcmp_role_csp_roles)를 수정하는 UpdateCspRole(/api/roles/csp/id/{roleId})과는 다른 리소스입니다 — 혼동 방지를 위해 operationId를 updateCspRoleMaster로 분리했습니다.
// @Tags roles
// @Accept json
// @Produce json
// @Param roleId path string true "CSP-type Platform Role ID"
// @Param role body model.CreateRoleRequest true "CSP-type Role Info"
// @Success 200 {object} model.RoleMaster
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/csp-roles/master/id/{roleId} [put]
// @Id updateCspRoleMaster
func (h *RoleHandler) UpdateCspRoleMaster(c echo.Context) error {
	roleIDInt, err := util.StringToUint(c.Param("roleId"))
	if err != nil {
		log.Printf("잘못된 csp 역할 ID 형식: %s", c.Param("roleId"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID 형식입니다"})
	}

	existing, err := h.roleService.GetRoleByID(roleIDInt, constants.RoleTypeCSP)
	if err != nil {
		log.Printf("csp 역할 조회 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
	}
	if existing == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "csp 타입의 해당 역할을 찾을 수 없습니다"})
	}

	var req model.CreateRoleRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("csp 역할 수정 요청 바인딩 실패 - 에러: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	if err := c.Validate(&req); err != nil {
		log.Printf("csp 역할 수정 입력값 검증 실패 - 에러: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("입력값 검증 실패: %v", err)})
	}

	role := model.RoleMaster{
		ID:          roleIDInt,
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
	}

	updatedRole, err := h.roleService.UpdateRoleWithSubs(role, req.RoleTypes)
	if err != nil {
		log.Printf("csp 역할 수정 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 수정 실패: %v", err)})
	}

	log.Printf("csp 역할 수정 성공 - ID: %d", roleIDInt)
	return c.JSON(http.StatusOK, updatedRole)
}

// @Summary Delete platform role
// @Description Delete a platform role
// @Tags roles
// @Accept json
// @Produce json
// @Param roleId path string true "Platform Role ID"
// @Success 204 "No Content"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/platform-roles/id/{roleId} [delete]
// @Id deletePlatformRole
func (h *RoleHandler) DeletePlatformRole(c echo.Context) error {
	roleType := constants.RoleTypePlatform

	var req model.CreateRoleRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("platform 역할 수정 요청 바인딩 실패 - 에러: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	roleIDInt, err := util.StringToUint(c.Param("roleId"))
	if err != nil {
		log.Printf("잘못된 platform 역할 ID 형식: %s", c.Param("roleId"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 Role ID 형식입니다"})
	}

	log.Printf("platform 역할 삭제 요청 - ID: %d", roleIDInt)

	// 역할 조회
	role, err := h.roleService.GetRoleByID(roleIDInt, roleType)
	if err != nil {
		log.Printf("platform 역할 조회 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
	}

	if role == nil {
		log.Printf("platform 역할을 찾을 수 없음 - ID: %d", roleIDInt)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 역할을 찾을 수 없습니다"})
	}

	// Predefined 역할은 삭제 불가
	if role.Predefined {
		log.Printf("미리 정의된 역할 삭제 시도 - ID: %d", roleIDInt)
		return c.JSON(http.StatusForbidden, map[string]string{"error": "미리 정의된 역할은 삭제할 수 없습니다"})
	}

	// role_master는 여러 role_sub(platform/workspace/csp)를 동시에 가질 수 있으므로,
	// 이 platform sub 삭제로 남은 sub이 없을 때만 role_master도 함께 삭제된다.
	if err := h.roleService.DeleteRoleWithSubsAndMappings(roleIDInt, []constants.IAMRoleType{constants.RoleTypePlatform}); err != nil {
		log.Printf("platform 역할 삭제 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 삭제 실패: %v", err)})
	}

	log.Printf("platform 역할 삭제 성공 - ID: %d", roleIDInt)
	return c.NoContent(http.StatusNoContent)
}

// @Summary Delete workspace role
// @Description Delete a workspace role
// @Tags roles
// @Accept json
// @Produce json
// @Param roleId path string true "Workspace Role ID"
// @Success 204 "No Content"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/workspace-roles/id/{roleId} [delete]
// @Id deleteWorkspaceRole
func (h *RoleHandler) DeleteWorkspaceRole(c echo.Context) error {
	roleType := constants.RoleTypeWorkspace

	var req model.CreateRoleRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("workspace 역할 수정 요청 바인딩 실패 - 에러: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	roleIDInt, err := util.StringToUint(c.Param("roleId"))
	if err != nil {
		log.Printf("잘못된 workspace 역할 ID 형식: %s", c.Param("roleId"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 Workspace Role ID 형식입니다"})
	}

	log.Printf("workspace 역할 삭제 요청 - ID: %d", roleIDInt)

	// 역할 조회
	role, err := h.roleService.GetRoleByID(roleIDInt, roleType)
	if err != nil {
		log.Printf("workspace 역할 조회 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
	}

	if role == nil {
		log.Printf("workspace 역할을 찾을 수 없음 - ID: %d", roleIDInt)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 역할을 찾을 수 없습니다"})
	}

	// Predefined 역할은 삭제 불가
	if role.Predefined {
		log.Printf("미리 정의된 역할 삭제 시도 - ID: %d", roleIDInt)
		return c.JSON(http.StatusForbidden, map[string]string{"error": "미리 정의된 역할은 삭제할 수 없습니다"})
	}

	// role_master는 여러 role_sub(platform/workspace/csp)를 동시에 가질 수 있으므로,
	// 이 workspace sub 삭제로 남은 sub이 없을 때만 role_master도 함께 삭제된다.
	if err := h.roleService.DeleteRoleWithSubsAndMappings(roleIDInt, []constants.IAMRoleType{constants.RoleTypeWorkspace}); err != nil {
		log.Printf("workspace 역할 삭제 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 삭제 실패: %v", err)})
	}

	log.Printf("workspace 역할 삭제 성공 - ID: %d", roleIDInt)
	return c.JSON(http.StatusOK, map[string]string{"message": "workspace 역할 삭제 성공"})
}

// @Summary Delete csp role
// @Description CspRole(mcmp_role_csp_roles) 레코드를 삭제합니다. 이 레코드를 참조하는 RoleMaster-CspRole
// @Description 매핑을 먼저 정리한 뒤 레코드를 삭제하며, AWS인 경우 실제 클라우드 Role도 함께 삭제합니다.
// @Description RoleMaster/RoleSub(csp)는 건드리지 않습니다 — 그 삭제는 DeleteCspRoleMaster를 사용하세요.
// @Tags roles
// @Accept json
// @Produce json
// @Param roleId path string true "CSP Role ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/csp/id/{roleId} [delete]
// @Id deleteCspRole
func (h *RoleHandler) DeleteCspRole(c echo.Context) error {
	cspRoleIDInt, err := util.StringToUint(c.Param("roleId"))
	if err != nil {
		log.Printf("잘못된 csp 역할 ID 형식: %s", c.Param("roleId"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 CSP Role ID 형식입니다"})
	}

	log.Printf("csp 역할 레코드 삭제 요청 - ID: %d", cspRoleIDInt)

	existing, err := h.cspRoleService.GetCspRoleByID(cspRoleIDInt)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("csp 역할을 찾을 수 없음 - ID: %d", cspRoleIDInt)
			return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 CSP 역할을 찾을 수 없습니다"})
		}
		log.Printf("csp 역할 조회 실패 - ID: %d, 에러: %v", cspRoleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
	}
	if existing == nil {
		log.Printf("csp 역할을 찾을 수 없음 - ID: %d", cspRoleIDInt)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 CSP 역할을 찾을 수 없습니다"})
	}

	if err := h.roleService.DeleteRoleCspRoleMappingsByCspRoleID(cspRoleIDInt); err != nil {
		log.Printf("csp 역할 매핑 삭제 실패 - ID: %d, 에러: %v", cspRoleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 매핑 삭제 실패: %v", err)})
	}

	if err := h.cspRoleService.DeleteCspRole(cspRoleIDInt); err != nil {
		log.Printf("csp 역할 삭제 실패 - ID: %d, 에러: %v", cspRoleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 삭제 실패: %v", err)})
	}

	log.Printf("csp 역할 레코드 삭제 성공 - ID: %d", cspRoleIDInt)
	return c.NoContent(http.StatusNoContent)
}

// @Summary Create csp role master
// @Description RoleMaster + RoleSub(csp)를 생성합니다. CreatePlatformRole/CreateWorkspaceRole과 대칭되는
// @Description csp 타입 전용 생성 API입니다. 대상 CSP의 실제 역할(CspRole 레코드)은 CreateCspRole로 별도 생성/매핑해야 합니다.
// @Tags roles
// @Accept json
// @Produce json
// @Param role body model.CreateRoleRequest true "Role Info"
// @Success 201 {object} model.RoleMaster
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/csp-roles/master [post]
// @Id createCspRoleMaster
func (h *RoleHandler) CreateCspRoleMaster(c echo.Context) error {
	var req model.CreateRoleRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Failed to bind role creation request: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	log.Printf("Role creation request - name: %s, description: %s, parentID: %d, roleTypes: %v",
		req.Name, req.Description, req.ParentID, req.RoleTypes)

	if err := c.Validate(&req); err != nil {
		log.Printf("Role creation input validation failed: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Input validation failed: %v", err)})
	}

	role := &model.RoleMaster{
		Name:        req.Name,
		Description: req.Description,
		Predefined:  false,
	}

	roleSubs := make([]model.RoleSub, len(req.RoleTypes))
	for i := range req.RoleTypes {
		roleSubs[i] = model.RoleSub{
			RoleID:   role.ID,
			RoleType: constants.RoleTypeCSP,
		}
	}

	createdRole, err := h.roleService.CreateRoleWithSubs(role, roleSubs)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	log.Printf("Role creation successful - ID: %d", createdRole.ID)
	return c.JSON(http.StatusCreated, createdRole)
}

// @Summary Delete csp role master
// @Description RoleMaster의 csp 타입 RoleSub 태그를 삭제합니다(DeletePlatformRole/DeleteWorkspaceRole과 대칭).
// @Description RoleMaster 자체나 CspRole 레코드는 삭제하지 않습니다 — CspRole 레코드 삭제는 DeleteCspRole을 사용하세요.
// @Description 이 역할에 CSP 역할 매핑이 남아있으면 삭제할 수 없습니다(먼저 매핑을 해제해야 합니다).
// @Tags roles
// @Accept json
// @Produce json
// @Param roleId path string true "Role ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/csp-roles/master/id/{roleId} [delete]
// @Id deleteCspRoleMaster
func (h *RoleHandler) DeleteCspRoleMaster(c echo.Context) error {
	roleType := constants.RoleTypeCSP

	roleIDInt, err := util.StringToUint(c.Param("roleId"))
	if err != nil {
		log.Printf("잘못된 csp 역할 ID 형식: %s", c.Param("roleId"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 Role ID 형식입니다"})
	}

	log.Printf("csp 역할(master) 삭제 요청 - ID: %d", roleIDInt)

	// 역할 조회
	role, err := h.roleService.GetRoleByID(roleIDInt, roleType)
	if err != nil {
		log.Printf("csp 역할 조회 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
	}

	if role == nil {
		log.Printf("csp 역할을 찾을 수 없음 - ID: %d", roleIDInt)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 역할을 찾을 수 없습니다"})
	}

	// Predefined 역할은 삭제 불가
	if role.Predefined {
		log.Printf("미리 정의된 역할 삭제 시도 - ID: %d", roleIDInt)
		return c.JSON(http.StatusForbidden, map[string]string{"error": "미리 정의된 역할은 삭제할 수 없습니다"})
	}

	// 역할에 매핑된 csp역할이 있으면 삭제 불가(먼저 삭제해야함.)
	mappingReq := model.FilterRoleMasterMappingRequest{}
	mappingReq.RoleID = c.Param("roleId")
	if mappingReq.RoleTypes == nil {
		mappingReq.RoleTypes = []constants.IAMRoleType{constants.RoleTypeCSP}
	}
	mappings, err := h.roleService.ListRoleMasterMappings(&mappingReq)
	if err != nil {
		log.Printf("csp 역할 매핑 조회 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 매핑 조회 실패: %v", err)})
	}
	// mappings는 이 role_id가 csp 타입 role_sub을 가지기만 해도 항상 1건(RoleMasterMapping
	// 컨테이너 자체) 채워지므로, len(mappings) > 0으로는 실제 CSP 역할 매핑 유무를 알 수
	// 없다 — 매핑 삭제 대상 role은 자기 자신도 csp sub을 가지므로 이 조건이 늘 참이 되어
	// DeleteCspRoleMaster가 항상 403을 반환하는 버그였다. 실제 매핑 목록
	// (RoleMasterCspRoleMappings)의 길이로 확인해야 한다.
	if len(mappings) > 0 && len(mappings[0].RoleMasterCspRoleMappings) > 0 {
		log.Printf("csp 역할 매핑이 있어 삭제 불가 - ID: %d", roleIDInt)
		return c.JSON(http.StatusForbidden, map[string]string{"error": "csp 역할 매핑이 있어 삭제 불가"})
	}

	// role_master는 여러 role_sub(platform/workspace/csp)를 동시에 가질 수 있으므로,
	// 이 csp sub 삭제로 남은 sub이 없을 때만 role_master도 함께 삭제된다.
	if err := h.roleService.DeleteRoleWithSubsAndMappings(roleIDInt, []constants.IAMRoleType{constants.RoleTypeCSP}); err != nil {
		log.Printf("csp 역할(master) 삭제 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 삭제 실패: %v", err)})
	}

	log.Printf("csp 역할(master) 삭제 성공 - ID: %d", roleIDInt)
	return c.NoContent(http.StatusNoContent)
}

// @Summary Get user workspace roles
// @Description Get roles assigned to a user in a workspace
// @Tags roles
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {array} model.RoleMaster
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/workspaces/id/{workspaceId}/users/id/{userId} [get]
// @Id getUserWorkspaceRoles
func (h *RoleHandler) GetUserWorkspaceRoles(c echo.Context) error {
	log.Printf("GetUserWorkspaceRoles : %v", c.Param("userId"))
	log.Printf("GetUserWorkspaceRoles : %v", c.Param("workspaceId"))
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 사용자 ID 형식입니다"})
	}

	workspaceID, err := strconv.ParseUint(c.Param("workspaceId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식입니다"})
	}

	roles, err := h.roleService.GetUserWorkspaceRoles(uint(userID), uint(workspaceID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 역할 목록 조회 실패: %v", err)})
	}

	return c.JSON(http.StatusOK, roles)
}

// @Summary Assign platform role
// @Description Assign a platform role to a user
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.AssignRoleRequest true "Platform Role Assignment Info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/assign/platform-role [post]
// @Id assignPlatformRole
func (h *RoleHandler) AssignPlatformRole(c echo.Context) error {
	var req model.AssignRoleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}
	c.Logger().Debugf("[DEBUG] AssignRole Request: UserID=%s, Username=%s, RoleID=%s, RoleName=%s, RoleType=%s, WorkspaceID=%s",
		req.UserID, req.Username, req.RoleID, req.RoleName, req.RoleType, req.WorkspaceID)
	if req.UserID == "" || req.RoleID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID와 역할 ID가 필요합니다"})
	}

	var roleID uint
	var userID uint
	if req.RoleID != "" {
		rid, err := strconv.ParseUint(req.RoleID, 10, 32)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID 형식입니다"})
		}
		roleID = uint(rid)
	}
	if req.UserID != "" {
		uid, err := util.StringToUint(req.UserID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 사용자 ID 형식입니다"})
		}
		userID = uint(uid)
	}

	// RoleName이 없으면 DB에서 roleID로 조회
	if req.RoleName == "" {
		role, err := h.roleService.GetRoleByID(roleID, constants.RoleTypePlatform)
		if err != nil || role == nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("역할 조회 실패 (roleId=%d): %v", roleID, err)})
		}
		req.RoleName = role.Name
	}

	// User 찾기
	//user, err := h.userService.GetUserByUsername(c.Request().Context(), req.Username)
	user, err := h.userService.GetUserByID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("사용자 조회 실패: %v", err)})
	}
	if user == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 사용자를 찾을 수 없습니다"})
	}

	// 이미 할당 되어있는지 확인.
	isAssignedPlatformRole, err := h.roleService.IsAssignedPlatformRole(userID, roleID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 할당 확인 실패: %v", err)})
	}
	if isAssignedPlatformRole {
		// DB에는 이미 있음 — Keycloak 동기화 상태 확인 (idempotent 처리)
		isKcAssigned, err := h.keycloakService.IsRealmRoleAssignedToUser(c.Request().Context(), user.KcId, req.RoleName)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("키클로크 역할 확인 실패: %v", err)})
		}
		if isKcAssigned {
			// DB + Keycloak 모두 할당됨 — idempotent 성공 반환
			return c.JSON(http.StatusOK, map[string]string{"message": "플랫폼 역할이 성공적으로 할당되었습니다"})
		}
		// DB에는 있지만 Keycloak에 없음 — Keycloak 동기화
		log.Printf("[WARN] Platform role in DB but missing in Keycloak for user %s, role %s — syncing", user.KcId, req.RoleName)
		roleExists, err := h.keycloakService.CheckRealmRoleExists(c.Request().Context(), req.RoleName)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("키클로크 역할 확인 실패: %v", err)})
		}
		if !roleExists {
			if err := h.keycloakService.CreateRealmRoleAndWait(c.Request().Context(), req.RoleName); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("키클로크 역할 생성 실패: %v", err)})
			}
		}
		if err := h.keycloakService.AssignRealmRoleToUser(c.Request().Context(), user.KcId, req.RoleName); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("키클로크 역할 할당 실패: %v", err)})
		}
	} else {
		// DB에 역할 할당
		if err := h.roleService.AssignPlatformRole(userID, roleID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("플랫폼 역할 할당 실패: %v", err)})
		}

		// Keycloak role 존재 여부 확인 및 생성
		roleExists, err := h.keycloakService.CheckRealmRoleExists(c.Request().Context(), req.RoleName)
		if err != nil {
			log.Printf("Failed to check realm role existence: %v", err)
			// DB 롤백을 위해 역할 제거
			if rollbackErr := h.roleService.RemovePlatformRole(userID, roleID); rollbackErr != nil {
				log.Printf("Failed to rollback platform role assignment: %v", rollbackErr)
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("키클로크 역할 확인 실패: %v", err)})
		}

		// Keycloak role이 없으면 생성하고 생성 완료까지 대기
		if !roleExists {
			if err := h.keycloakService.CreateRealmRoleAndWait(c.Request().Context(), req.RoleName); err != nil {
				log.Printf("Failed to create realm role: %v", err)
				// DB 롤백을 위해 역할 제거
				if rollbackErr := h.roleService.RemovePlatformRole(userID, roleID); rollbackErr != nil {
					log.Printf("Failed to rollback platform role assignment: %v", rollbackErr)
				}
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("키클로크 역할 생성 실패: %v", err)})
			}
		}

		// Keycloak에 역할 할당
		if err := h.keycloakService.AssignRealmRoleToUser(c.Request().Context(), user.KcId, req.RoleName); err != nil {
			log.Printf("Failed to assign realm role: %v", err)
			// DB 롤백을 위해 역할 제거
			if rollbackErr := h.roleService.RemovePlatformRole(userID, roleID); rollbackErr != nil {
				log.Printf("Failed to rollback platform role assignment: %v", rollbackErr)
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("키클로크 역할 할당 실패: %v", err)})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "플랫폼 역할이 성공적으로 할당되었습니다"})
}

// @Summary Remove platform role
// @Description Remove a platform role from a user
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.AssignRoleRequest true "Platform Role Removal Info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/unassign/platform-role [delete]
// @Id removePlatformRole
func (h *RoleHandler) RemovePlatformRole(c echo.Context) error {
	var req model.AssignRoleRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Error RemovePlatformRole : %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	// 사용자 ID 처리
	var userID uint
	if req.UserID != "" {
		// userId가 있으면 uint로 변환
		userIDInt, err := util.StringToUint(req.UserID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 사용자 ID 형식입니다"})
		}
		userID = userIDInt
	} else if req.Username != "" {
		// userId가 없고 username이 있으면 사용자 조회
		user, err := h.userService.GetUserByUsername(c.Request().Context(), req.Username)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("사용자 조회 실패: %v", err)})
		}
		if user == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 사용자명의 사용자를 찾을 수 없습니다"})
		}
		userID = user.ID
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID 또는 사용자명이 필요합니다"})
	}

	// 역할 ID 처리
	var roleID uint
	if req.RoleID != "" {
		// roleId가 있으면 uint로 변환
		roleIDInt, err := util.StringToUint(req.RoleID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID 형식입니다"})
		}
		roleID = roleIDInt
	} else if req.RoleName != "" {
		// roleId가 없고 roleName이 있으면 역할 조회
		role, err := h.roleService.GetRoleByName(req.RoleName, constants.IAMRoleType(req.RoleType))
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
		}
		if role == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 역할명의 역할을 찾을 수 없습니다"})
		}
		roleID = role.ID
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "역할 ID 또는 역할명이 필요합니다"})
	}

	// 사용자 정보 조회 (Keycloak ID 필요)
	user, err := h.userService.GetUserByID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("사용자 조회 실패: %v", err)})
	}
	if user == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 사용자를 찾을 수 없습니다"})
	}

	// 역할 정보 조회 (RoleName 필요) - Platform 역할로 조회
	role, err := h.roleService.GetRoleByID(roleID, constants.RoleTypePlatform)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
	}
	if role == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 역할을 찾을 수 없습니다"})
	}

	// DB에서 역할 제거
	if err := h.roleService.RemovePlatformRole(userID, roleID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("플랫폼 역할 제거 실패: %v", err)})
	}

	// Keycloak에서 역할 제거 (KC 유저가 없는 경우 404는 무시 — DB는 이미 정리됨)
	if err := h.keycloakService.RemoveRealmRoleFromUser(c.Request().Context(), user.KcId, role.Name); err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "User not found") {
			log.Printf("[WARN] RemovePlatformRole: KC user not found (%s), skipping KC removal", user.KcId)
		} else {
			log.Printf("Failed to remove realm role from user: %v", err)
			// DB 롤백을 위해 역할 재할당
			if rollbackErr := h.roleService.AssignPlatformRole(userID, roleID); rollbackErr != nil {
				log.Printf("Failed to rollback platform role removal: %v", rollbackErr)
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("키클로크 역할 제거 실패: %v", err)})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "플랫폼 역할이 성공적으로 제거되었습니다"})
}

// @Summary Assign workspace role
// @Description Assign a workspace role to a user
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.AssignWorkspaceRoleRequest true "Workspace Role Assignment Info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/assign/workspace-role [post]
// @Id assignWorkspaceRole
func (h *RoleHandler) AssignWorkspaceRole(c echo.Context) error {
	roleType := constants.RoleTypeWorkspace
	var req model.AssignWorkspaceRoleRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Error AssignWorkspaceRole : %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	log.Printf("AssignWorkspaceRoleReq : %v", req)

	// 사용자 ID 처리
	var userID uint
	if req.UserID != "" {
		// userId가 있으면 uint로 변환
		userIDInt, err := util.StringToUint(req.UserID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 사용자 ID 형식입니다"})
		}
		userID = userIDInt
	} else if req.Username != "" {
		// userId가 없고 username이 있으면 사용자 조회
		user, err := h.userService.GetUserByUsername(c.Request().Context(), req.Username)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("사용자 조회 실패: %v", err)})
		}
		if user == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 사용자명의 사용자를 찾을 수 없습니다"})
		}
		userID = user.ID
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID 또는 사용자명이 필요합니다"})
	}

	// 역할 ID 처리
	var roleID uint
	if req.RoleID != "" {
		// roleId가 있으면 uint로 변환
		roleIDInt, err := util.StringToUint(req.RoleID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID 형식입니다"})
		}
		roleID = roleIDInt
	} else if req.RoleName != "" {
		// roleId가 없고 roleName이 있으면 역할 조회
		role, err := h.roleService.GetRoleByName(req.RoleName, roleType)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
		}
		if role == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 역할명의 역할을 찾을 수 없습니다"})
		}
		roleID = role.ID
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "역할 ID 또는 역할명이 필요합니다"})
	}

	// 워크스페이스 ID 처리
	var workspaceID uint
	if req.WorkspaceID != "" {
		// workspaceId가 있으면 uint로 변환
		workspaceIDInt, err := util.StringToUint(req.WorkspaceID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식입니다"})
		}
		workspaceID = workspaceIDInt
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "워크스페이스 ID가 필요합니다"})
	}

	// 역할 할당
	err := h.roleService.AssignWorkspaceRole(userID, workspaceID, roleID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 할당 실패: %v", err)})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "역할이 성공적으로 할당되었습니다"})
}

// @Summary Remove workspace role
// @Description Remove a workspace role from a user
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.AssignRoleRequest true "Workspace Role Removal Info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/unassign/workspace-role [delete]
// @Id removeWorkspaceRole
func (h *RoleHandler) RemoveWorkspaceRole(c echo.Context) error {
	roleType := constants.RoleTypeWorkspace
	var req model.RemoveWorkspaceRoleRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Error RemoveWorkspaceRole : %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	log.Printf("RemoveWorkspaceRoleReq : %v", req)

	// 사용자 ID 처리
	var userID uint
	if req.UserID != "" {
		// userId가 있으면 uint로 변환
		userIDInt, err := util.StringToUint(req.UserID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 사용자 ID 형식입니다"})
		}
		userID = userIDInt
	} else if req.Username != "" {
		// userId가 없고 username이 있으면 사용자 조회
		user, err := h.userService.GetUserByUsername(c.Request().Context(), req.Username)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("사용자 조회 실패: %v", err)})
		}
		if user == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 사용자명의 사용자를 찾을 수 없습니다"})
		}
		userID = user.ID
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID 또는 사용자명이 필요합니다"})
	}

	// 역할 ID 처리
	var roleID uint
	if req.RoleID != "" {
		// roleId가 있으면 uint로 변환
		roleIDInt, err := util.StringToUint(req.RoleID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID 형식입니다"})
		}
		roleID = roleIDInt
	} else if req.RoleName != "" {
		// roleId가 없고 roleName이 있으면 역할 조회
		role, err := h.roleService.GetRoleByName(req.RoleName, roleType)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
		}
		if role == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 역할명의 역할을 찾을 수 없습니다"})
		}
		roleID = role.ID
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "역할 ID 또는 역할명이 필요합니다"})
	}

	// 워크스페이스 ID 처리
	var workspaceID uint
	if req.WorkspaceID != "" {
		// workspaceId가 있으면 uint로 변환
		workspaceIDInt, err := util.StringToUint(req.WorkspaceID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식입니다"})
		}
		workspaceID = workspaceIDInt
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "워크스페이스 ID가 필요합니다"})
	}

	// 역할 제거
	err := h.roleService.RemoveWorkspaceRole(userID, workspaceID, roleID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 제거 실패: %v", err)})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "역할이 성공적으로 제거되었습니다"})
}

// @Summary Create role-CSP role mapping
// @Description Create a new mapping between role and CSP role
// @Tags roles
// @Accept json
// @Produce json
// @Param mapping body model.RoleMasterCspRoleMappingRequest true "Mapping Info"
// @Success 201 {object} model.RoleMasterCspRoleMappingRequest
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/csp-roles [post]
// @Id addCspRoleMappings
func (h *RoleHandler) AddCspRoleMappings(c echo.Context) error {
	var req model.CreateRoleMasterCspRoleMappingRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Master 역할-CSP 역할 매핑 생성 요청 바인딩 실패: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	log.Printf("Master 역할-CSP 역할 매핑 생성 요청 - 역할 ID: %s, CSP 역할 ID: %s", req.RoleID, req.CspRoleID)

	// 입력값 검증
	if err := c.Validate(&req); err != nil {
		log.Printf("Master 역할-CSP 역할 매핑 생성 입력값 검증 실패: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("입력값 검증 실패: %v", err)})
	}

	// 문자열 ID를 uint로 변환
	roleID := req.RoleID
	roleIDInt, err := util.StringToUint(roleID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID 형식입니다"})
	}

	// 문자열 ID를 uint로 변환
	cspRoleIDInt, err := util.StringToUint(req.CspRoleID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID 형식입니다"})
	}

	// 해당 역할이 CSP 역할에 정의 되어 있는지 조회
	// . 해당 역할이 존재하지 않는다면 존재하지 않는 역할이라고 알려준다.
	cspRole, err := h.roleService.GetCspRoleByID(cspRoleIDInt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
	}
	if cspRole == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "해당 역할이 CSP 역할에 정의 되어 있지 않습니다"})
	}

	// 해당 역할이 할당 되어 있는지 확인
	isAssigned, err := h.roleService.IsAssignedRole(0, roleIDInt, constants.RoleTypeCSP)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 할당 확인 실패: %v", err)})
	}
	if !isAssigned {
		log.Printf("역할 할당 안됨 %v", isAssigned)
		// csp 역할 추가
		newCspRole := &model.RoleSub{
			RoleID:   roleIDInt,
			RoleType: constants.RoleTypeCSP,
		}
		err := h.roleService.AddRoleSub(roleIDInt, newCspRole)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 추가 실패: %v", err)})
		}
		cspRoleIDInt = newCspRole.ID
	}
	//RoleMasterCspRoleMappingRequest
	//RoleMasterCspRoleMapping
	// 해당 역할의 CSP 역할 매핑 조회 ::::::: 여기부터 작업하자.
	//CreateRoleMasterCspRoleMappingRequest
	// err = h.roleService.CreateRoleCspRoleMapping(req)
	// if err != nil {
	// 	log.Printf("Master 역할-CSP 역할 매핑 생성 실패: %v", err)
	// 	return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("매핑 생성 실패: %v", err)})
	// }

	// mapping 관계만 추가
	err = h.roleService.AddCspRolesMapping(&req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 할당 실패: %v", err)})
	}

	log.Printf("Master 역할-CSP 역할 매핑 생성 성공 - ID: %d", roleIDInt)
	return c.JSON(http.StatusCreated, map[string]string{"message": "Master 역할-CSP 역할 매핑 생성 성공"})
}

// @Summary Delete workspace role-CSP role mapping
// @Description Delete a mapping between workspace role and CSP role
// @Tags roles
// @Accept json
// @Produce json
// @Param mapping body model.RoleMasterCspRoleMappingRequest true "Mapping Info"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/csp-roles [delete]
// @Id removeCspRoleMappings
func (h *RoleHandler) RemoveCspRoleMappings(c echo.Context) error {

	var req model.RoleMasterCspRoleMappingRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Master 역할-CSP 역할 매핑 삭제 요청 바인딩 실패: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}
	reqAuthMethod := constants.AuthMethod(req.AuthMethod)

	log.Printf("Master 역할-CSP 역할 매핑 삭제 요청 - 역할 ID: %s, CSP 역할 ID: %s",
		req.RoleID, req.CspRoleID)

	// 문자열 ID를 uint로 변환
	roleIDInt, err := util.StringToUint(req.RoleID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식입니다"})
	}

	cspRoleIDInt, err := util.StringToUint(req.CspRoleID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식입니다"})
	}

	// 매핑 삭제
	err = h.roleService.DeleteRoleCspRoleMapping(roleIDInt, cspRoleIDInt, reqAuthMethod)
	if err != nil {
		log.Printf("Master 역할-CSP 역할 매핑 삭제 실패: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("매핑 삭제 실패: %v", err)})
	}

	log.Printf("Master 역할-CSP 역할 매핑 삭제 성공 - ID: %d", roleIDInt)
	return c.NoContent(http.StatusNoContent)
}

// @Summary Get role-CSP role mapping
// @Description Get a mapping between role and CSP role
// @Tags roles
// @Accept json
// @Produce json
// @Param mapping body model.RoleMasterCspRoleMappingRequest true "Mapping Info"
// @Success 200 {object} model.RoleMasterCspRoleMappingRequest
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/csp-roles/list [post]
// @Id listCspRoleMappings
func (h *RoleHandler) ListCspRoleMappings(c echo.Context) error {
	var req model.RoleMasterCspRoleMappingRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("역할-CSP 역할 매핑 조회 요청 바인딩 실패: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	mappings, err := h.roleService.ListRoleCspRoleMappings(&req)
	if err != nil {
		log.Printf(" 역할-CSP 역할 매핑 조회 실패: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("매핑 조회 실패: %v", err)})
	}

	log.Printf("역할-CSP 역할 매핑 조회 성공 - 조회된 매핑 수: %d", len(mappings))
	return c.JSON(http.StatusOK, mappings)
}

// @Summary Get role-CSP role mapping
// @Description Get a mapping between role and CSP role
// @Tags roles
// @Accept json
// @Produce json
// @Param mapping body model.RoleMasterCspRoleMappingRequest true "Mapping Info"
// @Success 200 {object} model.RoleMasterCspRoleMappingRequest
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/csp-roles/id/{roleId} [get]
// @Id getCspRoleMappingByRoleId
func (h *RoleHandler) GetCspRoleMappings(c echo.Context) error {
	roleID := c.Param("roleId")

	if roleID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "역할 ID가 필요합니다"})
	}

	req := model.RoleMasterCspRoleMappingRequest{
		RoleID: roleID,
	}

	mapping, err := h.roleService.GetRoleCspRoleMappings(&req)
	if err != nil {
		log.Printf("역할-CSP 역할 매핑 조회 실패: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("매핑 조회 실패: %v", err)})
	}

	return c.JSON(http.StatusOK, mapping)
}

// @Summary Create multiple csp roles
// @Description Create multiple new csp roles
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.CreateCspRolesRequest true "Multiple CSP Role Creation Info"
// @Success 201 {array} model.CspRole
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/csp/batch [post]
// @Id createCspRoles
func (h *RoleHandler) CreateCspRoles(c echo.Context) error {
	var req model.CreateCspRolesRequest

	if err := c.Bind(&req); err != nil {
		c.Logger().Debugf("Bind error: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	if err := c.Validate(&req); err != nil {
		c.Logger().Debugf("Validate error: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	createdRoles, err := h.cspRoleService.CreateCspRoles(&req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, createdRoles)
}

// Role 에 할당된 사용자 목록 조회
// @Summary List role master mappings
// @Description List role master mappings
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.FilterRoleMasterMappingRequest true "Filter Role Master Mapping Request"
// @Success 200 {array} model.RoleMasterMapping
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/mappings/list [post]
// @Id listRoleMasterMappings
func (h *RoleHandler) ListRoleMasterMappings(c echo.Context) error {

	var req model.FilterRoleMasterMappingRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("역할-사용자 매핑 조회 요청 바인딩 실패: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	// 역할 타입이 없으면 사용자와 연관되는 역할 타입을 조회
	if req.RoleTypes == nil {
		req.RoleTypes = []constants.IAMRoleType{constants.RoleTypePlatform, constants.RoleTypeWorkspace}
	}
	log.Printf("역할-사용자 매핑 조회 요청 - 역할 ID: %s, 역할 타입: %v", req.RoleID, req.RoleTypes)
	mappings, err := h.roleService.ListRoleMasterMappings(&req)
	if err != nil {
		log.Printf(" 역할-사용자 매핑 조회 실패: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("매핑 조회 실패: %v", err)})
	}

	log.Printf("역할-사용자 매핑 조회 성공 - 조회된 매핑 수: %d", len(mappings))
	return c.JSON(http.StatusOK, mappings)
}

// @Summary List users by platform role
// @Description List users by platform role
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.FilterRoleMasterMappingRequest true "Filter Role Master Mapping Request"
// @Success 200 {array} model.RoleMasterMapping
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/mappings/platform-roles/users/list [post]
// @Id listUsersByPlatformRole
func (h *RoleHandler) ListUsersByPlatformRole(c echo.Context) error {
	var req model.FilterRoleMasterMappingRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("역할-사용자 매핑 조회 요청 바인딩 실패: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	// platform 역할 사용자만 조회
	req.RoleTypes = []constants.IAMRoleType{constants.RoleTypePlatform}

	log.Printf("역할-사용자 매핑 조회 요청 - 역할 ID: %s, 역할 타입: %v", req.RoleID, req.RoleTypes)
	mappings, err := h.roleService.ListRoleMasterMappings(&req)
	if err != nil {
		log.Printf(" 역할-사용자 매핑 조회 실패: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("매핑 조회 실패: %v", err)})
	}

	// 사용자가 할당된 역할만 return
	resultRoleMasterMappings := make([]*model.RoleMasterMapping, 0)
	for _, mapping := range mappings {
		if len(mapping.UserPlatformRoles) > 0 {
			resultRoleMasterMappings = append(resultRoleMasterMappings, mapping)
		}
	}

	log.Printf("역할-사용자 매핑 조회 성공 - 조회된 매핑 수: %d", len(resultRoleMasterMappings))
	return c.JSON(http.StatusOK, resultRoleMasterMappings)
}

// @Summary List users by workspace role
// @Description List users by workspace role
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.FilterRoleMasterMappingRequest true "Filter Role Master Mapping Request"
// @Success 200 {array} model.RoleMasterMapping
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/mappings/workspace-roles/users/list [post]
// @Id listUsersByWorkspaceRole
func (h *RoleHandler) ListUsersByWorkspaceRole(c echo.Context) error {
	var req model.FilterRoleMasterMappingRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("역할-사용자 매핑 조회 요청 바인딩 실패: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	// workspace 역할 사용자만 조회
	req.RoleTypes = []constants.IAMRoleType{constants.RoleTypeWorkspace}

	log.Printf("역할-사용자 매핑 조회 요청 - 역할 ID: %s, 역할 타입: %v", req.RoleID, req.RoleTypes)
	mappings, err := h.roleService.ListRoleMasterMappings(&req)
	if err != nil {
		log.Printf(" 역할-사용자 매핑 조회 실패: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("매핑 조회 실패: %v", err)})
	}

	// 사용자가 할당된 역할만 return
	resultRoleMasterMappings := make([]*model.RoleMasterMapping, 0)
	for _, mapping := range mappings {
		if len(mapping.UserWorkspaceRoles) > 0 {
			resultRoleMasterMappings = append(resultRoleMasterMappings, mapping)
		}
	}

	log.Printf("역할-사용자 매핑 조회 성공 - 조회된 매핑 수: %d", len(resultRoleMasterMappings))
	return c.JSON(http.StatusOK, resultRoleMasterMappings)
}

// @Summary List users by csp role
// @Description List users by csp role
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.FilterRoleMasterMappingRequest true "Filter Role Master Mapping Request"
// @Success 200 {array} model.RoleMasterMapping
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/mappings/csp-roles/list [post]
// @Id listUsersByCspRole
func (h *RoleHandler) ListRoleMasterMappingsByCspRole(c echo.Context) error {
	var req model.FilterRoleMasterMappingRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("역할-csp 매핑 조회 요청 바인딩 실패: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	// csp 역할 사용자만 조회
	req.RoleTypes = []constants.IAMRoleType{constants.RoleTypeCSP}

	log.Printf("역할-csp 매핑 조회 요청 - 역할 ID: %s, 역할 타입: %v", req.RoleID, req.RoleTypes)
	mappings, err := h.roleService.ListRoleMasterMappings(&req)
	if err != nil {
		log.Printf(" 역할-csp 매핑 조회 실패: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("매핑 조회 실패: %v", err)})
	}

	// csp역할이이 할당된 역할만 return
	resultRoleMasterMappings := make([]*model.RoleMasterMapping, 0)
	for _, mapping := range mappings {
		if len(mapping.RoleMasterCspRoleMappings) > 0 {
			resultRoleMasterMappings = append(resultRoleMasterMappings, mapping)
		}
	}
	log.Printf("역할-csp 매핑 조회 성공 - 조회된 매핑 수: %d", len(resultRoleMasterMappings))
	return c.JSON(http.StatusOK, resultRoleMasterMappings)
}

// @Summary Get role master mappings
// @Description Get role master mappings
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.FilterRoleMasterMappingRequest true "Filter Role Master Mapping Request"
// @Success 200 {object} model.RoleMasterMapping
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/mappings/role/id/{roleId} [get]
// @Id getRoleMasterMappings
func (h *RoleHandler) GetRoleMasterMappings(c echo.Context) error {
	roleID := c.Param("roleId")

	if roleID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "역할 ID가 필요합니다"})
	}
	var req model.FilterRoleMasterMappingRequest
	// // if err := c.Bind(&req); err != nil {
	// // 	log.Printf("역할 매핑 조회 요청 바인딩 실패: %v", err)
	// // 	return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	// // }

	// if req.RoleID == "" {
	req.RoleID = roleID
	// }

	mappings, err := h.roleService.GetRoleMasterMappings(&req)
	if err != nil {
		log.Printf(" 역할 매핑 조회 실패: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("매핑 조회 실패: %v", err)})
	}

	return c.JSON(http.StatusOK, mappings)
}
