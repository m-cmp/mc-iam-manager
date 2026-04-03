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

// GcpCredentialService defines operations for obtaining GCP temporary credentials
// via Workload Identity Federation (WIF).
type GcpCredentialService interface {
	ExchangeTokenAndImpersonate(
		ctx context.Context,
		wifProviderResourceName string,
		serviceAccountEmail string,
		webIdentityToken string,
	) (*model.CspCredentialResponse, error)
}

type gcpCredentialService struct{}

// NewGcpCredentialService creates a new GcpCredentialService.
func NewGcpCredentialService() GcpCredentialService {
	return &gcpCredentialService{}
}

// gcpStsResponse represents the response from GCP STS token exchange.
type gcpStsResponse struct {
	AccessToken     string `json:"access_token"`
	IssuedTokenType string `json:"issued_token_type"`
	TokenType       string `json:"token_type"`
	ExpiresIn       int    `json:"expires_in"`
}

// gcpGenerateAccessTokenRequest is the request body for SA impersonation.
type gcpGenerateAccessTokenRequest struct {
	Scope    []string `json:"scope"`
	Lifetime string   `json:"lifetime,omitempty"`
}

// gcpGenerateAccessTokenResponse is the response from SA impersonation.
type gcpGenerateAccessTokenResponse struct {
	AccessToken string `json:"accessToken"`
	ExpireTime  string `json:"expireTime"`
}

// ExchangeTokenAndImpersonate exchanges a Keycloak OIDC token for a GCP OAuth2
// access token using the two-step Workload Identity Federation flow:
//  1. STS token exchange: Keycloak JWT → GCP federated access token
//  2. Service Account impersonation: federated token → SA access token
func (s *gcpCredentialService) ExchangeTokenAndImpersonate(
	ctx context.Context,
	wifProviderResourceName string,
	serviceAccountEmail string,
	webIdentityToken string,
) (*model.CspCredentialResponse, error) {
	log.Printf("[GCP_CREDENTIAL] Starting WIF token exchange for SA: %s", serviceAccountEmail)

	// Step 1: Exchange Keycloak JWT for GCP federated access token via GCP STS
	federatedToken, err := s.exchangeToken(ctx, wifProviderResourceName, webIdentityToken)
	if err != nil {
		log.Printf("[GCP_CREDENTIAL] STS token exchange failed: %v", err)
		return nil, fmt.Errorf("GCP STS token exchange failed: %w", err)
	}
	log.Printf("[GCP_CREDENTIAL] STS token exchange succeeded")

	// Step 2: Use federated token to impersonate Service Account
	saToken, expireTime, err := s.generateAccessToken(ctx, serviceAccountEmail, federatedToken)
	if err != nil {
		log.Printf("[GCP_CREDENTIAL] SA impersonation failed: %v", err)
		return nil, fmt.Errorf("GCP SA impersonation failed: %w", err)
	}
	log.Printf("[GCP_CREDENTIAL] SA impersonation succeeded, expiry: %s", expireTime)

	return &model.CspCredentialResponse{
		CspType:     "gcp",
		AccessToken: saToken,
		TokenType:   "Bearer",
		Expiration:  expireTime,
	}, nil
}

// exchangeToken calls GCP STS to exchange a Keycloak JWT for a federated access token.
func (s *gcpCredentialService) exchangeToken(ctx context.Context, audience, webIdentityToken string) (string, error) {
	stsURL := "https://sts.googleapis.com/v1/token"

	formData := url.Values{}
	formData.Set("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange")
	formData.Set("subject_token", webIdentityToken)
	formData.Set("subject_token_type", "urn:ietf:params:oauth:token-type:jwt")
	formData.Set("requested_token_type", "urn:ietf:params:oauth:token-type:access_token")
	formData.Set("audience", audience)
	formData.Set("scope", "https://www.googleapis.com/auth/cloud-platform")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, stsURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create STS request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("STS request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read STS response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GCP STS returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var stsResp gcpStsResponse
	if err := json.Unmarshal(body, &stsResp); err != nil {
		return "", fmt.Errorf("failed to parse STS response: %w", err)
	}

	if stsResp.AccessToken == "" {
		return "", fmt.Errorf("GCP STS returned empty access_token")
	}

	return stsResp.AccessToken, nil
}

// generateAccessToken calls GCP IAM Credentials API to impersonate a Service Account.
func (s *gcpCredentialService) generateAccessToken(ctx context.Context, serviceAccountEmail, federatedToken string) (string, time.Time, error) {
	iamURL := fmt.Sprintf(
		"https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/%s:generateAccessToken",
		serviceAccountEmail,
	)

	reqBody := gcpGenerateAccessTokenRequest{
		Scope: []string{"https://www.googleapis.com/auth/cloud-platform"},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to marshal generateAccessToken request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, iamURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to create generateAccessToken request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+federatedToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generateAccessToken request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to read generateAccessToken response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, fmt.Errorf("GCP IAM Credentials returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp gcpGenerateAccessTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to parse generateAccessToken response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", time.Time{}, fmt.Errorf("GCP IAM Credentials returned empty accessToken")
	}

	expireTime, err := time.Parse(time.RFC3339, tokenResp.ExpireTime)
	if err != nil {
		log.Printf("[GCP_CREDENTIAL] Warning: failed to parse expireTime %q: %v, using 1h from now", tokenResp.ExpireTime, err)
		expireTime = time.Now().Add(time.Hour)
	}

	return tokenResp.AccessToken, expireTime, nil
}
