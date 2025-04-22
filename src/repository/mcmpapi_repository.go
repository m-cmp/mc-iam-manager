package repository

import (
	"errors"
	"fmt"
	"log"

	// Import strings for ToLower
	"github.com/m-cmp/mc-iam-manager/model/mcmpapi" // Updated import path
	"gorm.io/gorm"
)

// McmpApiRepository 인터페이스 정의 (Renamed)
type McmpApiRepository interface {
	// SaveAPIDefinitions removed - logic moved to service
	GetServiceByNameAndVersion(name, version string) (*mcmpapi.McmpApiService, error)                     // Added for service logic
	CreateService(tx *gorm.DB, service *mcmpapi.McmpApiService) error                                     // Added for service logic
	CreateAction(tx *gorm.DB, action *mcmpapi.McmpApiAction) error                                        // Added for service logic
	GetAllAPIDefinitions(serviceNameFilter, actionNameFilter string) (*mcmpapi.McmpApiDefinitions, error) // Updated signature with filters
	GetService(serviceName string) (*mcmpapi.McmpApiService, error)                                       // Updated types
	GetServiceAction(serviceName, actionName string) (*mcmpapi.McmpApiAction, error)                      // Updated types
	SetActiveServiceVersion(serviceName, version string) error                                            // Added method signature
	GetActiveService(serviceName string) (*mcmpapi.McmpApiService, error)                                 // Added method signature
	UpdateService(serviceName string, updates map[string]interface{}) error                               // Added method signature for update
}

// mcmpApiRepository 구조체 정의 (Renamed)
type mcmpApiRepository struct {
	db *gorm.DB
}

// NewMcmpApiRepository 생성자 함수 (Renamed)
func NewMcmpApiRepository(db *gorm.DB) McmpApiRepository { // Renamed return type
	// AutoMigrate: 테이블이 없으면 생성 (개발 환경에서 유용)
	// 주의: 프로덕션 환경에서는 별도의 마이그레이션 도구 사용 권장
	// Ensure correct model types are used for AutoMigrate
	err := db.AutoMigrate(&mcmpapi.McmpApiService{}, &mcmpapi.McmpApiAction{})
	if err != nil {
		log.Printf("Warning: Failed to auto-migrate mcmp API tables: %v", err) // Updated log message
		// 실패해도 일단 리포지토리 인스턴스는 반환
	}
	return &mcmpApiRepository{db: db} // Renamed struct type
}

// GetServiceByNameAndVersion finds a specific version of a service.
func (r *mcmpApiRepository) GetServiceByNameAndVersion(name, version string) (*mcmpapi.McmpApiService, error) {
	var service mcmpapi.McmpApiService
	err := r.db.Where("name = ? AND version = ?", name, version).First(&service).Error
	if err != nil {
		// Return error directly (including gorm.ErrRecordNotFound)
		return nil, err
	}
	return &service, nil
}

// CreateService creates a new service record within a transaction.
func (r *mcmpApiRepository) CreateService(tx *gorm.DB, service *mcmpapi.McmpApiService) error {
	if err := tx.Create(service).Error; err != nil {
		log.Printf("Error creating service %s (Version: %s) in transaction: %v", service.Name, service.Version, err)
		return err
	}
	return nil
}

// CreateAction creates a new action record within a transaction.
func (r *mcmpApiRepository) CreateAction(tx *gorm.DB, action *mcmpapi.McmpApiAction) error {
	// Consider using FirstOrCreate if actions might be duplicated across service versions but should only exist once per service name.
	// For now, assume Create is sufficient as it's called only when a new service version is created.
	if err := tx.Create(action).Error; err != nil {
		log.Printf("Error creating action %s for service %s in transaction: %v", action.ActionName, action.ServiceName, err)
		return err
	}
	return nil
}

// GetAllAPIDefinitions retrieves API definitions from the DB, optionally filtered by serviceName and actionName.
func (r *mcmpApiRepository) GetAllAPIDefinitions(serviceNameFilter, actionNameFilter string) (*mcmpapi.McmpApiDefinitions, error) { // Added filters
	var dbServices []mcmpapi.McmpApiService
	var dbActions []mcmpapi.McmpApiAction

	serviceQuery := r.db
	actionQuery := r.db

	// Apply service name filter if provided
	if serviceNameFilter != "" {
		serviceQuery = serviceQuery.Where("name = ?", serviceNameFilter)
		actionQuery = actionQuery.Where("service_name = ?", serviceNameFilter)
	}

	// Apply action name filter if provided
	// Note: This filters actions directly. Services without matching actions might still be returned if serviceNameFilter is not set.
	if actionNameFilter != "" {
		actionQuery = actionQuery.Where("action_name = ?", actionNameFilter)
		// If filtering only by actionName, we might need to adjust the service query
		// to only include services that *have* this action. This requires a join or subquery.
		// For simplicity now, filter actions first, then filter services based on remaining action service names.
	}

	// Fetch filtered actions first if actionNameFilter is present
	if actionNameFilter != "" {
		if err := actionQuery.Find(&dbActions).Error; err != nil {
			return nil, fmt.Errorf("error fetching filtered actions: %w", err)
		}
		// If filtering only by action name, get the relevant service names
		if serviceNameFilter == "" && len(dbActions) > 0 {
			serviceNames := []string{}
			seenServices := make(map[string]bool)
			for _, action := range dbActions {
				if !seenServices[action.ServiceName] {
					serviceNames = append(serviceNames, action.ServiceName)
					seenServices[action.ServiceName] = true
				}
			}
			if len(serviceNames) > 0 {
				serviceQuery = serviceQuery.Where("name IN ?", serviceNames)
			} else {
				// No actions found matching the filter, so no services needed
				dbServices = []mcmpapi.McmpApiService{}
				defs := &mcmpapi.McmpApiDefinitions{
					Services:       make(map[string]mcmpapi.McmpApiServiceDefinition),
					ServiceActions: make(map[string]map[string]mcmpapi.McmpApiServiceAction),
				}
				return defs, nil // Return empty definitions
			}
		}
	}

	// Fetch services (potentially filtered)
	if err := serviceQuery.Find(&dbServices).Error; err != nil {
		return nil, fmt.Errorf("error fetching services: %w", err)
	}

	// Fetch actions if not already fetched (i.e., if actionNameFilter was empty)
	if actionNameFilter == "" {
		if err := actionQuery.Find(&dbActions).Error; err != nil {
			return nil, fmt.Errorf("error fetching actions: %w", err)
		}
	}

	defs := &mcmpapi.McmpApiDefinitions{ // Use renamed definitions struct
		Services:       make(map[string]mcmpapi.McmpApiServiceDefinition),        // Use renamed service definition
		ServiceActions: make(map[string]map[string]mcmpapi.McmpApiServiceAction), // Use renamed service action
	}

	for _, dbService := range dbServices {
		defs.Services[dbService.Name] = mcmpapi.McmpApiServiceDefinition{ // Use renamed service definition
			Version: dbService.Version,
			BaseURL: dbService.BaseURL,
			Auth: mcmpapi.McmpApiAuthInfo{ // Use renamed auth info
				Type:     dbService.AuthType,
				Username: dbService.AuthUser,
				Password: dbService.AuthPass,
			},
		}
	}

	for _, dbAction := range dbActions {
		if _, ok := defs.ServiceActions[dbAction.ServiceName]; !ok {
			defs.ServiceActions[dbAction.ServiceName] = make(map[string]mcmpapi.McmpApiServiceAction) // Use renamed service action
		}
		defs.ServiceActions[dbAction.ServiceName][dbAction.ActionName] = mcmpapi.McmpApiServiceAction{ // Use renamed service action
			Method:       dbAction.Method,
			ResourcePath: dbAction.ResourcePath,
			Description:  dbAction.Description,
		}
	}

	return defs, nil
}

// GetService DB에서 특정 서비스 정보를 조회
func (r *mcmpApiRepository) GetService(serviceName string) (*mcmpapi.McmpApiService, error) { // Renamed receiver and return type
	var service mcmpapi.McmpApiService // Use renamed service model
	result := r.db.Where("name = ?", serviceName).First(&service)
	if result.Error != nil {
		return nil, result.Error // gorm.ErrRecordNotFound 포함
	}
	return &service, nil
}

// UpdateService updates specific fields of a service identified by its name.
// Note: This updates ALL versions of the service with the same name.
// If version-specific updates are needed, the primary key or query needs adjustment.
func (r *mcmpApiRepository) UpdateService(serviceName string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return errors.New("no fields provided for update")
	}
	// Ensure non-updatable fields are not included (like name, version, created_at)
	delete(updates, "name")
	delete(updates, "version")
	delete(updates, "created_at")
	delete(updates, "serviceName") // Explicitly remove serviceName if present

	if len(updates) == 0 {
		return errors.New("no updatable fields provided")
	}

	// Ensure the map uses correct DB column names if necessary,
	// but GORM usually handles mapping from struct field names in Updates.
	// The issue was likely the presence of a key not matching a column.
	result := r.db.Model(&mcmpapi.McmpApiService{}).Where("name = ?", serviceName).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// This could mean the service name doesn't exist
		return errors.New("service not found") // Or return gorm.ErrRecordNotFound?
	}
	return nil
}

// GetServiceAction DB에서 특정 서비스의 특정 액션 정보를 조회 (대소문자 구분 없이)
func (r *mcmpApiRepository) GetServiceAction(serviceName, actionName string) (*mcmpapi.McmpApiAction, error) { // Renamed receiver and return type
	var action mcmpapi.McmpApiAction // Use renamed action model
	// Use LOWER() function for case-insensitive comparison on action_name
	result := r.db.Where("service_name = ? AND LOWER(action_name) = LOWER(?)", serviceName, actionName).First(&action)
	if result.Error != nil {
		// Log the error for debugging, especially if it's not ErrRecordNotFound
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			log.Printf("Error fetching action '%s' for service '%s': %v", actionName, serviceName, result.Error)
		}
		return nil, result.Error // gorm.ErrRecordNotFound 포함
	}
	return &action, nil
}

// SetActiveServiceVersion sets the specified version of a service as active,
// and deactivates all other versions of the same service.
func (r *mcmpApiRepository) SetActiveServiceVersion(serviceName, version string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Deactivate all versions for this service name
		if err := tx.Model(&mcmpapi.McmpApiService{}).Where("name = ?", serviceName).Update("is_active", false).Error; err != nil {
			log.Printf("Error deactivating existing versions for service %s: %v", serviceName, err)
			return err
		}

		// Activate the specified version
		result := tx.Model(&mcmpapi.McmpApiService{}).Where("name = ? AND version = ?", serviceName, version).Update("is_active", true)
		if result.Error != nil {
			log.Printf("Error activating version %s for service %s: %v", version, serviceName, result.Error)
			return result.Error
		}
		if result.RowsAffected == 0 {
			log.Printf("Service %s with version %s not found for activation.", serviceName, version)
			return errors.New("service version not found") // Or a more specific error
		}

		log.Printf("Successfully activated version %s for service %s", version, serviceName)
		return nil // Commit transaction
	})
}

// GetActiveService retrieves the currently active version of a service.
func (r *mcmpApiRepository) GetActiveService(serviceName string) (*mcmpapi.McmpApiService, error) {
	var service mcmpapi.McmpApiService
	// Find the service where name matches and is_active is true
	result := r.db.Where("name = ? AND is_active = ?", serviceName, true).First(&service)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Try to find *any* version if no active version is set
			log.Printf("No active version found for service %s, attempting to find latest/any version.", serviceName)
			// Example: Find the latest version based on semantic versioning or created_at
			// For simplicity, just find the first one found by name if no active one exists
			fallbackResult := r.db.Where("name = ?", serviceName).Order("version DESC").First(&service) // Example: order by version desc
			if fallbackResult.Error != nil {
				if errors.Is(fallbackResult.Error, gorm.ErrRecordNotFound) {
					return nil, errors.New("service not found") // No versions found at all
				}
				return nil, fallbackResult.Error // Other DB error during fallback
			}
			log.Printf("Found fallback version %s for service %s.", service.Version, serviceName)
			return &service, nil // Return the found fallback version
		}
		return nil, result.Error // Other DB errors when searching for active
	}
	return &service, nil
}
