package actions

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/validate/v3"
	"github.com/gobuffalo/validate/v3/validators"
	"github.com/m-cmp/mc-iam-manager/handler/keycloak"
)

func AuthLoginHandler(c buffalo.Context) error {
	user := &keycloak.UserLogin{}
	err := c.Bind(&user)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": err.Error()}))
	}

	validateErr := validate.Validate(
		&validators.StringIsPresent{Field: user.Id, Name: "id"},
		&validators.StringIsPresent{Field: user.Password, Name: "password"},
	)
	if validateErr.HasAny() {
		fmt.Println(validateErr)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": validateErr.Error()}))
	}

	accessTokenResponse, err := keycloak.KeycloakLogin(user.Id, user.Password)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(accessTokenResponse))
}

func AuthLoginRefreshHandler(c buffalo.Context) error {
	user := &keycloak.UserLoginRefresh{}
	err := c.Bind(&user)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": err.Error()}))
	}

	validateErr := validate.Validate(
		&validators.StringIsPresent{Field: user.RefreshToken, Name: "refresh_token"},
	)
	if validateErr.HasAny() {
		fmt.Println(validateErr)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": validateErr.Error()}))
	}

	accessTokenResponse, err := keycloak.KeycloakRefreshToken(user.RefreshToken)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(accessTokenResponse))
}

func AuthLogoutHandler(c buffalo.Context) error {
	user := &keycloak.UserLogout{}
	err := c.Bind(&user)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": err.Error()}))
	}

	validateErr := validate.Validate(
		&validators.StringIsPresent{Field: user.RefreshToken, Name: "refresh_token"},
	)
	if validateErr.HasAny() {
		fmt.Println(validateErr)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": validateErr.Error()}))
	}

	err = keycloak.KeycloakLogout(user.RefreshToken)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": err.Error()}))
	}

	return c.Render(http.StatusNoContent, nil)
}

func AuthGetUserInfo(c buffalo.Context) error {
	accessToken := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")

	userinfo, err := keycloak.KeycloakGetUserInfo(accessToken)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(userinfo))
}

func AuthGetUserValidate(c buffalo.Context) error {
	accessToken := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")

	_, err := keycloak.KeycloakGetUserInfo(accessToken)
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": err.Error()}))
	}

	return c.Render(http.StatusOK, nil)
}

func AuthGetCerts(c buffalo.Context) error {
	cert, err := keycloak.KeycloakGetCerts()
	if err != nil {
		log.Println(err)
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"message": err.Error()}))
	}

	return c.Render(http.StatusOK, r.JSON(cert))
}
