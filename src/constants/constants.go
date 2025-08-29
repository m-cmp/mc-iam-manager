package constants

// IAMManager가 제공하는 Role 종류(platform, workspace, csp)
type IAMRoleType string

// CSPType 클라우드 서비스 제공자 타입
type CSPType string

// IAMManager가 제공하는 Auth Method(OIDC, SAML)
type AuthMethod string

// RoleType 역할 타입 상수
const (
	RoleTypePlatform  IAMRoleType = "platform"  // 플랫폼 역할
	RoleTypeWorkspace IAMRoleType = "workspace" // 워크스페이스 역할
	RoleTypeCSP       IAMRoleType = "csp"       // CSP 역할

	AuthMethodOIDC AuthMethod = "OIDC"
	AuthMethodSAML AuthMethod = "SAML"

	CSPTypeAWS   CSPType = "aws"
	CSPTypeGCP   CSPType = "gcp"
	CSPTypeAzure CSPType = "azure"

	CspRoleNamePrefix = "mciam-" // csp에 role을 추가할 때 접두사를 붙인다.
)
