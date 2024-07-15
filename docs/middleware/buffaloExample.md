---
layout: default
title: Buffalo Example
parent: Middleware for your App
order: 2
---
## Middleware

### 0. `init` func

Use GetPubkeyIamManager to set up a public key.

```go
func init() {
	r = render.New(render.Options{
		DefaultContentType: "application/json",
	})
	IDP_CERT_ENDPOINT := os.Getenv("IDP_CERT_ENDPOINT")
	err := iamtokenvalidator.GetPubkeyIamManager(IDP_CERT_ENDPOINT)
	if err != nil {
		panic(err)
	}
}
```

### 1. `IsAuthMiddleware` func middleware

This middleware checks if the provided access token in the Authorization header is valid.

**Usage:**

```go
app.Use(middleware.IsAuthMiddleware)
```

**IsAuthMiddleware:**
```go
func IsAuthMiddleware(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		accessToken := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")
		err := iamtokenvalidator.IsTokenValid(accessToken)
		if err != nil {
			return c.Render(http.StatusUnauthorized, r.JSON(map[string]string{"error": "Unauthorized"}))
		}
		return next(c)
	}
}
```

### 2. `SetRolesMiddleware` func middleware

This middleware extracts roles from the access token claims and sets them in the context.

**Usage:**

```go
app.Use(middleware.SetRolesMiddleware)
```

**SetRolesMiddleware:**
```go
func SetRolesMiddleware(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		accessToken := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")
		claims, err := iamtokenvalidator.GetTokenClaimsByIamManagerClaims(accessToken)
		if err != nil {
			return c.Render(http.StatusUnauthorized, r.JSON(map[string]string{"error": "Unauthorized"}))
		}
		c.Set("roles", claims.RealmAccess.Roles)
		return next(c)
	}
}
```

### 3. `SetGrantedRolesMiddleware`

This middleware ensures that the user has one of the specified roles.

**Parameters:**

- `roles []string`: A list of roles that are granted access.

**Usage:**

```go
roles := []string{"admin", "viewer"}
app.Use(middleware.SetGrantedRolesMiddleware(roles))
```

```go
func SetGrantedRolesMiddleware(roles []string) buffalo.MiddlewareFunc {
	return func(next buffalo.Handler) buffalo.Handler {
		return func(c buffalo.Context) error {
			userRoles := c.Value("roles")
			userRolesArr := userRoles.([]string)
			userRolesArrSet := make(map[string]struct{}, len(userRolesArr))
			for _, v := range userRolesArr {
				userRolesArrSet[v] = struct{}{}
			}
			for _, v := range roles {
				if _, found := userRolesArrSet[v]; found {
					return next(c)
				}
			}
			return c.Render(http.StatusUnauthorized, r.JSON(map[string]string{"error": "Unauthorized"}))
		}
	}
}
```

## Buffalo FULL Example

Here is an example of how to use these middlewares in your Buffalo application:

**app.go:**

```go
tokenTestPath := app.Group(apiPath + "/tokentest")
tokenTestPath.Use(middleware.IsAuthMiddleware)
tokenTestPath.Use(middleware.SetRolesMiddleware)
tokenTestPath.GET("/", aliveSig)
tokenTestPath.GET("/admin", middleware.SetGrantedRolesMiddleware([]string{"admin"})(aliveSig))
tokenTestPath.GET("/operator", middleware.SetGrantedRolesMiddleware([]string{"admin", "operator"})(aliveSig))
tokenTestPath.GET("/viewer", middleware.SetGrantedRolesMiddleware([]string{"admin", "operator", "viewer"})(aliveSig))

func aliveSig(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.JSON(map[string]string{"ststus": "ok"}))
}
```