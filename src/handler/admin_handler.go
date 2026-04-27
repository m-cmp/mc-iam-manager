package handler

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/config"
	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm"
)

// Platform 관리자로서 실행할 수 있는 기능들을 정의함.

// AdminHandler 관리자 API 핸들러
type AdminHandler struct {
	keycloakService  service.KeycloakService
	userService      service.UserService
	roleService      service.RoleService
	workspaceService service.WorkspaceService
	menuService      service.MenuService
}

// NewAdminHandler 새 AdminHandler 인스턴스 생성
func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{
		keycloakService:  service.NewKeycloakService(),
		userService:      *service.NewUserService(db),
		roleService:      *service.NewRoleService(db),
		workspaceService: *service.NewWorkspaceService(db),
		menuService:      *service.NewMenuService(db),
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
	predefinedRoles := strings.Split(os.Getenv("PREDEFINED_PLATFORM_ROLE"), ",")
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
	defaultWsName := os.Getenv("DEFAULT_WORKSPACE_NAME")
	if defaultWsName == "" {
		defaultWsName = "ws01" // fallback
		log.Printf("[INFO] DEFAULT_WORKSPACE_NAME not set, using default: %s", defaultWsName)
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

	// 메뉴와 기본 역할 매핑
	err = h.menuService.InitializeMenuPermissionsFromCSV("")
	if err != nil {
		log.Printf("[ERROR] Initialize Menu Permissions failed: %v", err)
		// return c.JSON(http.StatusInternalServerError, model.Response{
		// 	Error:   true,
		// 	Message: "Failed to initialize menu permissions",
		// })
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
// @Summary Initialize menu permissions from CSV
// @Description CSV 파일을 읽어서 메뉴 권한을 초기화합니다
// @Tags admin
// @Accept json
// @Produce json
// @Param filePath query string false "CSV file path (optional, uses default if not provided)"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Security BearerAuth
// @Router /api/setup/initial-role-menu-permission [get]
// @Id initializeMenuPermissions
func (h *AdminHandler) InitializeMenuPermissions(c echo.Context) error {
	filePath := c.QueryParam("filePath")

	log.Printf("[INFO] Initializing menu permissions from CSV file: %s", filePath)

	err := h.menuService.InitializeMenuPermissionsFromCSV(filePath)
	if err != nil {
		log.Printf("[ERROR] Initialize Menu Permissions failed: %v", err)
		return c.JSON(http.StatusInternalServerError, model.Response{
			Error:   true,
			Message: fmt.Sprintf("Failed to initialize menu permissions: %v", err),
		})
	}

	log.Printf("[INFO] Menu permissions initialized successfully")
	return c.JSON(http.StatusOK, model.Response{
		Error:   false,
		Message: "Menu permissions initialized successfully",
	})
}
