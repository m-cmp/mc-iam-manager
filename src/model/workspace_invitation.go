package model

import "time"

// InvitationStatus 워크스페이스 초대 상태
type InvitationStatus string

const (
	InvitationStatusPending         InvitationStatus = "PENDING"
	InvitationStatusPendingApproval InvitationStatus = "PENDING_APPROVAL"
	InvitationStatusAccepted        InvitationStatus = "ACCEPTED"
	InvitationStatusRejected        InvitationStatus = "REJECTED"
)

// WorkspaceInvitation 워크스페이스 초대 모델 (DB 테이블: mcmp_workspace_invitations)
type WorkspaceInvitation struct {
	ID            uint             `json:"id" gorm:"primaryKey;column:id"`
	WorkspaceID   uint             `json:"workspaceId" gorm:"column:workspace_id;not null"`
	InviterUserID uint             `json:"inviterUserId" gorm:"column:inviter_user_id;not null"`
	InviteeUserID uint             `json:"inviteeUserId" gorm:"column:invitee_user_id;not null"`
	RoleID        *uint            `json:"roleId,omitempty" gorm:"column:role_id"`
	Status        InvitationStatus `json:"status" gorm:"column:status;not null;default:'PENDING'"`
	CreatedAt     time.Time        `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time        `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
}

// TableName WorkspaceInvitation의 테이블 이름 지정
func (WorkspaceInvitation) TableName() string {
	return "mcmp_workspace_invitations"
}

// SendInvitationRequest 워크스페이스 초대 발송 요청
type SendInvitationRequest struct {
	InviteeUserID uint  `json:"inviteeUserId" validate:"required"`
	RoleID        *uint `json:"roleId,omitempty"`
}

// InvitationFilterRequest 초대 목록 필터 요청
type InvitationFilterRequest struct {
	Status string `query:"status"`
}
