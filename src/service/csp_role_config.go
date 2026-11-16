package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"

	mciamConfig "github.com/m-cmp/mc-iam-manager/config"
	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
)

// CspRoleRequestError 요청 값이 잘못되어 처리할 수 없는 경우의 오류 (핸들러는 400으로 응답)
type CspRoleRequestError struct {
	Msg string
}

func (e *CspRoleRequestError) Error() string { return e.Msg }

// IsCspRoleRequestError err 가 CspRoleRequestError 인지 확인한다.
func IsCspRoleRequestError(err error) bool {
	var e *CspRoleRequestError
	return errors.As(err, &e)
}

func cspRoleRequestErrorf(format string, args ...interface{}) error {
	return &CspRoleRequestError{Msg: fmt.Sprintf(format, args...)}
}

// requestAuthMethod 요청의 인증 방식. 미지정 시 OIDC.
func requestAuthMethod(req *model.CreateCspRoleRequest) constants.AuthMethod {
	if req.AuthMethod == "" {
		return constants.AuthMethodOIDC
	}
	return req.AuthMethod
}

// configMatches Config 가 CSP 타입·인증 방식에 맞는 활성 설정인지 확인한다.
func configMatches(cfg *model.CspIdpConfig, cspType string, authMethod constants.AuthMethod) bool {
	return cfg != nil && cfg.IsActive && cfg.CspAccount != nil &&
		cfg.CspAccount.CspType == cspType && string(cfg.AuthMethod) == string(authMethod)
}

// resolveCspIdpConfig CspRole 이 소속될 IdP Config 를 확정한다 (CSP 계정 → IdP Config → CspRole).
//   - cspIdpConfigId 지정: 존재·활성·CSP 타입·인증 방식 일치를 검증하고, 아니면 요청 오류
//   - 미지정: 같은 CSP 타입·인증 방식의 활성 Config 가 정확히 1개면 그것을 쓰고 WARN (전환 기간 하위호환)
//     AWS 는 0개 또는 2개 이상이면 요청 오류. 그 외 CSP 는 소속 없이 진행(기존 동작 유지)하고 nil 을 돌려준다.
func (s *CspRoleService) resolveCspIdpConfig(req *model.CreateCspRoleRequest) (*model.CspIdpConfig, error) {
	authMethod := requestAuthMethod(req)

	if req.CspIdpConfigID != nil {
		cfg, err := s.cspIdpConfigRepo.GetByID(*req.CspIdpConfigID)
		if err != nil {
			return nil, err
		}
		if cfg == nil {
			return nil, cspRoleRequestErrorf("cspIdpConfigId %d not found", *req.CspIdpConfigID)
		}
		if !configMatches(cfg, req.CspType, authMethod) {
			cspType := ""
			if cfg.CspAccount != nil {
				cspType = cfg.CspAccount.CspType
			}
			return nil, cspRoleRequestErrorf("cspIdpConfigId %d does not match request (config: active=%t cspType=%s authMethod=%s, request: cspType=%s authMethod=%s)",
				cfg.ID, cfg.IsActive, cspType, cfg.AuthMethod, req.CspType, authMethod)
		}
		return cfg, nil
	}

	configs, err := s.cspIdpConfigRepo.GetActiveConfigs()
	if err != nil {
		return nil, err
	}
	var candidates []*model.CspIdpConfig
	for _, cfg := range configs {
		if configMatches(cfg, req.CspType, authMethod) {
			candidates = append(candidates, cfg)
		}
	}

	if len(candidates) == 1 {
		log.Printf("[WARN] CSP role %s: cspIdpConfigId not specified, using the only active %s %s IdP config (id=%d)",
			req.CspRoleName, req.CspType, authMethod, candidates[0].ID)
		return candidates[0], nil
	}
	if req.CspType != "aws" {
		log.Printf("[WARN] CSP role %s: cspIdpConfigId not specified and %d active %s %s IdP configs found — saved without IdP config",
			req.CspRoleName, len(candidates), req.CspType, authMethod)
		return nil, nil
	}
	if len(candidates) == 0 {
		return nil, cspRoleRequestErrorf("no active %s %s IdP config found: register an IdP config first or specify cspIdpConfigId", req.CspType, authMethod)
	}
	return nil, cspRoleRequestErrorf("%d active %s %s IdP configs found: specify cspIdpConfigId", len(candidates), req.CspType, authMethod)
}

// applyCspIdpConfig 소속 Config 와 그 계정을 CspRole 에 기록한다.
func applyCspIdpConfig(role *model.CspRole, cfg *model.CspIdpConfig) {
	if cfg == nil {
		return
	}
	cfgID := cfg.ID
	accountID := cfg.CspAccountID
	role.CspIdpConfigID = &cfgID
	role.CspAccountID = &accountID
}

// configIdpIdentifier Config 에 등록된 IdP(Provider) ARN. OIDC 는 oidc_provider_arn, SAML 은 saml_provider_arn.
func configIdpIdentifier(cfg *model.CspIdpConfig) string {
	if cfg == nil {
		return ""
	}
	if cfg.IsSAML() {
		return cfg.GetSamlProviderArn()
	}
	if cfg.IsOIDC() {
		return cfg.GetOidcProviderArn()
	}
	return ""
}

// federatedFromTrustPolicy AWS GetRole 의 AssumeRolePolicyDocument(URL 인코딩된 JSON)에서 Federated principal 을
// 문서 순서대로 모두 꺼낸다.
func federatedFromTrustPolicy(doc string) []string {
	if doc == "" {
		return nil
	}
	if decoded, err := url.QueryUnescape(doc); err == nil {
		doc = decoded
	}
	var policy struct {
		Statement []struct {
			Principal map[string]interface{} `json:"Principal"`
		} `json:"Statement"`
	}
	if err := json.Unmarshal([]byte(doc), &policy); err != nil {
		return nil
	}
	var federated []string
	for _, st := range policy.Statement {
		switch v := st.Principal["Federated"].(type) {
		case string:
			federated = append(federated, v)
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok && s != "" {
					federated = append(federated, s)
				}
			}
		}
	}
	return federated
}

// resolveIdpIdentifier 기존 AWS Role 을 등록할 때 idp_identifier 를 정한다 (IAM-BUG-042).
// 우선순위: 요청 값 → 소속 Config 의 Provider ARN → AWS 실제 신뢰 정책의 첫 Federated.
// 고른 값이 AWS 실제 신뢰 정책의 Federated 어느 것과도 다르면 WARN 을 남긴다.
func resolveIdpIdentifier(roleName, requested string, cfg *model.CspIdpConfig, awsFederated []string) string {
	chosen := requested
	source := "request"
	if chosen == "" {
		chosen = configIdpIdentifier(cfg)
		source = "idp config"
	}
	if chosen == "" && len(awsFederated) > 0 {
		return awsFederated[0]
	}
	if chosen != "" && len(awsFederated) > 0 && !containsString(awsFederated, chosen) {
		log.Printf("[WARN] CSP role %s: idp_identifier from %s (%s) is not trusted by the AWS role trust policy (Federated: %s)",
			roleName, source, chosen, strings.Join(awsFederated, ", "))
	}
	return chosen
}

func containsString(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

var awsAccountIDPattern = regexp.MustCompile(`^[0-9]{12}$`)

// validateAwsOidcTrustInputs IAM 이 신뢰 정책을 직접 만드는 경로(AWS 에 없는 Role 생성)의 입력값을 AWS 호출 전에 검사한다.
// 지금 단계에서는 Role·신뢰 정책을 AWS 에서 직접 만들고 등록하는 것이 운영 전제라, 샘플값(notyet)·내부 호스트로
// 잘못된 Role 이 조용히 만들어지는 것만 막는다.
func validateAwsOidcTrustInputs() error {
	accountID := os.Getenv("MC_IAM_MANAGER_AWS_ACCOUNT_ID")
	if !awsAccountIDPattern.MatchString(accountID) {
		return cspRoleRequestErrorf("MC_IAM_MANAGER_AWS_ACCOUNT_ID (%q) is not a 12-digit AWS account ID: create the role and its trust policy in AWS first, then register it", accountID)
	}
	audience := os.Getenv("MC_IAM_MANAGER_KEYCLOAK_OIDC_CLIENT_ID")
	if audience == "" || audience == "notyet" {
		return cspRoleRequestErrorf("MC_IAM_MANAGER_KEYCLOAK_OIDC_CLIENT_ID (%q) is not set: create the role and its trust policy in AWS first, then register it", audience)
	}
	host := ""
	if mciamConfig.KC != nil {
		host = mciamConfig.KC.ExternalURL
	}
	if isInternalHost(host) {
		return cspRoleRequestErrorf("Keycloak external URL (%q) is an internal host: set MC_IAM_MANAGER_KEYCLOAK_EXTERNAL_URL, or create the role and its trust policy in AWS first, then register it", host)
	}
	return nil
}

// isInternalHost AWS 가 접근할 수 없는 호스트(빈 값, localhost, 점이 없는 컨테이너 이름)인지 확인한다.
func isInternalHost(rawURL string) bool {
	if rawURL == "" {
		return true
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Hostname() == "" {
		return true
	}
	host := u.Hostname()
	return host == "localhost" || strings.HasPrefix(host, "127.") || !strings.Contains(host, ".")
}
