package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/gorm"
)

// WorkspaceInvitationHandler 워크스페이스 초대 핸들러
type WorkspaceInvitationHandler struct {
	invitationService *service.WorkspaceInvitationService
	userService       *service.UserService
}

// NewWorkspaceInvitationHandler 새 WorkspaceInvitationHandler 인스턴스 생성
func NewWorkspaceInvitationHandler(db *gorm.DB) *WorkspaceInvitationHandler {
	return &WorkspaceInvitationHandler{
		invitationService: service.NewWorkspaceInvitationService(db),
		userService:       service.NewUserService(db),
	}
}

// getCallerUserID JWT 컨텍스트에서 현재 사용자의 DB ID 조회
func (h *WorkspaceInvitationHandler) getCallerUserID(c echo.Context) (uint, error) {
	kcUserIdVal := c.Get("kcUserId")
	if kcUserIdVal == nil {
		return 0, errors.New("kcUserId not found in context")
	}
	kcUserID, ok := kcUserIdVal.(string)
	if !ok {
		return 0, errors.New("invalid kcUserId in context")
	}
	userID, err := h.userService.GetUserIDByKcID(c.Request().Context(), kcUserID)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

// SendInvitation godoc
// @Summary Send workspace invitation
// @Description Send an invitation to a platform user to join the workspace
// @Tags workspaces
// @Accept json
// @Produce json
// @Param wsId path int true "Workspace ID"
// @Param body body model.SendInvitationRequest true "Invitation request"
// @Success 201 {object} model.WorkspaceInvitation
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /api/workspaces/id/{wsId}/invitations [post]
// @Id sendWorkspaceInvitation
func (h *WorkspaceInvitationHandler) SendInvitation(c echo.Context) error {
	wsID, err := strconv.ParseUint(c.Param("wsId"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid workspace ID"})
	}

	callerID, err := h.getCallerUserID(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var req model.SendInvitationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request format"})
	}
	if req.InviteeUserID == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "inviteeUserId is required"})
	}

	invitation, err := h.invitationService.SendInvitation(uint(wsID), callerID, req.InviteeUserID, req.RoleID)
	if err != nil {
		if err.Error() == "user is already a member of this workspace" ||
			err.Error() == "pending invitation already exists for this user" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, invitation)
}

// ListWorkspaceInvitations godoc
// @Summary List workspace invitations
// @Description List invitations for a specific workspace
// @Tags workspaces
// @Accept json
// @Produce json
// @Param wsId path int true "Workspace ID"
// @Param status query string false "Filter by status (PENDING/ACCEPTED/REJECTED)"
// @Success 200 {array} model.WorkspaceInvitation
// @Security BearerAuth
// @Router /api/workspaces/id/{wsId}/invitations [get]
// @Id listWorkspaceInvitations
func (h *WorkspaceInvitationHandler) ListWorkspaceInvitations(c echo.Context) error {
	wsID, err := strconv.ParseUint(c.Param("wsId"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid workspace ID"})
	}
	status := c.QueryParam("status")

	invitations, err := h.invitationService.ListWorkspaceInvitations(uint(wsID), status)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, invitations)
}

// ListMyInvitations godoc
// @Summary List my invitations
// @Description List pending workspace invitations for the current user
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {array} model.WorkspaceInvitation
// @Security BearerAuth
// @Router /api/users/me/invitations [get]
// @Id listMyInvitations
func (h *WorkspaceInvitationHandler) ListMyInvitations(c echo.Context) error {
	callerID, err := h.getCallerUserID(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	invitations, err := h.invitationService.ListMyInvitations(callerID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, invitations)
}

// AcceptInvitation godoc
// @Summary Accept workspace invitation
// @Description Accept a workspace invitation and join as member
// @Tags users
// @Accept json
// @Produce json
// @Param invitationId path int true "Invitation ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Security BearerAuth
// @Router /api/users/me/invitations/{invitationId}/accept [put]
// @Id acceptInvitation
func (h *WorkspaceInvitationHandler) AcceptInvitation(c echo.Context) error {
	invitationID, err := strconv.ParseUint(c.Param("invitationId"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid invitation ID"})
	}

	callerID, err := h.getCallerUserID(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if err := h.invitationService.AcceptInvitation(uint(invitationID), callerID); err != nil {
		if err.Error() == "forbidden: not your invitation" {
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "invitation accepted"})
}

// RejectInvitation godoc
// @Summary Reject workspace invitation
// @Description Reject a workspace invitation
// @Tags users
// @Accept json
// @Produce json
// @Param invitationId path int true "Invitation ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Security BearerAuth
// @Router /api/users/me/invitations/{invitationId}/reject [put]
// @Id rejectInvitation
func (h *WorkspaceInvitationHandler) RejectInvitation(c echo.Context) error {
	invitationID, err := strconv.ParseUint(c.Param("invitationId"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid invitation ID"})
	}

	callerID, err := h.getCallerUserID(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if err := h.invitationService.RejectInvitation(uint(invitationID), callerID); err != nil {
		if err.Error() == "forbidden: not your invitation" {
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "invitation rejected"})
}

// ListAllInvitations godoc
// @Summary List all invitations (admin)
// @Description List all workspace invitations, optionally filtered by status
// @Tags invitations
// @Accept json
// @Produce json
// @Param status query string false "Filter by status (PENDING_APPROVAL)"
// @Success 200 {array} model.WorkspaceInvitation
// @Security BearerAuth
// @Router /api/invitations [get]
// @Id listAllInvitations
func (h *WorkspaceInvitationHandler) ListAllInvitations(c echo.Context) error {
	status := c.QueryParam("status")
	invitations, err := h.invitationService.ListPendingApprovals(status)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, invitations)
}

// ApproveInvitation godoc
// @Summary Approve workspace invitation (admin)
// @Description Admin approves a workspace invitation and registers the invitee as member
// @Tags invitations
// @Accept json
// @Produce json
// @Param invitationId path int true "Invitation ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/invitations/{invitationId}/approve [put]
// @Id approveInvitation
func (h *WorkspaceInvitationHandler) ApproveInvitation(c echo.Context) error {
	invitationID, err := strconv.ParseUint(c.Param("invitationId"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid invitation ID"})
	}

	if err := h.invitationService.ApproveInvitation(uint(invitationID)); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "invitation approved"})
}

// RejectInvitationByAdmin godoc
// @Summary Reject workspace invitation (admin)
// @Description Admin rejects a workspace invitation
// @Tags invitations
// @Accept json
// @Produce json
// @Param invitationId path int true "Invitation ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/invitations/{invitationId}/reject [put]
// @Id rejectInvitationByAdmin
func (h *WorkspaceInvitationHandler) RejectInvitationByAdmin(c echo.Context) error {
	invitationID, err := strconv.ParseUint(c.Param("invitationId"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid invitation ID"})
	}

	if err := h.invitationService.RejectInvitationByAdmin(uint(invitationID)); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "invitation rejected"})
}
