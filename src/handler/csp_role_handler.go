package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm"
)

type CspRoleHandler struct {
	service *service.CspRoleService
}

func NewCspRoleHandler(db *gorm.DB) *CspRoleHandler {
	cspRoleService := service.NewCspRoleService(db)
	return &CspRoleHandler{
		service: cspRoleService,
	}
}

// GetAllCSPRoles godoc
// @Summary 모든 CSP 역할 조회 (관리자용)
// @Description 관리자가 모든 CSP 역할을 조회합니다.
// @Tags admin-csp-roles
// @Accept json
// @Produce json
// @Success 200 {array} model.CspRole
// @Security BearerAuth
// @Router /api/admin/csp-roles [get]
func (h *CspRoleHandler) GetAllCSPRoles(c echo.Context) error {
	roles, err := h.service.GetAllCSPRoles()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "CSP 역할 목록을 가져오는데 실패했습니다",
		})
	}
	return c.JSON(http.StatusOK, roles)
}

// CreateCSPRole godoc
// @Summary CSP 역할 생성 (관리자용)
// @Description 관리자가 새로운 CSP 역할을 생성합니다.
// @Tags admin-csp-roles
// @Accept json
// @Produce json
// @Param role body model.CspRole true "CSP 역할 정보"
// @Success 201 {object} model.CspRole
// @Security BearerAuth
// @Router /api/admin/csp-roles [post]
func (h *CspRoleHandler) CreateCSPRole(c echo.Context) error {
	var role model.CspRole
	if err := c.Bind(&role); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "잘못된 요청 형식입니다",
		})
	}

	if err := h.service.CreateCSPRole(&role); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "CSP 역할 생성에 실패했습니다",
		})
	}

	return c.JSON(http.StatusCreated, role)
}

// UpdateCSPRole godoc
// @Summary CSP 역할 수정 (관리자용)
// @Description 관리자가 CSP 역할 정보를 수정합니다.
// @Tags admin-csp-roles
// @Accept json
// @Produce json
// @Param id path string true "CSP 역할 ID"
// @Param role body model.CspRole true "CSP 역할 정보"
// @Success 200 {object} model.CspRole
// @Security BearerAuth
// @Router /api/admin/csp-roles/{id} [put]
func (h *CspRoleHandler) UpdateCSPRole(c echo.Context) error {
	id := c.Param("id")
	var role model.CspRole
	if err := c.Bind(&role); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "잘못된 요청 형식입니다",
		})
	}

	role.ID = id
	if err := h.service.UpdateCSPRole(&role); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "CSP 역할 수정에 실패했습니다",
		})
	}

	return c.JSON(http.StatusOK, role)
}

// DeleteCSPRole godoc
// @Summary CSP 역할 삭제 (관리자용)
// @Description 관리자가 CSP 역할을 삭제합니다.
// @Tags admin-csp-roles
// @Accept json
// @Produce json
// @Param id path string true "CSP 역할 ID"
// @Success 204 "No Content"
// @Security BearerAuth
// @Router /api/admin/csp-roles/{id} [delete]
func (h *CspRoleHandler) DeleteCSPRole(c echo.Context) error {
	id := c.Param("id")
	if err := h.service.DeleteCSPRole(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "CSP 역할 삭제에 실패했습니다",
		})
	}

	return c.NoContent(http.StatusNoContent)
}
