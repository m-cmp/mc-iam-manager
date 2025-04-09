package model

import "time"

// PlatformRole 플랫폼 역할 모델
type PlatformRole struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:255;not null;unique" json:"name"`
	Description string    `gorm:"size:1000" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName PlatformRole의 테이블 이름을 지정합니다
func (PlatformRole) TableName() string {
	return "platform_roles"
}

// WorkspaceRole 워크스페이스 역할 모델
type WorkspaceRole struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:255;not null;unique" json:"name"`
	Description string    `gorm:"size:1000" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName WorkspaceRole의 테이블 이름을 지정합니다
func (WorkspaceRole) TableName() string {
	return "workspace_roles"
}
