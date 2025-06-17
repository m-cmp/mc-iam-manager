package handler

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/config"
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
// @Router /initial-admin [post]
// @OperationId setupInitialAdmin
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
	kcUserId, err := h.keycloakService.SetupInitialAdmin(c.Request().Context(), adminToken)
	if err != nil {
		log.Printf("[ERROR] Failed to setup initial admin: %v", err)
		return c.JSON(http.StatusInternalServerError, model.Response{
			Error:   true,
			Message: "Failed to setup initial admin",
		})
	}

	platformAdminID := os.Getenv("MCIAMMANAGER_PLATFORMADMIN_ID")
	platformAdminFirstName := os.Getenv("MCIAMMANAGER_PLATFORMADMIN_FIRSTNAME")
	platformAdminLastName := os.Getenv("MCIAMMANAGER_PLATFORMADMIN_LASTNAME")

	// 2. 유저 동기화 : keycloak에 먼저 만들었으므로 동기화 해준다.
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
		}, []model.RoleSub{{RoleType: model.RoleTypePlatform}, {RoleType: model.RoleTypeWorkspace}})
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
	err = h.workspaceService.CreateWorkspace(&model.Workspace{
		Name:        "ws01",
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
// @Router /setup/check-user-roles [get]
// @OperationId checkUserRoles
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

// ListPlatformRoles godoc
// @Summary 플랫폼 역할 목록 조회
// @Description 모든 플랫폼 역할 목록을 조회합니다
// @Tags admin
// @Accept json
// @Produce json
// @Success 200 {array} model.PlatformRole
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Security BearerAuth
// @Router /admin/platform-roles/list [post]
// @OperationId listPlatformRoles
func (h *AdminHandler) ListPlatformRoles(c echo.Context) error {
	// Implementation of ListPlatformRoles method
	return nil // Placeholder return, actual implementation needed
}

// GetPlatformRoleByID godoc
// @Summary 플랫폼 역할 ID로 조회
// @Description 특정 플랫폼 역할을 ID로 조회합니다
// @Tags admin
// @Accept json
// @Produce json
// @Param id path string true "Platform Role ID"
// @Success 200 {object} model.PlatformRole
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 404 {object} map[string]string "error: Platform Role not found"
// @Security BearerAuth
// @Router /admin/platform-roles/{id} [get]
// @OperationId getPlatformRoleByID
func (h *AdminHandler) GetPlatformRoleByID(c echo.Context) error {
	// Implementation of GetPlatformRoleByID method
	return nil // Placeholder return, actual implementation needed
}

// CreatePlatformRole godoc
// @Summary 새 플랫폼 역할 생성
// @Description 새로운 플랫폼 역할을 생성합니다
// @Tags admin
// @Accept json
// @Produce json
// @Param role body model.PlatformRole true "Platform Role Info"
// @Success 201 {object} model.PlatformRole
// @Failure 400 {object} map[string]string "error: Invalid request"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Security BearerAuth
// @Router /admin/platform-roles [post]
// @OperationId createPlatformRole
func (h *AdminHandler) CreatePlatformRole(c echo.Context) error {
	// Implementation of CreatePlatformRole method
	return nil // Placeholder return, actual implementation needed
}

// UpdatePlatformRole godoc
// @Summary 플랫폼 역할 업데이트
// @Description 플랫폼 역할 정보를 업데이트합니다
// @Tags admin
// @Accept json
// @Produce json
// @Param id path string true "Platform Role ID"
// @Param role body model.PlatformRole true "Platform Role Info"
// @Success 200 {object} model.PlatformRole
// @Failure 400 {object} map[string]string "error: Invalid request"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 404 {object} map[string]string "error: Platform Role not found"
// @Security BearerAuth
// @Router /admin/platform-roles/{id} [put]
// @OperationId updatePlatformRole
func (h *AdminHandler) UpdatePlatformRole(c echo.Context) error {
	// Implementation of UpdatePlatformRole method
	return nil // Placeholder return, actual implementation needed
}

// DeletePlatformRole godoc
// @Summary 플랫폼 역할 삭제
// @Description 플랫폼 역할을 삭제합니다
// @Tags admin
// @Accept json
// @Produce json
// @Param id path string true "Platform Role ID"
// @Success 204 "No Content"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 404 {object} map[string]string "error: Platform Role not found"
// @Security BearerAuth
// @Router /admin/platform-roles/{id} [delete]
// @OperationId deletePlatformRole
func (h *AdminHandler) DeletePlatformRole(c echo.Context) error {
	// Implementation of DeletePlatformRole method
	return nil // Placeholder return, actual implementation needed
}
