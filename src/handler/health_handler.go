package handler

import (
	"net/http"

	// "github.com/Nerzal/gocloak/v13" // Removed unused import
	"github.com/labstack/echo/v4"
	// "github.com/m-cmp/mc-iam-manager/config" // Removed unused import
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm" // Import gorm
)

// HealthCheckHandler 핸들러 구조체
type HealthCheckHandler struct {
	healthService service.HealthCheckService
	// db *gorm.DB // Not needed directly
	// keycloakConfig *config.KeycloakConfig // Not needed directly
	// keycloakClient *gocloak.GoCloak // Not needed directly
}

// NewHealthCheckHandler 생성자
func NewHealthCheckHandler(db *gorm.DB) *HealthCheckHandler { // Remove keycloakService parameter
	// Initialize service internally
	healthService := service.NewHealthCheckService(db) // Pass only db
	return &HealthCheckHandler{
		healthService: healthService,
	}
}

// ReadyzCheck godoc
// @Summary 애플리케이션 준비 상태 확인
// @Description 애플리케이션의 준비 상태를 확인합니다. status=detail 쿼리 파라미터로 상세 상태를 확인할 수 있습니다.
// @Tags Health
// @Produce json
// @Param status query string false "상세 상태 확인 여부 ('detail')"
// @Success 200 {object} map[string]string "status: ok"
// @Success 200 {object} service.HealthStatus "상세 상태 정보 (status=detail)"
// @Failure 503 {object} service.HealthStatus "상세 상태 확인 중 오류 발생 시"
// @Router /readyz [get]
func (h *HealthCheckHandler) ReadyzCheck(c echo.Context) error {
	statusParam := c.QueryParam("status")

	if statusParam == "detail" {
		detailedStatus, err := h.healthService.GetDetailedStatus(c.Request().Context())
		if err != nil {
			// 서비스 자체에서 오류가 발생한 경우 (거의 발생하지 않음)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get detailed status"})
		}

		// 개별 체크 항목 중 하나라도 실패했는지 확인 (예: DB 연결 실패)
		isHealthy := true
		if detailedStatus.DBConnection != "OK" ||
			detailedStatus.KeycloakAdminLogin != "OK" ||
			detailedStatus.KeycloakRealmCheck != "OK" ||
			detailedStatus.KeycloakClientCheck != "OK" {
			isHealthy = false
		}

		if isHealthy {
			return c.JSON(http.StatusOK, detailedStatus)
		} else {
			// 시스템 일부에 문제가 있을 경우 503 Service Unavailable 반환
			return c.JSON(http.StatusServiceUnavailable, detailedStatus)
		}
	} else {
		// 기본 응답
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}
