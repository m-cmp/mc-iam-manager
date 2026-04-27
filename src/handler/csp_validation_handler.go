package handler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm"
)

// CspValidationHandler CSP 인증 설정 검증 핸들러
type CspValidationHandler struct {
	validationService *service.CspValidationService
	userService       *service.UserService
}

// NewCspValidationHandler 새 CspValidationHandler 인스턴스 생성
func NewCspValidationHandler(db *gorm.DB) *CspValidationHandler {
	return &CspValidationHandler{
		validationService: service.NewCspValidationService(db),
		userService:       service.NewUserService(db),
	}
}

// ValidateCredentials godoc
// @Summary CSP 인증 설정 단계별 검증
// @Description 워크스페이스 사용자의 CSP×AuthMethod 조합 인증 설정을 단계별로 검증하고 임시자격증명 발급까지 확인한다. 실패 여부와 무관하게 모든 단계를 응답에 포함한다.
// @Tags csp-validation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.CspValidationRequest true "검증 요청"
// @Success 200 {object} model.CspValidationResponse
// @Failure 400 {object} map[string]string "error: invalid request"
// @Failure 401 {object} map[string]string "error: Unauthorized"
// @Failure 404 {object} map[string]string "error: User not found"
// @Router /api/workspaces/credentials/validate [post]
// @Id mciamValidateCredentials
func (h *CspValidationHandler) ValidateCredentials(c echo.Context) error {
	kcUserId, ok := c.Get("kcUserId").(string)
	if !ok || kcUserId == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "kcUserId not found in context"})
	}

	var req model.CspValidationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
	}
	if req.WorkspaceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "workspaceId is required"})
	}
	if req.CspType == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "cspType is required"})
	}
	if req.AuthMethod == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "authMethod is required"})
	}

	log.Printf("[CSP_VALIDATE] kcUserId=%s workspaceId=%s csp=%s method=%s", kcUserId, req.WorkspaceID, req.CspType, req.AuthMethod)

	user, err := h.userService.GetUserByKcID(c.Request().Context(), kcUserId)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	resp, err := h.validationService.ValidateCredentials(c.Request().Context(), user.ID, kcUserId, &req)
	if err != nil {
		log.Printf("[CSP_VALIDATE] Error: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("validation failed: %v", err)})
	}

	return c.JSON(http.StatusOK, resp)
}
