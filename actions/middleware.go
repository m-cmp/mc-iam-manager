package actions

import (
	"fmt"
	"net/http"

	"github.com/gobuffalo/buffalo"
)

func IsAuth(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		// AccessToken := c.Session().Get("AccessToken")
		AccessToken := c.Request().Header.Get("Authorization")

		// 아래로 DecodeAccessToken 을 사용
		token, tokenMap, err := KC_client.DecodeAccessToken(c, AccessToken, KC_realm)
		// token 이 expire 되면 err 로그 출력 됨 : could not decode accessToken with custom claims: Token is expired
		if err != nil {
			return c.Render(http.StatusOK, r.JSON(map[string]interface{}{
				"err": err.Error(),
			}))
		}
		if !token.Valid {
			return c.Render(http.StatusOK, r.JSON(map[string]interface{}{
				"token.Valid": false,
				"err":         err.Error(),
			}))
		}

		fmt.Println("******************************")
		fmt.Println("DecodeAccessToken err =", err)
		fmt.Println("******************************")
		fmt.Println("DecodeAccessToken token =", token)
		fmt.Println("******************************")
		fmt.Println("DecodeAccessToken tokenMapClaims =", tokenMap)
		fmt.Println("******************************")
		fmt.Println("DecodeAccessToken token.Valid =", token.Valid)

		return next(c)
	}
}
