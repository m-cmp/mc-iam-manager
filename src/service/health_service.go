package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	// "github.com/m-cmp/mc-iam-manager/config" // Removed unused import
	"github.com/m-cmp/mc-iam-manager/constants"
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
	db                *gorm.DB
	keycloakService   KeycloakService
	roleRepo          *repository.RoleRepository
	workspaceRoleRepo *repository.WorkspaceRoleRepository
	menuRepo          *repository.MenuRepository
	mcmpApiRepo       repository.McmpApiRepository
}

// NewHealthCheckService 생성자
func NewHealthCheckService(
	db *gorm.DB,
) HealthCheckService {
	// Initialize repositories internally
	roleRepo := repository.NewRoleRepository(db)
	wrRepo := repository.NewWorkspaceRoleRepository(db)
	mRepo := repository.NewMenuRepository(db)
	mcmpRepo := repository.NewMcmpApiRepository(db)

	return &healthCheckService{
		db:                db,
		roleRepo:          roleRepo,
		workspaceRoleRepo: wrRepo,
		menuRepo:          mRepo,
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
	err = s.db.Model(&model.RoleMaster{}).Joins("JOIN mcmp_role_subs ON mcmp_role_masters.id = mcmp_role_subs.role_id").
		Where("mcmp_role_subs.role_type = ?", constants.RoleTypePlatform).Count(&status.PlatformRolesCount).Error
	if err != nil {
		log.Printf("Error counting platform roles: %v", err)
	}

	// 5. Workspace Roles Count
	err = s.db.Model(&model.RoleMaster{}).Joins("JOIN mcmp_role_subs ON mcmp_role_masters.id = mcmp_role_subs.role_id").
		Where("mcmp_role_subs.role_type = ?", constants.RoleTypeWorkspace).Count(&status.WorkspaceRolesCount).Error
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

// CheckRoleTables 역할 관련 테이블 상태 확인
func (s *healthCheckService) CheckRoleTables() error {
	// RoleMaster 테이블 확인
	var roleMaster model.RoleMaster
	if err := s.db.First(&roleMaster).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("role master table check failed: %w", err)
	}

	// RoleSub 테이블 확인
	var roleSub model.RoleSub
	if err := s.db.First(&roleSub).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("role sub table check failed: %w", err)
	}

	// UserRole 테이블 확인
	var userRole model.UserPlatformRole
	if err := s.db.First(&userRole).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("user role table check failed: %w", err)
	}

	// UserWorkspaceRole 테이블 확인
	var userWorkspaceRole model.UserWorkspaceRole
	if err := s.db.First(&userWorkspaceRole).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("user workspace role table check failed: %w", err)
	}

	return nil
}

// CheckRoleData 역할 데이터 상태 확인
func (s *healthCheckService) CheckRoleData() error {
	// RoleMaster 데이터 확인
	var roleMasters []model.RoleMaster
	if err := s.db.Preload("RoleSubs").Find(&roleMasters).Error; err != nil {
		return fmt.Errorf("role master data check failed: %w", err)
	}

	// RoleSub 데이터 확인
	var roleSubs []model.RoleSub
	if err := s.db.Find(&roleSubs).Error; err != nil {
		return fmt.Errorf("role sub data check failed: %w", err)
	}

	// UserRole 데이터 확인
	var userRoles []model.UserPlatformRole
	if err := s.db.Find(&userRoles).Error; err != nil {
		return fmt.Errorf("user role data check failed: %w", err)
	}

	// UserWorkspaceRole 데이터 확인
	var userWorkspaceRoles []model.UserWorkspaceRole
	if err := s.db.Find(&userWorkspaceRoles).Error; err != nil {
		return fmt.Errorf("user workspace role data check failed: %w", err)
	}

	return nil
}
