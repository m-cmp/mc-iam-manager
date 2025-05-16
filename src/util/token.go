package util

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/m-cmp/mc-iam-manager/config"
)

// ValidateToken은 JWT 토큰을 검증하고 claims를 반환합니다.
func ValidateToken(tokenString string) (*jwt.MapClaims, error) {
	// 토큰 파싱
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 서명 알고리즘 확인
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Keycloak 공개키 가져오기
		publicKey, err := config.KC.GetPublicKey()
		if err != nil {
			return nil, fmt.Errorf("failed to get public key: %v", err)
		}

		return publicKey, nil
	})

	if err != nil {
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

	return &claims, nil
}
