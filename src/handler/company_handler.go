package handler

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm"
)

// CompanyHandler 회사 정보 관리 핸들러 (싱글톤 — URL에 ID 없음)
type CompanyHandler struct {
	companyService *service.CompanyService
}

// NewCompanyHandler 새 CompanyHandler 인스턴스 생성
func NewCompanyHandler(db *gorm.DB) *CompanyHandler {
	return &CompanyHandler{
		companyService: service.NewCompanyService(db),
	}
}

// CreateCompany godoc
// @Summary Create company
// @Description 플랫폼 회사 정보를 생성합니다. (싱글톤, platformAdmin 전용)
// @Tags company
// @Accept json
// @Produce json
// @Param request body model.CompanyRequest true "Company Info"
// @Success 201 {object} model.CompanyResponse
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/company [post]
// @Id createCompany
func (h *CompanyHandler) CreateCompany(c echo.Context) error {
	var req model.CompanyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}
	if req.RealmName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "realm_name is required"})
	}
	if req.KcClientID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "kc_client_id is required"})
	}
	if req.KcClientSecret == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "kc_client_secret is required"})
	}

	resp, err := h.companyService.CreateCompany(&req)
	if err != nil {
		if strings.HasPrefix(err.Error(), "CONFLICT:") {
			return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
		}
		if strings.HasPrefix(err.Error(), "REALM_ERROR:") {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, resp)
}

// GetCompany godoc
// @Summary Get company
// @Description 플랫폼 회사 정보를 조회합니다. (싱글톤)
// @Tags company
// @Produce json
// @Success 200 {object} model.CompanyResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/company [get]
// @Id getCompany
func (h *CompanyHandler) GetCompany(c echo.Context) error {
	resp, err := h.companyService.GetCompany()
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Company not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, resp)
}

// UpdateCompany godoc
// @Summary Update company
// @Description 플랫폼 회사 이름/설명을 수정합니다. (platformAdmin 전용)
// @Tags company
// @Accept json
// @Produce json
// @Param request body model.CompanyUpdateRequest true "Company Update Info"
// @Success 200 {object} model.CompanyResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/company [put]
// @Id updateCompany
func (h *CompanyHandler) UpdateCompany(c echo.Context) error {
	var req model.CompanyUpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}

	resp, err := h.companyService.UpdateCompany(&req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Company not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, resp)
}

// DeactivateCompany godoc
// @Summary Deactivate company
// @Description 플랫폼 회사를 비활성화합니다. (platformAdmin 전용, 멱등 처리)
// @Tags company
// @Produce json
// @Success 200 {object} model.CompanyResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/company [delete]
// @Id deactivateCompany
func (h *CompanyHandler) DeactivateCompany(c echo.Context) error {
	resp, err := h.companyService.DeactivateCompany()
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Company not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, resp)
}

// ActivateCompany godoc
// @Summary Activate company
// @Description 플랫폼 회사를 활성화합니다. (platformAdmin 전용, 멱등 처리)
// @Tags company
// @Produce json
// @Success 200 {object} model.CompanyResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/company/activate [post]
// @Id activateCompany
func (h *CompanyHandler) ActivateCompany(c echo.Context) error {
	resp, err := h.companyService.ActivateCompany()
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Company not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, resp)
}
