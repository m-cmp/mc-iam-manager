package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/model/mcmpapi"
	"github.com/m-cmp/mc-iam-manager/pkg/apiparser"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

// Local path to service-actions.yaml file (relative to working directory)
const localServiceActionsPath = "asset/mcmpapi/service-actions.yaml"

// McmpApiService defines the interface for mcmp API operations
type McmpApiService interface {
	GetDB() *gorm.DB
	GetServiceByNameAndVersion(name, version string) (*mcmpapi.McmpApiService, error)
	CreateService(tx *gorm.DB, service *mcmpapi.McmpApiService) error
	CreateAction(tx *gorm.DB, action *mcmpapi.McmpApiAction) error
	SetActiveVersion(serviceName, version string) error
	GetAllAPIDefinitions(serviceNameFilter, actionNameFilter string) (*mcmpapi.McmpApiDefinitions, error)
	UpdateService(serviceName string, updates map[string]interface{}) error
	McmpApiCall(ctx context.Context, req *model.McmpApiCallRequest) (int, []byte, string, string, error)
	SyncMcmpAPIsFromYAML() error
	ImportAPIs(req *model.ImportApiRequest) (*model.ImportApiResponse, error)
}

// mcmpApiService implements the McmpApiService interface.
type mcmpApiService struct {
	db   *gorm.DB
	repo repository.McmpApiRepository
}

// NewMcmpApiService creates a new McmpApiService.
func NewMcmpApiService(db *gorm.DB) McmpApiService {
	log.Printf("Creating new McmpApiService with db: %v", db)
	repo := repository.NewMcmpApiRepository(db)
	if repo == nil {
		log.Printf("Error: NewMcmpApiRepository returned nil")
		return nil
	}
	service := &mcmpApiService{db: db, repo: repo}
	log.Printf("Successfully created McmpApiService: %v", service)
	return service
}

// GetDB returns the database instance.
func (s *mcmpApiService) GetDB() *gorm.DB {
	return s.db
}

// GetServiceByNameAndVersion finds a specific version of a service.
func (s *mcmpApiService) GetServiceByNameAndVersion(name, version string) (*mcmpapi.McmpApiService, error) {
	return s.repo.GetServiceByNameAndVersion(name, version)
}

// CreateService creates a new service record within a transaction.
func (s *mcmpApiService) CreateService(tx *gorm.DB, service *mcmpapi.McmpApiService) error {
	return s.repo.CreateService(tx, service)
}

// CreateAction creates a new action record within a transaction.
func (s *mcmpApiService) CreateAction(tx *gorm.DB, action *mcmpapi.McmpApiAction) error {
	return s.repo.CreateAction(tx, action)
}

// SyncMcmpAPIsFromYAML loads API definitions from local service-actions.yaml file
// and saves them to the database via the repository with upsert logic.
func (s *mcmpApiService) SyncMcmpAPIsFromYAML() error {
	// Ensure tables exist
	if err := s.ensureTables(); err != nil {
		return fmt.Errorf("failed to ensure tables: %w", err)
	}

	// Read from local file only (no URL download)
	log.Printf("Reading MCMP API definitions from local file: %s", localServiceActionsPath)
	yamlData, err := os.ReadFile(localServiceActionsPath)
	if err != nil {
		return fmt.Errorf("failed to read service-actions.yaml from %s: %w", localServiceActionsPath, err)
	}

	// Parse the YAML structure with serviceActions containing _meta
	var rawData map[string]interface{}
	if err := yaml.Unmarshal(yamlData, &rawData); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	serviceActionsRaw, ok := rawData["serviceActions"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid YAML format: missing or invalid serviceActions")
	}

	return s.syncServicesAndActions(serviceActionsRaw)
}

// ensureTables creates MCMP API tables if they don't exist
func (s *mcmpApiService) ensureTables() error {
	var count int64
	if err := s.db.Table("mcmp_api_services").Count(&count).Error; err != nil {
		// Table doesn't exist, create it
		if err := s.db.AutoMigrate(&mcmpapi.McmpApiService{}, &mcmpapi.McmpApiAction{}, &mcmpapi.McmpApiServiceMeta{}); err != nil {
			return fmt.Errorf("failed to create mcmp API tables: %w", err)
		}
		log.Printf("Created mcmp API tables")
	}
	return nil
}

// syncServicesAndActions processes service actions from parsed YAML and syncs to database
func (s *mcmpApiService) syncServicesAndActions(serviceActionsRaw map[string]interface{}) error {
	tx := s.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	for serviceName, actionsRaw := range serviceActionsRaw {
		actionsMap, ok := actionsRaw.(map[string]interface{})
		if !ok {
			log.Printf("Warning: invalid actions format for service %s, skipping", serviceName)
			continue
		}

		// Extract _meta
		meta, err := s.extractMeta(serviceName, actionsMap)
		if err != nil {
			log.Printf("Warning: failed to extract meta for %s: %v", serviceName, err)
		}

		// Check if update is needed
		shouldUpdate, err := s.shouldUpdateService(serviceName, meta)
		if err != nil {
			log.Printf("Error checking service %s: %v", serviceName, err)
		}

		if shouldUpdate {
			// Upsert service meta
			if meta != nil {
				if err := s.repo.UpsertServiceMeta(tx, meta); err != nil {
					tx.Rollback()
					return fmt.Errorf("failed to upsert meta for %s: %w", serviceName, err)
				}
			}

			// Delete existing actions for this service (full replace)
			if err := s.repo.DeleteActionsByServiceName(tx, serviceName); err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to delete actions for %s: %w", serviceName, err)
			}

			// Create new actions
			actionCount := 0
			for actionName, actionRaw := range actionsMap {
				if actionName == "_meta" {
					continue // Skip meta entry
				}

				actionDef, ok := actionRaw.(map[string]interface{})
				if !ok {
					log.Printf("Warning: invalid action format for %s/%s, skipping", serviceName, actionName)
					continue
				}

				action := &mcmpapi.McmpApiAction{
					ServiceName:  serviceName,
					ActionName:   actionName,
					Method:       s.getString(actionDef, "method"),
					ResourcePath: s.getString(actionDef, "resourcePath"),
					Description:  s.getString(actionDef, "description"),
				}

				if err := s.repo.CreateAction(tx, action); err != nil {
					tx.Rollback()
					return fmt.Errorf("failed to create action %s for %s: %w", actionName, serviceName, err)
				}
				actionCount++
			}

			version := ""
			if meta != nil {
				version = meta.Version
			}
			log.Printf("Updated service: %s (version: %s, actions: %d)", serviceName, version, actionCount)
		} else {
			log.Printf("Skipping service %s - no changes detected", serviceName)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Println("Successfully synced mcmp API definitions to database.")
	return nil
}

// extractMeta extracts _meta information from service actions map
func (s *mcmpApiService) extractMeta(serviceName string, actionsMap map[string]interface{}) (*mcmpapi.McmpApiServiceMeta, error) {
	metaRaw, ok := actionsMap["_meta"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no _meta found for service %s", serviceName)
	}

	generatedAtStr := s.getString(metaRaw, "generatedAt")
	var generatedAt time.Time
	if generatedAtStr != "" {
		parsed, err := time.Parse(time.RFC3339, generatedAtStr)
		if err != nil {
			log.Printf("Warning: failed to parse generatedAt for %s: %v", serviceName, err)
		} else {
			generatedAt = parsed
		}
	}

	return &mcmpapi.McmpApiServiceMeta{
		ServiceName: serviceName,
		Version:     s.getString(metaRaw, "version"),
		Repository:  s.getString(metaRaw, "repository"),
		GeneratedAt: generatedAt,
	}, nil
}

// shouldUpdateService checks if a service needs to be updated based on version metadata
func (s *mcmpApiService) shouldUpdateService(serviceName string, newMeta *mcmpapi.McmpApiServiceMeta) (bool, error) {
	if newMeta == nil {
		return true, nil // No meta, always update
	}

	existingMeta, err := s.repo.GetServiceMeta(serviceName)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true, nil // New service
	}
	if err != nil {
		return false, err
	}

	// Compare version and generatedAt
	if existingMeta.Version != newMeta.Version {
		log.Printf("Service %s version changed: %s -> %s", serviceName, existingMeta.Version, newMeta.Version)
		return true, nil
	}
	if !existingMeta.GeneratedAt.Equal(newMeta.GeneratedAt) {
		log.Printf("Service %s generatedAt changed: %v -> %v", serviceName, existingMeta.GeneratedAt, newMeta.GeneratedAt)
		return true, nil
	}

	return false, nil // No changes
}

// getString safely extracts a string value from a map
func (s *mcmpApiService) getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// SetActiveVersion sets the specified version of a service as active.
func (s *mcmpApiService) SetActiveVersion(serviceName, version string) error {
	return s.repo.SetActiveVersion(serviceName, version)
}

// GetAllAPIDefinitions retrieves API definitions, optionally filtered.
func (s *mcmpApiService) GetAllAPIDefinitions(serviceNameFilter, actionNameFilter string) (*mcmpapi.McmpApiDefinitions, error) { // Added filters
	return s.repo.GetAllAPIDefinitions(serviceNameFilter, actionNameFilter) // Pass filters to repo
}

// UpdateService updates an existing MCMP API service definition.
func (s *mcmpApiService) UpdateService(serviceName string, updates map[string]interface{}) error {
	// Add validation if needed (e.g., check allowed fields, data types)
	// Ensure critical fields like 'name' or 'version' are not in the updates map if they shouldn't be changed.
	delete(updates, "name")    // Prevent changing the name via update
	delete(updates, "version") // Prevent changing the version via update

	if len(updates) == 0 {
		return errors.New("no updatable fields provided")
	}

	// Optional: Check if the service exists before attempting update
	// _, err := s.repo.GetService(serviceName) // This might fetch the active one, need GetServiceByName?
	// if err != nil {
	// 	 return errors.New("service not found")
	// }

	return s.repo.UpdateService(serviceName, updates)
}

// McmpApiCall executes a call to an external MCMP API based on stored definitions and provided parameters. (Renamed)
func (s *mcmpApiService) McmpApiCall(ctx context.Context, req *model.McmpApiCallRequest) (int, []byte, string, string, error) { // Renamed method and added return values
	// Initialize return values for error cases
	serviceVersion := ""
	calledURL := ""

	// 1. Get Service Info (use active version)
	serviceInfo, err := s.repo.GetService(req.ServiceName)
	if err != nil {
		log.Printf("Error getting active service '%s': %v", req.ServiceName, err)
		// Return default values for version and URL on error
		err = fmt.Errorf("active service '%s' not found or error fetching: %w", req.ServiceName, err)
		return http.StatusNotFound, nil, serviceVersion, calledURL, err
	}
	serviceVersion = serviceInfo.Version // Assign version early

	// 2. Get Action Info
	actionInfo, err := s.repo.GetServiceAction(req.ServiceName, req.ActionName)
	if err != nil {
		log.Printf("Error getting action '%s' for service '%s': %v", req.ActionName, req.ServiceName, err)
		// Return default values for version and URL on error
		err = fmt.Errorf("action '%s' not found for service '%s' or error fetching: %w", req.ActionName, req.ServiceName, err)
		return http.StatusNotFound, nil, serviceVersion, calledURL, err
	}

	// 3. Build URL with Path and Query Params
	targetURL := serviceInfo.BaseURL
	endpointPath := actionInfo.ResourcePath

	// Replace path parameters
	processedPath := endpointPath
	if req.RequestParams.PathParams != nil {
		for key, val := range req.RequestParams.PathParams {
			processedPath = strings.Replace(processedPath, "{"+key+"}", url.PathEscape(val), -1)
		}
	}

	// Append path to base URL (handle slashes)
	if !strings.HasSuffix(targetURL, "/") && !strings.HasPrefix(processedPath, "/") {
		targetURL += "/"
	} else if strings.HasSuffix(targetURL, "/") && strings.HasPrefix(processedPath, "/") {
		processedPath = strings.TrimPrefix(processedPath, "/")
	}
	finalURL := targetURL + processedPath

	// Add query parameters
	parsedURL, err := url.Parse(finalURL)
	if err != nil {
		log.Printf("Error parsing final URL '%s': %v", finalURL, err)
		err = fmt.Errorf("error constructing request URL: %w", err)
		return http.StatusInternalServerError, nil, serviceVersion, calledURL, err // Return finalURL even on parse error? Or empty? Let's return it.
	}
	query := parsedURL.Query()
	if req.RequestParams.QueryParams != nil {
		for key, val := range req.RequestParams.QueryParams {
			query.Set(key, val)
		}
	}
	parsedURL.RawQuery = query.Encode()
	requestUrlStr := parsedURL.String()
	calledURL = requestUrlStr // Assign called URL

	// 4. Prepare Request Body
	var requestBody io.Reader
	if req.RequestParams.Body != nil {
		// Marshal the interface{} back to JSON bytes
		bodyBytes, err := json.Marshal(req.RequestParams.Body)
		if err != nil {
			log.Printf("Error marshalling request body for %s/%s: %v", req.ServiceName, req.ActionName, err)
			err = fmt.Errorf("error preparing request body: %w", err)
			return http.StatusInternalServerError, nil, serviceVersion, calledURL, err
		}
		requestBody = bytes.NewBuffer(bodyBytes)
	} else {
		requestBody = nil // No body provided
	}

	// 5. Create HTTP Request
	httpReq, err := http.NewRequestWithContext(ctx, strings.ToUpper(actionInfo.Method), requestUrlStr, requestBody)
	if err != nil {
		log.Printf("Error creating HTTP request for %s/%s: %v", req.ServiceName, req.ActionName, err)
		err = fmt.Errorf("error creating request: %w", err)
		return http.StatusInternalServerError, nil, serviceVersion, calledURL, err
	}

	// Set Content-Type if body exists
	if requestBody != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	// Set Accept header explicitly
	httpReq.Header.Set("Accept", "application/json")
	// Remove Accept-Encoding header to prevent potential issues with gzip
	httpReq.Header.Del("Accept-Encoding")

	// 6. Add Authentication Header based on serviceInfo
	if serviceInfo.AuthType == "basic" && serviceInfo.AuthUser != "" {
		// Construct Basic Auth header manually using AuthUser and AuthPass from DB
		auth := serviceInfo.AuthUser + ":" + serviceInfo.AuthPass
		basicAuthHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
		httpReq.Header.Add("Authorization", basicAuthHeader)
		log.Printf("Added Basic Authentication header for service %s", req.ServiceName)
	} else if serviceInfo.AuthType != "" && serviceInfo.AuthType != "none" {
		// Handle other potential auth types here if needed in the future
		// Example: Bearer token might be stored in AuthPass
		// if serviceInfo.AuthType == "bearer" && serviceInfo.AuthPass != "" {
		// 	httpReq.Header.Add("Authorization", "Bearer "+serviceInfo.AuthPass)
		// }
		// For now, only basic auth is handled explicitly
		log.Printf("Warning: Unsupported auth type '%s' for service %s", serviceInfo.AuthType, req.ServiceName)
	}
	// Add auth header if manually constructed
	// if authHeader != "" {
	// 	httpReq.Header.Add("Authorization", authHeader)
	// }

	// 7. Dump Request for Debugging
	requestDump, dumpErr := httputil.DumpRequestOut(httpReq, true)
	if dumpErr != nil {
		log.Printf("Error dumping request for %s/%s: %v", req.ServiceName, req.ActionName, dumpErr)
	} else {
		log.Printf("Calling External API:\n%s\n", string(requestDump))
	}

	// 8. Execute Request
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("Error executing request for %s/%s: %v", req.ServiceName, req.ActionName, err)
		// Return 503 Service Unavailable for network errors?
		err = fmt.Errorf("error calling external API: %w", err)
		return http.StatusServiceUnavailable, nil, serviceVersion, calledURL, err
	}
	defer resp.Body.Close()

	// 9. Read Response Body
	respBody, ioerr := io.ReadAll(resp.Body) // Assign to the named return variable
	if ioerr != nil {
		log.Printf("Error reading response body for %s/%s: %v", req.ServiceName, req.ActionName, ioerr)
		// Return the status code but indicate body read error
		err = fmt.Errorf("error reading response body: %w", ioerr)
		return resp.StatusCode, nil, serviceVersion, calledURL, err
	}

	log.Printf("External API Call Response: Status=%s, Body Length=%d", resp.Status, len(respBody))

	// Return status code, raw body, version, URL, and nil error (even for non-2xx responses)
	// The handler will decide how to interpret the status code.
	statusCode := resp.StatusCode // Assign to named return variable
	// respBody is already assigned
	// serviceVersion is already assigned
	// calledURL is already assigned
	err = nil // Explicitly set error to nil for success case
	return statusCode, respBody, serviceVersion, calledURL, err
}

// ImportAPIs fetches API specifications from remote URLs and imports them to the database
func (s *mcmpApiService) ImportAPIs(req *model.ImportApiRequest) (*model.ImportApiResponse, error) {
	// Ensure tables exist
	if err := s.ensureTables(); err != nil {
		return nil, fmt.Errorf("failed to ensure tables: %w", err)
	}

	processor := apiparser.NewProcessor(30) // 30 second timeout

	response := &model.ImportApiResponse{
		TotalFrameworks:  len(req.Frameworks),
		FrameworkResults: make([]model.ImportApiFrameworkResult, 0, len(req.Frameworks)),
	}

	for _, fw := range req.Frameworks {
		result := model.ImportApiFrameworkResult{
			Name:    fw.Name,
			Version: fw.Version,
		}

		// Process the framework
		fwResult := processor.ProcessFramework(fw.Name, fw.Version, fw.Repository, fw.SourceType, fw.SourceURL)

		if fwResult.Error != nil {
			result.Success = false
			result.ErrorMessage = fwResult.Error.Error()
			response.FailureCount++
			log.Printf("Failed to import framework %s: %v", fw.Name, fwResult.Error)
		} else {
			// Sync to database - use the new method if service info is provided
			var err error
			if fw.BaseURL != "" {
				// Use the new method that saves service info
				err = s.syncFrameworkWithServiceInfo(fw.Name, fw.Version, fw.Repository, fw.BaseURL, fw.AuthType, fw.AuthUser, fw.AuthPass, fwResult.Actions)
			} else {
				// Use the old method (meta + actions only)
				err = s.syncFrameworkToDatabase(fw.Name, fw.Version, fw.Repository, fwResult.Actions)
				log.Printf("Warning: Framework '%s' imported without service info (baseUrl not provided)", fw.Name)
			}

			if err != nil {
				result.Success = false
				result.ErrorMessage = fmt.Sprintf("failed to save to database: %v", err)
				response.FailureCount++
				log.Printf("Failed to save framework %s to database: %v", fw.Name, err)
			} else {
				result.Success = true
				result.ActionCount = fwResult.ActionCount
				response.SuccessCount++
				log.Printf("Successfully imported framework %s (version: %s, actions: %d)", fw.Name, fw.Version, fwResult.ActionCount)
			}
		}

		response.FrameworkResults = append(response.FrameworkResults, result)
	}

	return response, nil
}

// syncFrameworkToDatabase saves a single framework's actions to the database
func (s *mcmpApiService) syncFrameworkToDatabase(serviceName, version, repository string, actions map[string]apiparser.ServiceAction) error {
	tx := s.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Upsert service meta
	meta := &mcmpapi.McmpApiServiceMeta{
		ServiceName: serviceName,
		Version:     version,
		Repository:  repository,
		GeneratedAt: time.Now(),
	}
	if err := s.repo.UpsertServiceMeta(tx, meta); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to upsert meta: %w", err)
	}

	// Delete existing actions for this service (full replace)
	if err := s.repo.DeleteActionsByServiceName(tx, serviceName); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete existing actions: %w", err)
	}

	// Create new actions
	for actionName, actionDef := range actions {
		action := &mcmpapi.McmpApiAction{
			ServiceName:  serviceName,
			ActionName:   actionName,
			Method:       actionDef.Method,
			ResourcePath: actionDef.ResourcePath,
			Description:  actionDef.Description,
		}

		if err := s.repo.CreateAction(tx, action); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create action %s: %w", actionName, err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// syncFrameworkWithServiceInfo saves a framework's actions and service info to the database
func (s *mcmpApiService) syncFrameworkWithServiceInfo(serviceName, version, repository, baseURL, authType, authUser, authPass string, actions map[string]apiparser.ServiceAction) error {
	tx := s.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Upsert service meta
	meta := &mcmpapi.McmpApiServiceMeta{
		ServiceName: serviceName,
		Version:     version,
		Repository:  repository,
		GeneratedAt: time.Now(),
	}
	if err := s.repo.UpsertServiceMeta(tx, meta); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to upsert meta: %w", err)
	}

	// Upsert service (with BaseURL and Auth info)
	service := &mcmpapi.McmpApiService{
		Name:     serviceName,
		Version:  version,
		BaseURL:  baseURL,
		AuthType: authType,
		AuthUser: authUser,
		AuthPass: authPass,
		IsActive: true,
	}
	if err := s.repo.UpsertService(tx, service); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to upsert service: %w", err)
	}

	// Delete existing actions for this service (full replace)
	if err := s.repo.DeleteActionsByServiceName(tx, serviceName); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete existing actions: %w", err)
	}

	// Create new actions
	for actionName, actionDef := range actions {
		action := &mcmpapi.McmpApiAction{
			ServiceName:  serviceName,
			ActionName:   actionName,
			Method:       actionDef.Method,
			ResourcePath: actionDef.ResourcePath,
			Description:  actionDef.Description,
		}

		if err := s.repo.CreateAction(tx, action); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create action %s: %w", actionName, err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
