package handler

import (
	"fmt"
	"io" // Ensure io package is imported
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
)

type MenuHandler struct {
	menuService *service.MenuService
}

func NewMenuHandler(menuService *service.MenuService) *MenuHandler {
	return &MenuHandler{menuService: menuService}
}

// GetMenus godoc
// @Summary 모든 메뉴 조회
// @Description 모든 메뉴 목록을 조회합니다
// @Tags menus
// @Accept json
// @Produce json
// @Success 200 {array} model.Menu
// @Security BearerAuth
// @Router /menus [get]
func (h *MenuHandler) GetMenus(c echo.Context) error {
	menus, err := h.menuService.GetMenus()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "메뉴 목록을 가져오는데 실패했습니다",
		})
	}
	return c.JSON(http.StatusOK, menus)
}

// GetByID godoc
// @Summary 메뉴 ID로 조회
// @Description 특정 메뉴를 ID로 조회합니다
// @Tags menus
// @Accept json
// @Produce json
// @Param id path string true "Menu ID"
// @Success 200 {object} model.Menu
// @Security BearerAuth
// @Router /menus/{id} [get]
func (h *MenuHandler) GetByID(c echo.Context) error {
	id := c.Param("id")
	menu, err := h.menuService.GetByID(id)
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
func (h *MenuHandler) Create(c echo.Context) error {
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
// @Router /menus/{id} [put]
func (h *MenuHandler) Update(c echo.Context) error {
	id := c.Param("id")
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
	updatedMenu, err := h.menuService.GetByID(id)
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
// @Router /menus/{id} [delete]
func (h *MenuHandler) Delete(c echo.Context) error {
	id := c.Param("id")
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
// @Router /menus/register-from-yaml [post]
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
// @Router /menus/register-from-body [post]
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
