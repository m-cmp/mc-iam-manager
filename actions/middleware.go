package actions

import (
	"fmt"
	"net/http"

	"github.com/gobuffalo/buffalo"
)

func IsAuth(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		AccessToken := c.Session().Get("AccessToken")

		// 아래로 DecodeAccessToken 을 사용

		token, tokenMap, err := KC_client.DecodeAccessToken(c, AccessToken.(string), KC_realm)
		// token 이 expire 되면 err 로그 출력 됨 : could not decode accessToken with custom claims: Token is expired
		if err != nil {
			// RefreshToken := c.Session().Get("RefreshToken") // db
			// AccessToken, err = KC_client.RefreshToken(c, RefreshToken.(string), KC_clientID, KC_clientSecret, KC_realm)
			// if err != nil {
			// 	fmt.Println("******************************")
			// 	fmt.Println("RefreshToken err", err)
			// 	fmt.Println("******************************")
			// 	return c.Redirect(302, "/buffalo/authuser/not")
			// }
			// c.Session().Set("AccessToken", AccessToken)
			c.Flash().Add("danger", err.Error())
			return c.Redirect(302, "/buffalo/login")
		}

		fmt.Println("******************************")
		fmt.Println("DecodeAccessToken err =", err)
		fmt.Println("******************************")
		fmt.Println("DecodeAccessToken token =", token)
		fmt.Println("******************************")
		fmt.Println("DecodeAccessToken tokenMapClaims =", tokenMap)
		fmt.Println("******************************")

		fmt.Println("DecodeAccessToken token.Valid =", token.Valid)

		if !token.Valid {
			c.Flash().Add("danger", "session expired")
			return c.Redirect(302, "/buffalo/login")
		}

		// 아래로 userinfo를 사용

		userinfo, err := KC_client.GetUserInfo(c, AccessToken.(string), KC_realm)
		if err != nil {
			c.Set("simplestr", err.Error())
			return c.Render(http.StatusOK, r.HTML("simplestr.html"))
		}
		fmt.Println("userinfo", userinfo)

		c.Session().Set("userinfo", userinfo.Name)

		return next(c)
	}
}
