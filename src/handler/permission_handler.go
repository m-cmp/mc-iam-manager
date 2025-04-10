package handler

import (
	"net/http"
	"strconv"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"

	"github.com/labstack/echo/v4"
)

// PermissionHandler 권한 관리 핸들러
type PermissionHandler struct {
	permissionService *service.PermissionService
}

// NewPermissionHandler 권한 관리 핸들러 생성
func NewPermissionHandler(permissionService *service.PermissionService) *PermissionHandler {
	return &PermissionHandler{
		permissionService: permissionService,
	}
}

// List 권한 목록 조회
// @Summary 권한 목록 조회
// @Description 모든 권한 목록을 조회합니다.
// @Tags permissions
// @Accept json
// @Produce json
// @Success 200 {array} model.Permission
// @Router /api/permissions [get]
func (h *PermissionHandler) List(c echo.Context) error {
	permissions, err := h.permissionService.List(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "권한 목록을 가져오는데 실패했습니다",
		})
	}
	return c.JSON(http.StatusOK, permissions)
}

// GetByID ID로 권한 조회
// @Summary ID로 권한 조회
// @Description ID로 특정 권한을 조회합니다.
// @Tags permissions
// @Accept json
// @Produce json
// @Param id path string true "권한 ID" // Changed type to string
// @Success 200 {object} model.Permission
// @Router /api/permissions/{id} [get]
func (h *PermissionHandler) GetByID(c echo.Context) error {
	id := c.Param("id") // Get ID as string
	// No need to parse uint

	permission, err := h.permissionService.GetByID(c.Request().Context(), id) // Pass string id
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "권한을 가져오는데 실패했습니다",
		})
	}
	if permission == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "권한을 찾을 수 없습니다",
		})
	}
	return c.JSON(http.StatusOK, permission)
}

// Create 권한 생성
// @Summary 권한 생성
// @Description 새로운 권한을 생성합니다.
// @Tags permissions
// @Accept json
// @Produce json
// @Param permission body model.Permission true "권한 정보"
// @Success 201 {object} model.Permission
// @Router /api/permissions [post]
func (h *PermissionHandler) Create(c echo.Context) error {
	var permission model.Permission
	if err := c.Bind(&permission); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "잘못된 요청입니다",
		})
	}

	if err := h.permissionService.Create(c.Request().Context(), &permission); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "권한 생성에 실패했습니다",
		})
	}
	return c.JSON(http.StatusCreated, permission)
}

// Update 권한 수정
// @Summary 권한 수정
// @Description 기존 권한을 수정합니다.
// @Tags permissions
// @Accept json
// @Produce json
// @Param id path string true "권한 ID" // Changed type to string
// @Param permission body model.Permission true "권한 정보"
// @Success 200 {object} model.Permission
// @Router /api/permissions/{id} [put]
func (h *PermissionHandler) Update(c echo.Context) error {
	id := c.Param("id") // Get ID as string
	// No need to parse uint

	var permission model.Permission
	if err := c.Bind(&permission); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "잘못된 요청입니다",
		})
	}
	permission.ID = id // Assign string id

	if err := h.permissionService.Update(c.Request().Context(), &permission); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "권한 수정에 실패했습니다",
		})
	}
	return c.JSON(http.StatusOK, permission)
}

// Delete 권한 삭제
// @Summary 권한 삭제
// @Description 권한을 삭제합니다.
// @Tags permissions
// @Accept json
// @Produce json
// @Param id path string true "권한 ID" // Changed type to string
// @Success 204 "No Content"
// @Router /api/permissions/{id} [delete]
func (h *PermissionHandler) Delete(c echo.Context) error {
	id := c.Param("id") // Get ID as string
	// No need to parse uint

	if err := h.permissionService.Delete(c.Request().Context(), id); err != nil { // Pass string id
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "권한 삭제에 실패했습니다",
		})
	}
	return c.NoContent(http.StatusNoContent)
}

// AssignRolePermission 역할에 권한 할당
// @Summary 역할에 권한 할당
// @Description 역할에 권한을 할당합니다.
// @Tags permissions
// @Accept json
// @Produce json
// @Param roleType path string true "역할 타입 ('platform' or 'workspace')"
// @Param roleId path int true "역할 ID"
// @Param permissionId path string true "권한 ID" // Changed type to string
// @Success 204 "No Content"
// @Router /api/roles/{roleType}/{roleId}/permissions/{permissionId} [post] // Added roleType to path
func (h *PermissionHandler) AssignRolePermission(c echo.Context) error {
	roleType := c.Param("roleType") // Get roleType from path
	if roleType != "platform" && roleType != "workspace" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 타입입니다. 'platform' 또는 'workspace'를 사용하세요."})
	}

	roleID, err := strconv.ParseUint(c.Param("roleId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID입니다"})
	}

	permissionID := c.Param("permissionId") // Get permissionID as string
	// No need to parse uint

	if err := h.permissionService.AssignRolePermission(c.Request().Context(), roleType, uint(roleID), permissionID); err != nil { // Pass roleType and string permissionID
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "권한 할당에 실패했습니다",
		})
	}
	return c.NoContent(http.StatusNoContent)
}

// RemoveRolePermission 역할에서 권한 제거
// @Summary 역할에서 권한 제거
// @Description 역할에서 권한을 제거합니다.
// @Tags permissions
// @Accept json
// @Produce json
// @Param roleType path string true "역할 타입 ('platform' or 'workspace')"
// @Param roleId path int true "역할 ID"
// @Param permissionId path string true "권한 ID" // Changed type to string
// @Success 204 "No Content"
// @Router /api/roles/{roleType}/{roleId}/permissions/{permissionId} [delete] // Added roleType to path
func (h *PermissionHandler) RemoveRolePermission(c echo.Context) error {
	roleType := c.Param("roleType") // Get roleType from path
	if roleType != "platform" && roleType != "workspace" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 타입입니다. 'platform' 또는 'workspace'를 사용하세요."})
	}

	roleID, err := strconv.ParseUint(c.Param("roleId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID입니다"})
	}

	permissionID := c.Param("permissionId") // Get permissionID as string
	// No need to parse uint

	if err := h.permissionService.RemoveRolePermission(c.Request().Context(), roleType, uint(roleID), permissionID); err != nil { // Pass roleType and string permissionID
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "권한 제거에 실패했습니다",
		})
	}
	return c.NoContent(http.StatusNoContent)
}

// GetRolePermissions 역할의 권한 목록 조회
// @Summary 역할의 권한 목록 조회
// @Description 특정 역할의 권한 목록을 조회합니다.
// @Tags permissions
// @Accept json
// @Produce json
// @Param roleType path string true "역할 타입 ('platform' or 'workspace')"
// @Param roleId path int true "역할 ID"
// @Success 200 {array} model.Permission
// @Router /api/roles/{roleType}/{roleId}/permissions [get] // Added roleType to path
func (h *PermissionHandler) GetRolePermissions(c echo.Context) error {
	roleType := c.Param("roleType") // Get roleType from path
	if roleType != "platform" && roleType != "workspace" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 타입입니다. 'platform' 또는 'workspace'를 사용하세요."})
	}

	roleID, err := strconv.ParseUint(c.Param("roleId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID입니다"})
	}

	permissions, err := h.permissionService.GetRolePermissions(c.Request().Context(), roleType, uint(roleID)) // Pass roleType
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "권한 목록을 가져오는데 실패했습니다",
		})
	}
	return c.JSON(http.StatusOK, permissions)
}
