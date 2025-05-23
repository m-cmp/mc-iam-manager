package model

import (
	"time"

	"gorm.io/gorm"
)

// CspRole CSP 역할 모델
type CspRole struct {
	ID          string         `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	CspType     string         `json:"csp_type"`
	CspRoleArn  string         `json:"role_arn"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}
