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
	KEYCLAOKREALM := os.Getenv("KEYCLOAK_REALM")
	log.Printf("Trying to fetch Pubkey %s/realms/%s/protocol/openid-connect/certs", KEYCLOAKHOST, KEYCLAOKREALM)
	err := iamtokenvalidator.GetPubkeyIamManager(KEYCLOAKHOST + "/realms/" + KEYCLAOKREALM + "/protocol/openid-connect/certs")
	if err != nil {
		panic(err)
	}
	log.Printf("Pubkey fetch Success")
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
		requestURI := c.Request().RequestURI
		if idx := strings.Index(requestURI, "?"); idx != -1 {
			requestURI = requestURI[:idx]
		}
		req := keycloak.RequestTicket{
			Uri: requestURI,
		}
		accessToken := c.Value("accessToken").(string)
		jwtToken, err := keycloak.KeycloakGetPermissionTicket(accessToken, req)
		if err != nil {
			log.Println("IsTicketValidMiddleware Error : ", err.Error(), requestURI)
			return c.Render(http.StatusUnauthorized, r.JSON(map[string]string{"error": "Unauthorized"}))
		}

		accessToken = jwtToken.AccessToken
		c.Set("accessToken", accessToken)
		err = iamtokenvalidator.IsTicketValidWithReqUri(accessToken, requestURI)
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
		log.Printf("User Claims : %+v\n", claims)
		c.Set("accessToken", accessToken)
		return next(c)
	}
}
