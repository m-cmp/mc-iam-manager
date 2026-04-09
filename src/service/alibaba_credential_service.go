package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/m-cmp/mc-iam-manager/model"
)

// AlibabaCredentialService defines operations for obtaining Alibaba Cloud
// temporary credentials via RAM federation.
type AlibabaCredentialService interface {
	AssumeRoleWithSAML(
		ctx context.Context,
		samlProviderArn string,
		roleArn string,
		samlAssertion string,
		region string,
	) (*model.CspCredentialResponse, error)

	AssumeRoleWithOIDC(
		ctx context.Context,
		oidcProviderArn string,
		roleArn string,
		oidcToken string,
		region string,
		audience string,
	) (*model.CspCredentialResponse, error)
}

type alibabaCredentialService struct{}

// NewAlibabaCredentialService creates a new AlibabaCredentialService.
func NewAlibabaCredentialService() AlibabaCredentialService {
	return &alibabaCredentialService{}
}

// alibabaStsCredentials represents the Credentials block in Alibaba STS response.
type alibabaStsCredentials struct {
	AccessKeyId     string `json:"AccessKeyId"`
	AccessKeySecret string `json:"AccessKeySecret"`
	SecurityToken   string `json:"SecurityToken"`
	Expiration      string `json:"Expiration"`
}

// alibabaStsResponse represents the Alibaba STS AssumeRoleWithSAML response.
type alibabaStsResponse struct {
	RequestId   string                `json:"RequestId"`
	Credentials alibabaStsCredentials `json:"Credentials"`
}

// alibabaErrorResponse represents an Alibaba STS error response.
type alibabaErrorResponse struct {
	RequestId string `json:"RequestId"`
	HostId    string `json:"HostId"`
	Code      string `json:"Code"`
	Message   string `json:"Message"`
}

const (
	alibabaStsEndpoint = "https://sts.aliyuncs.com/"
	alibabaStsVersion  = "2015-04-01"
)

// AssumeRoleWithSAML calls Alibaba Cloud STS to exchange a SAML assertion
// for temporary credentials (AccessKeyId + AccessKeySecret + SecurityToken).
//
// samlProviderArn: Alibaba RAM SAML provider ARN (e.g., acs:ram::123456:saml-provider/myProvider)
// roleArn:         Alibaba RAM Role ARN (e.g., acs:ram::123456:role/myRole)
// samlAssertion:   Base64-encoded SAML2 assertion from Keycloak
// region:          Alibaba Cloud region (e.g., cn-hangzhou); used in response only
func (s *alibabaCredentialService) AssumeRoleWithSAML(
	ctx context.Context,
	samlProviderArn string,
	roleArn string,
	samlAssertion string,
	region string,
) (*model.CspCredentialResponse, error) {
	log.Printf("[ALIBABA_CREDENTIAL] AssumeRoleWithSAML - RoleArn: %s, SAMLProviderArn: %s", roleArn, samlProviderArn)

	sessionName := fmt.Sprintf("mciam-%d", time.Now().Unix())
	if len(sessionName) > 32 {
		sessionName = sessionName[:32]
	}

	formData := url.Values{}
	formData.Set("Action", "AssumeRoleWithSAML")
	formData.Set("Version", alibabaStsVersion)
	formData.Set("RoleArn", roleArn)
	formData.Set("SAMLProviderArn", samlProviderArn)
	formData.Set("SAMLAssertion", samlAssertion)
	formData.Set("RoleSessionName", sessionName)
	formData.Set("Format", "JSON")
	formData.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	formData.Set("SignatureNonce", fmt.Sprintf("%d", time.Now().UnixNano()))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, alibabaStsEndpoint, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create Alibaba STS request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Alibaba STS request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Alibaba STS response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp alibabaErrorResponse
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.Code != "" {
			return nil, fmt.Errorf("Alibaba STS error [%s]: %s", errResp.Code, errResp.Message)
		}
		return nil, fmt.Errorf("Alibaba STS returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var stsResp alibabaStsResponse
	if err := json.Unmarshal(body, &stsResp); err != nil {
		return nil, fmt.Errorf("failed to parse Alibaba STS response: %w", err)
	}

	creds := stsResp.Credentials
	if creds.AccessKeyId == "" || creds.AccessKeySecret == "" {
		return nil, fmt.Errorf("Alibaba STS returned empty credentials")
	}

	expiration, err := time.Parse(time.RFC3339, creds.Expiration)
	if err != nil {
		log.Printf("[ALIBABA_CREDENTIAL] Warning: failed to parse Expiration %q: %v, using 1h from now", creds.Expiration, err)
		expiration = time.Now().Add(time.Hour)
	}

	log.Printf("[ALIBABA_CREDENTIAL] AssumeRoleWithSAML succeeded, Expiration: %s", expiration)

	return &model.CspCredentialResponse{
		CspType:         "alibaba",
		AccessKeyId:     creds.AccessKeyId,
		AccessKeySecret: creds.AccessKeySecret,
		SecurityToken:   creds.SecurityToken,
		Expiration:      expiration,
		Region:          region,
	}, nil
}

// AssumeRoleWithOIDC calls Alibaba Cloud STS to exchange an OIDC token
// for temporary credentials (AccessKeyId + AccessKeySecret + SecurityToken).
//
// oidcProviderArn: Alibaba RAM OIDC provider ARN (e.g., acs:ram::123456:oidc-idp/myProvider)
// roleArn:         Alibaba RAM Role ARN (e.g., acs:ram::123456:role/myRole)
// oidcToken:       OIDC ID token (JWT) from Keycloak
// region:          Alibaba Cloud region (e.g., cn-hangzhou); used in response only
func (s *alibabaCredentialService) AssumeRoleWithOIDC(
	ctx context.Context,
	oidcProviderArn string,
	roleArn string,
	oidcToken string,
	region string,
	audience string,
) (*model.CspCredentialResponse, error) {
	log.Printf("[ALIBABA_CREDENTIAL] AssumeRoleWithOIDC - RoleArn: %s, OIDCProviderArn: %s", roleArn, oidcProviderArn)

	sessionName := fmt.Sprintf("mciam-%d", time.Now().Unix())
	if len(sessionName) > 32 {
		sessionName = sessionName[:32]
	}

	formData := url.Values{}
	formData.Set("Action", "AssumeRoleWithOIDC")
	formData.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	formData.Set("Version", alibabaStsVersion)
	formData.Set("RoleArn", roleArn)
	formData.Set("OIDCProviderArn", oidcProviderArn)
	formData.Set("OIDCToken", oidcToken)
	formData.Set("RoleSessionName", sessionName)
	formData.Set("Format", "JSON")
	formData.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	formData.Set("SignatureNonce", fmt.Sprintf("%d", time.Now().UnixNano()))
	if audience != "" {
		formData.Set("OIDCTokenAudience", audience)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, alibabaStsEndpoint, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create Alibaba STS OIDC request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Alibaba STS OIDC request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Alibaba STS OIDC response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp alibabaErrorResponse
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.Code != "" {
			return nil, fmt.Errorf("Alibaba STS OIDC error [%s]: %s", errResp.Code, errResp.Message)
		}
		return nil, fmt.Errorf("Alibaba STS OIDC returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var stsResp alibabaStsResponse
	if err := json.Unmarshal(body, &stsResp); err != nil {
		return nil, fmt.Errorf("failed to parse Alibaba STS OIDC response: %w", err)
	}

	creds := stsResp.Credentials
	if creds.AccessKeyId == "" || creds.AccessKeySecret == "" {
		return nil, fmt.Errorf("Alibaba STS OIDC returned empty credentials")
	}

	expiration, err := time.Parse(time.RFC3339, creds.Expiration)
	if err != nil {
		log.Printf("[ALIBABA_CREDENTIAL] Warning: failed to parse Expiration %q: %v, using 1h from now", creds.Expiration, err)
		expiration = time.Now().Add(time.Hour)
	}

	log.Printf("[ALIBABA_CREDENTIAL] AssumeRoleWithOIDC succeeded, Expiration: %s", expiration)

	return &model.CspCredentialResponse{
		CspType:         "alibaba",
		AccessKeyId:     creds.AccessKeyId,
		AccessKeySecret: creds.AccessKeySecret,
		SecurityToken:   creds.SecurityToken,
		Expiration:      expiration,
		Region:          region,
	}, nil
}
