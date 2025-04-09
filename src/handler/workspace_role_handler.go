package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
)

type WorkspaceRoleHandler struct {
	service *service.WorkspaceRoleService
}

func NewWorkspaceRoleHandler(service *service.WorkspaceRoleService) *WorkspaceRoleHandler {
	return &WorkspaceRoleHandler{
		service: service,
	}
}

func (h *WorkspaceRoleHandler) List(c echo.Context) error {
	roles, err := h.service.List()
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, roles)
}

func (h *WorkspaceRoleHandler) GetByID(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}
	role, err := h.service.GetByID(uint(id))
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, role)
}

func (h *WorkspaceRoleHandler) Create(c echo.Context) error {
	var role model.WorkspaceRole
	if err := c.Bind(&role); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	if err := h.service.Create(&role); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, role)
}

func (h *WorkspaceRoleHandler) Update(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}
	var role model.WorkspaceRole
	role.ID = uint(id)
	if err := c.Bind(&role); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	if err := h.service.Update(&role); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, role)
}

func (h *WorkspaceRoleHandler) Delete(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}
	if err := h.service.Delete(uint(id)); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.NoContent(204)
}
