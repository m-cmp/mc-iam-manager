package csp

import (
	"context"
	"time"
)

// RolePolicy IAM 역할 정책
type RolePolicy struct {
	Version   string                `json:"Version"`
	Statement []RolePolicyStatement `json:"Statement"`
}

// RolePolicyStatement IAM 역할 정책 문장
type RolePolicyStatement struct {
	Effect    string     `json:"Effect"`
	Action    []string   `json:"Action"`
	Resource  []string   `json:"Resource"`
	Principal *Principal `json:"Principal,omitempty"`
}

// Principal IAM 역할 정책의 주체
type Principal struct {
	Service []string `json:"Service,omitempty"`
	AWS     []string `json:"AWS,omitempty"`
}

// Role IAM 역할 정보
type Role struct {
	Name        string
	Description string
	Policy      *RolePolicy
	Tags        map[string]string
}

// IAMClient CSP IAM 작업을 위한 인터페이스
type IAMClient interface {
	// CreateRole IAM 역할 생성
	CreateRole(ctx context.Context, role *Role) error

	// DeleteRole IAM 역할 삭제
	DeleteRole(ctx context.Context, roleName string) error

	// GetRole IAM 역할 정보 조회
	GetRole(ctx context.Context, roleName string) (*Role, error)

	// UpdateRole IAM 역할 정보 수정
	UpdateRole(ctx context.Context, role *Role) error

	// AttachRolePolicy IAM 역할에 정책 연결
	AttachRolePolicy(ctx context.Context, roleName string, policyArn string) error

	// DetachRolePolicy IAM 역할에서 정책 분리
	DetachRolePolicy(ctx context.Context, roleName string, policyArn string) error

	// ListRolePolicies IAM 역할에 연결된 정책 목록 조회
	ListRolePolicies(ctx context.Context, roleName string) ([]string, error)

	// GetRolePolicy IAM 역할의 특정 정책 조회
	GetRolePolicy(ctx context.Context, roleName string, policyName string) (*RolePolicy, error)

	// PutRolePolicy IAM 역할에 정책 추가/수정
	PutRolePolicy(ctx context.Context, roleName string, policyName string, policy *RolePolicy) error

	// DeleteRolePolicy IAM 역할에서 정책 삭제
	DeleteRolePolicy(ctx context.Context, roleName string, policyName string) error
}

// IAMClientConfig IAM 클라이언트 설정
type IAMClientConfig struct {
	Region           string
	CredentialsFile  string
	Profile          string
	Timeout          time.Duration
	RoleARN          string
	WebIdentityToken string
	WorkspaceTicket  string
}

// AssumeRoleConfig STS AssumeRole 요청 설정
type AssumeRoleConfig struct {
	RoleArn          string            `json:"role_arn"`
	RoleSessionName  string            `json:"role_session_name"`
	WebIdentityToken string            `json:"web_identity_token,omitempty"`
	DurationSeconds  int32             `json:"duration_seconds,omitempty"`
	ExternalID       string            `json:"external_id,omitempty"`
	Policy           string            `json:"policy,omitempty"`
	PolicyArns       []string          `json:"policy_arns,omitempty"`
	Tags             map[string]string `json:"tags,omitempty"`
}

// CredentialResponse 임시 자격 증명 응답
type CredentialResponse struct {
	AccessKeyID     string    `json:"access_key_id"`
	SecretAccessKey string    `json:"secret_access_key"`
	SessionToken    string    `json:"session_token"`
	Expiration      time.Time `json:"expiration"`
	Provider        string    `json:"provider"`
}

// CredentialService CSP 자격 증명 서비스 인터페이스
type CredentialService interface {
	// AssumeRoleWithWebIdentity OIDC 토큰으로 역할 인수
	AssumeRoleWithWebIdentity(ctx context.Context, config *AssumeRoleConfig) (*CredentialResponse, error)

	// AssumeRoleWithSAML SAML assertion으로 역할 인수
	AssumeRoleWithSAML(ctx context.Context, config *AssumeRoleConfig, samlAssertion string) (*CredentialResponse, error)

	// AssumeRole Secret Key로 역할 인수
	AssumeRole(ctx context.Context, config *AssumeRoleConfig) (*CredentialResponse, error)

	// ValidateCredentials 자격 증명 유효성 검증
	ValidateCredentials(ctx context.Context, credentials map[string]string) error

	// GetCspType CSP 타입 반환
	GetCspType() string
}

// PolicyDefinition 정책 정의
type PolicyDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	PolicyDoc   map[string]interface{} `json:"policy_doc"`
	Path        string                 `json:"path,omitempty"`
	Tags        map[string]string      `json:"tags,omitempty"`
}

// PolicyInfo 정책 정보
type PolicyInfo struct {
	Arn             string                 `json:"arn"`
	Name            string                 `json:"name"`
	PolicyID        string                 `json:"policy_id"`
	Description     string                 `json:"description,omitempty"`
	PolicyDoc       map[string]interface{} `json:"policy_doc,omitempty"`
	Path            string                 `json:"path,omitempty"`
	DefaultVersion  string                 `json:"default_version,omitempty"`
	AttachmentCount int                    `json:"attachment_count"`
	IsAttachable    bool                   `json:"is_attachable"`
	CreateDate      time.Time              `json:"create_date"`
	UpdateDate      time.Time              `json:"update_date"`
}

// PolicyFilter 정책 조회 필터
type PolicyFilter struct {
	Scope           string `json:"scope,omitempty"`           // All, AWS, Local
	PathPrefix      string `json:"path_prefix,omitempty"`     // 경로 접두사
	PolicyUsageType string `json:"policy_usage_type,omitempty"` // PermissionsPolicy, PermissionsBoundary
	OnlyAttached    bool   `json:"only_attached,omitempty"`   // 연결된 정책만
	MaxItems        int    `json:"max_items,omitempty"`
	Marker          string `json:"marker,omitempty"`
}

// PolicyManager CSP 정책 관리 인터페이스
type PolicyManager interface {
	// CreatePolicy 정책 생성
	CreatePolicy(ctx context.Context, policy *PolicyDefinition) (*PolicyInfo, error)

	// GetPolicy 정책 조회
	GetPolicy(ctx context.Context, policyArn string) (*PolicyInfo, error)

	// GetPolicyDocument 정책 문서 조회
	GetPolicyDocument(ctx context.Context, policyArn string, versionId string) (map[string]interface{}, error)

	// UpdatePolicy 정책 수정 (새 버전 생성)
	UpdatePolicy(ctx context.Context, policyArn string, policy *PolicyDefinition) (*PolicyInfo, error)

	// DeletePolicy 정책 삭제
	DeletePolicy(ctx context.Context, policyArn string) error

	// ListPolicies 정책 목록 조회
	ListPolicies(ctx context.Context, filter *PolicyFilter) ([]*PolicyInfo, string, error)

	// AttachPolicyToRole 역할에 정책 연결
	AttachPolicyToRole(ctx context.Context, roleName, policyArn string) error

	// DetachPolicyFromRole 역할에서 정책 분리
	DetachPolicyFromRole(ctx context.Context, roleName, policyArn string) error

	// ListAttachedRolePolicies 역할에 연결된 정책 목록 조회
	ListAttachedRolePolicies(ctx context.Context, roleName string) ([]*PolicyInfo, error)

	// GetCspType CSP 타입 반환
	GetCspType() string
}
