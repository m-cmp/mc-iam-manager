package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm" // Import gorm
)

type PlatformRoleHandler struct {
	service         *service.PlatformRoleService
	userService     *service.UserService
	keycloakService service.KeycloakService
	// db *gorm.DB // Not needed directly in handler
}

func NewPlatformRoleHandler(db *gorm.DB) *PlatformRoleHandler { // Accept db, remove service param
	// Initialize service internally
	platformRoleService := service.NewPlatformRoleService(db)
	userService := service.NewUserService(db)
	keycloakService := service.NewKeycloakService()
	return &PlatformRoleHandler{
		service:         platformRoleService,
		userService:     userService,
		keycloakService: keycloakService,
	}
}

// List godoc
// @Summary 플랫폼 역할 목록 조회
// @Description 모든 플랫폼 역할을 조회합니다.
// @Tags platform-roles
// @Accept json
// @Produce json
// @Success 200 {array} model.PlatformRole
// @Router /api/platform-roles [get]
func (h *PlatformRoleHandler) List(c echo.Context) error {
	roles, err := h.service.List()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "플랫폼 역할 목록을 가져오는데 실패했습니다",
		})
	}
	return c.JSON(http.StatusOK, roles)
}

// GetByID godoc
// @Summary 플랫폼 역할 조회
// @Description ID로 플랫폼 역할을 조회합니다.
// @Tags platform-roles
// @Accept json
// @Produce json
// @Param id path int true "Platform Role ID"
// @Success 200 {object} model.PlatformRole
// @Router /api/platform-roles/{id} [get]
func (h *PlatformRoleHandler) GetByID(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "잘못된 ID 형식입니다",
		})
	}

	role, err := h.service.GetByID(uint(id))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "플랫폼 역할을 찾을 수 없습니다",
		})
	}
	return c.JSON(http.StatusOK, role)
}

// Create godoc
// @Summary 플랫폼 역할 생성
// @Description 새로운 플랫폼 역할을 생성합니다.
// @Tags platform-roles
// @Accept json
// @Produce json
// @Param role body model.PlatformRole true "Platform Role"
// @Success 201 {object} model.PlatformRole
// @Router /api/platform-roles [post]
func (h *PlatformRoleHandler) Create(c echo.Context) error {
	role := new(model.PlatformRole)
	if err := c.Bind(role); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "잘못된 요청 형식입니다",
		})
	}

	if err := h.service.Create(role); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "플랫폼 역할 생성에 실패했습니다",
		})
	}
	return c.JSON(http.StatusCreated, role)
}

// Update godoc
// @Summary 플랫폼 역할 수정
// @Description 기존 플랫폼 역할을 수정합니다.
// @Tags platform-roles
// @Accept json
// @Produce json
// @Param id path int true "Platform Role ID"
// @Param role body model.PlatformRole true "Platform Role"
// @Success 200 {object} model.PlatformRole
// @Router /api/platform-roles/{id} [put]
func (h *PlatformRoleHandler) Update(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "잘못된 ID 형식입니다",
		})
	}

	role := new(model.PlatformRole)
	if err := c.Bind(role); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "잘못된 요청 형식입니다",
		})
	}
	role.ID = uint(id)

	if err := h.service.Update(role); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "플랫폼 역할 수정에 실패했습니다",
		})
	}
	return c.JSON(http.StatusOK, role)
}

// Delete godoc
// @Summary 플랫폼 역할 삭제
// @Description 플랫폼 역할을 삭제합니다.
// @Tags platform-roles
// @Accept json
// @Produce json
// @Param id path int true "Platform Role ID"
// @Success 204 "No Content"
// @Router /api/platform-roles/{id} [delete]
func (h *PlatformRoleHandler) Delete(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "잘못된 ID 형식입니다",
		})
	}

	if err := h.service.Delete(uint(id)); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "플랫폼 역할 삭제에 실패했습니다",
		})
	}
	return c.NoContent(http.StatusNoContent)
}

// AssignPlatformRoleToUser assigns a platform role to a user
// @Summary Assign a platform role to a user
// @Description Assigns a platform role to a user by their ID or username
// @Tags platform-roles
// @Accept json
// @Produce json
// @Param request body map[string]string true "Request body containing either 'id' or 'username' and 'role'"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/users/assign/platform-roles [post]
func (h *PlatformRoleHandler) AssignPlatformRoleToUser(c echo.Context) error {
	// 요청 본문에서 파라미터 가져오기
	var requestBody map[string]string
	if err := c.Bind(&requestBody); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body format",
		})
	}

	// 필수 파라미터 확인
	roleName, exists := requestBody["role"]
	if !exists {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Role name is required",
		})
	}

	var user *model.User
	var err error

	// ID 또는 username으로 사용자 찾기
	if id, exists := requestBody["id"]; exists {
		// ID를 uint로 변환
		userIDUint, err := strconv.ParseUint(id, 10, 32)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid user ID format",
			})
		}
		user, err = h.userService.GetUserByID(c.Request().Context(), uint(userIDUint))
	} else if username, exists := requestBody["username"]; exists {
		user, err = h.userService.GetUserByUsername(c.Request().Context(), username)
	} else {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Either 'id' or 'username' is required",
		})
	}

	// 사용자 찾기 오류 처리
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "User not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": fmt.Sprintf("Failed to find user: %v", err),
		})
	}

	// 역할 할당
	err = h.keycloakService.AssignRealmRoleToUser(c.Request().Context(), user.KcId, roleName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": fmt.Sprintf("Failed to assign role: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("Successfully assigned role %s to user %s", roleName, user.Username),
	})
}
