package handler

import (
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
// @Router /menus/{id} [put]
func (h *MenuHandler) Update(c echo.Context) error {
	id := c.Param("id")
	menu := new(model.Menu)
	if err := c.Bind(menu); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "잘못된 요청 형식입니다",
		})
	}

	menu.ID = id
	if err := h.menuService.Update(menu); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "메뉴 업데이트에 실패했습니다",
		})
	}

	return c.JSON(http.StatusOK, menu)
}

// Delete godoc
// @Summary 메뉴 삭제
// @Description 메뉴를 삭제합니다
// @Tags menus
// @Accept json
// @Produce json
// @Param id path string true "Menu ID"
// @Success 204 "No Content"
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
