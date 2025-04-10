package model

import "time"

// Workspace 워크스페이스 모델 (DB 테이블: mcmp_workspaces)
type Workspace struct {
	ID          uint       `json:"id" gorm:"primaryKey;column:id"`
	Name        string     `json:"name" gorm:"column:name;size:255;not null"`
	Description string     `json:"description" gorm:"column:description"`
	CreatedAt   time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	Projects    []*Project `json:"projects,omitempty" gorm:"many2many:mcmp_workspace_projects;"` // M:N relationship
}

// TableName Workspace의 테이블 이름을 지정합니다
func (Workspace) TableName() string {
	return "mcmp_workspaces"
}
