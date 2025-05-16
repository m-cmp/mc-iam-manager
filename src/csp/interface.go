package csp

import (
	"context"
	"time"
)

// CSPType 클라우드 서비스 제공자 타입
type CSPType string

const (
	CSPTypeAWS   CSPType = "aws"
	CSPTypeGCP   CSPType = "gcp"
	CSPTypeAzure CSPType = "azure"
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
