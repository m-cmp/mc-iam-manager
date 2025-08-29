package middleware

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// WorkspaceRoleMiddleware 워크스페이스 역할 기반 접근 제어 미들웨어
func WorkspaceRoleMiddleware(db *gorm.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 1. Context에서 kcUserId를 추출
			kcUserId, ok := c.Get("kcUserId").(string)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "user not authenticated")
			}

			// 2. Path에서 workspaceId를 추출
			workspaceIDStr := c.Param("workspaceId")
			workspaceID, err := strconv.ParseUint(workspaceIDStr, 10, 32)
			if err != nil {
				log.Printf("workspace_role_middleware: invalid workspace ID: %v", workspaceIDStr)
				return echo.NewHTTPError(http.StatusBadRequest, "workspace_role_middleware: invalid workspace ID")
			}

			// 3. DB에서 워크스페이스 티켓 정보 조회
			var workspaceTicket model.WorkspaceTicket
			err = db.Table("mcmp_workspace_tickets").Where("kc_user_id = ? AND workspace_id = ?", kcUserId, workspaceID).First(&workspaceTicket).Error
			if err != nil && err != gorm.ErrRecordNotFound {
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to check workspace ticket")
			}

			// 4. 티켓 발행 또는 갱신
			keycloakService := service.NewKeycloakService()
			var ticket string
			var permissions map[string]interface{}

			if err == gorm.ErrRecordNotFound {
				// 티켓이 없는 경우 새로 발행
				ticket, permissions, err = keycloakService.IssueWorkspaceTicket(c.Request().Context(), kcUserId, uint(workspaceID))
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to issue workspace ticket: %v", err))
				}
			} else {
				// 티켓이 있는 경우 유효기간 체크
				if time.Until(workspaceTicket.ExpiresAt) <= time.Minute {
					// 1분 이내로 남은 경우 갱신
					ticket, permissions, err = keycloakService.IssueWorkspaceTicket(c.Request().Context(), kcUserId, uint(workspaceID))
					if err != nil {
						return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to refresh workspace ticket: %v", err))
					}
				} else {
					// 유효한 티켓이 있는 경우 기존 정보 사용
					ticket = workspaceTicket.Ticket
					if err := json.Unmarshal(workspaceTicket.Permissions, &permissions); err != nil {
						return echo.NewHTTPError(http.StatusInternalServerError, "failed to parse permissions")
					}
				}
			}

			// 5. 티켓 정보 저장 또는 갱신
			expiresAt := time.Now().Add(30 * time.Minute) // 30분 유효기간
			permissionsJSON, err := json.Marshal(permissions)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to marshal permissions")
			}

			workspaceTicket = model.WorkspaceTicket{
				KcUserID:    kcUserId,
				WorkspaceID: uint(workspaceID),
				Ticket:      ticket,
				Permissions: datatypes.JSON(permissionsJSON),
				ExpiresAt:   expiresAt,
				LastUsedAt:  time.Now(),
			}

			if err := db.Table("mcmp_workspace_tickets").Save(&workspaceTicket).Error; err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to save workspace ticket")
			}

			// 6. Context에 정보 저장
			c.Set("workspace_id", uint(workspaceID))
			c.Set("workspace_ticket", ticket)
			c.Set("workspace_permissions", permissions)

			return next(c)
		}
	}
}
