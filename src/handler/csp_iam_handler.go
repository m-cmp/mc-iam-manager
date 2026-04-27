package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/csp"
	awscsp "github.com/m-cmp/mc-iam-manager/csp/aws"
	"github.com/m-cmp/mc-iam-manager/repository"
	"gorm.io/gorm"
)

// CspIAMHandler CSP IAM 직접 관리 핸들러
// CSP IAM Role CRUD를 CSP API를 직접 호출하여 처리합니다.
type CspIAMHandler struct {
	cspAccountRepo   *repository.CspAccountRepository
	cspIdpConfigRepo *repository.CspIdpConfigRepository
	db               *gorm.DB
}

// NewCspIAMHandler 새 CspIAMHandler 인스턴스 생성
func NewCspIAMHandler(db *gorm.DB) *CspIAMHandler {
	return &CspIAMHandler{
		cspAccountRepo:   repository.NewCspAccountRepository(db),
		cspIdpConfigRepo: repository.NewCspIdpConfigRepository(db),
		db:               db,
	}
}

// getIAMClient CSP 계정 ID를 기반으로 IAM 클라이언트를 생성합니다.
// OIDC 인증 방식을 기준으로 동작하며, IDP 설정의 role_arn을 사용합니다.
func (h *CspIAMHandler) getIAMClient(c echo.Context, cspAccountID uint) (csp.IAMClient, error) {
	// 1. CSP 계정 조회
	account, err := h.cspAccountRepo.GetByID(cspAccountID)
	if err != nil {
		return nil, err
	}

	// 2. 활성 IDP 설정 조회 (OIDC 우선)
	idpConfigs, err := h.cspIdpConfigRepo.GetActiveByAccountID(cspAccountID)
	if err != nil {
		return nil, err
	}
	if len(idpConfigs) == 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "no active IDP config found for CSP account")
	}

	// OIDC 설정 우선 선택
	idpConfig := idpConfigs[0]
	for _, cfg := range idpConfigs {
		if cfg.IsOIDC() {
			idpConfig = cfg
			break
		}
	}

	roleARN := idpConfig.Config["role_arn"]
	if roleARN == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "role_arn not configured in IDP config")
	}

	// 3. 요청 컨텍스트에서 JWT 액세스 토큰 추출
	accessToken, ok := c.Get("access_token").(string)
	if !ok || accessToken == "" {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "access token not found")
	}

	cfg := &csp.IAMClientConfig{
		Region:           account.GetRegion(),
		RoleARN:          roleARN,
		WebIdentityToken: accessToken,
	}

	switch account.CspType {
	case "aws":
		return awscsp.NewAWSIAMClient(cfg, h.db)
	default:
		return nil, echo.NewHTTPError(http.StatusBadRequest, "unsupported CSP type: "+account.CspType)
	}
}

// parseCspAccountID 요청에서 csp_account_id를 파싱합니다.
func parseCspAccountID(c echo.Context) (uint, error) {
	idStr := c.QueryParam("csp_account_id")
	if idStr == "" {
		return 0, echo.NewHTTPError(http.StatusBadRequest, "csp_account_id query parameter is required")
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return 0, echo.NewHTTPError(http.StatusBadRequest, "invalid csp_account_id")
	}
	return uint(id), nil
}

// CreateIAMRole godoc
// @Summary CSP IAM 역할 생성
// @Description CSP에 IAM 역할을 직접 생성합니다
// @Tags csp-iam
// @Accept json
// @Produce json
// @Param csp_account_id query int true "CSP 계정 ID"
// @Param role body csp.Role true "IAM 역할 정보"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/csp/iam/roles [post]
// @Id createCspIAMRole
func (h *CspIAMHandler) CreateIAMRole(c echo.Context) error {
	cspAccountID, err := parseCspAccountID(c)
	if err != nil {
		return err
	}

	var role csp.Role
	if err := c.Bind(&role); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	if role.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "role name is required"})
	}

	client, err := h.getIAMClient(c, cspAccountID)
	if err != nil {
		return handleIAMError(c, err)
	}

	if err := client.CreateRole(c.Request().Context(), &role); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]string{"message": "IAM role created successfully", "name": role.Name})
}

// GetIAMRole godoc
// @Summary CSP IAM 역할 조회
// @Description CSP에서 특정 IAM 역할 정보를 조회합니다
// @Tags csp-iam
// @Accept json
// @Produce json
// @Param roleName path string true "IAM 역할명"
// @Param csp_account_id query int true "CSP 계정 ID"
// @Success 200 {object} csp.Role
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/csp/iam/roles/{roleName} [get]
// @Id getCspIAMRole
func (h *CspIAMHandler) GetIAMRole(c echo.Context) error {
	roleName := c.Param("roleName")
	if roleName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "roleName is required"})
	}

	cspAccountID, err := parseCspAccountID(c)
	if err != nil {
		return err
	}

	client, err := h.getIAMClient(c, cspAccountID)
	if err != nil {
		return handleIAMError(c, err)
	}

	role, err := client.GetRole(c.Request().Context(), roleName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, role)
}

// UpdateIAMRole godoc
// @Summary CSP IAM 역할 수정
// @Description CSP의 IAM 역할 정보를 수정합니다
// @Tags csp-iam
// @Accept json
// @Produce json
// @Param roleName path string true "IAM 역할명"
// @Param csp_account_id query int true "CSP 계정 ID"
// @Param role body csp.Role true "IAM 역할 수정 정보"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/csp/iam/roles/{roleName} [put]
// @Id updateCspIAMRole
func (h *CspIAMHandler) UpdateIAMRole(c echo.Context) error {
	roleName := c.Param("roleName")
	if roleName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "roleName is required"})
	}

	cspAccountID, err := parseCspAccountID(c)
	if err != nil {
		return err
	}

	var role csp.Role
	if err := c.Bind(&role); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	role.Name = roleName

	client, err := h.getIAMClient(c, cspAccountID)
	if err != nil {
		return handleIAMError(c, err)
	}

	if err := client.UpdateRole(c.Request().Context(), &role); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "IAM role updated successfully", "name": roleName})
}

// DeleteIAMRole godoc
// @Summary CSP IAM 역할 삭제
// @Description CSP에서 IAM 역할을 삭제합니다
// @Tags csp-iam
// @Accept json
// @Produce json
// @Param roleName path string true "IAM 역할명"
// @Param csp_account_id query int true "CSP 계정 ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/csp/iam/roles/{roleName} [delete]
// @Id deleteCspIAMRole
func (h *CspIAMHandler) DeleteIAMRole(c echo.Context) error {
	roleName := c.Param("roleName")
	if roleName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "roleName is required"})
	}

	cspAccountID, err := parseCspAccountID(c)
	if err != nil {
		return err
	}

	client, err := h.getIAMClient(c, cspAccountID)
	if err != nil {
		return handleIAMError(c, err)
	}

	if err := client.DeleteRole(c.Request().Context(), roleName); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

// handleIAMError IAM 클라이언트 생성 오류를 HTTP 오류로 변환합니다.
func handleIAMError(c echo.Context, err error) error {
	if httpErr, ok := err.(*echo.HTTPError); ok {
		return c.JSON(httpErr.Code, map[string]string{"error": httpErr.Message.(string)})
	}
	return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
}
