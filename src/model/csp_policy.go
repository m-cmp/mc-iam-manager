package model

import (
	"time"
)

// PolicyType 정책 타입
type PolicyType string

const (
	PolicyTypeInline  PolicyType = "inline"  // 인라인 정책 (역할에 직접 포함)
	PolicyTypeManaged PolicyType = "managed" // 관리형 정책 (독립 정책)
	PolicyTypeCustom  PolicyType = "custom"  // 사용자 정의 정책
)

// CspPolicy CSP 정책 모델
// AWS IAM Policy, GCP IAM Role, Azure Role Definition 등을 관리
type CspPolicy struct {
	ID           uint                   `gorm:"primaryKey" json:"id"`
	Name         string                 `gorm:"size:255;not null" json:"name"`
	CspAccountID uint                   `gorm:"not null" json:"csp_account_id"`
	CspAccount   *CspAccount            `gorm:"foreignKey:CspAccountID" json:"csp_account,omitempty"`
	PolicyType   PolicyType             `gorm:"size:50;not null" json:"policy_type"` // inline, managed, custom
	PolicyArn    string                 `gorm:"size:500" json:"policy_arn,omitempty"`
	PolicyDoc    map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"policy_doc,omitempty"`
	Description  string                 `gorm:"size:500" json:"description"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// TableName CspPolicy 테이블 이름 반환
func (CspPolicy) TableName() string {
	return "mcmp_csp_policies"
}

// CspRolePolicyMapping CSP 역할-정책 매핑 모델
type CspRolePolicyMapping struct {
	CspRoleID   uint       `gorm:"primaryKey" json:"csp_role_id"`
	CspPolicyID uint       `gorm:"primaryKey" json:"csp_policy_id"`
	CspRole     *CspRole   `gorm:"foreignKey:CspRoleID" json:"csp_role,omitempty"`
	CspPolicy   *CspPolicy `gorm:"foreignKey:CspPolicyID" json:"csp_policy,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// TableName CspRolePolicyMapping 테이블 이름 반환
func (CspRolePolicyMapping) TableName() string {
	return "mcmp_csp_role_policy_mappings"
}

// PolicyDocument AWS IAM Policy Document 구조
// PolicyDoc 예시 (AWS):
//
//	{
//	  "Version": "2012-10-17",
//	  "Statement": [
//	    {
//	      "Effect": "Allow",
//	      "Action": ["s3:GetObject", "s3:PutObject"],
//	      "Resource": ["arn:aws:s3:::my-bucket/*"]
//	    }
//	  ]
//	}
//
// PolicyDoc 예시 (GCP):
//
//	{
//	  "included_permissions": ["storage.objects.get", "storage.objects.create"],
//	  "stage": "GA"
//	}
//
// PolicyDoc 예시 (Azure):
//
//	{
//	  "actions": ["Microsoft.Storage/storageAccounts/read"],
//	  "not_actions": [],
//	  "data_actions": [],
//	  "not_data_actions": []
//	}

// IsInline 인라인 정책인지 확인
func (p *CspPolicy) IsInline() bool {
	return p.PolicyType == PolicyTypeInline
}

// IsManaged 관리형 정책인지 확인
func (p *CspPolicy) IsManaged() bool {
	return p.PolicyType == PolicyTypeManaged
}

// IsCustom 사용자 정의 정책인지 확인
func (p *CspPolicy) IsCustom() bool {
	return p.PolicyType == PolicyTypeCustom
}

// GetPolicyVersion 정책 버전 반환 (AWS)
func (p *CspPolicy) GetPolicyVersion() string {
	if p.PolicyDoc == nil {
		return ""
	}
	if version, ok := p.PolicyDoc["Version"].(string); ok {
		return version
	}
	return ""
}

// GetStatements 정책 Statement 반환 (AWS)
func (p *CspPolicy) GetStatements() []interface{} {
	if p.PolicyDoc == nil {
		return nil
	}
	if statements, ok := p.PolicyDoc["Statement"].([]interface{}); ok {
		return statements
	}
	return nil
}

// CspPolicyFilter CSP 정책 조회 필터
type CspPolicyFilter struct {
	CspAccountID *uint      `json:"csp_account_id,omitempty"`
	PolicyType   PolicyType `json:"policy_type,omitempty"`
	Name         string     `json:"name,omitempty"`
}

// CreateCspPolicyRequest CSP 정책 생성 요청
type CreateCspPolicyRequest struct {
	Name         string                 `json:"name" binding:"required"`
	CspAccountID uint                   `json:"csp_account_id" binding:"required"`
	PolicyType   PolicyType             `json:"policy_type" binding:"required,oneof=inline managed custom"`
	PolicyArn    string                 `json:"policy_arn"`
	PolicyDoc    map[string]interface{} `json:"policy_doc"`
	Description  string                 `json:"description"`
}

// UpdateCspPolicyRequest CSP 정책 수정 요청
type UpdateCspPolicyRequest struct {
	Name        string                 `json:"name"`
	PolicyArn   string                 `json:"policy_arn"`
	PolicyDoc   map[string]interface{} `json:"policy_doc"`
	Description string                 `json:"description"`
}

// AttachPolicyRequest 정책 연결 요청
type AttachPolicyRequest struct {
	CspRoleID   uint `json:"csp_role_id" binding:"required"`
	CspPolicyID uint `json:"csp_policy_id" binding:"required"`
}

// SyncPoliciesRequest 정책 동기화 요청
type SyncPoliciesRequest struct {
	CspAccountID uint   `json:"csp_account_id" binding:"required"`
	PolicyScope  string `json:"policy_scope"` // All, AWS, Local
}
