package model

import (
	"time"

	"gorm.io/gorm"
)

// CspRolePermission CSP 역할의 권한 모델
type CspRolePermission struct {
	ID         string         `json:"id" gorm:"primaryKey"`
	CspRoleID  string         `json:"csp_role_id" gorm:"index"`
	Permission string         `json:"permission"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

func (CspRolePermission) TableName() string {
	return "mcmp_csp_role_permissions"
}
