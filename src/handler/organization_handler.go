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
// @Description 조직을 삭제합니다. cascade=true이면 하위 조직 및 사용자 매핑도 모두 삭제됩니다. 기본(cascade=false)이고 하위 조직이 있으면 400을 반환합니다.
// @Tags organizations
// @Produce json
// @Param organizationId path int true "조직 ID"
// @Param cascade query bool false "하위 조직 cascade 삭제 여부 (기본: false)"
// @Success 204 "삭제 성공"
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

	cascade := c.QueryParam("cascade") == "true"

	if cascade {
		if err := h.orgService.DeleteOrganizationCascade(uint(id)); err != nil {
			if errors.Is(err, repository.ErrOrganizationNotFound) {
				return c.JSON(http.StatusNotFound, map[string]string{"error": "조직을 찾을 수 없습니다"})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.NoContent(http.StatusNoContent)
	}

	if err := h.orgService.DeleteOrganization(uint(id)); err != nil {
		switch {
		case errors.Is(err, repository.ErrOrganizationNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{"error": "조직을 찾을 수 없습니다"})
		case errors.Is(err, repository.ErrOrganizationHasChildren):
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "하위 조직이 존재합니다. cascade=true 옵션을 사용하거나 하위 조직을 먼저 삭제해주세요."})
		case errors.Is(err, repository.ErrOrganizationHasUsers):
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "소속 사용자가 있습니다. cascade=true 옵션을 사용하거나 사용자를 먼저 제거해주세요."})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// GetOrganizationTree godoc
// @Summary 전체 조직 트리 조회
// @Description 전체 조직을 계층 트리 구조로 반환합니다. 최대 10단계 깊이를 지원하며 각 노드에 children 배열을 포함합니다.
// @Tags organizations
// @Produce json
// @Success 200 {array} model.OrganizationTree
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/organizations/tree [get]
// @Id getOrganizationTree
func (h *OrganizationHandler) GetOrganizationTree(c echo.Context) error {
	tree, err := h.orgService.GetOrganizationTree()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, tree)
}

// GetOrganizationSubtree godoc
// @Summary 특정 조직의 하위 트리 조회
// @Description 지정 조직을 루트로 하는 하위 트리를 반환합니다.
// @Tags organizations
// @Produce json
// @Param organizationId path int true "조직 ID"
// @Success 200 {array} model.OrganizationTree
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/organizations/id/{organizationId}/subtree [get]
// @Id getOrganizationSubtree
func (h *OrganizationHandler) GetOrganizationSubtree(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("organizationId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid organization ID"})
	}

	tree, err := h.orgService.GetOrganizationSubtree(uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrOrganizationNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "조직을 찾을 수 없습니다"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, tree)
}

// MoveOrganization godoc
// @Summary 조직 이동
// @Description 조직을 트리 내 다른 위치(다른 부모 조직)로 이동합니다. 하위 조직도 함께 이동하며 조직 코드가 자동 재생성됩니다.
// @Tags organizations
// @Accept json
// @Produce json
// @Param organizationId path int true "조직 ID"
// @Param body body model.MoveOrganizationRequest true "이동 요청 (new_parent_id: null이면 최상위로 이동)"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /api/organizations/id/{organizationId}/move [put]
// @Id moveOrganization
func (h *OrganizationHandler) MoveOrganization(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("organizationId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid organization ID"})
	}

	var req model.MoveOrganizationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	if err := h.orgService.MoveOrganization(uint(id), &req); err != nil {
		switch {
		case errors.Is(err, repository.ErrOrganizationNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{"error": "조직을 찾을 수 없습니다"})
		case errors.Is(err, repository.ErrCircularReference):
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "자기 자신 또는 하위 조직으로의 이동은 불가합니다"})
		case errors.Is(err, service.ErrMaxDepthExceeded):
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "이동 후 최대 깊이(10단계)를 초과합니다"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "조직이 이동되었습니다."})
}

// GetOrganizationDeletable godoc
// @Summary 조직 삭제 가능 여부 확인
// @Description 특정 조직의 삭제 가능 여부와 사유를 반환합니다.
// @Tags organizations
// @Produce json
// @Param organizationId path int true "조직 ID"
// @Success 200 {object} model.OrganizationDeletableResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/organizations/id/{organizationId}/deletable [get]
// @Id getOrganizationDeletable
func (h *OrganizationHandler) GetOrganizationDeletable(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("organizationId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid organization ID"})
	}

	resp, err := h.orgService.CheckOrganizationDeletable(uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrOrganizationNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "조직을 찾을 수 없습니다"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, resp)
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
// @Description 사용자가 소속된 조직 목록을 조회합니다. hierarchy=true이면 path/level 포함.
// @Tags organizations
// @Produce json
// @Param userId path int true "사용자 ID"
// @Param hierarchy query bool false "계층 정보(path, level) 포함 여부 (기본: false)"
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

	if c.QueryParam("hierarchy") == "true" {
		orgs, err := h.orgService.GetUserOrganizationsWithHierarchy(uint(userID))
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, orgs)
	}

	orgs, err := h.orgService.GetUserOrganizations(uint(userID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, orgs)
}

// ReplaceUserGroups godoc
// @Summary 사용자 그룹 멤버십 전체 교체
// @Description 사용자의 기존 그룹을 모두 제거하고 지정된 그룹으로 교체합니다.
// @Tags organizations
// @Accept json
// @Produce json
// @Param userId path int true "사용자 ID"
// @Param body body model.AssignUserGroupsRequest true "교체할 그룹 목록"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/users/id/{userId}/groups [put]
// @Id replaceUserGroups
func (h *OrganizationHandler) ReplaceUserGroups(c echo.Context) error {
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

	if err := h.orgService.ReplaceUserGroups(uint(userID), req.GroupIDs); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "사용자 그룹 멤버십이 교체되었습니다."})
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
