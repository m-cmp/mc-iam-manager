package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model/mcmpapi"
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm" // Import gorm
)

// McmpApiHandler handles requests related to mcmp API definitions. (Renamed)
type McmpApiHandler struct {
	service service.McmpApiService // Use renamed service interface
	// db *gorm.DB // Not needed directly in handler
}

// NewMcmpApiHandler creates a new McmpApiHandler. (Renamed)
func NewMcmpApiHandler(db *gorm.DB) *McmpApiHandler { // Accept db, remove service param
	// Initialize service internally
	mcmpApiService := service.NewMcmpApiService(db)
	return &McmpApiHandler{service: mcmpApiService} // Renamed struct type
}

// SyncMcmpAPIs godoc (Renamed)
// @Summary Sync MCMP API Definitions
// @Description Triggers the synchronization of MCMP API definitions from the configured YAML URL to the database.
// @Tags McmpAPI // Updated tag
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "message: Successfully triggered MCMP API sync" // Updated message
// @Failure 500 {object} map[string]string "message: Failed to trigger MCMP API sync" // Updated message
// @Router /mcmp-apis/sync [post] // Updated route suggestion (can be changed in main.go)
// @Security BearerAuth
func (h *McmpApiHandler) SyncMcmpAPIs(c echo.Context) error { // Renamed receiver and method
	err := h.service.SyncMcmpAPIsFromYAML() // Call renamed service method
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to trigger MCMP API sync: " + err.Error()}) // Updated message
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "Successfully triggered MCMP API sync"}) // Updated message
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
// @Router /mcmp-apis/{serviceName}/versions/{version}/activate [put] // Example route
// @Security BearerAuth
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

// McmpApiCall godoc (Existing handler, might be deprecated later)
// @Summary Call an external MCMP API action (Structured Request)
// @Description Executes a defined MCMP API action with parameters structured in McmpApiCallRequest.
// @Tags McmpAPI
// @Accept json
// @Produce json
// @Param callRequest body mcmpapi.McmpApiCallRequest true "API Call Request"
// @Success 200 {object} object "External API Response (structure depends on the called API)"
// @Failure 400 {object} map[string]string "error: Invalid request body or parameters"
// @Failure 404 {object} map[string]string "error: Service or action not found"
// @Failure 500 {object} map[string]string "error: Internal server error or failed to call external API"
// @Failure 503 {object} map[string]string "error: External API unavailable"
// @Router /mcmp-apis/call [post] // Example route (Consider changing if keeping both handlers)
// @Security BearerAuth
func (h *McmpApiHandler) McmpApiCall(c echo.Context) error { // Renamed function
	var req mcmpapi.McmpApiCallRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
	}

	// TODO: Add validation for the request body using validator if needed

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
// @Router /mcmp-apis [get] // Example route
// @Security BearerAuth
func (h *McmpApiHandler) GetAllAPIDefinitions(c echo.Context) error {
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
// @Router /mcmp-apis/test/mc-infra-manager/getallns [get] // Example test route
// @Security BearerAuth
func (h *McmpApiHandler) TestCallGetAllNs(c echo.Context) error {
	// Prepare the request for the CallApi service
	callReq := &mcmpapi.McmpApiCallRequest{
		ServiceName: "mc-infra-manager", // Target service
		ActionName:  "GetAllNs",         // Target action (operationId)
		RequestParams: mcmpapi.McmpApiRequestParams{ // No params needed for GetAllNs
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
// @Router /mcmp-apis/{serviceName} [put] // Example route
// @Security BearerAuth
func (h *McmpApiHandler) UpdateService(c echo.Context) error {
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
