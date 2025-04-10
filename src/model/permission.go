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

// Permission 권한 정보 (DB 테이블: mcmp_permissions)
type Permission struct {
	ID          string    `json:"id" gorm:"primaryKey;column:id;type:varchar(255)"` // Changed to string
	Name        string    `json:"name" gorm:"column:name;size:100;not null"`        // Assuming Name column exists or needs to be added
	Description string    `json:"description" gorm:"column:description;size:1000"`  // Increased size to match roles
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	// Type        PermissionType `json:"type" gorm:"column:type;size:20;not null"` // Type column removed based on migration, add back if needed
}

// TableName 테이블 이름 지정
func (Permission) TableName() string {
	return "mcmp_permissions"
}

// RolePermission 역할-권한 매핑 (DB 테이블: mcmp_role_permissions)
type RolePermission struct {
	RoleType     string    `json:"role_type" gorm:"primaryKey;column:role_type;type:varchar(50);not null"`          // 'platform' or 'workspace'
	RoleID       uint      `json:"role_id" gorm:"primaryKey;column:role_id;not null"`                               // Refers to mcmp_platform_roles.id or mcmp_workspace_roles.id
	PermissionID string    `json:"permission_id" gorm:"primaryKey;column:permission_id;type:varchar(255);not null"` // Refers to mcmp_permissions.id
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	// Removed UpdatedAt as it's not in the migration schema
	// Relationships can be added if needed, e.g., PlatformRole, WorkspaceRole, Permission
}

// TableName 테이블 이름 지정
func (RolePermission) TableName() string {
	return "mcmp_role_permissions"
}
