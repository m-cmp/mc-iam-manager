package model

import (
	"time"
)

// RoleLastUsed AWS IAM Role의 마지막 사용 정보
type RoleLastUsed struct {
	LastUsedDate time.Time `json:"last_used_date"`
	Region       string    `json:"region"`
}

// Tag AWS IAM Role의 태그 정보
type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// CspRole CSP 역할 모델
// 대상 CSP와 연결하기 위한 연결정보(AWS에 OIDC로 연결되는 경우 제대로 동작. TODO: SAML을 추가했을 때 Table형태나 다른Table을 추가하게 될 수 있음)
type CspRole struct {
	ID                  uint          `gorm:"primaryKey" json:"id"`
	Name                string        `gorm:"size:255;not null" json:"name"`
	Description         string        `gorm:"size:255" json:"description"`
	CspType             string        `gorm:"size:50;not null" json:"csp_type"`
	IdpIdentifier       string        `gorm:"size:255" json:"idp_identifier"`
	IamIdentifier       string        `gorm:"size:255" json:"iam_identifier"`
	Status              string        `gorm:"size:50" json:"status"`
	CreateDate          time.Time     `json:"create_date"`
	Path                string        `gorm:"size:255" json:"path"`
	IamRoleId           string        `gorm:"size:255" json:"iam_role_id"`
	MaxSessionDuration  *int32        `json:"max_session_duration"`
	Permissions         []string      `gorm:"-" json:"permissions"`
	PermissionsBoundary string        `gorm:"size:255" json:"permissions_boundary"`
	RoleLastUsed        *RoleLastUsed `gorm:"type:jsonb;serializer:json" json:"role_last_used"`
	Tags                []Tag         `gorm:"-" json:"tags"`
	CreatedAt           time.Time     `json:"created_at"`
	UpdatedAt           time.Time     `json:"updated_at"`
	DeletedAt           *time.Time    `json:"deleted_at" gorm:"index"`
}

func (CspRole) TableName() string { // Renamed receiver
	return "mcmp_role_csp_roles"
}
