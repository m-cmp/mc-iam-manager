package model

import (
	"time"

	"gorm.io/datatypes"
)

// WorkspaceTicket 워크스페이스 티켓 정보를 저장하는 모델
type WorkspaceTicket struct {
	ID          uint           `gorm:"primaryKey"`
	KcUserID    string         `gorm:"not null"`
	WorkspaceID uint           `gorm:"not null"`
	Ticket      string         `gorm:"not null"`
	Permissions datatypes.JSON `gorm:"type:jsonb;not null"`
	ExpiresAt   time.Time      `gorm:"not null"`
	LastUsedAt  time.Time      `gorm:"not null"`
	CreatedAt   time.Time      `gorm:"not null"`
	UpdatedAt   time.Time      `gorm:"not null"`
}
