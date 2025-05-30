package service

import (
	"bytes" // For request body
	"context"
	"encoding/base64" // For Basic Auth encoding
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil" // For dumping request
	"net/url"           // For query parameter mapping
	"os"
	"path/filepath" // Ensure filepath is imported
	"strings"       // Ensure strings is imported

	// "encoding/json" // Removed unused import

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/model/mcmpapi" // Updated import path
	"github.com/m-cmp/mc-iam-manager/repository"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm" // Needed if passing db directly, but better via repo
)

const apiYamlEnvVar = "MCADMINCLI_APIYAML" // Re-add constant definition

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
}

// mcmpApiService implements the McmpApiService interface.
type mcmpApiService struct {
	db   *gorm.DB
	repo repository.McmpApiRepository
}

// NewMcmpApiService creates a new McmpApiService.
func NewMcmpApiService(db *gorm.DB) McmpApiService {
	repo := repository.NewMcmpApiRepository(db)
	return &mcmpApiService{db: db, repo: repo}
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

// SyncMcmpAPIsFromYAML loads API definitions from the YAML URL specified by env var
// and saves them to the database via the repository.
func (s *mcmpApiService) SyncMcmpAPIsFromYAML() error {
	// 테이블이 없으면 생성
	var count int64
	if err := s.db.Table("mcmp_api_services").Count(&count).Error; err != nil {
		// 테이블이 없으면 생성
		if err := s.db.AutoMigrate(&mcmpapi.McmpApiService{}, &mcmpapi.McmpApiAction{}); err != nil {
			return fmt.Errorf("failed to create mcmp API tables: %w", err)
		}
		log.Printf("Created mcmp API tables")
	}

	yamlSource := os.Getenv(apiYamlEnvVar)
	if yamlSource == "" {
		err := fmt.Errorf("environment variable %s is not set", apiYamlEnvVar)
		log.Printf("Error syncing mcmp APIs: %v", err)
		return err
	}

	localYamlPath := filepath.Join("asset", "mcmpapi", "mcmp_api.yaml")

	// Check if yamlSource is a URL
	if strings.HasPrefix(yamlSource, "http://") || strings.HasPrefix(yamlSource, "https://") {
		log.Printf("Starting mcmp API sync: Downloading from URL %s to %s", yamlSource, localYamlPath)

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(localYamlPath), 0755); err != nil {
			err = fmt.Errorf("failed to create directory for local YAML file %s: %w", localYamlPath, err)
			log.Printf("Error syncing mcmp APIs: %v", err)
			return err
		}

		// Download the file
		resp, err := http.Get(yamlSource)
		if err != nil {
			err = fmt.Errorf("failed to fetch mcmp API YAML from %s: %w", yamlSource, err)
			log.Printf("Error syncing mcmp APIs: %v", err)
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("failed to fetch mcmp API YAML: status code %d", resp.StatusCode)
			log.Printf("Error syncing mcmp APIs: %v", err)
			return err
		}

		// Create the local file
		outFile, err := os.Create(localYamlPath)
		if err != nil {
			err = fmt.Errorf("failed to create local YAML file %s: %w", localYamlPath, err)
			log.Printf("Error syncing mcmp APIs: %v", err)
			return err
		}
		defer outFile.Close()

		// Write the body to the file
		_, err = io.Copy(outFile, resp.Body)
		if err != nil {
			err = fmt.Errorf("failed to write downloaded content to %s: %w", localYamlPath, err)
			log.Printf("Error syncing mcmp APIs: %v", err)
			return err
		}
		log.Printf("Successfully downloaded YAML to %s", localYamlPath)
	} else {
		// Assume yamlSource is a local file path relative to project root
		log.Printf("Starting mcmp API sync: Using local file path %s", yamlSource)
		localYamlPath = yamlSource
	}

	// Read the local YAML file
	log.Printf("Reading mcmp API definitions from %s", localYamlPath)
	yamlData, err := os.ReadFile(localYamlPath)
	if err != nil {
		err = fmt.Errorf("failed to read local mcmp API YAML file %s: %w", localYamlPath, err)
		log.Printf("Error syncing mcmp APIs: %v", err)
		return err
	}

	var defs mcmpapi.McmpApiDefinitions
	err = yaml.Unmarshal(yamlData, &defs)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal mcmp API YAML: %w", err)
		log.Printf("Error syncing mcmp APIs: %v", err)
		return err
	}

	// Save to Database
	err = s.saveDefinitionsToDB(&defs)
	if err != nil {
		log.Printf("Error saving mcmp API definitions to database: %v", err)
		return fmt.Errorf("failed to save mcmp API definitions to DB: %w", err)
	}

	log.Println("Successfully synced mcmp API definitions to database.")
	return nil
}

// saveDefinitionsToDB handles the transaction and logic for saving definitions.
func (s *mcmpApiService) saveDefinitionsToDB(defs *mcmpapi.McmpApiDefinitions) error {
	if defs == nil || len(defs.Services) == 0 {
		log.Println("No service definitions provided to save.")
		return nil // Nothing to save
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r) // Re-panic after rollback
		}
	}()

	for name, serviceDef := range defs.Services {
		// Check if service with the same name and version already exists (using non-transactional DB read is okay here)
		_, err := s.repo.GetServiceByNameAndVersion(name, serviceDef.Version)

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Service with this name and version does not exist, create it within the transaction
			log.Printf("Adding new service definition: %s (Version: %s)", name, serviceDef.Version)
			dbService := mcmpapi.McmpApiService{
				Name:     name,
				Version:  serviceDef.Version,
				BaseURL:  serviceDef.BaseURL,
				AuthType: serviceDef.Auth.Type,
				AuthUser: serviceDef.Auth.Username,
				AuthPass: serviceDef.Auth.Password, // Consider encryption
				// IsActive defaults to false or needs explicit handling if required
			}
			if createErr := s.repo.CreateService(tx, &dbService); createErr != nil {
				tx.Rollback()
				return fmt.Errorf("error creating service %s (Version: %s): %w", name, serviceDef.Version, createErr)
			}

			// Add Actions ONLY for the newly created service version
			if actions, ok := defs.ServiceActions[name]; ok {
				log.Printf("Adding actions for new service: %s (Version: %s)", name, serviceDef.Version)
				for actionName, actionDef := range actions {
					dbAction := mcmpapi.McmpApiAction{
						ServiceName:  name, // Link to the service name
						ActionName:   actionName,
						Method:       actionDef.Method,
						ResourcePath: actionDef.ResourcePath,
						Description:  actionDef.Description,
						// Version linking might be needed here if actions are version-specific
					}
					if createActionErr := s.repo.CreateAction(tx, &dbAction); createActionErr != nil {
						tx.Rollback()
						return fmt.Errorf("error creating action %s for service %s: %w", actionName, name, createActionErr)
					}
				}
			}
		} else if err != nil {
			// Other DB error during check
			tx.Rollback()
			return fmt.Errorf("error checking existing service %s (Version: %s): %w", name, serviceDef.Version, err)
		} else {
			// Service with this name and version already exists, skip.
			log.Printf("Skipping existing service definition: %s (Version: %s)", name, serviceDef.Version)
		}
	}

	if err := tx.Commit().Error; err != nil {
		// Rollback might have already happened, but doesn't hurt to call again if needed
		// tx.Rollback()
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil // Commit successful
}

// Implement other methods like GetMcmpService, GetMcmpAction if needed,
// likely by calling corresponding repository methods.
// Example:
// func (s *mcmpApiService) GetMcmpService(name string) (*mcmpapi.McmpApiService, error) { // Renamed
// 	return s.repo.GetService(name) // Assumes repo method exists
// }

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

// Removed ServiceApiCall function implementation
