package model

// JWTResponse mirrors gocloak.JWT for Swagger documentation purposes.
// The actual API response will contain the fields from gocloak.JWT.
type JWTResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	NotBeforePolicy  int    `json:"not-before-policy"` // Use the correct JSON tag if needed, often omitted
	SessionState     string `json:"session_state"`
	Scope            string `json:"scope"`
}
