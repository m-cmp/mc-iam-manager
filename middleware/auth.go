package middleware

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"

	"github.com/m-cmp/mc-iam-manager/handler/keycloak"
	"github.com/m-cmp/mc-iam-manager/iamtokenvalidator"
)

var r *render.Engine

func init() {
	r = render.New(render.Options{
		DefaultContentType: "application/json",
	})

	KEYCLOAKHOST := os.Getenv("KEYCLOAK_HOST")
	KEYCLAOKREALM := os.Getenv("KEYCLAOK_REALM")
	log.Printf("Trying to fetch Pubkey %s/realms/%s/protocol/openid-connect/certs", KEYCLOAKHOST, KEYCLAOKREALM)
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
			log.Println("IsAuthMiddleware :", err.Error())
			return c.Render(http.StatusUnauthorized, r.JSON(map[string]string{"error": "Unauthorized"}))
		}
		return next(c)
	}
}

func IsTicketValidMiddleware(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		accessToken := c.Value("accessToken").(string)
		requestURI := c.Request().RequestURI
		if idx := strings.Index(requestURI, "?"); idx != -1 {
			requestURI = requestURI[:idx]
		}
		err := iamtokenvalidator.IsTicketValidWithReqUri(accessToken, requestURI)
		if err != nil {
			log.Println("IsTicketValidMiddleware :", err.Error())
			return c.Render(http.StatusUnauthorized, r.JSON(map[string]string{"error": "Unauthorized"}))
		}
		return next(c)
	}
}

func SetContextMiddleware(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		accessToken := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")
		claims, err := iamtokenvalidator.GetTokenClaimsByIamManagerClaims(accessToken)
		if err != nil {
			return c.Render(http.StatusUnauthorized, r.JSON(map[string]string{"error": "Unauthorized"}))
		}

		requestURI := c.Request().RequestURI
		if idx := strings.Index(requestURI, "?"); idx != -1 {
			requestURI = requestURI[:idx]
		}

		req := keycloak.RequestTicket{
			Uri: requestURI,
		}
		jwtToken, err := keycloak.KeycloakGetPermissionTicket(accessToken, req)
		if err != nil {
			log.Println("SetContextMiddleware Error : ", err.Error(), c.Request().RequestURI)
			return c.Render(http.StatusUnauthorized, r.JSON(map[string]string{"error": "Unauthorized"}))
		}
		c.Set("accessToken", jwtToken.AccessToken)
		c.Set("roles", claims.RealmAccess.Roles)
		return next(c)
	}
}
