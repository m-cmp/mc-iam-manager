package handler

import (
	"context"
	"net/http"

	// "github.com/Nerzal/gocloak/v13" // Removed unused import
	"github.com/labstack/echo/v4"
	// "github.com/m-cmp/mc-iam-manager/config" // Removed unused import
	"github.com/m-cmp/mc-iam-manager/service"
	// Import gorm
)

// HealthHandler health check handler
type HealthHandler struct {
	keycloakService service.KeycloakService
}

// NewHealthHandler create new HealthHandler instance
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{
		keycloakService: service.NewKeycloakService(),
	}
}

// CheckHealth godoc
// @Summary Health check
// @Description Check the health status of the service.
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /readyz [get]
// @Id mciamCheckHealth
func (h *HealthHandler) CheckHealth(c echo.Context) error {
	status := c.QueryParam("status")
	if status == "detail" {
		// Check Keycloak connection
		ctx := context.Background()
		realmExists, err := h.keycloakService.CheckRealm(ctx)
		if err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{
				"status": "unhealthy",
				"error":  err.Error(),
			})
		}
		if !realmExists {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{
				"status": "unhealthy",
				"error":  "realm does not exist",
			})
		}

		// Check client
		clientExists, err := h.keycloakService.CheckClient(ctx)
		if err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{
				"status": "unhealthy",
				"error":  err.Error(),
			})
		}
		if !clientExists {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{
				"status": "unhealthy",
				"error":  "client does not exist",
			})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"status": "healthy",
			"realm":  "exists",
			"client": "exists",
		})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
}
