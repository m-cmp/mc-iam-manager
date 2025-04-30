package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository" // Import repository for error checking
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm"
)

// CspMappingHandler 역할-CSP 역할 매핑 관리 핸들러
type CspMappingHandler struct {
	service *service.CspMappingService
}

// NewCspMappingHandler 새 CspMappingHandler 인스턴스 생성
func NewCspMappingHandler(db *gorm.DB) *CspMappingHandler {
	service := service.NewCspMappingService(db)
	return &CspMappingHandler{service: service}
}

// CreateCspRoleMapping godoc
// @Summary 역할-CSP 역할 매핑 생성
// @Description 워크스페이스 역할과 CSP 역할(ARN)을 매핑합니다.
// @Tags workspace-roles, csp-mappings
// @Accept json
// @Produce json
// @Param roleId path int true "워크스페이스 역할 ID"
// @Param mapping body model.WorkspaceRoleCspRoleMapping true "CSP 역할 매핑 정보 (workspaceRoleId는 경로 파라미터 사용, 요청 본문에서는 제외 가능)"
// @Success 201 {object} model.WorkspaceRoleCspRoleMapping
// @Failure 400 {object} map[string]string "error: 잘못된 요청 형식 또는 ID"
// @Failure 404 {object} map[string]string "error: 워크스페이스 역할을 찾을 수 없음"
// @Failure 409 {object} map[string]string "error: 이미 매핑이 존재함 (PK 충돌)"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspace-roles/{roleId}/csp-role-mappings [post]
func (h *CspMappingHandler) CreateCspRoleMapping(c echo.Context) error {
	roleID, err := strconv.ParseUint(c.Param("roleId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 역할 ID 형식입니다"})
	}

	var mapping model.WorkspaceRoleCspRoleMapping
	if err := c.Bind(&mapping); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다: " + err.Error()})
	}

	// Ensure the workspaceRoleId from path matches the one in body if provided, or set it
	mapping.WorkspaceRoleID = uint(roleID)

	// Basic validation
	if mapping.CspType == "" || mapping.CspRoleArn == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "cspType, cspRoleArn 필드는 필수입니다"})
	}

	if err := h.service.Create(c.Request().Context(), &mapping); err != nil {
		// Handle potential conflict error from DB or service validation
		if errors.Is(err, repository.ErrCspMappingAlreadyExists) { // Assuming repo returns this
			return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
		}
		// Handle role not found error from service validation
		if strings.Contains(err.Error(), "failed to find workspace role") {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("CSP 역할 매핑 생성 실패: %v", err)})
	}
	return c.JSON(http.StatusCreated, mapping)
}

// ListCspRoleMappingsByRole godoc
// @Summary 특정 워크스페이스 역할의 CSP 역할 매핑 목록 조회
// @Description 특정 워크스페이스 역할에 매핑된 모든 CSP 역할 목록을 조회합니다.
// @Tags workspace-roles, csp-mappings
// @Accept json
// @Produce json
// @Param roleId path int true "워크스페이스 역할 ID"
// @Success 200 {array} model.WorkspaceRoleCspRoleMapping
// @Failure 400 {object} map[string]string "error: 잘못된 ID 형식"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspace-roles/{roleId}/csp-role-mappings [get]
func (h *CspMappingHandler) ListCspRoleMappingsByRole(c echo.Context) error {
	roleID, err := strconv.ParseUint(c.Param("roleId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 역할 ID 형식입니다"})
	}

	mappings, err := h.service.ListByWorkspaceRole(c.Request().Context(), uint(roleID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("CSP 역할 매핑 목록 조회 실패: %v", err)})
	}
	return c.JSON(http.StatusOK, mappings)
}

// UpdateCspRoleMapping godoc
// @Summary 역할-CSP 역할 매핑 수정
// @Description 기존 역할-CSP 역할 매핑 정보를 수정합니다 (Description, IdpIdentifier만 수정 가능).
// @Tags workspace-roles, csp-mappings
// @Accept json
// @Produce json
// @Param roleId path int true "워크스페이스 역할 ID"
// @Param cspType path string true "CSP 타입 (e.g., aws)"
// @Param cspRoleArn path string true "CSP 역할 ARN (URL Encoded)"
// @Param updates body object true "수정할 필드와 값 (예: {\"description\": \"New Desc\", \"idpIdentifier\": \"new_idp_arn\"})"
// @Success 200 {object} model.WorkspaceRoleCspRoleMapping "업데이트된 매핑 정보"
// @Failure 400 {object} map[string]string "error: 잘못된 요청 형식 또는 ID"
// @Failure 404 {object} map[string]string "error: 매핑을 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspace-roles/{roleId}/csp-role-mappings/{cspType}/{cspRoleArn} [put]
func (h *CspMappingHandler) UpdateCspRoleMapping(c echo.Context) error {
	roleID, err := strconv.ParseUint(c.Param("roleId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 역할 ID 형식입니다"})
	}
	cspType := c.Param("cspType")
	cspRoleArn := c.Param("cspRoleArn") // Note: ARN might contain slashes, ensure routing handles this or use query param

	if cspType == "" || cspRoleArn == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "cspType과 cspRoleArn은 필수입니다"})
	}

	updates := make(map[string]interface{})
	if err := c.Bind(&updates); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다: " + err.Error()})
	}

	// Allow updating only specific fields
	allowedUpdates := make(map[string]interface{})
	if description, ok := updates["description"].(string); ok {
		allowedUpdates["description"] = description
	}
	if idpIdentifier, ok := updates["idpIdentifier"].(string); ok {
		allowedUpdates["idpIdentifier"] = idpIdentifier
	}

	if len(allowedUpdates) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "업데이트할 필드(description, idpIdentifier)가 없습니다"})
	}

	if err := h.service.Update(c.Request().Context(), uint(roleID), cspType, cspRoleArn, allowedUpdates); err != nil {
		if errors.Is(err, repository.ErrCspMappingNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("CSP 역할 매핑 업데이트 실패: %v", err)})
	}

	updatedMapping, err := h.service.Get(c.Request().Context(), uint(roleID), cspType, cspRoleArn)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("업데이트된 CSP 역할 매핑 조회 실패: %v", err)})
	}
	return c.JSON(http.StatusOK, updatedMapping)
}

// DeleteCspRoleMapping godoc
// @Summary 역할-CSP 역할 매핑 삭제
// @Description 역할-CSP 역할 매핑을 삭제합니다.
// @Tags workspace-roles, csp-mappings
// @Accept json
// @Produce json
// @Param roleId path int true "워크스페이스 역할 ID"
// @Param cspType path string true "CSP 타입 (e.g., aws)"
// @Param cspRoleArn path string true "CSP 역할 ARN (URL Encoded)"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "error: 잘못된 ID 형식"
// @Failure 404 {object} map[string]string "error: 매핑을 찾을 수 없습니다"
// @Failure 500 {object} map[string]string "error: 서버 내부 오류"
// @Security BearerAuth
// @Router /workspace-roles/{roleId}/csp-role-mappings/{cspType}/{cspRoleArn} [delete]
func (h *CspMappingHandler) DeleteCspRoleMapping(c echo.Context) error {
	roleID, err := strconv.ParseUint(c.Param("roleId"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 워크스페이스 역할 ID 형식입니다"})
	}
	cspType := c.Param("cspType")
	cspRoleArn := c.Param("cspRoleArn") // Note: ARN might contain slashes

	if cspType == "" || cspRoleArn == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "cspType과 cspRoleArn은 필수입니다"})
	}

	if err := h.service.Delete(c.Request().Context(), uint(roleID), cspType, cspRoleArn); err != nil {
		if errors.Is(err, repository.ErrCspMappingNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("CSP 역할 매핑 삭제 실패: %v", err)})
	}
	return c.NoContent(http.StatusNoContent)
}
