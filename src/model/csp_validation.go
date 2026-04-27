package model

import "time"

// ValidationStepStatus 단계별 상태값
type ValidationStepStatus string

const (
	ValidationStepOk      ValidationStepStatus = "ok"
	ValidationStepFailed  ValidationStepStatus = "failed"
	ValidationStepSkipped ValidationStepStatus = "skipped"
)

// ValidationStep 단일 검증 단계 결과
type ValidationStep struct {
	Step   int                  `json:"step"`   // 순번 (1부터)
	Name   string               `json:"name"`   // 단계명
	Status ValidationStepStatus `json:"status"` // ok | failed | skipped
	Detail string               `json:"detail"` // 수행 내역 또는 오류 안내 (skipped면 "")
}

// CspValidationRequest 검증 요청
type CspValidationRequest struct {
	WorkspaceID string `json:"workspaceId"` // 대상 워크스페이스 ID
	CspType     string `json:"cspType"`     // CSP 유형 (aws, gcp, ...)
	AuthMethod  string `json:"authMethod"`  // 인증방식 (OIDC, SAML, SECRET_KEY)
}

// CspValidationResponse 검증 응답 — 실패 여부와 무관하게 모든 단계 포함
type CspValidationResponse struct {
	Valid       bool               `json:"valid"`
	CspType     string             `json:"cspType"`
	AuthMethod  string             `json:"authMethod"`
	FailedStep  int                `json:"failedStep"`           // 0=전체성공, N=N번 단계 실패
	Error       string             `json:"error,omitempty"`
	Steps       []ValidationStep   `json:"steps"`                // 항상 전체 단계 포함
	Credentials *CredentialSummary `json:"credentials,omitempty"` // valid=true일 때만
}

// CredentialSummary 발급된 자격증명 요약 (secret은 미포함)
type CredentialSummary struct {
	AccessKeyId string    `json:"accessKeyId"`
	Expiration  time.Time `json:"expiration"`
}
