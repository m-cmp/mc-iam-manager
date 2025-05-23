package handler

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm"
)

type Response struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

// AdminHandler 관리자 API 핸들러
type AdminHandler struct {
	db              *gorm.DB
	keycloakService service.KeycloakService
}

// NewAdminHandler 새 AdminHandler 인스턴스 생성
func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{
		db:              db,
		keycloakService: service.NewKeycloakService(),
	}
}

// SetupInitialAdminRequest represents the request body for setting up the initial admin
type SetupInitialAdminRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

// SetupInitialAdmin godoc
// @Summary Setup initial platform admin
// @Description Creates the initial platform admin user with necessary permissions
// @Tags admin
// @Accept json
// @Produce json
// @Param request body SetupInitialAdminRequest true "Setup Initial Admin Request"
// @Success 200 {object} Response
// @Router /api/setup/user [post]
func (h *AdminHandler) SetupInitialAdmin(c echo.Context) error {
	var req SetupInitialAdminRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, Response{
			Error:   true,
			Message: "Invalid request body",
		})
	}

	if err := h.keycloakService.SetupInitialAdmin(c.Request().Context()); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Error:   true,
			Message: "Failed to setup initial admin",
		})
	}

	return c.JSON(http.StatusOK, Response{
		Message: "Initial admin setup completed successfully",
	})
}

// CheckUserRoles godoc
// @Summary Check user roles
// @Description Check all roles assigned to a user
// @Tags admin
// @Accept json
// @Produce json
// @Param username query string true "Username to check roles"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /api/setup/check-roles [get]
func (h *AdminHandler) CheckUserRoles(c echo.Context) error {
	username := c.QueryParam("username")
	if username == "" {
		return c.JSON(http.StatusBadRequest, Response{
			Error:   true,
			Message: "username is required",
		})
	}

	err := h.keycloakService.CheckUserRoles(c.Request().Context(), username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Error:   true,
			Message: fmt.Sprintf("failed to check user roles: %v", err),
		})
	}

	return c.JSON(http.StatusOK, Response{
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
// @Router /api/v1/admin/platform-roles [get]

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
// @Router /api/v1/admin/platform-roles/{id} [get]

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
// @Router /api/v1/admin/platform-roles [post]

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
// @Router /api/v1/admin/platform-roles/{id} [put]

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
// @Router /api/v1/admin/platform-roles/{id} [delete]
