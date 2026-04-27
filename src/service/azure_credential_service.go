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

// AzureCredentialService defines operations for obtaining Azure temporary
// access tokens via Workload Identity Federation (Federated Credential).
type AzureCredentialService interface {
	GetTokenByFederatedCredential(
		ctx context.Context,
		tenantID string,
		clientID string,
		keycloakJWT string,
	) (*model.CspCredentialResponse, error)
}

type azureCredentialService struct{}

// NewAzureCredentialService creates a new AzureCredentialService.
func NewAzureCredentialService() AzureCredentialService {
	return &azureCredentialService{}
}

// azureTokenResponse represents the OAuth2 token response from Azure AD.
type azureTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// azureErrorResponse represents an OAuth2 error response from Azure AD.
type azureErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// GetTokenByFederatedCredential exchanges a Keycloak OIDC JWT for an Azure
// access token using the OAuth2 Federated Credential (Workload Identity Federation) flow.
//
// tenantID:    Azure AD tenant ID
// clientID:    Azure AD application (client) ID registered with a federated credential
// keycloakJWT: OIDC ID token / access token issued by Keycloak
func (s *azureCredentialService) GetTokenByFederatedCredential(
	ctx context.Context,
	tenantID string,
	clientID string,
	keycloakJWT string,
) (*model.CspCredentialResponse, error) {
	log.Printf("[AZURE_CREDENTIAL] GetTokenByFederatedCredential - tenantID: %s, clientID: %s", tenantID, clientID)

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)

	formData := url.Values{}
	formData.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	formData.Set("client_id", clientID)
	formData.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	formData.Set("client_assertion", keycloakJWT)
	formData.Set("scope", "https://management.azure.com/.default")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Azure token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Azure token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp azureErrorResponse
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.Error != "" {
			return nil, fmt.Errorf("Azure token error [%s]: %s", errResp.Error, errResp.ErrorDescription)
		}
		return nil, fmt.Errorf("Azure token endpoint returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp azureTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse Azure token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("Azure token endpoint returned empty access_token")
	}

	expiration := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	log.Printf("[AZURE_CREDENTIAL] GetTokenByFederatedCredential succeeded, expires in %ds", tokenResp.ExpiresIn)

	return &model.CspCredentialResponse{
		CspType:     "azure",
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
		Expiration:  expiration,
	}, nil
}
