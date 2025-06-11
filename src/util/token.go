package util

import (
	"fmt"
	"log"

	"crypto/rsa"

	"github.com/golang-jwt/jwt/v5"
	"github.com/m-cmp/mc-iam-manager/config"
)

// ValidateToken은 JWT 토큰을 검증하고 claims를 반환합니다.
func ValidateToken(tokenString string) (*jwt.MapClaims, error) {
	// 토큰 파싱
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 서명 알고리즘 확인
		if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// 토큰의 kid 확인
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid header not found")
		}
		log.Printf("[DEBUG] Token KID: %s", kid)

		// Keycloak 공개키 가져오기
		publicKey, err := config.KC.GetPublicKey()
		if err != nil {
			log.Printf("[DEBUG] Failed to get public key: %v", err)
			return nil, fmt.Errorf("failed to get public key: %v", err)
		}

		// 공개키 정보 로깅
		log.Printf("[DEBUG] Public Key Type: %T", publicKey)
		if rsaKey, ok := publicKey.(*rsa.PublicKey); ok {
			log.Printf("[DEBUG] RSA Key Size: %d bits", rsaKey.Size()*8)
		}

		return publicKey, nil
	})

	if err != nil {
		log.Printf("[DEBUG] Token validation error: %v", err)
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// claims 추출
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// 토큰 정보 로깅
	log.Printf("[DEBUG] Token Claims - Issuer: %v", claims["iss"])
	log.Printf("[DEBUG] Token Claims - Audience: %v", claims["aud"])
	log.Printf("[DEBUG] Token Claims - Subject: %v", claims["sub"])

	return &claims, nil
}
