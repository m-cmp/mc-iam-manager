package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings" // Import strings

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/config" // Import config for Keycloak client
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/model/mcmpapi"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm" // Import gorm
)

const apiYamlEnvVar = "MC_ADMIN_CLI_APIYAML"

// McmpApiHandler handles requests related to mcmp API definitions. (Renamed)
type McmpApiHandler struct {
	service          service.McmpApiService // Use renamed service interface
	projectService   *service.ProjectService
	workspaceService *service.WorkspaceService
	userService      *service.UserService
	mcmpApiRepo      repository.McmpApiRepository
	// db *gorm.DB // Not needed directly in handler
}

// NewMcmpApiHandler creates a new McmpApiHandler. (Renamed)
func NewMcmpApiHandler(db *gorm.DB) *McmpApiHandler { // Accept db, remove service param
	// Initialize service internally
	mcmpApiService := service.NewMcmpApiService(db)
	return &McmpApiHandler{
		service:          mcmpApiService, // Renamed struct type
		projectService:   service.NewProjectService(db),
		workspaceService: service.NewWorkspaceService(db),
		userService:      service.NewUserService(db),
		mcmpApiRepo:      repository.NewMcmpApiRepository(db),
	}
}

// SyncMcmpAPIs godoc
// @Summary Sync MCMP API Definitions
// @Description Triggers the synchronization of MCMP API definitions from the configured YAML URL to the database.
// @Tags McmpAPI
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "message: Successfully triggered MCMP API sync"
// @Failure 500 {object} map[string]string "message: Failed to trigger MCMP API sync"
// @Router /api/setup/sync-mcmp-apis [post]
// @Security BearerAuth
// @Id syncMcmpAPIs
func (h *McmpApiHandler) SyncMcmpAPIs(c echo.Context) error {
	err := h.service.SyncMcmpAPIsFromYAML()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to trigger MCMP API sync: " + err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "Successfully triggered MCMP API sync"})
}

// ImportAPIs godoc
// @Summary Import MCMP APIs from Remote Sources
// @Description Fetches API specifications from remote URLs and imports them to the database. Supports swagger and openapi source types. Optionally accepts baseUrl and authentication info to populate the mcmp_api_services table.
// @Tags McmpAPI
// @Accept json
// @Produce json
// @Param request body model.ImportApiRequest true "Frameworks to import (with optional baseUrl, authType, authUser, authPass)"
// @Success 200 {object} model.ImportApiResponse "Import results"
// @Failure 400 {object} map[string]string "error: Invalid request body"
// @Failure 500 {object} map[string]string "error: Failed to import APIs"
// @Router /api/mcmp-apis/import [post]
// @Security BearerAuth
// @Id importAPIs
func (h *McmpApiHandler) ImportAPIs(c echo.Context) error {
	var req model.ImportApiRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
	}

	// Validate request
	if len(req.Frameworks) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "At least one framework is required"})
	}

	for i, fw := range req.Frameworks {
		if fw.Name == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Framework at index %d: name is required", i)})
		}
		if fw.Version == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Framework '%s': version is required", fw.Name)})
		}
		if fw.SourceType == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Framework '%s': sourceType is required (swagger or openapi)", fw.Name)})
		}
		if fw.SourceURL == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Framework '%s': sourceUrl is required", fw.Name)})
		}
	}

	response, err := h.service.ImportAPIs(&req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to import APIs: " + err.Error()})
	}

	return c.JSON(http.StatusOK, response)
}

// Add other handler methods if needed, e.g., to get API definitions via API

// SetActiveVersion godoc
// @Summary Set Active Version for a Service
// @Description Sets the specified version of an MCMP API service as the active one.
// @Tags McmpAPI
// @Accept json
// @Produce json
// @Param serviceName path string true "Service Name"
// @Param version path string true "Version to activate"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "error: Invalid service name or version"
// @Failure 404 {object} map[string]string "error: Service or version not found"
// @Failure 500 {object} map[string]string "error: Failed to set active version"
// @Router /api/mcmp-apis/name/{serviceName}/versions/{version}/activate [put]
// @Security BearerAuth
// @Id setActiveVersion
func (h *McmpApiHandler) SetActiveVersion(c echo.Context) error {
	serviceName := c.Param("serviceName")
	version := c.Param("version")

	if serviceName == "" || version == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Service name and version are required"})
	}

	// Call the service method to set the active version
	err := h.service.SetActiveVersion(serviceName, version)
	if err != nil {
		// Handle specific errors like "service version not found"
		// Assuming the service layer returns errors defined there or in the repo layer
		if errors.Is(err, errors.New("service version not found")) { // Check for specific error if defined
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		// Handle other potential errors (e.g., DB connection issues)
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to set active version: " + err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

// McmpApiCall godoc
// @Summary Call an external MCMP API action (Structured Request)
// @Description Executes a defined MCMP API action with parameters structured in McmpApiCallRequest.
// @Tags McmpAPI
// @Accept json
// @Produce json
// @Param callRequest body model.McmpApiCallRequest true "API Call Request"
// @Success 200 {object} object "External API Response (structure depends on the called API)"
// @Failure 400 {object} map[string]string "error: Invalid request body or parameters"
// @Failure 404 {object} map[string]string "error: Service or action not found"
// @Failure 500 {object} map[string]string "error: Internal server error or failed to call external API"
// @Failure 503 {object} map[string]string "error: External API unavailable"
// @Router /api/mcmp-apis/call [post]
// @Security BearerAuth
// @Id mcmpApiCall
func (h *McmpApiHandler) McmpApiCall(c echo.Context) error { // Renamed function
	var req model.McmpApiCallRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
	}

	// --- RPT Validation and Permission Check START ---
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authorization 헤더가 없거나 형식이 잘못되었습니다 (RPT 필요)."})
	}
	rptToken := strings.TrimPrefix(authHeader, "Bearer ")

	// Validate RPT token
	_, claims, err := config.KC.Client.DecodeAccessToken(c.Request().Context(), rptToken, config.KC.Realm)
	if err != nil {
		log.Printf("RPT 토큰 검증 실패 (McmpApiCall): %v", err)
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "유효하지 않거나 만료된 RPT 토큰입니다."})
	}

	// Extract permissions from RPT
	authClaim, ok := (*claims)["authorization"].(map[string]interface{})
	if !ok {
		log.Println("RPT 토큰에 'authorization' 클레임이 없습니다 (McmpApiCall).")
		return c.JSON(http.StatusForbidden, map[string]string{"error": "권한 거부: RPT에 authorization 클레임이 없습니다."})
	}
	permissionsClaim, ok := authClaim["permissions"].([]interface{})
	if !ok {
		log.Println("RPT 토큰에 'authorization.permissions' 클레임이 없습니다 (McmpApiCall).")
		return c.JSON(http.StatusForbidden, map[string]string{"error": "권한 거부: RPT에 permissions 클레임이 없습니다."})
	}

	// Check if the required permission is in the RPT claims
	// Required permission format: "serviceName#actionName" (based on Keycloak UMA resource/scope)
	requiredPermission := fmt.Sprintf("%s#%s", req.ServiceName, req.ActionName)
	hasPermission := false
	requiredParts := strings.SplitN(requiredPermission, "#", 2)
	requiredResource := requiredParts[0]
	requiredScope := ""
	if len(requiredParts) > 1 {
		requiredScope = requiredParts[1]
	} else {
		log.Printf("경고: requiredPermission 형식 오류 (McmpApiCall): %s", requiredPermission)
		// Decide how to handle - maybe deny access if format is wrong?
		// return c.JSON(http.StatusInternalServerError, map[string]string{"error": "서버 설정 오류: 잘못된 내부 권한 형식"})
	}

	for _, p := range permissionsClaim {
		permMap, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		rsname, rsnameOk := permMap["rsname"].(string) // Or rsid
		scopes, scopesOk := permMap["scopes"].([]interface{})
		if !rsnameOk || !scopesOk {
			continue
		}

		if rsname == requiredResource {
			for _, scopeInterface := range scopes {
				scope, ok := scopeInterface.(string)
				if ok && (requiredScope == "" || scope == requiredScope) {
					hasPermission = true
					break
				}
			}
		}
		if hasPermission {
			break
		}
	}

	if !hasPermission {
		log.Printf("권한 거부 (McmpApiCall): '%s' 필요. RPT 권한: %v", requiredPermission, permissionsClaim)
		return c.JSON(http.StatusForbidden, fmt.Sprintf("권한 거부: '%s' 권한이 필요합니다.", requiredPermission))
	}
	// --- RPT Validation and Permission Check END ---

	// --- IAM-TECH-003 방안2: nsId 소유권 기반 세밀 권한 제어 START ---
	isPlatformAdmin := checkRoleFromContext(c, []string{"platformAdmin"})
	kcUserID, _ := c.Get("kcUserId").(string)
	if status, msg := h.authorizeMcmpApiAction(c.Request().Context(), isPlatformAdmin, kcUserID, &req); status != 0 {
		return c.JSON(status, map[string]string{"error": msg})
	}
	// --- IAM-TECH-003 방안2: nsId 소유권 기반 세밀 권한 제어 END ---

	// If permission check passed, proceed to call the service
	statusCode, respBody, serviceVersion, calledURL, err := h.service.McmpApiCall(c.Request().Context(), &req) // Get new return values
	if err != nil {
		// Handle errors from the service layer (e.g., service/action not found, network error)
		// Include version and URL in the error message
		errMsg := fmt.Sprintf("Failed to call external API %s(v%s) %s (URL: %s): %v", req.ServiceName, serviceVersion, req.ActionName, calledURL, err)
		log.Printf(errMsg+" (Status Code: %d)", statusCode) // Log with status code if available

		// Return appropriate HTTP status codes based on the error type
		// Use the statusCode returned by McmpApiCall if it's an HTTP error status, otherwise default
		respStatus := http.StatusInternalServerError // Default error status
		if statusCode >= 400 {
			respStatus = statusCode
		}
		// Return a more informative error message
		return c.JSON(respStatus, map[string]string{"error": errMsg})
	}

	// Return the raw response body and status code from the external API
	// Set Content-Type based on what the external API returned, or default to application/json?
	// For simplicity, assume JSON for now, but might need adjustment.
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	c.Response().WriteHeader(statusCode)
	_, writeErr := c.Response().Write(respBody)
	if writeErr != nil {
		// Log error, but response header/status might have already been sent
		log.Printf("Error writing response body for API call %s/%s: %v", req.ServiceName, req.ActionName, writeErr)
		// Cannot return another JSON error here easily
		return writeErr
	}
	return nil // Response already written
}

// authorizeMcmpApiAction은 IAM-TECH-003 방안2의 nsId 소유권 검증을 수행한다.
// platformAdmin은 무조건 통과하고, 액션의 resourcePath에 {nsId}가 있으면 요청된 nsId가
// 속한 프로젝트의 워크스페이스에 사용자가 소속되어 있는지 확인한다. {nsId}가 없는
// "전체 목록"류 액션은 platformAdmin 전용을 유지한다. 통과하면 (0, ""),
// 거부되면 (HTTP status, 사용자 노출용 메시지)를 반환한다.
func (h *McmpApiHandler) authorizeMcmpApiAction(ctx context.Context, isPlatformAdmin bool, kcUserID string, req *model.McmpApiCallRequest) (int, string) {
	if isPlatformAdmin {
		return 0, ""
	}

	action, err := h.mcmpApiRepo.GetServiceAction(req.ServiceName, req.ActionName)
	if err != nil || action == nil {
		return http.StatusForbidden, fmt.Sprintf("권한 거부: 정의되지 않은 액션 '%s#%s'입니다.", req.ServiceName, req.ActionName)
	}

	if !strings.Contains(action.ResourcePath, "{nsId}") {
		return http.StatusForbidden, "권한 거부: platformAdmin만 호출할 수 있는 액션입니다."
	}

	nsId := req.RequestParams.PathParams["nsId"]
	if nsId == "" {
		return http.StatusBadRequest, "요청 requestParams.pathParams에 nsId가 필요합니다."
	}

	if kcUserID == "" {
		return http.StatusUnauthorized, "인증 정보(kcUserId)를 확인할 수 없습니다."
	}

	localUserID, err := h.userService.GetUserIDByKcID(ctx, kcUserID)
	if err != nil || localUserID == 0 {
		return http.StatusInternalServerError, "사용자 정보를 확인할 수 없습니다."
	}

	return h.authorizeNsOwnership(nsId, localUserID)
}

// authorizeNsOwnership은 nsId가 속한 프로젝트의 워크스페이스에 localUserID가 소속되어
// 있는지 확인한다. kcUserID→localUserID 변환(Keycloak 동기화를 유발할 수 있는
// UserService.GetUserIDByKcID)과 분리해두어, Project/Workspace/UserWorkspaceRole
// 픽스처만으로 독립적으로 단위 테스트할 수 있다.
func (h *McmpApiHandler) authorizeNsOwnership(nsId string, localUserID uint) (int, string) {
	projects, err := h.projectService.ListProjects(&model.ProjectFilterRequest{NsId: nsId})
	if err != nil {
		return http.StatusInternalServerError, fmt.Sprintf("프로젝트 조회 실패: %v", err)
	}
	if len(projects) == 0 {
		return http.StatusForbidden, fmt.Sprintf("권한 거부: nsId '%s'에 해당하는 프로젝트를 찾을 수 없습니다.", nsId)
	}

	for _, project := range projects {
		memberWorkspaces, err := h.workspaceService.ListWorkspaces(&model.WorkspaceFilterRequest{
			ProjectID: fmt.Sprintf("%d", project.ID),
			UserID:    fmt.Sprintf("%d", localUserID),
		})
		if err != nil {
			return http.StatusInternalServerError, fmt.Sprintf("워크스페이스 소속 확인 실패: %v", err)
		}
		if len(memberWorkspaces) > 0 {
			return 0, ""
		}
	}

	return http.StatusForbidden, fmt.Sprintf("권한 거부: nsId '%s'에 대한 접근 권한이 없습니다.", nsId)
}

// GetAllAPIDefinitions godoc
// @Summary Get All Stored MCMP API Definitions
// @Description Retrieves all MCMP API service and action definitions currently stored in the database.
// @Tags McmpAPI
// @Accept json
// @Produce json
// @Success 200 {object} mcmpapi.McmpApiDefinitions "Successfully retrieved API definitions"
// @Failure 500 {object} map[string]string "message: Failed to retrieve API definitions"
// @Param serviceName query string false "Filter by service name"
// @Param actionName query string false "Filter by action name (operationId)"
// @Router /api/mcmp-apis/list [post]
// @Security BearerAuth
// @Id listServicesAndActions
func (h *McmpApiHandler) ListServicesAndActions(c echo.Context) error {
	// Read query parameters for filtering
	serviceNameFilter := c.QueryParam("serviceName")
	actionNameFilter := c.QueryParam("actionName")

	defs, err := h.service.GetAllAPIDefinitions(serviceNameFilter, actionNameFilter) // Pass filters to service
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to retrieve API definitions: " + err.Error()})
	}
	if defs == nil {
		// Return an empty object instead of null if nothing is found
		return c.JSON(http.StatusOK, mcmpapi.McmpApiDefinitions{
			Services:       make(map[string]mcmpapi.McmpApiServiceDefinition),
			ServiceActions: make(map[string]map[string]mcmpapi.McmpApiServiceAction),
		})
	}
	return c.JSON(http.StatusOK, defs)
}

// TestCallGetAllNs godoc
// @Summary Test Call to mc-infra-manager GetAllNs
// @Description Calls the GetAllNs action of the mc-infra-manager service via the CallApi service.
// @Tags McmpAPI, Test
// @Produce json
// @Success 200 {object} object "Response from mc-infra-manager GetAllNs"
// @Failure 400 {object} map[string]string "error: Bad Request (e.g., invalid parameters)"
// @Failure 404 {object} map[string]string "error: Service or Action Not Found"
// @Failure 500 {object} map[string]string "error: Internal Server Error"
// @Failure 503 {object} map[string]string "error: External API Service Unavailable"
// @Router /api/mcmp-apis/test/mc-infra-manager/getallns [get]
// @Security BearerAuth
// @Id testCallGetAllNs
func (h *McmpApiHandler) TestCallGetAllNs(c echo.Context) error {
	// Prepare the request for the CallApi service
	callReq := &model.McmpApiCallRequest{
		ServiceName: "mc-infra-manager", // Target service
		ActionName:  "GetAllNs",         // Target action (operationId)
		RequestParams: model.McmpApiRequestParams{ // No params needed for GetAllNs
			PathParams:  nil,
			QueryParams: nil,
			Body:        nil,
		},
	}

	log.Printf("Initiating test call via McmpApiCall service: %+v", callReq)

	// Call the generic McmpApiCall service method
	statusCode, respBody, serviceVersion, calledURL, err := h.service.McmpApiCall(c.Request().Context(), callReq) // Get new return values
	if err != nil {
		// Handle errors from the service layer
		errMsg := fmt.Sprintf("Failed test call to %s(v%s) %s (URL: %s): %v", callReq.ServiceName, serviceVersion, callReq.ActionName, calledURL, err)
		log.Printf(errMsg+" (Status Code: %d)", statusCode)

		respStatus := http.StatusInternalServerError // Default error status
		if statusCode >= 400 {
			respStatus = statusCode
		}
		return c.JSON(respStatus, map[string]string{"error": errMsg})
	}

	// Return the response received from the external API
	// Assuming the response is JSON, otherwise adjust Content-Type
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	c.Response().WriteHeader(statusCode)
	_, writeErr := c.Response().Write(respBody)
	if writeErr != nil {
		log.Printf("Error writing response body for test call %s/%s: %v", callReq.ServiceName, callReq.ActionName, writeErr)
		return writeErr
	}
	return nil
}

// CreateFrameworkService godoc
// @Summary Create MCMP API Service Definition
// @Description Creates a new MCMP API service (framework) record. Returns 409 if a service with the same name already exists.
// @Tags McmpAPI
// @Accept json
// @Produce json
// @Param body body model.CreateMcmpApiServiceRequest true "Service definition"
// @Success 201 {object} mcmpapi.McmpApiService
// @Failure 400 {object} map[string]string "error: invalid request body or validation failure"
// @Failure 409 {object} map[string]string "error: framework service already exists"
// @Failure 500 {object} map[string]string "error: internal server error"
// @Router /api/mcmp-apis [post]
// @Id CreateFrameworkService
// @Security BearerAuth
func (h *McmpApiHandler) CreateFrameworkService(c echo.Context) error {
	var req model.CreateMcmpApiServiceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body: " + err.Error()})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	svc := &mcmpapi.McmpApiService{
		Name:     req.Name,
		Version:  req.Version,
		BaseURL:  req.BaseURL,
		AuthType: req.AuthType,
		AuthUser: req.AuthUser,
		AuthPass: req.AuthPass,
		IsActive: req.IsActive,
	}
	if err := h.service.CreateFrameworkService(svc); err != nil {
		if errors.Is(err, service.ErrFrameworkServiceAlreadyExists) {
			return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, svc)
}

// UpdateService godoc
// @Summary Update MCMP API Service Definition
// @Description Updates specific fields (e.g., BaseURL, Auth info) of an MCMP API service definition identified by its name. Cannot update name or version.
// @Tags McmpAPI
// @Accept json
// @Produce json
// @Param serviceName path string true "Service Name to update"
// @Param updates body object true "Fields to update (e.g., {\"baseurl\": \"http://new-url\", \"auth_type\": \"none\"})"
// @Success 200 {object} map[string]string "message: Service updated successfully" // Or return updated service?
// @Failure 400 {object} map[string]string "error: Invalid service name or request body"
// @Failure 404 {object} map[string]string "error: Service not found"
// @Failure 500 {object} map[string]string "error: Failed to update service"
// @Router /api/mcmp-apis/name/{serviceName} [put]
// @Id UpdateFrameworkService
// @Security BearerAuth
func (h *McmpApiHandler) UpdateFrameworkService(c echo.Context) error {
	serviceName := c.Param("serviceName")
	if serviceName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Service name is required"})
	}

	updates := make(map[string]interface{})
	if err := c.Bind(&updates); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
	}

	// Call the service method to update the service
	err := h.service.UpdateService(serviceName, updates)
	if err != nil {
		// Handle specific errors like "service not found" or "no updatable fields"
		if errors.Is(err, errors.New("service not found")) || errors.Is(err, errors.New("no updatable fields provided")) {
			// Use StatusNotFound for "service not found", StatusBadRequest for "no fields"
			status := http.StatusInternalServerError // Default
			if err.Error() == "service not found" {
				status = http.StatusNotFound
			} else if err.Error() == "no updatable fields provided" {
				status = http.StatusBadRequest
			}
			return c.JSON(status, map[string]string{"error": err.Error()})
		}
		// Handle other potential errors
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to update service: " + err.Error()})
	}

	// Optionally fetch and return the updated service? For now, just success message.
	return c.JSON(http.StatusOK, map[string]string{"message": fmt.Sprintf("Service '%s' updated successfully", serviceName)})
}

