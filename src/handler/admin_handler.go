package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/config"
	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

// Platform 관리자로서 실행할 수 있는 기능들을 정의함.

// AdminHandler 관리자 API 핸들러
type AdminHandler struct {
	keycloakService      service.KeycloakService
	userService          service.UserService
	roleService          service.RoleService
	workspaceService     service.WorkspaceService
	menuService          service.MenuService
	organizationService  *service.OrganizationService
	companyService       *service.CompanyService
}

// NewAdminHandler 새 AdminHandler 인스턴스 생성
func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{
		keycloakService:     service.NewKeycloakService(),
		userService:         *service.NewUserService(db),
		roleService:         *service.NewRoleService(db),
		workspaceService:    *service.NewWorkspaceService(db),
		menuService:         *service.NewMenuService(db),
		organizationService: service.NewOrganizationService(db),
		companyService:      service.NewCompanyService(db),
	}
}

// SetupInitialAdmin godoc
// @Summary Setup initial platform admin
// @Description Creates the initial platform admin user with necessary permissions. platform admin 생성인데
// @Tags admin
// @Accept json
// @Produce json
// @Param request body model.SetupInitialAdminRequest true "Setup Initial Admin Request"
// @Success 200 {object} model.Response
// @Router /api/initial-admin [post]
// @Id setupInitialAdmin
func (h *AdminHandler) SetupInitialAdmin(c echo.Context) error {
	var req model.SetupInitialAdminRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, model.Response{
			Error:   true,
			Message: "Invalid request body",
		})
	}

	// Check if Keycloak is configured
	if config.KC == nil {
		log.Printf("[ERROR] Keycloak configuration is not initialized")
		return c.JSON(http.StatusInternalServerError, model.Response{
			Error:   true,
			Message: "Keycloak configuration is not initialized",
		})
	}

	adminToken, err := h.keycloakService.KeycloakAdminLogin(c.Request().Context())
	if err != nil {
		log.Printf("[ERROR] Failed to login as admin: %v", err)
		return c.JSON(http.StatusInternalServerError, model.Response{
			Error:   true,
			Message: "Failed to login as admin",
		})
	}

	if adminToken == nil {
		log.Printf("[ERROR] Admin token is nil")
		return c.JSON(http.StatusInternalServerError, model.Response{
			Error:   true,
			Message: "Failed to get admin token",
		})
	}

	log.Printf("[DEBUG] Setting Platform Admin") // KC에 user, realm, client 생성
	kcUserId, err := h.keycloakService.SetupInitialKeycloakAdmin(c.Request().Context(), adminToken)
	if err != nil {
		log.Printf("[ERROR] Failed to setup initial admin: %v", err)
		return c.JSON(http.StatusInternalServerError, model.Response{
			Error:   true,
			Message: "Failed to setup initial admin",
		})
	}

	platformAdminID := os.Getenv("MC_IAM_MANAGER_PLATFORMADMIN_ID")
	platformAdminFirstName := os.Getenv("MC_IAM_MANAGER_PLATFORMADMIN_FIRSTNAME")
	platformAdminLastName := os.Getenv("MC_IAM_MANAGER_PLATFORMADMIN_LASTNAME")

	// 2. 유저 동기화 : keycloak에 먼저 만들었으므로 DB에 동기화 해준다.
	err = h.userService.SyncUserByKeycloak(c.Request().Context(), &model.User{
		Username:  platformAdminID,
		Email:     req.Email,
		FirstName: platformAdminFirstName,
		LastName:  platformAdminLastName,
		KcId:      kcUserId,
	})
	if err != nil {
		log.Printf("[ERROR] Failed to create user: %v", err)
		// return c.JSON(http.StatusInternalServerError, model.Response{
		// 	Error:   true,
		// 	Message: "Failed to create user",
		// })
	}

	// 3. 기본 역할 설정
	log.Printf("[DEBUG] Setting default roles")

	// 3-1. keycloak에 기본 역할 생성
	err = h.keycloakService.SetupPredefinedRoles(c.Request().Context(), adminToken.AccessToken)
	if err != nil {
		log.Printf("[ERROR] Setup Realm roles failed: %v", err)
		// return c.JSON(http.StatusInternalServerError, model.Response{
		// 	Error:   true,
		// 	Message: "Failed to Setup default roles",
		// })
	}

	// 3-2. db에 기본 역할 생성
	predefinedRoles := strings.Split(os.Getenv("MC_IAM_MANAGER_PREDEFINED_PLATFORM_ROLE"), ",")
	registeredRoles := []uint{}
	for _, roleName := range predefinedRoles {
		role, err := h.roleService.CreateRoleWithSubs(&model.RoleMaster{
			Name: roleName,
		}, []model.RoleSub{{RoleType: constants.RoleTypePlatform}, {RoleType: constants.RoleTypeWorkspace}})
		if err != nil {
			log.Printf("[ERROR] Create Role with Subs failed: %v", err)
			// return c.JSON(http.StatusInternalServerError, model.Response{
			// 	Error:   true,
			// 	Message: "Failed to Create Role with Subs",
			// })
		}
		registeredRoles = append(registeredRoles, role.ID)
		log.Printf("[DEBUG] Create Role with Subs success: %v", role)
	}

	// 기본 workspace 생성
	defaultWsName := os.Getenv("MC_IAM_MANAGER_DEFAULT_WORKSPACE_NAME")
	if defaultWsName == "" {
		defaultWsName = "ws01" // fallback
		log.Printf("[INFO] MC_IAM_MANAGER_DEFAULT_WORKSPACE_NAME not set, using default: %s", defaultWsName)
	}
	
	err = h.workspaceService.CreateWorkspace(&model.Workspace{
		Name:        defaultWsName,
		Description: "Default Workspace",
	})
	if err != nil {
		log.Printf("[ERROR] Create Workspace failed: %v", err)
		// return c.JSON(http.StatusInternalServerError, model.Response{
		// 	Error:   true,
		// 	Message: "Failed to create default workspace",
		// })
	}

	// 메뉴 등록
	err = h.menuService.LoadAndRegisterMenusFromYAML("")
	if err != nil {
		log.Printf("[ERROR] Register Menu failed: %v", err)
		// return c.JSON(http.StatusInternalServerError, model.Response{
		// 	Error:   true,
		// 	Message: "Failed to register menus",
		// })
	}

	// 메뉴와 기본 역할 매핑 (permission.yaml)
	err = h.menuService.InitializeMenuPermissionsFromYAML("")
	if err != nil {
		log.Printf("[ERROR] Initialize Menu Permissions (YAML) failed: %v", err)
		// return c.JSON(http.StatusInternalServerError, model.Response{
		// 	Error:   true,
		// 	Message: "Failed to initialize menu permissions",
		// })
	}

	// 기본 조직 등록
	err = h.organizationService.LoadAndRegisterOrganizationsFromYAML("")
	if err != nil {
		log.Printf("[ERROR] Register default organizations failed: %v", err)
	}

	// 기본 회사 생성 (COMP-006: 이미 존재하면 skip, 실패해도 non-fatal)
	if err := h.companyService.CreateDefaultCompany(); err != nil {
		log.Printf("[WARNING] Failed to create default company: %v", err)
	}

	// Platform Admin 역할에 모든 메뉴 매핑 추가 : 메뉴 목록 조회에 구현되어 있음.

	// TODO : 해당 Realm에 scope 추가
	// TODO : OIDC Client 생성
	// TODO : 해당 Client에 해당 역할 매핑
	// (직접) CSP에 idp 연동 by oidc
	// TODO : CSP 역할 매핑에 provider와 arn 정보 설정
	return c.JSON(http.StatusOK, model.Response{
		Message: "Initial admin setup completed successfully",
	})
}

// CheckUserRoles godoc
// @Summary Check user roles
// @Description Check all roles assigned to a user. 특정 유저가 가진 role 목록을 조회합니다.
// @Tags admin
// @Accept json
// @Produce json
// @Param username query string true "Username to check roles"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/setup/check-user-roles [get]
// @Id checkUserRoles
func (h *AdminHandler) CheckUserRoles(c echo.Context) error {
	username := c.QueryParam("username")
	if username == "" {
		return c.JSON(http.StatusBadRequest, model.Response{
			Error:   true,
			Message: "username is required",
		})
	}

	err := h.keycloakService.CheckUserRoles(c.Request().Context(), username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.Response{
			Error:   true,
			Message: fmt.Sprintf("failed to check user roles: %v", err),
		})
	}

	return c.JSON(http.StatusOK, model.Response{
		Error:   false,
		Message: "User roles checked successfully. Check server logs for details.",
	})
}

// InitializeMenuPermissions godoc
// @Summary Initialize menu permissions from CSV (deprecated)
// @Description Deprecated. permission.csv로 메뉴 권한을 초기화합니다. 제거 예정이니 InitializeMenuPermissionsFromYAML을 사용하세요.
// @Tags admin
// @Accept json
// @Produce json
// @Param filePath query string false "CSV file path (optional, default asset/menu/permission.csv)"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Security BearerAuth
// @Deprecated
// @Router /api/setup/initial-role-menu-permission [get]
// @Id initializeMenuPermissions
func (h *AdminHandler) InitializeMenuPermissions(c echo.Context) error {
	filePath := c.QueryParam("filePath")

	log.Printf(
		"[WARN] Deprecated API /api/setup/initial-role-menu-permission (CSV). "+
			"Use /api/setup/initial-role-menu-permission-yaml instead. filePath=%s",
		filePath,
	)

	err := h.menuService.InitializeMenuPermissionsFromCSV(filePath)
	if err != nil {
		log.Printf("[ERROR] Initialize Menu Permissions (CSV) failed: %v", err)
		return c.JSON(http.StatusInternalServerError, model.Response{
			Error:   true,
			Message: fmt.Sprintf("Failed to initialize menu permissions from CSV: %v", err),
		})
	}

	log.Printf("[INFO] Menu permissions initialized successfully from CSV (deprecated)")
	return c.JSON(http.StatusOK, model.Response{
		Error:   false,
		Message: "Menu permissions initialized successfully from CSV (deprecated; prefer YAML API)",
	})
}

// InitializeMenuPermissionsFromYAML godoc
// @Summary Initialize role-menu permissions from YAML
// @Description asset/menu/permission.yaml(permissions→role→menus|operations|csps)을 읽어 역할-메뉴 매핑을 DB에 시드합니다
// @Tags admin
// @Accept json
// @Produce json
// @Param filePath query string false "YAML file path (optional, default asset/menu/permission.yaml)"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Security BearerAuth
// @Router /api/setup/initial-role-menu-permission-yaml [get]
// @Id initializeMenuPermissionsFromYAML
func (h *AdminHandler) InitializeMenuPermissionsFromYAML(c echo.Context) error {
	filePath := c.QueryParam("filePath")

	log.Printf("[INFO] Initializing menu permissions from YAML: %s", filePath)

	err := h.menuService.InitializeMenuPermissionsFromYAML(filePath)
	if err != nil {
		log.Printf("[ERROR] Initialize Menu Permissions (YAML) failed: %v", err)
		return c.JSON(http.StatusInternalServerError, model.Response{
			Error:   true,
			Message: fmt.Sprintf("Failed to initialize menu permissions from YAML: %v", err),
		})
	}

	log.Printf("[INFO] Menu permissions initialized successfully from YAML")
	return c.JSON(http.StatusOK, model.Response{
		Error:   false,
		Message: "Menu permissions initialized successfully from YAML",
	})
}

// BackupRolePermissions godoc
// @Summary Backup current role permissions from DB
// @Description 플랫폼 역할의 현재 메뉴(및 reserved ops/csp) 권한을 role-permission-backup 문서로 내보냅니다
// @Tags admin
// @Produce json
// @Produce application/yaml
// @Param roles query string false "Comma-separated role names (default: all platform roles)"
// @Param sections query string false "Comma-separated sections (menus,operations,csps). Default: menus"
// @Param format query string false "yaml or json (default yaml)"
// @Param save query bool false "If true, also write under asset/menu/backups/"
// @Success 200 {object} model.RolePermissionBackup
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Security BearerAuth
// @Router /api/setup/backup-role-permissions [get]
// @Id backupRolePermissions
func (h *AdminHandler) BackupRolePermissions(c echo.Context) error {
	roleNames := splitCSVQuery(c.QueryParam("roles"))
	sections := splitCSVQuery(c.QueryParam("sections"))
	format := strings.ToLower(strings.TrimSpace(c.QueryParam("format")))
	if format == "" {
		format = "yaml"
	}
	save := strings.EqualFold(c.QueryParam("save"), "true") || c.QueryParam("save") == "1"

	backup, err := h.menuService.BackupRolePermissions(roleNames, sections)
	if err != nil {
		log.Printf("[ERROR] BackupRolePermissions failed: %v", err)
		return c.JSON(http.StatusInternalServerError, model.Response{
			Error:   true,
			Message: fmt.Sprintf("Failed to backup role permissions: %v", err),
		})
	}

	savedPath := ""
	if save {
		savedPath, err = h.menuService.SaveRolePermissionBackupFile(backup, "")
		if err != nil {
			log.Printf("[ERROR] SaveRolePermissionBackupFile failed: %v", err)
			return c.JSON(http.StatusInternalServerError, model.Response{
				Error:   true,
				Message: fmt.Sprintf("Failed to save role permission backup file: %v", err),
			})
		}
		log.Printf("[INFO] Role permission backup saved: %s", savedPath)
	}

	if format == "json" {
		if savedPath != "" {
			c.Response().Header().Set("X-Role-Permission-Backup-Path", savedPath)
		}
		return c.JSON(http.StatusOK, backup)
	}

	body, err := yaml.Marshal(backup)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.Response{
			Error:   true,
			Message: fmt.Sprintf("Failed to marshal backup yaml: %v", err),
		})
	}
	if savedPath != "" {
		c.Response().Header().Set("X-Role-Permission-Backup-Path", savedPath)
	}
	c.Response().Header().Set(
		"Content-Disposition",
		`attachment; filename="role-permission-backup.yaml"`,
	)
	return c.Blob(http.StatusOK, "application/yaml", body)
}

// RestoreRolePermissions godoc
// @Summary Restore role permissions from backup document
// @Description role-permission-backup YAML/JSON(또는 filePath)으로 역할 메뉴 권한을 복구합니다. mode=additive|replace-role
// @Tags admin
// @Accept json
// @Accept application/yaml
// @Produce json
// @Param mode query string false "additive (default) or replace-role"
// @Param sections query string false "Comma-separated sections. Default: menus"
// @Param filePath query string false "Local backup file path (if body empty)"
// @Param body body model.RolePermissionBackup false "Backup document"
// @Success 200 {object} model.RolePermissionRestoreResult
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Security BearerAuth
// @Router /api/setup/restore-role-permissions [post]
// @Id restoreRolePermissions
func (h *AdminHandler) RestoreRolePermissions(c echo.Context) error {
	mode := c.QueryParam("mode")
	sections := splitCSVQuery(c.QueryParam("sections"))
	filePath := strings.TrimSpace(c.QueryParam("filePath"))

	var backup *model.RolePermissionBackup
	var err error

	if filePath != "" {
		backup, err = h.menuService.LoadRolePermissionBackupFile(filePath)
		if err != nil {
			return c.JSON(http.StatusBadRequest, model.Response{
				Error:   true,
				Message: fmt.Sprintf("Failed to load backup file: %v", err),
			})
		}
	} else {
		bodyBytes, readErr := io.ReadAll(c.Request().Body)
		if readErr != nil {
			return c.JSON(http.StatusBadRequest, model.Response{
				Error:   true,
				Message: fmt.Sprintf("Failed to read request body: %v", readErr),
			})
		}
		if len(bytes.TrimSpace(bodyBytes)) == 0 {
			return c.JSON(http.StatusBadRequest, model.Response{
				Error:   true,
				Message: "request body or filePath is required",
			})
		}
		ct := strings.ToLower(c.Request().Header.Get(echo.HeaderContentType))
		if strings.Contains(ct, "json") {
			backup = &model.RolePermissionBackup{}
			if err := json.Unmarshal(bodyBytes, backup); err != nil {
				return c.JSON(http.StatusBadRequest, model.Response{
					Error:   true,
					Message: fmt.Sprintf("Invalid backup JSON: %v", err),
				})
			}
		} else {
			backup, err = service.ParseRolePermissionBackupYAML(bodyBytes)
			if err != nil {
				return c.JSON(http.StatusBadRequest, model.Response{
					Error:   true,
					Message: err.Error(),
				})
			}
		}
	}

	result, err := h.menuService.RestoreRolePermissions(backup, mode, sections)
	if err != nil {
		log.Printf("[ERROR] RestoreRolePermissions failed: %v", err)
		return c.JSON(http.StatusInternalServerError, model.Response{
			Error:   true,
			Message: fmt.Sprintf("Failed to restore role permissions: %v", err),
		})
	}

	log.Printf(
		"[INFO] Role permissions restored: mode=%s roles=%d added=%d removed=%d",
		result.Mode, result.RolesProcessed, result.MenusAdded, result.MenusRemoved,
	)
	return c.JSON(http.StatusOK, result)
}

func splitCSVQuery(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
