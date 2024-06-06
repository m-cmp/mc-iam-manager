package actions

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/Nerzal/gocloak/v13"
	"github.com/gobuffalo/envy"
)

var (
	KEYCLOAK_USE bool
)

func init() {
	var err error
	KEYCLOAK_USE, err = strconv.ParseBool(envy.Get("KEYCLOAK_USE", "true"))
	if err != nil {
		panic(errors.New("environment variable file setting error : KEYCLOAK_USE :" + err.Error()))
	}
}

func CreateDefaultAdminUserOnIdp() error {
	var err error
	if KEYCLOAK_USE {
		err = KeycloakCreateDefaultAdminUser()
		if err != nil {
			panicErr := errors.New("KeycloakCreateDefaultAdminUser() error :" + err.Error())
			// panic(errors.New("KeycloakCreateDefaultAdminUser() error :" + err.Error()))
			fmt.Println(panicErr)
		}
	}
	return nil
}

func KeycloakCreateDefaultAdminUser() error {

	ctx := context.Background()

	token, err := KEYCLOAK.LoginAdmin(ctx, KEYCLAOK_ADMIN, KEYCLAOK_ADMIN_PASSWORD, "master")
	if err != nil {
		fmt.Println(err)
	}

	user := gocloak.User{
		FirstName: gocloak.StringP(ADMINUSERID),
		LastName:  gocloak.StringP(ADMINUSERID),
		Enabled:   gocloak.BoolP(true),
		Email:     gocloak.StringP(ADMINUSERID + "@example.com"),
		Username:  gocloak.StringP(ADMINUSERID),
	}

	userId, err := KEYCLOAK.CreateUser(ctx, token.AccessToken, KEYCLAOK_REALM, user)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = KEYCLOAK.SetPassword(ctx, token.AccessToken, userId, KEYCLAOK_REALM, ADMINUSERPASSWORD, false)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}
