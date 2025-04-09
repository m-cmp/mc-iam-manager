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

// Permission 권한 정보
type Permission struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:100;not null"`
	Type        PermissionType `json:"type" gorm:"size:20;not null"`
	Description string         `json:"description" gorm:"size:200"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// TableName 테이블 이름 지정
func (Permission) TableName() string {
	return "permissions"
}

// RolePermission 역할-권한 매핑
type RolePermission struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	RoleID       uint      `json:"role_id" gorm:"not null"`
	PermissionID uint      `json:"permission_id" gorm:"not null"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName 테이블 이름 지정
func (RolePermission) TableName() string {
	return "role_permissions"
}
