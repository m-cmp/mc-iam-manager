package model

import (
	"time"
)

// CspAccount CSP 계정 정보 모델
// AWS Account ID, GCP Project ID, Azure Subscription ID 등 CSP별 계정 정보를 관리
type CspAccount struct {
	ID          uint              `gorm:"primaryKey" json:"id"`
	Name        string            `gorm:"size:255;not null" json:"name"`
	CspType     string            `gorm:"size:50;not null" json:"csp_type"` // aws, gcp, azure
	AccountInfo map[string]string `gorm:"type:jsonb;serializer:json" json:"account_info"`
	IsActive    bool              `gorm:"default:true" json:"is_active"`
	Description string            `gorm:"size:500" json:"description"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// TableName CspAccount 테이블 이름 반환
func (CspAccount) TableName() string {
	return "mcmp_csp_accounts"
}

// CspAccountInfo CSP별 계정 정보 구조
// AWS AccountInfo 예시:
//
//	{
//	  "account_id": "050864702683",
//	  "alias": "my-aws-account",
//	  "region": "ap-northeast-2"
//	}
//
// GCP AccountInfo 예시:
//
//	{
//	  "project_id": "my-gcp-project",
//	  "project_number": "123456789"
//	}
//
// Azure AccountInfo 예시:
//
//	{
//	  "subscription_id": "xxx-xxx-xxx",
//	  "tenant_id": "yyy-yyy-yyy",
//	  "directory_id": "zzz-zzz-zzz"
//	}

// GetAccountID AWS Account ID 반환
func (c *CspAccount) GetAccountID() string {
	if c.AccountInfo == nil {
		return ""
	}
	return c.AccountInfo["account_id"]
}

// GetProjectID GCP Project ID 반환
func (c *CspAccount) GetProjectID() string {
	if c.AccountInfo == nil {
		return ""
	}
	return c.AccountInfo["project_id"]
}

// GetSubscriptionID Azure Subscription ID 반환
func (c *CspAccount) GetSubscriptionID() string {
	if c.AccountInfo == nil {
		return ""
	}
	return c.AccountInfo["subscription_id"]
}

// GetTenantID Azure Tenant ID 반환
func (c *CspAccount) GetTenantID() string {
	if c.AccountInfo == nil {
		return ""
	}
	return c.AccountInfo["tenant_id"]
}

// GetRegion 리전 정보 반환
func (c *CspAccount) GetRegion() string {
	if c.AccountInfo == nil {
		return ""
	}
	return c.AccountInfo["region"]
}

// GetAlibabaAccountID Alibaba Cloud Account ID 반환
// Alibaba는 AWS와 동일하게 "account_id" 키를 사용하므로 GetAccountID()와 동일 동작이지만,
// CSP별 getter 네이밍 컨벤션(Get<Csp>ID)을 맞추기 위해 별도로 노출한다.
func (c *CspAccount) GetAlibabaAccountID() string {
	return c.GetAccountID()
}

// GetTencentAppID Tencent Cloud APPID 반환
func (c *CspAccount) GetTencentAppID() string {
	if c.AccountInfo == nil {
		return ""
	}
	return c.AccountInfo["app_id"]
}

// CspAccountFilter CSP 계정 조회 필터
type CspAccountFilter struct {
	CspType  string `json:"csp_type,omitempty"`
	IsActive *bool  `json:"is_active,omitempty"`
	Name     string `json:"name,omitempty"`
}

// CreateCspAccountRequest CSP 계정 생성 요청
type CreateCspAccountRequest struct {
	Name        string            `json:"name" binding:"required"`
	CspType     string            `json:"csp_type" binding:"required,oneof=aws gcp azure alibaba tencent ibm ncp nhn kt openstack"`
	AccountInfo map[string]string `json:"account_info"`
	Description string            `json:"description"`
}

// UpdateCspAccountRequest CSP 계정 수정 요청
type UpdateCspAccountRequest struct {
	Name        string            `json:"name"`
	AccountInfo map[string]string `json:"account_info"`
	IsActive    *bool             `json:"is_active"`
	Description string            `json:"description"`
}

// CspAccountValidationResult 개별 CspRole(provider+method) 검증 결과
type CspAccountValidationResult struct {
	CspRoleID   uint   `json:"csp_role_id"`
	CspRoleName string `json:"csp_role_name"`
	CspType     string `json:"csp_type"`
	AuthMethod  string `json:"auth_method"`
	Valid       bool   `json:"valid"`
	Detail      string `json:"detail,omitempty"`
	Error       string `json:"error,omitempty"`
}

// CspAccountValidationResponse CSP 계정 인프라 검증 응답
type CspAccountValidationResponse struct {
	AccountID   uint                         `json:"account_id"`
	AccountName string                       `json:"account_name"`
	CspType     string                       `json:"csp_type"`
	Results     []CspAccountValidationResult `json:"results"`
}
