package iamtokenvalidator

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func startJWKSServer(t *testing.T, key *rsa.PrivateKey, kid string) *httptest.Server {
	t.Helper()

	n := base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}) // 65537

	jwks := map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"kid": kid,
				"use": "sig",
				"alg": "RS256",
				"n":   n,
				"e":   e,
			},
		},
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))
}

func signToken(t *testing.T, key *rsa.PrivateKey, kid string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return signed
}

func TestIsTokenValidAndGetClaims(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	const kid = "test-kid-1"

	srv := startJWKSServer(t, key, kid)
	defer srv.Close()

	if err := Init(srv.URL); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	claims := jwt.MapClaims{
		"iss": "https://keycloak.example.com/realms/mcmp-demo",
		"sub": "user-123",
		"exp": time.Now().Add(time.Hour).Unix(),
		"realm_access": map[string]any{
			"roles": []string{"admin", "offline_access", "uma_authorization", "default-roles-mcmp-demo"},
		},
		"resource_access": map[string]any{
			"mciamClient": map[string]any{"roles": []string{"workspaceAdmin"}},
		},
		"preferred_username": "alice",
	}
	tokenString := signToken(t, key, kid, claims)

	if err := IsTokenValid(tokenString); err != nil {
		t.Fatalf("expected valid token, got error: %v", err)
	}

	parsed, err := GetClaims(tokenString)
	if err != nil {
		t.Fatalf("GetClaims failed: %v", err)
	}

	if parsed.Subject != "user-123" {
		t.Errorf("expected subject user-123, got %q", parsed.Subject)
	}

	platformRoles := parsed.PlatformRoles()
	if len(platformRoles) != 1 || platformRoles[0] != "admin" {
		t.Errorf("expected PlatformRoles [admin] (built-in/default roles filtered out), got %v", platformRoles)
	}

	resourceRoles := parsed.ResourceRoles("mciamClient")
	if len(resourceRoles) != 1 || resourceRoles[0] != "workspaceAdmin" {
		t.Errorf("expected ResourceRoles(mciamClient) [workspaceAdmin], got %v", resourceRoles)
	}
}

func TestIsTokenValidRejectsExpiredToken(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	const kid = "test-kid-2"

	srv := startJWKSServer(t, key, kid)
	defer srv.Close()

	if err := Init(srv.URL); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	claims := jwt.MapClaims{
		"iss": "https://keycloak.example.com/realms/mcmp-demo",
		"sub": "user-123",
		"exp": time.Now().Add(-time.Hour).Unix(),
	}
	tokenString := signToken(t, key, kid, claims)

	if err := IsTokenValid(tokenString); err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestIsTokenValidRejectsWrongSigningKey(t *testing.T) {
	trustedKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	attackerKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	const kid = "test-kid-3"

	srv := startJWKSServer(t, trustedKey, kid)
	defer srv.Close()

	if err := Init(srv.URL); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	claims := jwt.MapClaims{
		"iss": "https://keycloak.example.com/realms/mcmp-demo",
		"sub": "attacker",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenString := signToken(t, attackerKey, kid, claims)

	if err := IsTokenValid(tokenString); err == nil {
		t.Fatal("expected token signed by an untrusted key to be rejected")
	}
}

func TestInitFailsOnUnreachableURL(t *testing.T) {
	if err := Init("http://127.0.0.1:0/does-not-exist"); err == nil {
		t.Fatal("expected Init to fail for an unreachable JWKS URL")
	}
}

func TestKeyfuncOrErrorBeforeInit(t *testing.T) {
	jwks.Store(nil)
	if _, err := keyfuncOrError(); err == nil {
		t.Fatal("expected error when Init has not been called")
	}
}

func TestInitFailsOnNonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	if err := Init(srv.URL); err == nil {
		t.Fatal("expected Init to fail when the JWKS endpoint returns a non-200 status")
	}
}

func TestGetClaimsRejectsMalformedToken(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	srv := startJWKSServer(t, key, "test-kid-malformed")
	defer srv.Close()

	if err := Init(srv.URL); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if _, err := GetClaims("not-a-jwt"); err == nil {
		t.Fatal("expected GetClaims to reject a malformed token string")
	}
}

func TestPlatformRolesDedupesAcrossClaimSources(t *testing.T) {
	claims := &IamManagerClaims{
		Roles: []string{"admin", "operator"},
	}
	claims.Issuer = "https://keycloak.example.com/realms/mcmp-demo"
	claims.RealmAccess.Roles = []string{"operator", "viewer", "default-roles-mcmp-demo"}

	roles := claims.PlatformRoles()
	want := []string{"admin", "operator", "viewer"}
	if len(roles) != len(want) {
		t.Fatalf("expected %v, got %v", want, roles)
	}
	for i, r := range want {
		if roles[i] != r {
			t.Fatalf("expected %v, got %v", want, roles)
		}
	}
}

func TestPlatformRolesWithoutRealmsMarkerInIssuer(t *testing.T) {
	claims := &IamManagerClaims{}
	claims.Issuer = "https://keycloak.example.com/some-other-path"
	claims.RealmAccess.Roles = []string{"admin", "offline_access"}

	roles := claims.PlatformRoles()
	if len(roles) != 1 || roles[0] != "admin" {
		t.Errorf("expected [admin], got %v", roles)
	}
}

func TestHasAnyRole(t *testing.T) {
	cases := []struct {
		name    string
		granted []string
		user    []string
		want    bool
	}{
		{"overlap", []string{"admin", "operator"}, []string{"viewer", "operator"}, true},
		{"no overlap", []string{"admin"}, []string{"viewer"}, false},
		{"empty user roles", []string{"admin"}, nil, false},
		{"empty granted roles", nil, []string{"admin"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := HasAnyRole(tc.granted, tc.user); got != tc.want {
				t.Errorf("HasAnyRole(%v, %v) = %v, want %v", tc.granted, tc.user, got, tc.want)
			}
		})
	}
}
