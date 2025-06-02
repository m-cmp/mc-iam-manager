package handler

import (
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm"
)

type CspRoleHandler struct {
	service *service.CspRoleService
	db      *gorm.DB
}

func NewCspRoleHandler(db *gorm.DB) *CspRoleHandler {
	svc := service.NewCspRoleService(db)

	return &CspRoleHandler{
		service: svc,
		db:      db,
	}
}

/////////// ROLE 관련은 role_handler.go 에 있음 ///////////

// // @Summary Get all CSP roles
// // @Description Get all CSP roles
// // @Tags csp-roles
// // @Accept json
// // @Produce json
// // @Success 200 {array} model.CspRole
// // @Failure 500 {object} map[string]string
// // @Security BearerAuth
// // @Router /api/v1/csp-roles/all [get]
// func (h *CspRoleHandler) GetAllCSPRoles(c echo.Context) error {
// 	roles, err := h.service.GetAllCSPRoles(c.Request().Context(), "aws")
// 	if err != nil {
// 		return c.JSON(http.StatusInternalServerError, map[string]string{
// 			"error": err.Error(),
// 		})
// 	}

// 	return c.JSON(http.StatusOK, roles)
// }

// // @Summary Create CSP role
// // @Description Create a new CSP role
// // @Tags csp-roles
// // @Accept json
// // @Produce json
// // @Param role body model.CspRole true "CSP Role Info"
// // @Success 201 {object} model.CspRole
// // @Failure 400 {object} map[string]string
// // @Failure 500 {object} map[string]string
// // @Security BearerAuth
// // @Router /api/v1/csp-roles [post]
// func (h *CspRoleHandler) CreateCSPRole(c echo.Context) error {
// 	var role model.CspRole
// 	if err := c.Bind(&role); err != nil {
// 		return c.JSON(http.StatusBadRequest, map[string]string{
// 			"error": err.Error(),
// 		})
// 	}

// 	if role.CspType == "" {
// 		role.CspType = "aws" // for the test
// 	}

// 	cspRole, err := h.service.CreateCSPRole(&role)
// 	if err != nil {
// 		return c.JSON(http.StatusInternalServerError, map[string]string{
// 			"error": err.Error(),
// 		})
// 	}

// 	return c.JSON(http.StatusCreated, cspRole)
// }

// // @Summary Update CSP role
// // @Description Update CSP role information
// // @Tags csp-roles
// // @Accept json
// // @Produce json
// // @Param id path string true "CSP Role ID"
// // @Param role body model.CspRole true "CSP Role Info"
// // @Success 200 {object} model.CspRole
// // @Failure 400 {object} map[string]string
// // @Failure 404 {object} map[string]string
// // @Failure 500 {object} map[string]string
// // @Security BearerAuth
// // @Router /api/v1/csp-roles/{id} [put]
// func (h *CspRoleHandler) UpdateCSPRole(c echo.Context) error {
// 	var role model.CspRole
// 	if err := c.Bind(&role); err != nil {
// 		return c.JSON(http.StatusBadRequest, map[string]string{
// 			"error": err.Error(),
// 		})
// 	}

// 	if err := h.service.UpdateCSPRole(&role); err != nil {
// 		return c.JSON(http.StatusInternalServerError, map[string]string{
// 			"error": err.Error(),
// 		})
// 	}

// 	return c.JSON(http.StatusOK, role)
// }

// // @Summary Delete CSP role
// // @Description Delete a CSP role
// // @Tags csp-roles
// // @Accept json
// // @Produce json
// // @Param id path string true "CSP Role ID"
// // @Success 204 "No Content"
// // @Failure 404 {object} map[string]string
// // @Failure 500 {object} map[string]string
// // @Security BearerAuth
// // @Router /api/v1/csp-roles/{id} [delete]
// func (h *CspRoleHandler) DeleteCSPRole(c echo.Context) error {
// 	id := c.Param("cspRoleId")
// 	if id == "" {
// 		return c.JSON(http.StatusBadRequest, map[string]string{
// 			"error": "role id is required",
// 		})
// 	}

// 	if err := h.service.DeleteCSPRole(id); err != nil {
// 		return c.JSON(http.StatusInternalServerError, map[string]string{
// 			"error": err.Error(),
// 		})
// 	}

// 	return c.NoContent(http.StatusNoContent)
// }

// // @Summary Get CSP roles
// // @Description Get CSP roles with filters : db에 저장된 cspRole만 조회 ( csp 안찌름 )
// // @Tags csp-roles
// // @Accept json
// // @Produce json
// // @Success 200 {array} model.CspRole
// // @Failure 500 {object} map[string]string
// // @Security BearerAuth
// // @Router /api/v1/csp-roles [get]
// func (h *CspRoleHandler) GetMciamCSPRoles(c echo.Context) error {
// 	cspType := c.QueryParam("csp_type")
// 	if cspType == "" {
// 		cspType = "aws" // 기본값으로 aws 설정
// 	}

// 	roles, err := h.service.GetMciamCSPRoles(c.Request().Context(), cspType)
// 	if err != nil {
// 		return c.JSON(http.StatusInternalServerError, map[string]string{
// 			"error": err.Error(),
// 		})
// 	}

// 	return c.JSON(http.StatusOK, roles)
// }

// // @Summary Add permissions to CSP role
// // @Description Add permissions to a CSP role
// // @Tags csp-roles
// // @Accept json
// // @Produce json
// // @Param id path string true "CSP Role ID"
// // @Param permissions body []string true "Permissions to add"
// // @Success 200 {object} map[string]string
// // @Failure 400 {object} map[string]string
// // @Failure 404 {object} map[string]string
// // @Failure 500 {object} map[string]string
// // @Security BearerAuth
// // @Router /api/v1/csp-roles/{id}/permissions [post]
// func (h *CspRoleHandler) AddPermissionsToCSPRole(c echo.Context) error {
// 	id := c.Param("cspRoleId")
// 	if id == "" {
// 		return c.JSON(http.StatusBadRequest, map[string]string{
// 			"error": "role id is required",
// 		})
// 	}

// 	var permissions []string
// 	if err := c.Bind(&permissions); err != nil {
// 		return c.JSON(http.StatusBadRequest, map[string]string{
// 			"error": err.Error(),
// 		})
// 	}

// 	if err := h.service.AddPermissionsToCSPRole(id, permissions); err != nil {
// 		return c.JSON(http.StatusInternalServerError, map[string]string{
// 			"error": err.Error(),
// 		})
// 	}

// 	return c.JSON(http.StatusOK, map[string]string{
// 		"message": "Permissions added successfully",
// 	})
// }

// // @Summary Remove permissions from CSP role
// // @Description Remove permissions from a CSP role
// // @Tags csp-roles
// // @Accept json
// // @Produce json
// // @Param id path string true "CSP Role ID"
// // @Param permissions body []string true "Permissions to remove"
// // @Success 200 {object} map[string]string
// // @Failure 400 {object} map[string]string
// // @Failure 404 {object} map[string]string
// // @Failure 500 {object} map[string]string
// // @Security BearerAuth
// // @Router /api/v1/csp-roles/{id}/permissions [delete]
// func (h *CspRoleHandler) RemovePermissionsFromCSPRole(c echo.Context) error {
// 	id := c.Param("cspRoleId")
// 	if id == "" {
// 		return c.JSON(http.StatusBadRequest, map[string]string{
// 			"error": "role id is required",
// 		})
// 	}

// 	var permissions []string
// 	if err := c.Bind(&permissions); err != nil {
// 		return c.JSON(http.StatusBadRequest, map[string]string{
// 			"error": err.Error(),
// 		})
// 	}

// 	if err := h.service.RemovePermissionsFromCSPRole(id, permissions); err != nil {
// 		return c.JSON(http.StatusInternalServerError, map[string]string{
// 			"error": err.Error(),
// 		})
// 	}

// 	return c.JSON(http.StatusOK, map[string]string{
// 		"message": "Permissions removed successfully",
// 	})
// }

// // @Summary Get CSP role permissions
// // @Description Get permissions of a CSP role
// // @Tags csp-roles
// // @Accept json
// // @Produce json
// // @Param id path string true "CSP Role ID"
// // @Success 200 {array} string
// // @Failure 400 {object} map[string]string
// // @Failure 404 {object} map[string]string
// // @Failure 500 {object} map[string]string
// // @Security BearerAuth
// // @Router /api/v1/csp-roles/{id}/permissions [get]
// func (h *CspRoleHandler) GetCSPRolePermissions(c echo.Context) error {
// 	id := c.Param("cspRoleId")
// 	if id == "" {
// 		return c.JSON(http.StatusBadRequest, map[string]string{
// 			"error": "role id is required",
// 		})
// 	}

// 	permissions, err := h.service.GetCSPRolePermissions(id)
// 	if err != nil {
// 		return c.JSON(http.StatusInternalServerError, map[string]string{
// 			"error": err.Error(),
// 		})
// 	}

// 	return c.JSON(http.StatusOK, permissions)
// }

// // @Summary Get role policies
// // @Description Get all policies attached to a role
// // @Tags csp-roles
// // @Accept json
// // @Produce json
// // @Param roleName path string true "Role Name"
// // @Success 200 {object} model.CspRole
// // @Failure 500 {object} map[string]string
// // @Security BearerAuth
// // @Router /api/v1/csp-roles/{roleName}/policies [get]
// func (h *CspRoleHandler) GetRolePolicies(c echo.Context) error {
// 	roleName := c.Param("roleName")
// 	policies, err := h.service.GetRolePolicies(c.Request().Context(), roleName)
// 	if err != nil {
// 		return c.JSON(http.StatusInternalServerError, map[string]string{
// 			"error": err.Error(),
// 		})
// 	}
// 	return c.JSON(http.StatusOK, policies)
// }

// // @Summary Get role policy
// // @Description Get a specific inline policy of a role
// // @Tags csp-roles
// // @Accept json
// // @Produce json
// // @Param roleName path string true "Role Name"
// // @Param policyName path string true "Policy Name"
// // @Success 200 {object} csp.RolePolicy
// // @Failure 500 {object} map[string]string
// // @Security BearerAuth
// // @Router /api/v1/csp-roles/{roleName}/policies/{policyName} [get]
// func (h *CspRoleHandler) GetRolePolicy(c echo.Context) error {
// 	roleName := c.Param("roleName")
// 	policyName := c.Param("policyName")
// 	policy, err := h.service.GetRolePolicy(c.Request().Context(), roleName, policyName)
// 	if err != nil {
// 		return c.JSON(http.StatusInternalServerError, map[string]string{
// 			"error": err.Error(),
// 		})
// 	}
// 	return c.JSON(http.StatusOK, policy)
// }

// // @Summary Put role policy
// // @Description Add or update an inline policy for a role
// // @Tags csp-roles
// // @Accept json
// // @Produce json
// // @Param roleName path string true "Role Name"
// // @Param policyName path string true "Policy Name"
// // @Param policy body csp.RolePolicy true "Policy Document"
// // @Success 200 {object} csp.RolePolicy
// // @Failure 500 {object} map[string]string
// // @Security BearerAuth
// // @Router /api/v1/csp-roles/{roleName}/policies/{policyName} [put]
// func (h *CspRoleHandler) PutRolePolicy(c echo.Context) error {
// 	roleName := c.Param("roleName")
// 	policyName := c.Param("policyName")

// 	var policy csp.RolePolicy
// 	if err := c.Bind(&policy); err != nil {
// 		return c.JSON(http.StatusBadRequest, map[string]string{
// 			"error": err.Error(),
// 		})
// 	}

// 	err := h.service.PutRolePolicy(c.Request().Context(), roleName, policyName, &policy)
// 	if err != nil {
// 		return c.JSON(http.StatusInternalServerError, map[string]string{
// 			"error": err.Error(),
// 		})
// 	}
// 	return c.JSON(http.StatusOK, policy)
// }

// // @Summary Delete role policy
// // @Description Delete an inline policy from a role
// // @Tags csp-roles
// // @Accept json
// // @Produce json
// // @Param roleName path string true "Role Name"
// // @Param policyName path string true "Policy Name"
// // @Success 204 "No Content"
// // @Failure 500 {object} map[string]string
// // @Security BearerAuth
// // @Router /api/v1/csp-roles/{roleName}/policies/{policyName} [delete]
// func (h *CspRoleHandler) DeleteRolePolicy(c echo.Context) error {
// 	roleName := c.Param("roleName")
// 	policyName := c.Param("policyName")

// 	err := h.service.DeleteRolePolicy(c.Request().Context(), roleName, policyName)
// 	if err != nil {
// 		return c.JSON(http.StatusInternalServerError, map[string]string{
// 			"error": err.Error(),
// 		})
// 	}
// 	return c.NoContent(http.StatusNoContent)
// }
