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

	ClientName       string
	ClientID         string
	ClientSecret     string
	OIDCClientName   string
	OIDCClientID     string
	OIDCClientSecret string
}

var KC *KeycloakConfig

// InitKeycloak Keycloak 초기화
func InitKeycloak() error {
	host := os.Getenv("MC_IAM_MANAGER_KEYCLOAK_HOST")
	if host == "" {
		return fmt.Errorf("MC_IAM_MANAGER_KEYCLOAK_HOST is not set")
	}
	fmt.Printf("MC_IAM_MANAGER_KEYCLOAK_HOST: %s\n", host)

	realm := os.Getenv("MC_IAM_MANAGER_KEYCLOAK_REALM")
	if realm == "" {
		return fmt.Errorf("MC_IAM_MANAGER_KEYCLOAK_REALM is not set")
	}
	fmt.Printf("MC_IAM_MANAGER_KEYCLOAK_REALM: %s\n", realm)

	clientName := os.Getenv("MC_IAM_MANAGER_KEYCLOAK_CLIENT_NAME")
	if clientName == "" {
		return fmt.Errorf("MC_IAM_MANAGER_KEYCLOAK_CLIENT_NAME is not set")
	}
	fmt.Printf("MC_IAM_MANAGER_KEYCLOAK_CLIENT_NAME: %s\n", clientName)

	// clientID := os.Getenv("KEYCLOAK_CLIENT_ID")
	// if clientID == "" {
	// 	return fmt.Errorf("KEYCLOAK_CLIENT_ID is not set")
	// }
	// fmt.Printf("KEYCLOAK_CLIENT_ID: %s\n", clientID)

	oidcClientID := os.Getenv("MC_IAM_MANAGER_KEYCLOAK_OIDC_CLIENT_ID")
	if oidcClientID == "" {
		return fmt.Errorf("MC_IAM_MANAGER_KEYCLOAK_OIDC_CLIENT_ID is not set")
	}
	fmt.Printf("MC_IAM_MANAGER_KEYCLOAK_OIDC_CLIENT_ID: %s\n", oidcClientID)

	oidcClientName := os.Getenv("MC_IAM_MANAGER_KEYCLOAK_OIDC_CLIENT_NAME")
	if oidcClientName == "" {
		return fmt.Errorf("MC_IAM_MANAGER_KEYCLOAK_OIDC_CLIENT_NAME is not set")
	}
	fmt.Printf("MC_IAM_MANAGER_KEYCLOAK_OIDC_CLIENT_NAME: %s\n", oidcClientName)

	clientSecret := os.Getenv("MC_IAM_MANAGER_KEYCLOAK_CLIENT_SECRET")
	if clientSecret == "" {
		return fmt.Errorf("MC_IAM_MANAGER_KEYCLOAK_CLIENT_SECRET is not set")
	}
	fmt.Printf("MC_IAM_MANAGER_KEYCLOAK_CLIENT_SECRET: %s\n", clientSecret)

	oidcClientSecret := os.Getenv("MC_IAM_MANAGER_KEYCLOAK_OIDC_CLIENT_SECRET")
	if oidcClientSecret == "" {
		return fmt.Errorf("MC_IAM_MANAGER_KEYCLOAK_OIDC_CLIENT_SECRET is not set")
	}
	fmt.Printf("MC_IAM_MANAGER_KEYCLOAK_OIDC_CLIENT_SECRET: %s\n", oidcClientSecret)

	platformAdminID := os.Getenv("MC_IAM_MANAGER_PLATFORMADMIN_ID")
	fmt.Printf("MC_IAM_MANAGER_PLATFORMADMIN_ID: %s\n", platformAdminID)

	keycloakAdmin := os.Getenv("MC_IAM_MANAGER_KEYCLOAK_ADMIN")
	fmt.Printf("MC_IAM_MANAGER_KEYCLOAK_ADMIN: %s\n", keycloakAdmin)

	client := gocloak.NewClient(host)

	KC = &KeycloakConfig{
		Realm:      realm,
		Host:       host,
		Client:     client,
		ClientName: clientName,
		// ClientID:         clientID,
		ClientSecret:     clientSecret,
		OIDCClientID:     oidcClientID,
		OIDCClientName:   oidcClientName,
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
	token, err := kc.Client.LoginClient(ctx, kc.ClientName, kc.ClientSecret, kc.Realm)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %v", err)
	}
	return token, nil
}

// ValidateToken validates the given token
func (kc *KeycloakConfig) ValidateToken(ctx context.Context, accessToken string) (*gocloak.IntroSpectTokenResult, error) {
	result, err := kc.Client.RetrospectToken(ctx, accessToken, kc.ClientName, kc.ClientSecret, kc.Realm)
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
	adminUsername := os.Getenv("MC_IAM_MANAGER_KEYCLOAK_ADMIN")
	adminPassword := os.Getenv("MC_IAM_MANAGER_MC_IAM_MANAGER_KEYCLOAK_ADMIN_PASSWORD")

	log.Printf("[DEBUG] Attempting admin login with:")
	log.Printf("[DEBUG] - Host: %s", kc.Host)
	log.Printf("[DEBUG] - Realm: %s", kc.Realm)
	log.Printf("[DEBUG] - Admin Username: %s", adminUsername)
	// log.Printf("[DEBUG] - Admin Password: %s", adminPassword)

	if adminUsername == "" || adminPassword == "" {
		return nil, fmt.Errorf("MC_IAM_MANAGER_KEYCLOAK_ADMIN or MC_IAM_MANAGER_MC_IAM_MANAGER_KEYCLOAK_ADMIN_PASSWORD not set")
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

	// 인증서 정보 로깅
	log.Printf("[DEBUG] Number of keys in certs: %d", len(*certs.Keys))
	for i, key := range *certs.Keys {
		log.Printf("[DEBUG] Key %d - KID: %s, Algorithm: %s", i, *key.Kid, *key.Alg)
	}

	// 첫 번째 인증서의 공개키 추출
	if certs.Keys == nil || len(*certs.Keys) == 0 {
		return nil, fmt.Errorf("no public keys found in Keycloak certs")
	}

	// RS256 알고리즘을 사용하는 키 찾기
	var selectedKey *gocloak.CertResponseKey
	for _, key := range *certs.Keys {
		if *key.Alg == "RS256" {
			selectedKey = &key
			break
		}
	}

	if selectedKey == nil {
		return nil, fmt.Errorf("no RS256 key found in Keycloak certs")
	}

	// RSA 공개키 구성
	if selectedKey.N == nil || selectedKey.E == nil {
		return nil, fmt.Errorf("invalid key format: missing modulus or exponent")
	}

	n, err := base64.RawURLEncoding.DecodeString(*selectedKey.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %v", err)
	}

	e, err := base64.RawURLEncoding.DecodeString(*selectedKey.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %v", err)
	}

	// RSA 공개키 생성
	publicKey := &rsa.PublicKey{
		N: new(big.Int).SetBytes(n),
		E: int(new(big.Int).SetBytes(e).Int64()),
	}

	log.Printf("[DEBUG] Generated RSA public key - Size: %d bits, KID: %s", publicKey.Size()*8, *selectedKey.Kid)
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

	adminUsername := os.Getenv("MC_IAM_MANAGER_KEYCLOAK_ADMIN")
	adminPassword := os.Getenv("MC_IAM_MANAGER_MC_IAM_MANAGER_KEYCLOAK_ADMIN_PASSWORD")

	if adminUsername == "" || adminPassword == "" {
		return nil, fmt.Errorf("MC_IAM_MANAGER_KEYCLOAK_ADMIN or MC_IAM_MANAGER_MC_IAM_MANAGER_KEYCLOAK_ADMIN_PASSWORD not set")
	}

	log.Printf("[DEBUG] Attempting admin login with:")
	log.Printf("[DEBUG] - Host: %s", kc.Host)
	log.Printf("[DEBUG] - Realm: %s", kc.Realm)
	log.Printf("[DEBUG] - Admin Username: %s", adminUsername)
	// log.Printf("[DEBUG] - Admin Password: %s", adminPassword)

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
