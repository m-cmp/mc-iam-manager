package mcimw

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"
	"github.com/golang-jwt/jwt"
	"github.com/lestrrat-go/jwx/jwk"
)

type mcimwAuth interface {
	AuthLoginHandler(http.ResponseWriter, *http.Request)
	AuthLoginRefreshHandler(http.ResponseWriter, *http.Request)
	AuthLogoutHandler(http.ResponseWriter, *http.Request)
	AuthGetUserInfo(http.ResponseWriter, *http.Request)
	AuthGetUserValidate(http.ResponseWriter, *http.Request)
}
type mcimw struct {
	authMethod mcimwAuth
	grantRoles []string
}

var (
	jwkSet        jwk.Set
	McimwRoleList = []string{}
)

func init() {
	var err error
	ctx := context.Background()
	jwkSet, err = jwk.Fetch(ctx, envKeycloak.KEYCLAOK_JWKSURL)
	if err != nil {
		panic("failed to fetch JWK: " + err.Error())
	}
}

func (mw mcimw) BeginAuthHandler(res http.ResponseWriter, req *http.Request) {
	reqUri := req.RequestURI
	reqMethod := req.Method
	res.Header().Set("Content-Type", "application/json")

	if strings.HasSuffix(reqUri, "/login") && reqMethod == "POST" {
		mw.authMethod.AuthLoginHandler(res, req)
		return
	} else if strings.HasSuffix(reqUri, "/login/refresh") && reqMethod == "POST" {
		mw.authMethod.AuthLoginRefreshHandler(res, req)
		return
	} else if strings.HasSuffix(reqUri, "/logout") && reqMethod == "POST" {
		mw.authMethod.AuthLogoutHandler(res, req)
		return
	} else if strings.HasSuffix(reqUri, "/userinfo") && reqMethod == "GET" {
		mw.authMethod.AuthGetUserInfo(res, req)
		return
	} else if strings.HasSuffix(reqUri, "/validate") && reqMethod == "GET" {
		mw.authMethod.AuthGetUserValidate(res, req)
		return
	}

	res.WriteHeader(http.StatusBadRequest)
	fmt.Fprintln(res, map[string]interface{}{
		"error": http.StatusText(http.StatusBadRequest),
	})
}

func BuffaloMcimw(next buffalo.Handler) buffalo.Handler {
	mr := mcimw{
		grantRoles: McimwRoleList,
	}
	return mr.BuffaloMiddleware(next)
}

func (mr mcimw) BuffaloMiddleware(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		err := mr.istokenValid(strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer "))
		if err != nil {
			return c.Render(http.StatusInternalServerError,
				render.JSON(map[string]string{"err": err.Error()}))
		}
		return next(c)
	}
}

func (mr mcimw) istokenValid(tokenString string) error {
	fmt.Println("istokenValid")
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		fmt.Println("1")

		kid := token.Header["kid"].(string)
		keys, nokey := jwkSet.LookupKeyID(kid)
		if !nokey {
			return nil, fmt.Errorf("no keys found for kid: %s", kid)
		}
		fmt.Println("2")

		var raw interface{}
		if err := keys.Raw(&raw); err != nil {
			return nil, fmt.Errorf("failed to get key: %s", err)
		}
		fmt.Println("3")

		return raw, nil
	})
	fmt.Println("ParseWithClaims")

	if err != nil {
		return fmt.Errorf("failed to parse token: %s", err)
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		fmt.Printf("user: %s\n", claims.PreferredUsername)
		if !mr.isRoleContains(claims.RealmRole) {
			return fmt.Errorf("token is invalid: %s", err.Error())
		}
		return nil
	} else {
		return fmt.Errorf("token is invalid: %s", err.Error())
	}
}

func (mr mcimw) isRoleContains(arr []string) bool {
	fmt.Println("mr :", mr)
	set := make(map[string]bool)
	for _, v := range mr.grantRoles {
		set[v] = true
	}
	fmt.Println(set)
	for _, v := range arr {
		if set[v] {
			return true
		}
	}
	return false
}

// func EchoMcimw(next echo.HandlerFunc) echo.HandlerFunc {
// 	mr := mcimw{
// 		roleNames: McimwRoleList,
// 	}
// 	return mr.EchoMiddleware(next)
// }

// func (mr mcimw) EchoMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
// 	return func(c echo.Context) error {
// 		defer func() {

// 		}()
// 		return next(c)
// 	}
// }
