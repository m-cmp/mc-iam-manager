package repository

import (
	"errors"
	"fmt"
	"log"

	"github.com/m-cmp/mc-iam-manager/model/mcmpapi"
	"gorm.io/gorm"
)

// McmpApiRepository defines the interface for mcmp API data access.
type McmpApiRepository interface {
	GetServiceByNameAndVersion(name, version string) (*mcmpapi.McmpApiService, error)
	CreateService(tx *gorm.DB, service *mcmpapi.McmpApiService) error
	CreateAction(tx *gorm.DB, action *mcmpapi.McmpApiAction) error
	GetAllAPIDefinitions(serviceNameFilter, actionNameFilter string) (*mcmpapi.McmpApiDefinitions, error)
	SetActiveVersion(serviceName, version string) error
	UpdateService(serviceName string, updates map[string]interface{}) error
	GetService(serviceName string) (*mcmpapi.McmpApiService, error)
	GetServiceAction(serviceName, actionName string) (*mcmpapi.McmpApiAction, error)
	// New methods for upsert logic and version tracking
	DeleteActionsByServiceName(tx *gorm.DB, serviceName string) error
	GetServiceMeta(serviceName string) (*mcmpapi.McmpApiServiceMeta, error)
	UpsertServiceMeta(tx *gorm.DB, meta *mcmpapi.McmpApiServiceMeta) error
}

// mcmpApiRepository implements the McmpApiRepository interface.
type mcmpApiRepository struct {
	db *gorm.DB
}

// InitializeMcmpApiTables 테이블이 없을 경우에만 생성
func InitializeMcmpApiTables(db *gorm.DB) error {
	// 테이블이 존재하는지 확인
	var count int64
	if err := db.Table("mcmp_api_services").Count(&count).Error; err != nil {
		// 테이블이 없으면 생성
		if err := db.AutoMigrate(&mcmpapi.McmpApiService{}, &mcmpapi.McmpApiAction{}); err != nil {
			return fmt.Errorf("failed to create mcmp API tables: %w", err)
		}
		log.Printf("Created mcmp API tables")
	}
	return nil
}

// NewMcmpApiRepository creates a new McmpApiRepository.
func NewMcmpApiRepository(db *gorm.DB) McmpApiRepository {
	log.Printf("Creating new McmpApiRepository with db: %v", db)
	if db == nil {
		log.Printf("Error: database is nil in NewMcmpApiRepository")
		return nil
	}
	repo := &mcmpApiRepository{db: db}
	log.Printf("Successfully created McmpApiRepository: %v", repo)
	return repo
}

// GetServiceByNameAndVersion finds a specific version of a service.
func (r *mcmpApiRepository) GetServiceByNameAndVersion(name, version string) (*mcmpapi.McmpApiService, error) {
	var service mcmpapi.McmpApiService
	query := r.db.Where("name = ? AND version = ?", name, version).First(&service)
	if err := query.Error; err != nil {
		// 에러 발생 시에만 쿼리 로깅
		sql := query.Statement.SQL.String()
		args := query.Statement.Vars
		log.Printf("GetServiceByNameAndVersion SQL Query (ERROR): %s", sql)
		log.Printf("GetServiceByNameAndVersion SQL Args (ERROR): %v", args)
		return nil, err
	}

	return &service, nil
}

// CreateService creates a new service record within a transaction.
func (r *mcmpApiRepository) CreateService(tx *gorm.DB, service *mcmpapi.McmpApiService) error {
	query := tx.Create(service)
	if err := query.Error; err != nil {
		// 에러 발생 시에만 쿼리 로깅
		sql := query.Statement.SQL.String()
		args := query.Statement.Vars
		log.Printf("CreateService SQL Query (ERROR): %s", sql)
		log.Printf("CreateService SQL Args (ERROR): %v", args)
		log.Printf("Error creating service %s (Version: %s) in transaction: %v", service.Name, service.Version, err)
		return err
	}

	return nil
}

// CreateAction creates a new action record within a transaction.
func (r *mcmpApiRepository) CreateAction(tx *gorm.DB, action *mcmpapi.McmpApiAction) error {
	// Consider using FirstOrCreate if actions might be duplicated across service versions but should only exist once per service name.
	// For now, assume Create is sufficient as it's called only when a new service version is created.
	query := tx.Create(action)
	if err := query.Error; err != nil {
		// 에러 발생 시에만 쿼리 로깅
		sql := query.Statement.SQL.String()
		args := query.Statement.Vars
		log.Printf("CreateAction SQL Query (ERROR): %s", sql)
		log.Printf("CreateAction SQL Args (ERROR): %v", args)
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
		// 에러 발생 시에만 쿼리 로깅
		sql := serviceQuery.Statement.SQL.String()
		args := serviceQuery.Statement.Vars
		log.Printf("GetAllAPIDefinitions Service SQL Query (ERROR): %s", sql)
		log.Printf("GetAllAPIDefinitions Service SQL Args (ERROR): %v", args)
		return nil, fmt.Errorf("error fetching services: %w", err)
	}

	// Fetch actions if not already fetched (i.e., if actionNameFilter was empty)
	if actionNameFilter == "" {
		if err := actionQuery.Find(&dbActions).Error; err != nil {
			// 에러 발생 시에만 쿼리 로깅
			sql := actionQuery.Statement.SQL.String()
			args := actionQuery.Statement.Vars
			log.Printf("GetAllAPIDefinitions Action SQL Query (ERROR): %s", sql)
			log.Printf("GetAllAPIDefinitions Action SQL Args (ERROR): %v", args)
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
	query := r.db.Where("name = ?", serviceName).First(&service)
	if err := query.Error; err != nil {
		// 에러 발생 시에만 쿼리 로깅
		sql := query.Statement.SQL.String()
		args := query.Statement.Vars
		log.Printf("GetService SQL Query (ERROR): %s", sql)
		log.Printf("GetService SQL Args (ERROR): %v", args)
		return nil, err // gorm.ErrRecordNotFound 포함
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
	query := r.db.Model(&mcmpapi.McmpApiService{}).Where("name = ?", serviceName).Updates(updates)
	if err := query.Error; err != nil {
		// 에러 발생 시에만 쿼리 로깅
		sql := query.Statement.SQL.String()
		args := query.Statement.Vars
		log.Printf("UpdateService SQL Query (ERROR): %s", sql)
		log.Printf("UpdateService SQL Args (ERROR): %v", args)
		return err
	}

	if query.RowsAffected == 0 {
		// This could mean the service name doesn't exist
		return errors.New("service not found") // Or return gorm.ErrRecordNotFound?
	}
	return nil
}

// GetServiceAction DB에서 특정 서비스의 특정 액션 정보를 조회 (대소문자 구분 없이)
func (r *mcmpApiRepository) GetServiceAction(serviceName, actionName string) (*mcmpapi.McmpApiAction, error) { // Renamed receiver and return type
	var action mcmpapi.McmpApiAction // Use renamed action model
	// Use LOWER() function for case-insensitive comparison on action_name
	query := r.db.Where("service_name = ? AND LOWER(action_name) = LOWER(?)", serviceName, actionName).First(&action)
	if err := query.Error; err != nil {
		// 에러 발생 시에만 쿼리 로깅
		sql := query.Statement.SQL.String()
		args := query.Statement.Vars
		log.Printf("GetServiceAction SQL Query (ERROR): %s", sql)
		log.Printf("GetServiceAction SQL Args (ERROR): %v", args)
		// Log the error for debugging, especially if it's not ErrRecordNotFound
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Error fetching action '%s' for service '%s': %v", actionName, serviceName, err)
		}
		return nil, err // gorm.ErrRecordNotFound 포함
	}

	return &action, nil
}

// SetActiveVersion sets the active version for a service.
func (r *mcmpApiRepository) SetActiveVersion(serviceName, version string) error {
	// First, set all versions of the service to inactive
	if err := r.db.Model(&mcmpapi.McmpApiService{}).
		Where("name = ?", serviceName).
		Update("is_active", false).Error; err != nil {
		return fmt.Errorf("failed to deactivate all versions: %w", err)
	}

	// Then, set the specified version to active
	if err := r.db.Model(&mcmpapi.McmpApiService{}).
		Where("name = ? AND version = ?", serviceName, version).
		Update("is_active", true).Error; err != nil {
		return fmt.Errorf("failed to activate version %s: %w", version, err)
	}

	return nil
}

// GetActiveService retrieves the currently active version of a service.
func (r *mcmpApiRepository) GetActiveService(serviceName string) (*mcmpapi.McmpApiService, error) {
	var service mcmpapi.McmpApiService
	// Find the service where name matches and is_active is true
	query := r.db.Where("name = ? AND is_active = ?", serviceName, true).First(&service)
	if err := query.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Try to find *any* version if no active version is set
			log.Printf("No active version found for service %s, attempting to find latest/any version.", serviceName)
			// Example: Find the latest version based on semantic versioning or created_at
			// For simplicity, just find the first one found by name if no active one exists
			fallbackQuery := r.db.Where("name = ?", serviceName).Order("version DESC").First(&service) // Example: order by version desc
			if err := fallbackQuery.Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, errors.New("service not found") // No versions found at all
				}
				// 에러 발생 시에만 쿼리 로깅
				sql := fallbackQuery.Statement.SQL.String()
				args := fallbackQuery.Statement.Vars
				log.Printf("GetActiveService Fallback SQL Query (ERROR): %s", sql)
				log.Printf("GetActiveService Fallback SQL Args (ERROR): %v", args)
				return nil, fallbackQuery.Error // Other DB error during fallback
			}

			log.Printf("Found fallback version %s for service %s.", service.Version, serviceName)
			return &service, nil // Return the found fallback version
		}
		// 에러 발생 시에만 쿼리 로깅
		sql := query.Statement.SQL.String()
		args := query.Statement.Vars
		log.Printf("GetActiveService SQL Query (ERROR): %s", sql)
		log.Printf("GetActiveService SQL Args (ERROR): %v", args)
		return nil, query.Error // Other DB errors when searching for active
	}

	return &service, nil
}

// DeleteActionsByServiceName deletes all actions for a service within a transaction.
func (r *mcmpApiRepository) DeleteActionsByServiceName(tx *gorm.DB, serviceName string) error {
	query := tx.Where("service_name = ?", serviceName).Delete(&mcmpapi.McmpApiAction{})
	if err := query.Error; err != nil {
		sql := query.Statement.SQL.String()
		args := query.Statement.Vars
		log.Printf("DeleteActionsByServiceName SQL Query (ERROR): %s", sql)
		log.Printf("DeleteActionsByServiceName SQL Args (ERROR): %v", args)
		return fmt.Errorf("failed to delete actions for service %s: %w", serviceName, err)
	}
	log.Printf("Deleted %d actions for service %s", query.RowsAffected, serviceName)
	return nil
}

// GetServiceMeta retrieves version metadata for a service.
func (r *mcmpApiRepository) GetServiceMeta(serviceName string) (*mcmpapi.McmpApiServiceMeta, error) {
	var meta mcmpapi.McmpApiServiceMeta
	query := r.db.Where("service_name = ?", serviceName).First(&meta)
	if err := query.Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			sql := query.Statement.SQL.String()
			args := query.Statement.Vars
			log.Printf("GetServiceMeta SQL Query (ERROR): %s", sql)
			log.Printf("GetServiceMeta SQL Args (ERROR): %v", args)
		}
		return nil, err
	}
	return &meta, nil
}

// UpsertServiceMeta creates or updates version metadata within a transaction.
func (r *mcmpApiRepository) UpsertServiceMeta(tx *gorm.DB, meta *mcmpapi.McmpApiServiceMeta) error {
	// Use Save which does upsert based on primary key
	query := tx.Save(meta)
	if err := query.Error; err != nil {
		sql := query.Statement.SQL.String()
		args := query.Statement.Vars
		log.Printf("UpsertServiceMeta SQL Query (ERROR): %s", sql)
		log.Printf("UpsertServiceMeta SQL Args (ERROR): %v", args)
		return fmt.Errorf("failed to upsert meta for service %s: %w", meta.ServiceName, err)
	}
	return nil
}
