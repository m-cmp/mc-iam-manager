# iamtokenvalidator

`iamtokenvalidator` is a standalone Go module for validating access tokens
issued by [mc-iam-manager](https://github.com/m-cmp/mc-iam-manager) in other
Go services (cb-tumblebug, cb-spider, and other mc-cmp components), without
pulling in mc-iam-manager's own dependency tree (gorm, gocloak, echo, ...).

It verifies a token's signature against mc-iam-manager's published JWKS
(`GET /api/auth/certs`) and extracts platform/resource roles using the same
rules mc-iam-manager's `AuthMiddleware` applies, so role interpretation stays
consistent across services.

## Installation

```bash
go get github.com/m-cmp/mc-iam-manager/iamtokenvalidator
```

## Usage

Call `Init` once at startup with mc-iam-manager's certs endpoint. The
returned key set refreshes itself in the background (and on an unknown
`kid`), so it survives Keycloak key rotation without a restart.

```go
import "github.com/m-cmp/mc-iam-manager/iamtokenvalidator"

func main() {
    if err := iamtokenvalidator.Init("https://mciam.example.com/api/auth/certs"); err != nil {
        log.Fatalf("failed to initialize iamtokenvalidator: %v", err)
    }
    // ...
}
```

### Validate only

```go
if err := iamtokenvalidator.IsTokenValid(tokenString); err != nil {
    // reject request: token is missing, expired, or signed by an untrusted key
}
```

### Validate and read roles

```go
claims, err := iamtokenvalidator.GetClaims(tokenString)
if err != nil {
    // reject request
}

claims.Subject                          // Keycloak user id
claims.PreferredUsername                // username
claims.PlatformRoles()                  // e.g. ["admin"] (built-in/default realm roles filtered out)
claims.ResourceRoles("mciamClient")     // e.g. ["workspaceAdmin"]

if iamtokenvalidator.HasAnyRole([]string{"admin", "operator"}, claims.PlatformRoles()) {
    // authorized
}
```

### echo middleware example (cb-tumblebug)

```go
func IamAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        authHeader := c.Request().Header.Get("Authorization")
        if !strings.HasPrefix(authHeader, "Bearer ") {
            return echo.NewHTTPError(http.StatusUnauthorized, "missing bearer token")
        }
        token := strings.TrimPrefix(authHeader, "Bearer ")

        claims, err := iamtokenvalidator.GetClaims(token)
        if err != nil {
            return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
        }

        c.Set("platformRoles", claims.PlatformRoles())
        c.Set("kcUserId", claims.Subject)
        return next(c)
    }
}
```

### net/http middleware example

```go
func IamAuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

        claims, err := iamtokenvalidator.GetClaims(token)
        if err != nil {
            http.Error(w, "invalid token", http.StatusUnauthorized)
            return
        }

        ctx := context.WithValue(r.Context(), userClaimsKey, claims)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## Calling from outside Go (REST-only, no SDK)

`iamtokenvalidator` only helps Go services. Everything it does is a thin
wrapper over two plain REST endpoints on mc-iam-manager, so any other
language/framework can validate a token the same way without an SDK.

### Option A — verify the JWT locally (recommended, no round-trip per request)

1. Fetch the JWKS once at startup and cache it:

   ```bash
   curl https://mciam.example.com/api/auth/certs
   ```

   ```json
   {
     "keys": [
       { "kid": "abc123", "kty": "RSA", "alg": "RS256", "use": "sig", "n": "...", "e": "AQAB" }
     ]
   }
   ```

   No auth header is required — this endpoint is public.

2. In your own language, verify the incoming access token's signature against
   that JWKS (RS256/384/512 only), matched by the token header's `kid`.

   **Python** (`pip install pyjwt cryptography`):

   ```python
   import jwt
   from jwt import PyJWKClient

   jwk_client = PyJWKClient("https://mciam.example.com/api/auth/certs")  # cache/reuse this client

   def validate(token: str) -> dict:
       signing_key = jwk_client.get_signing_key_from_jwt(token)
       return jwt.decode(token, signing_key.key, algorithms=["RS256", "RS384", "RS512"])
   ```

   **Node.js** (`npm install jsonwebtoken jwks-rsa`):

   ```js
   const jwt = require("jsonwebtoken");
   const jwksClient = require("jwks-rsa");

   const client = jwksClient({ jwksUri: "https://mciam.example.com/api/auth/certs" });

   function getKey(header, callback) {
     client.getSigningKey(header.kid, (err, key) => callback(err, key?.getPublicKey()));
   }

   function validate(token) {
     return new Promise((resolve, reject) => {
       jwt.verify(token, getKey, { algorithms: ["RS256", "RS384", "RS512"] }, (err, claims) =>
         err ? reject(err) : resolve(claims)
       );
     });
   }
   ```

3. Extract roles from the decoded claims with the same rule
   `IamManagerClaims.PlatformRoles()` uses, so your service agrees with
   mc-iam-manager and the Go clients on who has what role:

   - Platform roles = top-level `roles` claim (if present) + `realm_access.roles`,
     minus `offline_access`, `uma_authorization`, and `default-roles-{realm}`
     (the realm name is the last path segment of the `iss` claim, e.g.
     `.../realms/mcmp-demo` → `mcmp-demo`).
   - Resource/workspace roles = `resource_access.<clientId>.roles`.

### Option B — ask mc-iam-manager to validate it for you (remote check)

If you'd rather not implement JWKS verification at all, call mc-iam-manager
directly per request. This adds a network round-trip and requires the
caller's refresh token, but needs no library beyond a plain HTTP client:

```bash
curl -X POST https://mciam.example.com/api/auth/validate \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}"
```

- `200 {"valid": true, "user_id": "..."}` — token is valid.
- `200 {"valid": false, "token": {...}}` — the access token had expired; a new
  one was already issued using the refresh token and is returned.
- `401` — both the access token and the refresh attempt failed.

This endpoint does not return role claims — decode the (possibly refreshed)
access token client-side (Option A) if you need roles, or introspect the
same fields Option A describes.

## Notes

- `Init` must be called exactly once per process before `IsTokenValid` /
  `GetClaims` are used.
- Only RS256/RS384/RS512-signed tokens are accepted, matching
  mc-iam-manager's Keycloak realm configuration.
- This package only checks that a token is valid and reads its role claims.
  It does not call back into mc-iam-manager, so it does not perform
  fine-grained UMA permission/ticket checks — for that, use
  mc-iam-manager's `POST /api/auth/validate` API directly.
