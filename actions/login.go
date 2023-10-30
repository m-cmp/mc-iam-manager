package actions

import (
	"fmt"
	"net/http"

	"github.com/gobuffalo/buffalo"
)

// HomeHandler is a default handler to serve up
// a home page.
func LoginHandler(c buffalo.Context) error {
	if c.Request().Method == "GET" {
		return c.Render(http.StatusOK, r.HTML("login/index.html"))
	}

	var username = c.Request().FormValue("username")
	var password = c.Request().FormValue("password")

	token, err := KC_client.Login(c, KC_clientID, KC_clientSecret, KC_realm, username, password)
	if err != nil {
		c.Set("simplestr", err.Error())
		return c.Render(http.StatusOK, r.HTML("simplestr.html"))
	}
	userinfo, err := KC_client.GetUserInfo(c, token.AccessToken, KC_realm)
	if err != nil {
		c.Set("simplestr", err.Error())
		return c.Render(http.StatusOK, r.HTML("simplestr.html"))
	}
	fmt.Println("userinfo", userinfo)

	c.Session().Set("AccessToken", token.AccessToken)
	// c.Session().Set("RefreshToken", token.RefreshToken) // TODO : save in db to reduce cookie

	return c.Redirect(302, "/buffalo/authuser")
}

func AuthUserTestPageHandler(c buffalo.Context) error {
	c.Set("simplestr", "You are good to go")
	return c.Render(http.StatusOK, r.HTML("simplestr.html"))
}

func NotAuthUserTestPageHandler(c buffalo.Context) error {
	c.Set("simplestr", "You are blocked by middleware")
	return c.Render(http.StatusOK, r.HTML("simplestr.html"))
}
