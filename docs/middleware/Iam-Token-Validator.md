---
layout: default
title: Iam Token Validator
parent: Middleware for your App
order: 0
---

# Iam Token Validator

## Overview

`iamtokenvalidator` is a Go package designed to validate and decode JWT tokens using JSON Web Keys (JWK) fetched from a specified MC-IAM-MANAGER endpoint(https://example.com:5000/api/auth/certs).
It provides functionality to verify tokens and extract claims, supporting the RS256, RS384, and RS512 signing methods.

## Installation

To install the package, use the following command:

```bash
go get github.com/m-cmp/mc-iam-manager/iamtokenvalidator
```

## Usage

### Importing the Package

To use `iamtokenvalidator` in your Go project, import it as follows:

```go
import "github.com/m-cmp/mc-iam-manager/iamtokenvalidator"
```

### Functions

#### GetPubkeyIamManager

Fetches the JWK set from the provided MC-IAM-MANAGER URL and prepares the public key for token validation.

```go
func GetPubkeyIamManager(host string) error
```

**Parameters:**

- `host`: The URL of the MC-IAM-MANAGER service certs endpoint (https://example.com:5000/api/auth/certs).

**Returns:**

- `error`: An error if fetching the JWK set fails.

**Example:**

```go
err := iamtokenvalidator.GetPubkeyIamManager("https://your-iam-manager-host")
if err != nil {
    log.Fatalf("Failed to get public key: %v", err)
}
```

#### IsTokenValid

Validates the given JWT token string using the previously fetched JWK set.

```go
func IsTokenValid(tokenString string) error
```

**Parameters:**

- `tokenString`: The JWT token string to validate.

**Returns:**

- `error`: An error if the token is invalid.

**Example:**

```go
err := iamtokenvalidator.IsTokenValid("your-jwt-token")
if err != nil {
    fmt.Printf("Token is invalid: %v", err)
} else {
    fmt.Println("Token is valid")
}
```

#### GetTokenClaimsByIamManagerClaims

Parses the given JWT token string and extracts claims defined in `IamManagerClaims`.

```go
func GetTokenClaimsByIamManagerClaims(tokenString string) (*IamManagerClaims, error)
```

**Parameters:**

- `tokenString`: The JWT token string to parse.

**Returns:**

- `*IamManagerClaims`: The extracted claims.
- `error`: An error if the token is invalid.

**Example:**

```go
claims, err := iamtokenvalidator.GetTokenClaimsByIamManagerClaims("your-jwt-token")
if err != nil {
    fmt.Printf("Failed to get claims: %v", err)
} else {
    fmt.Printf("UserID: %s, UserName: %s", claims.UserId, claims.UserName)
}
```

#### GetTokenClaimsByCustomClaims

Parses the given JWT token string and extracts custom claims defined by the user.

```go
func GetTokenClaimsByCustomClaims(tokenString string, myclaims interface{}) (interface{}, error)
```

**Parameters:**

- `tokenString`: The JWT token string to parse.
- `myclaims`: A custom claims struct to extract.

**Returns:**

- `interface{}`: The extracted custom claims.
- `error`: An error if the token is invalid.

**Example:**

```go
type CustomClaims struct {
    jwt.StandardClaims
    Email string `json:"email"`
}

var customClaims CustomClaims
claims, err := iamtokenvalidator.GetTokenClaimsByCustomClaims("your-jwt-token", &customClaims)
if err != nil {
    fmt.Printf("Failed to get custom claims: %v", err)
} else {
    fmt.Printf("Email: %s", claims.(*CustomClaims).Email)
}
```

### Supporting Functions

#### keyfunction

A helper function to support the RS256, RS384, and RS512 signing methods.

```go
func keyfunction(token *jwt.Token) (interface{}, error)
```