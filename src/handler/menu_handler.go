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
	"github.com/m-cmp/mc-iam-manager/model"
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
// @Summary 현재 사용자의 메뉴 트리 조회
// @Description 현재 로그인한 사용자의 Platform Role에 따라 접근 가능한 메뉴 목록을 트리 구조로 조회합니다.
// @Tags menus
// @Accept json
// @Produce json
// @Success 200 {array} model.MenuTreeNode
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /users/menus-tree/list [post]
// @OperationId listUserMenuTree
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
		role, err := h.roleService.GetRoleByName(roleName, model.RoleTypePlatform)
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
			"error": fmt.Sprintf("메뉴 트리 조회 실패: %v", err),
		})
	}

	return c.JSON(http.StatusOK, menuTree)
}

// @Summary 현재 사용자의 메뉴 목록 조회
// @Description 현재 로그인한 사용자의 Platform Role에 따라 접근 가능한 메뉴 목록을 조회합니다.
// @Tags menus
// @Accept json
// @Produce json
// @Success 200 {array} model.Menu
// @Router /api/users/menus/list [post]
// @OperationId listUserMenu
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
	menuList, err := h.menuService.MenuList(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("사용자 메뉴 조회 실패: %v", err),
		})
	}

	return c.JSON(http.StatusOK, menuList)

}

// ListAllMenusTree godoc
// @Summary 모든 메뉴 트리 조회 (관리자용)
// @Description 모든 메뉴 목록을 트리 구조로 조회합니다. 관리자 권한이 필요합니다.
// @Tags menus
// @Accept json
// @Produce json
// @Success 200 {array} model.MenuTreeNode
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /api/menus/list [post]
// @OperationId listMenus
func (h *MenuHandler) ListMenus(c echo.Context) error {

	// 3. 메뉴 트리 조회
	menus, err := h.menuService.ListAllMenus()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get menu list")
	}

	// 메뉴가 없는 경우
	if len(menus) == 0 {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "메뉴가 없습니다. 메뉴를 등록하세요.",
			"menus":   []interface{}{},
		})
	}

	// // 4. platformAdmin이 아닌 경우 권한에 따라 메뉴 필터링
	// if !isPlatformAdmin {
	// 	// 권한에 따라 메뉴 필터링
	// 	filteredMenus := h.filterMenusByPermission(menus, userRoles)
	// 	return c.JSON(http.StatusOK, filteredMenus)
	// }

	// platformAdmin인 경우 모든 메뉴 반환
	return c.JSON(http.StatusOK, menus)
}

// @Summary 모든 메뉴 트리 조회 (관리자용)
// @Description 모든 메뉴 목록을 트리 구조로 조회합니다. 관리자 권한이 필요합니다.
// @Tags menus
// @Accept json
// @Produce json
// @Success 200 {array} model.MenuTreeNode
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /api/menus/list [post]
// @OperationId listMenusTree
func (h *MenuHandler) ListMenusTree(c echo.Context) error {
	// 관리자 전용기능이면 middleware 에서 체크하도록 하자.

	// // 1. 컨텍스트에서 platformRoles 가져오기
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

	// // 2. platformAdmin 역할 확인
	// isPlatformAdmin := false
	// for _, role := range userRoles {
	// 	if role == "platformAdmin" {
	// 		isPlatformAdmin = true
	// 		break
	// 	}
	// }

	// 3. 메뉴 트리 조회
	menus, err := h.menuService.GetAllMenusTree()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get menu tree")
	}

	// 메뉴가 없는 경우
	if len(menus) == 0 {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "메뉴가 없습니다. 메뉴를 등록하세요.",
			"menus":   []interface{}{},
		})
	}

	// // 4. platformAdmin이 아닌 경우 권한에 따라 메뉴 필터링
	// if !isPlatformAdmin {
	// 	// 권한에 따라 메뉴 필터링
	// 	filteredMenus := h.filterMenusByPermission(menus, userRoles)
	// 	return c.JSON(http.StatusOK, filteredMenus)
	// }

	// platformAdmin인 경우 모든 메뉴 반환
	return c.JSON(http.StatusOK, menus)
}

// filterMenusByPermission 사용자의 역할에 따라 메뉴를 필터링합니다.
func (h *MenuHandler) filterMenusByPermission(menus []*model.MenuTreeNode, userRoles []string) []*model.MenuTreeNode {
	var filteredMenus []*model.MenuTreeNode

	for _, menu := range menus {
		// 메뉴의 권한이 없거나 사용자가 해당 권한을 가지고 있는 경우
		if menu.ResType == "" || h.hasPermission(userRoles, menu.ResType) {
			// 하위 메뉴가 있는 경우 재귀적으로 필터링
			if len(menu.Children) > 0 {
				menu.Children = h.filterMenusByPermission(menu.Children, userRoles)
			}
			filteredMenus = append(filteredMenus, menu)
		}
	}

	return filteredMenus
}

// hasPermission 사용자가 특정 권한을 가지고 있는지 확인합니다.
func (h *MenuHandler) hasPermission(userRoles []string, requiredRole string) bool {
	for _, role := range userRoles {
		// platformAdmin은 모든 권한을 가짐
		if role == "platformAdmin" {
			return true
		}
		// 역할이 권한과 일치하는 경우
		if role == requiredRole {
			return true
		}
	}
	return false
}

// GetByID godoc
// @Summary 메뉴 ID로 조회
// @Description 특정 메뉴를 ID로 조회합니다
// @Tags menus
// @Accept json
// @Produce json
// @Param menuId path string true "Menu ID"
// @Success 200 {object} model.Menu
// @Security BearerAuth
// @Router /menus/id/{menuId} [post]
// @OperationId getMenuByID
func (h *MenuHandler) GetMenuByID(c echo.Context) error {
	id := c.Param("menuId")
	menu, err := h.menuService.GetMenuByID(&id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "메뉴를 찾는데 실패했습니다",
		})
	}
	if menu == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "메뉴를 찾을 수 없습니다",
		})
	}
	return c.JSON(http.StatusOK, menu)
}

// Create godoc
// @Summary 새 메뉴 생성
// @Description 새로운 메뉴를 생성합니다
// @Tags menus
// @Accept json
// @Produce json
// @Param menu body model.Menu true "Menu Info"
// @Success 201 {object} model.Menu
// @Security BearerAuth
// @Router /menus [post]
// @OperationId createMenu
func (h *MenuHandler) CreateMenu(c echo.Context) error {
	menu := new(model.Menu)
	if err := c.Bind(menu); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "잘못된 요청 형식입니다",
		})
	}

	if err := h.menuService.Create(menu); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "메뉴 생성에 실패했습니다",
		})
	}

	return c.JSON(http.StatusCreated, menu)
}

// Update godoc
// @Summary 메뉴 정보 업데이트
// @Description 메뉴 정보를 업데이트합니다
// @Tags menus
// @Accept json
// @Produce json
// @Param id path string true "Menu ID"
// @Param menu body model.Menu true "Menu Info"
// @Success 200 {object} model.Menu
// @Security BearerAuth
// @Router /menus/id/{menuId} [put]
// @OperationId updateMenu
func (h *MenuHandler) UpdateMenu(c echo.Context) error {
	id := c.Param("menuId")
	updates := make(map[string]interface{}) // Bind to a map

	// Bind the request body to the map
	// This automatically handles JSON unmarshalling into the map
	if err := c.Bind(&updates); err != nil {
		// Check for specific binding errors if needed
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("잘못된 요청 형식입니다: %v", err),
		})
	}

	// Prevent updating the ID via the request body
	delete(updates, "id")

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
// @Summary 메뉴 삭제
// @Description 메뉴를 삭제합니다
// @Tags menus
// @Accept json
// @Produce json
// @Param id path string true "Menu ID"
// @Success 204 "No Content"
// @Security BearerAuth
// @Router /menus/id/{menuId} [delete]
// @OperationId deleteMenu
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
// @Summary YAML 파일 또는 URL에서 메뉴 등록/업데이트
// @Description filePath 쿼리 파라미터로 지정된 로컬 YAML 파일 또는 파라미터가 없을 경우 .env 파일의 MCWEBCONSOLE_MENUYAML URL에서 메뉴를 가져와 데이터베이스에 등록/업데이트합니다. URL에서 가져올 경우 asset/menu/menu.yaml에 저장됩니다.
// @Tags menus
// @Accept json
// @Produce json
// @Param filePath query string false "YAML 파일 경로 (선택 사항, 없으면 .env의 URL 또는 기본 로컬 경로 사용)"
// @Success 200 {object} map[string]string "message: Successfully registered menus from YAML"
// @Failure 500 {object} map[string]string "error: 실패 메시지"
// @Security BearerAuth
// @Router /api/menus/setup/initial-menu [post]
// @OperationId registerMenusFromYAML
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
// @Summary 요청 본문의 YAML 내용으로 메뉴 등록/업데이트
// @Description 요청 본문에 포함된 YAML 텍스트를 파싱하여 메뉴를 데이터베이스에 등록하거나 업데이트합니다. Content-Type은 text/plain, text/yaml, application/yaml 등을 권장합니다.
// @Tags menus
// @Accept plain
// @Produce json
// @Param yaml body string true "Menu definitions in YAML format (must contain 'menus:' root key)" example("menus:\n  - id: new-item\n    parentid: dashboard\n    displayname: New Menu Item\n    restype: menu\n    isaction: false\n    priority: 10\n    menunumber: 9999")
// @Success 200 {object} map[string]string "message: Successfully registered menus from request body"
// @Failure 400 {object} map[string]string "error: 잘못된 요청 본문 또는 YAML 형식 오류"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /api/menus/setup/initial-menu2 [post]
// @OperationId registerMenusFromBody
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
// @Summary 플랫폼 역할에 매핑된 메뉴 목록 조회
// @Description 특정 플랫폼 역할에 매핑된 메뉴 목록을 조회합니다.
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
// @OperationId listMappedMenusByRole
func (h *MenuHandler) ListMappedMenusByRole(c echo.Context) error {
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
// @Success 201 {object} model.MenuMapping
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/menus/platform-roles [post]
// @OperationId createMenuMapping
func (h *MenuHandler) CreateMenuMapping(c echo.Context) error {
	var req model.CreateMenuMappingRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	for _, menuID := range req.MenuID {
		roleIDInt, err := util.StringToUint(req.RoleID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid role ID"})
		}
		menuIDInt, err := util.StringToUint(menuID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid menu ID"})
		}

		mapping := &model.MenuMapping{
			RoleID:    roleIDInt,
			MenuID:    menuIDInt,
			CreatedAt: time.Now(),
		}

		if err := h.menuService.CreateMenuMapping(mapping); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("메뉴 매핑 생성 실패: %v", err)})
		}
	}

	return c.JSON(http.StatusCreated, map[string]string{"message": "Menu mapping created successfully"})
}

// DeleteMenuMapping godoc
// @Summary 플랫폼 역할-메뉴 매핑 삭제
// @Description 플랫폼 역할과 메뉴 간의 매핑을 삭제합니다.
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
// @OperationId deleteMenuMapping
func (h *MenuHandler) DeleteMenuMapping(c echo.Context) error {
	platformRoleID, err := strconv.ParseUint(c.Param("role"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid platform role ID"})
	}
	menuID := c.Param("menuId")

	err = h.menuService.DeleteMenuMapping(uint(platformRoleID), menuID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Menu mapping deleted successfully"})
}

// GetUserMenuTree 사용자의 플랫폼 역할에 따른 메뉴 트리 조회
func (h *MenuHandler) GetUserMenuTree(c echo.Context) error {
	// Get platform roles from context (set by auth middleware)
	platformRoles := c.Get("platform_roles").([]string)
	if len(platformRoles) == 0 {
		return c.JSON(http.StatusOK, []*model.MenuTreeNode{})
	}

	// Convert platform role strings to uint IDs
	platformRoleIDs := make([]uint, 0, len(platformRoles))
	for _, roleName := range platformRoles {
		role, err := h.roleService.GetRoleByName(roleName, model.RoleTypePlatform)
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
