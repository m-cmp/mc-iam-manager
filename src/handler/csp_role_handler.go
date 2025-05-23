package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
)

type CspRoleHandler struct {
	service *service.CspRoleService
}

func NewCspRoleHandler() *CspRoleHandler {
	svc := service.NewCspRoleService()

	return &CspRoleHandler{
		service: svc,
	}
}

// GetAllCSPRoles 모든 CSP 역할을 조회합니다.
func (h *CspRoleHandler) GetAllCSPRoles(c echo.Context) error {
	roles, err := h.service.GetAllCSPRoles(c.Request().Context(), "aws")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, roles)
}

// CreateCSPRole 새로운 CSP 역할을 생성합니다.
func (h *CspRoleHandler) CreateCSPRole(c echo.Context) error {
	var role model.CspRole
	if err := c.Bind(&role); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	if err := h.service.CreateCSPRole(&role); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, role)
}

// UpdateCSPRole CSP 역할 정보를 수정합니다.
func (h *CspRoleHandler) UpdateCSPRole(c echo.Context) error {
	var role model.CspRole
	if err := c.Bind(&role); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	if err := h.service.UpdateCSPRole(&role); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, role)
}

// DeleteCSPRole CSP 역할을 삭제합니다.
func (h *CspRoleHandler) DeleteCSPRole(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "role id is required",
		})
	}

	if err := h.service.DeleteCSPRole(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.NoContent(http.StatusNoContent)
}

// GetCSPRoles 특정 CSP 타입의 역할을 조회합니다.
func (h *CspRoleHandler) GetCSPRoles(c echo.Context) error {
	cspType := c.QueryParam("csp_type")
	if cspType == "" {
		cspType = "aws" // 기본값으로 aws 설정
	}

	roles, err := h.service.GetCSPRoles(c.Request().Context(), cspType)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, roles)
}
