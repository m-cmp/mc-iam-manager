package handler

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/repository" // Import repository for error checking
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm"
)

// ResourceTypeHandler 리소스 유형 관리 핸들러
type ResourceTypeHandler struct {
	service *service.ResourceTypeService
}

// NewResourceTypeHandler 새 ResourceTypeHandler 인스턴스 생성
func NewResourceTypeHandler(db *gorm.DB) *ResourceTypeHandler {
	service := service.NewResourceTypeService(db)
	return &ResourceTypeHandler{service: service}
}

// CreateResourceType godoc
// @Summary 새 리소스 타입 생성
// @Description 새로운 리소스 타입을 생성합니다
// @Tags resource-types
// @Accept json
// @Produce json
// @Param resourceType body model.ResourceType true "Resource Type Info"
// @Success 201 {object} model.ResourceType
// @Failure 400 {object} map[string]string "error: Invalid request"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Security BearerAuth
// @Router /api/v1/resource-types [post]
func (h *ResourceTypeHandler) CreateResourceType(c echo.Context) error {
	var rt model.ResourceType
	if err := c.Bind(&rt); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다: " + err.Error()})
	}

	// Basic validation
	if rt.FrameworkID == "" || rt.ID == "" || rt.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "frameworkId, id, name 필드는 필수입니다"})
	}

	if err := h.service.Create(&rt); err != nil {
		if err == repository.ErrResourceTypeAlreadyExists {
			return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("리소스 유형 생성 실패: %v", err)})
	}
	return c.JSON(http.StatusCreated, rt)
}

// ListResourceTypes godoc
// @Summary 리소스 타입 목록 조회
// @Description 모든 리소스 타입 목록을 조회합니다
// @Tags resource-types
// @Accept json
// @Produce json
// @Success 200 {array} model.ResourceType
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Security BearerAuth
// @Router /api/v1/resource-types [get]
func (h *ResourceTypeHandler) ListResourceTypes(c echo.Context) error {
	frameworkID := c.QueryParam("frameworkId")
	resourceTypes, err := h.service.List(frameworkID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("리소스 유형 목록 조회 실패: %v", err)})
	}
	return c.JSON(http.StatusOK, resourceTypes)
}

// GetResourceTypeByID godoc
// @Summary 리소스 타입 ID로 조회
// @Description 특정 리소스 타입을 ID로 조회합니다
// @Tags resource-types
// @Accept json
// @Produce json
// @Param id path string true "Resource Type ID"
// @Success 200 {object} model.ResourceType
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 404 {object} map[string]string "error: Resource Type not found"
// @Security BearerAuth
// @Router /api/v1/resource-types/{id} [get]
func (h *ResourceTypeHandler) GetResourceTypeByID(c echo.Context) error {
	frameworkID := c.Param("frameworkId")
	id := c.Param("id")
	if frameworkID == "" || id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "프레임워크 ID와 리소스 유형 ID는 필수입니다"})
	}

	resourceType, err := h.service.GetByID(frameworkID, id)
	if err != nil {
		if err == repository.ErrResourceTypeNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("리소스 유형 조회 실패: %v", err)})
	}
	return c.JSON(http.StatusOK, resourceType)
}

// UpdateResourceType godoc
// @Summary 리소스 타입 업데이트
// @Description 리소스 타입 정보를 업데이트합니다
// @Tags resource-types
// @Accept json
// @Produce json
// @Param id path string true "Resource Type ID"
// @Param resourceType body model.ResourceType true "Resource Type Info"
// @Success 200 {object} model.ResourceType
// @Failure 400 {object} map[string]string "error: Invalid request"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 404 {object} map[string]string "error: Resource Type not found"
// @Security BearerAuth
// @Router /api/v1/resource-types/{id} [put]
func (h *ResourceTypeHandler) UpdateResourceType(c echo.Context) error {
	frameworkID := c.Param("frameworkId")
	resourceTypeId := c.Param("resourceTypeId")
	if frameworkID == "" || resourceTypeId == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "프레임워크 ID와 리소스 유형 ID는 필수입니다"})
	}

	updates := make(map[string]interface{})
	if err := c.Bind(&updates); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다: " + err.Error()})
	}

	// Allow updating only specific fields
	allowedUpdates := make(map[string]interface{})
	if name, ok := updates["name"].(string); ok {
		allowedUpdates["name"] = name
	}
	if description, ok := updates["description"].(string); ok {
		allowedUpdates["description"] = description
	}

	if len(allowedUpdates) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "업데이트할 필드(name, description)가 없습니다"})
	}

	if err := h.service.Update(frameworkID, resourceTypeId, allowedUpdates); err != nil {
		if err == repository.ErrResourceTypeNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("리소스 유형 업데이트 실패: %v", err)})
	}

	updatedResourceType, err := h.service.GetByID(frameworkID, resourceTypeId)
	if err != nil {
		// This shouldn't happen ideally after a successful update, but handle it
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("업데이트된 리소스 유형 조회 실패: %v", err)})
	}
	return c.JSON(http.StatusOK, updatedResourceType)
}

// DeleteResourceType godoc
// @Summary 리소스 타입 삭제
// @Description 리소스 타입을 삭제합니다
// @Tags resource-types
// @Accept json
// @Produce json
// @Param id path string true "Resource Type ID"
// @Success 204 "No Content"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 403 {object} map[string]string "error: Forbidden"
// @Failure 404 {object} map[string]string "error: Resource Type not found"
// @Security BearerAuth
// @Router /api/v1/resource-types/{id} [delete]
func (h *ResourceTypeHandler) DeleteResourceType(c echo.Context) error {
	frameworkID := c.Param("frameworkId")
	id := c.Param("id")
	if frameworkID == "" || id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "프레임워크 ID와 리소스 유형 ID는 필수입니다"})
	}

	if err := h.service.Delete(frameworkID, id); err != nil {
		if err == repository.ErrResourceTypeNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("리소스 유형 삭제 실패: %v", err)})
	}
	return c.NoContent(http.StatusNoContent)
}
