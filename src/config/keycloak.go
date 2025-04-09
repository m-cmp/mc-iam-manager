package config

import (
	"context"
	"fmt"
	"os"

	"github.com/Nerzal/gocloak/v13"
)

// KeycloakConfig Keycloak 설정
type KeycloakConfig struct {
	ClientID     string
	ClientSecret string
	Realm        string
	Host         string
	Client       *gocloak.GoCloak
}

var KC *KeycloakConfig

// InitKeycloak Keycloak 초기화
func InitKeycloak() error {
	host := os.Getenv("KEYCLOAK_HOST")
	if host == "" {
		return fmt.Errorf("KEYCLOAK_HOST is not set")
	}

	realm := os.Getenv("KEYCLOAK_REALM")
	if realm == "" {
		return fmt.Errorf("KEYCLOAK_REALM is not set")
	}

	clientID := os.Getenv("KEYCLOAK_CLIENT")
	if clientID == "" {
		return fmt.Errorf("KEYCLOAK_CLIENT is not set")
	}

	clientSecret := os.Getenv("KEYCLOAK_CLIENT_SECRET")
	if clientSecret == "" {
		return fmt.Errorf("KEYCLOAK_CLIENT_SECRET is not set")
	}

	client := gocloak.NewClient(host)

	KC = &KeycloakConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Realm:        realm,
		Host:         host,
		Client:       client,
	}

	// Test connection and get certs
	ctx := context.Background()
	_, err := KC.Client.GetCerts(ctx, KC.Realm)
	if err != nil {
		return fmt.Errorf("failed to get Keycloak certs: %v", err)
	}

	return nil
}

// GetToken gets a new token from Keycloak
func (kc *KeycloakConfig) GetToken(ctx context.Context) (*gocloak.JWT, error) {
	token, err := kc.Client.LoginClient(ctx, kc.ClientID, kc.ClientSecret, kc.Realm)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %v", err)
	}
	return token, nil
}

// ValidateToken validates the given token
func (kc *KeycloakConfig) ValidateToken(ctx context.Context, accessToken string) (*gocloak.IntroSpectTokenResult, error) {
	result, err := kc.Client.RetrospectToken(ctx, accessToken, kc.ClientID, kc.ClientSecret, kc.Realm)
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %v", err)
	}
	return result, nil
}

// GetUserInfo gets user info from the token
func (kc *KeycloakConfig) GetUserInfo(ctx context.Context, accessToken string) (*gocloak.UserInfo, error) {
	fmt.Printf("Getting user info with token: %s\n", accessToken)
	userInfo, err := kc.Client.GetUserInfo(ctx, accessToken, kc.Realm)
	if err != nil {
		fmt.Printf("Error getting user info: %v\n", err)
		return nil, fmt.Errorf("failed to get user info: %v", err)
	}
	fmt.Printf("User info received: %+v\n", userInfo)
	return userInfo, nil
}

// NewKeycloakClient 함수 정의
func NewKeycloakClient(config *KeycloakConfig) *gocloak.GoCloak {
	return gocloak.NewClient(config.Host)
}

// LoginUser 사용자 로그인을 수행하고 토큰을 반환합니다.
func (c *KeycloakConfig) LoginUser(ctx context.Context, client *gocloak.GoCloak, username, password string) (*gocloak.JWT, error) {
	token, err := client.Login(ctx,
		c.ClientID,
		c.ClientSecret,
		c.Realm,
		username,
		password,
	)
	if err != nil {
		return nil, fmt.Errorf("로그인 실패: %v", err)
	}
	return token, nil
}

// LoginAdmin 관리자 로그인을 수행하고 토큰을 반환합니다.
func (c *KeycloakConfig) LoginAdmin(ctx context.Context) (*gocloak.JWT, error) {
	adminID := os.Getenv("MCIAMMANAGER_PLATFORMADMIN_ID")
	adminPassword := os.Getenv("MCIAMMANAGER_PLATFORMADMIN_PASSWORD")

	if adminID == "" || adminPassword == "" {
		return nil, fmt.Errorf("관리자 계정 정보가 설정되지 않았습니다")
	}

	token, err := c.Client.Login(ctx,
		c.ClientID,
		c.ClientSecret,
		c.Realm,
		adminID,
		adminPassword,
	)
	if err != nil {
		return nil, fmt.Errorf("관리자 로그인 실패: %v", err)
	}
	return token, nil
}
