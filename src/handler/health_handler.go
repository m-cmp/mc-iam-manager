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

// HealthHandler 헬스 체크 핸들러
type HealthHandler struct {
	keycloakService service.KeycloakService
}

// NewHealthHandler 새 HealthHandler 인스턴스 생성
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{
		keycloakService: service.NewKeycloakService(),
	}
}

// CheckHealth godoc
// @Summary Health check
// @Description Check if the service is healthy
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/health [get]
func (h *HealthHandler) CheckHealth(c echo.Context) error {
	status := c.QueryParam("status")
	if status == "detail" {
		// Keycloak 연결 확인
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

		// 클라이언트 확인
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
