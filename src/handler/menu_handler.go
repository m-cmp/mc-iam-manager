package handler

import (
	"fmt"
	"io" // Ensure io package is imported
	"net/http"
	"strconv"
	"time"

	// "strings" // Removed unused import

	// "github.com/Nerzal/gocloak/v13" // Keep gocloak removed
	// "github.com/golang-jwt/jwt/v5" // jwt import moved to util package
	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/m-cmp/mc-iam-manager/service" // Corrected import path
	"github.com/m-cmp/mc-iam-manager/util"

	// Import the new util package
	"gorm.io/gorm" // Ensure gorm is imported
)

type MenuHandler struct {
	menuService *service.MenuService
	roleService *service.RoleService
	// db *gorm.DB // Not needed directly in handler
}

func NewMenuHandler(db *gorm.DB) *MenuHandler {
	return &MenuHandler{
		menuService: service.NewMenuService(db),
		roleService: service.NewRoleService(db),
	}
}

// Helper function moved to util package

// ListUserMenuTree godoc
// @Summary Get current user's menu tree
// @Description Get the menu tree accessible to the current user's platform role.
// @Tags menus
// @Accept json
// @Produce json
// @Success 200 {array} model.MenuTreeNode
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /api/users/menus-tree/list [post]
// @Id listUserMenuTree
func (h *MenuHandler) ListUserMenuTree(c echo.Context) error {
	platformRolesIntf := c.Get("platformRoles")
	if platformRolesIntf == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized: Platform roles not found"})
	}

	platformRoles, ok := platformRolesIntf.([]string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to process platform roles"})
	}
	c.Logger().Debug("GetUserMenuTree: platformRoles %s", platformRoles)

	// Convert platform role strings to uint IDs
	platformRoleNames := make([]uint, 0, len(platformRoles))
	for _, roleName := range platformRoles {
		role, err := h.roleService.GetRoleByName(roleName, constants.RoleTypePlatform)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("failed to find role: %v", err)})
		}
		if role == nil {
			continue // Skip if role not found
		}
		platformRoleNames = append(platformRoleNames, role.ID)
	}

	// Call the service method with platform role IDs
	menuTree, err := h.menuService.BuildUserMenuTree(c.Request().Context(), platformRoleNames)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to retrieve menu tree: %v", err),
		})
	}

	return c.JSON(http.StatusOK, menuTree)
}

// @Summary Get current user's menu list
// @Description Get the menu list accessible to the current user's platform role.
// @Tags menus
// @Accept json
// @Produce json
// @Success 200 {array} model.Menu
// @Router /api/users/menus/list [post]
// @Id listUserMenu
func (h *MenuHandler) ListUserMenu(c echo.Context) error {
	// platformRolesIntf := c.Get("platformRoles")
	// if platformRolesIntf == nil {
	// 	return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized: Platform roles not found"})
	// }

	// platformRoles, ok := platformRolesIntf.([]string)
	// if !ok {
	// 	return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to process platform roles"})
	// }
	// c.Logger().Debug("GetUserMenuTree: platformRoles %s", platformRoles)

	// // Convert platform role strings to uint IDs
	// platformRoleNames := make([]uint, 0, len(platformRoles))
	// for _, roleName := range platformRoles {
	// 	role, err := h.roleService.GetRoleByName(roleName, model.RoleTypePlatform)
	// 	if err != nil {
	// 		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("failed to find role: %v", err)})
	// 	}
	// 	if role == nil {
	// 		continue // Skip if role not found
	// 	}
	// 	platformRoleNames = append(platformRoleNames, role.ID)
	// }

	// Call the service method with platform role IDs
	req := &model.MenuMappingFilterRequest{}
	if err := c.Bind(req); err != nil {
		c.Logger().Debug("ListUserMenu err %s", err)
	}

	c.Logger().Debug("ListUserMenu: platformRoles %s", c.Get("platformRoles"))
	userPlatformRoles, ok := c.Get("platformRoles").([]string)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "Unable to retrieve platform roles")
	}
	c.Logger().Debug("getMenus: userPlatformRoles %s", userPlatformRoles)
	// platformRole is string, so query id from db
	platformRoleIDs := make([]string, 0, len(userPlatformRoles))
	for _, roleName := range userPlatformRoles {
		role, err := h.roleService.GetRoleByName(roleName, constants.RoleTypePlatform)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("failed to find role: %v", err)})
		}
		// Check if role is nil before accessing its ID
		if role == nil {
			c.Logger().Debug("Role not found for roleName: %s", roleName)
			continue // Skip this role and continue with others
		}
		platformRoleIDs = append(platformRoleIDs, strconv.FormatUint(uint64(role.ID), 10))
	}
	req.RoleIDs = platformRoleIDs

	menuList, err := h.menuService.MenuList(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to retrieve user menu: %v", err),
		})
	}

	return c.JSON(http.StatusOK, menuList)

}

// ListAllMenusTree godoc
// @Summary List all menus
// @Description List all menus as a tree structure. Admin permission required.
// @Tags menus
// @Accept json
// @Produce json
// @Success 200 {array} model.MenuTreeNode
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /api/menus/list [post]
// @Id listMenus
func (h *MenuHandler) ListMenus(c echo.Context) error {
	req := &model.MenuFilterRequest{}
	if err := c.Bind(req); err != nil {
		c.Logger().Debug("ListMenus err %s", err)
	}
	// 3. Retrieve menu tree
	menus, err := h.menuService.ListAllMenus(req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get menu list")
	}

	// If no menus exist
	if len(menus) == 0 {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "No menus exist. Please register menus.",
			"menus":   []interface{}{},
		})
	}

	// // 4. If not platformAdmin, filter menus by permissions
	// if !isPlatformAdmin {
	// 	// Filter menus by permissions
	// 	filteredMenus := h.filterMenusByPermission(menus, userRoles)
	// 	return c.JSON(http.StatusOK, filteredMenus)
	// }

	// Return all menus for platformAdmin
	return c.JSON(http.StatusOK, menus)
}

// @Summary List all menus Tree
// @Description List all menus as a tree structure. Admin permission required.
// @Tags menus
// @Accept json
// @Produce json
// @Success 200 {array} model.MenuTreeNode
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /api/menus/tree/list [post]
// @Id listMenusTree
func (h *MenuHandler) ListMenusTree(c echo.Context) error {
	// If this is an admin-only function, let middleware handle the check.

	// // 1. Get platformRoles from context
	// platformRolesInterface := c.Get("platformRoles")
	// if platformRolesInterface == nil {
	// 	c.Logger().Debug("GetAllMenusTree: platformRoles not found in context")
	// 	return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token claims")
	// }

	// userRoles, ok := platformRolesInterface.([]string)
	// if !ok {
	// 	c.Logger().Debugf("GetAllMenusTree: platformRoles type assertion failed: %T", platformRolesInterface)
	// 	return echo.NewHTTPError(http.StatusInternalServerError, "Invalid platform roles format")
	// }

	// c.Logger().Debugf("GetAllMenusTree: Found platformRoles in context: %v", userRoles)

	// // 2. Check platformAdmin role
	// isPlatformAdmin := false
	// for _, role := range userRoles {
	// 	if role == "platformAdmin" {
	// 		isPlatformAdmin = true
	// 		break
	// 	}
	// }

	// 3. Retrieve menu tree
	req := &model.MenuFilterRequest{}
	if err := c.Bind(req); err != nil {
		c.Logger().Debug("ListMenusTree err %s", err)
	}
	menus, err := h.menuService.GetAllMenusTree(req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get menu tree")
	}

	// If no menus exist
	if len(menus) == 0 {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "No menus exist. Please register menus.",
			"menus":   []interface{}{},
		})
	}

	// // 4. If not platformAdmin, filter menus by permissions
	// if !isPlatformAdmin {
	// 	// Filter menus by permissions
	// 	filteredMenus := h.filterMenusByPermission(menus, userRoles)
	// 	return c.JSON(http.StatusOK, filteredMenus)
	// }

	// Return all menus for platformAdmin
	return c.JSON(http.StatusOK, menus)
}

// filterMenusByPermission filters menus based on user roles.
func (h *MenuHandler) filterMenusByPermission(menus []*model.MenuTreeNode, userRoles []string) []*model.MenuTreeNode {
	var filteredMenus []*model.MenuTreeNode

	for _, menu := range menus {
		// If menu has no permission requirement or user has the required permission
		if menu.ResType == "" || h.hasPermission(userRoles, menu.ResType) {
			// If submenus exist, filter recursively
			if len(menu.Children) > 0 {
				menu.Children = h.filterMenusByPermission(menu.Children, userRoles)
			}
			filteredMenus = append(filteredMenus, menu)
		}
	}

	return filteredMenus
}

// hasPermission checks if user has specific permission.
func (h *MenuHandler) hasPermission(userRoles []string, requiredRole string) bool {
	for _, role := range userRoles {
		// platformAdmin has all permissions
		if role == "platformAdmin" {
			return true
		}
		// If role matches the required permission
		if role == requiredRole {
			return true
		}
	}
	return false
}

// GetByID godoc
// @Summary Get menu by ID
// @Description Get menu details by ID
// @Tags menus
// @Accept json
// @Produce json
// @Param menuId path string true "Menu ID"
// @Success 200 {object} model.Menu
// @Security BearerAuth
// @Router /api/menus/id/{menuId} [post]
// @Id getMenuByID
func (h *MenuHandler) GetMenuByID(c echo.Context) error {
	id := c.Param("menuId")
	menu, err := h.menuService.GetMenuByID(&id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to find menu",
		})
	}
	if menu == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Menu not found",
		})
	}
	return c.JSON(http.StatusOK, menu)
}

// Create godoc
// @Summary Create new menu
// @Description Create a new menu
// @Tags menus
// @Accept json
// @Produce json
// @Param menu body model.Menu true "Menu Info"
// @Success 201 {object} model.Menu
// @Security BearerAuth
// @Router /api/menus [post]
// @Id createMenu
func (h *MenuHandler) CreateMenu(c echo.Context) error {
	req := new(model.CreateMenuRequest)
	if err := c.Bind(req); err != nil {
		c.Logger().Debugf("CreateMenu err %s", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// menuId로 기존 메뉴 조회
	existingMenu, err := h.menuService.GetMenuByID(&req.ID)
	if err != nil {
		c.Logger().Debugf("Failed to check existing menu: %v", err)
		if err == repository.ErrMenuNotFound {
			// Menu doesn't exist.
		} else {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Error occurred while retrieving menu",
			})
		}
	}

	if existingMenu != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Menu already exists",
		})
		// // 기존 메뉴가 있으면 업데이트
		// updates := map[string]interface{}{
		// 	"parent_id":    req.ParentID,
		// 	"display_name": req.DisplayName,
		// 	"res_type":     req.ResType,
		// 	"is_action":    req.IsAction,
		// }

		// // Priority와 MenuNumber는 문자열에서 변환 필요
		// if req.Priority != "" {
		// 	priorityInt, err := util.StringToUint(req.Priority)
		// 	if err != nil {
		// 		return c.JSON(http.StatusBadRequest, map[string]string{
		// 			"error": "잘못된 priority 값입니다",
		// 		})
		// 	}
		// 	updates["priority"] = priorityInt
		// }

		// if req.MenuNumber != "" {
		// 	menuNumberInt, err := util.StringToUint(req.MenuNumber)
		// 	if err != nil {
		// 		return c.JSON(http.StatusBadRequest, map[string]string{
		// 			"error": "잘못된 menu number 값입니다",
		// 		})
		// 	}
		// 	updates["menu_number"] = menuNumberInt
		// }

		// if err := h.menuService.Update(req.ID, updates); err != nil {
		// 	c.Logger().Debugf("Menu update err %s", err)
		// 	return c.JSON(http.StatusInternalServerError, map[string]string{
		// 		"error": "메뉴 업데이트에 실패했습니다",
		// 	})
		// }

		// return c.JSON(http.StatusOK, map[string]string{"message": "메뉴 업데이트에 성공했습니다"})
	} else {
		// 기존 메뉴가 없으면 생성
		if err := h.menuService.Create(req); err != nil {
			c.Logger().Debugf("CreateMenu err %s", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "메뉴 생성에 실패했습니다",
			})
		}

		return c.JSON(http.StatusCreated, map[string]string{"message": "메뉴 생성에 성공했습니다"})
	}
}

// Update godoc
// @Summary Update menu information
// @Description Update menu information
// @Tags menus
// @Accept json
// @Produce json
// @Param id path string true "Menu ID"
// @Param menu body model.Menu true "Menu Info"
// @Success 200 {object} model.Menu
// @Security BearerAuth
// @Router /api/menus/id/{menuId} [put]
// @Id updateMenu
func (h *MenuHandler) UpdateMenu(c echo.Context) error {
	id := c.Param("menuId")
	var menu model.CreateMenuRequest

	// Bind the request body to the Menu struct
	if err := c.Bind(&menu); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("잘못된 요청 형식입니다: %v", err),
		})
	}

	// Prevent updating the ID via the request body
	menu.ID = ""

	// Convert struct to map for partial updates
	updates := make(map[string]interface{})
	if menu.DisplayName != "" {
		updates["display_name"] = menu.DisplayName
	}
	if menu.ParentID != "" {
		updates["parent_id"] = menu.ParentID
	}
	if menu.ResType != "" {
		updates["res_type"] = menu.ResType
	}
	updates["is_action"] = menu.IsAction
	if menu.Priority != "" {
		priorityInt, err := util.StringToUint(menu.Priority)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "잘못된 priority 값입니다",
			})
		}
		updates["priority"] = priorityInt
	}
	if menu.MenuNumber != "" {
		menuNumberInt, err := util.StringToUint(menu.MenuNumber)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "잘못된 menu number 값입니다",
			})
		}
		updates["menu_number"] = menuNumberInt
	}

	if len(updates) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "업데이트할 필드가 없습니다",
		})
	}

	// Call the service method with id and the map of updates
	if err := h.menuService.Update(id, updates); err != nil {
		// Handle specific errors like "not found" if needed
		if err.Error() == "menu not found" { // Assuming service/repo returns this specific error string
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("메뉴 업데이트 실패: %v", err),
		})
	}

	// Optionally, fetch the updated menu and return it
	updatedMenu, err := h.menuService.GetMenuByID(&id)
	if err != nil {
		// Log error but return success as update itself was successful
		fmt.Printf("Warning: Failed to fetch updated menu (id: %s): %v\n", id, err)
		return c.JSON(http.StatusOK, updates) // Return the updates map as confirmation
	}
	if updatedMenu == nil {
		// Should not happen if update was successful, but handle defensively
		return c.JSON(http.StatusNotFound, map[string]string{"error": "업데이트 후 메뉴를 찾을 수 없습니다"})
	}

	return c.JSON(http.StatusOK, updatedMenu) // Return the full updated menu
}

// Delete godoc
// @Summary Delete menu
// @Description Delete a menu
// @Tags menus
// @Accept json
// @Produce json
// @Param id path string true "Menu ID"
// @Success 204 "No Content"
// @Security BearerAuth
// @Router /api/menus/id/{menuId} [delete]
// @Id deleteMenu
func (h *MenuHandler) DeleteMenu(c echo.Context) error {
	id := c.Param("menuId")
	if err := h.menuService.Delete(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "메뉴 삭제에 실패했습니다",
		})
	}
	return c.NoContent(http.StatusNoContent)
}

// RegisterMenusFromYAML godoc
// @Summary Register/Update menus from YAML file or URL
// @Description Register or update menus from a local YAML file specified by the filePath query parameter, or from the MCWEBCONSOLE_MENUYAML URL in .env if not provided. If loaded from URL, the file is saved to asset/menu/menu.yaml.
// @Tags menus
// @Accept json
// @Produce json
// @Param filePath query string false "YAML file path (optional, uses .env URL or default local path if not provided)"
// @Success 200 {object} map[string]string "message: Successfully registered menus from YAML"
// @Failure 500 {object} map[string]string "error: 실패 메시지"
// @Security BearerAuth
// @Router /api/menus/setup/initial-menus [post]
// @Id registerMenusFromYAML
func (h *MenuHandler) RegisterMenusFromYAML(c echo.Context) error {
	filePath := c.QueryParam("filePath") // 쿼리 파라미터로 파일 경로 받기 (선택 사항)

	if err := h.menuService.LoadAndRegisterMenusFromYAML(filePath); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("메뉴 YAML 등록 실패: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Successfully registered menus from YAML",
	})
}

// RegisterMenusFromBody godoc
// @Summary Register/Update menus from YAML in request body
// @Description Parse YAML text in the request body and register or update menus in the database. Recommended Content-Type: text/plain, text/yaml, application/yaml.
// @Tags menus
// @Accept plain
// @Produce json
// @Param yaml body string true "Menu definitions in YAML format (must contain 'menus:' root key)" example("menus:\n  - id: new-item\n    parentid: dashboard\n    displayname: New Menu Item\n    restype: menu\n    isaction: false\n    priority: 10\n    menunumber: 9999")
// @Success 200 {object} map[string]string "message: Successfully registered menus from request body"
// @Failure 400 {object} map[string]string "error: 잘못된 요청 본문 또는 YAML 형식 오류"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /api/menus/setup/initial-menus2 [post]
// @Id registerMenusFromBody
func (h *MenuHandler) RegisterMenusFromBody(c echo.Context) error {
	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("요청 본문을 읽는데 실패했습니다: %v", err),
		})
	}
	defer c.Request().Body.Close()

	if len(bodyBytes) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "요청 본문이 비어있습니다",
		})
	}

	if err := h.menuService.RegisterMenusFromContent(bodyBytes); err != nil {
		// Differentiate between bad request (parsing error) and server error (db error)
		// Note: The service currently returns a generic error for unmarshalling.
		// Consider refining error types in service/repo for better error handling here.
		if err.Error()[:len("error unmarshalling")] == "error unmarshalling" { // Basic check
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": fmt.Sprintf("YAML 파싱 오류: %v", err),
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("메뉴 등록 실패: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Successfully registered menus from request body",
	})
}

// ListMappedMenusByRole godoc
// @Summary List menus mapped to platform role
// @Description List menus mapped to a specific platform role.
// @Tags menus
// @Accept json
// @Produce json
// @Param roleId query string false "Platform Role ID"
// @Param menuId query string false "Menu ID"
// @Success 200 {array} model.Menu
// @Failure 400 {object} map[string]string "error: platform role is required"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /api/menus/platform-roles/list [post]
// @Id listMappedMenusByRole
func (h *MenuHandler) ListMenusRolesMapping(c echo.Context) error {
	req := &model.MenuMappingFilterRequest{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	menus, err := h.menuService.ListMappedMenusByRole(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, menus)
}

// CreateMenuMapping godoc
// @Summary Create menu mapping
// @Description Create a new menu mapping
// @Tags menu
// @Accept json
// @Produce json
// @Param mapping body model.CreateMenuMappingRequest true "Menu Mapping"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/menus/platform-roles [post]
// @Id createMenusRolesMapping
func (h *MenuHandler) CreateMenusRolesMapping(c echo.Context) error {
	var req model.CreateMenuMappingRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	var mappings []*model.RoleMenuMapping
	for _, menuID := range req.MenuIDs {
		roleIDInt, err := util.StringToUint(req.RoleID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid role ID"})
		}

		mapping := &model.RoleMenuMapping{
			RoleID:    roleIDInt,
			MenuID:    menuID,
			CreatedAt: time.Now(),
		}
		mappings = append(mappings, mapping)
	}
	if err := h.menuService.CreateRoleMenuMappings(mappings); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("메뉴 매핑 생성 실패: %v", err)})
	}
	return c.JSON(http.StatusCreated, map[string]string{"message": "Menu mapping created successfully"})
}

// DeleteMenuMapping godoc
// @Summary Delete platform role-menu mapping
// @Description Delete the mapping between a platform role and a menu.
// @Tags menus
// @Accept json
// @Produce json
// @Param roleId query string false "Platform Role ID"
// @Param menuId query string false "Menu ID"
// @Success 200 {object} map[string]string "message: Menu mapping deleted successfully"
// @Failure 400 {object} map[string]string "error: platform role and menu ID are required"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /api/menus/platform-roles [delete]
// @Id deleteMenusRolesMapping
func (h *MenuHandler) DeleteMenusRolesMapping(c echo.Context) error {
	roleID := c.Param("roleId")
	menuID := c.Param("menuId")

	roleIDInt, err := util.StringToUint(roleID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid role ID"})
	}

	mappings := []*model.RoleMenuMapping{
		{
			RoleID: roleIDInt,
			MenuID: menuID,
		},
	}
	err = h.menuService.DeleteRoleMenuMapping(mappings)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Menu mapping deleted successfully"})
}

// @Summary Get user menu tree by platform roles
// @Description Get menu tree based on user's platform roles
// @Tags menus
// @Accept json
// @Produce json
// @Success 200 {array} model.MenuTreeNode
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/menus/user-menu-tree [get]
// @Id getUserMenuTree
func (h *MenuHandler) GetUserMenuTree(c echo.Context) error {
	// Get platform roles from context (set by auth middleware)
	platformRoles := c.Get("platform_roles").([]string)
	if len(platformRoles) == 0 {
		return c.JSON(http.StatusOK, []*model.MenuTreeNode{})
	}

	// Convert platform role strings to uint IDs
	platformRoleIDs := make([]uint, 0, len(platformRoles))
	for _, roleName := range platformRoles {
		role, err := h.roleService.GetRoleByName(roleName, constants.RoleTypePlatform)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("failed to find role: %v", err)})
		}
		if role == nil {
			continue // Skip if role not found
		}
		platformRoleIDs = append(platformRoleIDs, role.ID)
	}

	menuTree, err := h.menuService.BuildUserMenuTree(c.Request().Context(), platformRoleIDs)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, menuTree)
}
