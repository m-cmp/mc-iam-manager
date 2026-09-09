// Package iamtokenvalidator validates access tokens issued by mc-iam-manager
// against its published JWKS (GET /api/auth/certs) and extracts role claims
// using the same rules mc-iam-manager's own AuthMiddleware applies, so that
// external services (e.g. cb-tumblebug) interpret roles consistently with
// mc-iam-manager itself.
package iamtokenvalidator

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// IamManagerClaims mirrors the claim shape mc-iam-manager's Keycloak realm
// issues: standard registered claims plus realm_access/resource_access role
// grants and a few commonly used profile fields.
type IamManagerClaims struct {
	jwt.RegisteredClaims

	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`

	ResourceAccess map[string]struct {
		Roles []string `json:"roles"`
	} `json:"resource_access"`

	// Roles is set when a top-level "roles" claim mapper is configured.
	Roles []string `json:"roles,omitempty"`

	PreferredUsername string `json:"preferred_username,omitempty"`
	Email             string `json:"email,omitempty"`
}

// excludedRealmRoles are Keycloak's built-in realm roles that carry no
// mc-cmp platform meaning and are filtered out of PlatformRoles(), matching
// mc-iam-manager's src/middleware/auth.go AuthMiddleware.
var excludedRealmRoles = map[string]bool{
	"offline_access":    true,
	"uma_authorization": true,
}

// realm extracts the Keycloak realm name from the token issuer
// (".../realms/{realm}"), used to filter out the realm's own
// "default-roles-{realm}" entry.
func (c *IamManagerClaims) realm() string {
	const marker = "/realms/"
	idx := strings.LastIndex(c.Issuer, marker)
	if idx == -1 {
		return ""
	}
	return c.Issuer[idx+len(marker):]
}

// PlatformRoles returns the caller's platform-level roles: the top-level
// "roles" claim plus realm_access.roles, minus Keycloak's built-in roles
// and the realm's own "default-roles-{realm}", deduplicated. This matches
// mc-iam-manager's AuthMiddleware role extraction exactly.
func (c *IamManagerClaims) PlatformRoles() []string {
	excluded := map[string]bool{"default-roles-" + c.realm(): true}
	for role, v := range excludedRealmRoles {
		excluded[role] = v
	}

	seen := make(map[string]bool)
	var roles []string
	appendRole := func(role string) {
		if excluded[role] || seen[role] {
			return
		}
		seen[role] = true
		roles = append(roles, role)
	}
	for _, role := range c.Roles {
		appendRole(role)
	}
	for _, role := range c.RealmAccess.Roles {
		appendRole(role)
	}
	return roles
}

// ResourceRoles returns the roles granted for a specific Keycloak client
// (resource_access.{client}.roles), e.g. ResourceRoles("mciamClient").
func (c *IamManagerClaims) ResourceRoles(client string) []string {
	return c.ResourceAccess[client].Roles
}

// HasAnyRole reports whether userRoles contains any role from grantedRoles.
func HasAnyRole(grantedRoles []string, userRoles []string) bool {
	granted := make(map[string]struct{}, len(grantedRoles))
	for _, r := range grantedRoles {
		granted[r] = struct{}{}
	}
	for _, r := range userRoles {
		if _, ok := granted[r]; ok {
			return true
		}
	}
	return false
}

var jwks atomic.Pointer[keyfunc.Keyfunc]

// Init fetches mc-iam-manager's JWKS from certURL (its "/api/auth/certs"
// endpoint) and must be called once before IsTokenValid/GetClaims are used.
// The returned keyset refreshes itself in the background and on encountering
// an unknown "kid", so callers do not need to re-fetch after Keycloak
// rotates its signing keys.
func Init(certURL string) error {
	// keyfunc's background refresh only logs a fetch failure rather than
	// surfacing it here, so probe the endpoint once up front to fail fast
	// on a bad URL/host at startup.
	resp, err := http.Get(certURL)
	if err != nil {
		return fmt.Errorf("iamtokenvalidator: failed to reach JWKS endpoint %s: %w", certURL, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("iamtokenvalidator: JWKS endpoint %s returned status %d", certURL, resp.StatusCode)
	}

	kf, err := keyfunc.NewDefaultCtx(context.Background(), []string{certURL})
	if err != nil {
		return fmt.Errorf("iamtokenvalidator: failed to initialize JWKS client for %s: %w", certURL, err)
	}
	jwks.Store(&kf)
	return nil
}

func keyfuncOrError() (jwt.Keyfunc, error) {
	kf := jwks.Load()
	if kf == nil {
		return nil, fmt.Errorf("iamtokenvalidator: Init must be called before validating tokens")
	}
	return (*kf).Keyfunc, nil
}

// IsTokenValid verifies tokenString's signature and standard claims
// (exp/nbf/iat) against the JWKS fetched by Init.
func IsTokenValid(tokenString string) error {
	kf, err := keyfuncOrError()
	if err != nil {
		return err
	}
	token, err := jwt.Parse(tokenString, kf)
	if err != nil {
		return fmt.Errorf("iamtokenvalidator: token is invalid: %w", err)
	}
	if !token.Valid {
		return fmt.Errorf("iamtokenvalidator: token is invalid")
	}
	return nil
}

// GetClaims verifies tokenString the same way IsTokenValid does and returns
// its parsed IamManagerClaims (subject, platform/resource roles, etc).
func GetClaims(tokenString string) (*IamManagerClaims, error) {
	kf, err := keyfuncOrError()
	if err != nil {
		return nil, err
	}
	claims := &IamManagerClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, kf)
	if err != nil {
		return nil, fmt.Errorf("iamtokenvalidator: token is invalid: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("iamtokenvalidator: token is invalid")
	}
	return claims, nil
}
