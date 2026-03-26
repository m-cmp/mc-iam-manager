package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm"
)

// GroupRoleHandler 그룹 역할 관리 핸들러
type GroupRoleHandler struct {
	groupRoleService *service.GroupRoleService
	db               *gorm.DB
}

// NewGroupRoleHandler GroupRoleHandler 생성자
func NewGroupRoleHandler(db *gorm.DB) *GroupRoleHandler {
	return &GroupRoleHandler{
		groupRoleService: service.NewGroupRoleService(db),
		db:               db,
	}
}

// getUserKcID DB에서 사용자의 Keycloak ID를 조회합니다
func (h *GroupRoleHandler) getUserKcID(userID uint) string {
	var user model.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return ""
	}
	return user.KcId
}

// AssignGroupPlatformRole godoc
// @Summary 그룹에 Platform Role 할당
// @Description 그룹에 플랫폼 역할을 할당합니다. DB + Keycloak 이중 관리.
// @Tags groups
// @Accept json
// @Produce json
// @Param groupId path int true "그룹 ID"
// @Param body body model.AssignGroupPlatformRoleRequest true "역할 할당 요청"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Security BearerAuth
// @Router /api/groups/id/{groupId}/platform-roles [post]
// @Id assignGroupPlatformRole
func (h *GroupRoleHandler) AssignGroupPlatformRole(c echo.Context) error {
	groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid group ID"})
	}

	var req model.AssignGroupPlatformRoleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := h.groupRoleService.AssignGroupPlatformRole(c.Request().Context(), uint(groupID), req.RoleID); err != nil {
		switch {
		case errors.Is(err, repository.ErrOrganizationNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{"error": "그룹을 찾을 수 없습니다"})
		case errors.Is(err, repository.ErrRoleMasterNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{"error": "플랫폼 역할을 찾을 수 없습니다"})
		case errors.Is(err, repository.ErrGroupPlatformRoleDuplicate):
			return c.JSON(http.StatusConflict, map[string]string{"error": "이미 할당된 역할입니다"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return c.JSON(http.StatusCreated, map[string]string{"message": "그룹에 플랫폼 역할이 할당되었습니다."})
}

// GetGroupPlatformRoles godoc
// @Summary 그룹 Platform Role 목록 조회
// @Description 그룹에 할당된 플랫폼 역할 목록을 조회합니다.
// @Tags groups
// @Produce json
// @Param groupId path int true "그룹 ID"
// @Success 200 {array} model.GroupPlatformRoleResponse
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/groups/id/{groupId}/platform-roles [get]
// @Id getGroupPlatformRoles
func (h *GroupRoleHandler) GetGroupPlatformRoles(c echo.Context) error {
	groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid group ID"})
	}

	roles, err := h.groupRoleService.GetGroupPlatformRoles(uint(groupID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, roles)
}

// GetAvailableGroupPlatformRoles godoc
// @Summary 그룹에 미할당된 Platform Role 목록 조회
// @Description 그룹에 아직 할당되지 않은 플랫폼 역할 목록을 조회합니다.
// @Tags groups
// @Produce json
// @Param groupId path int true "그룹 ID"
// @Success 200 {array} model.AvailablePlatformRoleResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /api/groups/id/{groupId}/platform-roles/available [get]
// @Id getAvailableGroupPlatformRoles
func (h *GroupRoleHandler) GetAvailableGroupPlatformRoles(c echo.Context) error {
	groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "그룹 ID가 올바르지 않습니다"})
	}

	roles, err := h.groupRoleService.GetAvailableGroupPlatformRoles(uint(groupID))
	if err != nil {
		if errors.Is(err, repository.ErrOrganizationNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "그룹을 찾을 수 없습니다"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, roles)
}

// RemoveGroupPlatformRole godoc
// @Summary 그룹 Platform Role 해제
// @Description 그룹에 할당된 플랫폼 역할을 해제합니다. DB + Keycloak 동시 제거.
// @Tags groups
// @Produce json
// @Param groupId path int true "그룹 ID"
// @Param roleId path int true "역할 ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /api/groups/id/{groupId}/platform-roles/{roleId} [delete]
// @Id removeGroupPlatformRole
func (h *GroupRoleHandler) RemoveGroupPlatformRole(c echo.Context) error {
	groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid group ID"})
	}
	roleID, err := strconv.ParseUint(c.Param("roleId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid role ID"})
	}

	if err := h.groupRoleService.RemoveGroupPlatformRole(c.Request().Context(), uint(groupID), uint(roleID)); err != nil {
		switch {
		case errors.Is(err, repository.ErrOrganizationNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{"error": "그룹을 찾을 수 없습니다"})
		case errors.Is(err, repository.ErrRoleMasterNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{"error": "플랫폼 역할을 찾을 수 없습니다"})
		case errors.Is(err, repository.ErrGroupPlatformRoleNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{"error": "할당된 역할 매핑을 찾을 수 없습니다"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "그룹의 플랫폼 역할이 해제되었습니다."})
}

// AssignGroupWorkspace godoc
// @Summary 그룹-워크스페이스 매핑
// @Description 그룹을 워크스페이스에 매핑하고 역할을 지정합니다. DB 전용 관리.
// @Tags groups
// @Accept json
// @Produce json
// @Param groupId path int true "그룹 ID"
// @Param body body model.AssignGroupWorkspaceRequest true "워크스페이스 매핑 요청"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Security BearerAuth
// @Router /api/groups/id/{groupId}/workspaces [post]
// @Id assignGroupWorkspace
func (h *GroupRoleHandler) AssignGroupWorkspace(c echo.Context) error {
	groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid group ID"})
	}

	var req model.AssignGroupWorkspaceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := h.groupRoleService.AssignGroupWorkspace(uint(groupID), req.WorkspaceID, req.RoleID); err != nil {
		if errors.Is(err, repository.ErrGroupWorkspaceRoleDuplicate) {
			return c.JSON(http.StatusConflict, map[string]string{"error": "이미 매핑된 워크스페이스입니다"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]string{"message": "그룹이 워크스페이스에 매핑되었습니다."})
}

// GetGroupWorkspaces godoc
// @Summary 그룹 워크스페이스 매핑 목록 조회
// @Description 그룹에 매핑된 워크스페이스 및 역할 목록을 조회합니다.
// @Tags groups
// @Produce json
// @Param groupId path int true "그룹 ID"
// @Success 200 {array} model.GroupWorkspaceRoleResponse
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/groups/id/{groupId}/workspaces [get]
// @Id getGroupWorkspaces
func (h *GroupRoleHandler) GetGroupWorkspaces(c echo.Context) error {
	groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid group ID"})
	}

	workspaces, err := h.groupRoleService.GetGroupWorkspaces(uint(groupID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, workspaces)
}

// UpdateGroupWorkspaceRole godoc
// @Summary 그룹 워크스페이스 역할 변경
// @Description 그룹-워크스페이스 매핑의 역할을 변경합니다.
// @Tags groups
// @Accept json
// @Produce json
// @Param groupId path int true "그룹 ID"
// @Param workspaceId path int true "워크스페이스 ID"
// @Param body body model.UpdateGroupWorkspaceRoleRequest true "역할 변경 요청"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /api/groups/id/{groupId}/workspaces/{workspaceId} [put]
// @Id updateGroupWorkspaceRole
func (h *GroupRoleHandler) UpdateGroupWorkspaceRole(c echo.Context) error {
	groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid group ID"})
	}
	workspaceID, err := strconv.ParseUint(c.Param("workspaceId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid workspace ID"})
	}

	var req model.UpdateGroupWorkspaceRoleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := h.groupRoleService.UpdateGroupWorkspaceRole(uint(groupID), uint(workspaceID), req.RoleID); err != nil {
		if errors.Is(err, repository.ErrGroupWorkspaceRoleNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "매핑을 찾을 수 없습니다"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "그룹 워크스페이스 역할이 변경되었습니다."})
}

// RemoveGroupWorkspaceRole godoc
// @Summary 그룹-워크스페이스 매핑 제거
// @Description 그룹-워크스페이스 매핑을 제거합니다.
// @Tags groups
// @Produce json
// @Param groupId path int true "그룹 ID"
// @Param workspaceId path int true "워크스페이스 ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /api/groups/id/{groupId}/workspaces/{workspaceId} [delete]
// @Id removeGroupWorkspaceRole
func (h *GroupRoleHandler) RemoveGroupWorkspaceRole(c echo.Context) error {
	groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid group ID"})
	}
	workspaceID, err := strconv.ParseUint(c.Param("workspaceId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid workspace ID"})
	}

	if err := h.groupRoleService.RemoveGroupWorkspaceRole(uint(groupID), uint(workspaceID)); err != nil {
		if errors.Is(err, repository.ErrGroupWorkspaceRoleNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "매핑을 찾을 수 없습니다"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "그룹-워크스페이스 매핑이 제거되었습니다."})
}

// AssignGroupUsers godoc
// @Summary 그룹에 사용자 일괄 할당 (Keycloak 동기화 포함)
// @Description 그룹에 사용자 목록을 일괄 할당합니다. DB + Keycloak 그룹 동기화.
// @Tags groups
// @Accept json
// @Produce json
// @Param groupId path int true "그룹 ID"
// @Param body body model.AssignGroupUsersRequest true "사용자 일괄 할당 요청"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /api/groups/id/{groupId}/users [post]
// @Id assignGroupUsers
func (h *GroupRoleHandler) AssignGroupUsers(c echo.Context) error {
	groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid group ID"})
	}

	var req model.AssignGroupUsersRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := h.groupRoleService.AssignUsersToGroup(c.Request().Context(), uint(groupID), req.UserIDs); err != nil {
		switch {
		case errors.Is(err, repository.ErrOrganizationNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{"error": "그룹을 찾을 수 없습니다"})
		default:
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "사용자가 그룹에 할당되었습니다."})
}

// RemoveGroupUser godoc
// @Summary 그룹에서 사용자 제거 (Keycloak 동기화 포함)
// @Description 그룹에서 특정 사용자를 제거합니다. DB + Keycloak 그룹 동기화.
// @Tags groups
// @Produce json
// @Param groupId path int true "그룹 ID"
// @Param userId path int true "사용자 ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /api/groups/id/{groupId}/users/{userId} [delete]
// @Id removeGroupUser
func (h *GroupRoleHandler) RemoveGroupUser(c echo.Context) error {
	groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid group ID"})
	}
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
	}

	kcUserID := h.getUserKcID(uint(userID))

	if err := h.groupRoleService.RemoveUserFromGroup(c.Request().Context(), uint(userID), uint(groupID), kcUserID); err != nil {
		switch {
		case errors.Is(err, repository.ErrOrganizationNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{"error": "그룹을 찾을 수 없습니다"})
		case errors.Is(err, repository.ErrUserOrganizationNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{"error": "사용자가 해당 그룹에 소속되어 있지 않습니다"})
		default:
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "사용자가 그룹에서 제거되었습니다."})
}

// AssignUserGroups godoc
// @Summary 사용자를 그룹에 할당 (Keycloak 동기화 포함)
// @Description 사용자를 하나 이상의 그룹에 할당합니다. DB + Keycloak 그룹 동기화.
// @Tags groups
// @Accept json
// @Produce json
// @Param userId path int true "사용자 ID"
// @Param body body model.AssignUserGroupsRequest true "그룹 할당 요청"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/users/id/{userId}/groups [post]
// @Id assignUserGroups
func (h *GroupRoleHandler) AssignUserGroups(c echo.Context) error {
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
	}

	var req model.AssignUserGroupsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	kcUserID := h.getUserKcID(uint(userID))

	if err := h.groupRoleService.AssignUserToGroups(c.Request().Context(), uint(userID), req.GroupIDs, kcUserID); err != nil {
		switch {
		case errors.Is(err, repository.ErrOrganizationNotFound):
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "사용자가 그룹에 할당되었습니다."})
}

// RemoveUserFromGroup godoc
// @Summary 사용자를 그룹에서 제거 (Keycloak 동기화 포함)
// @Description 사용자를 특정 그룹에서 제거합니다. DB + Keycloak 그룹 동기화.
// @Tags groups
// @Produce json
// @Param userId path int true "사용자 ID"
// @Param groupId path int true "그룹 ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/users/id/{userId}/groups/{groupId} [delete]
// @Id removeUserFromGroup
func (h *GroupRoleHandler) RemoveUserFromGroup(c echo.Context) error {
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
	}
	groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid group ID"})
	}

	kcUserID := h.getUserKcID(uint(userID))

	if err := h.groupRoleService.RemoveUserFromGroup(c.Request().Context(), uint(userID), uint(groupID), kcUserID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "사용자가 그룹에서 제거되었습니다."})
}
