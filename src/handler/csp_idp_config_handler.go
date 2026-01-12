package handler

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
	"github.com/m-cmp/mc-iam-manager/util"
	"gorm.io/gorm"
)

// CspIdpConfigHandler CSP IDP 설정 관리 핸들러
type CspIdpConfigHandler struct {
	cspIdpConfigService *service.CspIdpConfigService
}

// NewCspIdpConfigHandler 새 CspIdpConfigHandler 인스턴스 생성
func NewCspIdpConfigHandler(db *gorm.DB) *CspIdpConfigHandler {
	keycloakService := service.NewKeycloakService()
	return &CspIdpConfigHandler{
		cspIdpConfigService: service.NewCspIdpConfigService(db, keycloakService),
	}
}

// CreateCspIdpConfig godoc
// @Summary Create CSP IDP config
// @Description Create a new CSP IDP configuration
// @Tags csp-idp-configs
// @Accept json
// @Produce json
// @Param config body model.CreateCspIdpConfigRequest true "CSP IDP Config Info"
// @Success 201 {object} model.CspIdpConfig
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/csp-idp-configs [post]
// @Id createCspIdpConfig
func (h *CspIdpConfigHandler) CreateCspIdpConfig(c echo.Context) error {
	var req model.CreateCspIdpConfigRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	// 필수 필드 검증
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Name is required"})
	}
	if req.CspAccountID == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "CSP Account ID is required"})
	}
	if req.AuthMethod == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Auth method is required"})
	}
	if req.AuthMethod != model.AuthMethodOIDC && req.AuthMethod != model.AuthMethodSAML && req.AuthMethod != model.AuthMethodSecretKey {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid auth method. Must be one of: OIDC, SAML, SECRET_KEY"})
	}
	if req.Config == nil || len(req.Config) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Config is required"})
	}

	idpConfig, err := h.cspIdpConfigService.CreateCspIdpConfig(&req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to create IDP config: %v", err)})
	}

	return c.JSON(http.StatusCreated, idpConfig)
}

// ListCspIdpConfigs godoc
// @Summary List CSP IDP configs
// @Description Retrieve a list of CSP IDP configurations with optional filters
// @Tags csp-idp-configs
// @Accept json
// @Produce json
// @Param filter body model.CspIdpConfigFilter false "Filter options"
// @Success 200 {array} model.CspIdpConfig
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/csp-idp-configs/list [post]
// @Id listCspIdpConfigs
func (h *CspIdpConfigHandler) ListCspIdpConfigs(c echo.Context) error {
	var filter model.CspIdpConfigFilter
	if err := c.Bind(&filter); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	configs, err := h.cspIdpConfigService.ListCspIdpConfigs(&filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to list IDP configs: %v", err)})
	}

	if configs == nil {
		configs = []*model.CspIdpConfig{}
	}

	return c.JSON(http.StatusOK, configs)
}

// GetCspIdpConfigByID godoc
// @Summary Get CSP IDP config by ID
// @Description Retrieve CSP IDP configuration details by ID
// @Tags csp-idp-configs
// @Accept json
// @Produce json
// @Param configId path string true "Config ID"
// @Success 200 {object} model.CspIdpConfig
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/csp-idp-configs/id/{configId} [get]
// @Id getCspIdpConfigByID
func (h *CspIdpConfigHandler) GetCspIdpConfigByID(c echo.Context) error {
	configID, err := util.StringToUint(c.Param("configId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid config ID"})
	}

	idpConfig, err := h.cspIdpConfigService.GetCspIdpConfigByID(configID)
	if err != nil {
		if err.Error() == fmt.Sprintf("IDP config not found with ID: %d", configID) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to get IDP config: %v", err)})
	}

	return c.JSON(http.StatusOK, idpConfig)
}

// UpdateCspIdpConfig godoc
// @Summary Update CSP IDP config
// @Description Update CSP IDP configuration details
// @Tags csp-idp-configs
// @Accept json
// @Produce json
// @Param configId path string true "Config ID"
// @Param config body model.UpdateCspIdpConfigRequest true "CSP IDP Config Info"
// @Success 200 {object} model.CspIdpConfig
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/csp-idp-configs/id/{configId} [put]
// @Id updateCspIdpConfig
func (h *CspIdpConfigHandler) UpdateCspIdpConfig(c echo.Context) error {
	configID, err := util.StringToUint(c.Param("configId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid config ID"})
	}

	var req model.UpdateCspIdpConfigRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	idpConfig, err := h.cspIdpConfigService.UpdateCspIdpConfig(configID, &req)
	if err != nil {
		if err.Error() == fmt.Sprintf("IDP config not found with ID: %d", configID) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to update IDP config: %v", err)})
	}

	return c.JSON(http.StatusOK, idpConfig)
}

// DeleteCspIdpConfig godoc
// @Summary Delete CSP IDP config
// @Description Delete a CSP IDP configuration by ID
// @Tags csp-idp-configs
// @Accept json
// @Produce json
// @Param configId path string true "Config ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/csp-idp-configs/id/{configId} [delete]
// @Id deleteCspIdpConfig
func (h *CspIdpConfigHandler) DeleteCspIdpConfig(c echo.Context) error {
	configID, err := util.StringToUint(c.Param("configId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid config ID"})
	}

	if err := h.cspIdpConfigService.DeleteCspIdpConfig(configID); err != nil {
		if err.Error() == fmt.Sprintf("IDP config not found with ID: %d", configID) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to delete IDP config: %v", err)})
	}

	return c.NoContent(http.StatusNoContent)
}

// TestCspIdpConnection godoc
// @Summary Test CSP IDP connection
// @Description Test connection to CSP using IDP configuration
// @Tags csp-idp-configs
// @Accept json
// @Produce json
// @Param configId path string true "Config ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/csp-idp-configs/id/{configId}/test [post]
// @Id testCspIdpConnection
func (h *CspIdpConfigHandler) TestCspIdpConnection(c echo.Context) error {
	configID, err := util.StringToUint(c.Param("configId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid config ID"})
	}

	if err := h.cspIdpConfigService.TestConnection(c.Request().Context(), configID); err != nil {
		if err.Error() == fmt.Sprintf("IDP config not found with ID: %d", configID) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Connection test failed: %v", err)})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Connection test successful"})
}

// ActivateCspIdpConfig godoc
// @Summary Activate CSP IDP config
// @Description Activate a CSP IDP configuration
// @Tags csp-idp-configs
// @Accept json
// @Produce json
// @Param configId path string true "Config ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/csp-idp-configs/id/{configId}/activate [post]
// @Id activateCspIdpConfig
func (h *CspIdpConfigHandler) ActivateCspIdpConfig(c echo.Context) error {
	configID, err := util.StringToUint(c.Param("configId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid config ID"})
	}

	if err := h.cspIdpConfigService.ActivateIdpConfig(configID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to activate IDP config: %v", err)})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "IDP config activated successfully"})
}

// DeactivateCspIdpConfig godoc
// @Summary Deactivate CSP IDP config
// @Description Deactivate a CSP IDP configuration
// @Tags csp-idp-configs
// @Accept json
// @Produce json
// @Param configId path string true "Config ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/csp-idp-configs/id/{configId}/deactivate [post]
// @Id deactivateCspIdpConfig
func (h *CspIdpConfigHandler) DeactivateCspIdpConfig(c echo.Context) error {
	configID, err := util.StringToUint(c.Param("configId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid config ID"})
	}

	if err := h.cspIdpConfigService.DeactivateIdpConfig(configID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to deactivate IDP config: %v", err)})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "IDP config deactivated successfully"})
}
