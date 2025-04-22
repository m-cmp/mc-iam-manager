package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm" // Import gorm
)

type PlatformRoleHandler struct {
	service *service.PlatformRoleService
	// db *gorm.DB // Not needed directly in handler
}

func NewPlatformRoleHandler(db *gorm.DB) *PlatformRoleHandler { // Accept db, remove service param
	// Initialize service internally
	platformRoleService := service.NewPlatformRoleService(db)
	return &PlatformRoleHandler{
		service: platformRoleService,
	}
}

// List godoc
// @Summary 플랫폼 역할 목록 조회
// @Description 모든 플랫폼 역할을 조회합니다.
// @Tags platform-roles
// @Accept json
// @Produce json
// @Success 200 {array} model.PlatformRole
// @Router /api/platform-roles [get]
func (h *PlatformRoleHandler) List(c echo.Context) error {
	roles, err := h.service.List()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "플랫폼 역할 목록을 가져오는데 실패했습니다",
		})
	}
	return c.JSON(http.StatusOK, roles)
}

// GetByID godoc
// @Summary 플랫폼 역할 조회
// @Description ID로 플랫폼 역할을 조회합니다.
// @Tags platform-roles
// @Accept json
// @Produce json
// @Param id path int true "Platform Role ID"
// @Success 200 {object} model.PlatformRole
// @Router /api/platform-roles/{id} [get]
func (h *PlatformRoleHandler) GetByID(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "잘못된 ID 형식입니다",
		})
	}

	role, err := h.service.GetByID(uint(id))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "플랫폼 역할을 찾을 수 없습니다",
		})
	}
	return c.JSON(http.StatusOK, role)
}

// Create godoc
// @Summary 플랫폼 역할 생성
// @Description 새로운 플랫폼 역할을 생성합니다.
// @Tags platform-roles
// @Accept json
// @Produce json
// @Param role body model.PlatformRole true "Platform Role"
// @Success 201 {object} model.PlatformRole
// @Router /api/platform-roles [post]
func (h *PlatformRoleHandler) Create(c echo.Context) error {
	role := new(model.PlatformRole)
	if err := c.Bind(role); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "잘못된 요청 형식입니다",
		})
	}

	if err := h.service.Create(role); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "플랫폼 역할 생성에 실패했습니다",
		})
	}
	return c.JSON(http.StatusCreated, role)
}

// Update godoc
// @Summary 플랫폼 역할 수정
// @Description 기존 플랫폼 역할을 수정합니다.
// @Tags platform-roles
// @Accept json
// @Produce json
// @Param id path int true "Platform Role ID"
// @Param role body model.PlatformRole true "Platform Role"
// @Success 200 {object} model.PlatformRole
// @Router /api/platform-roles/{id} [put]
func (h *PlatformRoleHandler) Update(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "잘못된 ID 형식입니다",
		})
	}

	role := new(model.PlatformRole)
	if err := c.Bind(role); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "잘못된 요청 형식입니다",
		})
	}
	role.ID = uint(id)

	if err := h.service.Update(role); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "플랫폼 역할 수정에 실패했습니다",
		})
	}
	return c.JSON(http.StatusOK, role)
}

// Delete godoc
// @Summary 플랫폼 역할 삭제
// @Description 플랫폼 역할을 삭제합니다.
// @Tags platform-roles
// @Accept json
// @Produce json
// @Param id path int true "Platform Role ID"
// @Success 204 "No Content"
// @Router /api/platform-roles/{id} [delete]
func (h *PlatformRoleHandler) Delete(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "잘못된 ID 형식입니다",
		})
	}

	if err := h.service.Delete(uint(id)); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "플랫폼 역할 삭제에 실패했습니다",
		})
	}
	return c.NoContent(http.StatusNoContent)
}
