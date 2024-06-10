package mcimw

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"
	"github.com/gobuffalo/envy"
	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/lestrrat-go/jwx/jwk"
)

var (
	USE_IDP string
	jwkSet  jwk.Set
)

func init() {
	USE_IDP = envy.Get("USE_IDP", "keycloak") // [self, keycloak]
	jwksURL := envy.Get("JWKSURL", "")
	ctx := context.Background()
	var err error
	jwkSet, err = jwk.Fetch(ctx, jwksURL)
	if err != nil {
		panic("failed to fetch JWK: " + err.Error())
	}

}

type mcimw struct {
	roleNames []string
}

var McimwRoleList = []string{}

func BeginAuthHandler(res http.ResponseWriter, req *http.Request) {
	switch USE_IDP {
	case "keycloak":
		BeginKeycloakAuth(res, req)
		return
	case "mciammanager":
		BeginMCIAMAuth(res, req)
		return
	default:
		res.WriteHeader(http.StatusBadRequest)
		err := errors.New("NO MATCH IDP")
		fmt.Fprintln(res, err)
		return
	}
}

func BuffaloMcimw(next buffalo.Handler) buffalo.Handler {
	mr := mcimw{
		roleNames: McimwRoleList,
	}
	return mr.BuffaloMiddleware(next)
}

func (mr mcimw) BuffaloMiddleware(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		tkn := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")
		err := mr.istokenValid(tkn)
		if err != nil {
			return c.Render(http.StatusInternalServerError,
				render.JSON(map[string]string{"err": err.Error()}))
		}

		return next(c)
	}
}

func EchoMcimw(next echo.HandlerFunc) echo.HandlerFunc {
	mr := mcimw{
		roleNames: McimwRoleList,
	}
	return mr.EchoMiddleware(next)
}

func (mr mcimw) EchoMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer func() {

		}()
		return next(c)
	}
}

type CustomClaims struct {
	*jwt.StandardClaims
	PreferredUsername string `json:"preferred_username"`
}

func (mr mcimw) istokenValid(tokenString string) error {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		kid := token.Header["kid"].(string)
		keys, nokey := jwkSet.LookupKeyID(kid)
		if !nokey {
			return nil, fmt.Errorf("no keys found for kid: %s", kid)
		}
		var raw interface{}
		if err := keys.Raw(&raw); err != nil {
			return nil, fmt.Errorf("failed to get key: %s", err)
		}
		return raw, nil
	})

	if err != nil {
		return fmt.Errorf("failed to parse token: %s", err)
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		fmt.Printf("user: %s\n", claims.PreferredUsername)
		if !mr.isRoleContains(claims.PreferredUsername) {
			return fmt.Errorf("token is invalid: %s", err.Error())
		}
		return nil
	} else {
		return fmt.Errorf("token is invalid: %s", err.Error())
	}
}

func (mr mcimw) isRoleContains(target string) bool {
	for _, str := range mr.roleNames {
		if str == target {
			return true
		}
	}
	return false
}
