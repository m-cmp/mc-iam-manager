package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"

	"github.com/m-cmp/mc-iam-manager/iamtokenvalidator"
)

var r *render.Engine

func init() {
	r = render.New(render.Options{
		DefaultContentType: "application/json",
	})

	KEYCLOAKHOST := os.Getenv("KEYCLOAK_HOST")
	KEYCLAOKREALM := os.Getenv("KEYCLAOK_REALM")
	fmt.Println("Trying to fetch Pubkey : URL :", KEYCLOAKHOST)
	err := iamtokenvalidator.GetPubkeyIamManager(KEYCLOAKHOST + "/realms/" + KEYCLAOKREALM + "/protocol/openid-connect/certs")
	if err != nil {
		panic(err)
	}
}

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
