package model

import "time"

// CspCredentialsRequest 임시 자격 증명 발급 요청 모델
type CspCredentialsRequest struct {
	WorkspaceID string `json:"workspaceId" validate:"required"`                 // 대상 워크스페이스 ID
	CspType     string `json:"cspType" validate:"required,oneof=aws gcp azure"` // 대상 CSP 타입
}

// CspCredentialsResponse 임시 자격 증명 발급 응답 모델 (현재는 AWS STS 기준)
// TODO: 추후 다른 CSP 지원 시 구조 확장 또는 인터페이스 사용 고려
type CspCredentialsResponse struct {
	CspType         string    `json:"cspType"`          // e.g., "aws"
	AccessKeyId     string    `json:"accessKeyId"`      // AWS Access Key ID
	SecretAccessKey string    `json:"secretAccessKey"`  // AWS Secret Access Key
	SessionToken    string    `json:"sessionToken"`     // AWS Session Token
	Expiration      time.Time `json:"expiration"`       // Expiration time
	Region          string    `json:"region,omitempty"` // Optional: AWS Region
}
