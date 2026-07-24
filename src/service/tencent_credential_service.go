package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/m-cmp/mc-iam-manager/model"
)

// TencentCredentialService defines operations for obtaining Tencent Cloud
// temporary credentials via CAM AssumeRoleWithSAML / AssumeRoleWithWebIdentity.
type TencentCredentialService interface {
	AssumeRoleWithSAML(
		ctx context.Context,
		secretID string,
		secretKey string,
		roleArn string,
		principalArn string,
		samlAssertion string,
		region string,
	) (*model.CspCredentialResponse, error)
	AssumeRoleWithWebIdentity(
		ctx context.Context,
		secretID string,
		secretKey string,
		roleArn string,
		providerId string,
		webIdentityToken string,
		region string,
	) (*model.CspCredentialResponse, error)
}

type tencentCredentialService struct{}

// NewTencentCredentialService creates a new TencentCredentialService.
func NewTencentCredentialService() TencentCredentialService {
	return &tencentCredentialService{}
}

// tencentCredentials represents the Credentials block in Tencent STS response.
type tencentCredentials struct {
	TmpSecretId  string `json:"TmpSecretId"`
	TmpSecretKey string `json:"TmpSecretKey"`
	Token        string `json:"Token"`
}

// tencentStsResponseBody represents the inner Response in Tencent STS response.
type tencentStsResponseBody struct {
	Credentials tencentCredentials `json:"Credentials"`
	ExpiredTime int64              `json:"ExpiredTime"`
	Expiration  string             `json:"Expiration"`
	RequestId   string             `json:"RequestId"`
}

// tencentStsResponse wraps the Tencent API response envelope.
type tencentStsResponse struct {
	Response tencentStsResponseBody `json:"Response"`
}

// tencentErrorBody represents the Error block in Tencent API error response.
type tencentErrorBody struct {
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

// tencentErrorResponse represents a Tencent API error response.
type tencentErrorResponse struct {
	Response struct {
		Error     tencentErrorBody `json:"Error"`
		RequestId string           `json:"RequestId"`
	} `json:"Response"`
}

const (
	tencentStsEndpoint = "https://sts.tencentcloudapi.com/"
	tencentStsVersion  = "2018-08-13"
	tencentStsService  = "sts"
	tencentStsHost     = "sts.tencentcloudapi.com"
)

// AssumeRoleWithSAML calls Tencent Cloud CAM STS to exchange a SAML assertion
// for temporary credentials (TmpSecretId + TmpSecretKey + Token).
//
// The request is signed with TC3-HMAC-SHA256 using the provided secretID/secretKey.
//
// secretID:      Tencent Cloud API Secret ID (for signing; sub-account with STS permission)
// secretKey:     Tencent Cloud API Secret Key (for signing)
// roleArn:       Tencent CAM Role ARN (e.g., qcs::cam::uin/123:roleName/myRole)
// principalArn:  Tencent CAM SAML provider ARN (e.g., qcs::cam::uin/123:saml-provider/myIdP)
// samlAssertion: Base64-encoded SAML2 assertion from Keycloak
// region:        Tencent Cloud region (used in response only; STS itself is global)
func (s *tencentCredentialService) AssumeRoleWithSAML(
	ctx context.Context,
	secretID string,
	secretKey string,
	roleArn string,
	principalArn string,
	samlAssertion string,
	region string,
) (*model.CspCredentialResponse, error) {
	log.Printf("[TENCENT_CREDENTIAL] AssumeRoleWithSAML - RoleArn: %s, PrincipalArn: %s", roleArn, principalArn)

	sessionName := fmt.Sprintf("mciam-%d", time.Now().Unix())

	payload := map[string]string{
		"RoleArn":         roleArn,
		"PrincipalArn":    principalArn,
		"SAMLAssertion":   samlAssertion,
		"RoleSessionName": sessionName,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Tencent STS request: %w", err)
	}

	now := time.Now().UTC()
	timestamp := fmt.Sprintf("%d", now.Unix())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tencentStsEndpoint, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Tencent STS request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", tencentStsHost)
	req.Header.Set("X-TC-Action", "AssumeRoleWithSAML")
	req.Header.Set("X-TC-Version", tencentStsVersion)
	req.Header.Set("X-TC-Timestamp", timestamp)
	req.Header.Set("X-TC-Region", region)
	// AssumeRoleWithSAML은 AWS/Alibaba/GCP의 SAML/OIDC federation entry point와 마찬가지로
	// 사전 자격증명 없이 호출 가능한 진입점이라 TC3-HMAC-SHA256 서명을 요구하지 않는다.
	// Tencent 문서상 Authorization 헤더는 리터럴 문자열 "SKIP"이어야 한다 — 서명을 보내면
	// "Must be SKIP" 오류로 거부된다(실 API 호출로 확인, 039 Phase 2).
	req.Header.Set("Authorization", "SKIP")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Tencent STS request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Tencent STS response: %w", err)
	}

	// Check for Tencent API error in response body (Tencent always returns 200 for API-level errors)
	var errResp tencentErrorResponse
	if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.Response.Error.Code != "" {
		return nil, fmt.Errorf("Tencent STS error [%s]: %s", errResp.Response.Error.Code, errResp.Response.Error.Message)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Tencent STS returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var stsResp tencentStsResponse
	if err := json.Unmarshal(body, &stsResp); err != nil {
		return nil, fmt.Errorf("failed to parse Tencent STS response: %w", err)
	}

	creds := stsResp.Response.Credentials
	if creds.TmpSecretId == "" || creds.TmpSecretKey == "" {
		return nil, fmt.Errorf("Tencent STS returned empty credentials")
	}

	expiration := time.Unix(stsResp.Response.ExpiredTime, 0)
	if stsResp.Response.ExpiredTime == 0 {
		log.Printf("[TENCENT_CREDENTIAL] Warning: ExpiredTime is 0, using 1h from now")
		expiration = time.Now().Add(time.Hour)
	}

	log.Printf("[TENCENT_CREDENTIAL] AssumeRoleWithSAML succeeded, Expiration: %s", expiration)

	return &model.CspCredentialResponse{
		CspType:         "tencent",
		AccessKeyId:     creds.TmpSecretId,
		SecretAccessKey: creds.TmpSecretKey,
		SessionToken:    creds.Token,
		Expiration:      expiration,
		Region:          region,
	}, nil
}

// AssumeRoleWithWebIdentity calls Tencent Cloud CAM STS to exchange an OIDC
// Web Identity token (e.g. Keycloak ID Token) for temporary credentials
// (TmpSecretId + TmpSecretKey + Token).
//
// secretID:         Tencent Cloud API Secret ID (SAML 경로와의 일관성을 위해 유지 — SKIP 인증이라 STS
//                    호출 자체에는 사용되지 않지만, 설정 검증 용도로 SAML과 동일하게 요구한다)
// secretKey:         Tencent Cloud API Secret Key (위와 동일한 이유로 유지)
// roleArn:           Tencent CAM Role ARN (e.g., qcs::cam::uin/123:roleName/myRole)
// providerId:        CAM에 등록한 OIDC Provider의 Name(전체 ARN이 아님). 문서 예시의 리터럴
//                    "OIDC"를 그대로 보내면 "identity no exist"로 항상 실패함을 실 API로 확인함.
// webIdentityToken:  Keycloak이 발급한 OIDC ID Token (AccessToken이 아님 — aud가 클라이언트 ID인 ID Token 필요)
// region:            Tencent Cloud region (X-TC-Region 헤더로 전달; STS 자체는 글로벌 엔드포인트)
func (s *tencentCredentialService) AssumeRoleWithWebIdentity(
	ctx context.Context,
	secretID string,
	secretKey string,
	roleArn string,
	providerId string,
	webIdentityToken string,
	region string,
) (*model.CspCredentialResponse, error) {
	log.Printf("[TENCENT_CREDENTIAL] AssumeRoleWithWebIdentity - RoleArn: %s, ProviderId: %s", roleArn, providerId)

	sessionName := fmt.Sprintf("mciam-%d", time.Now().Unix())

	payload := map[string]string{
		"RoleArn":          roleArn,
		"ProviderId":       providerId,
		"WebIdentityToken": webIdentityToken,
		"RoleSessionName":  sessionName,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Tencent STS request: %w", err)
	}

	now := time.Now().UTC()
	timestamp := fmt.Sprintf("%d", now.Unix())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tencentStsEndpoint, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Tencent STS request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", tencentStsHost)
	req.Header.Set("X-TC-Action", "AssumeRoleWithWebIdentity")
	req.Header.Set("X-TC-Version", tencentStsVersion)
	req.Header.Set("X-TC-Timestamp", timestamp)
	req.Header.Set("X-TC-Region", region)
	// AssumeRoleWithSAML과 동일한 federation entry point 계열이므로 TC3-HMAC-SHA256 서명을 요구하지
	// 않을 것으로 예상된다 — Authorization 헤더는 리터럴 문자열 "SKIP" (039 Phase 2에서 SAML로 실 API
	// 확인된 내용을 준용; OIDC 자체는 아직 실 API로 검증되지 않았으니 인프라 검증 단계에서 재확인 필요).
	req.Header.Set("Authorization", "SKIP")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Tencent STS request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Tencent STS response: %w", err)
	}

	// Check for Tencent API error in response body (Tencent always returns 200 for API-level errors)
	var errResp tencentErrorResponse
	if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.Response.Error.Code != "" {
		return nil, fmt.Errorf("Tencent STS error [%s]: %s", errResp.Response.Error.Code, errResp.Response.Error.Message)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Tencent STS returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var stsResp tencentStsResponse
	if err := json.Unmarshal(body, &stsResp); err != nil {
		return nil, fmt.Errorf("failed to parse Tencent STS response: %w", err)
	}

	creds := stsResp.Response.Credentials
	if creds.TmpSecretId == "" || creds.TmpSecretKey == "" {
		return nil, fmt.Errorf("Tencent STS returned empty credentials")
	}

	expiration := time.Unix(stsResp.Response.ExpiredTime, 0)
	if stsResp.Response.ExpiredTime == 0 {
		log.Printf("[TENCENT_CREDENTIAL] Warning: ExpiredTime is 0, using 1h from now")
		expiration = time.Now().Add(time.Hour)
	}

	log.Printf("[TENCENT_CREDENTIAL] AssumeRoleWithWebIdentity succeeded, Expiration: %s", expiration)

	return &model.CspCredentialResponse{
		CspType:         "tencent",
		AccessKeyId:     creds.TmpSecretId,
		SecretAccessKey: creds.TmpSecretKey,
		SessionToken:    creds.Token,
		Expiration:      expiration,
		Region:          region,
	}, nil
}

// buildTencentTC3Auth constructs the TC3-HMAC-SHA256 Authorization header.
func buildTencentTC3Auth(secretID, secretKey, date, timestamp, payload string) (string, error) {
	// Step 1: Build canonical request
	httpMethod := "POST"
	canonicalURI := "/"
	canonicalQueryString := ""
	canonicalHeaders := fmt.Sprintf("content-type:application/json\nhost:%s\nx-tc-action:%s\n",
		tencentStsHost, strings.ToLower("AssumeRoleWithSAML"))
	signedHeaders := "content-type;host;x-tc-action"
	hashedPayload := sha256hex(payload)
	canonicalRequest := strings.Join([]string{
		httpMethod, canonicalURI, canonicalQueryString,
		canonicalHeaders, signedHeaders, hashedPayload,
	}, "\n")

	// Step 2: Build string to sign
	algorithm := "TC3-HMAC-SHA256"
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, tencentStsService)
	stringToSign := strings.Join([]string{
		algorithm, timestamp, credentialScope, sha256hex(canonicalRequest),
	}, "\n")

	// Step 3: Derive signing key
	secretDate := hmacSHA256([]byte("TC3"+secretKey), date)
	secretService := hmacSHA256(secretDate, tencentStsService)
	secretSigning := hmacSHA256(secretService, "tc3_request")

	// Step 4: Calculate signature
	signature := hex.EncodeToString(hmacSHA256(secretSigning, stringToSign))

	// Step 5: Build Authorization header
	auth := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, secretID, credentialScope, signedHeaders, signature)

	return auth, nil
}

func sha256hex(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}
