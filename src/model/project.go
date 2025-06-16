package model

import "time"

// Project 프로젝트 모델 (DB 테이블: mcmp_projects)
type Project struct {
	ID          uint         `json:"id" gorm:"primaryKey;column:id"`
	NsId        string       `json:"nsid" gorm:"column:nsid;size:255"` // Namespace ID
	Name        string       `json:"name" gorm:"column:name;size:255;not null"`
	Description string       `json:"description" gorm:"column:description;size:1000"`
	CreatedAt   time.Time    `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time    `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	Workspaces  []*Workspace `json:"workspaces,omitempty" gorm:"many2many:mcmp_workspace_projects;"` // M:N relationship
}

type ProjectFilterRequest struct {
	ProjectID     string `json:"project_id"`
	ProjectName   string `json:"project_name"`
	WorkspaceID   string `json:"workspace_id"`
	WorkspaceName string `json:"workspace_name"`
}

// TableName Project의 테이블 이름을 지정합니다
func (Project) TableName() string {
	return "mcmp_projects"
}
