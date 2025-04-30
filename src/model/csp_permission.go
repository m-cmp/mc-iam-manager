package model

import "time"

// WorkspaceRoleCspRoleMapping 워크스페이스 역할 - CSP 역할 매핑 (DB 테이블: mcmp_workspace_role_csp_role_mapping) - Renamed
type WorkspaceRoleCspRoleMapping struct {
	WorkspaceRoleID uint      `json:"workspaceRoleId" gorm:"primaryKey;column:workspace_role_id;not null"`         // FK to mcmp_workspace_roles.id
	CspType         string    `json:"cspType" gorm:"primaryKey;column:csp_type;type:varchar(50);not null"`         // e.g., "aws", "gcp", "azure"
	CspRoleArn      string    `json:"cspRoleArn" gorm:"primaryKey;column:csp_role_arn;type:varchar(255);not null"` // The actual ARN or identifier of the role in the CSP
	IdpIdentifier   string    `json:"idpIdentifier" gorm:"column:idp_identifier;type:varchar(255)"`                // e.g., AWS OIDC Provider ARN (Nullable)
	Description     string    `json:"description" gorm:"column:description;size:1000"`                             // Description of this specific mapping (Nullable)
	CreatedAt       time.Time `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	// WorkspaceRole   WorkspaceRole `json:"-" gorm:"foreignKey:WorkspaceRoleID"` // Optional relationship
}

// TableName 테이블 이름 지정
func (WorkspaceRoleCspRoleMapping) TableName() string { // Renamed receiver
	return "mcmp_workspace_role_csp_role_mapping" // Updated table name
}

// CspPermission struct removed as it's no longer needed
