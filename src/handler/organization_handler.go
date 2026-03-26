package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository"
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm"
)

// OrganizationHandler 조직 관리 핸들러
type OrganizationHandler struct {
	orgService *service.OrganizationService
}

// NewOrganizationHandler OrganizationHandler 생성자
func NewOrganizationHandler(db *gorm.DB) *OrganizationHandler {
	return &OrganizationHandler{
		orgService: service.NewOrganizationService(db),
	}
}

// SetupInitialOrganizations godoc
// @Summary 기본 조직 초기화
// @Description YAML 시드 파일에서 기본 조직 구조(MZC + 8개 프레임워크)를 로드하여 등록합니다. 멱등성 보장.
// @Tags organizations
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/setup/initial-organizations [post]
// @Id setupInitialOrganizations
func (h *OrganizationHandler) SetupInitialOrganizations(c echo.Context) error {
	if err := h.orgService.LoadAndRegisterOrganizationsFromYAML(""); err != nil {
		log.Printf("[ERROR] SetupInitialOrganizations failed: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "기본 조직이 등록되었습니다."})
}

// CreateOrganization godoc
// @Summary 조직 생성
// @Description 플랫폼 관리자가 조직을 생성합니다. parent_id가 없으면 최상위 조직 생성.
// @Tags organizations
// @Accept json
// @Produce json
// @Param body body model.CreateOrganizationRequest true "조직 생성 요청"
// @Success 201 {object} model.Organization
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Security BearerAuth
// @Router /api/organizations [post]
// @Id createOrganization
func (h *OrganizationHandler) CreateOrganization(c echo.Context) error {
	var req model.CreateOrganizationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	org, err := h.orgService.CreateOrganization(&req)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrOrganizationNameDuplicate):
			return c.JSON(http.StatusConflict, map[string]string{"error": "조직 이름이 이미 존재합니다 (동일 부모 하)"})
		case errors.Is(err, repository.ErrOrganizationCodeDuplicate):
			return c.JSON(http.StatusConflict, map[string]string{"error": "조직 코드가 이미 존재합니다"})
		case errors.Is(err, repository.ErrMaxOrganizationsPerLevel):
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "동일 레벨에 최대 99개 조직만 생성 가능합니다"})
		case errors.Is(err, repository.ErrOrganizationNotFound):
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "부모 조직을 찾을 수 없습니다"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return c.JSON(http.StatusCreated, org)
}

// GetOrganizations godoc
// @Summary 조직 목록 조회
// @Description 전체 조직 목록을 조회합니다. tree=true이면 Tree 구조로 반환. name/code로 검색 가능 (검색 시 tree 파라미터 무시).
// @Tags organizations
// @Produce json
// @Param tree query bool false "Tree 구조 반환 여부 (기본: false)"
// @Param name query string false "조직명 검색 (부분 일치, ILIKE)"
// @Param code query string false "조직 코드 검색 (부분 일치, ILIKE)"
// @Success 200 {array} model.OrganizationTree
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/organizations [get]
// @Id listOrganizations
func (h *OrganizationHandler) GetOrganizations(c echo.Context) error {
	nameParam := c.QueryParam("name")
	codeParam := c.QueryParam("code")

	// 검색 파라미터가 있으면 검색 모드 (tree 무시)
	if nameParam != "" || codeParam != "" {
		result, err := h.orgService.SearchOrganizations(nameParam, codeParam)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, result)
	}

	treeParam := c.QueryParam("tree")
	tree := treeParam == "true"

	result, err := h.orgService.GetOrganizations(tree)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, result)
}

// GetOrganizationByID godoc
// @Summary 조직 상세 조회 (ID)
// @Description 조직 ID로 조직 정보를 조회합니다.
// @Tags organizations
// @Produce json
// @Param organizationId path int true "조직 ID"
// @Success 200 {object} model.Organization
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /api/organizations/id/{organizationId} [get]
// @Id getOrganizationByID
func (h *OrganizationHandler) GetOrganizationByID(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("organizationId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid organization ID"})
	}

	org, err := h.orgService.GetOrganizationByID(uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrOrganizationNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "조직을 찾을 수 없습니다"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, org)
}

// GetOrganizationByCode godoc
// @Summary 조직 상세 조회 (코드)
// @Description 조직 코드로 조직 정보를 조회합니다.
// @Tags organizations
// @Produce json
// @Param code path string true "조직 코드 (예: 0101)"
// @Success 200 {object} model.Organization
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /api/organizations/code/{code} [get]
// @Id getOrganizationByCode
func (h *OrganizationHandler) GetOrganizationByCode(c echo.Context) error {
	code := c.Param("code")
	if code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Organization code is required"})
	}

	org, err := h.orgService.GetOrganizationByCode(code)
	if err != nil {
		if errors.Is(err, repository.ErrOrganizationNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "조직을 찾을 수 없습니다"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, org)
}

// UpdateOrganization godoc
// @Summary 조직 수정
// @Description 조직 정보를 수정합니다. 부모 변경 시 하위 조직 코드 자동 재생성.
// @Tags organizations
// @Accept json
// @Produce json
// @Param organizationId path int true "조직 ID"
// @Param body body model.UpdateOrganizationRequest true "조직 수정 요청"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Security BearerAuth
// @Router /api/organizations/id/{organizationId} [put]
// @Id updateOrganization
func (h *OrganizationHandler) UpdateOrganization(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("organizationId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid organization ID"})
	}

	var req model.UpdateOrganizationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	if err := h.orgService.UpdateOrganization(uint(id), &req); err != nil {
		switch {
		case errors.Is(err, repository.ErrOrganizationNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{"error": "조직을 찾을 수 없습니다"})
		case errors.Is(err, repository.ErrOrganizationNameDuplicate):
			return c.JSON(http.StatusConflict, map[string]string{"error": "조직 이름이 이미 존재합니다 (동일 부모 하)"})
		case errors.Is(err, repository.ErrOrganizationCodeDuplicate):
			return c.JSON(http.StatusConflict, map[string]string{"error": "조직 코드가 이미 존재합니다"})
		case errors.Is(err, repository.ErrCircularReference):
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "순환 참조가 발생합니다"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "조직 정보가 수정되었습니다."})
}

// DeleteOrganization godoc
// @Summary 조직 삭제
// @Description 조직을 삭제합니다. 하위 조직 또는 소속 사용자가 있으면 삭제 불가.
// @Tags organizations
// @Produce json
// @Param organizationId path int true "조직 ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /api/organizations/id/{organizationId} [delete]
// @Id deleteOrganization
func (h *OrganizationHandler) DeleteOrganization(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("organizationId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid organization ID"})
	}

	if err := h.orgService.DeleteOrganization(uint(id)); err != nil {
		switch {
		case errors.Is(err, repository.ErrOrganizationNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{"error": "조직을 찾을 수 없습니다"})
		case errors.Is(err, repository.ErrOrganizationHasChildren):
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "하위 조직이 존재합니다. 먼저 하위 조직을 삭제해주세요."})
		case errors.Is(err, repository.ErrOrganizationHasUsers):
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "소속 사용자가 있습니다. 먼저 사용자를 조직에서 제거해주세요."})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "조직이 삭제되었습니다."})
}

// AssignUserOrganizations godoc
// @Summary 사용자-조직 할당
// @Description 사용자를 하나 이상의 조직에 할당합니다 (다중 소속 가능).
// @Tags organizations
// @Accept json
// @Produce json
// @Param userId path int true "사용자 ID"
// @Param body body model.AssignUserOrganizationsRequest true "조직 할당 요청"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/users/{userId}/organizations [post]
// @Id assignUserOrganizations
func (h *OrganizationHandler) AssignUserOrganizations(c echo.Context) error {
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
	}

	var req model.AssignUserOrganizationsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := h.orgService.AssignUserToOrganizations(uint(userID), req.OrganizationIDs); err != nil {
		if errors.Is(err, repository.ErrOrganizationNotFound) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "사용자가 조직에 할당되었습니다."})
}

// GetUserOrganizations godoc
// @Summary 사용자 소속 조직 조회
// @Description 사용자가 소속된 조직 목록을 조회합니다. path, level 계층 정보가 포함됩니다.
// @Tags organizations
// @Produce json
// @Param userId path int true "사용자 ID"
// @Success 200 {array} model.OrganizationTree
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/users/{userId}/organizations [get]
// @Id getUserOrganizations
func (h *OrganizationHandler) GetUserOrganizations(c echo.Context) error {
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
	}

	orgs, err := h.orgService.GetUserOrganizations(uint(userID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, orgs)
}

// ReplaceUserGroups godoc
// @Summary 사용자 그룹 전체 교체
// @Description 사용자가 소속된 그룹을 전체 교체합니다. 기존 그룹을 모두 제거하고 새로운 그룹을 할당합니다.
// @Tags groups
// @Accept json
// @Produce json
// @Param userId path int true "사용자 ID"
// @Param body body model.ReplaceUserGroupsRequest true "그룹 교체 요청"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /api/users/{userId}/groups [put]
// @Id replaceUserGroups
func (h *OrganizationHandler) ReplaceUserGroups(c echo.Context) error {
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
	}

	var req model.ReplaceUserGroupsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	if err := h.orgService.ReplaceUserOrganizations(uint(userID), req.GroupIDs); err != nil {
		if errors.Is(err, repository.ErrOrganizationNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "사용자 그룹이 교체되었습니다."})
}

// RemoveUserOrganization godoc
// @Summary 사용자-조직 매핑 제거
// @Description 사용자를 특정 조직에서 제거합니다.
// @Tags organizations
// @Produce json
// @Param userId path int true "사용자 ID"
// @Param organizationId path int true "조직 ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/users/{userId}/organizations/{organizationId} [delete]
// @Id removeUserOrganization
func (h *OrganizationHandler) RemoveUserOrganization(c echo.Context) error {
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
	}
	orgID, err := strconv.ParseUint(c.Param("organizationId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid organization ID"})
	}

	if err := h.orgService.RemoveUserFromOrganization(uint(userID), uint(orgID)); err != nil {
		if errors.Is(err, repository.ErrUserOrganizationNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "사용자가 해당 조직에 소속되어 있지 않습니다"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "사용자가 조직에서 제거되었습니다."})
}

// GetOrganizationUsers godoc
// @Summary 조직 소속 사용자 조회
// @Description 특정 조직에 소속된 사용자 목록을 조회합니다.
// @Tags organizations
// @Produce json
// @Param organizationId path int true "조직 ID"
// @Success 200 {array} model.User
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /api/organizations/id/{organizationId}/users [get]
// @Id getOrganizationUsers
func (h *OrganizationHandler) GetOrganizationUsers(c echo.Context) error {
	orgID, err := strconv.ParseUint(c.Param("organizationId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid organization ID"})
	}

	users, err := h.orgService.GetOrganizationUsers(uint(orgID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, users)
}
