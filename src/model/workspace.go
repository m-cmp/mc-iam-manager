package model

import (
	"time"
)

// Workspace 워크스페이스 모델 (DB 테이블: mcmp_workspaces)
type Workspace struct {
	ID          uint       `json:"id" gorm:"primaryKey;column:id"`
	Name        string     `json:"name" gorm:"column:name;size:255;not null"`
	Description string     `json:"description" gorm:"column:description;size:1000"`
	CreatedAt   time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	Projects    []*Project `json:"projects,omitempty" gorm:"many2many:mcmp_workspace_projects;"`
}

// TableName Workspace의 테이블 이름을 지정합니다
func (Workspace) TableName() string {
	return "mcmp_workspaces"
}

// WorkspaceWithProjects 워크스페이스와 연관된 프로젝트 정보를 포함하는 구조체
type WorkspaceWithProjects struct {
	ID          uint      `json:"id" gorm:"primaryKey;column:id"`
	Name        string    `json:"name" gorm:"column:name"`
	Description string    `json:"description" gorm:"column:description"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:updated_at"`
	Projects    []Project `json:"projects" gorm:"many2many:mcmp_workspace_projects;foreignKey:ID;joinForeignKey:workspace_id;References:ID;joinReferences:project_id"`
}

// WorkspaceWithUsersAndRoles 워크스페이스와 연관된 사용자 및 역할 정보를 포함하는 구조체
type WorkspaceWithUsersAndRoles struct {
	ID          uint                `json:"id" gorm:"primaryKey;column:id"`
	Name        string              `json:"name" gorm:"column:name"`
	Description string              `json:"description" gorm:"column:description"`
	CreatedAt   time.Time           `json:"created_at" gorm:"column:created_at"`
	UpdatedAt   time.Time           `json:"updated_at" gorm:"column:updated_at"`
	Users       []UserWorkspaceRole `json:"users" gorm:"foreignKey:WorkspaceID;references:ID"`
}

// TableName WorkspaceWithUsersAndRoles의 테이블 이름을 지정합니다
func (WorkspaceWithUsersAndRoles) TableName() string {
	return "mcmp_workspaces"
}

// UserWorkspace 사용자의 워크스페이스 정보를 담는 구조체
type UserWorkspace struct {
	WorkspaceID   uint         `json:"workspace_id"`
	WorkspaceName string       `json:"workspace_name"`
	Roles         []RoleMaster `json:"roles"`
}
