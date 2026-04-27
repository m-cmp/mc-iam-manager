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

// IbmCredentialService defines operations for obtaining IBM Cloud IAM
// access tokens via Trusted Profile (CR token exchange).
type IbmCredentialService interface {
	GetTokenByTrustedProfile(
		ctx context.Context,
		profileID string,
		crToken string,
	) (*model.CspCredentialResponse, error)
}

type ibmCredentialService struct{}

// NewIbmCredentialService creates a new IbmCredentialService.
func NewIbmCredentialService() IbmCredentialService {
	return &ibmCredentialService{}
}

// ibmTokenResponse represents the IBM IAM token endpoint response.
type ibmTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Expiration  int64  `json:"expiration"`
}

// ibmErrorResponse represents an IBM IAM token endpoint error response.
type ibmErrorResponse struct {
	ErrorCode        string `json:"errorCode"`
	ErrorMessage     string `json:"errorMessage"`
	Context          string `json:"context"`
}

const ibmIamTokenURL = "https://iam.cloud.ibm.com/identity/token"

// GetTokenByTrustedProfile exchanges a Keycloak OIDC JWT for an IBM Cloud IAM
// access token using the Trusted Profile (CR token) flow.
//
// profileID: IBM Cloud Trusted Profile ID (e.g., Profile-xxxx-yyyy-zzzz)
// crToken:   OIDC access token / ID token from Keycloak (used as CR token)
func (s *ibmCredentialService) GetTokenByTrustedProfile(
	ctx context.Context,
	profileID string,
	crToken string,
) (*model.CspCredentialResponse, error) {
	log.Printf("[IBM_CREDENTIAL] GetTokenByTrustedProfile - profileID: %s", profileID)

	formData := url.Values{}
	formData.Set("grant_type", "urn:ibm:params:oauth:grant-type:cr-token")
	formData.Set("cr_token", crToken)
	formData.Set("profile_id", profileID)
	formData.Set("response_type", "cloud_iam")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ibmIamTokenURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create IBM IAM token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("IBM IAM token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read IBM IAM token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ibmErrorResponse
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.ErrorCode != "" {
			return nil, fmt.Errorf("IBM IAM error [%s]: %s", errResp.ErrorCode, errResp.ErrorMessage)
		}
		return nil, fmt.Errorf("IBM IAM token endpoint returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp ibmTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse IBM IAM token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("IBM IAM token endpoint returned empty access_token")
	}

	var expiration time.Time
	if tokenResp.Expiration != 0 {
		expiration = time.Unix(tokenResp.Expiration, 0)
	} else {
		expiration = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}
	log.Printf("[IBM_CREDENTIAL] GetTokenByTrustedProfile succeeded, expires: %s", expiration)

	return &model.CspCredentialResponse{
		CspType:     "ibm",
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
		Expiration:  expiration,
	}, nil
}
