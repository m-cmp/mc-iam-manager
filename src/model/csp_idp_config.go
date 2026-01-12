package model

import (
	"time"
)

// AuthMethodType IDP 인증 방식 타입
type AuthMethodType string

const (
	AuthMethodOIDC      AuthMethodType = "OIDC"
	AuthMethodSAML      AuthMethodType = "SAML"
	AuthMethodSecretKey AuthMethodType = "SECRET_KEY"
)

// CspIdpConfig CSP IDP 연동 설정 모델
// OIDC, SAML, Secret Key 등 CSP별 IDP 연동 정보를 관리
type CspIdpConfig struct {
	ID           uint              `gorm:"primaryKey" json:"id"`
	Name         string            `gorm:"size:255;not null" json:"name"`
	CspAccountID uint              `gorm:"not null" json:"csp_account_id"`
	CspAccount   *CspAccount       `gorm:"foreignKey:CspAccountID" json:"csp_account,omitempty"`
	AuthMethod   AuthMethodType    `gorm:"size:50;not null" json:"auth_method"` // OIDC, SAML, SECRET_KEY
	Config       map[string]string `gorm:"type:jsonb;serializer:json" json:"config"`
	IsActive     bool              `gorm:"default:true" json:"is_active"`
	Description  string            `gorm:"size:500" json:"description"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// TableName CspIdpConfig 테이블 이름 반환
func (CspIdpConfig) TableName() string {
	return "mcmp_csp_idp_configs"
}

// CspIdpConfigInfo 인증 방식별 설정 정보 구조
// OIDC Config 예시:
//
//	{
//	  "oidc_provider_arn": "arn:aws:iam::050864702683:oidc-provider/keycloak.example.com",
//	  "audience": "mciam-client",
//	  "sts_endpoint": "https://sts.amazonaws.com"
//	}
//
// SAML Config 예시:
//
// AWS SAML:
//
//	{
//	  "saml_provider_arn": "arn:aws:iam::050864702683:saml-provider/keycloak",
//	  "sso_service_location": "https://devkc.onecloudcon.com/realms/saml-demo/protocol/saml",
//	  "issuer_url": "https://devkc.onecloudcon.com/realms/saml-demo",
//	  "signin_url": "https://signin.aws.amazon.com/saml/acs/SAMLSPD9IMTW2LM46J042K",
//	  "metadata_url": "https://signin.aws.amazon.com/static/saml/SAMLSPD9IMTW2LM46J042K/saml-metadata.xml",
//	  "valid_until": "2125-02-20"
//	}
//
// GCP SAML:
//
//	{
//	  "saml_provider_resource_name": "projects/123456789/locations/global/workloadIdentityPools/mciam-pool/providers/keycloak-saml",
//	  "sso_service_location": "https://devkc.onecloudcon.com/realms/gcp-demo/protocol/saml",
//	  "issuer_url": "https://devkc.onecloudcon.com/realms/gcp-demo",
//	  "audience": "//iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/mciam-pool/providers/keycloak-saml",
//	  "metadata_url": "https://devkc.onecloudcon.com/realms/gcp-demo/protocol/saml/descriptor"
//	}
//
// Azure SAML:
//
//	{
//	  "application_id": "12345678-1234-1234-1234-123456789012",
//	  "tenant_id": "87654321-4321-4321-4321-210987654321",
//	  "sso_service_location": "https://devkc.onecloudcon.com/realms/azure-demo/protocol/saml",
//	  "issuer_url": "https://devkc.onecloudcon.com/realms/azure-demo",
//	  "reply_url": "https://login.microsoftonline.com/login/saml2",
//	  "metadata_url": "https://login.microsoftonline.com/TENANT_ID/federationmetadata/2007-06/federationmetadata.xml",
//	  "federated_credential_id": "credential-12345"
//	}
//
// SECRET_KEY Config 예시:
//
//	{
//	  "access_key_id": "AKIAIOSFODNN7EXAMPLE",
//	  "secret_access_key": "encrypted_value",
//	  "encrypted": "true"
//	}

// GetOidcProviderArn OIDC Provider ARN 반환
func (c *CspIdpConfig) GetOidcProviderArn() string {
	if c.Config == nil {
		return ""
	}
	return c.Config["oidc_provider_arn"]
}

// GetAudience OIDC Audience 반환
func (c *CspIdpConfig) GetAudience() string {
	if c.Config == nil {
		return ""
	}
	return c.Config["audience"]
}

// GetStsEndpoint STS Endpoint 반환
func (c *CspIdpConfig) GetStsEndpoint() string {
	if c.Config == nil {
		return ""
	}
	return c.Config["sts_endpoint"]
}

// GetSamlProviderArn SAML Provider ARN 반환 (AWS)
func (c *CspIdpConfig) GetSamlProviderArn() string {
	if c.Config == nil {
		return ""
	}
	return c.Config["saml_provider_arn"]
}

// ========== Common SAML Fields (All CSPs) ==========

// GetSsoServiceLocation SSO Service Location 반환 (SAML)
func (c *CspIdpConfig) GetSsoServiceLocation() string {
	if c.Config == nil {
		return ""
	}
	// Prefer sso_service_location, fallback to assertion_endpoint for backward compatibility
	if loc := c.Config["sso_service_location"]; loc != "" {
		return loc
	}
	return c.Config["assertion_endpoint"]
}

// GetIssuerUrl Issuer URL 반환 (SAML)
func (c *CspIdpConfig) GetIssuerUrl() string {
	if c.Config == nil {
		return ""
	}
	return c.Config["issuer_url"]
}

// GetMetadataUrl SAML Metadata Document URL 반환
func (c *CspIdpConfig) GetMetadataUrl() string {
	if c.Config == nil {
		return ""
	}
	return c.Config["metadata_url"]
}

// ========== AWS-Specific SAML Fields ==========

// GetSigninUrl AWS SAML Sign-in URL (ACS endpoint) 반환
func (c *CspIdpConfig) GetSigninUrl() string {
	if c.Config == nil {
		return ""
	}
	return c.Config["signin_url"]
}

// GetValidUntil SAML Assertion Valid Until Date 반환 (AWS)
func (c *CspIdpConfig) GetValidUntil() string {
	if c.Config == nil {
		return ""
	}
	return c.Config["valid_until"]
}

// ========== GCP-Specific SAML Fields ==========

// GetSamlProviderResourceName GCP SAML Provider Resource Name 반환
func (c *CspIdpConfig) GetSamlProviderResourceName() string {
	if c.Config == nil {
		return ""
	}
	return c.Config["saml_provider_resource_name"]
}

// ========== Azure-Specific SAML Fields ==========

// GetApplicationId Azure AD Application ID 반환
func (c *CspIdpConfig) GetApplicationId() string {
	if c.Config == nil {
		return ""
	}
	return c.Config["application_id"]
}

// GetTenantId Azure AD Tenant ID 반환
func (c *CspIdpConfig) GetTenantId() string {
	if c.Config == nil {
		return ""
	}
	return c.Config["tenant_id"]
}

// GetReplyUrl Azure AD Reply URL 반환
func (c *CspIdpConfig) GetReplyUrl() string {
	if c.Config == nil {
		return ""
	}
	return c.Config["reply_url"]
}

// GetFederatedCredentialId Azure Federated Credential ID 반환
func (c *CspIdpConfig) GetFederatedCredentialId() string {
	if c.Config == nil {
		return ""
	}
	return c.Config["federated_credential_id"]
}

// GetAccessKeyID Secret Key Access Key ID 반환
func (c *CspIdpConfig) GetAccessKeyID() string {
	if c.Config == nil {
		return ""
	}
	return c.Config["access_key_id"]
}

// GetSecretAccessKey Secret Key Secret Access Key 반환 (암호화된 값)
func (c *CspIdpConfig) GetSecretAccessKey() string {
	if c.Config == nil {
		return ""
	}
	return c.Config["secret_access_key"]
}

// IsEncrypted Secret Key 암호화 여부 확인
func (c *CspIdpConfig) IsEncrypted() bool {
	if c.Config == nil {
		return false
	}
	return c.Config["encrypted"] == "true"
}

// IsOIDC OIDC 인증 방식인지 확인
func (c *CspIdpConfig) IsOIDC() bool {
	return c.AuthMethod == AuthMethodOIDC
}

// IsSAML SAML 인증 방식인지 확인
func (c *CspIdpConfig) IsSAML() bool {
	return c.AuthMethod == AuthMethodSAML
}

// IsSecretKey Secret Key 인증 방식인지 확인
func (c *CspIdpConfig) IsSecretKey() bool {
	return c.AuthMethod == AuthMethodSecretKey
}

// CspIdpConfigFilter CSP IDP 설정 조회 필터
type CspIdpConfigFilter struct {
	CspAccountID *uint          `json:"csp_account_id,omitempty"`
	AuthMethod   AuthMethodType `json:"auth_method,omitempty"`
	IsActive     *bool          `json:"is_active,omitempty"`
	Name         string         `json:"name,omitempty"`
}

// CreateCspIdpConfigRequest CSP IDP 설정 생성 요청
type CreateCspIdpConfigRequest struct {
	Name         string            `json:"name" binding:"required"`
	CspAccountID uint              `json:"csp_account_id" binding:"required"`
	AuthMethod   AuthMethodType    `json:"auth_method" binding:"required,oneof=OIDC SAML SECRET_KEY"`
	Config       map[string]string `json:"config" binding:"required"`
	Description  string            `json:"description"`
}

// UpdateCspIdpConfigRequest CSP IDP 설정 수정 요청
type UpdateCspIdpConfigRequest struct {
	Name        string            `json:"name"`
	Config      map[string]string `json:"config"`
	IsActive    *bool             `json:"is_active"`
	Description string            `json:"description"`
}
