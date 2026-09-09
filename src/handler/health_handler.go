package handler

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	// "github.com/Nerzal/gocloak/v13" // Removed unused import
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	// "github.com/m-cmp/mc-iam-manager/config" // Removed unused import
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/m-cmp/mc-iam-manager/service"
	// Import gorm
)

// mcInfraManagerServiceName is the registered mcmp-api framework name for mc-infra-manager,
// used to look up its BaseURL (same registry McmpApiCall uses, e.g. project_service.go).
const mcInfraManagerServiceName = "mc-infra-manager"

// HealthHandler health check handler
type HealthHandler struct {
	keycloakService service.KeycloakService
	db              *gorm.DB
	mcmpApiRepo     repository.McmpApiRepository
}

// NewHealthHandler create new HealthHandler instance
func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{
		keycloakService: service.NewKeycloakService(),
		db:              db,
		mcmpApiRepo:     repository.NewMcmpApiRepository(db),
	}
}

// CheckHealth godoc
// @Summary Health check
// @Description Check the health status of the service.
// @Tags health
// @Accept json
// @Produce json
// @Param detail query string false "Detail check components (nginx,db,keycloak,all)"
// @Success 200 {object} map[string]string
// @Router /readyz [get]
// @Id mciamCheckHealth
func (h *HealthHandler) CheckHealth(c echo.Context) error {
	status := c.QueryParam("status")

	// 기본 health check (파라미터 없음) - 모든 dependency 체크
	if status == "" {
		return h.checkBasicHealth(c)
	}

	// mc-iam-manager만 체크
	if status == "mc-iam-manager" {
		return h.checkMcIamManagerOnly(c)
	}

	// 상세 health check (특정 컴포넌트들)
	return h.checkDetailedHealth(c, status)
}

// checkBasicHealth performs basic health check for all components
func (h *HealthHandler) checkBasicHealth(c echo.Context) error {
	log.Println("Starting basic health check...")

	// 모든 컴포넌트 상태 확인
	keycloakHealthy := h.isKeycloakHealthy()
	dbHealthy := h.isDatabaseHealthy()
	nginxHealthy := h.isNginxHealthy()
	mcInfraManagerHealthy := h.isMcInfraManagerHealthy()

	// 모든 컴포넌트가 healthy인지 확인
	if keycloakHealthy && dbHealthy && nginxHealthy && mcInfraManagerHealthy {
		log.Println("All components are healthy.")
		return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
	}

	// 하나라도 unhealthy면 전체 unhealthy
	unhealthyComponents := []string{}
	if !keycloakHealthy {
		unhealthyComponents = append(unhealthyComponents, "keycloak")
	}
	if !dbHealthy {
		unhealthyComponents = append(unhealthyComponents, "db")
	}
	if !nginxHealthy {
		unhealthyComponents = append(unhealthyComponents, "nginx")
	}
	if !mcInfraManagerHealthy {
		unhealthyComponents = append(unhealthyComponents, "mc-infra-manager")
	}

	log.Printf("Health check failed. Unhealthy components: %v", unhealthyComponents)
	return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
		"status":               "unhealthy",
		"error":                "unhealthy components: " + strings.Join(unhealthyComponents, ", "),
		"unhealthy_components": unhealthyComponents,
	})
}

// checkDetailedHealth performs detailed health check based on detail parameter
func (h *HealthHandler) checkDetailedHealth(c echo.Context, detail string) error {
	log.Printf("Starting detailed health check for components: %s", detail)

	result := make(map[string]interface{})
	allHealthy := true
	unhealthyComponents := []string{}

	// 요청된 컴포넌트들 확인
	components := parseDetailComponents(detail)

	for _, component := range components {
		switch component {
		case "keycloak":
			if h.isKeycloakHealthy() {
				result["keycloak"] = "healthy"
			} else {
				result["keycloak"] = "unhealthy"
				unhealthyComponents = append(unhealthyComponents, "keycloak")
				allHealthy = false
			}
		case "db":
			if h.isDatabaseHealthy() {
				result["db"] = "healthy"
			} else {
				result["db"] = "unhealthy"
				unhealthyComponents = append(unhealthyComponents, "db")
				allHealthy = false
			}
		case "nginx":
			if h.isNginxHealthy() {
				result["nginx"] = "healthy"
			} else {
				result["nginx"] = "unhealthy"
				unhealthyComponents = append(unhealthyComponents, "nginx")
				allHealthy = false
			}
		case "mc-infra-manager":
			if h.isMcInfraManagerHealthy() {
				result["mc-infra-manager"] = "healthy"
			} else {
				result["mc-infra-manager"] = "unhealthy"
				unhealthyComponents = append(unhealthyComponents, "mc-infra-manager")
				allHealthy = false
			}
		}
	}

	// 전체 상태 추가
	if allHealthy {
		log.Printf("Detailed health check passed for components: %v", components)
		result["status"] = "healthy"
		return c.JSON(http.StatusOK, result)
	} else {
		log.Printf("Detailed health check failed. Unhealthy components: %v", unhealthyComponents)
		result["status"] = "unhealthy"
		result["error"] = "unhealthy components: " + strings.Join(unhealthyComponents, ", ")
		result["unhealthy_components"] = unhealthyComponents
		return c.JSON(http.StatusServiceUnavailable, result)
	}
}

// checkMcIamManagerOnly checks only mc-iam-manager service health
func (h *HealthHandler) checkMcIamManagerOnly(c echo.Context) error {
	log.Println("Checking mc-iam-manager service health...")

	// mc-iam-manager 서비스 자체의 상태만 확인
	// 데이터베이스 연결 상태 확인
	if err := h.db.Exec("SELECT 1").Error; err != nil {
		log.Printf("mc-iam-manager database connection failed: %v", err)
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"status": "unhealthy",
			"error":  "database connection failed",
		})
	}

	log.Println("mc-iam-manager service is healthy.")
	return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
}

// isKeycloakHealthy checks if Keycloak is healthy
func (h *HealthHandler) isKeycloakHealthy() bool {
	log.Println("Checking Keycloak health...")
	ctx := context.Background()

	// Check realm
	realmExists, err := h.keycloakService.CheckRealm(ctx)
	if err != nil || !realmExists {
		if err != nil {
			log.Printf("Keycloak realm check failed: %v", err)
		} else {
			log.Println("Keycloak realm does not exist")
		}
		return false
	}

	// Check client
	clientExists, err := h.keycloakService.CheckClient(ctx)
	if err != nil || !clientExists {
		if err != nil {
			log.Printf("Keycloak client check failed: %v", err)
		} else {
			log.Println("Keycloak client does not exist")
		}
		return false
	}

	log.Println("Keycloak is healthy.")
	return true
}

// isDatabaseHealthy checks if database is healthy
func (h *HealthHandler) isDatabaseHealthy() bool {
	log.Println("Checking database health...")
	if err := h.db.Exec("SELECT 1").Error; err != nil {
		log.Printf("Database health check failed: %v", err)
		return false
	}
	log.Println("Database is healthy.")
	return true
}

// isNginxHealthy checks if nginx is healthy
func (h *HealthHandler) isNginxHealthy() bool {
	log.Println("Checking nginx health...")
	response, err := http.Get("http://mc-iam-manager-nginx/nginx-health")
	if err != nil {
		log.Printf("Nginx health check failed: %v", err)
		return false
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		log.Println("Nginx is healthy.")
		return true
	}
	log.Printf("Nginx returned status code: %d", response.StatusCode)
	return false
}

// isMcInfraManagerHealthy checks if mc-infra-manager is healthy
func (h *HealthHandler) isMcInfraManagerHealthy() bool {
	log.Println("Checking mc-infra-manager health...")
	healthURL, err := h.frameworkReadyzURL(mcInfraManagerServiceName)
	if err != nil {
		log.Printf("mc-infra-manager health check failed: could not resolve health URL: %v", err)
		return false
	}
	response, err := http.Get(healthURL)
	if err != nil {
		log.Printf("mc-infra-manager health check failed: %v", err)
		return false
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		log.Println("mc-infra-manager is healthy.")
		return true
	}
	log.Printf("mc-infra-manager returned status code: %d", response.StatusCode)
	return false
}

// frameworkHealthURLEnvVar returns the per-framework health URL override env var name,
// e.g. "mc-infra-manager" -> "MC_INFRA_MANAGER_HEALTH_URL",
// "mc-observability" -> "MC_OBSERVABILITY_HEALTH_URL". No MC_IAM_MANAGER_ prefix —
// this names the target framework, not an mc-iam-manager setting. Every mcmp-api
// framework gets its own override this way, without a new hardcoded var per framework.
func frameworkHealthURLEnvVar(serviceName string) string {
	name := strings.ToUpper(strings.ReplaceAll(serviceName, "-", "_"))
	return name + "_HEALTH_URL"
}

// frameworkReadyzURL resolves the readyz health check URL for a registered mcmp-api
// framework. The per-framework env var (see frameworkHealthURLEnvVar) overrides it
// entirely; otherwise it's derived from the framework's registered BaseURL
// (mcmp_api_services) — the same source McmpApiCall uses for actual API calls — so
// the health check URL never drifts from the framework's real base URL.
func (h *HealthHandler) frameworkReadyzURL(serviceName string) (string, error) {
	if override := os.Getenv(frameworkHealthURLEnvVar(serviceName)); override != "" {
		return override, nil
	}
	svc, err := h.mcmpApiRepo.GetService(serviceName)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(svc.BaseURL, "/") + "/readyz", nil
}

// parseDetailComponents parses detail parameter into component list
func parseDetailComponents(detail string) []string {
	components := make([]string, 0)

	// 쉼표로 구분된 컴포넌트들 파싱
	parts := strings.Split(detail, ",")
	for _, part := range parts {
		component := strings.TrimSpace(part)
		if component != "" {
			components = append(components, component)
		}
	}

	// 디버그 로깅 추가
	log.Printf("Parsed components: %v", components)

	// "all"이 포함되어 있으면 모든 컴포넌트 반환
	for _, component := range components {
		if component == "all" {
			return []string{"keycloak", "db", "nginx", "mc-infra-manager"}
		}
	}

	return components
}
