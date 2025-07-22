package model

import "time"

// CspCredentialRequest CSP 임시 자격 증명 발급 요청 모델
type CspCredentialRequest struct {
	WorkspaceID string `json:"workspaceId"` // 대상 워크스페이스 ID
	CspType     string `json:"cspType"`     // 대상 CSP 타입
	Region      string `json:"region"`      // AWS 리전 (선택적)
}

// CspCredentialResponse CSP 임시 자격 증명 발급 응답 모델 (현재는 AWS STS 기준)
// TODO: 추후 다른 CSP 지원 시 구조 확장 또는 인터페이스 사용 고려
type CspCredentialResponse struct {
	CspType         string    `json:"cspType"`          // e.g., "aws"
	AccessKeyId     string    `json:"accessKeyId"`      // AWS Access Key ID
	SecretAccessKey string    `json:"secretAccessKey"`  // AWS Secret Access Key
	SessionToken    string    `json:"sessionToken"`     // AWS Session Token
	Expiration      time.Time `json:"expiration"`       // Expiration time
	Region          string    `json:"region,omitempty"` // Optional: AWS Region
}

// TempCredential 임시 자격 증명 관리 테이블 모델
type TempCredential struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	Provider        string    `json:"provider" gorm:"not null"`        // aws, gcp, azure, nhn
	AuthType        string    `json:"authType" gorm:"not null"`        // oidc, saml, secret_key
	AccessKeyId     string    `json:"accessKeyId" gorm:"not null"`     // Access Key ID
	SecretAccessKey string    `json:"secretAccessKey" gorm:"not null"` // Secret Access Key
	SessionToken    string    `json:"sessionToken"`                    // Session Token (OIDC/SAML용)
	Region          string    `json:"region" gorm:"not null"`          // 리전
	IssuedAt        time.Time `json:"issuedAt" gorm:"not null"`        // 발급 시간
	ExpiresAt       time.Time `json:"expiresAt" gorm:"not null"`       // 만료 시간
	IsActive        bool      `json:"isActive" gorm:"default:true"`    // 활성 상태
	IssuedBy        string    `json:"issuedBy" gorm:"not null"`        // 발급 요청자 (Keycloak User ID)
	RoleMasterID    *uint     `json:"roleMasterId"`                    // RoleMaster ID (선택적)
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// TableName 테이블명 지정
func (TempCredential) TableName() string {
	return "mcmp_temp_credentials"
}

// IsExpired 만료 여부 확인
func (tc *TempCredential) IsExpired() bool {
	return time.Now().After(tc.ExpiresAt)
}

// IsValid 유효성 확인 (활성 상태이고 만료되지 않음)
func (tc *TempCredential) IsValid() bool {
	return tc.IsActive && !tc.IsExpired()
}

// GetExpirationTime 만료까지 남은 시간
func (tc *TempCredential) GetExpirationTime() time.Duration {
	return tc.ExpiresAt.Sub(time.Now())
}
