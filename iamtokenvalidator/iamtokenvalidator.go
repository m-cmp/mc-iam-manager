package iamtokenvalidator

import (
	"context"
	"fmt"

	"github.com/golang-jwt/jwt"
	"github.com/lestrrat-go/jwx/jwk"
)

type DefaultClaims struct {
	*jwt.StandardClaims
}

type IamManagerClaims struct {
	*jwt.StandardClaims
	UserId            string `json:"upn"`
	UserName          string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	RealmAccess       struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
}

var (
	jwkSet jwk.Set
)

// GetPubkeyIamManager는 제공된 MC-IAM-MANAGER url을 통해 "/api/auth/certs" 의 인증서를 받아 공용키를 준비합니다.
// 정상시 error 를 반환하지 않습니다.
// jwkSet fetch 오류 발생시 에러를 반환합니다. (panic, fatal 권장)
func GetPubkeyIamManager(certUrl string) error {
	var err error
	ctx := context.Background()
	jwkSet, err = jwk.Fetch(ctx, certUrl)
	if err != nil {
		return err
	}
	return nil
}

// IsTokenValid는 GetPubkeyIamManager에서 설정된 jwkSet을 바탕으로
// tokenString 값을 ParseWithClaims하여 token.Valid를 검증하고 마칩니다.
// 검증이 성공했을때, error를 반환하지 않습니다. valid 하지 않을시, token is invalid 와 함께 오류 내용을 반환합니다.
func IsTokenValid(tokenString string) error {
	token, err := jwt.ParseWithClaims(tokenString, &DefaultClaims{}, keyfunction)
	if err != nil {
		return fmt.Errorf("token is invalid : %s", err.Error())
	}
	if token.Valid {
		return nil
	} else {
		return fmt.Errorf("token is invalid")
	}
}

// GetTokenInfoByIamManagerClaim는 GetPubkeyIamManager에서 설정된 jwkSet을 바탕으로
// tokenString 값을 IamManagerClaims에 정의된 UserId, UserName, PreferredUsername, RealmAccess 및 jwt.StandardClaims 를 사용하여
// ParseWithClaims하여 valid 를 검증하고 IamManagerClaims를 반환합니다.
// token이 valid 하지 않을시, token is invalid 와 함께 오류 내용을 반환합니다.
func GetTokenClaimsByIamManagerClaims(tokenString string) (*IamManagerClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &IamManagerClaims{}, keyfunction)
	if err != nil {
		return nil, fmt.Errorf("token is invalid : %s", err.Error())
	}
	if claims, ok := token.Claims.(*IamManagerClaims); ok && token.Valid {
		return claims, nil
	} else {
		return nil, fmt.Errorf("token is invalid")
	}
}

// GetTokenClaimsByCustomClaims는 GetPubkeyIamManager에서 설정된 jwkSet을 바탕으로
// tokenString 값을 임의로 정의한 Claims를 사용하여
// ParseWithClaims하여 valid 를 검증하고 Claims를 반환합니다.
// token이 valid 하지 않을시, token is invalid 와 함께 오류 내용을 반환합니다.
func GetTokenClaimsByCustomClaims(tokenString string, myclaims interface{}) (interface{}, error) {
	token, err := jwt.ParseWithClaims(tokenString, myclaims.(jwt.Claims), keyfunction)
	if err != nil {
		return nil, fmt.Errorf("token is invalid : %s", err.Error())
	}
	if token.Valid {
		return token.Claims, nil
	} else {
		return nil, fmt.Errorf("token is invalid")
	}
}

// RS256, RS384, RS512 can be Signing Method
func keyfunction(token *jwt.Token) (interface{}, error) {
	if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
		return nil, fmt.Errorf("keyfunctionErr : unexpected signing method[%v]: RS256, RS384, RS512 can be Signing Method...", token.Header["alg"])
	}
	kid := token.Header["kid"].(string)
	keys, nokey := jwkSet.LookupKeyID(kid)
	if !nokey {
		return nil, fmt.Errorf("keyfunctionErr : no match 'kid' from provideded token: %s", kid)
	}
	var raw interface{}
	if err := keys.Raw(&raw); err != nil {
		return nil, fmt.Errorf("keyfunctionErr : failed to get keys : %s", err)
	}
	return raw, nil
}
