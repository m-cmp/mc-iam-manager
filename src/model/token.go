package model

import (
	"time"
)

type Token struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    string    `json:"user_id" gorm:"not null"`
	Token     string    `json:"token" gorm:"not null"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WorkspaceTicketRequest 워크스페이스 티켓(RPT) 발급 요청 모델
type WorkspaceTicketRequest struct {
	WorkspaceID string   `json:"workspaceId" validate:"required"` // 대상 워크스페이스 ID (문자열로 처리하는 것이 Keycloak 리소스 ID와 일치시키기 용이할 수 있음)
	Permissions []string `json:"permissions" validate:"required"` // 요청할 권한 목록 (예: ["resource_vm#scope_create", "resource_vm#scope_read"])
}

func (Token) TableName() string {
	return "mcmp_token"
}
