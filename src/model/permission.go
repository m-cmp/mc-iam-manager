package model

import (
	"time"
)

// PermissionType 권한 유형
type PermissionType string

const (
	// PermissionTypeMenu 메뉴 권한
	PermissionTypeMenu PermissionType = "menu"
	// PermissionTypeResource 리소스 권한
	PermissionTypeResource PermissionType = "resource"
)

// MciamPermission 권한 정보 (DB 테이블: mcmp_mciam_permissions) - Renamed
type MciamPermission struct {
	ID             string    `json:"id" gorm:"primaryKey;column:id;type:varchar(255)"`                         // Format: <framework_id>:<resource_type_id>:<action>
	FrameworkID    string    `json:"frameworkId" gorm:"column:framework_id;type:varchar(100);not null"`        // FK to mcmp_resource_types.framework_id
	ResourceTypeID string    `json:"resourceTypeId" gorm:"column:resource_type_id;type:varchar(100);not null"` // FK to mcmp_resource_types.id
	Action         string    `json:"action" gorm:"column:action;type:varchar(100);not null"`                   // e.g., create, read, update, delete
	Name           string    `json:"name" gorm:"column:name;size:100;not null"`
	Description    string    `json:"description" gorm:"column:description;size:1000"`
	CreatedAt      time.Time `json:"createdAt" gorm:"column:created_at;not null;default:now()"` // Match DB schema
	UpdatedAt      time.Time `json:"updatedAt" gorm:"column:updated_at;not null;default:now()"` // Match DB schema
	// ResourceType   ResourceType `gorm:"foreignKey:FrameworkID,ResourceTypeID;references:FrameworkID,ID"` // Optional: Define relationship if needed
}

// TableName 테이블 이름 지정
func (MciamPermission) TableName() string { // Renamed receiver
	return "mcmp_mciam_permissions" // Updated table name
}

// MciamRoleMciamPermission 역할-MC-IAM 권한 매핑 (DB 테이블: mcmp_mciam_role_permissions) - Renamed
type MciamRoleMciamPermission struct {
	RoleType        string    `json:"role_type" gorm:"primaryKey;column:role_type;type:varchar(50);not null"`          // 'platform' or 'workspace'
	WorkspaceRoleID uint      `json:"workspace_role_id" gorm:"primaryKey;column:workspace_role_id;not null"`           // Renamed, Refers to mcmp_workspace_roles.id
	PermissionID    string    `json:"permission_id" gorm:"primaryKey;column:permission_id;type:varchar(255);not null"` // Refers to mcmp_mciam_permissions.id
	CreatedAt       time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	// Relationships can be added if needed, e.g., WorkspaceRole, MciamPermission
}

// TableName 테이블 이름 지정
func (MciamRoleMciamPermission) TableName() string { // Renamed receiver
	return "mcmp_mciam_role_permissions" // Updated table name
}
