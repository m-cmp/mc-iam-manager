package actions

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gobuffalo/buffalo"

	"github.com/Nerzal/gocloak/v13"
)

var KC_admin = os.Getenv("KC_admin")
var KC_passwd = os.Getenv("KC_passwd")
var KC_uri = os.Getenv("KC_uri")
var KC_clientID = os.Getenv("KC_clientID")
var KC_clientSecret = os.Getenv("KC_clientSecret")
var KC_realm = os.Getenv("KC_realm")
var KC_client = gocloak.NewClient(KC_uri)

func KcHomeHandler(c buffalo.Context) error {
	c.Set("simplestr", "welcome mcloak home!")
	return c.Render(http.StatusOK, r.HTML("kctest/index.html"))
}

func KcCreateUserHandler(c buffalo.Context) error {
	token, err := KC_client.LoginAdmin(c, KC_admin, KC_passwd, "master")
	if err != nil {
		c.Set("simplestr", err.Error()+"### Something wrong with the credentials or url ###")
		return c.Render(http.StatusOK, r.HTML("kctest/index.html"))
		// panic("Something wrong with the credentials or url")
	}

	fmt.Println(token)

	user := gocloak.User{
		FirstName: gocloak.StringP("ra"),
		LastName:  gocloak.StringP("ccoon"),
		Email:     gocloak.StringP("mega@zone.cloud"),
		Enabled:   gocloak.BoolP(true),
		Username:  gocloak.StringP("raccoon"),
	}

	_, err = KC_client.CreateUser(c, token.AccessToken, "master", user)
	if err != nil {
		c.Set("simplestr", err.Error()+"### Oh no!, failed to create user :( ###")
		return c.Render(http.StatusOK, r.HTML("kctest/index.html"))
		// panic("Oh no!, failed to create user :(")
	}

	c.Set("simplestr", "success")
	return c.Render(http.StatusOK, r.HTML("kctest/index.html"))
}

func KcLoginAdminHandler(c buffalo.Context) error {
	token, err := KC_client.LoginAdmin(c, KC_admin, KC_passwd, "master")
	if err != nil {
		c.Set("simplestr", err.Error()+"### Something wrong with the credentials or url ###")
		return c.Render(http.StatusOK, r.HTML("kctest/index.html"))
	}

	return c.Render(http.StatusOK, r.JSON(token))
}
