package handler

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
	"github.com/m-cmp/mc-iam-manager/util"
	"gorm.io/gorm"
)

// RoleHandler 역할 관리 핸들러
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

// @Summary List all roles
// @Description Get a list of all roles
// @Tags roles
// @Accept json
// @Produce json
// @Success 200 {array} model.Role
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles [get]
func (h *RoleHandler) ListRoles(c echo.Context) error {
	roleType := c.QueryParam("roleType")
	log.Printf("역할 목록 조회 요청 - 타입: %s", roleType)

	roles, err := h.service.ListRoles(roleType)
	if err != nil {
		log.Printf("역할 목록 조회 실패: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 목록 조회 실패: %v", err)})
	}

	log.Printf("역할 목록 조회 성공 - 조회된 역할 수: %d", len(roles))
	return c.JSON(http.StatusOK, roles)
}

// @Summary Create role
// @Description Create a new role
// @Tags roles
// @Accept json
// @Produce json
// @Param role body model.RoleMasterSubRequest true "Role Info"
// @Success 201 {object} model.Role
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/createRole [post]
func (h *RoleHandler) CreateRole(c echo.Context) error {
	var req model.RoleMasterSubRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("역할 생성 요청 바인딩 실패: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	log.Printf("역할 생성 요청 - 이름: %s, 설명: %s, 부모ID: %d, 역할타입: %v",
		req.Name, req.Description, req.ParentID, req.RoleTypes)

	// 입력값 검증
	if err := c.Validate(&req); err != nil {
		log.Printf("역할 생성 입력값 검증 실패: %v", err)
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
		log.Printf("역할 생성 실패: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 생성 실패: %v", err)})
	}

	log.Printf("역할 생성 성공 - ID: %d", createdRole.ID)
	return c.JSON(http.StatusCreated, createdRole)
}

// @Summary Get role by ID
// @Description Get role details by ID
// @Tags roles
// @Accept json
// @Produce json
// @Param id path string true "Role ID"
// @Success 200 {object} model.Role
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/{id} [get]
func (h *RoleHandler) GetRoleByID(c echo.Context) error {
	roleType := c.QueryParam("roleType")
	log.Printf("역할 목록 조회 요청 - 타입: %s", roleType)

	id, err := strconv.ParseUint(c.Param("roleId"), 10, 32)
	if err != nil {
		log.Printf("잘못된 역할 ID 형식: %s", c.Param("id"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 ID 형식입니다"})
	}

	log.Printf("역할 조회 요청 - ID: %d", id)

	role, err := h.service.GetRoleByID(uint(id), roleType)
	if err != nil {
		log.Printf("역할 조회 실패 - ID: %d, 에러: %v", id, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
	}

	if role == nil {
		log.Printf("역할을 찾을 수 없음 - ID: %d", id)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 역할을 찾을 수 없습니다"})
	}

	log.Printf("역할 조회 성공 - ID: %d", id)
	return c.JSON(http.StatusOK, role)
}

// @Summary Get role by Name
// @Description Get role details by Name
// @Tags roles
// @Accept json
// @Produce json
// @Param name path string true "Role Name"
// @Success 200 {object} model.Role
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/name/{name} [get]
func (h *RoleHandler) GetRoleByName(c echo.Context) error {
	roleType := c.QueryParam("roleType")
	log.Printf("역할 목록 조회 요청 - 타입: %s", roleType)

	roleName := c.Param("roleName")

	role, err := h.service.GetRoleByName(roleName, roleType)
	if err != nil {
		log.Printf("역할 조회 실패 - Name: %s, 에러: %v", roleName, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
	}

	if role == nil {
		log.Printf("역할을 찾을 수 없음 - ID: %v", roleName)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 역할을 찾을 수 없습니다"})
	}

	return c.JSON(http.StatusOK, role)
}

// @Summary Update role
// @Description Update role information
// @Tags roles
// @Accept json
// @Produce json
// @Param id path string true "Role ID"
// @Param role body model.RoleMasterSubRequest true "Role Info"
// @Success 200 {object} model.Role
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/{id} [put]
func (h *RoleHandler) UpdateRole(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("roleId"), 10, 32)
	if err != nil {
		log.Printf("잘못된 역할 ID 형식: %s", c.Param("id"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 ID 형식입니다"})
	}

	var req model.RoleMasterSubRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("역할 수정 요청 바인딩 실패 - ID: %d, 에러: %v", id, err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	log.Printf("역할 수정 요청 - ID: %d, 이름: %s, 설명: %s, 부모ID: %d, 역할타입: %v",
		id, req.Name, req.Description, req.ParentID, req.RoleTypes)

	if err := c.Validate(&req); err != nil {
		log.Printf("역할 수정 입력값 검증 실패 - ID: %d, 에러: %v", id, err)
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
		log.Printf("역할 수정 실패 - ID: %d, 에러: %v", id, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 수정 실패: %v", err)})
	}

	log.Printf("역할 수정 성공 - ID: %d", id)
	return c.JSON(http.StatusOK, updatedRole)
}

// @Summary Delete role
// @Description Delete a role
// @Tags roles
// @Accept json
// @Produce json
// @Param id path string true "Role ID"
// @Success 204 "No Content"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/{id} [delete]
func (h *RoleHandler) DeleteRole(c echo.Context) error {
	roleIdInt, err := util.StringToUint(c.Param("roleId"))
	if err != nil {

		log.Printf("잘못된 역할 ID 형식: %s", c.Param("id"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 ID 형식입니다"})
	}
	log.Printf("역할 삭제 요청 - ID: %d", roleIdInt)
	var req model.AssignRoleRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("역할 삭제 요청 바인딩 실패: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	// 역할 조회
	role, err := h.service.GetRoleByID(roleIdInt, req.RoleType)
	if err != nil {
		log.Printf("역할 조회 실패 - ID: %d, 에러: %v", roleIdInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
	}

	if role == nil {
		log.Printf("역할을 찾을 수 없음 - ID: %d", roleIdInt)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 역할을 찾을 수 없습니다"})
	}

	// Predefined 역할은 삭제 불가
	if role.Predefined {
		log.Printf("미리 정의된 역할 삭제 시도 - ID: %d", roleIdInt)
		return c.JSON(http.StatusForbidden, map[string]string{"error": "미리 정의된 역할은 삭제할 수 없습니다"})
	}

	if err := h.service.DeleteRoleWithSubs(roleIdInt); err != nil {
		log.Printf("역할 삭제 실패 - ID: %d, 에러: %v", roleIdInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 삭제 실패: %v", err)})
	}

	log.Printf("역할 삭제 성공 - ID: %d", roleIdInt)
	return c.NoContent(http.StatusNoContent)
}

// @Summary Assign role
// @Description Assign a role to a user
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.AssignRoleRequest true "Role Assignment Info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/assignRole [post]
func (h *RoleHandler) AssignRole(c echo.Context) error {
	var req model.AssignRoleRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("역할 할당 요청 바인딩 실패: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	log.Printf("역할 할당 요청 - 사용자ID: %d, 사용자명: %s, 역할ID: %d, 역할타입: %s, 워크스페이스ID: %d",
		req.UserID, req.Username, req.RoleID, req.RoleType, req.WorkspaceID)

	if err := c.Validate(&req); err != nil {
		log.Printf("역할 할당 입력값 검증 실패: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("입력값 검증 실패: %v", err)})
	}

	// 사용자 ID 처리
	var userID uint
	var err error
	if req.UserID != "" {
		userID, err = util.StringToUint(req.UserID)
		if err != nil {
			log.Printf("사용자 ID 변환 오류: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 사용자 ID 형식입니다"})
		}
	} else if req.Username != "" {
		// username으로 사용자 조회
		user, err := h.userService.GetUserByUsername(c.Request().Context(), req.Username)
		if err != nil {
			log.Printf("사용자 조회 실패 - 사용자명: %s, 에러: %v", req.Username, err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("사용자 조회 실패: %v", err)})
		}
		if user == nil {
			log.Printf("사용자를 찾을 수 없음 - 사용자명: %s", req.Username)
			return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 사용자명의 사용자를 찾을 수 없습니다"})
		}
		userID = user.ID
	} else {
		log.Printf("사용자 식별자 누락 - UserID: %d, Username: %s", req.UserID, req.Username)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID 또는 사용자명이 필요합니다"})
	}

	var roleID uint
	if req.RoleID != "" {
		roleID, err = util.StringToUint(req.RoleID)
		if err != nil {
			log.Printf("역할 ID 변환 오류: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID 형식입니다"})
		}
	}
	// 역할이 존재하는지 확인
	role, err := h.service.GetRoleByID(roleID, req.RoleType)
	if err != nil {
		log.Printf("역할 조회 실패 - 역할ID: %d, 에러: %v", req.RoleID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
	}
	if role == nil {
		log.Printf("역할을 찾을 수 없음 - 역할ID: %d", req.RoleID)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 역할을 찾을 수 없습니다"})
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
		log.Printf("지원하지 않는 역할 타입 - 역할ID: %d, 요청된 타입: %s", req.RoleID, req.RoleType)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("해당 역할은 %s 타입을 지원하지 않습니다", req.RoleType)})
	}

	var workspaceID uint
	if req.WorkspaceID != "" {
		workspaceID, err = util.StringToUint(req.WorkspaceID)
		if err != nil {
			log.Printf("워크스페이스 ID 변환 오류: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식입니다"})
		}
	}

	// 역할 할당
	if req.RoleType == "platform" {
		if err := h.service.AssignPlatformRole(userID, roleID); err != nil {
			log.Printf("플랫폼 역할 할당 실패 - 사용자ID: %d, 역할ID: %d, 에러: %v", userID, req.RoleID, err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("플랫폼 역할 할당 실패: %v", err)})
		}
	} else if req.RoleType == "workspace" {
		if workspaceID == 0 {
			log.Printf("워크스페이스 ID 누락 - 사용자ID: %d, 역할ID: %d", userID, req.RoleID)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "워크스페이스 ID가 필요합니다"})
		}
		if err := h.service.AssignWorkspaceRole(userID, workspaceID, roleID); err != nil {
			log.Printf("워크스페이스 역할 할당 실패 - 사용자ID: %d, 워크스페이스ID: %d, 역할ID: %d, 에러: %v",
				userID, workspaceID, req.RoleID, err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 역할 할당 실패: %v", err)})
		}
	}

	log.Printf("역할 할당 성공 - 사용자ID: %d, 역할ID: %d, 역할타입: %s", userID, req.RoleID, req.RoleType)
	return c.JSON(http.StatusOK, map[string]string{"message": fmt.Sprintf("%s 역할이 성공적으로 할당되었습니다", req.RoleType)})
}

// @Summary Remove role
// @Description Remove a role from a user
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.AssignRoleRequest true "Role Removal Info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/removeRole [post]
func (h *RoleHandler) RemoveRole(c echo.Context) error {
	var req model.AssignRoleRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Error RemoveRole : %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	// 사용자 ID 처리
	var userID uint
	if req.UserID != "" {
		// userId가 있으면 uint로 변환
		userIDInt, err := util.StringToUint(req.UserID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 사용자 ID 형식입니다"})
		}
		userID = userIDInt
	} else if req.Username != "" {
		// userId가 없고 username이 있으면 사용자 조회
		user, err := h.userService.GetUserByUsername(c.Request().Context(), req.Username)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("사용자 조회 실패: %v", err)})
		}
		if user == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 사용자명의 사용자를 찾을 수 없습니다"})
		}
		userID = user.ID
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID 또는 사용자명이 필요합니다"})
	}

	// 역할 ID 처리
	var roleID uint
	if req.RoleID != "" {
		// roleId가 있으면 uint로 변환
		roleIDInt, err := util.StringToUint(req.RoleID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID 형식입니다"})
		}
		roleID = roleIDInt
	} else if req.RoleName != "" {
		// roleId가 없고 roleName이 있으면 역할 조회
		role, err := h.service.GetRoleByName(req.RoleName, req.RoleType)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
		}
		if role == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 역할명의 역할을 찾을 수 없습니다"})
		}
		roleID = role.ID
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "역할 ID 또는 역할명이 필요합니다"})
	}

	// 역할 타입 확인
	if req.RoleType == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "역할 타입이 필요합니다"})
	}

	// 역할 제거
	if req.RoleType == "platform" {
		if err := h.service.RemovePlatformRole(userID, roleID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("플랫폼 역할 제거 실패: %v", err)})
		}
	} else if req.RoleType == "workspace" {
		var workspaceID uint
		if req.WorkspaceID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "워크스페이스 ID가 필요합니다"})
		}
		// WorkspaceID가 있으면 uint로 변환
		workspaceIDInt, err := util.StringToUint(req.WorkspaceID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID 형식입니다"})
		}
		workspaceID = workspaceIDInt

		if err := h.service.RemoveWorkspaceRole(userID, workspaceID, roleID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 역할 제거 실패: %v", err)})
		}
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "지원하지 않는 역할 타입입니다"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": fmt.Sprintf("%s 역할이 성공적으로 제거되었습니다", req.RoleType)})
}

// @Summary List menu roles
// @Description Get a list of all menu roles
// @Tags roles
// @Accept json
// @Produce json
// @Success 200 {array} model.Role
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/menu-roles [get]
func (h *RoleHandler) ListMenuRoles(c echo.Context) error {
	roleType := "menu"

	roles, err := h.service.ListRoles(roleType)
	if err != nil {
		log.Printf("역할 목록 조회 실패: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 목록 조회 실패: %v", err)})
	}

	log.Printf("역할 목록 조회 성공 - 조회된 역할 수: %d", len(roles))
	return c.JSON(http.StatusOK, roles)
}

// @Summary List workspace roles
// @Description Get a list of all workspace roles
// @Tags roles
// @Accept json
// @Produce json
// @Success 200 {array} model.Role
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/workspace-roles [get]
func (h *RoleHandler) ListWorkspaceRoles(c echo.Context) error {
	roleType := "workspace"

	roles, err := h.service.ListRoles(roleType)
	if err != nil {
		log.Printf("역할 목록 조회 실패: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 목록 조회 실패: %v", err)})
	}

	log.Printf("역할 목록 조회 성공 - 조회된 역할 수: %d", len(roles))
	return c.JSON(http.StatusOK, roles)
}

// @Summary List csp roles
// @Description Get a list of all csp roles
// @Tags roles
// @Accept json
// @Produce json
// @Success 200 {array} model.Role
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/csp-roles [get]
func (h *RoleHandler) ListCspRoles(c echo.Context) error {
	roleType := "csp"

	roles, err := h.service.ListRoles(roleType)
	if err != nil {
		log.Printf("역할 목록 조회 실패: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 목록 조회 실패: %v", err)})
	}

	log.Printf("역할 목록 조회 성공 - 조회된 역할 수: %d", len(roles))
	return c.JSON(http.StatusOK, roles)
}

// @Summary Create menu role
// @Description Create a new menu role
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.RoleMasterSubRequest true "Menu Role Creation Info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/menu-roles [post]
func (h *RoleHandler) CreateMenuRole(c echo.Context) error {
	var req model.RoleMasterSubRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("역할 생성 요청 바인딩 실패: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	log.Printf("역할 생성 요청 - 이름: %s, 설명: %s, 부모ID: %d, 역할타입: %v",
		req.Name, req.Description, req.ParentID, req.RoleTypes)

	// 입력값 검증
	if err := c.Validate(&req); err != nil {
		log.Printf("역할 생성 입력값 검증 실패: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("입력값 검증 실패: %v", err)})
	}

	req.RoleTypes = []string{"menu"}

	// 역할 생성
	role := model.RoleMaster{
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
	}

	createdRole, err := h.service.CreateRoleWithSubs(role, req.RoleTypes)
	if err != nil {
		log.Printf("역할 생성 실패: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 생성 실패: %v", err)})
	}

	log.Printf("역할 생성 성공 - ID: %d", createdRole.ID)
	return c.JSON(http.StatusCreated, createdRole)
}

// @Summary Create workspace role
// @Description Create a new workspace role
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.RoleMasterSubRequest true "Workspace Role Creation Info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/workspace-roles [post]
func (h *RoleHandler) CreateWorkspaceRole(c echo.Context) error {
	var req model.RoleMasterSubRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("역할 생성 요청 바인딩 실패: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	log.Printf("역할 생성 요청 - 이름: %s, 설명: %s, 부모ID: %d, 역할타입: %v",
		req.Name, req.Description, req.ParentID, req.RoleTypes)

	// 입력값 검증
	if err := c.Validate(&req); err != nil {
		log.Printf("역할 생성 입력값 검증 실패: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("입력값 검증 실패: %v", err)})
	}

	req.RoleTypes = []string{"workspace"}

	// 역할 생성
	role := model.RoleMaster{
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
	}

	createdRole, err := h.service.CreateRoleWithSubs(role, req.RoleTypes)
	if err != nil {
		log.Printf("역할 생성 실패: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 생성 실패: %v", err)})
	}

	log.Printf("역할 생성 성공 - ID: %d", createdRole.ID)
	return c.JSON(http.StatusCreated, createdRole)
}

// @Summary Create csp role
// @Description Create a new csp role
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.RoleMasterSubRequest true "CSP Role Creation Info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/csp-roles [post]
func (h *RoleHandler) CreateCspRole(c echo.Context) error {
	var req model.RoleMasterSubRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("역할 생성 요청 바인딩 실패: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	log.Printf("역할 생성 요청 - 이름: %s, 설명: %s, 부모ID: %d, 역할타입: %v",
		req.Name, req.Description, req.ParentID, req.RoleTypes)

	// 입력값 검증
	if err := c.Validate(&req); err != nil {
		log.Printf("역할 생성 입력값 검증 실패: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("입력값 검증 실패: %v", err)})
	}

	req.RoleTypes = []string{"csp"}

	// 역할 생성
	role := model.RoleMaster{
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
	}

	createdRole, err := h.service.CreateRoleWithSubs(role, req.RoleTypes)
	if err != nil {
		log.Printf("역할 생성 실패: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 생성 실패: %v", err)})
	}

	log.Printf("역할 생성 성공 - ID: %d", createdRole.ID)
	return c.JSON(http.StatusCreated, createdRole)
}

// @Summary Get menu role by ID
// @Description Get menu role details by ID
// @Tags roles
// @Accept json
// @Produce json
// @Param id path string true "Menu Role ID"
// @Success 200 {object} model.Role
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/menu-roles/id/{id} [get]
func (h *RoleHandler) GetMenuRoleByID(c echo.Context) error {
	roleType := "menu"
	log.Printf("역할 목록 조회 요청 - 타입: %s", roleType)

	roleIDInt, err := util.StringToUint(c.Param("roleId"))
	if err != nil {
		log.Printf("잘못된 역할 ID 형식: %s", c.Param("roleId"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 ID 형식입니다"})
	}

	log.Printf("역할 조회 요청 - ID: %d", roleIDInt)

	role, err := h.service.GetRoleByID(roleIDInt, roleType)
	if err != nil {
		log.Printf("역할 조회 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
	}

	if role == nil {
		log.Printf("역할을 찾을 수 없음 - ID: %d", roleIDInt)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 역할을 찾을 수 없습니다"})
	}

	log.Printf("역할 조회 성공 - ID: %d", roleIDInt)
	return c.JSON(http.StatusOK, role)
}

// @Summary Get menu role by Name
// @Description Get menu role details by Name
// @Tags roles
// @Accept json
// @Produce json
// @Param name path string true "Menu Role Name"
// @Success 200 {object} model.Role
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/menu-roles/name/{name} [get]
func (h *RoleHandler) GetMenuRoleByName(c echo.Context) error {
	roleType := "menu"
	log.Printf("역할 목록 조회 요청 - 타입: %s", roleType)

	roleName := c.Param("roleName")

	role, err := h.service.GetRoleByName(roleName, roleType)
	if err != nil {
		log.Printf("역할 조회 실패 - Name: %s, 에러: %v", roleName, err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 ID 형식입니다"})
	}

	if role == nil {
		log.Printf("역할을 찾을 수 없음 - Name: %v", roleName)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 역할을 찾을 수 없습니다"})
	}

	log.Printf("역할 조회 성공 - Name: %v", roleName)
	return c.JSON(http.StatusOK, role)
}

// @Summary Get workspace role by ID
// @Description Get workspace role details by ID
// @Tags roles
// @Accept json
// @Produce json
// @Param id path string true "Workspace Role ID"
// @Success 200 {object} model.Role
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/workspace-roles/id/{id} [get]
func (h *RoleHandler) GetWorkspaceRoleByID(c echo.Context) error {
	roleType := "workspace"
	log.Printf("역할 목록 조회 요청 - 타입: %s", roleType)

	roleIDInt, err := util.StringToUint(c.Param("roleId"))
	if err != nil {
		log.Printf("잘못된 역할 ID 형식: %s", c.Param("roleId"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 ID 형식입니다"})
	}

	log.Printf("역할 조회 요청 - ID: %d", roleIDInt)

	role, err := h.service.GetRoleByID(roleIDInt, roleType)
	if err != nil {
		log.Printf("역할 조회 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
	}

	if role == nil {
		log.Printf("역할을 찾을 수 없음 - ID: %d", roleIDInt)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 역할을 찾을 수 없습니다"})
	}

	log.Printf("역할 조회 성공 - ID: %d", roleIDInt)
	return c.JSON(http.StatusOK, role)
}

// @Summary Get workspace role by Name
// @Description Get workspace role details by Name
// @Tags roles
// @Accept json
// @Produce json
// @Param name path string true "Workspace Role Name"
// @Success 200 {object} model.Role
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/workspace-roles/name/{name} [get]
func (h *RoleHandler) GetWorkspaceRoleByName(c echo.Context) error {
	roleType := "workspace"
	log.Printf("역할 목록 조회 요청 - 타입: %s", roleType)

	roleName := c.Param("roleName")

	role, err := h.service.GetRoleByName(roleName, roleType)
	if err != nil {
		log.Printf("역할 조회 실패 - Name: %s, 에러: %v", roleName, err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 ID 형식입니다"})
	}

	if role == nil {
		log.Printf("역할을 찾을 수 없음 - Name: %v", roleName)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 역할을 찾을 수 없습니다"})
	}

	log.Printf("역할 조회 성공 - Name: %v", roleName)
	return c.JSON(http.StatusOK, role)
}

// @Summary Get csp role by ID
// @Description Get csp role details by ID
// @Tags roles
// @Accept json
// @Produce json
// @Param id path string true "CSP Role ID"
// @Success 200 {object} model.Role
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/csp-roles/id/{id} [get]
func (h *RoleHandler) GetCspRoleByID(c echo.Context) error {
	roleType := "csp"
	log.Printf("역할 목록 조회 요청 - 타입: %s", roleType)

	roleIDInt, err := util.StringToUint(c.Param("roleId"))
	if err != nil {
		log.Printf("잘못된 역할 ID 형식: %s", c.Param("roleId"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 ID 형식입니다"})
	}

	log.Printf("역할 조회 요청 - ID: %d", roleIDInt)

	role, err := h.service.GetRoleByID(roleIDInt, roleType)
	if err != nil {
		log.Printf("역할 조회 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
	}

	if role == nil {
		log.Printf("역할을 찾을 수 없음 - ID: %d", roleIDInt)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 역할을 찾을 수 없습니다"})
	}

	log.Printf("역할 조회 성공 - ID: %d", roleIDInt)
	return c.JSON(http.StatusOK, role)
}

// @Summary Get csp role by Name
// @Description Get csp role details by Name
// @Tags roles
// @Accept json
// @Produce json
// @Param name path string true "CSP Role Name"
// @Success 200 {object} model.Role
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/csp-roles/name/{name} [get]
func (h *RoleHandler) GetCspRoleByName(c echo.Context) error {
	roleType := "csp"
	log.Printf("역할 목록 조회 요청 - 타입: %s", roleType)

	roleName := c.Param("roleName")

	role, err := h.service.GetRoleByName(roleName, roleType)
	if err != nil {
		log.Printf("역할 조회 실패 - Name: %s, 에러: %v", roleName, err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 ID 형식입니다"})
	}

	if role == nil {
		log.Printf("역할을 찾을 수 없음 - Name: %v", roleName)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 역할을 찾을 수 없습니다"})
	}

	log.Printf("역할 조회 성공 - Name: %v", roleName)
	return c.JSON(http.StatusOK, role)
}

// @Summary Update csp role
// @Description Update role information
// @Tags roles
// @Accept json
// @Produce json
// @Param roleId path string true "Role ID"
// @Param role body model.RoleMasterSubRequest true "Role Info"
// @Success 200 {object} model.Role
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/csp-roles/{roleId} [put]
func (h *RoleHandler) UpdateCspRole(c echo.Context) error {
	roleType := "csp"

	var req model.RoleMasterSubRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("역할 수정 요청 바인딩 실패 - 에러: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	roleIDInt, err := util.StringToUint(c.Param("roleId"))
	if err != nil {
		log.Printf("잘못된 역할 ID 형식: %s", c.Param("roleId"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 ID 형식입니다"})
	}

	if !util.CheckValueInArray(req.RoleTypes, roleType) { //roleType이 csp인지 확인
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "역할 타입이 일치하지 않습니다"})
	}

	if err := c.Validate(&req); err != nil {
		log.Printf("역할 수정 입력값 검증 실패 - 에러: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("입력값 검증 실패: %v", err)})
	}

	role := model.RoleMaster{
		ID:          roleIDInt,
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
	}

	updatedRole, err := h.service.UpdateRoleWithSubs(role, req.RoleTypes)
	if err != nil {
		log.Printf("역할 수정 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 수정 실패: %v", err)})
	}

	log.Printf("역할 수정 성공 - ID: %d", roleIDInt)
	return c.JSON(http.StatusOK, updatedRole)
}

// @Summary Delete menu role
// @Description Delete a menu role
// @Tags roles
// @Accept json
// @Produce json
// @Param roleId path string true "Menu Role ID"
// @Success 204 "No Content"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/menu-roles/{roleId} [delete]
func (h *RoleHandler) DeleteMenuRole(c echo.Context) error {
	roleType := "menu"

	var req model.RoleMasterSubRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("역할 수정 요청 바인딩 실패 - 에러: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	roleIDInt, err := util.StringToUint(c.Param("roleId"))
	if err != nil {
		log.Printf("잘못된 역할 ID 형식: %s", c.Param("roleId"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 ID 형식입니다"})
	}

	log.Printf("역할 삭제 요청 - ID: %d", roleIDInt)

	// 역할 조회
	role, err := h.service.GetRoleByID(roleIDInt, roleType)
	if err != nil {
		log.Printf("역할 조회 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
	}

	if role == nil {
		log.Printf("역할을 찾을 수 없음 - ID: %d", roleIDInt)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 역할을 찾을 수 없습니다"})
	}

	// Predefined 역할은 삭제 불가
	if role.Predefined {
		log.Printf("미리 정의된 역할 삭제 시도 - ID: %d", roleIDInt)
		return c.JSON(http.StatusForbidden, map[string]string{"error": "미리 정의된 역할은 삭제할 수 없습니다"})
	}

	if err := h.service.DeleteRoleWithSubs(roleIDInt); err != nil {
		log.Printf("역할 삭제 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 삭제 실패: %v", err)})
	}

	log.Printf("역할 삭제 성공 - ID: %d", roleIDInt)
	return c.NoContent(http.StatusNoContent)
}

// @Summary Delete workspace role
// @Description Delete a workspace role
// @Tags roles
// @Accept json
// @Produce json
// @Param roleId path string true "Workspace Role ID"
// @Success 204 "No Content"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/workspace-roles/{roleId} [delete]
func (h *RoleHandler) DeleteWorkspaceRole(c echo.Context) error {
	roleType := "workspace"

	var req model.RoleMasterSubRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("역할 수정 요청 바인딩 실패 - 에러: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	roleIDInt, err := util.StringToUint(c.Param("roleId"))
	if err != nil {
		log.Printf("잘못된 역할 ID 형식: %s", c.Param("roleId"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 ID 형식입니다"})
	}

	log.Printf("역할 삭제 요청 - ID: %d", roleIDInt)

	// 역할 조회
	role, err := h.service.GetRoleByID(roleIDInt, roleType)
	if err != nil {
		log.Printf("역할 조회 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
	}

	if role == nil {
		log.Printf("역할을 찾을 수 없음 - ID: %d", roleIDInt)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 역할을 찾을 수 없습니다"})
	}

	// Predefined 역할은 삭제 불가
	if role.Predefined {
		log.Printf("미리 정의된 역할 삭제 시도 - ID: %d", roleIDInt)
		return c.JSON(http.StatusForbidden, map[string]string{"error": "미리 정의된 역할은 삭제할 수 없습니다"})
	}

	if err := h.service.DeleteRoleWithSubs(roleIDInt); err != nil {
		log.Printf("역할 삭제 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 삭제 실패: %v", err)})
	}

	log.Printf("역할 삭제 성공 - ID: %d", roleIDInt)
	return c.NoContent(http.StatusNoContent)
}

// @Summary Delete csp role
// @Description Delete a role
// @Tags roles
// @Accept json
// @Produce json
// @Param roleId path string true "Role ID"
// @Success 204 "No Content"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/csp-roles/{roleId} [delete]
func (h *RoleHandler) DeleteCspRole(c echo.Context) error {
	roleType := "csp"

	var req model.RoleMasterSubRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("역할 수정 요청 바인딩 실패 - 에러: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	roleIDInt, err := util.StringToUint(c.Param("roleId"))
	if err != nil {
		log.Printf("잘못된 역할 ID 형식: %s", c.Param("roleId"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 ID 형식입니다"})
	}

	log.Printf("역할 삭제 요청 - ID: %d", roleIDInt)

	// 역할 조회
	role, err := h.service.GetRoleByID(roleIDInt, roleType)
	if err != nil {
		log.Printf("역할 조회 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
	}

	if role == nil {
		log.Printf("역할을 찾을 수 없음 - ID: %d", roleIDInt)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 ID의 역할을 찾을 수 없습니다"})
	}

	// Predefined 역할은 삭제 불가
	if role.Predefined {
		log.Printf("미리 정의된 역할 삭제 시도 - ID: %d", roleIDInt)
		return c.JSON(http.StatusForbidden, map[string]string{"error": "미리 정의된 역할은 삭제할 수 없습니다"})
	}

	if err := h.service.DeleteRoleWithSubs(roleIDInt); err != nil {
		log.Printf("역할 삭제 실패 - ID: %d, 에러: %v", roleIDInt, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 삭제 실패: %v", err)})
	}

	log.Printf("역할 삭제 성공 - ID: %d", roleIDInt)
	return c.NoContent(http.StatusNoContent)
}

// @Summary Get user workspace roles
// @Description Get roles assigned to a user in a workspace
// @Tags roles
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {array} model.Role
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/users/{userId}/workspaces/{workspaceId} [get]
func (h *RoleHandler) GetUserWorkspaceRoles(c echo.Context) error {
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 사용자 ID 형식입니다"})
	}

	workspaceID, err := strconv.ParseUint(c.Param("workspaceId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식입니다"})
	}

	roles, err := h.service.GetUserWorkspaceRoles(uint(userID), uint(workspaceID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("워크스페이스 역할 목록 조회 실패: %v", err)})
	}

	return c.JSON(http.StatusOK, roles)
}

// @Summary Assign platform role
// @Description Assign a platform role to a user
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.AssignRoleRequest true "Platform Role Assignment Info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/assignPlatformRole [post]
func (h *RoleHandler) AssignPlatformRole(c echo.Context) error {
	var req model.AssignRoleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	if req.UserID == "" || req.RoleID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID와 역할 ID가 필요합니다"})
	}

	var userID uint
	var roleID uint
	var err error
	if req.UserID != "" {
		userID, err = util.StringToUint(req.UserID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 사용자 ID 형식입니다"})
		}
	}

	if err := h.service.AssignPlatformRole(userID, roleID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("플랫폼 역할 할당 실패: %v", err)})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "플랫폼 역할이 성공적으로 할당되었습니다"})
}

// @Summary Remove platform role
// @Description Remove a platform role from a user
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.RemoveRoleRequest true "Platform Role Removal Info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/removePlatformRole [delete]
func (h *RoleHandler) RemovePlatformRole(c echo.Context) error {
	var req model.AssignRoleRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Error RemovePlatformRole : %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	// 사용자 ID 처리
	var userID uint
	if req.UserID != "" {
		// userId가 있으면 uint로 변환
		userIDInt, err := util.StringToUint(req.UserID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 사용자 ID 형식입니다"})
		}
		userID = userIDInt
	} else if req.Username != "" {
		// userId가 없고 username이 있으면 사용자 조회
		user, err := h.userService.GetUserByUsername(c.Request().Context(), req.Username)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("사용자 조회 실패: %v", err)})
		}
		if user == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 사용자명의 사용자를 찾을 수 없습니다"})
		}
		userID = user.ID
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID 또는 사용자명이 필요합니다"})
	}

	// 역할 ID 처리
	var roleID uint
	if req.RoleID != "" {
		// roleId가 있으면 uint로 변환
		roleIDInt, err := util.StringToUint(req.RoleID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID 형식입니다"})
		}
		roleID = roleIDInt
	} else if req.RoleName != "" {
		// roleId가 없고 roleName이 있으면 역할 조회
		role, err := h.service.GetRoleByName(req.RoleName, req.RoleType)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
		}
		if role == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 역할명의 역할을 찾을 수 없습니다"})
		}
		roleID = role.ID
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "역할 ID 또는 역할명이 필요합니다"})
	}

	// 역할 제거
	if err := h.service.RemovePlatformRole(userID, roleID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("플랫폼 역할 제거 실패: %v", err)})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "플랫폼 역할이 성공적으로 제거되었습니다"})
}

// @Summary Assign workspace role
// @Description Assign a workspace role to a user
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.AssignWorkspaceRoleRequest true "Workspace Role Assignment Info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/assignWorkspaceRole [post]
func (h *RoleHandler) AssignWorkspaceRole(c echo.Context) error {
	roleType := "workspace"
	var req model.AssignWorkspaceRoleRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Error AssignWorkspaceRole : %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	log.Printf("AssignWorkspaceRoleReq : %v", req)

	// 사용자 ID 처리
	var userID uint
	if req.UserID != "" {
		// userId가 있으면 uint로 변환
		userIDInt, err := util.StringToUint(req.UserID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 사용자 ID 형식입니다"})
		}
		userID = userIDInt
	} else if req.Username != "" {
		// userId가 없고 username이 있으면 사용자 조회
		user, err := h.userService.GetUserByUsername(c.Request().Context(), req.Username)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("사용자 조회 실패: %v", err)})
		}
		if user == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 사용자명의 사용자를 찾을 수 없습니다"})
		}
		userID = user.ID
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID 또는 사용자명이 필요합니다"})
	}

	// 역할 ID 처리
	var roleID uint
	if req.RoleID != "" {
		// roleId가 있으면 uint로 변환
		roleIDInt, err := util.StringToUint(req.RoleID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID 형식입니다"})
		}
		roleID = roleIDInt
	} else if req.RoleName != "" {
		// roleId가 없고 roleName이 있으면 역할 조회
		role, err := h.service.GetRoleByName(req.RoleName, roleType)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
		}
		if role == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 역할명의 역할을 찾을 수 없습니다"})
		}
		roleID = role.ID
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "역할 ID 또는 역할명이 필요합니다"})
	}

	// 워크스페이스 ID 처리
	var workspaceID uint
	if req.WorkspaceID != "" {
		// workspaceId가 있으면 uint로 변환
		workspaceIDInt, err := util.StringToUint(req.WorkspaceID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식입니다"})
		}
		workspaceID = workspaceIDInt
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "워크스페이스 ID가 필요합니다"})
	}

	// 역할 할당
	err := h.service.AssignWorkspaceRole(userID, workspaceID, roleID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 할당 실패: %v", err)})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "역할이 성공적으로 할당되었습니다"})
}

// @Summary Remove workspace role
// @Description Remove a workspace role from a user
// @Tags roles
// @Accept json
// @Produce json
// @Param request body model.RemoveWorkspaceRoleRequest true "Workspace Role Removal Info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/removeWorkspaceRole [delete]
func (h *RoleHandler) RemoveWorkspaceRole(c echo.Context) error {
	roleType := "workspace"
	var req model.RemoveWorkspaceRoleRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Error RemoveWorkspaceRole : %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	log.Printf("RemoveWorkspaceRoleReq : %v", req)

	// 사용자 ID 처리
	var userID uint
	if req.UserID != "" {
		// userId가 있으면 uint로 변환
		userIDInt, err := util.StringToUint(req.UserID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 사용자 ID 형식입니다"})
		}
		userID = userIDInt
	} else if req.Username != "" {
		// userId가 없고 username이 있으면 사용자 조회
		user, err := h.userService.GetUserByUsername(c.Request().Context(), req.Username)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("사용자 조회 실패: %v", err)})
		}
		if user == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 사용자명의 사용자를 찾을 수 없습니다"})
		}
		userID = user.ID
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID 또는 사용자명이 필요합니다"})
	}

	// 역할 ID 처리
	var roleID uint
	if req.RoleID != "" {
		// roleId가 있으면 uint로 변환
		roleIDInt, err := util.StringToUint(req.RoleID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 역할 ID 형식입니다"})
		}
		roleID = roleIDInt
	} else if req.RoleName != "" {
		// roleId가 없고 roleName이 있으면 역할 조회
		role, err := h.service.GetRoleByName(req.RoleName, roleType)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 조회 실패: %v", err)})
		}
		if role == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "해당 역할명의 역할을 찾을 수 없습니다"})
		}
		roleID = role.ID
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "역할 ID 또는 역할명이 필요합니다"})
	}

	// 워크스페이스 ID 처리
	var workspaceID uint
	if req.WorkspaceID != "" {
		// workspaceId가 있으면 uint로 변환
		workspaceIDInt, err := util.StringToUint(req.WorkspaceID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식입니다"})
		}
		workspaceID = workspaceIDInt
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "워크스페이스 ID가 필요합니다"})
	}

	// 역할 제거
	err := h.service.RemoveWorkspaceRole(userID, workspaceID, roleID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("역할 제거 실패: %v", err)})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "역할이 성공적으로 제거되었습니다"})
}

// @Summary Create workspace role-CSP role mapping
// @Description Create a new mapping between workspace role and CSP role
// @Tags roles
// @Accept json
// @Produce json
// @Param mapping body model.CreateWorkspaceRoleCspRoleMappingRequest true "Mapping Info"
// @Success 201 {object} model.WorkspaceRoleCspRoleMapping
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/workspace-csp-mapping [post]
func (h *RoleHandler) CreateWorkspaceRoleCspRoleMapping(c echo.Context) error {
	var req model.RoleMasterCspRoleMappingRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("워크스페이스 역할-CSP 역할 매핑 생성 요청 바인딩 실패: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	log.Printf("워크스페이스 역할-CSP 역할 매핑 생성 요청 - 워크스페이스 역할 ID: %s, CSP 역할 ID: %s",
		req.RoleID, req.CspRoleID)

	// 입력값 검증
	if err := c.Validate(&req); err != nil {
		log.Printf("워크스페이스 역할-CSP 역할 매핑 생성 입력값 검증 실패: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("입력값 검증 실패: %v", err)})
	}

	// 문자열 ID를 uint로 변환
	roleID, err := util.StringToUint(req.RoleID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식입니다"})
	}

	cspRoleID, err := util.StringToUint(req.CspRoleID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식입니다"})
	}

	// 매핑 생성
	mapping := model.RoleMasterCspRoleMapping{
		RoleID:    roleID,
		CspRoleID: cspRoleID,
		CspType:   req.CspType,
		CreatedAt: time.Now(),
	}

	createdMapping, err := h.service.CreateWorkspaceRoleCspRoleMapping(mapping)
	if err != nil {
		log.Printf("워크스페이스 역할-CSP 역할 매핑 생성 실패: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("매핑 생성 실패: %v", err)})
	}

	log.Printf("워크스페이스 역할-CSP 역할 매핑 생성 성공 - ID: %d", createdMapping.RoleID)
	return c.JSON(http.StatusCreated, createdMapping)
}

// @Summary Delete workspace role-CSP role mapping
// @Description Delete a mapping between workspace role and CSP role
// @Tags roles
// @Accept json
// @Produce json
// @Param mapping body model.WorkspaceRoleCspRoleMappingRequest true "Mapping Info"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/workspace-csp-mapping [delete]
func (h *RoleHandler) DeleteWorkspaceRoleCspRoleMapping(c echo.Context) error {
	cspType := "workspace"
	var req model.RoleMasterCspRoleMappingRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("워크스페이스 역할-CSP 역할 매핑 삭제 요청 바인딩 실패: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	log.Printf("워크스페이스 역할-CSP 역할 매핑 삭제 요청 - 워크스페이스 역할 ID: %s, CSP 역할 ID: %s",
		req.RoleID, req.CspRoleID)

	// 문자열 ID를 uint로 변환
	roleIDInt, err := util.StringToUint(req.RoleID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식입니다"})
	}

	cspRoleIDInt, err := util.StringToUint(req.CspRoleID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 ID 형식입니다"})
	}

	// 매핑 삭제
	err = h.service.DeleteWorkspaceRoleCspRoleMapping(roleIDInt, cspRoleIDInt, cspType)
	if err != nil {
		log.Printf("워크스페이스 역할-CSP 역할 매핑 삭제 실패: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("매핑 삭제 실패: %v", err)})
	}

	log.Printf("워크스페이스 역할-CSP 역할 매핑 삭제 성공 - ID: %d", roleIDInt)
	return c.NoContent(http.StatusNoContent)
}

// @Summary Get workspace role-CSP role mapping
// @Description Get a mapping between workspace role and CSP role
// @Tags roles
// @Accept json
// @Produce json
// @Param mapping body model.WorkspaceRoleCspRoleMappingRequest true "Mapping Info"
// @Success 200 {object} model.WorkspaceRoleCspRoleMapping
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/roles/workspace-csp-mapping [get]
func (h *RoleHandler) GetWorkspaceRoleCspRoleMappings(c echo.Context) error {
	cspType := "workspace"
	var req model.RoleMasterCspRoleMappingRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("워크스페이스 역할-CSP 역할 매핑 조회 요청 바인딩 실패: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	log.Printf("워크스페이스 역할-CSP 역할 매핑 조회 요청 - 워크스페이스 역할 ID: %s, CSP 역할 ID: %s",
		req.RoleID, req.CspRoleID)

	// 문자열 ID를 uint로 변환
	var roleIDInt, cspRoleIDInt uint
	var err error

	if req.RoleID != "" {
		roleIDInt, err = util.StringToUint(req.RoleID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 역할 ID 형식입니다"})
		}
	}

	if req.CspRoleID != "" {
		cspRoleIDInt, err = util.StringToUint(req.CspRoleID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 CSP 역할 ID 형식입니다"})
		}
	}

	mappings, err := h.service.GetWorkspaceRoleCspRoleMappings(roleIDInt, cspRoleIDInt, cspType)
	if err != nil {
		log.Printf("워크스페이스 역할-CSP 역할 매핑 조회 실패: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("매핑 조회 실패: %v", err)})
	}

	log.Printf("워크스페이스 역할-CSP 역할 매핑 조회 성공 - 조회된 매핑 수: %d", len(mappings))
	return c.JSON(http.StatusOK, mappings)
}
