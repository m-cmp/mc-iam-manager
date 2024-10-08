package iamtokenvalidator

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/jwk"
)

type DefaultClaims struct {
	*jwt.RegisteredClaims
}

type IamManagerClaims struct {
	*jwt.RegisteredClaims
	Authorization struct {
		Permissions []struct {
			Rsid   string `json:"rsid"`
			Rsname string `json:"rsname"`
		} `json:"permissions"`
	}
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	ResourceAccess struct {
		RealmManagement struct {
			Roles []string `json:"roles"`
		} `json:"realm-management"`
		MciamClient struct {
			Roles []string `json:"roles"`
		} `json:"mciamClient"`
	} `json:"resource_access"`
	Scope       string   `json:"scope"`
	Roles       []string `json:"roles"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Company     string   `json:"company"`
	UserID      string   `json:"userid"`
}

var (
	jwkSet jwk.Set
)

// 해당 기능은 1회 반드시 호출되어야 합니다.
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

// GetPubkeyIamManager는 제공된 MC-IAM-MANAGER url을 통해 "/api/auth/certs" 의 인증서를 받아 공용키를 준비합니다.
// 정상시 error 를 반환하지 않습니다.
// jwkSet fetch 오류 발생시 에러를 반환합니다. (panic, fatal 권장) Tls 오류시 사용합니다. 개발환경에서만 권장됩니다.
func GetPubkeyIamManagerTlsSkipped(certUrl string) error {
	var err error
	ctx := context.Background()

	// Create a custom HTTP client that skips TLS verification
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Skip verification
		},
	}
	client := &http.Client{Transport: transport}

	jwkSet, err = jwk.Fetch(ctx, certUrl, jwk.WithHTTPClient(client))
	if err != nil {
		return err
	}
	return nil
}

// IsTokenValid는 GetPubkeyIamManager에서 설정된 jwkSet을 바탕으로
// tokenString 값을 ParseWithClaims하여 token.Valid를 검증하고 마칩니다.
// 검증이 성공했을때, error를 반환하지 않습니다. valid 하지 않을시, token is invalid 와 함께 오류 내용을 반환합니다.
func IsTokenValid(tokenString string) error {
	token, err := jwt.ParseWithClaims(tokenString, &DefaultClaims{}, Keyfunction)
	if err != nil {
		fmt.Println("@@@ ParseWithClaims", err.Error())
		return fmt.Errorf("token is invalid : %s", err.Error())
	}
	if token.Valid {
		return nil
	} else {
		return fmt.Errorf("token is invalid")
	}
}

// IsTicketValidWithOperationId 함수는 주어진 JWT 토큰과 operationId를 검증합니다.
// - JWT 토큰이 유효하고 operationId와 일치하면 nil을 반환합니다.
// - 토큰이 유효하지 않거나 operationId와 일치하지 않으면 에러를 반환합니다.
func IsTicketValidWithOperationId(tokenString string, operationId string) error {
	token, err := jwt.ParseWithClaims(tokenString, &IamManagerClaims{}, Keyfunction)
	if err != nil {
		return fmt.Errorf("ticket is invalid : %s", err.Error())
	}
	if claims, ok := token.Claims.(*IamManagerClaims); ok && token.Valid {
		for _, permission := range claims.Authorization.Permissions {
			permissionParts := strings.Split(permission.Rsname, ":")
			if len(permissionParts) < 4 {
				continue
			}
			if strings.EqualFold(permissionParts[2], operationId) {
				return nil
			}
		}
		return fmt.Errorf("ticket mismatch with operationId")
	} else {
		return fmt.Errorf("ticket is invalid")
	}
}

// IsTicketValidWithReqUri 함수는 주어진 JWT 토큰과 요청 URI(requestUri)를 검증합니다.
// - JWT 토큰이 유효하고 requestUri와 일치하면 nil을 반환합니다.
// - 토큰이 유효하지 않거나 requestUri와 일치하지 않으면 에러를 반환합니다.
func IsTicketValidWithReqUri(tokenString string, requestUri string) error {
	token, err := jwt.ParseWithClaims(tokenString, &IamManagerClaims{}, Keyfunction)
	if err != nil {
		return fmt.Errorf("ticket is invalid : %s", err.Error())
	}
	if claims, ok := token.Claims.(*IamManagerClaims); ok && token.Valid {
		for _, permission := range claims.Authorization.Permissions {
			permissionParts := strings.Split(permission.Rsname, ":")
			if len(permissionParts) < 4 {
				continue
			}
			if isEqualUri(permissionParts[4], requestUri) {
				return nil
			}
		}
		return fmt.Errorf("ticket mismatch with requestUri")
	} else {
		return fmt.Errorf("ticket is invalid")
	}
}

// isEqualUri 함수는 주어진 패턴과 문자열을 비교하여 일치 여부를 판단합니다.
// - pattern: 비교할 URI 패턴 문자열 (예: "/path/{variable}/endpoint")
// - str: 비교 대상이 되는 실제 URI 문자열
func isEqualUri(pattern string, str string) bool {
	regexPattern := regexp.MustCompile(`\{[^/]+\}`).ReplaceAllString(pattern, `[^/]+`)
	regex := regexp.MustCompile("^" + regexPattern + "$")
	return regex.MatchString(str)
}

// GetTokenInfoByIamManagerClaim는 GetPubkeyIamManager에서 설정된 jwkSet을 바탕으로
// tokenString 값을 IamManagerClaims에 정의된 UserId, UserName, PreferredUsername, RealmAccess 및 jwt.StandardClaims 를 사용하여
// ParseWithClaims하여 valid 를 검증하고 IamManagerClaims를 반환합니다.
// token이 valid 하지 않을시, token is invalid 와 함께 오류 내용을 반환합니다.
func GetTokenClaimsByIamManagerClaims(tokenString string) (*IamManagerClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &IamManagerClaims{}, Keyfunction)
	if err != nil {
		return nil, fmt.Errorf("token is invalid : %s", err.Error())
	}
	if claims, ok := token.Claims.(*IamManagerClaims); ok {
		return claims, nil
	}
	return nil, fmt.Errorf("token is not parse with IamManagerClaims")
}

// GetTokenClaimsByCustomClaims는 GetPubkeyIamManager에서 설정된 jwkSet을 바탕으로
// tokenString 값을 임의로 정의한 Claims를 사용하여
// ParseWithClaims하여 valid 를 검증하고 Claims를 반환합니다.
// token이 valid 하지 않을시, token is invalid 와 함께 오류 내용을 반환합니다.
func GetTokenClaimsByCustomClaims(tokenString string, myclaims interface{}) (interface{}, error) {
	token, err := jwt.ParseWithClaims(tokenString, myclaims.(jwt.Claims), Keyfunction)
	if err != nil {
		return nil, fmt.Errorf("token is invalid : %s", err.Error())
	}
	if token.Valid {
		return token.Claims, nil
	} else {
		return nil, fmt.Errorf("token is invalid")
	}
}

// Keyfunction은 토큰 검증을 위한 rawkey 를 반환합니다.
// RS256, RS384, RS512 can be Signing Method
func Keyfunction(token *jwt.Token) (interface{}, error) {
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

// IsHasRoleInUserRolesArr는 제공된 role string arr와 user roles arr 를 비교하여 한개라도 roles 에 있을시 True 를 반환합니다.
func IsHasRoleInUserRolesArr(grandtedRoleArr []string, userRolesArr []string) bool {
	userRolesArrSet := make(map[string]struct{}, len(userRolesArr))
	for _, v := range userRolesArr {
		userRolesArrSet[v] = struct{}{}
	}
	for _, v := range grandtedRoleArr {
		if _, found := userRolesArrSet[v]; found {
			return true
		}
	}
	return false
}
