package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm"
)

// CreateRoleRequest 역할 생성 요청 구조체
type CreateRoleRequest struct {
	Name        string   `json:"name" validate:"required"`
	Description string   `json:"description"`
	ParentID    *uint    `json:"parent_id"`
	RoleTypes   []string `json:"role_types" validate:"required,dive,oneof=platform workspace"`
}

// RoleHandler 통합 역할 관리 핸들러
type RoleHandler struct {
	service         *service.RoleService
	userService     *service.UserService
	keycloakService service.KeycloakService
}

// NewRoleHandler 새 RoleHandler 인스턴스 생성
func NewRoleHandler(db *gorm.DB) *RoleHandler {
	roleService := service.NewRoleService(db)
	userService := service.NewUserService(db)
	keycloakService := service.NewKeycloakService()
	return &RoleHandler{
		service:         roleService,
		userService:     userService,
		keycloakService: keycloakService,
	}
}

// List 모든 역할 목록 조회
func (h *RoleHandler) RoleList(c echo.Context) error {
	roleType := c.QueryParam("type")
	roles, err := h.service.List(roleType)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, roles)
}

// Create 새 역할 생성
func (h *RoleHandler) RoleCreate(c echo.Context) error {
	var req CreateRoleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식"})
	}

	// 입력값 검증
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("입력값 검증 실패: %v", err)})
	}

	// 역할 생성
	role := model.RoleMaster{
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
	}

	createdRole, err := h.service.CreateRoleWithSubs(role, req.RoleTypes)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, createdRole)
}

// GetRoleByID ID로 역할 조회
func (h *RoleHandler) GetRoleByID(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 ID 형식"})
	}

	role, err := h.service.GetByID(uint(id))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if role == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "역할을 찾을 수 없습니다"})
	}

	return c.JSON(http.StatusOK, role)
}

// RoleUpdate 역할 정보 수정
func (h *RoleHandler) RoleUpdate(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 ID 형식"})
	}

	var req struct {
		Name        string   `json:"name" validate:"required"`
		Description string   `json:"description"`
		ParentID    *uint    `json:"parent_id"`
		RoleTypes   []string `json:"role_types" validate:"required,dive,oneof=platform workspace"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("입력값 검증 실패: %v", err)})
	}

	role := model.RoleMaster{
		ID:          uint(id),
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
	}

	updatedRole, err := h.service.UpdateRoleWithSubs(role, req.RoleTypes)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, updatedRole)
}

// RoleDelete 역할 삭제
func (h *RoleHandler) RoleDelete(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 ID 형식"})
	}

	// 역할 조회
	role, err := h.service.GetByID(uint(id))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if role == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "역할을 찾을 수 없습니다"})
	}

	// Predefined 역할은 삭제 불가
	if role.Predefined {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "미리 정의된 역할은 삭제할 수 없습니다"})
	}

	if err := h.service.DeleteRoleWithSubs(uint(id)); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

// AssignRoleRequest 역할 할당 요청 구조체
type AssignRoleRequest struct {
	UserID      uint   `json:"userId"`
	Username    string `json:"username"`
	RoleID      uint   `json:"roleId"`
	RoleType    string `json:"roleType"`
	WorkspaceID uint   `json:"workspaceId"`
}

// RemoveRoleRequest 역할 제거 요청 구조체
type RemoveRoleRequest struct {
	UserID      uint   `json:"user_id" validate:"required"`
	RoleID      uint   `json:"role_id" validate:"required"`
	RoleType    string `json:"role_type" validate:"required,oneof=platform workspace"`
	WorkspaceID *uint  `json:"workspace_id"`
}

// AssignRole 역할 할당
func (h *RoleHandler) AssignRole(c echo.Context) error {
	var req AssignRoleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("입력값 검증 실패: %v", err)})
	}

	// 역할이 존재하는지 확인
	role, err := h.service.GetByID(req.RoleID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if role == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "역할을 찾을 수 없습니다"})
	}

	// 역할 타입이 지원되는지 확인
	var hasRoleType bool
	for _, sub := range role.RoleSubs {
		if sub.RoleType == req.RoleType {
			hasRoleType = true
			break
		}
	}
	if !hasRoleType {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("해당 역할은 %s 타입을 지원하지 않습니다", req.RoleType)})
	}

	// 역할 할당
	if req.RoleType == "platform" {
		if err := h.service.AssignPlatformRole(req.UserID, req.RoleID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	} else if req.RoleType == "workspace" {
		if req.WorkspaceID == 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "워크스페이스 ID가 필요합니다"})
		}
		if err := h.service.AssignWorkspaceRole(req.UserID, req.WorkspaceID, req.RoleID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "역할이 성공적으로 할당되었습니다"})
}

// RemoveRole 역할 제거
func (h *RoleHandler) RemoveRole(c echo.Context) error {
	var req RemoveRoleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("입력값 검증 실패: %v", err)})
	}

	// 역할이 존재하는지 확인
	role, err := h.service.GetByID(req.RoleID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if role == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "역할을 찾을 수 없습니다"})
	}

	// 역할 타입이 지원되는지 확인
	var hasRoleType bool
	for _, sub := range role.RoleSubs {
		if sub.RoleType == req.RoleType {
			hasRoleType = true
			break
		}
	}
	if !hasRoleType {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("해당 역할은 %s 타입을 지원하지 않습니다", req.RoleType)})
	}

	// 역할 제거
	if req.RoleType == "platform" {
		if err := h.service.RemovePlatformRole(req.UserID, req.RoleID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	} else if req.RoleType == "workspace" {
		if req.WorkspaceID == nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "워크스페이스 ID가 필요합니다"})
		}
		if err := h.service.RemoveWorkspaceRole(req.UserID, *req.WorkspaceID, req.RoleID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "역할이 성공적으로 제거되었습니다"})
}

// GetUserWorkspaceRoles 사용자의 워크스페이스 역할 목록 조회
func (h *RoleHandler) GetUserWorkspaceRoles(c echo.Context) error {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 사용자 ID 형식"})
	}

	workspaceID, err := strconv.ParseUint(c.Param("workspace_id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식"})
	}

	roles, err := h.service.GetUserWorkspaceRoles(uint(userID), uint(workspaceID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, roles)
}

// AssignPlatformRole 플랫폼 역할 할당
func (h *RoleHandler) AssignPlatformRole(c echo.Context) error {
	var req AssignRoleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	if req.UserID == 0 || req.RoleID == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID와 역할 ID가 필요합니다"})
	}

	if err := h.service.AssignPlatformRole(req.UserID, req.RoleID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "플랫폼 역할이 성공적으로 할당되었습니다"})
}

// RemovePlatformRole 플랫폼 역할 제거
func (h *RoleHandler) RemovePlatformRole(c echo.Context) error {
	var req AssignRoleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	if req.UserID == 0 || req.RoleID == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID와 역할 ID가 필요합니다"})
	}

	if err := h.service.RemovePlatformRole(req.UserID, req.RoleID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "플랫폼 역할이 성공적으로 제거되었습니다"})
}

// AssignWorkspaceRole 워크스페이스 역할 할당
func (h *RoleHandler) AssignWorkspaceRole(c echo.Context) error {
	var req AssignRoleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	if req.UserID == 0 || req.RoleID == 0 || req.WorkspaceID == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID, 역할 ID, 워크스페이스 ID가 필요합니다"})
	}

	if err := h.service.AssignWorkspaceRole(req.UserID, req.WorkspaceID, req.RoleID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "워크스페이스 역할이 성공적으로 할당되었습니다"})
}

// RemoveWorkspaceRole 워크스페이스 역할 제거
func (h *RoleHandler) RemoveWorkspaceRole(c echo.Context) error {
	var req AssignRoleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	if req.UserID == 0 || req.RoleID == 0 || req.WorkspaceID == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID, 역할 ID, 워크스페이스 ID가 필요합니다"})
	}

	if err := h.service.RemoveWorkspaceRole(req.UserID, req.WorkspaceID, req.RoleID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "워크스페이스 역할이 성공적으로 제거되었습니다"})
}
