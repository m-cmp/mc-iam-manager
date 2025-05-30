package model

import "time"

// CspCredentialRequest CSP 임시 자격 증명 발급 요청 모델
type CspCredentialRequest struct {
	WorkspaceID uint   `json:"workspaceId"` // 대상 워크스페이스 ID
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
