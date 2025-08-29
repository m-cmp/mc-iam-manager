package util

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// GetMapClaimsFromContext extracts JWT claims as MapClaims from the access token in Echo context.
// It assumes the access token string is stored under the "access_token" key.
func GetMapClaimsFromContext(c echo.Context) (jwt.MapClaims, error) {
	tokenValue := c.Get("access_token") // Get the raw token string stored by middleware
	if tokenValue == nil {
		return nil, fmt.Errorf("'access_token' not found in context")
	}
	tokenString, ok := tokenValue.(string)
	if !ok {
		return nil, fmt.Errorf("failed to assert 'access_token' to string. Actual type: %T", tokenValue)
	}

	// Parse the token string without validation (already validated by middleware)
	// We only need to decode the payload to access claims.
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		// Log the parsing error for more details
		fmt.Printf("[ERROR] GetMapClaimsFromContext: Failed to parse token: %v (Token: %s)\n", err, tokenString)
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}

	return nil, fmt.Errorf("failed to assert claims to jwt.MapClaims")
}
