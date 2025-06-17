package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository" // Keep repository import
	"github.com/m-cmp/mc-iam-manager/service"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm" // Import gorm
)

// MciamPermissionHandler MC-IAM 권한 관리 핸들러 - Renamed
type MciamPermissionHandler struct {
	permissionService *service.MciamPermissionService // Use renamed service type
	// db *gorm.DB // Not needed directly in handler
}

// NewMciamPermissionHandler MC-IAM 권한 관리 핸들러 생성 - Renamed
func NewMciamPermissionHandler(db *gorm.DB) *MciamPermissionHandler {
	// Initialize service internally
	permissionService := service.NewMciamPermissionService(db) // Use renamed constructor
	return &MciamPermissionHandler{
		permissionService: permissionService,
	}
}

// ListMciamPermissions MC-IAM 권한 목록 조회 - Renamed
// @Summary List MCIAM permissions
// @Description Get a list of all MCIAM permissions
// @Tags permissions
// @Accept json
// @Produce json
// @Success 200 {array} model.MciamPermission
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/permissions/mciam/list [post]
// @OperationId listMciamPermissions
func (h *MciamPermissionHandler) ListMciamPermissions(c echo.Context) error { // Renamed method
	frameworkID := c.QueryParam("frameworkId")
	resourceTypeID := c.QueryParam("resourceTypeId")
	permissions, err := h.permissionService.List(c.Request().Context(), frameworkID, resourceTypeID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "권한 목록을 가져오는데 실패했습니다",
		})
	}
	return c.JSON(http.StatusOK, permissions) // Return permissions variable
}

// GetByID ID로 권한 조회
// @Summary Get MCIAM permission by ID
// @Description Get MCIAM permission details by ID
// @Tags permissions
// @Accept json
// @Produce json
// @Param id path string true "Permission ID"
// @Success 200 {object} model.MciamPermission
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/permissions/mciam/id/{id} [get]
// @OperationId getMciamPermissionByID
func (h *MciamPermissionHandler) GetMciamPermissionByID(c echo.Context) error { // Renamed method
	id := c.Param("id")

	permission, err := h.permissionService.GetByID(c.Request().Context(), id)
	if err != nil {
		// Handle not found error from service/repo
		if errors.Is(err, repository.ErrPermissionNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
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
// @Summary Create MCIAM permission
// @Description Create a new MCIAM permission
// @Tags permissions
// @Accept json
// @Produce json
// @Param permission body model.MciamPermission true "Permission Info"
// @Success 201 {object} model.MciamPermission
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/permissions/mciam [post]
// @OperationId createMciamPermission
func (h *MciamPermissionHandler) CreateMciamPermission(c echo.Context) error { // Renamed method
	var permission model.MciamPermission // Use renamed model
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
// @Summary Update MCIAM permission
// @Description Update an existing MCIAM permission
// @Tags permissions
// @Accept json
// @Produce json
// @Param id path string true "Permission ID"
// @Param permission body model.MciamPermission true "Permission Info"
// @Success 200 {object} model.MciamPermission
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/permissions/mciam/{id} [put]
// @OperationId updateMciamPermission
func (h *MciamPermissionHandler) UpdateMciamPermission(c echo.Context) error { // Renamed method
	id := c.Param("id")

	updates := make(map[string]interface{})
	if err := c.Bind(&updates); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다: " + err.Error()})
	}

	// Allow updating only specific fields (e.g., name, description)
	allowedUpdates := make(map[string]interface{})
	if name, ok := updates["name"].(string); ok {
		allowedUpdates["name"] = name
	}
	if description, ok := updates["description"].(string); ok {
		allowedUpdates["description"] = description
	}
	// Add other updatable fields if necessary

	if len(allowedUpdates) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "업데이트할 필드(name, description)가 없습니다"})
	}

	if err := h.permissionService.Update(c.Request().Context(), id, allowedUpdates); err != nil {
		// Check for specific errors like not found
		if err == repository.ErrPermissionNotFound { // Assuming repo returns this error
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "권한 수정에 실패했습니다: " + err.Error()})
	}

	// Fetch and return the updated permission
	updatedPermission, err := h.permissionService.GetByID(c.Request().Context(), id)
	if err != nil {
		// Log error, but return the updates map as a fallback? Or return 200 OK with no body?
		// Let's return the updates map for now.
		return c.JSON(http.StatusOK, allowedUpdates)
	}
	return c.JSON(http.StatusOK, updatedPermission)
}

// Delete 권한 삭제
// @Summary Delete MCIAM permission
// @Description Delete an existing MCIAM permission
// @Tags permissions
// @Accept json
// @Produce json
// @Param id path string true "Permission ID"
// @Success 204 "No Content"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/permissions/mciam/{id} [delete]
// @OperationId deleteMciamPermission
func (h *MciamPermissionHandler) DeleteMciamPermission(c echo.Context) error { // Renamed method
	id := c.Param("id")

	if err := h.permissionService.Delete(c.Request().Context(), id); err != nil {
		if errors.Is(err, repository.ErrPermissionNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "권한 삭제에 실패했습니다",
		})
	}
	return c.NoContent(http.StatusNoContent)
}

// AssignRolePermission 역할에 권한 할당
// @Summary 역할에 MC-IAM 권한 할당 - Renamed
// @Description 역할에 MC-IAM 권한을 할당합니다.
// @Tags roles, mciam-permissions
// @Accept json
// @Produce json
// @Param roleType path string true "역할 타입 ('platform' or 'workspace')"
// @Param roleId path int true "역할 ID"
// @Param permissionId path string true "MC-IAM 권한 ID"
// @Success 204 "No Content"
// @Router /roles/{roleType}/{roleId}/mciam-permissions/{permissionId} [post] // Updated route
func (h *MciamPermissionHandler) AssignMciamPermissionToRole(c echo.Context) error { // Renamed method
	roleType := c.Param("roleType")
	if roleType != model.RoleTypePlatform && roleType != model.RoleTypeWorkspace {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 타입입니다. 'platform' 또는 'workspace' 만 가능합니다."})
	}

	roleID, err := strconv.ParseUint(c.Param("roleId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID입니다"})
	}

	permissionID := c.Param("permissionId")

	if err := h.permissionService.AssignMciamPermissionToRole(c.Request().Context(), roleType, uint(roleID), permissionID); err != nil { // Use renamed service method
		// Handle specific errors like permission not found or role not found
		if errors.Is(err, repository.ErrPermissionNotFound) { // Check for specific error
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		// TODO: Add check for role not found error if service implements it
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "권한 할당에 실패했습니다",
		})
	}
	return c.NoContent(http.StatusNoContent)
}

// RemoveRolePermission 역할에서 권한 제거
// @Summary 역할에서 MC-IAM 권한 제거 - Renamed
// @Description 역할에서 MC-IAM 권한을 제거합니다.
// @Tags roles, mciam-permissions
// @Accept json
// @Produce json
// @Param roleType path string true "역할 타입 ('platform' or 'workspace')"
// @Param roleId path int true "역할 ID"
// @Param permissionId path string true "MC-IAM 권한 ID"
// @Success 204 "No Content"
// @Router /roles/{roleType}/{roleId}/mciam-permissions/{permissionId} [delete] // Updated route
func (h *MciamPermissionHandler) RemoveMciamPermissionFromRole(c echo.Context) error { // Renamed method
	roleType := c.Param("roleType")
	if roleType != model.RoleTypePlatform && roleType != model.RoleTypeWorkspace {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 타입입니다. 'platform' 또는 'workspace' 만 가능합니다."})
	}

	roleID, err := strconv.ParseUint(c.Param("roleId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID입니다"})
	}

	permissionID := c.Param("permissionId")

	if err := h.permissionService.RemoveMciamPermissionFromRole(c.Request().Context(), roleType, uint(roleID), permissionID); err != nil { // Use renamed service method
		// Handle specific error like mapping not found
		if err.Error() == "role mciam permission mapping not found" { // Check specific error text from repo
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "권한 제거에 실패했습니다",
		})
	}
	return c.NoContent(http.StatusNoContent)
}

// GetRolePermissions 역할의 권한 목록 조회
// @Summary 역할의 MC-IAM 권한 목록 조회 - Renamed
// @Description 특정 역할의 MC-IAM 권한 ID 목록을 조회합니다.
// @Tags roles, mciam-permissions
// @Accept json
// @Produce json
// @Param roleType path string true "역할 타입 ('platform' or 'workspace')"
// @Param roleId path int true "역할 ID"
// @Success 200 {array} string "권한 ID 목록"
// @Router /roles/{roleType}/{roleId}/mciam-permissions [get] // Updated route
func (h *MciamPermissionHandler) GetRoleMciamPermissions(c echo.Context) error { // Renamed method
	roleType := c.Param("roleType")
	if roleType != model.RoleTypePlatform && roleType != model.RoleTypeWorkspace {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 타입입니다. 'platform' 또는 'workspace' 만 가능합니다."})
	}

	roleID, err := strconv.ParseUint(c.Param("roleId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID입니다"})
	}

	permissionIDs, err := h.permissionService.GetRoleMciamPermissions(c.Request().Context(), roleType, uint(roleID)) // Use renamed service method
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "권한 목록을 가져오는데 실패했습니다",
		})
	}
	return c.JSON(http.StatusOK, permissionIDs) // Return permissionIDs variable
}
