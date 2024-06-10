package actions

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gobuffalo/buffalo"

	"github.com/Nerzal/gocloak/v13"
)

var KC_admin = "admin"
var KC_passwd = "admin"
var KC_uri = os.Getenv("keycloakHost")
var KC_clientID = os.Getenv("KC_clientID")
var KC_clientSecret = os.Getenv("KC_clientSecret")
var KC_realm = os.Getenv("KC_realm")
var KC_client = gocloak.NewClient(KC_uri)

func KcHomeHandler(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.JSON("OK"))
}

func KcCreateUserHandler(c buffalo.Context) error {
	token, err := KC_client.LoginAdmin(c, KC_admin, KC_passwd, "master")
	if err != nil {
		fmt.Println(err)
		return c.Render(http.StatusOK, r.JSON(err.Error()))
	}

	fmt.Println(token)

	user := gocloak.User{
		FirstName: gocloak.StringP("MCPUSER"),
		LastName:  gocloak.StringP("ADMIN"),
		Enabled:   gocloak.BoolP(true),
		Username:  gocloak.StringP("mcpuser"),
	}

	userId, err := KC_client.CreateUser(c, token.AccessToken, "master", user)
	if err != nil {
		fmt.Println(err)
		return c.Render(http.StatusOK, r.JSON(err.Error()))
	}

	// func (g *GoCloak) SetPassword(ctx context.Context, token, userID, realm, password string, temporary bool) error {
	err = KC_client.SetPassword(c, token.AccessToken, userId, "master", "admin", false)
	if err != nil {
		fmt.Println(err)
		return c.Render(http.StatusOK, r.JSON(err.Error()))
	}

	return c.Render(http.StatusOK, r.JSON("good"))
}

func KcLoginAdminHandler(c buffalo.Context) error {
	token, err := KC_client.LoginAdmin(c, KC_admin, KC_passwd, "master")
	if err != nil {
		c.Set("simplestr", err.Error()+"### Something wrong with the credentials or url ###")
		return c.Render(http.StatusOK, r.HTML("kctest/index.html"))
	}

	return c.Render(http.StatusOK, r.JSON(token))
}

func DebugGetRealmRoleByID(c buffalo.Context) error {

	token, err := KC_client.LoginAdmin(c, KC_admin, KC_passwd, "master")
	if err != nil {
		log.Println("ERR : while get admin console token")
		log.Println(err.Error())
		return c.Render(http.StatusOK, r.JSON(err))
	}
	role, err := KC_client.GetRealmRoleByID(c, token.AccessToken, KC_realm, c.Param("roleid"))
	if err != nil {
		log.Println("ERR : while GetRealmRoleByID")
		log.Println(err.Error())
		return c.Render(http.StatusOK, r.JSON(err))
	}

	fmt.Println("#########################")
	fmt.Printf("Request Role is : %+v\n", role)
	fmt.Println("#########################")

	return c.Render(http.StatusOK, r.JSON(role))
}
