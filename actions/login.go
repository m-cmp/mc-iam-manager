package actions

import (
	"fmt"
	"net/http"

	"github.com/gobuffalo/buffalo"
)

// Iam Manager 로그인 화면
func IamLoginForm(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.HTML("login/index.html"))
}

// Iam Manager Login 처리
func IamLogin(c buffalo.Context) error {

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

// Iam Manager Login 처리
func IamLogin(c buffalo.Context) error {

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

// Iam Manager Login 처리
func IamLoginApi(c buffalo.Context) error {

	username := c.Param("username")
	password := c.Param("password")

	token, err := KC_client.Login(c, KC_clientID, KC_clientSecret, KC_realm, username, password)
	if err != nil {
		c.Set("simplestr", err.Error())
		return c.Render(http.StatusOK, r.HTML("simplestr.html"))
	}

	return map[string]interface{}{
		"message": "something query error",
		"status":  "301",
	}

	//token.AccessToken
	return c.Render(http.StatusOK, r.JSON(map[string]interface{}{
		"iamAccessToken": token.AccessToken,
	}))
}

func AuthUserTestPageHandler(c buffalo.Context) error {
	c.Set("simplestr", "You are good to go")
	return c.Render(http.StatusOK, r.HTML("simplestr.html"))
}

func NotAuthUserTestPageHandler(c buffalo.Context) error {
	c.Set("simplestr", "You are blocked by middleware")
	return c.Render(http.StatusOK, r.HTML("simplestr.html"))
}
