package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/service"
)

type Response struct {
	Error   string `json:"error,omitempty"`
	Message string `json:"message"`
}

type adminHandler struct {
	keycloakService service.KeycloakService
}

func NewAdminHandler(keycloakService service.KeycloakService) *adminHandler {
	return &adminHandler{
		keycloakService: keycloakService,
	}
}

// SetupInitialAdminRequest represents the request body for setting up the initial admin
type SetupInitialAdminRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

// SetupInitialAdmin godoc
// @Summary Setup initial platform admin
// @Description Creates the initial platform admin user with necessary permissions
// @Tags admin
// @Accept json
// @Produce json
// @Param request body SetupInitialAdminRequest true "Setup Initial Admin Request"
// @Success 200 {object} Response
// @Router /api/admin/setup [post]
func (h *adminHandler) SetupInitialAdmin(c echo.Context) error {
	var req SetupInitialAdminRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, Response{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
	}

	if err := h.keycloakService.SetupInitialAdmin(c.Request().Context()); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Error:   "Failed to setup initial admin",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, Response{
		Message: "Initial admin setup completed successfully",
	})
}
