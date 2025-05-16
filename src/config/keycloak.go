package config

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"sync"
	"time"

	"crypto/rsa"
	"encoding/base64"

	"github.com/Nerzal/gocloak/v13"
)

// KeycloakConfig Keycloak 설정
type KeycloakConfig struct {
	Realm       string
	Host        string
	Client      *gocloak.GoCloak
	adminToken  *gocloak.JWT
	tokenExpiry time.Time
	tokenMutex  sync.RWMutex

	ClientID         string
	ClientSecret     string
	OIDCClientID     string
	OIDCClientSecret string
}

var KC *KeycloakConfig

// InitKeycloak Keycloak 초기화
func InitKeycloak() error {
	host := os.Getenv("KEYCLOAK_HOST")
	if host == "" {
		return fmt.Errorf("KEYCLOAK_HOST is not set")
	}
	fmt.Printf("KEYCLOAK_HOST: %s\n", host)

	realm := os.Getenv("KEYCLOAK_REALM")
	if realm == "" {
		return fmt.Errorf("KEYCLOAK_REALM is not set")
	}
	fmt.Printf("KEYCLOAK_REALM: %s\n", realm)

	clientID := os.Getenv("KEYCLOAK_CLIENT")
	if clientID == "" {
		return fmt.Errorf("KEYCLOAK_CLIENT is not set")
	}
	fmt.Printf("KEYCLOAK_CLIENT: %s\n", clientID)
	oidcClientID := os.Getenv("KEYCLOAK_OIDC_CLIENT")
	if oidcClientID == "" {
		return fmt.Errorf("KEYCLOAK_OIDC_CLIENT is not set")
	}
	fmt.Printf("KEYCLOAK_OIDC_CLIENT: %s\n", oidcClientID)
	clientSecret := os.Getenv("KEYCLOAK_CLIENT_SECRET")
	if clientSecret == "" {
		return fmt.Errorf("KEYCLOAK_CLIENT_SECRET is not set")
	}
	fmt.Printf("KEYCLOAK_CLIENT_SECRET: %s\n", clientSecret)

	oidcClientSecret := os.Getenv("KEYCLOAK_OIDC_CLIENT_SECRET")
	if oidcClientSecret == "" {
		return fmt.Errorf("KEYCLOAK_OIDC_CLIENT_SECRET is not set")
	}
	fmt.Printf("KEYCLOAK_OIDC_CLIENT_SECRET: %s\n", oidcClientSecret)

	platformAdminID := os.Getenv("MCIAMMANAGER_PLATFORMADMIN_ID")
	fmt.Printf("MCIAMMANAGER_PLATFORMADMIN_ID: %s\n", platformAdminID)

	keycloakAdmin := os.Getenv("KEYCLOAK_ADMIN")
	fmt.Printf("KEYCLOAK_ADMIN: %s\n", keycloakAdmin)

	client := gocloak.NewClient(host)

	KC = &KeycloakConfig{
		Realm:            realm,
		Host:             host,
		Client:           client,
		ClientID:         clientID,
		ClientSecret:     clientSecret,
		OIDCClientID:     oidcClientID,
		OIDCClientSecret: oidcClientSecret,
	}

	// Test connection and get certs
	// ctx := context.Background()
	// _, err := KC.Client.GetCerts(ctx, KC.Realm)
	// if err != nil {
	// 	return fmt.Errorf("failed to get Keycloak certs: %v", err)
	// }

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

// LoginAdmin performs admin login to Keycloak
func (kc *KeycloakConfig) LoginAdmin(ctx context.Context) (*gocloak.JWT, error) {
	adminUsername := os.Getenv("KEYCLOAK_ADMIN")
	adminPassword := os.Getenv("KEYCLOAK_ADMIN_PASSWORD")

	log.Printf("[DEBUG] Attempting admin login with:")
	log.Printf("[DEBUG] - Host: %s", kc.Host)
	log.Printf("[DEBUG] - Realm: %s", kc.Realm)
	log.Printf("[DEBUG] - Admin Username: %s", adminUsername)
	log.Printf("[DEBUG] - Admin Password: %s", adminPassword)

	if adminUsername == "" || adminPassword == "" {
		return nil, fmt.Errorf("KEYCLOAK_ADMIN or KEYCLOAK_ADMIN_PASSWORD not set")
	}

	token, err := kc.Client.LoginAdmin(ctx, adminUsername, adminPassword, "master")
	if err != nil {
		log.Printf("[DEBUG] Admin login failed: %v", err)
		return nil, fmt.Errorf("관리자 로그인 실패: %w", err)
	}

	log.Printf("[DEBUG] Admin login successful")
	return token, nil
}

// GetPublicKey는 Keycloak의 공개키를 가져옵니다.
func (kc *KeycloakConfig) GetPublicKey() (interface{}, error) {
	ctx := context.Background()

	// Keycloak의 인증서 정보 가져오기
	certs, err := kc.Client.GetCerts(ctx, kc.Realm)
	if err != nil {
		return nil, fmt.Errorf("failed to get Keycloak certs: %v", err)
	}

	// 첫 번째 인증서의 공개키 추출
	if certs.Keys == nil || len(*certs.Keys) == 0 {
		return nil, fmt.Errorf("no public keys found in Keycloak certs")
	}

	// 첫 번째 키의 공개키 추출
	key := (*certs.Keys)[0]

	// RSA 공개키 구성
	if key.N == nil || key.E == nil {
		return nil, fmt.Errorf("invalid key format: missing modulus or exponent")
	}

	n, err := base64.RawURLEncoding.DecodeString(*key.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %v", err)
	}

	e, err := base64.RawURLEncoding.DecodeString(*key.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %v", err)
	}

	// RSA 공개키 생성
	publicKey := &rsa.PublicKey{
		N: new(big.Int).SetBytes(n),
		E: int(new(big.Int).SetBytes(e).Int64()),
	}

	return publicKey, nil
}

// GetAdminToken admin 토큰을 가져옵니다. 캐시된 토큰이 있으면 재사용하고, 없거나 만료되었으면 새로 발급받습니다.
func (kc *KeycloakConfig) GetAdminToken(ctx context.Context) (*gocloak.JWT, error) {
	kc.tokenMutex.RLock()
	if kc.adminToken != nil && time.Now().Before(kc.tokenExpiry) {
		token := kc.adminToken
		kc.tokenMutex.RUnlock()
		return token, nil
	}
	kc.tokenMutex.RUnlock()

	kc.tokenMutex.Lock()
	defer kc.tokenMutex.Unlock()

	// Double check
	if kc.adminToken != nil && time.Now().Before(kc.tokenExpiry) {
		return kc.adminToken, nil
	}

	adminUsername := os.Getenv("KEYCLOAK_ADMIN")
	adminPassword := os.Getenv("KEYCLOAK_ADMIN_PASSWORD")

	if adminUsername == "" || adminPassword == "" {
		return nil, fmt.Errorf("KEYCLOAK_ADMIN or KEYCLOAK_ADMIN_PASSWORD not set")
	}

	log.Printf("[DEBUG] Attempting admin login with:")
	log.Printf("[DEBUG] - Host: %s", kc.Host)
	log.Printf("[DEBUG] - Realm: %s", kc.Realm)
	log.Printf("[DEBUG] - Admin Username: %s", adminUsername)
	log.Printf("[DEBUG] - Admin Password: %s", adminPassword)

	token, err := kc.Client.LoginAdmin(ctx, adminUsername, adminPassword, "master")
	if err != nil {
		log.Printf("[DEBUG] Admin login failed: %v", err)
		return nil, fmt.Errorf("관리자 로그인 실패: %w", err)
	}

	kc.adminToken = token
	// 토큰 만료 60초 전에 갱신
	kc.tokenExpiry = time.Now().Add(time.Duration(token.ExpiresIn-60) * time.Second)
	log.Printf("[DEBUG] Admin login successful, token expires in %d seconds", token.ExpiresIn)

	return token, nil
}
