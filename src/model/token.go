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

// WorkspaceTicketRequest 워크스페이스 티켓 요청 모델
type WorkspaceTicketRequest struct {
	WorkspaceID string `json:"workspace_id" validate:"required"`
}

func (Token) TableName() string {
	return "mcmp_token"
}
