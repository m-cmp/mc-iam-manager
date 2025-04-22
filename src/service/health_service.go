package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	// "github.com/m-cmp/mc-iam-manager/config" // Removed unused import
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/model/mcmpapi"
	"github.com/m-cmp/mc-iam-manager/repository"

	// "github.com/Nerzal/gocloak/v13" // Removed unused import
	"gorm.io/gorm"
)

// HealthStatus 상세 상태 정보를 담는 구조체
type HealthStatus struct {
	DBConnection        string `json:"db_connection"`
	KeycloakAdminLogin  string `json:"keycloak_admin_login"`
	KeycloakRealmCheck  string `json:"keycloak_realm_check"`
	KeycloakClientCheck string `json:"keycloak_client_check"`
	PlatformRolesCount  int64  `json:"platform_roles_count"`
	WorkspaceRolesCount int64  `json:"workspace_roles_count"`
	MenusCount          int64  `json:"menus_count"`
	McmpServicesCount   int64  `json:"mcmp_services_count"`
	McmpActionsCount    int64  `json:"mcmp_actions_count"`
}

// HealthCheckService 인터페이스
type HealthCheckService interface {
	GetDetailedStatus(ctx context.Context) (*HealthStatus, error)
}

// healthCheckService 구현체
type healthCheckService struct {
	db              *gorm.DB
	keycloakService KeycloakService // Add KeycloakService dependency
	// Change repository fields to use pointer types to match constructor parameters
	platformRoleRepo  *repository.PlatformRoleRepository  // Change to pointer type
	workspaceRoleRepo *repository.WorkspaceRoleRepository // Change to pointer type
	menuRepo          *repository.MenuRepository          // Change to pointer type
	mcmpApiRepo       repository.McmpApiRepository        // Keep interface type for McmpApiRepository
}

// NewHealthCheckService 생성자
func NewHealthCheckService(
	db *gorm.DB,
	// keycloakService KeycloakService, // Remove KeycloakService parameter
) HealthCheckService {
	// Initialize repositories internally
	prRepo := repository.NewPlatformRoleRepository(db)
	wrRepo := repository.NewWorkspaceRoleRepository(db)
	mRepo := repository.NewMenuRepository(db)
	mcmpRepo := repository.NewMcmpApiRepository(db)

	return &healthCheckService{
		db: db,
		// keycloakService:   keycloakService, // Remove field assignment
		platformRoleRepo:  prRepo, // Assign pointer
		workspaceRoleRepo: wrRepo, // Assign pointer
		menuRepo:          mRepo,  // Assign pointer
		mcmpApiRepo:       mcmpRepo,
	}
}

// GetDetailedStatus 상세 상태 확인 로직
func (s *healthCheckService) GetDetailedStatus(ctx context.Context) (*HealthStatus, error) {
	status := &HealthStatus{}
	var err error
	var sqlDB *sql.DB

	// 1. DB Connection Check
	sqlDB, err = s.db.DB()
	if err != nil {
		status.DBConnection = fmt.Sprintf("Failed to get DB instance: %v", err)
	} else if err = sqlDB.PingContext(ctx); err != nil {
		status.DBConnection = fmt.Sprintf("Failed to ping DB: %v", err)
	} else {
		status.DBConnection = "OK"
	}

	// 2. Keycloak Admin Login Check
	ks := NewKeycloakService() // Create KeycloakService instance when needed
	ok, loginErr := ks.CheckAdminLogin(ctx)
	if !ok {
		status.KeycloakAdminLogin = fmt.Sprintf("Failed: %v", loginErr)
	} else {
		status.KeycloakAdminLogin = "OK"
	}

	// 3. Keycloak Realm & Client Check (only if admin login succeeded)
	if status.KeycloakAdminLogin == "OK" {
		// Realm Check
		ok, realmErr := ks.CheckRealm(ctx) // Use ks instance
		if !ok {
			status.KeycloakRealmCheck = fmt.Sprintf("Failed: %v", realmErr)
		} else {
			status.KeycloakRealmCheck = "OK"
		}

		// Client Check
		ok, clientErr := ks.CheckClient(ctx) // Use ks instance
		if !ok {
			status.KeycloakClientCheck = fmt.Sprintf("Failed: %v", clientErr)
		} else {
			status.KeycloakClientCheck = "OK"
		}
	} else {
		status.KeycloakRealmCheck = "Skipped (Admin login failed)"
		status.KeycloakClientCheck = "Skipped (Admin login failed)"
	}

	// 4. Platform Roles Count
	err = s.db.Model(&model.PlatformRole{}).Count(&status.PlatformRolesCount).Error
	if err != nil {
		log.Printf("Error counting platform roles: %v", err)
		// Optionally set a specific error message in status
	}

	// 5. Workspace Roles Count
	err = s.db.Model(&model.WorkspaceRole{}).Count(&status.WorkspaceRolesCount).Error
	if err != nil {
		log.Printf("Error counting workspace roles: %v", err)
	}

	// 6. Menus Count
	err = s.db.Model(&model.Menu{}).Count(&status.MenusCount).Error
	if err != nil {
		log.Printf("Error counting menus: %v", err)
	}

	// 7. Mcmp Services Count
	err = s.db.Model(&mcmpapi.McmpApiService{}).Count(&status.McmpServicesCount).Error
	if err != nil {
		log.Printf("Error counting mcmp services: %v", err)
	}

	// 8. Mcmp Actions Count
	err = s.db.Model(&mcmpapi.McmpApiAction{}).Count(&status.McmpActionsCount).Error
	if err != nil {
		log.Printf("Error counting mcmp actions: %v", err)
	}

	// Return status even if some checks failed
	return status, nil
}
