package handler

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
	"github.com/m-cmp/mc-iam-manager/util"
	"gorm.io/gorm"
)

// CspAccountHandler CSP 계정 관리 핸들러
type CspAccountHandler struct {
	cspAccountService *service.CspAccountService
}

// NewCspAccountHandler 새 CspAccountHandler 인스턴스 생성
func NewCspAccountHandler(db *gorm.DB) *CspAccountHandler {
	return &CspAccountHandler{
		cspAccountService: service.NewCspAccountService(db),
	}
}

// CreateCspAccount godoc
// @Summary Create CSP account
// @Description Create a new CSP account
// @Tags csp-accounts
// @Accept json
// @Produce json
// @Param account body model.CreateCspAccountRequest true "CSP Account Info"
// @Success 201 {object} model.CspAccount
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/csp-accounts [post]
// @Id createCspAccount
func (h *CspAccountHandler) CreateCspAccount(c echo.Context) error {
	var req model.CreateCspAccountRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	// 필수 필드 검증
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Name is required"})
	}
	if req.CspType == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "CSP type is required"})
	}
	if req.CspType != "aws" && req.CspType != "gcp" && req.CspType != "azure" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid CSP type. Must be one of: aws, gcp, azure"})
	}

	account, err := h.cspAccountService.CreateCspAccount(&req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to create CSP account: %v", err)})
	}

	return c.JSON(http.StatusCreated, account)
}

// ListCspAccounts godoc
// @Summary List CSP accounts
// @Description Retrieve a list of CSP accounts with optional filters
// @Tags csp-accounts
// @Accept json
// @Produce json
// @Param filter body model.CspAccountFilter false "Filter options"
// @Success 200 {array} model.CspAccount
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/csp-accounts/list [post]
// @Id listCspAccounts
func (h *CspAccountHandler) ListCspAccounts(c echo.Context) error {
	var filter model.CspAccountFilter
	if err := c.Bind(&filter); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	accounts, err := h.cspAccountService.ListCspAccounts(&filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to list CSP accounts: %v", err)})
	}

	if accounts == nil {
		accounts = []*model.CspAccount{}
	}

	return c.JSON(http.StatusOK, accounts)
}

// GetCspAccountByID godoc
// @Summary Get CSP account by ID
// @Description Retrieve CSP account details by ID
// @Tags csp-accounts
// @Accept json
// @Produce json
// @Param accountId path string true "Account ID"
// @Success 200 {object} model.CspAccount
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/csp-accounts/id/{accountId} [get]
// @Id getCspAccountByID
func (h *CspAccountHandler) GetCspAccountByID(c echo.Context) error {
	accountID, err := util.StringToUint(c.Param("accountId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid account ID"})
	}

	account, err := h.cspAccountService.GetCspAccountByID(accountID)
	if err != nil {
		if err.Error() == fmt.Sprintf("CSP account not found with ID: %d", accountID) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to get CSP account: %v", err)})
	}

	return c.JSON(http.StatusOK, account)
}

// UpdateCspAccount godoc
// @Summary Update CSP account
// @Description Update CSP account details
// @Tags csp-accounts
// @Accept json
// @Produce json
// @Param accountId path string true "Account ID"
// @Param account body model.UpdateCspAccountRequest true "CSP Account Info"
// @Success 200 {object} model.CspAccount
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/csp-accounts/id/{accountId} [put]
// @Id updateCspAccount
func (h *CspAccountHandler) UpdateCspAccount(c echo.Context) error {
	accountID, err := util.StringToUint(c.Param("accountId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid account ID"})
	}

	var req model.UpdateCspAccountRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	account, err := h.cspAccountService.UpdateCspAccount(accountID, &req)
	if err != nil {
		if err.Error() == fmt.Sprintf("CSP account not found with ID: %d", accountID) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to update CSP account: %v", err)})
	}

	return c.JSON(http.StatusOK, account)
}

// DeleteCspAccount godoc
// @Summary Delete CSP account
// @Description Delete a CSP account by ID
// @Tags csp-accounts
// @Accept json
// @Produce json
// @Param accountId path string true "Account ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/csp-accounts/id/{accountId} [delete]
// @Id deleteCspAccount
func (h *CspAccountHandler) DeleteCspAccount(c echo.Context) error {
	accountID, err := util.StringToUint(c.Param("accountId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid account ID"})
	}

	if err := h.cspAccountService.DeleteCspAccount(accountID); err != nil {
		if err.Error() == fmt.Sprintf("CSP account not found with ID: %d", accountID) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to delete CSP account: %v", err)})
	}

	return c.NoContent(http.StatusNoContent)
}

// ValidateCspAccount godoc
// @Summary Validate CSP account
// @Description Validate CSP account configuration
// @Tags csp-accounts
// @Accept json
// @Produce json
// @Param accountId path string true "Account ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/csp-accounts/id/{accountId}/validate [post]
// @Id validateCspAccount
func (h *CspAccountHandler) ValidateCspAccount(c echo.Context) error {
	accountID, err := util.StringToUint(c.Param("accountId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid account ID"})
	}

	if err := h.cspAccountService.ValidateCspAccount(accountID); err != nil {
		if err.Error() == fmt.Sprintf("CSP account not found with ID: %d", accountID) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Validation failed: %v", err)})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "CSP account is valid"})
}

// ActivateCspAccount godoc
// @Summary Activate CSP account
// @Description Activate a CSP account
// @Tags csp-accounts
// @Accept json
// @Produce json
// @Param accountId path string true "Account ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/csp-accounts/id/{accountId}/activate [post]
// @Id activateCspAccount
func (h *CspAccountHandler) ActivateCspAccount(c echo.Context) error {
	accountID, err := util.StringToUint(c.Param("accountId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid account ID"})
	}

	if err := h.cspAccountService.ActivateCspAccount(accountID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to activate CSP account: %v", err)})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "CSP account activated successfully"})
}

// DeactivateCspAccount godoc
// @Summary Deactivate CSP account
// @Description Deactivate a CSP account
// @Tags csp-accounts
// @Accept json
// @Produce json
// @Param accountId path string true "Account ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/csp-accounts/id/{accountId}/deactivate [post]
// @Id deactivateCspAccount
func (h *CspAccountHandler) DeactivateCspAccount(c echo.Context) error {
	accountID, err := util.StringToUint(c.Param("accountId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid account ID"})
	}

	if err := h.cspAccountService.DeactivateCspAccount(accountID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to deactivate CSP account: %v", err)})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "CSP account deactivated successfully"})
}
